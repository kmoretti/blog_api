<template>
  <ExtensionCardShell label="GitHub 仓库">
    <template #header-icon>
      <el-icon><Connection /></el-icon>
    </template>
    <a :href="payload.repo_url" target="_blank" rel="noopener noreferrer" class="card-link">
      <div class="github-card">
        <div class="github-avatar">
          <el-icon :size="28"><Connection /></el-icon>
        </div>
        <div class="github-meta">
          <span class="github-name">{{ repoName }}</span>
          <span class="github-url">{{ payload.repo_url }}</span>
        </div>
      </div>
    </a>
  </ExtensionCardShell>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Connection } from '@element-plus/icons-vue'
import type { GithubPayload } from '../types'
import ExtensionCardShell from '../ExtensionCardShell.vue'

const props = defineProps<{ payload: GithubPayload }>()

const repoName = computed(() => {
  const parts = props.payload.repo_url.replace(/\/+$/, '').split('/')
  return parts.slice(-2).join('/')
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
.github-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
}
.github-avatar {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color);
  border: 1px solid var(--el-border-color-light);
  color: var(--el-text-color-secondary);
}
.github-meta {
  min-width: 0;
  flex: 1;
}
.github-name {
  display: block;
  font-size: 15px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.github-url {
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
