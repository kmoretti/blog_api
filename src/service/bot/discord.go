package bot

import (
	"blog_api/src/config"
	"blog_api/src/model"
	momentRepositories "blog_api/src/repositories/moment"
	coreService "blog_api/src/service"
	"blog_api/src/service/oss"
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type discordListener struct {
	db            *gorm.DB
	session       *discordgo.Session
	channelID     string
	filterUserIDs map[string]bool
	ossService    oss.OSSService
	syncDelete    bool
}

func StartDiscordListener(db *gorm.DB, cfg *model.Config) {
	dCfg := cfg.MomentsIntegrated.Integrated.Discord
	if !cfg.MomentsIntegrated.Enable || !dCfg.Enable || dCfg.BotToken == "" {
		return
	}

	session, err := discordgo.New("Bot " + dCfg.BotToken)
	if err != nil {
		log.Printf("[discord] init session failed: %v", err)
		return
	}
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	listener := &discordListener{
		db:            db,
		session:       session,
		channelID:     strings.TrimSpace(dCfg.ChannelID),
		filterUserIDs: make(map[string]bool),
		syncDelete:    dCfg.SyncDelete,
	}

	for _, id := range dCfg.FilterUserid {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			listener.filterUserIDs[trimmed] = true
		}
	}

	if cfg.OSS.Enable {
		if ossService, err := oss.NewOSSService(); err == nil {
			listener.ossService = ossService
		} else {
			log.Printf("[discord] oss init failed: %v", err)
		}
	}

	session.AddHandler(listener.onMessageCreate)
	session.AddHandler(listener.onMessageDelete)
	session.AddHandler(listener.onMessageDeleteBulk)
	if err := session.Open(); err != nil {
		log.Printf("[discord] open session failed: %v", err)
		return
	}
	SetDiscordSession(session)

	_ = session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: string(discordgo.StatusOnline),
	})

	log.Println("[discord] listener started")
}

func (l *discordListener) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil || m.Message == nil || m.Author == nil {
		return
	}
	if s.State != nil && s.State.User != nil && m.Author.ID == s.State.User.ID {
		return
	}
	if l.channelID != "" && m.ChannelID != l.channelID {
		return
	}
	if len(l.filterUserIDs) > 0 && !l.filterUserIDs[m.Author.ID] {
		return
	}

	channelID, _ := parseDiscordID(m.ChannelID)
	messageID, _ := parseDiscordID(m.ID)
	var guildID int64
	if m.GuildID != "" {
		if parsed, err := parseDiscordID(m.GuildID); err == nil {
			guildID = parsed
		}
	}

	media := l.downloadAttachments(m.Attachments)
	messageLink := buildDiscordMessageLink(m.GuildID, m.ChannelID, m.ID)
	l.saveMoment(guildID, channelID, messageID, m.Timestamp.Unix(), messageLink, m.Content, media)
}

func (l *discordListener) onMessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	if !l.syncDelete || e == nil {
		return
	}
	if l.channelID != "" && e.ChannelID != l.channelID {
		return
	}

	channelID, err := parseDiscordID(e.ChannelID)
	if err != nil {
		return
	}
	messageID, err := parseDiscordID(e.ID)
	if err != nil {
		return
	}

	_ = momentRepositories.DeleteMomentByChannelMessage(l.db, channelID, messageID)
}

func (l *discordListener) onMessageDeleteBulk(s *discordgo.Session, e *discordgo.MessageDeleteBulk) {
	if !l.syncDelete || e == nil {
		return
	}
	if l.channelID != "" && e.ChannelID != l.channelID {
		return
	}

	channelID, err := parseDiscordID(e.ChannelID)
	if err != nil {
		return
	}
	for _, id := range e.Messages {
		if messageID, err := parseDiscordID(id); err == nil {
			_ = momentRepositories.DeleteMomentByChannelMessage(l.db, channelID, messageID)
		}
	}
}

func (l *discordListener) downloadAttachments(attachments []*discordgo.MessageAttachment) []model.MomentMedia {
	if len(attachments) == 0 {
		return nil
	}

	var media []model.MomentMedia
	for _, att := range attachments {
		mediaType := detectDiscordMediaType(att)
		if mediaType == "" {
			continue
		}

		if item, err := l.downloadAttachment(att, mediaType); err == nil && item != nil {
			media = append(media, *item)
		}
	}
	return media
}

