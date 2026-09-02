<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { MessageSquare, Pin, Sparkles, HelpCircle, Lightbulb, FileText } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { formatNumber, timeAgo } from '@/runtime/format'
import { topicDescription } from '@/runtime/topic-description'
import AvatarStack from '@/site/components/AvatarStack.vue'
import TopicFeedPreview from '@/site/components/TopicFeedPreview.vue'
import type { TopicPayload } from '@gooseforum/client'

withDefaults(defineProps<{
  topic: TopicPayload
  home?: boolean
  showCategories?: boolean
  showHot?: boolean
  showPinned?: boolean
}>(), {
  home: false,
  showCategories: true,
  showHot: true,
  showPinned: false,
})

const { t } = useI18n()

// 桌面端：悬停行停留片刻后，以弹层形式预览信息流卡片。
// 弹层 fixed 定位在视口上，不推挤表格布局；带延迟避免扫过列表时误触。
// 位置策略：默认出现在鼠标指向的标题右侧，右侧空间不足时主动缩宽，
// 仍不足则移到鼠标左侧，保证不溢出屏幕。
const EXPAND_DELAY = 300
const COLLAPSE_DELAY = 200
const rowEl = ref<HTMLElement | null>(null)
const expanded = ref(false)
const popoverStyle = ref<{ top: string; left: string; width: string }>({ top: '0px', left: '0px', width: '0px' })
let enterTimer: number | undefined
let leaveTimer: number | undefined

function onRowEnter(event: MouseEvent) {
  // 移动端（<1024px）不启用 hover 弹层：触屏没有悬停语义，弹层易误触遮挡内容
  if (window.innerWidth < 1024) return
  window.clearTimeout(leaveTimer)
  if (expanded.value) return
  enterTimer = window.setTimeout(() => openPreview(event), EXPAND_DELAY)
}

function onRowLeave() {
  window.clearTimeout(enterTimer)
  leaveTimer = window.setTimeout(closePreview, COLLAPSE_DELAY)
}

function openPreview(event: MouseEvent) {
  // 移动端（<1024px）不展示 hover 弹层，与 onRowEnter 的判断保持一致
  if (window.innerWidth < 1024) return
  if (!rowEl.value || expanded.value) return
  const rect = rowEl.value.getBoundingClientRect()
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight
  const gap = 16
  const previewHeight = 460

  // 水平：默认在鼠标右侧；右侧空间不足先缩宽，仍不足则移到鼠标左侧
  let left = Math.round(event.clientX + gap)
  let width = Math.min(viewportWidth - left - gap, 840)
  if (width < 360) {
    width = Math.min(event.clientX - gap * 2, 840)
    left = Math.max(12, Math.round(event.clientX - gap - width))
  }
  width = Math.max(width, 300)

  // 垂直：与所在行顶部对齐，视口底部不足时整体上移
  let top = rect.top
  if (top + previewHeight > viewportHeight - 12) {
    top = Math.max(12, viewportHeight - previewHeight - 12)
  }

  popoverStyle.value = {
    top: `${Math.round(top)}px`,
    left: `${Math.round(left)}px`,
    width: `${width}px`,
  }
  expanded.value = true
}

function closePreview() {
  expanded.value = false
}

// 弹层展开期间，滚动或缩放视口即关闭，避免卡片停留位置错位
watch(expanded, (isExpanded) => {
  if (isExpanded) {
    window.addEventListener('scroll', closePreview, { passive: true })
    window.addEventListener('resize', closePreview)
  } else {
    window.removeEventListener('scroll', closePreview)
    window.removeEventListener('resize', closePreview)
  }
})

onBeforeUnmount(() => {
  window.clearTimeout(enterTimer)
  window.clearTimeout(leaveTimer)
  window.removeEventListener('scroll', closePreview)
  window.removeEventListener('resize', closePreview)
})
</script>

