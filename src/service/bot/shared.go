package bot

import (
	"blog_api/src/service/oss"
	"context"
	"io"
	"sync"

	"github.com/bwmarrin/discordgo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	sharedMu        sync.RWMutex
	discordSession  *discordgo.Session
	telegramSession *tgbotapi.BotAPI
)

// SetDiscordSession stores the shared Discord session for reuse.
func SetDiscordSession(session *discordgo.Session) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	discordSession = session
}

// GetDiscordSession returns the shared Discord session.
func GetDiscordSession() *discordgo.Session {
	sharedMu.RLock()
	defer sharedMu.RUnlock()
	return discordSession
}

// SetTelegramBot stores the shared Telegram bot for reuse.
func SetTelegramBot(bot *tgbotapi.BotAPI) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	telegramSession = bot
}

// GetTelegramBot returns the shared Telegram bot.
func GetTelegramBot() *tgbotapi.BotAPI {
	sharedMu.RLock()
	defer sharedMu.RUnlock()
	return telegramSession
}

// UploadToOSS uploads a replayable stream without taking ownership of body.
func UploadToOSS(ctx context.Context, svc oss.OSSService, name, mimeType string, size int64, body io.ReadSeeker) (string, error) {
	url, _, err := svc.Upload(ctx, oss.UploadInput{
		Name:        name,
		ContentType: mimeType,
		Size:        size,
		Body:        body,
	})
	return url, err
}
