<template>
  <ExtensionCardShell label="音乐">
    <template #header-icon>
      <el-icon><Headset /></el-icon>
    </template>
    <a :href="payload.url" target="_blank" rel="noopener noreferrer" class="card-link">
      <div class="music-card">
        <div class="music-icon-wrap">
          <el-icon :size="24"><Headset /></el-icon>
        </div>
        <div class="music-meta">
          <span class="music-title">收听音乐</span>
          <span class="music-domain">{{ displayDomain }}</span>
        </div>
        <el-icon><TopRight /></el-icon>
      </div>
    </a>
  </ExtensionCardShell>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Headset, TopRight } from '@element-plus/icons-vue'
import type { MusicPayload } from '../types'
import ExtensionCardShell from '../ExtensionCardShell.vue'

const props = defineProps<{ payload: MusicPayload }>()

const displayDomain = computed(() => {
  try {
    return new URL(props.payload.url).hostname.replace(/^www\./, '')
  } catch {
    return props.payload.url
  }
})
</script>

<style scoped>
.card-link {
  display: block;
  text-decoration: none;
  border-radius: inherit;
}
.card-link:hover { background: var(--el-fill-color); }
.music-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
}
.music-icon-wrap {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.music-meta { min-width: 0; flex: 1; }
.music-title {
  display: block;
  font-size: 15px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.3;
}
.music-domain {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-family: monospace;
}
</style>
