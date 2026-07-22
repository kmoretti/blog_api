<template>
  <div class="settings-container">
    <el-card class="settings-card">
      <template #header>
        <div class="card-header">
          <span>系统设置</span>
          <el-button type="primary" @click="saveConfig" :loading="saving">保存配置</el-button>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <!-- 安全配置 -->
        <el-tab-pane label="安全配置" name="safe">
          <el-form :model="config" label-width="150px">
            <el-form-item label="CORS 白名单">
              <el-tag
                v-for="(host, index) in config.system_conf.safe_conf.cors_allow_hostlist"
                :key="index"
                closable
                @close="removeArrayItem('cors_allow_hostlist', index)"
                style="margin-right: 8px; margin-bottom: 8px"
              >
                {{ host }}
              </el-tag>
              <el-input
                v-model="newCorsHost"
                placeholder="输入域名后按回车添加"
                @keyup.enter="addCorsHost"
                style="width: 300px"
              />
            </el-form-item>

            <el-form-item label="排除路径">
              <el-tag
                v-for="(path, index) in config.system_conf.safe_conf.exclude_paths"
                :key="index"
                closable
                @close="removeArrayItem('exclude_paths', index)"
                style="margin-right: 8px; margin-bottom: 8px"
              >
                {{ path }}
              </el-tag>
              <el-input
                v-model="newExcludePath"
                placeholder="输入路径后按回车添加"
                @keyup.enter="addExcludePath"
                style="width: 300px"
              />
            </el-form-item>

            <el-form-item label="允许的扩展名">
              <el-tag
                v-for="(ext, index) in config.system_conf.safe_conf.allow_extension"
                :key="index"
                closable
                @close="removeArrayItem('allow_extension', index)"
                style="margin-right: 8px; margin-bottom: 8px"
              >
                {{ ext }}
              </el-tag>
              <el-input
                v-model="newAllowExtension"
                placeholder="输入扩展名后按回车添加"
                @keyup.enter="addAllowExtension"
                style="width: 300px"
              />
            </el-form-item>

            <el-divider content-position="left">验证配置</el-divider>
            <el-form-item label="启用 Turnstile">
              <el-switch v-model="config.system_conf.verify_conf.turnstile.enable" />
            </el-form-item>
            <template v-if="config.system_conf.verify_conf.turnstile.enable">
              <el-form-item label="Turnstile Site Key">
                <el-input
                  v-model="config.system_conf.verify_conf.turnstile.site_key"
                  placeholder="Turnstile Site Key"
                />
                <div class="form-item-help">用于前端渲染 Turnstile 组件。</div>
              </el-form-item>
              <el-form-item label="Turnstile Secret (敏感)">
                <el-input
                  v-model="config.system_conf.verify_conf.turnstile.secret"
                  placeholder="Turnstile Secret"
                  show-password
                />
                <div class="env-override-notice">
                  此配置可被环境变量 <code>TURNSTILE_SECRET</code> 覆盖。
                </div>
                <div class="form-item-help">
                  仅限受信任前端环境配置；密钥会存储在配置文件中。
                </div>
              </el-form-item>
            </template>
            <el-form-item label="Fingerprint Secret">
              <el-input
                v-model="config.system_conf.verify_conf.fingerprint.secret"
                placeholder="用于指纹签名的服务端密钥"
                show-password
              />
              <div class="env-override-notice">
                此配置用于指纹签名，不是 Turnstile 的密钥。可被环境变量 <code>FINGERPRINT_SECRET</code> 覆盖。
              </div>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 数据配置 -->
        <el-tab-pane label="数据配置" name="data">
          <el-form :model="config" label-width="150px">
            <el-divider content-position="left">数据库配置</el-divider>
            <el-form-item label="数据库路径">
              <el-input
                v-model="config.system_conf.data_conf.database.path"
                placeholder="例如: data/blog.db"
              />
            </el-form-item>

            <el-divider content-position="left">图片配置</el-divider>
            <el-form-item label="图片存储路径">
              <el-input
                v-model="config.system_conf.data_conf.image.path"
                placeholder="例如: data/images"
              />
            </el-form-item>
            <el-form-item label="图片转换格式">
              <el-select v-model="config.system_conf.data_conf.image.conv_to">
                <el-option label="webp" value="webp" />
                <el-option label="jpeg" value="jpeg" />
                <el-option label="png" value="png" />
                <el-option label="不转换" value="" />
              </el-select>
            </el-form-item>

            <el-divider content-position="left">资源配置</el-divider>
            <el-form-item label="资源存储路径">
              <el-input
                v-model="config.system_conf.data_conf.resource.path"
                placeholder="例如: data/resources"
              />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 邮件配置 -->
        <el-tab-pane label="邮件配置" name="email">
          <el-form :model="config" label-width="150px">
            <el-form-item label="启用邮件">
              <el-switch v-model="config.system_conf.email_conf.enable" />
            </el-form-item>
            <template v-if="config.system_conf.email_conf.enable">
              <el-form-item label="SMTP Host">
                <el-input
                  v-model="config.system_conf.email_conf.host"
                  placeholder="smtp.example.com"
                />
              </el-form-item>
              <el-form-item label="SMTP 端口">
                <el-input-number v-model="config.system_conf.email_conf.port" :min="1" :max="65535" />
              </el-form-item>
              <el-form-item label="SMTP 用户名">
                <el-input
                  v-model="config.system_conf.email_conf.user_name"
                  placeholder="user@example.com"
                />
              </el-form-item>
              <el-form-item label="SMTP 密码 (敏感)">
                <el-input
                  v-model="config.system_conf.email_conf.password"
                  placeholder="SMTP Password"
                  show-password
                />
                <div class="env-override-notice">
                  此配置可被环境变量 <code>EMAIL_PASSWORD</code> 覆盖。
                </div>
              </el-form-item>
              <el-form-item label="发件人">
                <el-input
                  v-model="config.system_conf.email_conf.sender"
                  placeholder="Blog <no-reply@example.com>"
                />
              </el-form-item>
            </template>
          </el-form>
        </el-tab-pane>

        <!-- 爬虫配置 -->
        <el-tab-pane label="爬虫配置" name="crawler">
          <el-form :model="config" label-width="150px">
            <el-form-item label="并发数量">
              <el-input-number
                v-model="config.system_conf.crawler_conf.concurrency"
                :min="1"
                :max="20"
              />
              <div style="color: var(--el-text-color-secondary); font-size: 12px; margin-top: 8px">
                设置 RSS 爬虫的并发数量，建议值为 5-10
              </div>
            </el-form-item>
            <el-form-item label="RSS 超时 (秒)">
              <el-input-number
                v-model="config.system_conf.crawler_conf.rss_timeout_seconds"
                :min="1"
                :max="120"
              />
              <div style="color: var(--el-text-color-secondary); font-size: 12px; margin-top: 8px">
                设置 RSS 解析请求的超时时间，建议值为 10-30
              </div>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 动态集成配置 -->
        <el-tab-pane label="动态集成" name="moments">
          <el-form
            v-if="config.system_conf.moments_integrated_conf"
            :model="config.system_conf.moments_integrated_conf"
            label-width="180px"
          >
            <el-form-item label="启用动态集成">
              <el-switch v-model="config.system_conf.moments_integrated_conf.enable" />
            </el-form-item>

            <template v-if="config.system_conf.moments_integrated_conf.enable">
              <el-divider content-position="left">通用设置</el-divider>
              <el-form-item label="API 单次返回数量">
                <el-input-number
                  v-model="config.system_conf.moments_integrated_conf.api_single_return_entries"
                  :min="1"
                  :max="100"
                />
                <div class="form-item-help">设置动态 API 单次返回的最大条目数。</div>
              </el-form-item>

              <el-divider content-position="left">Telegram 配置</el-divider>
              <el-form-item label="启用 Telegram 集成">
                <el-switch
                  v-model="config.system_conf.moments_integrated_conf.integrated.telegram.enable"
                />
              </el-form-item>
              <template v-if="config.system_conf.moments_integrated_conf.integrated.telegram.enable">
                <el-form-item label="同步删除">
                  <el-switch
                    v-model="config.system_conf.moments_integrated_conf.integrated.telegram.sync_delete"
                  />
                </el-form-item>
                <el-form-item label="Bot Token">
                  <el-input
                    v-model="config.system_conf.moments_integrated_conf.integrated.telegram.bot_token"
                    placeholder="Telegram Bot Token"
                    show-password
                  />
                  <div class="env-override-notice">
                    此配置可被环境变量 <code>TELEGRAM_BOT_TOKEN</code> 覆盖。
                  </div>
                </el-form-item>
                <el-form-item label="Channel ID">
                  <el-input
                    v-model="config.system_conf.moments_integrated_conf.integrated.telegram.channel_id"
                  />
                </el-form-item>
                <el-form-item label="媒体目录">
                  <el-input
                    v-model="config.system_conf.moments_integrated_conf.integrated.telegram.media_path"
                    placeholder="默认 telegram"
                  />
                </el-form-item>
                <el-form-item label="过滤用户 ID">
                  <el-tag
                    v-for="(id, index) in config.system_conf.moments_integrated_conf.integrated
                      .telegram.filter_userid"
                    :key="index"
                    closable
                    @close="removeMomentsArrayItem('telegram', 'filter_userid', index)"
                    style="margin-right: 8px; margin-bottom: 8px"
                  >
                    {{ id }}
                  </el-tag>
                  <el-input
                    v-model="newTelegramFilterUserid"
                    placeholder="输入 User ID 后按回车添加"
                    @keyup.enter="addTelegramFilterUserid"
                    style="width: 300px"
                  />
                </el-form-item>
              </template>

              <el-divider content-position="left">Discord 配置</el-divider>
              <el-form-item label="启用 Discord 集成">
                <el-switch
                  v-model="config.system_conf.moments_integrated_conf.integrated.discord.enable"
                />
              </el-form-item>
              <template v-if="config.system_conf.moments_integrated_conf.integrated.discord.enable">
                <el-form-item label="同步删除">
                  <el-switch
                    v-model="config.system_conf.moments_integrated_conf.integrated.discord.sync_delete"
                  />
                </el-form-item>
                <el-form-item label="Bot Token">
                  <el-input
                    v-model="config.system_conf.moments_integrated_conf.integrated.discord.bot_token"
                    placeholder="Discord Bot Token"
                    show-password
                  />
                  <div class="env-override-notice">
                    此配置可被环境变量 <code>DISCORD_BOT_TOKEN</code> 覆盖。
                  </div>
                </el-form-item>
                <el-form-item label="Guild ID">
                  <el-input
                    v-model="config.system_conf.moments_integrated_conf.integrated.discord.guild_id"
                  />
                </el-form-item>
                <el-form-item label="Channel ID">
                  <el-input
                    v-model="config.system_conf.moments_integrated_conf.integrated.discord.channel_id"
                  />
                </el-form-item>
                <el-form-item label="过滤用户 ID">
                  <el-tag
                    v-for="(id, index) in config.system_conf.moments_integrated_conf.integrated
                      .discord.filter_userid"
                    :key="index"
                    closable
                    @close="removeMomentsArrayItem('discord', 'filter_userid', index)"
                    style="margin-right: 8px; margin-bottom: 8px"
                  >
                    {{ id }}
                  </el-tag>
                  <el-input
                    v-model="newDiscordFilterUserid"
                    placeholder="输入 User ID 后按回车添加"
                    @keyup.enter="addDiscordFilterUserid"
                    style="width: 300px"
                  />
                </el-form-item>
              </template>
            </template>
          </el-form>
        </el-tab-pane>

        <!-- OSS 配置 -->
        <el-tab-pane label="OSS 配置" name="oss">
          <el-form
            v-if="config.system_conf.oss_conf"
            :model="config.system_conf.oss_conf"
            label-width="180px"
          >
            <el-form-item label="启用 OSS">
              <el-switch v-model="config.system_conf.oss_conf.enable" />
            </el-form-item>
            <template v-if="config.system_conf.oss_conf.enable">
              <el-form-item label="提供商">
                <el-select v-model="config.system_conf.oss_conf.provider" placeholder="选择 OSS 提供商">
                  <el-option label="阿里云" value="aliyun" />
                  <el-option label="腾讯云 (暂未支持)" value="tencent" disabled />
                  <el-option label="AWS S3" value="s3" />
                </el-select>
                <div class="form-item-help">选择您的对象存储服务提供商。</div>
              </el-form-item>
              <el-form-item label="Access Key ID">
                <el-input
                  v-model="config.system_conf.oss_conf.accessKeyId"
                  placeholder="OSS Access Key ID"
                />
                <div class="env-override-notice">
                  此配置可被环境变量 <code>OSS_ACCESS_KEY_ID</code> 覆盖。
                </div>
              </el-form-item>
              <el-form-item label="Access Key Secret">
                <el-input
                  v-model="config.system_conf.oss_conf.accessKeySecret"
                  placeholder="OSS Access Key Secret"
                  show-password
                />
                <div class="env-override-notice">
                  此配置可被环境变量 <code>OSS_ACCESS_KEY_SECRET</code> 覆盖。
                </div>
              </el-form-item>
              <el-form-item label="Bucket">
                <el-input v-model="config.system_conf.oss_conf.bucket" />
              </el-form-item>
              <el-form-item label="Endpoint">
                <el-input v-model="config.system_conf.oss_conf.endpoint" />
              </el-form-item>
              <el-form-item label="Region">
                <el-input v-model="config.system_conf.oss_conf.region" />
              </el-form-item>
              <el-form-item label="自定义域名">
                <el-input
                  v-model="config.system_conf.oss_conf.customDomain"
                  placeholder="例如: https://oss.example.com"
                />
              </el-form-item>
              <el-form-item label="上传路径前缀">
                <el-input v-model="config.system_conf.oss_conf.prefix" />
              </el-form-item>
              <el-form-item label="超时时间 (秒)">
                <el-input-number v-model="config.system_conf.oss_conf.timeout" :min="1" />
              </el-form-item>
              <el-form-item label="使用 HTTPS">
                <el-switch v-model="config.system_conf.oss_conf.secure" />
              </el-form-item>
            </template>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="PWA 配置" name="pwa">
          <el-form :model="config.system_conf.pwa_conf" label-width="150px">
            <el-form-item label="启用 PWA">
              <el-switch v-model="config.system_conf.pwa_conf.enable" />
              <div class="form-item-help">
                启用后，下次刷新页面时将自动注册 Service Worker，实现离线访问和应用安装功能。
              </div>
            </el-form-item>
            <el-form-item label="安装应用" v-if="pwaEnabled && canInstall">
              <el-button type="success" :icon="Download" @click="handleInstallPwa">
                安装到桌面
              </el-button>
              <div class="form-item-help">点击后浏览器将弹出安装确认窗口。</div>
            </el-form-item>
            <el-alert type="info" :closable="false" show-icon>
              <template #title>
                PWA 安装后可在手机桌面或电脑任务栏创建快捷方式，打开体验接近原生应用。启用后需刷新页面生效。
              </template>
            </el-alert>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="危险区域" name="danger">
          <div class="danger-zone">
            <div class="danger-zone__content">
              <div class="danger-zone__title">重启后端服务</div>
              <div class="danger-zone__desc">
                这会让当前后端进程主动退出，并依赖外部自启动机制重新拉起。请先确认已配置好自动重启。
              </div>
            </div>
            <el-button type="danger" :loading="restarting" @click="handleRestart">
              重启服务
            </el-button>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download } from '@element-plus/icons-vue'
