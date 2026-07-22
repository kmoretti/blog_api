<template>
  <el-button
    :icon="isDark ? Sunny : Moon"
    :title="isDark ? '切换到明亮模式' : '切换到暗色模式'"
    class="theme-toggle-btn"
    circle
    size="small"
    @click="toggleTheme"
  />
</template>

<script setup lang="ts">
import { Moon, Sunny } from '@element-plus/icons-vue'
import { useTheme } from '@/composables/useTheme'

const { isDark, toggleTheme } = useTheme()
</script>

<style scoped>
.theme-toggle-btn {
  border: 1px solid var(--el-border-color);
  background: transparent;
  transition: transform 0.3s ease, color 0.3s ease;
  color: var(--el-text-color-secondary);
}

.theme-toggle-btn:hover {
  transform: rotate(45deg);
  color: var(--color-accent);
  border-color: var(--color-accent);
}
</style>