func detectDiscordMediaType(att *discordgo.MessageAttachment) string {
	if att == nil {
		return ""
	}
	contentType := strings.TrimSpace(att.ContentType)
	if strings.HasPrefix(contentType, "image/") {
		return "image"
	}
	if strings.HasPrefix(contentType, "video/") {
		return "video"
	}

	switch strings.ToLower(filepath.Ext(att.Filename)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return "image"
	case ".mp4", ".webm":
		return "video"
	default:
		return ""
	}
}

func (l *discordListener) downloadAttachment(att *discordgo.MessageAttachment, mediaType string) (*model.MomentMedia, error) {
	if att == nil || att.URL == "" {
		return nil, nil
	}

	file, size, sample, err := downloadToTemp(context.Background(), att.URL)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	defer os.Remove(file.Name())

	fileName, contentType, err := normalizeDiscordFile(att.Filename, att.ContentType, mediaType, sample)
	if err != nil {
		return nil, err
	}

	storedURL, isLocal, err := l.storeFile(fileName, contentType, file, size)
	if err != nil {
		return nil, err
	}

	return &model.MomentMedia{
		Name:      fileName,
		MediaURL:  storedURL,
		MediaType: mediaType,
		IsLocal:   isLocal,
	}, nil
}

func normalizeDiscordFile(fileName, mimeType, mediaType string, sample []byte) (string, string, error) {
	contentType := strings.TrimSpace(mimeType)
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	detectedType := http.DetectContentType(sample)
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectedType
	}

	if mediaType == "image" && !strings.HasPrefix(contentType, "image/") && !strings.HasPrefix(detectedType, "image/") {
		return "", "", fmt.Errorf("unexpected content type for image: %s", contentType)
	}
	if mediaType == "video" && !strings.HasPrefix(contentType, "video/") && !strings.HasPrefix(detectedType, "video/") {
		return "", "", fmt.Errorf("unexpected content type for video: %s", contentType)
	}

	if fileName == "" {
		fileName = "discord"
	}
	if filepath.Ext(fileName) == "" {
		exts, _ := mime.ExtensionsByType(contentType)
		if len(exts) == 0 {
			exts, _ = mime.ExtensionsByType(detectedType)
		}
		if len(exts) > 0 {
			fileName += exts[0]
		}
	}

	return fileName, contentType, nil
}

func (l *discordListener) storeFile(name, mimeType string, file *os.File, size int64) (string, int, error) {
	datePath := time.Now().Format("060102")
	finalSubPath := filepath.Join("moments", datePath)
	if l.ossService != nil {
		path := filepath.Join(finalSubPath, name)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", 0, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		url, err := UploadToOSS(ctx, l.ossService, path, mimeType, size, file)
		cancel()
		if err == nil {
			return url, 0, nil
		}
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}
	svc := coreService.NewResourceService(config.GetConfig())
	_, url, err := svc.SaveReader(name, file, finalSubPath, false)
	return url, 1, err
}

func (l *discordListener) saveMoment(guildID, channelID, msgID, date int64, messageLink, content string, media []model.MomentMedia) {
	if content == "" && len(media) == 0 {
		return
	}

	exists, err := momentRepositories.MomentExistsByChannelMessage(l.db, channelID, msgID)
	if err != nil || exists {
		return
	}

	moment := model.Moment{
		Content:     content,
		Status:      "visible",
		GuildID:     guildID,
		ChannelID:   channelID,
		MessageID:   msgID,
		MessageLink: messageLink,
		CreatedAt:   date,
	}

	if err := momentRepositories.CreateMoment(l.db, &moment, media); err != nil {
		log.Printf("[discord] create moment failed: %v", err)
	} else {
		log.Printf("[discord] saved moment channel=%d msg=%d media=%d", channelID, msgID, len(media))
	}
}

func parseDiscordID(raw string) (int64, error) {
	if raw = strings.TrimSpace(raw); raw == "" {
		return 0, fmt.Errorf("empty id")
	}
	return strconv.ParseInt(raw, 10, 64)
}

func buildDiscordMessageLink(guildID, channelID, messageID string) string {
	channelID = strings.TrimSpace(channelID)
	messageID = strings.TrimSpace(messageID)
	if channelID == "" || messageID == "" {
		return ""
	}
	if guildID = strings.TrimSpace(guildID); guildID == "" {
		guildID = "@me"
	}
	return fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, channelID, messageID)
}