import { getSystemConfig, restartSystem, updateSystemConfig } from '@/api/config'
import type { SystemConfig } from '@/model/config'
import { usePwa } from '@/composables/usePwa'

const { pwaEnabled, canInstall, install: installPwa } = usePwa()

const activeTab = ref('safe')
const saving = ref(false)
const restarting = ref(false)
const newCorsHost = ref('')
const newExcludePath = ref('')
const newAllowExtension = ref('')
const newTelegramFilterUserid = ref('')
const newDiscordFilterUserid = ref('')
// 用于存储原始配置以进行比较
const originalConfig = ref<SystemConfig | null>(null)

const config = ref<SystemConfig>({
  system_conf: {
    safe_conf: {
      cors_allow_hostlist: [],
      exclude_paths: [],
      allow_extension: []
    },
    data_conf: {
      database: {
        path: ''
      },
      image: {
        path: '',
        conv_to: ''
      },
      resource: {
        path: ''
      }
    },
    crawler_conf: {
      concurrency: 5,
      rss_timeout_seconds: 15
    },
    moments_integrated_conf: {
      enable: false,
      api_single_return_entries: 20,
      integrated: {
        telegram: {
          enable: false,
          sync_delete: false,
          bot_token: '',
          channel_id: '',
          media_path: '',
          filter_userid: []
        },
        discord: {
          enable: false,
          sync_delete: false,
          bot_token: '',
          guild_id: '',
          channel_id: '',
          filter_userid: []
        }
      }
    },
    oss_conf: {
      provider: 'aliyun',
      enable: false,
      accessKeyId: '',
      accessKeySecret: '',
      bucket: '',
      endpoint: '',
      region: '',
      secure: true,
      timeout: 30,
      prefix: '',
      customDomain: ''
    },
    verify_conf: {
      turnstile: {
        enable: false,
        secret: '',
        site_key: ''
      },
      fingerprint: {
        secret: ''
      }
    },
    email_conf: {
      enable: false,
      host: '',
      user_name: '',
      password: '',
      port: 465,
      sender: ''
    },
    pwa_conf: {
      enable: false
    }
  }
})

