<template>
  <div class="image-management" v-loading="actionLoading" element-loading-text="处理中...">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>图片管理</span>
          <el-button type="primary" :icon="Plus" @click="handleShowAddDialog">
            新增图片
          </el-button>
        </div>
      </template>

      <div class="table-actions stack-mobile">
        <el-select
          v-model="searchParams.status"
          placeholder="按状态筛选"
          clearable
          @change="handleFilter"
          style="width: 150px; margin-right: 10px"
        >
          <el-option label="正常" value="normal"></el-option>
          <el-option label="暂停" value="pause"></el-option>
          <el-option label="损坏" value="broken"></el-option>
          <el-option label="待定" value="pending"></el-option>
        </el-select>
        <el-input
          v-model="searchParams.search"
          placeholder="搜索图片名称"
          clearable
          @input="handleSearch"
          style="width: 200px; margin-right: 10px"
        />
      </div>

      <div class="table-responsive">
      <el-scrollbar height="60vh">
        <el-table
          :data="images"
          v-loading="loading"
          style="width: 100%; min-width: 800px"
        >
          <el-table-column prop="id" label="ID" width="80"></el-table-column>
          <el-table-column prop="name" label="名称"></el-table-column>
          <el-table-column label="预览" width="120">
            <template #default="{ row }">
              <el-image
                style="width: 100px; height: 100px"
                :src="row.url"
                fit="cover"
                :preview-src-list="[row.url]"
                preview-teleported
              />
            </template>
          </el-table-column>
          <el-table-column prop="url" label="URL" show-overflow-tooltip></el-table-column>
          <el-table-column label="存储" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.is_oss" type="success">OSS</el-tag>
              <el-tag v-else-if="row.is_local" type="primary">本地</el-tag>
              <el-tag v-else type="info">外链</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link :icon="Edit" @click="handleShowEditDialog(row)">
                编辑
              </el-button>
              <el-popconfirm title="确定删除吗？" @confirm="handleDelete(row)">
                <template #reference>
                  <el-button type="danger" link :icon="Delete">删除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
      </el-scrollbar>
      </div>

      <el-pagination
        background
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        :page-size="pageSize"
        :current-page="currentPage"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
        class="pagination-container"
      ></el-pagination>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px" @close="handleCloseDialog">
      <el-form :model="form" label-width="80px" ref="formRef" :rules="rules">
        <el-form-item label="来源" prop="uploadType" v-if="!isEdit">
          <el-radio-group v-model="uploadType">
            <el-radio-button value="url">URL</el-radio-button>
            <el-radio-button value="upload">本地</el-radio-button>
            <el-radio-button value="upload_oss">OSS</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <template v-if="!isEdit">
          <el-form-item label="图片" prop="file" v-if="uploadType === 'upload' || uploadType === 'upload_oss'">
            <el-upload
              class="image-uploader"
              :auto-upload="false"
              :on-change="handleFileChange"
              :show-file-list="false"
              accept="image/*"
            >
              <el-image v-if="previewUrl" :src="previewUrl" class="preview-image" fit="cover" />
              <el-icon v-else class="image-uploader-icon"><Plus /></el-icon>
            </el-upload>
          </el-form-item>
          <el-form-item label="URL" prop="url" v-if="uploadType === 'url'">
            <el-input v-model="form.url" placeholder="请输入图片 URL"></el-input>
          </el-form-item>
        </template>

        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name"></el-input>
        </el-form-item>

        <el-form-item label="URL" prop="url" v-if="isEdit">
          <el-input v-model="form.url" placeholder="请输入图片 URL" disabled></el-input>
        </el-form-item>
        <el-form-item v-if="isEdit" label="状态" prop="status">
          <el-select v-model="form.status" placeholder="请选择状态">
            <el-option label="正常" value="normal"></el-option>
            <el-option label="暂停" value="pause"></el-option>
            <el-option label="损坏" value="broken"></el-option>
            <el-option label="待定" value="pending"></el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="handleSubmit">确 定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Delete, Edit } from '@element-plus/icons-vue'
import { getImages, createImage, updateImage, deleteImage } from '@/api/image'
import { uploadFile, deleteFile } from '@/api/resource'
import type { Image, CreateImagePayload, UpdateImagePayload } from '@/model/image'
import type { FormInstance, FormRules, UploadFile } from 'element-plus'
import { usePagination } from '@/utils/pagination'

const images = ref<Image[]>([])
const loading = ref(false)
const actionLoading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const dialogTitle = ref('')
const formRef = ref<FormInstance>()
const uploadType = ref<'url' | 'upload' | 'upload_oss'>('url')
const fileToUpload = ref<UploadFile | null>(null)
const previewUrl = ref<string>('')
const searchParams = reactive({
  status: '',
  search: ''
})

const { currentPage, pageSize, total, handlePageChange, handleSizeChange, reset } = usePagination(
  () => fetchImages(),
  10
)

