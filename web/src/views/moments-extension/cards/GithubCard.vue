<template>
  <ExtensionCardShell label="GitHub 仓库">
    <template #header-icon>
      <GithubIcon />
    </template>
    <a :href="payload.repo_url" target="_blank" rel="noopener noreferrer" class="card-link">
      <div class="github-card">
        <div class="github-avatar">
          <img
            v-if="repoData?.owner?.avatar_url"
            :src="repoData.owner.avatar_url"
            alt="GitHub 仓库所有者头像"
            class="github-avatar-image"
          />
          <GithubIcon v-else class="github-avatar-icon" />
        </div>
        <div class="github-meta">
          <span class="github-name">{{ repoData?.name || repoName }}</span>
          <p class="github-description">
            {{ repoData?.description || repoName }}
          </p>
          <div v-if="repoData" class="github-stats">
            <span class="github-stat">
              <StarIcon />
              <span>{{ repoData.stargazers_count }}</span>
            </span>
            <span class="github-stat-divider" aria-hidden="true"></span>
            <span class="github-stat">
              <ForkIcon />
              <span>{{ repoData.forks_count }}</span>
            </span>
          </div>
        </div>
      </div>
    </a>
  </ExtensionCardShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import ExtensionCardShell from '../ExtensionCardShell.vue'
import GithubIcon from '@/components/GithubIcon.vue'
import StarIcon from '@/components/StarIcon.vue'
import ForkIcon from '@/components/ForkIcon.vue'
import type { GithubPayload } from '../types'

type GithubRepository = {
  name: string
  description: string | null
  stargazers_count: number
  forks_count: number
  owner?: {
    avatar_url?: string
  }
}

const repositoryCache = new Map<string, GithubRepository | null>()
const repositoryRequests = new Map<string, Promise<GithubRepository | null>>()

const props = defineProps<{ payload: GithubPayload }>()

const repositoryPath = computed(() => {
  const url = props.payload.repo_url.replace(/\/+$/, '')
  const match = url.match(/^https?:\/\/github\.com\/([^/]+)\/([^/]+?)(?:\.git)?$/i)
  return match ? `${match[1]}/${match[2]}` : ''
})

const repoName = computed(() => {
  if (repositoryPath.value) return repositoryPath.value
  const parts = props.payload.repo_url.replace(/\/+$/, '').split('/')
  return parts.slice(-2).join('/')
})

const repoData = ref<GithubRepository | null>(null)

async function fetchRepository(path: string): Promise<GithubRepository | null> {
  if (repositoryCache.has(path)) {
    return repositoryCache.get(path) ?? null
  }

  if (!repositoryRequests.has(path)) {
    const request = fetch(`/api/public/github/repository/${encodeURIComponent(path.split('/')[0])}/${encodeURIComponent(path.split('/')[1])}`, {
      headers: { Accept: 'application/vnd.github+json' }
    })
      .then(async (response) => {
        if (!response.ok) return null
        return await response.json() as GithubRepository
      })
      .catch(() => null)
      .finally(() => {
        repositoryRequests.delete(path)
      })
    repositoryRequests.set(path, request)
  }

  const data = await repositoryRequests.get(path)!
  repositoryCache.set(path, data)
  return data
}

onMounted(async () => {
  if (!repositoryPath.value) return
  repoData.value = await fetchRepository(repositoryPath.value)
})
</script>

<style scoped>
.card-link {
  display: block;
  text-decoration: none;
  border-radius: inherit;
}

.card-link:focus-visible {
  outline: none;
  box-shadow: 0 0 0 1px var(--el-color-primary), 0 0 0 4px var(--el-color-primary-light-8);
}

.card-link:hover {
  background: var(--el-fill-color);
}

.github-card {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 12px;
}

.github-avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  overflow: hidden;
  border-radius: 50%;
  background: var(--el-fill-color);
  border: 1px solid var(--el-border-color-light);
}

.github-avatar-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.github-avatar-icon {
  width: 30px;
  height: 30px;
}

.github-meta {
  min-width: 0;
  flex: 1;
}

.github-name {
  display: block;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 700;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.github-description {
  display: -webkit-box;
  overflow: hidden;
  margin: 3px 0 0;
  color: var(--el-text-color-secondary);
  font-family: monospace;
  font-size: 12px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow-wrap: anywhere;
}

.github-stats {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-top: 7px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.github-stat {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.github-stat svg {
  width: 16px;
  height: 16px;
}

.github-stat-divider {
  width: 1px;
  height: 12px;
  background: var(--el-border-color-light);
}
</style>