onMounted(async () => {
  try {
    const res = await getSystemConfig()
    if (!res.system_conf.verify_conf) {
      res.system_conf.verify_conf = {
        turnstile: {
          enable: false,
          secret: '',
          site_key: ''
        },
        fingerprint: {
          secret: ''
        }
      }
    } else if (!res.system_conf.verify_conf.turnstile) {
      res.system_conf.verify_conf.turnstile = {
        enable: false,
        secret: '',
        site_key: ''
      }
    } else if (!('site_key' in res.system_conf.verify_conf.turnstile)) {
      ;(res.system_conf.verify_conf.turnstile as any).site_key = ''
    }
    if (!res.system_conf.email_conf) {
      res.system_conf.email_conf = {
        enable: false,
        host: '',
        user_name: '',
        password: '',
        port: 465,
        sender: ''
      }
    }
    if (!res.system_conf.pwa_conf) {
      res.system_conf.pwa_conf = {
        enable: false
      }
    }
    config.value = res
    // 深度克隆初始配置，用于后续比较
    originalConfig.value = JSON.parse(JSON.stringify(res))
  } catch (error) {
    ElMessage.error('请求配置时出错')
    console.error(error)
  }
})

const addCorsHost = () => {
  if (newCorsHost.value.trim()) {
    config.value.system_conf.safe_conf.cors_allow_hostlist.push(newCorsHost.value.trim())
    newCorsHost.value = ''
  }
}

