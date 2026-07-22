import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import Login from '@/views/Login.vue'
import Panel from '@/views/Panel.vue'
import FriendLink from '@/views/FriendLink.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    redirect: '/dashboard'
  },
  {
    path: '/',
    name: 'Panel',
    component: Panel,
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue')
      },
      {
        path: 'moments',
        name: 'Moments',
        component: () => import('@/views/Moments.vue')
      },
      {
        path: 'friend',
        name: 'Friend',
        component: FriendLink
      },
      {
        path: 'rss',
        name: 'Rss',
        component: () => import('@/views/Rss.vue')
      },
      {
        path: 'image',
        name: 'Image',
        component: () => import('@/views/Image.vue')
      },
      {
        path: 'resource',
        name: 'Resource',
        component: () => import('@/views/Resource.vue')
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/Settings.vue')
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory('/panel/'),
  routes
})

// 路由守卫
router.beforeEach((to, _from) => {
  const token = localStorage.getItem('token')

  if (to.meta.requiresAuth) {
    if (!token) {
      return '/login'
    }
  } else {
    if (token && to.path === '/login') {
      return '/'
    }
  }
})

export default router
