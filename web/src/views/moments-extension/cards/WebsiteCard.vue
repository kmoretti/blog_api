<template>
  <ExtensionCardShell label="网站链接">
    <template #header-icon>
      <el-icon><Link /></el-icon>
    </template>
    <a :href="payload.site" target="_blank" rel="noopener noreferrer" class="card-link">
      <div class="website-card">
        <div class="website-icon-wrap">
          <el-icon><Link /></el-icon>
        </div>
        <div class="website-meta">
          <span class="website-title">{{ payload.title }}</span>
          <span class="website-domain">{{ displayDomain }}</span>
        </div>
      </div>
    </a>
  </ExtensionCardShell>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Link } from '@element-plus/icons-vue'
import type { WebsitePayload } from '../types'
import ExtensionCardShell from '../ExtensionCardShell.vue'

const props = defineProps<{ payload: WebsitePayload }>()

const displayDomain = computed(() => {
  try {
    return new URL(props.payload.site).hostname.replace(/^www\./, '')
  } catch {
    return props.payload.site
  }
})
</script>

<style scoped>
.card-link {
  display: block;
  text-decoration: none;
  border-radius: inherit;
}
.card-link:hover {
  background: var(--el-fill-color);
}
.website-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
}
.website-icon-wrap {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
  border: 1px solid var(--el-border-color-light);
}
.website-meta {
  min-width: 0;
  flex: 1;
}
.website-title {
  display: block;
  font-size: 15px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.website-domain {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
