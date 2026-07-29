package config

import (
	"blog_api/src/model"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/spf13/viper"
)

//go:embed defaults/*.json
var defaultConfigFiles embed.FS

var (
	currentConfig atomic.Pointer[model.Config]
	loadErr       error
	once          sync.Once
	reloadMu      sync.Mutex
	updateMu      sync.Mutex
)

// UpdateResult describes the runtime effects of a persisted config update.
//
// Reloaded is true when the in-memory snapshot was atomically replaced.
// RestartRequiredKeys contains accepted update keys whose full effect still
// depends on startup-only resources such as listeners, cron jobs, or clients.
type UpdateResult struct {
	Reloaded            bool     `json:"reloaded"`
	RestartRequiredKeys []string `json:"restart_required_keys"`
}

// Load initializes the global configuration snapshot once.
//
// It releases missing default JSON files, reads .env and JSON config files,
// applies environment overrides, and publishes the resulting immutable snapshot
// for GetConfig. After the first successful call, use Reload to publish a new
// snapshot.
func Load() (*model.Config, error) {
	once.Do(func() {
		reloadMu.Lock()
		defer reloadMu.Unlock()

		var cfg *model.Config
		cfg, loadErr = loadConfig()
		if loadErr == nil {
			currentConfig.Store(cfg)
		}
	})
	reloadMu.Lock()
	defer reloadMu.Unlock()
	return currentConfig.Load(), loadErr
}

// GetConfig returns the currently published configuration snapshot.
//
// Callers must treat the returned Config as read-only. A later Reload can
// atomically publish a different snapshot, but the returned pointer remains
// valid for the caller that already acquired it.
func GetConfig() *model.Config {
	cfg := currentConfig.Load()
	if cfg == nil {
		log.Fatal("配置未初始化,请先调用 Load()")
	}
	return cfg
}

// ReplaceConfig swaps the published configuration snapshot and returns a
// restore function. Intended for tests that need a temporary config.
func ReplaceConfig(cfg *model.Config) func() {
	previous := currentConfig.Swap(cfg)
	return func() {
		currentConfig.Store(previous)
	}
}

// Reload reads config files again and atomically publishes the new snapshot.
//
// If loading fails, the previous snapshot remains active and the error is
// returned to the caller.
func Reload() (*model.Config, error) {
	reloadMu.Lock()
	defer reloadMu.Unlock()

	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	currentConfig.Store(cfg)
	loadErr = nil
	return cfg, nil
}

func loadConfig() (*model.Config, error) {
	v := newConfigReader()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, os.ErrNotExist) {
			log.Printf("未找到 .env 文件，将跳过加载")
		} else {
			return nil, fmt.Errorf("解析 .env 文件时发生错误: %w", err)
		}
	}

	configPath := v.GetString("CONFIG_PATH")
	if configPath == "" {
		configPath = "data/config"
	}
	if err := ensureDefaultConfigFiles(configPath); err != nil {
		return nil, err
	}
	if err := mergeJSONConfig(v, "system_config", configPath); err != nil {
		return nil, err
	}
	if err := mergeJSONConfig(v, "friend_list", configPath); err != nil {
		return nil, err
	}
	cfg := &model.Config{}
	if err := unmarshalConfig(v, cfg); err != nil {
		return nil, fmt.Errorf("解析配置到结构体失败: %w", err)
	}

	return cfg, nil
}

func newConfigReader() *viper.Viper {
	v := viper.New()
	v.SetDefault("CRON_SCAN_ON_STARTUP", true)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	return v
}

func ensureDefaultConfigFiles(configPath string) error {
	if err := os.MkdirAll(configPath, 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	defaults := []string{"system_config.json", "friend_list.json"}
	for _, name := range defaults {
		content, err := fs.ReadFile(defaultConfigFiles, "defaults/"+name)
		if err != nil {
			return fmt.Errorf("读取内嵌配置 %s 失败: %w", name, err)
		}

		path := filepath.Join(configPath, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("创建默认配置 %s 失败: %w", path, err)
		}

		if _, err := file.Write(content); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return fmt.Errorf("写入默认配置 %s 失败: %w", path, err)
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(path)
			return fmt.Errorf("关闭默认配置 %s 失败: %w", path, err)
		}
		log.Printf("[config]已释放默认配置: %s", path)
	}
	return nil
}

