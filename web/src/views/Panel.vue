<template>
  <el-container class="panel-container">
    <el-header class="panel-header">
      <div class="tape tape--header-left" />
      <div class="header-left">
        <el-button v-if="isMobile" class="hide-desktop menu-btn" :icon="Operation"
          @click="drawerVisible = true" />
        <h2>Blog API</h2>
      </div>
      <div class="header-right">
        <ThemeToggle />
        <el-dropdown @command="handleCommand">
          <span class="user-info">
            <el-icon><User /></el-icon>
            <span class="hide-mobile">{{ username }}</span>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">
                <el-icon><SwitchButton /></el-icon>
                退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>

    <el-container>
      <!-- Desktop / Tablet sidebar -->
      <el-aside v-if="!isMobile" :width="asideWidth" class="panel-aside">
        <el-menu
          :default-active="activeMenu"
          :collapse="isTablet"
          class="sidebar-menu"
          @select="handleMenuSelect"
        >
          <el-menu-item index="dashboard">
            <el-icon><HomeFilled /></el-icon>
            <span>仪表板</span>
          </el-menu-item>
          <el-menu-item index="moments">
            <el-icon><ChatLineRound /></el-icon>
            <span>我的动态</span>
          </el-menu-item>
          <el-menu-item index="friend">
            <el-icon><Link /></el-icon>
            <span>友链管理</span>
          </el-menu-item>
          <el-menu-item index="rss">
            <el-icon><Document /></el-icon>
            <span>RSS 管理</span>
          </el-menu-item>
          <el-menu-item index="image">
            <el-icon><Picture /></el-icon>
            <span>图片管理</span>
          </el-menu-item>
          <el-menu-item index="resource">
            <el-icon><FolderOpened /></el-icon>
            <span>本地资源</span>
          </el-menu-item>
          <el-menu-item index="settings">
            <el-icon><Setting /></el-icon>
            <span>系统设置</span>
          </el-menu-item>
        </el-menu>
      </el-aside>

      <!-- Mobile drawer -->
      <el-drawer v-model="drawerVisible" direction="ltr" size="220px"
        :with-header="false" :modal="true">
        <el-menu :default-active="activeMenu" @select="handleDrawerSelect"
          class="sidebar-menu">
          <el-menu-item index="dashboard">
            <el-icon><HomeFilled /></el-icon>
            <span>仪表板</span>
          </el-menu-item>
          <el-menu-item index="moments">
            <el-icon><ChatLineRound /></el-icon>
            <span>我的动态</span>
          </el-menu-item>
          <el-menu-item index="friend">
            <el-icon><Link /></el-icon>
            <span>友链管理</span>
          </el-menu-item>
          <el-menu-item index="rss">
            <el-icon><Document /></el-icon>
            <span>RSS 管理</span>
          </el-menu-item>
          <el-menu-item index="image">
            <el-icon><Picture /></el-icon>
            <span>图片管理</span>
          </el-menu-item>
          <el-menu-item index="resource">
            <el-icon><FolderOpened /></el-icon>
            <span>本地资源</span>
          </el-menu-item>
          <el-menu-item index="settings">
            <el-icon><Setting /></el-icon>
            <span>系统设置</span>
          </el-menu-item>
        </el-menu>
      </el-drawer>

      <el-main class="panel-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import {
  User,
  SwitchButton,
  HomeFilled,
  ChatLineRound,
  Link,
  Document,
  Setting,
  Picture,
  FolderOpened,
  Operation
} from '@element-plus/icons-vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { useTheme } from '@/composables/useTheme'

const { initTheme } = useTheme()

const breakpoint = reactive({ width: window.innerWidth })
const isMobile = computed(() => breakpoint.width < 768)
const isTablet = computed(() => breakpoint.width >= 768 && breakpoint.width < 1024)
const asideWidth = computed(() => isTablet.value ? '64px' : '200px')
const drawerVisible = ref(false)

const handleDrawerSelect = (index: string) => {
  drawerVisible.value = false
  router.push(`/${index}`)
}

const router = useRouter()
const route = useRoute()
const username = ref('')

const activeMenu = computed(() => {
  return route.path.substring(1) // e.g., /friend-link -> friend-link
})

onMounted(() => {
  initTheme()
  username.value = localStorage.getItem('username') || '管理员'
  const handler = () => { breakpoint.width = window.innerWidth }
  window.addEventListener('resize', handler)
})

const handleMenuSelect = (index: string) => {
  router.push(`/${index}`)
}

const handleCommand = (command: string) => {
  if (command === 'logout') {
    ElMessageBox.confirm('确定要退出登录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }).then(() => {
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      ElMessage.success('已退出登录')
      router.push('/login')
    })
  }
}
</script>

<style scoped>
.panel-container {
  width: 100%;
  height: 100vh;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--paper);
  border-bottom: 1px solid var(--line);
  padding: 0 20px;
  box-shadow: var(--soft-shadow);
  position: relative;
  z-index: 10;
}

.header-left h2 {
  margin: 0;
  font-size: 20px;
  color: var(--ink);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 4px;
  transition: background 0.3s;
  color: var(--ink);
}

.user-info:hover {
  background: var(--color-accent-soft);
}

.panel-aside {
  background: var(--paper);
  border-right: 1px solid var(--line);
}

.sidebar-menu {
  border-right: none;
}

.panel-main {
  background: var(--bg);
  padding: 6px;
  overflow-y: auto;
}

/* Tape decoration on header */
.tape {
  position: absolute;
  z-index: -1;
  width: 70px;
  height: 20px;
  background-color: var(--tape-pink);
  background-image: var(--tape-stripes);
  filter: saturate(0.88);
  pointer-events: none;
}

.tape--header-left {
  top: -8px;
  left: 40px;
  transform: rotate(-4deg);
}

/* Mobile menu button */
.menu-btn {
  font-size: 18px;
  padding: 8px;
  margin-right: 4px;
  background: transparent;
  border: none;
  color: var(--ink);
}

/* Drawer menu */
.el-drawer .sidebar-menu {
  border-right: none;
  height: 100%;
}

/* Responsive header */
@media (max-width: 767px) {
  .panel-header {
    padding: 0 12px;
  }
  .header-left h2 {
    font-size: 16px;
  }
}
</style>
