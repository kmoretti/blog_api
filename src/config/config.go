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

// Load 加载所有配置 (单例模式)
// 配置加载顺序:
// 环境变量会覆盖配置文件中的同名设置
func Load() (*model.Config, error) {
	var err error
	once.Do(func() {
		globalConfig, err = loadConfig()
	})
	return globalConfig, err
}

// GetConfig 获取全局配置实例
func GetConfig() *model.Config {
	if globalConfig == nil {
		log.Fatal("配置未初始化,请先调用 Load()")
	}
	return globalConfig
}

// loadConfig 执行实际的配置加载
func loadConfig() (*model.Config, error) {
	// 初始化全局 viper 实例
	v = viper.New()

	// 设置默认值
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

	// 2. 获取配置路径 (从环境变量或使用默认值)
	configPath := v.GetString("CONFIG_PATH")
	if configPath == "" {
		configPath = "data/config"
	}

	// 3. 合并 system_config.json
	if err := mergeJSONConfig("system_config", configPath); err != nil {
		return nil, err
	}

	// 4. 合并 friend_list.json
	if err := mergeJSONConfig("friend_list", configPath); err != nil {
		return nil, err
	}

	// 5. 解析配置到结构体
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
	// 解析环境变量
	cfg.Port = v.GetString("PORT")
	cfg.ListenAddress = v.GetString("LISTEN_ADDRESS")
	cfg.WebPanelUser = v.GetString("WEB_PANEL_USER")
	cfg.WebPanelPwd = v.GetString("WEB_PANEL_PWD")
	cfg.ConfigPath = v.GetString("CONFIG_PATH")
	cfg.CronScanOnStartup = v.GetBool("CRON_SCAN_ON_STARTUP")
	cfg.IsDev = parseEnvBool(v.GetString("IS_DEV"))

	// 设置默认值
	if cfg.Port == "" {
		cfg.Port = "10024"
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = "0.0.0.0"
	}
	if cfg.ConfigPath == "" {
		cfg.ConfigPath = "data/config"
	}

	// 解析系统配置
	if err := v.UnmarshalKey("system_conf", cfg); err != nil {
		return fmt.Errorf("解析系统配置失败: %w", err)
	}

	// 动态地将核心数据路径添加到排除列表，以防止被意外删除
	if cfg.Data.Database.Path != "" {
		cfg.Safe.ExcludePaths = append(cfg.Safe.ExcludePaths, cfg.Data.Database.Path)
	}

	// 设置爬虫默认并发数
	if cfg.Crawler.Concurrency <= 0 {
		cfg.Crawler.Concurrency = 5
	}
	if cfg.Crawler.RssTimeoutSeconds <= 0 {
		cfg.Crawler.RssTimeoutSeconds = 15
	}

	// 从环境变量加载覆盖敏感信息
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

	// 手动解析友链配置
	friendListPath := filepath.Join(cfg.ConfigPath, "friend_list.json")
	friendListData, err := os.ReadFile(friendListPath)
	if err != nil {
		log.Printf("无法读取 friend_list.json 文件: %v, 将跳过加载友链", err)
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

// Reload 重新加载配置 (用于热更新)
func Reload() error {
	once = sync.Once{} // 重置 once，允许重新加载
	newConfig, err := loadConfig()
	if err != nil {
		return err
	}
	globalConfig = newConfig
	log.Println("配置已重新加载")
	return nil
}

// UpdateAndSaveConfigs 批量更新并保存配置到 system_config.json
func UpdateAndSaveConfigs(updates []model.UpdateConfigReq) error {
	configPath := GetConfig().ConfigPath
	if configPath == "" {
		return fmt.Errorf("配置路径未设置")
	}
	filePath := filepath.Join(configPath, "system_config.json")

	// 读取现有的配置文件
	existingData, err := os.ReadFile(filePath)
	if err != nil {
		// 如果文件不存在，创建一个空的配置
		if os.IsNotExist(err) {
			existingData = []byte("{}")
		} else {
			return fmt.Errorf("读取现有配置文件失败: %w", err)
		}
	}

	// 解析现有配置
	var existingConfig map[string]interface{}
	if err := json.Unmarshal(existingData, &existingConfig); err != nil {
		return fmt.Errorf("解析现有配置失败: %w", err)
	}

	// 批量更新指定的配置项
	for _, update := range updates {
		// 检查 key 是否以 "system_conf." 开头
		if !strings.HasPrefix(update.Key, "system_conf.") {
			log.Printf("跳过不支持的配置键: %s", update.Key)
			continue
		}

		// 将 key 拆分为路径
		keys := strings.Split(update.Key, ".")
		if len(keys) < 2 {
			log.Printf("跳过无效的配置键: %s", update.Key)
			continue
		}

		// 导航到需要更新的位置
		current := existingConfig
		for i := 0; i < len(keys)-1; i++ {
			// 确保路径中的每个节点都是 map[string]interface{}
			if _, ok := current[keys[i]]; !ok {
				current[keys[i]] = make(map[string]interface{})
			}
			if next, ok := current[keys[i]].(map[string]interface{}); ok {
				current = next
			} else {
				// 如果路径中的某个节点不是 map，这可能是个问题，但我们尝试创建它
				newMap := make(map[string]interface{})
				current[keys[i]] = newMap
				current = newMap
			}
		}
		// 设置最终的值
		current[keys[len(keys)-1]] = update.Value
	}

	// 序列化并写入文件
	jsonData, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("写入 system_config.json 失败: %w", err)
	}
	return Reload()
}
