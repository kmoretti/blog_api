<template>
  <div v-if="ext" class="extension-renderer">
    <GithubCard v-if="ext.type === 'github'" :payload="ext.payload as import('../types').GithubPayload" />
    <WebsiteCard v-else-if="ext.type === 'website'" :payload="ext.payload as import('../types').WebsitePayload" />
    <LocationCard v-else-if="ext.type === 'location'" :payload="ext.payload as import('../types').LocationPayload" />
    <MusicCard v-else-if="ext.type === 'music'" :payload="ext.payload as import('../types').MusicPayload" />
    <TweetCard v-else-if="ext.type === 'tweet'" :payload="ext.payload as import('../types').TweetPayload" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { parseExtension } from '../types'
import GithubCard from '../cards/GithubCard.vue'
import WebsiteCard from '../cards/WebsiteCard.vue'
import LocationCard from '../cards/LocationCard.vue'
import MusicCard from '../cards/MusicCard.vue'
import TweetCard from '../cards/TweetCard.vue'

const props = defineProps<{
  extension: string | null | undefined
}>()

const ext = computed(() => parseExtension(props.extension))
</script>

<style scoped>
.extension-renderer {
  width: 100%;
  margin-top: 8px;
}
</style>
