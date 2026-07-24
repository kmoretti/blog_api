<template>
  <ExtensionCardShell label="X / Twitter">
    <template #header-icon>
      <el-icon><ChatDotRound /></el-icon>
    </template>
    <template #header-actions>
      <el-link :href="payload.url" target="_blank" type="primary" :underline="'never'" size="small">
        查看 <el-icon><TopRight /></el-icon>
      </el-link>
    </template>
    <a :href="payload.url" target="_blank" rel="noopener noreferrer" class="card-link">
      <div class="tweet-card">
        <div class="tweet-avatar">
          <el-icon :size="20"><User /></el-icon>
        </div>
        <div class="tweet-meta">
          <span class="tweet-username">@{{ payload.username }}</span>
          <span class="tweet-action">在 X 上查看推文</span>
        </div>
      </div>
    </a>
  </ExtensionCardShell>
</template>

<script setup lang="ts">
import { ChatDotRound, TopRight, User } from '@element-plus/icons-vue'
import type { TweetPayload } from '../types'
import ExtensionCardShell from '../ExtensionCardShell.vue'

defineProps<{ payload: TweetPayload }>()
</script>

<style scoped>
.card-link {
  display: block;
  text-decoration: none;
  border-radius: inherit;
}
.card-link:hover { background: var(--el-fill-color); }
.tweet-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
}
.tweet-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: var(--el-color-info-light-9);
  color: var(--el-text-color-secondary);
  border: 1px solid var(--el-border-color-light);
}
.tweet-meta { min-width: 0; flex: 1; }
.tweet-username {
  display: block;
  font-size: 15px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.3;
}
.tweet-action {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-family: monospace;
}
</style>
