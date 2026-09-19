<script setup lang="ts">
import type { CampusMessageDetail, CampusMessageSummary } from '@gooseforum/client'
import { ArrowUpRight, BookOpen, X } from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import EmptyState from './EmptyState.vue'

defineProps<{
  open: boolean
  summary: CampusMessageSummary | null
  detail: CampusMessageDetail | null
  loading: boolean
  error: string
  needsAuthorization: boolean
  pendingAuthorization: boolean
  busy: boolean
}>()
const emit = defineEmits<{ close: []; retry: []; authorize: []; confirmAuthorization: [] }>()
</script>

<template>
  <DialogRoot :open="open" @update:open="value => { if (!value) emit('close') }">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40 backdrop-blur-[2px]" />
      <DialogContent class="fixed left-1/2 top-1/2 z-[2101] flex max-h-[85dvh] w-[calc(100%_-_2rem)] max-w-2xl -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-box border border-line bg-base-100 text-base-content shadow-2xl outline-none">
        <header class="flex shrink-0 items-start justify-between gap-3 border-b border-line p-4 sm:p-5">
          <div class="min-w-0">
            <DialogTitle class="text-base font-semibold leading-7">{{ summary?.title || '校园消息' }}</DialogTitle>
            <DialogDescription class="mt-1 text-xs text-base-content/55">{{ summary?.publisher || '学校消息' }}</DialogDescription>
          </div>
          <button class="gf-icon-button h-8 w-8 shrink-0" aria-label="关闭消息" @click="emit('close')"><X class="h-4 w-4" /></button>
        </header>
        <div class="min-h-0 overflow-y-auto overscroll-contain">
          <EmptyState v-if="loading" loading title="正在读取消息…" />
          <EmptyState v-else-if="needsAuthorization && pendingAuthorization" :icon="BookOpen" title="学校认证已完成" description="还差最后一步：确认将新授权用于当前绑定，随后自动重新打开这条通知。">
            <button class="gf-button gf-button-sm gf-button-primary" :disabled="busy" @click="emit('confirmAuthorization')">确认更新授权</button>
          </EmptyState>
          <EmptyState v-else-if="error" :icon="BookOpen" title="暂时无法打开" :description="error">
            <button v-if="needsAuthorization" class="gf-button gf-button-sm gf-button-primary" :disabled="busy" @click="emit('authorize')">更新学校授权</button>
            <button v-else class="gf-button gf-button-sm gf-button-secondary" @click="emit('retry')">重试</button>
          </EmptyState>
          <article v-else-if="detail" class="p-4 sm:p-5">
            <p class="whitespace-pre-wrap break-words text-sm leading-7">{{ detail.content || '这条消息没有提供文字正文。' }}</p>
            <div v-if="detail.links.length" class="mt-6 border-t border-line pt-4">
              <h3 class="mb-2 text-xs font-semibold text-base-content/55">相关链接与图片</h3>
              <a v-for="link in detail.links" :key="link.url" :href="link.url" target="_blank" rel="noopener noreferrer" class="flex items-start gap-2 py-2 text-sm text-primary hover:underline">
                <span class="min-w-0 break-words">{{ link.label }}</span><ArrowUpRight class="mt-0.5 h-4 w-4 shrink-0" />
              </a>
            </div>
          </article>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
