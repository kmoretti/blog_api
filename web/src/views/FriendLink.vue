<template>
  <div class="friend-link-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>友链管理</span>
          <el-button type="primary" :icon="Plus" @click="openFormDialog()">
            新增友链
          </el-button>
        </div>
      </template>

      <!-- Filter and Actions -->
      <div class="table-actions stack-mobile">
        <el-select v-model="filterIsDied" placeholder="按失效状态筛选" clearable @change="handleFilter"
          style="width: 150px; margin-right: 10px">
          <el-option label="已失效" :value="true"></el-option>
          <el-option label="未失效" :value="false"></el-option>
        </el-select>
        <el-select v-model="filterStatus" placeholder="按状态筛选" clearable @change="handleFilter"
          style="width: 150px; margin-right: 10px">
          <el-option label="正常" value="survival"></el-option>
          <el-option label="待定" value="pending"></el-option>
          <el-option label="超时" value="timeout"></el-option>
          <el-option label="错误" value="error"></el-option>
        </el-select>
        <el-input v-model="searchQuery" placeholder="搜索友链" clearable @input="handleSearch"
          style="width: 200px; margin-right: 10px" />
      </div>

      <!-- Friend Link Table -->
        <div class="table-responsive">
        <el-scrollbar height="60vh">
          <el-table :data="friendLinks" v-loading="loading" style="width: 100%; min-width: 900px">
            <el-table-column prop="name" label="网站名称" width="180" />
            <el-table-column prop="link" label="链接">
              <template #default="{ row }">
                <a :href="row.link" target="_blank">{{ row.link }}</a>
              </template>
            </el-table-column>
            <el-table-column prop="email" label="邮箱" width="200" />
            <el-table-column prop="times" label="失败次数" width="100" />
            <el-table-column label="是否失效" width="100">
              <template #default="{ row }">
                <el-tag :type="row.is_died ? 'danger' : 'success'">{{ row.is_died ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="statusTagType(row.status)">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="updated_at" label="更新时间" width="180">
              <template #default="{ row }">
                {{ formatDate(row.updated_at) }}
              </template>
            </el-table-column>
            <el-table-column label="不巡查" width="100">
              <template #default="{ row }">
                <el-switch :model-value="row.skip_health_check" @change="handleHealthCheckToggle(row)" />
              </template>
            </el-table-column>
            <el-table-column label="订阅 RSS" width="100">
              <template #default="{ row }">
                <el-switch :model-value="row.enable_rss" @change="handleRssToggle(row)" />
              </template>
            </el-table-column>
            <el-table-column prop="snapshot" label="封面" width="180">
              <template #default="{ row }">
                <el-image v-if="row.snapshot" :src="row.snapshot" fit="cover" style="width: 80px; height: 45px; border-radius: 4px;" />
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="feed" label="RSS" width="200">
              <template #default="{ row }">
                <a v-if="row.feed" :href="row.feed" target="_blank" style="color: var(--el-color-primary);">{{ row.feed }}</a>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="260" fixed="right">
              <template #default="{ row }">
                <el-button type="success" link :icon="Refresh"
                  :loading="recheckingId === row.id"
                  :disabled="recheckingId !== null && recheckingId !== row.id"
                  @click="handleRecheck(row.id)">
                  重新巡查
                </el-button>
                <el-button type="primary" link :icon="Edit" @click="openFormDialog(row)">
                  编辑
                </el-button>
                <el-button type="danger" link :icon="Delete" @click="handleDelete(row.id)">
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-scrollbar>
      </div>

      <!-- Pagination -->
      <el-pagination background layout="total, sizes, prev, pager, next, jumper" :total="totalLinks"
        :page-sizes="[10, 20, 50, 100]" :page-size="pageSize" :current-page="currentPage"
        @size-change="handleSizeChange" @current-change="handlePageChange" class="pagination-container" />
    </el-card>

    <!-- Form Dialog for Add/Edit -->
    <el-dialog :title="isEditMode ? '编辑友链' : '新增友链'" v-model="dialogVisible" width="500px" @close="resetForm">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="网站名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="网站链接" prop="link">
          <el-input v-model="form.link" />
        </el-form-item>
        <el-form-item label="网站图标" prop="avatar">
          <el-input v-model="form.avatar" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input type="textarea" v-model="form.description" />
        </el-form-item>
        <el-form-item label="站长邮箱" prop="email">
          <el-input v-model="form.email" />
        </el-form-item>
        <el-form-item label="订阅 RSS" prop="enable_rss">
          <el-switch v-model="form.enable_rss" />
        </el-form-item>
        <el-form-item label="网站封面">
          <el-input v-model="form.snapshot" placeholder="封面图片 URL" />
        </el-form-item>
        <el-form-item label="友链页面">
          <el-input v-model="form.friend_link_page" placeholder="https://example.com/links" />
        </el-form-item>
        <el-form-item label="博客 RSS">
          <el-input v-model="form.feed" placeholder="RSS 订阅地址" />
        </el-form-item>
        <el-form-item label="是否失效" prop="is_died" v-if="isEditMode">
          <el-switch v-model="form.is_died" />
        </el-form-item>
        <el-form-item label="失败次数" prop="times" v-if="isEditMode">
          <el-input-number v-model="form.times" :min="0" />
        </el-form-item>
        <el-form-item label="状态" prop="status" v-if="isEditMode">
          <el-select v-model="form.status">
            <el-option label="正常" value="survival"></el-option>
            <el-option label="待定" value="pending"></el-option>
            <el-option label="超时" value="timeout"></el-option>
            <el-option label="错误" value="error"></el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete, Edit, Refresh } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  getFriendLinks,
  createFriendLink,
  updateFriendLink,
  deleteFriendLink,
  recheckFriendLink
} from '@/api/friendLink'
import type { FriendLink } from '@/model/friendLink'
import { usePagination } from '@/utils/pagination'
import { formatDate } from '@/utils/date'

