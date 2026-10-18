<script setup lang="ts">
import { RouterView } from 'vue-router'
import { watch } from 'vue'
import { usePrivateNotesSession } from '@/runtime/private-notes'
import type { PreparedPage } from '@/runtime/router'
import ExternalLinkGuard from '@/site/components/ExternalLinkGuard.vue'
import { setExternalLinkGuardInternalOrigins, setExternalLinkGuardTrustedDomains } from '@/runtime/external-link-guard'

const props = defineProps<{
  page: PreparedPage
}>()

watch(
  () => props.page.payload.layout.site.url,
  url => setExternalLinkGuardInternalOrigins(url ? [url] : []),
  { immediate: true },
)

// 外链免确认白名单随页面 payload 下发（issue #1200）：管理端保存后，下一次
// 页面数据刷新即生效，无需额外请求。
watch(
  () => props.page.payload.layout.site.trustedExternalDomains,
  domains => setExternalLinkGuardTrustedDomains(domains ?? []),
  { immediate: true },
)
usePrivateNotesSession(() => props.page.payload.layout.viewer)
</script>

<template>
  <RouterView v-slot="{ Component }">
    <component :is="Component" :page="props.page" />
  </RouterView>
  <ExternalLinkGuard />
</template>
