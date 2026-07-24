<template>
  <div class="resource-management">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <span class="title">本地资源管理</span>
            <el-breadcrumb separator="/">
              <el-breadcrumb-item
                v-for="crumb in breadcrumbs"
                :key="crumb.path"
                class="breadcrumb-item"
              >
                <el-link
                  :underline="'never'"
                  @click="handleBreadcrumbClick(crumb.path)"
                >
                  {{ crumb.label }}
                </el-link>
              </el-breadcrumb-item>
            </el-breadcrumb>
          </div>
          <div class="header-actions">
            <el-button :disabled="!currentPath" @click="goToParent">
              返回上级
            </el-button>
            <el-button type="primary" :icon="Upload" @click="openUploadDialog">
              上传文件
            </el-button>
            <el-button :icon="Refresh" @click="refreshList">
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <div class="table-responsive">
      <el-table
        :data="sortedEntries"
        v-loading="loading"
        style="width: 100%"
      >
        <el-table-column label="名称" min-width="240">
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon class="name-icon">
                <Folder v-if="row.is_dir" />
                <Document v-else />
              </el-icon>
              <el-link
                v-if="row.is_dir"
                :underline="'never'"
                @click="handleOpen(row)"
              >
                {{ row.name }}
              </el-link>
              <span v-else>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路径" min-width="260" show-overflow-tooltip />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="row.is_dir ? 'info' : 'success'">
              {{ row.is_dir ? '文件夹' : '文件' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="大小" width="120">
          <template #default="{ row }">
            {{ row.is_dir ? '-' : formatSize(row.size) }}
          </template>
        </el-table-column>
        <el-table-column label="修改时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.mod_time) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.is_dir"
              type="primary"
              link
              :icon="FolderOpened"
              @click="handleOpen(row)"
            >
              打开
            </el-button>
            <el-button
              v-else
              type="primary"
              link
              :icon="Download"
              @click="handleDownload(row)"
            >
              下载
            </el-button>
            <el-button
              v-if="isEditableFile(row)"
              type="primary"
              link
              :icon="Edit"
              @click="handleEdit(row)"
            >
              编辑
            </el-button>
            <el-button
              type="danger"
              link
              :icon="Delete"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      </div>
    </el-card>

    <el-dialog v-model="uploadDialogVisible" title="上传文件" width="520px" @close="resetUploadForm">
      <el-form label-width="90px">
        <el-form-item label="目标目录">
          <el-input v-model="uploadPath" placeholder="相对 data 目录路径（可为空）" />
        </el-form-item>
        <el-form-item label="覆盖同名">
          <el-switch v-model="overwrite" />
        </el-form-item>
        <el-form-item label="文件">
          <el-upload
            v-model:file-list="uploadFiles"
            drag
            multiple
            :auto-upload="false"
          >
            <el-icon class="upload-icon"><UploadFilled /></el-icon>
            <div class="el-upload__text">拖拽文件到此处，或点击选择</div>
          </el-upload>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="uploadDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="uploading" @click="handleUpload">
          上传
        </el-button>
      </template>
    </el-dialog>

        <el-dialog
      v-model="editorDialogVisible"
      :title="editorTitle"
      width="80%"
      top="5vh"
      :before-close="handleEditorBeforeClose"
      @closed="resetEditor"
    >
      <div v-loading="editorLoading">
        <div ref="editorContainerRef" class="code-editor" />
      </div>
      <template #footer>
        <el-button @click="editorDialogVisible = false">关闭</el-button>
        <el-button
          type="primary"
          :loading="editorSaving"
          :disabled="editorLoading || !editingEntry || !hasEditorChanges"
          @click="handleSaveEdit"
        >
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  UploadFilled,
  Upload,
  Refresh,
  Folder,
  FolderOpened,
  Document,
  Delete,
  Download,
  Edit
} from '@element-plus/icons-vue'
import type { UploadFile } from 'element-plus'
import { listResources, uploadFile, deleteFile, getResourceFileBlob } from '@/api/resource'
import type { ResourceEntry } from '@/model/resource'
import { formatDate } from '@/utils/date'
import { EditorView, basicSetup } from 'codemirror'

const entries = ref<ResourceEntry[]>([])
const loading = ref(false)
const currentPath = ref('')
const uploadDialogVisible = ref(false)
const uploadFiles = ref<UploadFile[]>([])
const uploadPath = ref('')
const overwrite = ref(false)
const uploading = ref(false)
const editorDialogVisible = ref(false)
const editorLoading = ref(false)
const editorSaving = ref(false)
const editorContent = ref('')
const originalEditorContent = ref('')
const editingEntry = ref<ResourceEntry | null>(null)
const editorContainerRef = ref<HTMLElement | null>(null)
let editorView: EditorView | null = null

const editableExtensions = new Set([
  'txt', 'md', 'json', 'js', 'ts', 'css', 'html',
  'xml', 'yaml', 'yml', 'toml', 'ini', 'env', 'log', 'sql', 'sh', 'go', 'vue'
])

const normalizePath = (path: string) => path.replace(/^\/+/, '').replace(/\\/g, '/')
const encodePath = (path: string) =>
  encodeURI(normalizePath(path)).replace(/[?#]/g, (match) => encodeURIComponent(match))

const breadcrumbs = computed(() => {
  const parts = currentPath.value ? currentPath.value.split('/').filter(Boolean) : []
  const crumbs = [{ label: '根目录', path: '' as string }]
  let acc = ''
  for (const part of parts) {
    acc = acc ? `${acc}/${part}` : part
    crumbs.push({ label: part, path: acc })
  }
  return crumbs
})

const sortedEntries = computed(() => {
  return [...entries.value].sort((a, b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
    return a.name.localeCompare(b.name)
  })
})

const hasEditorChanges = computed(() => editorContent.value !== originalEditorContent.value)

const editorTitle = computed(() => {
  return editingEntry.value ? `编辑文件: ${editingEntry.value.path}` : '编辑文件'
})

const fetchResources = async (path = currentPath.value) => {
  loading.value = true
  try {
    const res = await listResources(path)
    if (Array.isArray(res.data)) {
      entries.value = res.data.map((item) => ({
        ...item,
        path: normalizePath(item.path)
      }))
    } else {
      entries.value = []
    }
  } catch (error) {
    ElMessage.error('获取资源列表失败')
  } finally {
    loading.value = false
  }
}

const refreshList = () => {
  fetchResources()
}

const handleOpen = (entry: ResourceEntry) => {
  if (!entry.is_dir) return
  currentPath.value = normalizePath(entry.path)
  fetchResources(currentPath.value)
}

const goToParent = () => {
  if (!currentPath.value) return
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  currentPath.value = parts.join('/')
  fetchResources(currentPath.value)
}

const handleBreadcrumbClick = (path: string) => {
  currentPath.value = path
  fetchResources(currentPath.value)
}

const handleDownload = (entry: ResourceEntry) => {
  const url = `/api/action/resource/${encodePath(entry.path)}`
  window.open(url, '_blank')
}

const isEditableFile = (entry: ResourceEntry) => {
  if (entry.is_dir) return false
  const ext = entry.name.includes('.') ? entry.name.split('.').pop()?.toLowerCase() : ''
  return Boolean(ext && editableExtensions.has(ext))
}

const handleEdit = async (entry: ResourceEntry) => {
  editorLoading.value = true
  try {
    const blob = await getResourceFileBlob(normalizePath(entry.path))
    editingEntry.value = entry
    editorContent.value = await blob.text()
    originalEditorContent.value = editorContent.value
    editorDialogVisible.value = true
    await nextTick()
    mountEditor(editorContent.value)
  } catch (error) {
    ElMessage.error('读取文件失败')
  } finally {
    editorLoading.value = false
  }
}

const mountEditor = (content: string) => {
  if (!editorContainerRef.value) return
  if (editorView) {
    editorView.destroy()
    editorView = null
  }
  editorView = new EditorView({
    doc: content,
    extensions: [
      basicSetup,
      EditorView.lineWrapping,
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          editorContent.value = update.state.doc.toString()
        }
      })
    ],
    parent: editorContainerRef.value
  })
  editorContent.value = content
}

