import { ref, onMounted, onUnmounted } from 'vue'

const SW_URL = '/panel/sw.js'
const STORAGE_KEY = 'pwa_enabled'

// 在 Vite dev server 下跳过 SW 注册（sw.js 是构建产物，dev 时不存在）
const isDev = typeof window !== 'undefined' && window.location.port === '5173'

// 浏览器 PWA 安装事件（全局唯一，缓存即可）
let deferredPrompt: any = null

export function usePwa() {
  const pwaEnabled = ref(localStorage.getItem(STORAGE_KEY) === 'true')
  const swRegistered = ref(false)
  const canInstall = ref(false)

  function onBeforeInstallPrompt(e: Event) {
    e.preventDefault()
    deferredPrompt = e
    canInstall.value = true
  }

  function onAppInstalled() {
    canInstall.value = false
    deferredPrompt = null
  }

  async function register() {
    if (!('serviceWorker' in navigator)) return
    if (isDev) {
      console.log('[PWA] 开发模式下跳过 Service Worker 注册')
      return
    }
    try {
      const registration = await navigator.serviceWorker.register(SW_URL)
      swRegistered.value = true
      registration.addEventListener('updatefound', () => {
        const newWorker = registration.installing
        if (newWorker) {
          newWorker.addEventListener('statechange', () => {
            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
              // New version available — auto-update via registerType: 'autoUpdate'
            }
          })
        }
      })
    } catch (error) {
      console.error('SW registration failed:', error)
      swRegistered.value = false
    }
  }

  async function unregister() {
    if (!('serviceWorker' in navigator)) return
    try {
      const registration = await navigator.serviceWorker.getRegistration(SW_URL)
      if (registration) {
        await registration.unregister()
        swRegistered.value = false
      }
    } catch (error) {
      console.error('SW unregistration failed:', error)
    }
  }

  async function syncFromConfig() {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'true') {
      await register()
    } else {
      await unregister()
    }
  }

  function setEnabled(enabled: boolean) {
    localStorage.setItem(STORAGE_KEY, String(enabled))
    pwaEnabled.value = enabled
  }

  /** 触发浏览器安装弹窗 */
  async function install(): Promise<boolean> {
    if (!deferredPrompt) return false
    deferredPrompt.prompt()
    try {
      const result = await deferredPrompt.userChoice
      canInstall.value = false
      deferredPrompt = null
      return result.outcome === 'accepted'
    } catch {
      deferredPrompt = null
      return false
    }
  }

  onMounted(() => {
    syncFromConfig()
    window.addEventListener('beforeinstallprompt', onBeforeInstallPrompt)
    window.addEventListener('appinstalled', onAppInstalled)
  })

  onUnmounted(() => {
    window.removeEventListener('beforeinstallprompt', onBeforeInstallPrompt)
    window.removeEventListener('appinstalled', onAppInstalled)
  })

  return {
    pwaEnabled,
    swRegistered,
    canInstall,
    register,
    unregister,
    syncFromConfig,
    setEnabled,
    install,
  }
}
