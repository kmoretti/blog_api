package bot

import (
	"blog_api/src/config"
	"blog_api/src/model"
	momentRepositories "blog_api/src/repositories/moment"
	coreService "blog_api/src/service"
	"blog_api/src/service/oss"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

type telegramListener struct {
	db              *gorm.DB
	bot             *tgbotapi.BotAPI
	channelID       int64
	channelUsername string
	filterUserIDs   map[int64]bool
	ossService      oss.OSSService
	pendingGroups   map[string]*telegramMediaGroup
	startTime       int64 // Unix timestamp, 只接受此时间之后的消息
}

type telegramMediaGroup struct {
	Messages []*tgbotapi.Message
	LastSeen time.Time
}

func StartTelegramListener(db *gorm.DB, cfg *model.Config) {
	tgCfg := cfg.MomentsIntegrated.Integrated.Telegram
	if !cfg.MomentsIntegrated.Enable || !tgCfg.Enable || tgCfg.BotToken == "" {
		return
	}

	bot, err := tgbotapi.NewBotAPI(tgCfg.BotToken)
	if err != nil {
		log.Printf("[telegram] init bot failed: %v", err)
		return
	}
	SetTelegramBot(bot)

	listener := &telegramListener{
		db:            db,
		bot:           bot,
		filterUserIDs: make(map[int64]bool),
		pendingGroups: make(map[string]*telegramMediaGroup),
		startTime:     tgCfg.StartTime,
	}
	if listener.startTime == 0 {
		listener.startTime = time.Now().Unix()
	}

	for _, id := range tgCfg.FilterUserid {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
				listener.filterUserIDs[parsed] = true
			}
		}
	}

	if cid, username, err := parseTelegramChannel(tgCfg.ChannelID); err == nil {
		listener.channelID = cid
		listener.channelUsername = username
	} else {
		log.Printf("[telegram] invalid channel id: %v", err)
		return
	}

	if cfg.OSS.Enable {
		if ossService, err := oss.NewOSSService(); err == nil {
			listener.ossService = ossService
		} else {
			log.Printf("[telegram] oss init failed: %v", err)
		}
	}

	go listener.run()
}

func parseTelegramChannel(raw string) (int64, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, "", nil
	}
	if strings.HasPrefix(raw, "@") {
		return 0, strings.TrimPrefix(raw, "@"), nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, "", err
}

func (l *telegramListener) run() {
	log.Println("[telegram] listener started")
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := l.bot.GetUpdatesChan(u)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				return
			}
			l.handleUpdate(update)
		case <-ticker.C:
			l.flushGroups()
		}
	}
}

func (l *telegramListener) handleUpdate(update tgbotapi.Update) {
	msg := update.Message
	if msg == nil {
		msg = update.ChannelPost
	}
	if msg == nil || !l.isValidMessage(msg) {
		return
	}

	if msg.MediaGroupID != "" {
		l.collectGroup(msg)
	} else {
		l.processMessage(msg)
	}
}

func (l *telegramListener) isValidMessage(msg *tgbotapi.Message) bool {
	// 剔除贴纸（表情包）
	if msg.Sticker != nil {
		return false
	}

	// 只接受 startTime 之后的消息
	if l.startTime > 0 && int64(msg.Date) < l.startTime {
		return false
	}

	if l.channelID != 0 && msg.Chat.ID != l.channelID {
		return false
	}
	if l.channelUsername != "" && !strings.EqualFold(msg.Chat.UserName, l.channelUsername) {
		return false
	}

	if msg.SenderChat != nil && msg.SenderChat.Type == "channel" {
		return true
	}
	if len(l.filterUserIDs) == 0 {
		return true
	}

	senderID := int64(0)
	if msg.From != nil {
		senderID = msg.From.ID
	} else if msg.SenderChat != nil {
		senderID = msg.SenderChat.ID
	}

	return l.filterUserIDs[senderID]
}

func (l *telegramListener) collectGroup(msg *tgbotapi.Message) {
	group, exists := l.pendingGroups[msg.MediaGroupID]
	if !exists {
		group = &telegramMediaGroup{}
		l.pendingGroups[msg.MediaGroupID] = group
	}
	group.Messages = append(group.Messages, msg)
	group.LastSeen = time.Now()
}

func (l *telegramListener) flushGroups() {
	now := time.Now()
	for id, group := range l.pendingGroups {
		if now.Sub(group.LastSeen) >= 2*time.Second {
			l.processMediaGroup(group.Messages)
			delete(l.pendingGroups, id)
		}
	}
}

func (l *telegramListener) processMediaGroup(msgs []*tgbotapi.Message) {
	if len(msgs) == 0 {
		return
	}

	var content string
	var media []model.MomentMedia

	firstMsg := msgs[0]
	minMsgID := int64(firstMsg.MessageID)
	minDate := firstMsg.Date
	chatID := firstMsg.Chat.ID

	for _, msg := range msgs {
		if int64(msg.MessageID) < minMsgID {
			minMsgID = int64(msg.MessageID)
		}
		if msg.Date < minDate {
			minDate = msg.Date
		}
		if txt := resolveContent(msg); txt != "" && content == "" {
			content = txt
		}
		if m, err := l.downloadAndStore(msg); err == nil && len(m) > 0 {
			media = append(media, m...)
		}
	}

	messageLink := l.buildMessageLink(firstMsg.Chat, int(minMsgID))
	l.saveMoment(chatID, minMsgID, int64(minDate), messageLink, content, media)
}