const resetEditor = () => {
  if (editorView) {
    editorView.destroy()
    editorView = null
  }
  editorContent.value = ''
  originalEditorContent.value = ''
  editingEntry.value = null
  editorLoading.value = false
  editorSaving.value = false
}

const handleEditorBeforeClose = (done: () => void) => {
  if (!hasEditorChanges.value) {
    done()
    return
  }
  ElMessageBox.confirm('文件内容尚未保存，确认关闭吗？', '提示', {
    type: 'warning'
  }).then(() => done())
}

const handleSaveEdit = async () => {
  if (!editingEntry.value) return
  if (editorView) {
    editorContent.value = editorView.state.doc.toString()
  }
  const normalized = normalizePath(editingEntry.value.path)
  const filename = normalized.split('/').pop()
  if (!filename) {
    ElMessage.error('无效文件名')
    return
  }
  const pathSegments = normalized.split('/')
  pathSegments.pop()
  const targetPath = pathSegments.join('/')

  editorSaving.value = true
  try {
    const blob = new Blob([editorContent.value], { type: 'text/plain;charset=utf-8' })
    const file = new File([blob], filename, { type: 'text/plain;charset=utf-8' })
    const formData = new FormData()
    formData.append('file', file)
    formData.append('path', targetPath)
    formData.append('overwrite', 'true')
    await uploadFile(formData, 'local')
    originalEditorContent.value = editorContent.value
    ElMessage.success('保存成功')
    fetchResources(currentPath.value)
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    editorSaving.value = false
  }
}

