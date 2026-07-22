<template>
  <div class="extension-editor">
    <div class="extension-editor-toolbar">
      <el-select v-model="selectedType" placeholder="添加卡片" size="small" clearable style="width: 140px" @change="handleTypeChange">
        <el-option label="GitHub 仓库" value="github" />
        <el-option label="网站链接" value="website" />
        <el-option label="位置" value="location" />
        <el-option label="音乐" value="music" />
        <el-option label="推文" value="tweet" />
      </el-select>
      <el-button v-if="currentValue" size="small" type="danger" plain @click="handleClear">
        移除卡片
      </el-button>
    </div>

    <div v-if="selectedType" class="extension-editor-form">
      <template v-if="selectedType === 'github'">
        <el-input v-model="form.github.repo_url" placeholder="GitHub 仓库 URL，如 https://github.com/user/repo" size="small" />
      </template>

      <template v-if="selectedType === 'website'">
        <el-input v-model="form.website.title" placeholder="网站标题" size="small" class="mb-1" />
        <el-input v-model="form.website.site" placeholder="网站 URL" size="small" />
      </template>

      <template v-if="selectedType === 'location'">
        <el-input v-model="form.location.placeholder" placeholder="地点名称" size="small" class="mb-1" />
        <div class="location-coords-row">
          <el-input-number v-model="form.location.latitude" :precision="6" :step="0.1" size="small" placeholder="纬度" controls-position="right" class="coord-input" />
          <el-input-number v-model="form.location.longitude" :precision="6" :step="0.1" size="small" placeholder="经度" controls-position="right" class="coord-input" />
        </div>
      </template>

      <template v-if="selectedType === 'music'">
        <el-input v-model="form.music.url" placeholder="音乐链接 URL" size="small" />
      </template>

      <template v-if="selectedType === 'tweet'">
        <el-input v-model="form.tweet.url" placeholder="推文 URL" size="small" class="mb-1" />
        <el-input v-model="form.tweet.username" placeholder="用户名（不含 @）" size="small" class="mb-1" />
        <el-input v-model="form.tweet.status_id" placeholder="推文 ID" size="small" />
      </template>

      <div class="extension-editor-actions">
        <el-button size="small" type="primary" @click="handleConfirm">保存卡片</el-button>
        <el-button size="small" @click="selectedType = ''">取消</el-button>
      </div>
    </div>

    <div v-if="currentValue" class="extension-preview">
      <el-tag size="small" type="info">卡片已添加，可继续编辑</el-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import type { ExtensionType, MomentExtension } from '../types'

const props = defineProps<{
  modelValue: string | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | null): void
}>()

const currentValue = ref(props.modelValue)
const selectedType = ref<ExtensionType | ''>('')

const form = reactive({
  github: { repo_url: '' },
  website: { title: '', site: '' },
  location: { placeholder: '', latitude: 0, longitude: 0 },
  music: { url: '' },
  tweet: { url: '', username: '', status_id: '' }
})

watch(() => props.modelValue, (val) => {
  currentValue.value = val
  resetForm()
  if (val) {
    try {
      const ext = JSON.parse(val) as MomentExtension
      selectedType.value = ext.type
      fillForm(ext)
    } catch {
      selectedType.value = ''
    }
  } else {
    selectedType.value = ''
  }
}, { immediate: true })

function resetForm() {
  Object.assign(form, {
    github: { repo_url: '' },
    website: { title: '', site: '' },
    location: { placeholder: '', latitude: 0, longitude: 0 },
    music: { url: '' },
    tweet: { url: '', username: '', status_id: '' }
  })
}

function fillForm(ext: MomentExtension) {
  switch (ext.type) {
    case 'github':
      form.github.repo_url = ext.payload.repo_url
      break
    case 'website':
      form.website.title = ext.payload.title
      form.website.site = ext.payload.site
      break
    case 'location':
      form.location.placeholder = ext.payload.placeholder
      form.location.latitude = ext.payload.latitude
      form.location.longitude = ext.payload.longitude
      break
    case 'music':
      form.music.url = ext.payload.url
      break
    case 'tweet':
      form.tweet.url = ext.payload.url
      form.tweet.username = ext.payload.username
      form.tweet.status_id = ext.payload.status_id
      break
  }
}

function buildExtension(): MomentExtension | null {
  switch (selectedType.value) {
    case 'github':
      if (!form.github.repo_url.trim()) return null
      return { type: 'github', payload: { repo_url: form.github.repo_url.trim() } }
    case 'website':
      if (!form.website.title.trim() || !form.website.site.trim()) return null
      return { type: 'website', payload: { title: form.website.title.trim(), site: form.website.site.trim() } }
    case 'location':
      return { type: 'location', payload: { placeholder: form.location.placeholder.trim(), latitude: form.location.latitude, longitude: form.location.longitude } }
    case 'music':
      if (!form.music.url.trim()) return null
      return { type: 'music', payload: { url: form.music.url.trim() } }
    case 'tweet':
      if (!form.tweet.url.trim() || !form.tweet.username.trim() || !form.tweet.status_id.trim()) return null
      return { type: 'tweet', payload: { url: form.tweet.url.trim(), username: form.tweet.username.trim(), status_id: form.tweet.status_id.trim() } }
    default:
      return null
  }
}

function handleTypeChange(type: ExtensionType | '') {
  if (!type) {
    handleClear()
    return
  }
  resetForm()
  currentValue.value = null
}

function handleConfirm() {
  const ext = buildExtension()
  if (ext) {
    const json = JSON.stringify(ext)
    currentValue.value = json
    emit('update:modelValue', json)
    selectedType.value = ext.type
  }
}

function handleClear() {
  currentValue.value = null
  selectedType.value = ''
  Object.assign(form, {
    github: { repo_url: '' },
    website: { title: '', site: '' },
    location: { placeholder: '', latitude: 0, longitude: 0 },
    music: { url: '' },
    tweet: { url: '', username: '', status_id: '' }
  })
  emit('update:modelValue', null)
}
</script>

<style scoped>
.extension-editor { margin-top: 8px; }
.extension-editor-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}
.extension-editor-form {
  margin-top: 8px;
  padding: 10px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  border: 1px solid var(--el-border-color-light);
}
.location-coords-row { display: flex; gap: 8px; }
.coord-input { flex: 1; }
.mb-1 { margin-bottom: 6px; }
.extension-editor-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}
.extension-preview { margin-top: 6px; }
</style>
