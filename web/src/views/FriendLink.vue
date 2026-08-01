<template>
  <div class="friend-link-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>友链管理</span>
          <div>
            <el-button :icon="Edit" @click="openGroupDialog">分组管理</el-button>
            <el-button type="primary" :icon="Plus" @click="openFormDialog()">
              新增友链
            </el-button>
          </div>
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
          <el-option label="已拒绝" value="rejected"></el-option>
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
                <a v-if="isSafeUrl(row.link)" :href="row.link" target="_blank" rel="noopener noreferrer">{{ row.link }}</a>
                <span v-else>-</span>
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
                <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
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
                <el-image v-if="isSafeUrl(row.snapshot)" :src="row.snapshot" fit="cover" :lazy="true" style="width: 80px; height: 45px; border-radius: 4px;">
                  <template #error><span>-</span></template>
                </el-image>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="feed" label="RSS" width="200">
              <template #default="{ row }">
                <a v-if="isSafeUrl(row.feed)" :href="row.feed" target="_blank" rel="noopener noreferrer" style="color: var(--el-color-primary);">{{ row.feed }}</a>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="rejection_reason" label="拒绝理由" width="220">
              <template #default="{ row }">{{ row.rejection_reason || '-' }}</template>
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
          <el-input v-model="form.email" placeholder="选填，用于接收审核通知" />
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
        <el-form-item label="颜色" prop="color">
          <div class="color-input-row">
            <el-color-picker v-model="form.color" />
            <el-input v-model="form.color" placeholder="留空则在审核通过时自动生成" />
          </div>
        </el-form-item>
        <el-form-item label="标签" prop="tags">
          <el-select
            v-model="form.tags"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入标签后按回车"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="分组">
          <el-select v-model="form.group_ids" multiple filterable placeholder="请选择分组" style="width: 100%">
            <el-option v-for="group in groups" :key="group.id" :label="group.name" :value="group.id" />
          </el-select>
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
            <el-option label="已拒绝" value="rejected"></el-option>
            <el-option label="超时" value="timeout"></el-option>
            <el-option label="错误" value="error"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="拒绝理由" prop="rejection_reason" v-if="isEditMode && form.status === 'rejected'">
          <el-input type="textarea" v-model="form.rejection_reason" placeholder="填写拒绝原因，将随通知邮件发送给申请人" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" :disabled="saving || editGroupsLoadFailed" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="groupDialogVisible" title="友链分组管理" width="620px">
      <div class="group-create-row">
        <el-input v-model="groupForm.name" placeholder="分组名称" />
        <el-input v-model="groupForm.description" placeholder="分组描述" />
        <el-input-number v-model="groupForm.sort_order" :min="0" />
        <el-button type="primary" @click="saveGroup">{{ groupForm.id ? '保存' : '新增' }}</el-button>
      </div>
      <el-table :data="groups" v-loading="groupsLoading" style="width: 100%">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="description" label="描述" />
        <el-table-column prop="sort_order" label="排序" width="90" />
        <el-table-column label="操作" width="130">
          <template #default="{ row }">
            <el-button type="primary" link @click="editGroup(row)">编辑</el-button>
            <el-button type="danger" link @click="removeGroup(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
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
  recheckFriendLink,
  getFriendLinkGroups,
  createFriendLinkGroup,
  updateFriendLinkGroup,
  deleteFriendLinkGroup,
  getFriendLinkGroupIDs,
  setFriendLinkGroups
} from '@/api/friendLink'
import type { FriendLink, FriendLinkGroup } from '@/model/friendLink'
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
const groupDialogVisible = ref(false)
const groupsLoading = ref(false)
const groups = ref<FriendLinkGroup[]>([])
const groupForm = reactive({ id: 0, name: '', description: '', sort_order: 0 })
const editGroupsLoadFailed = ref(false)
const isEditMode = ref(false)
const saving = ref(false)
const formRef = ref<FormInstance>()
const form = reactive<{
  id: number
  name: string
  link: string
  avatar: string
  description: string
  email: string
  times: number
  status: 'survival' | 'timeout' | 'error' | 'pending' | 'rejected'
  enable_rss: boolean
  skip_health_check: boolean
  snapshot: string
  friend_link_page: string
  feed: string
  is_died: boolean
  rejection_reason: string
  color: string
  tags: string[]
  group_ids: number[]
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
  is_died: false,
  rejection_reason: '',
  color: '',
  tags: [],
  group_ids: []
})

const rules = reactive<FormRules>({
  name: [{ required: true, message: '请输入网站名称', trigger: 'blur' }],
  link: [
    { required: true, message: '请输入网站链接', trigger: 'blur' },
    { validator: (_rule, value, callback) => validateHttpUrl(value, callback), trigger: 'blur' }
  ],
  avatar: [
    { required: true, message: '请输入网站图标地址', trigger: 'blur' },
    { validator: (_rule, value, callback) => validateHttpUrl(value, callback), trigger: 'blur' }
  ],
  email: [{ validator: (_rule, value, callback) => validateEmail(value, callback), trigger: 'blur' }],
  feed: [{ validator: (_rule, value, callback) => validateOptionalHttpUrl(value, callback), trigger: 'blur' }],
  friend_link_page: [{ validator: (_rule, value, callback) => validateOptionalHttpUrl(value, callback), trigger: 'blur' }],
  snapshot: [{ validator: (_rule, value, callback) => validateOptionalHttpUrl(value, callback), trigger: 'blur' }]
})

