<script setup lang="ts">
// 外部选课与排课工具推荐（小且醒目的气泡 Popover）：
// 1. 同济排课助手 (xk.xialing.icu)：老版排课模拟器，模拟一系统体验
// 2. 通济-模拟选课系统 (course.f1justin.com)：强大的课程筛选工具
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CalendarClock,
  ExternalLink,
  Hammer,
  SlidersHorizontal,
  X,
} from '@lucide/vue'
import {
  PopoverArrow,
  PopoverClose,
  PopoverContent,
  PopoverPortal,
  PopoverRoot,
  PopoverTrigger,
} from 'reka-ui'

const { t } = useI18n()
const open = ref(false)
const pinned = ref(false)
let hoverTimer: ReturnType<typeof setTimeout> | null = null

function handleMouseEnter() {
  if (pinned.value) return
  if (hoverTimer) clearTimeout(hoverTimer)
  hoverTimer = setTimeout(() => {
    open.value = true
  }, 120)
}

function handleMouseLeave() {
  if (pinned.value) return
  if (hoverTimer) clearTimeout(hoverTimer)
  hoverTimer = setTimeout(() => {
    open.value = false
  }, 240)
}

function handleContentMouseEnter() {
  if (pinned.value) return
  if (hoverTimer) {
    clearTimeout(hoverTimer)
    hoverTimer = null
  }
}

function handleContentMouseLeave() {
  if (pinned.value) return
  if (hoverTimer) clearTimeout(hoverTimer)
  hoverTimer = setTimeout(() => {
    open.value = false
  }, 240)
}

function handleTriggerClick() {
  if (open.value && pinned.value) {
    open.value = false
    pinned.value = false
  } else {
    open.value = true
    pinned.value = true
  }
}

function handleOpenChange(val: boolean) {
  open.value = val
  if (!val) {
    pinned.value = false
  }
}

onBeforeUnmount(() => {
  if (hoverTimer) clearTimeout(hoverTimer)
})

interface ExternalToolItem {
  id: string
  title: string
  desc: string
  domain: string
  url: string
  icon: typeof CalendarClock
}

const tools = computed<ExternalToolItem[]>(() => [
  {
    id: 'xk-assistant',
    title: t('schedule.externalTool1Title'),
    desc: t('schedule.externalTool1Desc'),
    domain: 'xk.xialing.icu',
    url: 'https://xk.xialing.icu/',
    icon: CalendarClock,
  },
  {
    id: 'tongji-course-explorer',
    title: t('schedule.externalTool2Title'),
    desc: t('schedule.externalTool2Desc'),
    domain: 'course.f1justin.com',
    url: 'https://course.f1justin.com/',
    icon: SlidersHorizontal,
  },
])
</script>

