<template>
  <div class="extension-card-shell">
    <div class="extension-card-shell__header">
      <span class="extension-card-shell__header-icon">
        <slot name="header-icon" />
      </span>
      <span class="extension-card-shell__header-label">{{ label }}</span>
      <div v-if="$slots['header-actions']" class="extension-card-shell__header-actions">
        <slot name="header-actions" />
      </div>
    </div>
    <div class="extension-card-shell__body">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
}>()
</script>

<style scoped>
.extension-card-shell {
  width: 100%;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-fill-color-lighter);
  overflow: hidden;
}

.extension-card-shell__header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
  border-bottom: 1px solid var(--el-border-color-light);
  user-select: none;
}

.extension-card-shell__header-icon {
  display: inline-flex;
  align-items: center;
  width: 16px;
  height: 16px;
}

.extension-card-shell__header-label {
  font-weight: 600;
  letter-spacing: 0.01em;
}

.extension-card-shell__header-actions {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.extension-card-shell__body {
  padding: 0;
}
</style>