// Reactive State
const friendLinks = ref<FriendLink[]>([])
const loading = ref(false)
const recheckingId = ref<number | null>(null)
const filterStatus = ref('')
const filterIsDied = ref<boolean | null>(null)
const searchQuery = ref('')
const dialogVisible = ref(false)
const isEditMode = ref(false)
const formRef = ref<FormInstance>()
const form = reactive<{
  id: number
  name: string
  link: string
  avatar: string
  description: string
  email: string
  times: number
  status: 'survival' | 'timeout' | 'error' | 'pending'
  enable_rss: boolean
  skip_health_check: boolean
  snapshot: string
  friend_link_page: string
  feed: string
  is_died: boolean
}>({
  id: 0,
  name: '',
  link: '',
  avatar: '',
  description: '',
  email: '',
  times: 0,
  status: 'survival',
  enable_rss: true,
  skip_health_check: false,
  snapshot: '',
  friend_link_page: '',
  feed: '',
  is_died: false
})

const rules = reactive<FormRules>({
  name: [{ required: true, message: '请输入网站名称', trigger: 'blur' }],
  link: [{ required: true, message: '请输入网站链接', trigger: 'blur' }]
})

// Pagination
const { currentPage, pageSize, total, handlePageChange, handleSizeChange, reset } = usePagination(
  () => fetchFriendLinks(),
  10
)
const totalLinks = total // Alias for template

// Fetch data
const fetchFriendLinks = async () => {
  loading.value = true
  try {
    const res = await getFriendLinks({
      page: currentPage.value,
      page_size: pageSize.value,
      status: filterStatus.value,
      search: searchQuery.value,
      is_died: filterIsDied.value === null ? undefined : filterIsDied.value
    })
    if (res.code === 200) {
      friendLinks.value = res.data.items
      totalLinks.value = res.data.total
    } else {
      ElMessage.error(res.message || '获取友链列表失败')
    }
  } catch (error) {
    ElMessage.error('请求友链列表时出错')
  } finally {
    loading.value = false
  }
}

onMounted(fetchFriendLinks)

// Table and Actions
const handleFilter = () => {
  reset()
  fetchFriendLinks()
}