<template>
  <PopoverRoot :open="open" @update:open="handleOpenChange">
    <!-- 小且精致的工具气泡触发胶囊 -->
    <PopoverTrigger as-child>
      <button
        type="button"
        class="group inline-flex items-center gap-1.5 rounded-full border border-line/70 bg-base-100 px-2.5 py-1 text-[11px] font-medium text-base-content/75 transition-all duration-150 hover:border-primary/40 hover:bg-primary/5 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 active:scale-[0.96] cursor-pointer select-none"
        :class="{ 'bg-primary/10 border-primary/45 text-primary ring-1 ring-primary/25': open }"
        :aria-label="t('schedule.externalToolsTitle')"
        data-testid="schedule-external-tools-trigger"
        @mouseenter="handleMouseEnter"
        @mouseleave="handleMouseLeave"
        @click="handleTriggerClick"
      >
        <Hammer
          class="h-3.5 w-3.5 shrink-0 text-base-content/65 transition-all duration-200 group-hover:scale-110 group-hover:rotate-12 group-hover:text-primary"
          :class="{ 'text-primary rotate-12': open }"
        />
        <span>{{ t('schedule.externalToolsBadge') }}</span>
      </button>
    </PopoverTrigger>

    <!-- 气泡弹出浮层（通过 Portal 挂载，彻底杜绝容器溢出截断） -->
    <PopoverPortal>
      <PopoverContent
        side="bottom"
        align="start"
        :side-offset="8"
        :collision-padding="16"
        class="z-[2200] w-[min(352px,calc(100vw-2rem))] rounded-2xl border border-line/80 bg-base-100/95 p-3.5 shadow-2xl backdrop-blur-xl outline-none text-xs text-base-content animate-in fade-in-0 zoom-in-95 duration-150"
        data-testid="schedule-external-tools-popover"
        @mouseenter="handleContentMouseEnter"
        @mouseleave="handleContentMouseLeave"
      >
        <!-- 气泡顶栏：标题 + 描述 + 关闭按钮 -->
        <div class="mb-3 flex items-start justify-between gap-2 border-b border-line/50 pb-2.5">
          <div class="flex items-center gap-2 min-w-0">
            <div
              class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary shadow-2xs"
            >
              <Hammer class="h-3.5 w-3.5" />
            </div>
            <div class="min-w-0">
              <h4 class="text-xs font-bold text-base-content leading-tight">
                {{ t('schedule.externalToolsTitle') }}
              </h4>
              <p class="text-[10.5px] text-base-content/60 leading-tight mt-0.5">
                {{ t('schedule.externalToolsSubtitle') }}
              </p>
            </div>
          </div>

          <PopoverClose as-child>
            <button
              type="button"
              class="flex h-6 w-6 shrink-0 items-center justify-center rounded-lg text-base-content/40 hover:bg-base-200 hover:text-base-content transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 active:scale-95"
              :aria-label="t('schedule.close')"
            >
              <X class="h-3.5 w-3.5" />
            </button>
          </PopoverClose>
        </div>

        <!-- 外部工具列表（无截断、完整展示、带箭头提示） -->
        <div class="space-y-2">
          <a
            v-for="tool in tools"
            :key="tool.id"
            :href="tool.url"
            target="_blank"
            rel="noopener noreferrer"
            class="group flex items-start justify-between gap-2.5 rounded-xl border border-line/60 bg-base-200/30 p-2.5 transition-all duration-150 hover:border-primary/40 hover:bg-base-200/70 hover:shadow-2xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 active:scale-[0.99]"
            :aria-label="`${tool.title} - ${tool.desc} (${t('schedule.externalToolOpen')})`"
          >
            <div class="flex min-w-0 flex-1 items-start gap-2.5">
              <!-- 同心圆角图标徽标 -->
              <div
                class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-line/60 bg-base-100 text-base-content/70 transition-colors duration-150 group-hover:border-primary/30 group-hover:bg-primary/10 group-hover:text-primary"
              >
                <component :is="tool.icon" class="h-3.5 w-3.5" />
              </div>

              <!-- 文案与域名微标签 -->
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-1.5">
                  <span
                    class="text-xs font-bold text-base-content transition-colors group-hover:text-primary"
                  >
                    {{ tool.title }}
                  </span>
                  <span
                    class="inline-flex items-center rounded border border-line/40 bg-base-100 px-1.5 py-0.2 font-mono text-[9.5px] text-base-content/60"
                  >
                    {{ tool.domain }}
                  </span>
                </div>
                <p class="mt-1 text-[11px] leading-relaxed text-base-content/70 break-words">
                  {{ tool.desc }}
                </p>
              </div>
            </div>

            <!-- 外跳指示箭头 -->
            <div
              class="shrink-0 pt-0.5 text-base-content/40 transition-all duration-150 group-hover:-translate-y-0.5 group-hover:translate-x-0.5 group-hover:text-primary"
            >
              <ExternalLink class="h-3.5 w-3.5" />
            </div>
          </a>
        </div>

        <!-- 气泡底栏微提示 -->
        <div
          class="mt-2.5 flex items-center justify-between border-t border-line/40 pt-2 text-[10px] text-base-content/50"
        >
          <span>{{ t('schedule.externalToolOpen') }}</span>
        </div>

        <!-- 气泡箭头指示角 -->
        <PopoverArrow class="fill-base-100 stroke-line/80" />
      </PopoverContent>
    </PopoverPortal>
  </PopoverRoot>
</template>
