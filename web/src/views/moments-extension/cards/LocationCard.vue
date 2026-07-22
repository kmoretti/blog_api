<template>
  <ExtensionCardShell label="位置">
    <template #header-icon>
      <el-icon><Location /></el-icon>
    </template>
    <div class="location-card">
      <button type="button" class="location-trigger" @click="open = !open">
        <el-icon><Location /></el-icon>
        <span class="location-meta">
          <span class="location-text">{{ displayText }}</span>
          <span class="location-coords">{{ coordsText }}</span>
        </span>
        <el-icon class="location-arrow" :class="{ 'is-open': open }">
          <ArrowDown />
        </el-icon>
      </button>
      <div v-if="open" class="location-map">
        <iframe
          width="100%" height="200" frameborder="0" style="border:0"
          :src="mapUrl"
          allowfullscreen
          loading="lazy"
          referrerpolicy="no-referrer-when-downgrade"
        />
      </div>
    </div>
  </ExtensionCardShell>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Location, ArrowDown } from '@element-plus/icons-vue'
import type { LocationPayload } from '../types'
import ExtensionCardShell from '../ExtensionCardShell.vue'

const props = defineProps<{ payload: LocationPayload }>()
const open = ref(false)

const displayText = computed(() => props.payload.placeholder || coordsText.value)
const coordsText = computed(() => `${props.payload.latitude.toFixed(2)}°, ${props.payload.longitude.toFixed(2)}°`)
const mapUrl = computed(() =>
  `https://www.openstreetmap.org/export/embed.html?bbox=${props.payload.longitude - 0.01},${props.payload.latitude - 0.01},${props.payload.longitude + 0.01},${props.payload.latitude + 0.01}&layer=mapnik&marker=${props.payload.latitude},${props.payload.longitude}`
)
</script>

<style scoped>
.location-card { padding: 0; }
.location-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 12px;
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  text-align: left;
  cursor: pointer;
  font: inherit;
}
.location-trigger:hover { background: var(--el-fill-color); }
.location-meta {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}
.location-text {
  font-size: 14px;
  color: var(--el-text-color-primary);
  font-weight: 500;
  line-height: 1.25;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.location-coords {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  font-family: monospace;
  line-height: 1.25;
}
.location-arrow { transition: transform 0.2s ease; }
.location-arrow.is-open { transform: rotate(180deg); }
.location-map { border-top: 1px solid var(--el-border-color-light); }
.location-map iframe { display: block; }
</style>