const addExcludePath = () => {
  if (newExcludePath.value.trim()) {
    config.value.system_conf.safe_conf.exclude_paths.push(newExcludePath.value.trim())
    newExcludePath.value = ''
  }
}

const addAllowExtension = () => {
  if (newAllowExtension.value.trim()) {
    config.value.system_conf.safe_conf.allow_extension.push(newAllowExtension.value.trim())
    newAllowExtension.value = ''
  }
}

const addTelegramFilterUserid = () => {
  if (newTelegramFilterUserid.value.trim()) {
    const id = newTelegramFilterUserid.value.trim()
    if (/^\d+$/.test(id)) {
      config.value.system_conf.moments_integrated_conf.integrated.telegram.filter_userid.push(id)
      newTelegramFilterUserid.value = ''
    } else {
      ElMessage.warning('请输入有效的用户 ID (仅数字)')
    }
  }
}

const addDiscordFilterUserid = () => {
  if (newDiscordFilterUserid.value.trim()) {
    const id = newDiscordFilterUserid.value.trim()
    if (/^\d+$/.test(id)) {
      config.value.system_conf.moments_integrated_conf.integrated.discord.filter_userid.push(id)
      newDiscordFilterUserid.value = ''
    } else {
      ElMessage.warning('请输入有效的用户 ID (仅数字)')
    }
  }
}

