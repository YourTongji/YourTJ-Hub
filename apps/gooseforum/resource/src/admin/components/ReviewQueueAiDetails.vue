<script setup lang="ts">
import { Loader2 } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { Badge } from '@/admin/components/ui/badge'
import type { ReviewQueueItem } from '@/admin/types'
import { reasonText } from '@/admin/utils/aiModerationReasons'

// 审核队列条目的 AI 触因与图片预览（issue #975）。待审图片经 /file/img 授权预览
// 读取（站点管理员会话），不会进入共享缓存。
defineProps<{ item: ReviewQueueItem }>()
const { t } = useI18n()
</script>

<template>
  <div v-if="item.aiReview || item.aiChecking || item.images?.length" class="mt-1.5 space-y-1.5">
    <div v-if="item.aiChecking" class="flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
      <Badge variant="outline" class="gap-1 px-1.5 py-0 text-[10px]"><Loader2 class="size-3 motion-safe:animate-spin" />{{ t('aiModerationAdmin.queueChecking') }}</Badge>
      <span>{{ t('aiModerationAdmin.queueCheckingHint') }}</span>
    </div>
    <div v-else-if="item.aiReview" class="flex flex-wrap items-center gap-1 text-[11px] text-muted-foreground">
      <Badge variant="secondary" class="px-1.5 py-0 text-[10px]">{{ t('aiModerationAdmin.queueBadge') }}</Badge>
    </div>
    <ul v-if="item.aiReview?.reasons?.length" class="space-y-0.5 text-[11px] leading-4 text-muted-foreground">
      <li v-for="(reason, index) in item.aiReview.reasons" :key="index">{{ reasonText(t, reason) }}</li>
    </ul>
    <div v-if="item.images?.length" class="flex flex-wrap gap-1">
      <a v-for="url in item.images" :key="url" :href="url" target="_blank" rel="noopener">
        <img :src="url" alt="" loading="lazy" class="size-12 rounded border object-cover">
      </a>
    </div>
    <template v-if="item.aiReview">
      <p v-for="(image, index) in item.aiReview.images.filter(entry => entry.evidence)" :key="index" class="line-clamp-2 text-[11px] leading-4 text-muted-foreground">
        {{ image.evidence }}
      </p>
    </template>
  </div>
</template>
