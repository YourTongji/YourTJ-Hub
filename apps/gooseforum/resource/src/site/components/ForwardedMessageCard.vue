<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChatForwardBundle } from '@gooseforum/client'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { ChevronRight, X } from '@lucide/vue'
import { formatDateTime } from '@/runtime/format'
import { parseStickerSegments, stickerPreviewLabel } from '@/site/utils/sticker-token'
import UserAvatar from './UserAvatar.vue'

const props = defineProps<{ bundle: ChatForwardBundle; stickerUrls: Map<string, string> }>()
const { t } = useI18n()
const open = ref(false)
</script>

<template>
  <button class="block w-full min-w-0 max-w-72 text-left" @click="open = true">
    <span class="block font-semibold">{{ t('messages.forwardHistory') }}</span>
    <span v-for="(entry, index) in bundle.messages.slice(0, 3)" :key="index" class="mt-1 block truncate text-xs">{{ entry.senderName }}: {{ entry.forwarded ? t('messages.forwardHistory') : stickerPreviewLabel(entry.content, t('stickers.previewAnimation')) }}</span>
    <span class="mt-2 flex items-center justify-between gap-4 border-t border-current/20 pt-2 text-xs">{{ t('messages.forwardCount', { count: bundle.messages.length }) }}<ChevronRight class="h-4 w-4" /></span>
  </button>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40" />
      <DialogContent class="fixed left-1/2 top-1/2 z-[2101] flex max-h-[85dvh] w-[calc(100%_-_2rem)] max-w-xl -translate-x-1/2 -translate-y-1/2 flex-col rounded-box border border-line bg-base-100 text-base-content shadow-xl outline-none">
        <header class="flex items-start justify-between gap-3 border-b border-line p-4">
          <div><DialogTitle class="font-semibold">{{ t('messages.forwardHistory') }}</DialogTitle><DialogDescription class="mt-1 text-xs text-base-content/60">{{ t('messages.forwardCount', { count: bundle.messages.length }) }}</DialogDescription></div>
          <button class="gf-icon-button h-9 w-9 shrink-0" :aria-label="t('messages.forwardClose')" @click="open = false"><X class="h-5 w-5" /></button>
        </header>
        <div class="min-h-0 overflow-y-auto overscroll-contain px-4">
          <article v-for="(entry, index) in bundle.messages" :key="index" class="flex items-start gap-2 py-2 first:pt-4 last:pb-4">
            <UserAvatar :src="entry.avatarUrl || '/static/pic/default-avatar.webp'" :alt="entry.senderName" class="h-8 w-8 shrink-0 rounded-full object-cover ring-1 ring-line" />
            <div class="min-w-0 max-w-[82%]">
            <div class="mb-1 break-words text-xs text-base-content/60">{{ entry.senderName }}</div>
            <div class="whitespace-pre-wrap break-words bg-base-300 px-3 py-2 text-sm leading-relaxed shadow-sm [border-radius:var(--gf-radius-box)] md:px-4">
              <ForwardedMessageCard v-if="entry.forwarded" :bundle="entry.forwarded" :sticker-urls="props.stickerUrls" />
              <template v-else><template v-for="(segment, segmentIndex) in parseStickerSegments(entry.content, props.stickerUrls)" :key="segmentIndex"><img v-if="segment.type === 'sticker'" :src="segment.url" :alt="`[:sticker:${segment.name}:]`" class="inline-block h-14 w-14 max-w-full object-contain align-middle" loading="lazy" /><template v-else>{{ segment.text }}</template></template></template>
            </div>
            <time class="mt-1 block text-[11px] text-base-content/55">{{ formatDateTime(entry.createdAt) }}</time>
            </div>
          </article>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