const openUploadDialog = () => {
  uploadPath.value = currentPath.value
  uploadDialogVisible.value = true
}

const resetUploadForm = () => {
  uploadFiles.value = []
  uploadPath.value = currentPath.value
  overwrite.value = false
}

const handleUpload = async () => {
  if (!uploadFiles.value.length) {
    ElMessage.error('请选择要上传的文件')
    return
  }
  uploading.value = true
  const targetPath = normalizePath(uploadPath.value)
  try {
    for (const file of uploadFiles.value) {
      if (!file.raw) continue
      const formData = new FormData()
      formData.append('file', file.raw)
      formData.append('path', targetPath)
      formData.append('overwrite', overwrite.value ? 'true' : 'false')
      await uploadFile(formData, 'local')
    }
    ElMessage.success('上传成功')
    uploadDialogVisible.value = false
    fetchResources(currentPath.value)
  } catch (error) {
    ElMessage.error('上传失败')
  } finally {
    uploading.value = false
  }
}

const handleDelete = (entry: ResourceEntry) => {
  ElMessageBox.confirm(`确定要删除 ${entry.name} 吗？`, '提示', {
    type: 'warning'
  }).then(async () => {
    try {
      await deleteFile(normalizePath(entry.path), 'local')
      ElMessage.success('删除成功')
      fetchResources(currentPath.value)
    } catch (error) {
      ElMessage.error('删除失败')
    }
  })
}

const formatSize = (size: number) => {
  if (size <= 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let index = 0
  let value = size
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index++
  }
  return `${value.toFixed(value >= 10 || index === 0 ? 0 : 1)} ${units[index]}`
}

onMounted(() => {
  fetchResources()
})

onBeforeUnmount(() => {
  if (editorView) {
    editorView.destroy()
    editorView = null
  }
})
</script>

<style scoped>
.resource-management {
  padding: 6px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}
.header-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.title {
  font-size: 16px;
  font-weight: 600;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
.name-icon {
  color: var(--el-color-primary);
}
.upload-icon {
  font-size: 28px;
  color: var(--el-text-color-secondary);
}
.code-editor {
  min-height: 520px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  overflow: hidden;
}
.code-editor :deep(.cm-editor) {
  height: 520px;
  font-size: 13px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}
.code-editor :deep(.cm-scroller) {
  overflow: auto;
}

@media (max-width: 767px) {
  .resource-management .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  .resource-management .header-left {
    width: 100%;
  }
  .resource-management .header-actions {
    width: 100%;
    display: flex;
    gap: 4px;
  }
  .resource-management .header-actions .el-button {
    flex: 1;
    font-size: 12px;
    padding: 8px 4px;
  }
}
</style>
