import { ref } from 'vue'

const THEME_KEY = 'theme'
const DARK_CLASS = 'dark-mode'

const isDark = ref(false)
let mediaQuery: MediaQueryList | null = null
let mediaListener: ((e: MediaQueryListEvent) => void) | null = null
let userSet = false

export function useTheme() {
  function applyTheme(dark: boolean) {
    isDark.value = dark
    if (dark) {
      document.documentElement.classList.add(DARK_CLASS)
    } else {
      document.documentElement.classList.remove(DARK_CLASS)
    }
  }

  function setTheme(dark: boolean) {
    userSet = true
    localStorage.setItem(THEME_KEY, dark ? 'dark' : 'light')
    applyTheme(dark)
  }

  function toggleTheme() {
    setTheme(!isDark.value)
  }

  function initTheme() {
    const saved = localStorage.getItem(THEME_KEY)
    if (saved === 'dark') {
      applyTheme(true)
      return
    }
    if (saved === 'light') {
      applyTheme(false)
      return
    }
    userSet = false
    mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    applyTheme(mediaQuery.matches)
    mediaListener = (e: MediaQueryListEvent) => {
      if (!userSet) {
        applyTheme(e.matches)
      }
    }
    mediaQuery.addEventListener('change', mediaListener)
  }

  function cleanup() {
    if (mediaQuery && mediaListener) {
      mediaQuery.removeEventListener('change', mediaListener)
    }
  }

  return {
    isDark,
    toggleTheme,
    setTheme,
    initTheme,
    cleanup
  }
}