const removeArrayItem = (field: string, index: number) => {
  const safeConf = config.value.system_conf.safe_conf as any
  safeConf[field].splice(index, 1)
}

const removeMomentsArrayItem = (
  target: 'telegram' | 'discord',
  field: 'filter_userid',
  index: number
) => {
  config.value.system_conf.moments_integrated_conf.integrated[target][field].splice(index, 1)
}

const saveConfig = async () => {
  saving.value = true
  try {
    if (!originalConfig.value) {
      ElMessage.error('原始配置加载失败，无法保存')
      return
    }

    // 定义所有可能的配置项及其路径
    const configItems = [
      {
        key: 'system_conf.safe_conf.cors_allow_hostlist',
        currentValue: config.value.system_conf.safe_conf.cors_allow_hostlist,
        originalValue: originalConfig.value.system_conf.safe_conf.cors_allow_hostlist
      },
      {
        key: 'system_conf.safe_conf.exclude_paths',
        currentValue: config.value.system_conf.safe_conf.exclude_paths,
        originalValue: originalConfig.value.system_conf.safe_conf.exclude_paths
      },
      {
        key: 'system_conf.safe_conf.allow_extension',
        currentValue: config.value.system_conf.safe_conf.allow_extension,
        originalValue: originalConfig.value.system_conf.safe_conf.allow_extension
      },
      {
        key: 'system_conf.data_conf.database.path',
        currentValue: config.value.system_conf.data_conf.database.path,
        originalValue: originalConfig.value.system_conf.data_conf.database.path
      },
      {
        key: 'system_conf.data_conf.image.path',
        currentValue: config.value.system_conf.data_conf.image.path,
        originalValue: originalConfig.value.system_conf.data_conf.image.path
      },
      {
        key: 'system_conf.data_conf.image.conv_to',
        currentValue: config.value.system_conf.data_conf.image.conv_to,
        originalValue: originalConfig.value.system_conf.data_conf.image.conv_to
      },
      {
        key: 'system_conf.data_conf.resource.path',
        currentValue: config.value.system_conf.data_conf.resource.path,
        originalValue: originalConfig.value.system_conf.data_conf.resource.path
      },
      {
        key: 'system_conf.crawler_conf.concurrency',
        currentValue: config.value.system_conf.crawler_conf.concurrency,
        originalValue: originalConfig.value.system_conf.crawler_conf.concurrency
      },
      {
        key: 'system_conf.crawler_conf.rss_timeout_seconds',
        currentValue: config.value.system_conf.crawler_conf.rss_timeout_seconds,
        originalValue: originalConfig.value.system_conf.crawler_conf.rss_timeout_seconds
      },
      {
        key: 'system_conf.moments_integrated_conf',
        currentValue: config.value.system_conf.moments_integrated_conf,
        originalValue: originalConfig.value.system_conf.moments_integrated_conf
      },
      {
        key: 'system_conf.oss_conf',
        currentValue: config.value.system_conf.oss_conf,
        originalValue: originalConfig.value.system_conf.oss_conf
      },
      {
        key: 'system_conf.verify_conf',
        currentValue: config.value.system_conf.verify_conf,
        originalValue: originalConfig.value.system_conf.verify_conf
      },
      {
        key: 'system_conf.email_conf',
        currentValue: config.value.system_conf.email_conf,
        originalValue: originalConfig.value.system_conf.email_conf
      },
      {
        key: 'system_conf.pwa_conf',
        currentValue: config.value.system_conf.pwa_conf,
        originalValue: originalConfig.value.system_conf.pwa_conf
      }
    ]

    // 过滤出被修改过的配置项
    const updates = configItems
      .filter((item) => JSON.stringify(item.currentValue) !== JSON.stringify(item.originalValue))
      .map((item) => ({ key: item.key, value: item.currentValue }))

    if (updates.length === 0) {
      ElMessage.info('配置未发生更改')
      return
    }

    // 一次性发送所有更新
    await updateSystemConfig(updates)

    // 保存成功后，更新原始配置
    originalConfig.value = JSON.parse(JSON.stringify(config.value))

    ElMessage.success(`成功保存 ${updates.length} 项配置`)
    // Sync PWA toggle state to localStorage for usePwa to read
    localStorage.setItem('pwa_enabled', String(config.value.system_conf.pwa_conf.enable))

    // Guide user to refresh for PWA activation
    if (config.value.system_conf.pwa_conf.enable) {
      ElMessage.success('PWA 已启用，刷新页面后即可安装应用')
    }
  } catch (error) {
    ElMessage.error('保存配置失败')
    console.error(error)
  } finally {
    saving.value = false
  }
}