const isSafeUrl = (value?: string) => {
  if (!value) return false
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}
const validateHttpUrl = (value: string, callback: (error?: Error) => void) => {
  callback(isSafeUrl(value) ? undefined : new Error('请输入有效的 HTTP(S) 地址'))
}
const validateOptionalHttpUrl = (value: string, callback: (error?: Error) => void) => {
  callback(!value || isSafeUrl(value) ? undefined : new Error('请输入有效的 HTTP(S) 地址'))
}
const validateEmail = (value: string, callback: (error?: Error) => void) => {
  callback(!value || /^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(value) ? undefined : new Error('请输入有效的邮箱地址'))
}

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

const fetchGroups = async () => {
  groupsLoading.value = true
  try {
    const res = await getFriendLinkGroups()
    if (res.code === 200) {
      groups.value = res.data ?? []
    } else {
      ElMessage.error(res.message || '获取友链分组失败')
    }
  } catch {
    ElMessage.error('获取友链分组失败')
  } finally {
    groupsLoading.value = false
  }
}

onMounted(async () => {
  await Promise.all([fetchFriendLinks(), fetchGroups()])
})

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
const openFormDialog = async (link?: FriendLink) => {
  if (link) {
    isEditMode.value = true
    Object.assign(form, {
      id: link.id,
      name: link.name,
      link: link.link,
      avatar: link.avatar,
      description: link.description,
      email: link.email || '',
      times: link.times ?? 0,
      status: link.status,
      enable_rss: link.enable_rss,
      skip_health_check: link.skip_health_check,
      snapshot: link.snapshot || '',
      friend_link_page: link.friend_link_page || '',
      feed: link.feed || '',
      is_died: link.is_died ?? false,
      rejection_reason: link.rejection_reason || '',
      color: link.color || '',
      tags: link.tags || [],
      group_ids: []
    })
    editGroupsLoadFailed.value = false
    try {
      const groupRes = await getFriendLinkGroupIDs(link.id)
      if (groupRes.code === 200) {
        form.group_ids = groupRes.data.group_ids
      } else {
        editGroupsLoadFailed.value = true
        ElMessage.error(groupRes.message || '获取友链分组失败')
      }
    } catch {
      editGroupsLoadFailed.value = true
      ElMessage.error('获取友链分组失败')
    }
  } else {
    isEditMode.value = false
    editGroupsLoadFailed.value = false
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
    is_died: false,
    rejection_reason: '',
    color: '',
    tags: [],
    group_ids: []
  })
  editGroupsLoadFailed.value = false
}

const submitForm = async () => {
  if (!formRef.value || saving.value || editGroupsLoadFailed.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    let friendLinkId = form.id
    if (isEditMode.value) {
          const { id, group_ids: _groupIds, ...data } = form
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
            color: form.color || undefined,
            tags: form.tags.length > 0 ? form.tags : undefined,
          }
          const response = await createFriendLink(payload)
          friendLinkId = response.data?.id ?? 0
          ElMessage.success('创建成功')
        }
        if (friendLinkId) {
          try {
            const groupsResponse = await setFriendLinkGroups(friendLinkId, form.group_ids)
            if (groupsResponse.code !== 200) {
              ElMessage.warning('友链已保存，但分组更新失败')
            }
          } catch {
            ElMessage.warning('友链已保存，但分组更新失败')
          }
        }
    dialogVisible.value = false
    await fetchFriendLinks()
  } catch (error) {
    ElMessage.error(isEditMode.value ? '更新失败，请稍后重试' : '创建失败，请稍后重试')
  } finally {
    saving.value = false
  }
}

const openGroupDialog = async () => {
  resetGroupForm()
  groupDialogVisible.value = true
  await fetchGroups()
}

const resetGroupForm = () => {
  Object.assign(groupForm, { id: 0, name: '', description: '', sort_order: 0 })
}

const editGroup = (group: FriendLinkGroup) => {
  Object.assign(groupForm, group)
}

const saveGroup = async () => {
  if (!groupForm.name.trim()) {
    ElMessage.warning('请输入分组名称')
    return
  }
  try {
    const payload = {
      name: groupForm.name,
      description: groupForm.description,
      sort_order: groupForm.sort_order
    }
    const res = groupForm.id
      ? await updateFriendLinkGroup(groupForm.id, payload)
      : await createFriendLinkGroup(payload)
    if (res.code !== 200 && res.code !== 201) {
      ElMessage.error(res.message || '保存分组失败')
      return
    }
    resetGroupForm()
    await fetchGroups()
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存分组失败')
  }
}

const removeGroup = (id: number) => {
  ElMessageBox.confirm('确定要删除这个分组吗？', '警告', { type: 'warning' }).then(async () => {
    try {
      const res = await deleteFriendLinkGroup(id)
      if (res.code !== 200) {
        ElMessage.error(res.message || '删除分组失败')
        return
      }
      await fetchGroups()
      ElMessage.success('删除成功')
    } catch {
      ElMessage.error('删除分组失败')
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
const statusLabel = (status: string) => ({
  survival: '正常',
  timeout: '超时',
  error: '错误',
  pending: '待审核',
  rejected: '已拒绝'
}[status] || status)

const statusTagType = (status: string) => {
  switch (status) {
    case 'survival':
      return 'success'
    case 'pending':
      return 'info'
    case 'rejected':
      return 'danger'
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

.color-input-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

.color-input-row .el-input {
  flex: 1
}

.group-create-row {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}

.group-create-row .el-input:first-child {
  flex: 1;
}

.group-create-row .el-input:nth-child(2) {
  flex: 1.5;
}

/* Responsive */
@media (max-width: 767px) {
  .friend-link-container .table-actions > * {
    width: 100% !important;
    margin-right: 0 !important;
  }
}
</style>