// mergeJSONConfig 合并指定的 JSON 配置文件
func mergeJSONConfig(v *viper.Viper, configName, configPath string) error {
	v.SetConfigName(configName)
	v.SetConfigType("json")
	v.AddConfigPath(configPath)

	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("未找到配置文件 (%s/%s.json)，将跳过合并", configPath, configName)
			return nil
		}
		return fmt.Errorf("合并配置文件 %s 时发生错误: %w", configName, err)
	}
	return nil
}

// unmarshalConfig 将 viper 配置解析到 Config 结构体
func unmarshalConfig(v *viper.Viper, cfg *model.Config) error {
	cfg.Port = v.GetString("PORT")
	cfg.ListenAddress = v.GetString("LISTEN_ADDRESS")
	cfg.WebPanelUser = v.GetString("WEB_PANEL_USER")
	cfg.WebPanelPwd = v.GetString("WEB_PANEL_PWD")
	cfg.StateAPIMasterPassword = v.GetString("STATE_API_MASTER_PASSWORD")
	cfg.ConfigPath = v.GetString("CONFIG_PATH")
	cfg.CronScanOnStartup = v.GetBool("CRON_SCAN_ON_STARTUP")
	cfg.IsDev = parseEnvBool(v.GetString("IS_DEV"))

	if cfg.Port == "" {
		cfg.Port = "10024"
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = "0.0.0.0"
	}
	if cfg.ConfigPath == "" {
		cfg.ConfigPath = "data/config"
	}

	if err := v.UnmarshalKey("system_conf", cfg); err != nil {
		return fmt.Errorf("解析系统配置失败: %w", err)
	}

	// 友链邮件通知开关默认值（兼容旧配置，避免零值导致通知被静默关闭）
	if !v.IsSet("system_conf.email_conf.friend_link_admin_notify") {
		cfg.Email.FriendLinkAdminNotify = true
	}
	if !v.IsSet("system_conf.email_conf.friend_link_user_notify") {
		cfg.Email.FriendLinkUserNotify = false
	}

	if cfg.Data.Database.Path != "" {
		cfg.Safe.ExcludePaths = append(cfg.Safe.ExcludePaths, cfg.Data.Database.Path)
	}

	if cfg.Crawler.Concurrency <= 0 {
		cfg.Crawler.Concurrency = 5
	}
	if cfg.Crawler.RssTimeoutSeconds <= 0 {
		cfg.Crawler.RssTimeoutSeconds = 15
	}

	if telegramBotToken := v.GetString("TELEGRAM_BOT_TOKEN"); telegramBotToken != "" {
		cfg.MomentsIntegrated.Integrated.Telegram.BotToken = telegramBotToken
	}
	if discordBotToken := v.GetString("DISCORD_BOT_TOKEN"); discordBotToken != "" {
		cfg.MomentsIntegrated.Integrated.Discord.BotToken = discordBotToken
	}
	if ossAccessKeyId := v.GetString("OSS_ACCESS_KEY_ID"); ossAccessKeyId != "" {
		cfg.OSS.AccessKeyID = ossAccessKeyId
	}
	if ossAccessKeySecret := v.GetString("OSS_ACCESS_KEY_SECRET"); ossAccessKeySecret != "" {
		cfg.OSS.AccessKeySecret = ossAccessKeySecret
	}
	if turnstileSecret := v.GetString("TURNSTILE_SECRET"); turnstileSecret != "" {
		cfg.Verify.Turnstile.Secret = turnstileSecret
	}
	if fingerprintSecret := v.GetString("FINGERPRINT_SECRET"); fingerprintSecret != "" {
		cfg.Verify.Fingerprint.Secret = fingerprintSecret
	}
	if emailPassword := v.GetString("EMAIL_PASSWORD"); emailPassword != "" {
		cfg.Email.Password = emailPassword
	}

	log.Printf("[config] email enable=%t, admin_notify=%t, user_notify=%t",
		cfg.Email.Enable, cfg.Email.FriendLinkAdminNotify, cfg.Email.FriendLinkUserNotify)

	friendListPath := filepath.Join(cfg.ConfigPath, "friend_list.json")
	friendListData, err := os.ReadFile(friendListPath)
	if err != nil {
		log.Printf("[config]无法读取 friend_list.json 文件: %v, 将跳过加载友链", err)
	} else {
		var friendLinksConf model.FriendLinksConf
		if err := json.Unmarshal(friendListData, &friendLinksConf); err != nil {
			return fmt.Errorf("解析 friend_list.json 文件失败: %w", err)
		}
		cfg.FriendLinks = friendLinksConf.FriendLinksData.Website
	}

	return nil
}

