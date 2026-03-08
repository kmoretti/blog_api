<template>
  <el-container class="rss-container">
    <!-- RSS Feeds List -->
    <el-aside width="45%" class="feed-aside">
      <el-card shadow="never" class="full-height-card">
        <template #header>
          <div class="card-header">
            <span>RSS 订阅源</span>
            <el-button type="primary" @click="handleCreate" style="margin-left: auto">
              创建
            </el-button>
          </div>
        </template>
        <el-table
          :data="feeds"
          v-loading="feedsLoading"
          highlight-current-row
          @row-click="handleFeedSelect"
          height="calc(100vh - 210px)"
          style="width: 100%"
        >
          <el-table-column prop="name" label="名称">
            <template #default="{ row }">
              <el-tooltip :content="row.rss_url" placement="top">
                <a
                  :href="row.rss_url"
                  target="_blank"
                  style="margin-right: 8px; vertical-align: middle; color: inherit"
                >
                  <el-icon><Link /></el-icon>
                </a>
              </el-tooltip>
              <span>{{ row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="times" label="失败次数" width="90" />
          <el-table-column label="失效" width="80">
            <template #default="{ row }">
              <el-tag :type="row.is_died ? 'danger' : 'success'" size="small">
                {{ row.is_died ? '是' : '否' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160">
            <template #default="{ row }">
              <el-button type="primary" link :icon="Edit" @click.stop="handleEdit(row)">
                编辑
              </el-button>
              <el-button type="danger" link :icon="Delete" @click.stop="handleDelete(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          background
          layout="total, sizes, prev, pager, next"
          :total="totalFeeds"
          :page-sizes="[10, 20, 50]"
          :page-size="feedPageSize"
          :current-page="currentFeedPage"
          @size-change="handleFeedSizeChange"
          @current-change="handleFeedPageChange"
          class="pagination-container"
        />
      </el-card>
    </el-aside>

    <!-- RSS Posts List -->
    <el-main class="posts-main">
      <el-card shadow="never" class="full-height-card">
        <template #header>
          <div class="card-header">
            <span>{{ viewTitle }}</span>
            <el-button type="primary" link @click="showAllPosts" style="margin-left: auto">
              查看所有
            </el-button>
          </div>
        </template>
        <el-table
          :data="posts"
          v-loading="postsLoading"
          height="calc(100vh - 210px)"
          style="width: 100%"
        >
          <el-table-column prop="title" label="文章标题">
            <template #default="{ row }">
              <a :href="row.link" target="_blank" class="post-link">{{ row.title }}</a>
            </template>
          </el-table-column>
          <el-table-column prop="time" label="发布时间" width="180">
            <template #default="{ row }">
              {{ formatDate(row.time) }}
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          background
          layout="total, sizes, prev, pager, next, jumper"
          :total="totalPosts"
          :page-sizes="[10, 20, 50, 100]"
          :page-size="postPageSize"
          :current-page="currentPostPage"
          @size-change="handlePostSizeChange"
          @current-change="handlePostPageChange"
          class="pagination-container"
        />
      </el-card>
    </el-main>

    <!-- Edit Dialog -->
    <el-dialog v-model="editDialogVisible" title="编辑订阅源" width="500px">
      <el-form :model="editForm" ref="editFormRef" label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="URL" prop="rss_url">
          <el-input v-model="editForm.rss_url" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-select v-model="editForm.status" placeholder="请选择状态">
            <el-option label="正常" value="survival"></el-option>
            <el-option label="暂停" value="pause"></el-option>
            <el-option label="超时" value="timeout"></el-option>
            <el-option label="错误" value="error"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="失效" prop="is_died">
          <el-switch v-model="editForm.is_died" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- Create Dialog -->
    <el-dialog v-model="createDialogVisible" title="创建订阅源" width="500px">
      <el-form :model="createForm" ref="createFormRef" label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="createForm.name" />
        </el-form-item>
        <el-form-item label="URL" prop="rss_url">
          <el-input v-model="createForm.rss_url" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateSave">保存</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import { usePagination } from '@/utils/pagination'
import { formatDate } from '@/utils/date'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Delete, Link } from '@element-plus/icons-vue'
import {
  getRssFeeds,
  getPostsByFeed,
  getAllPosts,
  updateRssFeed,
  deleteRssFeed,
  createRssFeed
} from '@/api/rss'
import type { RssFeed, RssPost } from '@/model/rss'

const feeds = ref<RssFeed[]>([])
const posts = ref<RssPost[]>([])
const selectedFeed = ref<RssFeed | null>(null)
const isAllPostsView = ref(false)

const feedsLoading = ref(false)
const postsLoading = ref(false)

const viewTitle = computed(() => {
  if (isAllPostsView.value) {
    return '所有文章'
  }
  return selectedFeed.value ? selectedFeed.value.name : '请选择一个订阅源'
})

const editDialogVisible = ref(false)
const editForm = reactive({
  id: 0,
  name: '',
  rss_url: '',
  status: 'survival',
  is_died: false
})

const createDialogVisible = ref(false)
const createForm = reactive({
  name: '',
  rss_url: ''
})

const handleCreate = () => {
  createDialogVisible.value = true
}

const handleCreateSave = async () => {
  try {
    const res = await createRssFeed(createForm.name, createForm.rss_url)
    if (res.code === 201) {
      ElMessage.success('创建成功')
      createDialogVisible.value = false
      await fetchFeeds()
    } else {
      ElMessage.error(res.message || '创建失败')
    }
  } catch (error) {
    // The request interceptor handles error messages
  }
}

const fetchFeeds = async () => {
  feedsLoading.value = true
  try {
    const res = await getRssFeeds(currentFeedPage.value, feedPageSize.value)
    if (res.code === 200) {
      feeds.value = res.data.items
      totalFeeds.value = res.data.total
    } else {
      ElMessage.error(res.message || '获取订阅源失败')
    }
  } catch (error) {
    // The request interceptor handles error messages
  } finally {
    feedsLoading.value = false
  }
}

const fetchPosts = async () => {
  if (!selectedFeed.value) return
  postsLoading.value = true
  try {
    const res = await getPostsByFeed(selectedFeed.value.id, currentPostPage.value, postPageSize.value)
    if (res.code === 200) {
      posts.value = res.data.items
      totalPosts.value = res.data.total
    } else {
      ElMessage.error(res.message || '获取文章列表失败')
    }
  } catch (error) {
    // The request interceptor handles error messages
  } finally {
    postsLoading.value = false
  }
}

const fetchAllPosts = async () => {
  postsLoading.value = true
  try {
    const res = await getAllPosts(currentPostPage.value, postPageSize.value)
    if (res.code === 200) {
      posts.value = res.data.items
      totalPosts.value = res.data.total
    } else {
      ElMessage.error(res.message || '获取所有文章列表失败')
    }
  } catch (error) {
    // The request interceptor handles error messages
  } finally {
    postsLoading.value = false
  }
}

const fetchCurrentViewPosts = () => {
  if (isAllPostsView.value) {
    fetchAllPosts()
  } else {
    fetchPosts()
  }
}

// Feeds pagination
const {
  currentPage: currentFeedPage,
  pageSize: feedPageSize,
  total: totalFeeds,
  handlePageChange: handleFeedPageChange,
  handleSizeChange: handleFeedSizeChange
} = usePagination(fetchFeeds, 20)

// Posts pagination
const {
  currentPage: currentPostPage,
  pageSize: postPageSize,
  total: totalPosts,
  handlePageChange: handlePostPageChange,
  handleSizeChange: handlePostSizeChange,
  reset: resetPostPagination
} = usePagination(fetchCurrentViewPosts, 20)

const handleFeedSelect = (feed: RssFeed) => {
  if (selectedFeed.value?.id === feed.id && !isAllPostsView.value) return
  isAllPostsView.value = false
  selectedFeed.value = feed
  resetPostPagination()
  fetchPosts()
}

const showAllPosts = () => {
  isAllPostsView.value = true
  selectedFeed.value = null
  resetPostPagination()
  fetchAllPosts()
}

const handleEdit = (feed: RssFeed) => {
  editForm.id = feed.id
  editForm.name = feed.name
  editForm.rss_url = feed.rss_url
  editForm.status = feed.status
  editForm.is_died = feed.is_died
  editDialogVisible.value = true
}

const handleSave = async () => {
  try {
    const res = await updateRssFeed(editForm.id, {
      name: editForm.name,
      rss_url: editForm.rss_url,
      status: editForm.status,
      is_died: editForm.is_died
    })
    if (res.code === 200) {
      ElMessage.success('更新成功')
      editDialogVisible.value = false
      await fetchFeeds()
    } else {
      ElMessage.error(res.message || '更新失败')
    }
  } catch (error) {
    // The request interceptor handles error messages
  }
}

const handleDelete = (feed: RssFeed) => {
  ElMessageBox.confirm(`确定要删除订阅源 "${feed.name}" 吗？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      const res = await deleteRssFeed([feed.id])
      if (res.code === 200) {
        ElMessage.success('删除成功')
        await fetchFeeds()
        // If the deleted feed was the selected one, clear the posts list
        if (selectedFeed.value?.id === feed.id) {
          selectedFeed.value = null
          posts.value = []
        }
      } else {
        ElMessage.error(res.message || '删除失败')
      }
    } catch (error) {
      // The request interceptor handles error messages
    }
  })
}

const statusTagType = (status: string) => {
  switch (status) {
    case 'survival':
      return 'success'
    case 'pause':
      return 'info'
    case 'timeout':
      return 'warning'
    case 'error':
      return 'danger'
    default:
      return 'info'
  }
}

onMounted(() => {
  fetchFeeds()
})
</script>

<style scoped>
.rss-container {
  height: 100%;
}

.feed-aside {
  padding-right: 10px;
  border-right: 1px solid #e4e7ed;
}

.posts-main {
  padding: 0px;
}

.full-height-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.full-height-card .el-card__body {
  flex-grow: 1;
  overflow: hidden;
}

.card-header {
  font-weight: bold;
  display: flex;
  align-items: center;
}

.post-link {
  text-decoration: none;
  color: #409eff;
}

.post-link:hover {
  text-decoration: underline;
}

.pagination-container {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