const form = reactive<CreateImagePayload & { id?: number; status?: string }>({
  name: '',
  url: '',
  status: 'normal'
})

const rules = reactive<FormRules>({
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  url: [
    {
      validator: (_rule, value, callback) => {
        if (!isEdit.value && uploadType.value === 'url' && !value) {
          callback(new Error('请输入 URL'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
})

const fetchImages = async () => {
  loading.value = true
  try {
    const res = await getImages({
      page: currentPage.value,
      page_size: pageSize.value,
      status: searchParams.status,
      search: searchParams.search
    })
    images.value = res.data.items
    total.value = res.data.total
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

const handleFilter = () => {
  reset()
  fetchImages()
}

const handleSearch = () => {
  reset()
  fetchImages()
}

const handleShowAddDialog = () => {
  isEdit.value = false
  dialogTitle.value = '新增图片'
  uploadType.value = 'url'
  dialogVisible.value = true
}

const handleShowEditDialog = (row: Image) => {
  isEdit.value = true
  dialogTitle.value = '编辑图片'
  Object.assign(form, row)
  dialogVisible.value = true
}

const handleFileChange = (uploadFile: UploadFile) => {
  fileToUpload.value = uploadFile
  previewUrl.value = URL.createObjectURL(uploadFile.raw!)

  // 自动填充名称
  const fileName = uploadFile.name
  const nameWithoutExt = fileName.substring(0, fileName.lastIndexOf('.'))
  if (!form.name) {
    form.name = nameWithoutExt
  }
}

const handleCloseDialog = () => {
  formRef.value?.resetFields()
  fileToUpload.value = null
  previewUrl.value = ''
  uploadType.value = 'url'
  form.id = undefined
  form.name = ''
  form.url = ''
  form.status = 'normal'
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      actionLoading.value = true
      try {
        if (isEdit.value) {
          // 编辑模式
          const payload: UpdateImagePayload = {
            name: form.name,
            status: form.status
          }
          await updateImage(form.id!, payload)
          ElMessage.success('更新成功')
        } else {
          // 新增模式
          let payload: CreateImagePayload

          if (uploadType.value === 'upload' || uploadType.value === 'upload_oss') {
            if (!fileToUpload.value?.raw) {
              ElMessage.error('请选择要上传的图片')
              return
            }
            const formData = new FormData()
            formData.append('file', fileToUpload.value.raw)
            formData.append('path', 'image')
            const target = uploadType.value === 'upload' ? 'local' : 'oss'
            const res = await uploadFile(formData, target)

            payload = {
              name: form.name,
              url: res.data.url,
              local_path: res.data.local_path || res.data.objectKey,
              is_local: target === 'local' ? 1 : 0,
              is_oss: target === 'oss' ? 1 : 0
            }
          } else {
            if (!form.url) {
              ElMessage.error('请输入图片 URL')
              return
            }
            payload = {
              name: form.name,
              url: form.url,
              is_local: 0
            }
          }
          await createImage(payload)
          ElMessage.success('新增成功')
        }
        dialogVisible.value = false
        fetchImages()
      } catch (error) {
        console.error(error)
        // 错误消息已在各自的 api 调用中处理
      } finally {
        actionLoading.value = false
      }
    }
  })
}

const handleDelete = async (row: Image) => {
  actionLoading.value = true
  try {
    // 如果是本地或 OSS 文件，先删除文件
    if (row.is_local === 1 && row.local_path) {
      await deleteFile(row.local_path, 'local')
    } else if (row.is_oss === 1 && row.local_path) {
      await deleteFile(row.local_path, 'oss')
    }
    // 然后删除数据库记录
    await deleteImage(row.id)
    ElMessage.success('删除成功')
    fetchImages()
  } catch (error) {
    console.error(error)
    // 错误消息已在 api 调用中处理，或在此处提供通用消息
    ElMessage.error('删除失败，请查看控制台获取更多信息')
  } finally {
    actionLoading.value = false
  }
}

const getStatusTagType = (status: string) => {
  switch (status) {
    case 'normal':
      return 'success'
    case 'pause':
      return 'warning'
    case 'broken':
      return 'danger'
    case 'pending':
      return 'info'
    default:
      return ''
  }
}

onMounted(() => {
  fetchImages()
})
</script>

<style scoped>
.image-management {
  padding: 6px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.table-actions {
  margin-bottom: 16px;
}
.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
.image-uploader {
  width: 178px;
  height: 178px;
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: var(--el-transition-duration-fast);
  display: flex;
  justify-content: center;
  align-items: center;
}
.image-uploader:hover {
  border-color: var(--el-color-primary);
}
.image-uploader-icon {
  font-size: 28px;
  color: var(--el-text-color-secondary);
}
.preview-image {
  width: 100%;
  height: 100%;
}
</style>
