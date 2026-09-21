<script setup lang="ts">
import { RouterView } from 'vue-router'
import { watch } from 'vue'
import type { PreparedPage } from '@/runtime/router'
import ExternalLinkGuard from '@/site/components/ExternalLinkGuard.vue'
import { setExternalLinkGuardInternalOrigins } from '@/runtime/external-link-guard'

const props = defineProps<{
  page: PreparedPage
}>()

watch(
  () => props.page.payload.layout.site.url,
  url => setExternalLinkGuardInternalOrigins(url ? [url] : []),
  { immediate: true },
)
</script>

<template>
  <RouterView v-slot="{ Component }">
    <component :is="Component" :page="props.page" />
  </RouterView>
  <ExternalLinkGuard />
</template>
