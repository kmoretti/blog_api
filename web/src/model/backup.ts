export interface ImportResult {
  imported: number
  skipped: number
  replaced: number
}

export interface RestoreResult {
  backup_path: string
  notice: string
}

export type BackupModule =
  | 'system_config'
  | 'friend_links'
  | 'moments'
  | 'friend_rss'
  | 'images'

export type ImportStrategy = 'replace' | 'skip'

export interface ModuleOption {
  key: BackupModule
  label: string
}

export const BACKUP_MODULES: ModuleOption[] = [
  { key: 'system_config', label: '系统配置' },
  { key: 'friend_links', label: '友链' },
  { key: 'moments', label: '动态' },
  { key: 'friend_rss', label: 'RSS' },
  { key: 'images', label: '图片清单' },
]