func (l *telegramListener) processMessage(msg *tgbotapi.Message) {
	media, _ := l.downloadAndStore(msg)
	content := resolveContent(msg)
	messageLink := l.buildMessageLink(msg.Chat, msg.MessageID)
	l.saveMoment(msg.Chat.ID, int64(msg.MessageID), int64(msg.Date), messageLink, content, media)
}

func (l *telegramListener) saveMoment(chatID, msgID, date int64, messageLink, content string, media []model.MomentMedia) {
	if content == "" && len(media) == 0 {
		return
	}

	exists, err := momentRepositories.MomentExistsByChannelMessage(l.db, chatID, msgID)
	if err != nil || exists {
		return
	}

	moment := model.Moment{
		Content:     content,
		Status:      "visible",
		ChannelID:   chatID,
		MessageID:   msgID,
		MessageLink: messageLink,
		CreatedAt:   date,
	}

	if err := momentRepositories.CreateMoment(l.db, &moment, media); err != nil {
		log.Printf("[telegram] create moment failed: %v", err)
	} else {
		log.Printf("[telegram] saved moment chat=%d msg=%d media=%d", chatID, msgID, len(media))
	}
}

func resolveContent(msg *tgbotapi.Message) string {
	if msg.Text != "" {
		return msg.Text
	}
	return msg.Caption
}

func (l *telegramListener) buildMessageLink(chat *tgbotapi.Chat, messageID int) string {
	if chat == nil || messageID == 0 {
		return ""
	}

	username := strings.TrimSpace(chat.UserName)
	if username == "" {
		username = l.channelUsername
	}
	if username != "" {
		return fmt.Sprintf("https://t.me/%s/%d", username, messageID)
	}

	chatID := strconv.FormatInt(chat.ID, 10)
	if strings.HasPrefix(chatID, "-100") {
		trimmed := strings.TrimPrefix(chatID, "-100")
		if trimmed != "" {
			return fmt.Sprintf("https://t.me/c/%s/%d", trimmed, messageID)
		}
	}

	return ""
}

func (l *telegramListener) downloadAndStore(msg *tgbotapi.Message) ([]model.MomentMedia, error) {
	fileID, fileName, mimeType, mediaType := pickMedia(msg)
	if fileID == "" {
		return nil, nil
	}

	file, err := l.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, err
	}
	if file.FilePath == "" {
		return nil, fmt.Errorf("telegram file path is empty")
	}

	resp, err := http.Get(file.Link(l.bot.Token))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if fileName == "" {
		fileName = filepath.Base(file.FilePath)
	}
	fileName, mimeType, err = normalizeTelegramFile(fileName, mimeType, mediaType, data)
	if err != nil {
		return nil, err
	}

	storedURL, isLocal, err := l.storeFile(fileName, mimeType, data)
	if err != nil {
		return nil, err
	}

	return []model.MomentMedia{{
		Name:      fileName,
		MediaURL:  storedURL,
		MediaType: mediaType,
		IsLocal:   isLocal,
	}}, nil
}

func pickMedia(msg *tgbotapi.Message) (id, name, mimeType, mType string) {
	if len(msg.Photo) > 0 {
		best := msg.Photo[0]
		for _, p := range msg.Photo {
			if p.FileSize > best.FileSize {
				best = p
			}
		}
		return best.FileID, "", "image/jpeg", "image"
	}
	if msg.Video != nil {
		return msg.Video.FileID, msg.Video.FileName, msg.Video.MimeType, "video"
	}
	if msg.Animation != nil {
		return msg.Animation.FileID, msg.Animation.FileName, msg.Animation.MimeType, "video"
	}
	if msg.Document != nil {
		mType = "image"
		if strings.HasPrefix(msg.Document.MimeType, "video/") {
			mType = "video"
		}
		return msg.Document.FileID, msg.Document.FileName, msg.Document.MimeType, mType
	}
	return "", "", "", ""
}

func normalizeTelegramFile(fileName, mimeType, mediaType string, data []byte) (string, string, error) {
	contentType := strings.TrimSpace(mimeType)
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	detectedType := http.DetectContentType(data)
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectedType
	}

	if mediaType == "image" && !strings.HasPrefix(contentType, "image/") && !strings.HasPrefix(detectedType, "image/") {
		return "", "", fmt.Errorf("unexpected content type for image: %s", contentType)
	}

	if fileName == "" {
		fileName = "telegram"
	}
	if filepath.Ext(fileName) == "" {
		exts, _ := mime.ExtensionsByType(contentType)
		if len(exts) == 0 && strings.HasPrefix(detectedType, "image/") {
			exts, _ = mime.ExtensionsByType(detectedType)
		}
		if len(exts) > 0 {
			fileName += exts[0]
		}
	}

	return fileName, contentType, nil
}

func (l *telegramListener) storeFile(name, mimeType string, data []byte) (string, int, error) {
	datePath := time.Now().Format("060102")
	finalSubPath := filepath.Join("moments", datePath)
	if l.ossService != nil {
		path := filepath.Join(finalSubPath, name)
		if url, err := UploadToOSS(l.ossService, path, mimeType, data); err == nil {
			return url, 0, nil
		}
	}

	svc := coreService.NewResourceService(config.GetConfig())
	_, url, err := svc.SaveBytes(name, data, finalSubPath, false)
	return url, 1, err
}
