package model

// Config 全局配置结构 - 支持点号访问
type Config struct {
	// 环境变量配置
	Port                   string
	ListenAddress          string
	WebPanelUser           string
	WebPanelPwd            string
	StateAPIMasterPassword string
	ConfigPath             string
	CronScanOnStartup      bool
	IsDev                  bool

	// 系统配置 - 使用小写字段名，通过 Safe 和 Data 访问
	Safe              SafeConfig              `mapstructure:"safe_conf"`
	Data              DataConfig              `mapstructure:"data_conf"`
	Crawler           CrawlerConfig           `mapstructure:"crawler_conf"`
	MomentsIntegrated MomentsIntegratedConfig `mapstructure:"moments_integrated_conf"`
	OSS               OSSConfig               `mapstructure:"oss_conf"`
	Verify            VerifyConfig            `mapstructure:"verify_conf"`
	Email             EmailConf               `mapstructure:"email_conf"`

	// 友链配置
	FriendLinks []FriendWebsite
}

// CrawlerConfig 爬虫配置
type CrawlerConfig struct {
	Concurrency       int `mapstructure:"concurrency"`         // 并发数量，默认 5
	RssTimeoutSeconds int `mapstructure:"rss_timeout_seconds"` // RSS 解析超时（秒）
}

// FriendLinksConf 对应 friend_list.json 的结构
type FriendLinksConf struct {
	FriendLinksData struct {
		Website []FriendWebsite `json:"website"`
	} `json:"friend_links_conf"`
}

// SafeConfig 安全配置
type SafeConfig struct {
	CorsAllowHostlist []string `mapstructure:"cors_allow_hostlist"`
	ExcludePaths      []string `mapstructure:"exclude_paths"`
	AllowExtension    []string `mapstructure:"allow_extension"`
}

// DataConfig 数据配置
type DataConfig struct {
	Database DatabaseConfig `mapstructure:"database"`
	Image    ImageConfig    `mapstructure:"image"`
	Resource ResourceConfig `mapstructure:"resource"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// ImageConfig 图片配置
type ImageConfig struct {
	Path   string `mapstructure:"path"`
	ConvTo string `mapstructure:"conv_to"`
}

// ResourceConfig 资源配置
type ResourceConfig struct {
	Path string `mapstructure:"path"`
}

// MomentsIntegratedConfig 动态集成配置
type MomentsIntegratedConfig struct {
	Enable                 bool              `mapstructure:"enable"`
	ApiSingleReturnEntries int               `mapstructure:"api_single_return_entries"`
	Integrated             IntegratedTargets `mapstructure:"integrated"`
}

// OSSConfig OSS 配置
type OSSConfig struct {
	Provider        string `mapstructure:"provider"`
	Enable          bool   `mapstructure:"enable"`
	AccessKeyID     string `mapstructure:"accessKeyId"`
	AccessKeySecret string `mapstructure:"accessKeySecret"`
	Bucket          string `mapstructure:"bucket"`
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	CustomDomain    string `mapstructure:"customDomain"`
	Secure          bool   `mapstructure:"secure"`
	Timeout         int    `mapstructure:"timeout"`
	Prefix          string `mapstructure:"prefix"`
}

// VerifyConfig 验证配置
type VerifyConfig struct {
	Turnstile   TurnstileConfig   `mapstructure:"turnstile"`
	Fingerprint FingerprintConfig `mapstructure:"fingerprint"`
}

// TurnstileConfig Turnstile 配置
type TurnstileConfig struct {
	Enable  bool   `mapstructure:"enable"`
	Secret  string `mapstructure:"secret"`
	SiteKey string `mapstructure:"site_key"`
}

// FingerprintConfig 指纹配置
type FingerprintConfig struct {
	Secret string `mapstructure:"secret"`
}

// IntegratedTargets 集成目标
type IntegratedTargets struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
	Discord  DiscordConfig  `mapstructure:"discord"`
}

// TelegramConfig Telegram 配置
type TelegramConfig struct {
	Enable       bool     `mapstructure:"enable"`
	SyncDelete   bool     `mapstructure:"sync_delete"`
	BotToken     string   `mapstructure:"bot_token"`
	ChannelID    string   `mapstructure:"channel_id"`
	FilterUserid []string `mapstructure:"filter_userid"`
	StartTime    int64    `mapstructure:"start_time"` // Unix timestamp, 只接受此时间之后的消息（0 表示使用 bot 启动时间）
}

// EmailConf 邮箱配置
type EmailConf struct {
	Enable   bool   `mapstructure:"enable"`
	Host     string `mapstructure:"host"`
	UserName string `mapstructure:"user_name"`
	Password string `mapstructure:"password"`
	Port     int    `mapstructure:"port"`
	Sender   string `mapstructure:"sender"`
}

// PwaConfig PWA 配置
type PwaConfig struct {
	Enable bool `mapstructure:"enable"`
}

// DiscordConfig Discord 配置
type DiscordConfig struct {
	Enable       bool     `mapstructure:"enable"`
	SyncDelete   bool     `mapstructure:"sync_delete"`
	BotToken     string   `mapstructure:"bot_token"`
	GuildID      string   `mapstructure:"guild_id"`
	ChannelID    string   `mapstructure:"channel_id"`
	FilterUserid []string `mapstructure:"filter_userid"`
}
