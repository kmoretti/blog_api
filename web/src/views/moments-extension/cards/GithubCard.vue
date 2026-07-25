<template>
  <ExtensionCardShell label="GitHub 仓库">
    <template #header-icon>
      <GithubIcon />
    </template>
    <a :href="payload.repo_url" target="_blank" rel="noopener noreferrer" class="card-link">
      <div class="github-card">
        <div class="github-avatar">
          <GithubIcon class="github-avatar-icon" />
        </div>
        <div class="github-meta">
          <span class="github-name">{{ repoName }}</span>
          <p class="github-description">{{ payload.repo_url }}</p>
        </div>
      </div>
    </a>
  </ExtensionCardShell>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ExtensionCardShell from '../ExtensionCardShell.vue'
import GithubIcon from '@/components/GithubIcon.vue'
import type { GithubPayload } from '../types'

const props = defineProps<{ payload: GithubPayload }>()

const repoName = computed(() => {
  const url = props.payload.repo_url.replace(/\/+$/, '')
  const match = url.match(/^https?:\/\/github\.com\/([^/]+\/[^/]+?)(?:\.git)?$/i)
  if (match) return match[1]
  const parts = url.split('/')
  return parts.slice(-2).join('/')
})
</script>

<style scoped>
.card-link {
  display: block;
  text-decoration: none;
  border-radius: inherit;
}

.card-link:focus-visible {
  outline: none;
  box-shadow: 0 0 0 1px var(--el-color-primary), 0 0 0 4px var(--el-color-primary-light-8);
}

.card-link:hover {
  background: var(--el-fill-color);
}

.github-card {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 12px;
}

.github-avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  overflow: hidden;
  border-radius: 50%;
  background: var(--el-fill-color);
  border: 1px solid var(--el-border-color-light);
}

.github-avatar-icon {
  width: 30px;
  height: 30px;
}

.github-meta {
  min-width: 0;
  flex: 1;
}

.github-name {
  display: block;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 700;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.github-description {
  display: -webkit-box;
  overflow: hidden;
  margin: 3px 0 0;
  color: var(--el-text-color-secondary);
  font-family: monospace;
  font-size: 12px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow-wrap: anywhere;
}
</style>
