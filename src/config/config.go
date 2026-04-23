package config

import (
	"blog_api/src/model"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"os"

	"github.com/spf13/viper"
)

var (
	globalConfig *model.Config
	once         sync.Once
	v            *viper.Viper // 全局 viper 实例
)

func Load() (*model.Config, error) {
	var err error
	once.Do(func() {
		globalConfig, err = loadConfig()
	})
	return globalConfig, err
}

func GetConfig() *model.Config {
	if globalConfig == nil {
		log.Fatal("配置未初始化,请先调用 Load()")
	}
	return globalConfig
}

func loadConfig() (*model.Config, error) {
	v = viper.New()
	v.SetDefault("CRON_SCAN_ON_STARTUP", true)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()                                   // 自动读取匹配的环境变量
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // 将配置键中的'.'替换为'_'以匹配环境变量

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("未找到 .env 文件，将跳过加载")
		} else {
			return nil, fmt.Errorf("解析 .env 文件时发生错误: %w", err)
		}
	}

	configPath := v.GetString("CONFIG_PATH")
	if configPath == "" {
		configPath = "data/config"
	}
	if err := mergeJSONConfig("system_config", configPath); err != nil {
		return nil, err
	}
	if err := mergeJSONConfig("friend_list", configPath); err != nil {
		return nil, err
	}
	cfg := &model.Config{}
	if err := unmarshalConfig(cfg); err != nil {
		return nil, fmt.Errorf("解析配置到结构体失败: %w", err)
	}

	return cfg, nil
}

// mergeJSONConfig 合并指定的 JSON 配置文件
func mergeJSONConfig(configName, configPath string) error {
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
func unmarshalConfig(cfg *model.Config) error {
	cfg.Port = v.GetString("PORT")
	cfg.ListenAddress = v.GetString("LISTEN_ADDRESS")
	cfg.WebPanelUser = v.GetString("WEB_PANEL_USER")
	cfg.WebPanelPwd = v.GetString("WEB_PANEL_PWD")
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

// UpdateAndSaveConfigs 批量更新并保存配置到 system_config.json
func UpdateAndSaveConfigs(updates []model.UpdateConfigReq) error {
	configPath := GetConfig().ConfigPath
	if configPath == "" {
		return fmt.Errorf("配置路径未设置")
	}
	filePath := filepath.Join(configPath, "system_config.json")

	existingData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			existingData = []byte("{}")
		} else {
			return fmt.Errorf("读取现有配置文件失败: %w", err)
		}
	}
	var existingConfig map[string]interface{}
	if err := json.Unmarshal(existingData, &existingConfig); err != nil {
		return fmt.Errorf("解析现有配置失败: %w", err)
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
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("写入 system_config.json 失败: %w", err)
	}

	return nil
}
