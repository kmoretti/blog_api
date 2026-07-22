<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <div class="card-header">
          <h2>欢迎</h2>
        </div>
      </template>

      <el-form ref="formRef" :model="loginForm" :rules="rules" label-width="80px" @submit.prevent="handleLogin">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="loginForm.username" placeholder="请输入用户名" clearable />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input v-model="loginForm.password" type="password" placeholder="请输入密码" show-password
            @keyup.enter="handleLogin" />
        </el-form-item>

        <el-form-item v-if="turnstileEnabled" label-width="0">
          <div class="turnstile-wrap">
            <div id="turnstile-widget"></div>
          </div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="loading" style="width: 100%" @click="handleLogin">
            登录
          </el-button>
        </el-form-item>
      </el-form>
      <div class="extra-links">
        <el-link type="info" @click="handleForgotPassword">忘记密码？</el-link>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { authApi } from '@/api/auth'
import { useTheme } from '@/composables/useTheme'
const { initTheme } = useTheme()

type TurnstileConfig = {
  enable?: boolean
  site_key?: string
}

const router = useRouter()
const formRef = ref<FormInstance>()
const loading = ref(false)
const turnstileEnabled = ref(false)
const turnstileToken = ref('')
const turnstileWidgetId = ref<string | null>(null)
const turnstileSiteKey = ref('')

const loginForm = reactive({
  username: '',
  password: ''
})

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 3, message: '密码至少3个字符', trigger: 'blur' }
  ]
}

const loadTurnstileConfig = async () => {
  try {
    const res = await fetch('/api/public/verify_conf', { cache: 'no-store' })
    if (!res.ok) return
    const data = await res.json()
    const turnstile = (data?.data?.turnstile || {}) as TurnstileConfig
    turnstileSiteKey.value = turnstile.site_key || ''
    turnstileEnabled.value = !!turnstile.enable && !!turnstileSiteKey.value
  } catch (error) {
    console.error('Load turnstile config error:', error)
    turnstileEnabled.value = false
    turnstileSiteKey.value = ''
  }
}

const loadTurnstileScript = () => {
  const win = window as any
  if (win.turnstile) {
    renderTurnstile()
    return
  }

  const script = document.createElement('script')
  script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
  script.async = true
  script.defer = true
  script.onload = renderTurnstile
  document.head.appendChild(script)
}

const renderTurnstile = () => {
  const win = window as any
  if (!win.turnstile || !turnstileSiteKey.value) return
  if (turnstileWidgetId.value) {
    win.turnstile.remove(turnstileWidgetId.value)
    turnstileWidgetId.value = null
  }
  turnstileWidgetId.value = win.turnstile.render('#turnstile-widget', {
    sitekey: turnstileSiteKey.value,
    callback: (token: string) => {
      turnstileToken.value = token
    },
    'error-callback': () => {
      turnstileToken.value = ''
    },
    'expired-callback': () => {
      turnstileToken.value = ''
    }
  })
}

const resetTurnstile = () => {
  const win = window as any
  if (win.turnstile && turnstileWidgetId.value) {
    win.turnstile.reset(turnstileWidgetId.value)
  }
  turnstileToken.value = ''
}

const handleLogin = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true
      try {
        if (turnstileEnabled.value && !turnstileToken.value) {
          ElMessage.warning('请先完成人机验证')
          loading.value = false
          return
        }

        const response = await authApi.login({
          username: loginForm.username,
          password: loginForm.password,
          turnstile_token: turnstileToken.value || undefined
        })

        if (response.code === 200) {
          localStorage.setItem('token', response.data.token)
          localStorage.setItem('username', loginForm.username)
          ElMessage.success('登录成功')
          router.push('/')
        } else {
          ElMessage.error(response.message || '登录失败')
        }
      } catch (error) {
        console.error('Login error:', error)
        resetTurnstile()
      } finally {
        loading.value = false
      }
    }
  })
}

onMounted(async () => {
  initTheme()
  await loadTurnstileConfig()
  if (turnstileEnabled.value && turnstileSiteKey.value) {
    loadTurnstileScript()
  }
  // Set background image
  document.body.style.backgroundImage = `url(https://picsum.photos/1900/1000)`
  document.body.style.backgroundSize = 'cover'
  document.body.style.backgroundPosition = 'center'
  document.body.style.transition = 'background-image 1s ease-in-out'
})

onUnmounted(() => {
  // Reset background image when leaving the page
  document.body.style.backgroundImage = ''
  document.body.style.backgroundSize = ''
  document.body.style.backgroundPosition = ''
})

const handleForgotPassword = () => {
  ElMessageBox.alert(
    '有服务器访问权限的请自行修改环境变量 <code>WEB_PANEL_PWD</code><br>没有服务器访问权限的请联系管理员。',
    '蛤？怎么做到的？',
    {
      confirmButtonText: '我明白了',
      dangerouslyUseHTMLString: true,
    }
  )
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  height: 100%;
}

.login-card {
  width: 400px;
  max-width: 90vw;
  box-shadow: var(--shadow);
  background: var(--paper);
  border-radius: var(--radius-handdrawn);
  padding: 1rem;
  border: 1px solid var(--line);
}

@media (max-width: 767px) {
  .login-card {
    padding: 0.75rem;
    border-radius: var(--radius-handdrawn-wide);
  }
}

.extra-links {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.card-header {
  text-align: center;
}

.card-header h2 {
  margin: 0;
  color: var(--ink);
  font-weight: 600;
}

.turnstile-wrap {
  width: 100%;
  display: flex;
  justify-content: center;
}

.form-item-help {
  color: var(--muted);
  font-size: 12px;
  margin-top: 4px;
  line-height: 1.2;
}
</style>
