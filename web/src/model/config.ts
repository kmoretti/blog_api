export interface SystemConfig {
  system_conf: {
    safe_conf: SafeConfig;
    data_conf: DataConfig;
    crawler_conf: CrawlerConfig;
    moments_integrated_conf: MomentsIntegratedConfig;
    oss_conf: OSSConfig;
    verify_conf: VerifyConfig;
    email_conf: EmailConfig;
    pwa_conf: PwaConfig;
  };
}

export interface SafeConfig {
  cors_allow_hostlist: string[];
  exclude_paths: string[];
  allow_extension: string[];
}

export interface DataConfig {
  database: DatabaseConfig;
  image: ImageConfig;
  resource: ResourceConfig;
}

export interface DatabaseConfig {
  path: string;
}

export interface ImageConfig {
  path: string;
  conv_to: string;
}

export interface ResourceConfig {
  path: string;
}

export interface CrawlerConfig {
  concurrency: number;
  rss_timeout_seconds: number;
}

export interface MomentsIntegratedConfig {
  enable: boolean;
  api_single_return_entries: number;
  integrated: IntegratedTargets;
}

export interface OSSConfig {
  provider: string;
  enable: boolean;
  accessKeyId: string;
  accessKeySecret: string;
  bucket: string;
  endpoint: string;
  region: string;
  secure: boolean;
  timeout: number;
  prefix: string;
  customDomain: string;
 }

export interface VerifyConfig {
  turnstile: TurnstileConfig;
  fingerprint: FingerprintConfig;
}

export interface TurnstileConfig {
  enable: boolean;
  secret: string;
  site_key: string;
}

export interface FingerprintConfig {
  secret: string;
}

export interface EmailConfig {
  enable: boolean;
  host: string;
  user_name: string;
  password: string;
  port: number;
  sender: string;
}

export interface IntegratedTargets {
  telegram: TelegramConfig;
  discord: DiscordConfig;
}

export interface TelegramConfig {
  enable: boolean;
  sync_delete: boolean;
  bot_token: string;
  channel_id: string;
  media_path: string;
  filter_userid: string[];
}

export interface DiscordConfig {
  enable: boolean;
  sync_delete: boolean;
  bot_token: string;
  guild_id: string;
  channel_id: string;
  filter_userid: string[];
}

export interface PwaConfig {
  enable: boolean;
}