const handleSearch = () => {
  reset()
  fetchFriendLinks()
}

// Dialog and Form
const openFormDialog = (link?: FriendLink) => {
  if (link) {
    isEditMode.value = true
    Object.assign(form, link)
  } else {
    isEditMode.value = false
  }
  dialogVisible.value = true
}

const resetForm = () => {
  formRef.value?.resetFields()
  Object.assign(form, {
    id: 0,
    name: '',
    link: '',
    avatar: '',
    description: '',
    email: '',
    times: 0,
    status: 'survival',
    enable_rss: true,
    skip_health_check: false,
    snapshot: '',
    friend_link_page: '',
    feed: '',
    is_died: false
  })
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      try {
        if (isEditMode.value) {
          const { id, ...data } = form
          await updateFriendLink(id, { data })
          ElMessage.success('更新成功')
        } else {
          const payload = {
            name: form.name,
            link: form.link,
            avatar: form.avatar,
            description: form.description,
            email: form.email,
            enable_rss: form.enable_rss,
            snapshot: form.snapshot || undefined,
            friend_link_page: form.friend_link_page || undefined,
            feed: form.feed || undefined,
          }
          await createFriendLink(payload)
          ElMessage.success('创建成功')
        }
        dialogVisible.value = false
        fetchFriendLinks()
      } catch (error) {
        ElMessage.error(isEditMode.value ? '更新失败' : '创建失败')
      }
    }
  })
}

// Delete operations
const handleDelete = (id: number) => {
  ElMessageBox.confirm('确定要删除这个友链吗？', '警告', {
    type: 'warning'
  }).then(async () => {
    try {
      await deleteFriendLink(id)
      ElMessage.success('删除成功')
      fetchFriendLinks()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  })
}

const handleRecheck = async (id: number) => {
  if (recheckingId.value !== null) return

  recheckingId.value = id
  try {
    await recheckFriendLink(id)
    ElMessage.success('巡查完成')
    await fetchFriendLinks()
  } catch {
    // The response interceptor reports request failures.
  } finally {
    recheckingId.value = null
  }
}

// UI Helpers
const statusTagType = (status: string) => {
  switch (status) {
    case 'survival':
      return 'success'
    case 'pending':
      return 'info'
    case 'timeout':
      return 'warning'
    case 'error':
      return 'danger'
    default:
      return 'info'
  }
}
const handleHealthCheckToggle = async (link: FriendLink) => {
  const originalValue = link.skip_health_check
  const newValue = !originalValue

  link.skip_health_check = newValue
  try {
    await updateFriendLink(link.id, { data: { skip_health_check: newValue } })
    ElMessage.success(`已${newValue ? '停止' : '恢复'}巡查`)
    fetchFriendLinks()
  } catch (error) {
    link.skip_health_check = originalValue
    ElMessage.error('更新巡查状态失败')
  }
}

const handleRssToggle = async (link: FriendLink) => {
  const originalValue = link.enable_rss
  const newValue = !originalValue

  // If turning off, show confirmation dialog
  if (!newValue) {
    try {
      await ElMessageBox.confirm(
        '关闭 RSS 订阅将删除所有相关的订阅源和已抓取的文章。此操作不可逆，确定要继续吗？',
        '警告',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    } catch {
      // User canceled, do nothing, the switch state is not yet changed in the UI data
      return
    }
  }

  // Optimistically update the UI
  link.enable_rss = newValue

  // Proceed with the API call
  try {
    await updateFriendLink(link.id, { data: { enable_rss: newValue } })
    ElMessage.success(`已${newValue ? '开启' : '关闭'} RSS 订阅`)
    // On success, fetch the data again to ensure consistency
    fetchFriendLinks()
  } catch (error) {
    ElMessage.error('更新 RSS 订阅状态失败')
    // Revert the switch on API failure
    link.enable_rss = originalValue
  }
}
</script>

<style scoped>
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

/* Responsive */
@media (max-width: 767px) {
  .friend-link-container .table-actions > * {
    width: 100% !important;
    margin-right: 0 !important;
  }
}
</style>
