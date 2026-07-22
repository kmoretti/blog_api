<template>
  <div class="page-grain" />
  <router-view />
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount } from 'vue'
import { useTheme } from '@/composables/useTheme'
import { usePwa } from '@/composables/usePwa'
import '@/styles/theme.css'

const { initTheme, cleanup } = useTheme()
usePwa()

onMounted(() => {
  initTheme()
})

onBeforeUnmount(() => {
  cleanup()
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  width: 100%;
  min-height: 100vh;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}
</style>