<template>
  <article
    ref="rowEl"
    class="group gf-topic-row"
    :class="[
      home ? 'gf-topic-row-home' : '',
      topic.pinWeight > 0 ? 'gf-topic-row-pinned' : '',
    ]"
    @mouseenter="onRowEnter"
    @mouseleave="onRowLeave"
  >
    <div class="min-w-0">
      <div class="flex min-h-6 min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
        <span class="inline-flex min-w-0 max-w-full items-center gap-2">
          <Pin
            v-if="showPinned && topic.pinWeight > 0"
            class="h-3.5 w-3.5 shrink-0 rotate-45 text-error"
            :aria-label="t('topicList.pinned')"
          />
          <a :href="topic.url" class="min-w-0 truncate text-[15px] font-medium leading-6 text-base-content group-hover:text-primary sm:text-base" @click="closePreview">
            {{ topic.title }}
          </a>
          <span
            v-if="topic.unseen"
            class="h-2 w-2 shrink-0 rounded-full bg-primary"
            aria-hidden="true"
          />
          <!-- Content type badge -->
          <span v-if="topic.contentType === 1" class="inline-flex h-5 items-center gap-1 rounded-full bg-success/20 px-1.5 text-[11px] font-semibold text-success">
            <HelpCircle class="h-3 w-3" />
            <span class="hidden sm:inline">{{ t('publish.contentTypes.question') }}</span>
          </span>
          <span v-else-if="topic.contentType === 2" class="inline-flex h-5 items-center gap-1 rounded-full bg-warning/20 px-1.5 text-[11px] font-semibold text-warning">
            <Lightbulb class="h-3 w-3" />
            <span class="hidden sm:inline">{{ t('publish.contentTypes.thought') }}</span>
          </span>
          <span v-else-if="topic.contentType === 3" class="inline-flex h-5 items-center gap-1 rounded-full bg-info/20 px-1.5 text-[11px] font-semibold text-info">
            <FileText class="h-3 w-3" />
            <span class="hidden sm:inline">{{ t('publish.contentTypes.article') }}</span>
          </span>
        </span>
        <a
          v-for="category in showCategories ? topic.categories : []"
          :key="category.id"
          :href="category.url"
          class="gf-topic-chip"
        >
          <span class="h-1.5 w-1.5 rounded-full" :style="{ backgroundColor: category.color }" />
          {{ category.name }}
        </a>
        <span v-if="showHot && topic.viewCount > 500" class="inline-flex h-5 items-center gap-1 text-[11px] font-semibold text-warning">
          <Sparkles class="h-3 w-3" /> hot
        </span>
      </div>
      <p class="mt-1 min-h-5 truncate text-[13px] leading-5 text-base-content/55">{{ topicDescription(topic) }}</p>
      <div class="mt-1.5 flex min-h-6 flex-wrap items-center gap-x-3 gap-y-1 text-xs text-base-content/55 lg:hidden">
        <AvatarStack :users="topic.participants" size="sm" />
        <span>{{ timeAgo(topic.lastUpdateTime) }}</span>
        <span class="inline-flex items-center gap-1">
          <MessageSquare class="h-3.5 w-3.5" /> {{ formatNumber(topic.replyCount) }}
        </span>
        <slot name="mobile-action" :topic="topic" />
      </div>
    </div>
    <div class="hidden justify-center lg:flex">
      <AvatarStack :users="topic.participants" />
    </div>
    <div class="hidden text-center text-sm font-semibold tabular-nums text-base-content/75 lg:block">{{ formatNumber(topic.replyCount) }}</div>
    <div class="hidden text-center text-sm tabular-nums text-base-content/55 lg:block">{{ formatNumber(topic.viewCount) }}</div>
    <div class="hidden text-right text-[13px] font-medium tabular-nums text-base-content/55 lg:block">
      <slot name="activity" :topic="topic">
        {{ timeAgo(topic.lastUpdateTime) }}
      </slot>
    </div>

    <Teleport to="body">
      <Transition name="preview-pop">
        <div
          v-if="expanded"
          class="fixed z-50 overflow-hidden rounded-xl border border-line bg-base-100 shadow-xl shadow-black/10"
          :style="popoverStyle"
          @mouseenter="onRowEnter"
          @mouseleave="onRowLeave"
          @click="closePreview"
        >
          <a :href="topic.url" class="block outline-none focus-visible:ring-2 focus-visible:ring-primary/50">
            <TopicFeedPreview :topic="topic" />
          </a>
        </div>
      </Transition>
    </Teleport>
  </article>
</template>

<style scoped>
/* 弹层入场：轻微上浮 + 果冻弹性回位；退场快速收敛 */
.preview-pop-enter-active {
  transition:
    opacity 160ms ease-out,
    transform 260ms cubic-bezier(0.34, 1.56, 0.64, 1);
}

.preview-pop-enter-from {
  opacity: 0;
  transform: translateY(10px) scale(0.96);
}

.preview-pop-leave-active {
  transition:
    opacity 110ms ease-out,
    transform 150ms ease-in;
}

.preview-pop-leave-to {
  opacity: 0;
  transform: translateY(4px) scale(0.98);
}
</style>