const handleInstallPwa = async () => {
  const ok = await installPwa()
  if (ok) {
    ElMessage.success('应用已安装')
  } else {
    ElMessage.info('安装已取消')
  }
}

const handleRestart = async () => {
  try {
    await ElMessageBox.confirm(
      '确认重启后端服务吗？当前进程会主动退出，并等待外部自启动拉起。',
      '危险操作',
      {
        confirmButtonText: '确认重启',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
  } catch {
    return
  }

  restarting.value = true
  try {
    const res = await restartSystem()
    ElMessage.success(res.message || '已发送重启请求，请等待服务重新上线')
  } catch (error) {
    ElMessage.error('发送重启请求失败')
    console.error(error)
  } finally {
    window.setTimeout(() => {
      restarting.value = false
    }, 3000)
  }
}
</script>

<style scoped>
.settings-container {
  padding: 20px;
}

.settings-card {
  max-width: 1200px;
  margin: 0 auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

:deep(.el-form-item__label) {
  font-weight: 500;
}

:deep(.el-divider__text) {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.form-item-help {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-top: 4px;
  line-height: 1.2;
}

.env-override-notice {
  color: var(--el-color-warning);
  font-size: 12px;
  margin-top: 4px;
  line-height: 1.2;
}
.env-override-notice code {
  background-color: var(--el-fill-color);
  padding: 2px 4px;
  border-radius: 4px;
  color: var(--el-text-color-secondary);
}

/* 设置 Tab 内容区域可滚动 */
:deep(.el-tabs__content) {
  max-height: 65vh;
  overflow-y: auto;
  padding-right: 15px; /* 为滚动条留出空间，防止内容跳动 */
}

.danger-zone {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px;
  margin-top: 8px;
  border: 1px solid var(--el-color-danger);
  border-radius: 8px;
  background: var(--el-color-danger-light-9);
}

.danger-zone__content {
  min-width: 0;
}

.danger-zone__title {
  color: var(--el-color-danger);
  font-size: 16px;
  font-weight: 600;
}

.danger-zone__desc {
  margin-top: 6px;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.5;
}

@media (max-width: 767px) {
  .settings-container .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  .settings-container .card-header .el-button {
    width: 100%;
  }
}
</style>