func parseEnvBool(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "yes", "y", "on", "ture":
		return true
	default:
		return false
	}
}

// UpdateAndSaveConfigs persists supported config updates and reloads config.
//
// Only keys under system_conf are accepted. Unsupported or invalid keys are
// skipped for backward compatibility. On write success, the function reloads
// the global snapshot. If reload fails, the previous snapshot remains active and
// the previous file content is restored when possible.
func UpdateAndSaveConfigs(updates []model.UpdateConfigReq) (*UpdateResult, error) {
	updateMu.Lock()
	defer updateMu.Unlock()

	configPath := GetConfig().ConfigPath
	if configPath == "" {
		return nil, fmt.Errorf("配置路径未设置")
	}
	filePath := filepath.Join(configPath, "system_config.json")

	existingData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			existingData = []byte("{}")
		} else {
			return nil, fmt.Errorf("读取现有配置文件失败: %w", err)
		}
	}
	var existingConfig map[string]interface{}
	if err := json.Unmarshal(existingData, &existingConfig); err != nil {
		return nil, fmt.Errorf("解析现有配置失败: %w", err)
	}
	for _, update := range updates {
		if !strings.HasPrefix(update.Key, "system_conf.") {
			log.Printf("跳过不支持的配置键: %s", update.Key)
			continue
		}
		keys := strings.Split(update.Key, ".")
		if len(keys) < 2 {
			log.Printf("跳过无效的配置键: %s", update.Key)
			continue
		}
		current := existingConfig
		for i := 0; i < len(keys)-1; i++ {
			if _, ok := current[keys[i]]; !ok {
				current[keys[i]] = make(map[string]interface{})
			}
			if next, ok := current[keys[i]].(map[string]interface{}); ok {
				current = next
			} else {
				newMap := make(map[string]interface{})
				current[keys[i]] = newMap
				current = newMap
			}
		}
		current[keys[len(keys)-1]] = update.Value
	}
	jsonData, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := writeFileAtomic(filePath, jsonData, 0o644); err != nil {
		return nil, fmt.Errorf("写入 system_config.json 失败: %w", err)
	}

	if _, err := Reload(); err != nil {
		if restoreErr := writeFileAtomic(filePath, existingData, 0o644); restoreErr != nil {
			return nil, fmt.Errorf("刷新配置失败: %w; 恢复旧配置失败: %v", err, restoreErr)
		}
		return nil, fmt.Errorf("刷新配置失败: %w", err)
	}

	return &UpdateResult{
		Reloaded:            true,
		RestartRequiredKeys: restartRequiredKeys(updates),
	}, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".system_config-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}

func restartRequiredKeys(updates []model.UpdateConfigReq) []string {
	restartPrefixes := []string{
		"system_conf.data_conf.database.path",
		"system_conf.data_conf.image.path",
		"system_conf.data_conf.image.conv_to",
		"system_conf.moments_integrated_conf",
		"system_conf.oss_conf",
	}

	seen := make(map[string]bool, len(updates))
	keys := make([]string, 0)
	for _, update := range updates {
		for _, prefix := range restartPrefixes {
			if update.Key == prefix || strings.HasPrefix(update.Key, prefix+".") {
				if !seen[update.Key] {
					seen[update.Key] = true
					keys = append(keys, update.Key)
				}
				break
			}
		}
	}
	return keys
}
