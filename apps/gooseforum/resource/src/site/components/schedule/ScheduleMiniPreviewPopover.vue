<script setup lang="ts">
// 教学班选课后即时缩略课表预览气泡（v2：防裁切 + 智能碰撞避让 + 3次闪烁高亮 + 悬停暂停 + RAF节流 + 点外退场）。
// - Teleport 到 body，层级设为 z-[2300]，彻底避免外层 overflow-y-auto 及 DialogContent 裁剪。
// - 移动端：自适应为底部悬浮 HUD 卡片（touch-manipulation），消除窄屏溢出与手势阻碍。
// - 桌面端：根据加入按钮 getBoundingClientRect() 计算定位，右侧避让课评浮动面板，上下视口安全夹取。
// - 滚动与尺寸变化采用 requestAnimationFrame 节流，避免高频滚动触发 DOM 同步重排（Layout Thrashing）。
// - 点击浮窗外部非阻塞即时退场，不拦截外部点击事件；4s 黄金阅读时长后自动退场，悬停保持。
// - 新加入课程的时段以品牌主色（冲突时警示红色）带有 3 次柔和缩放脉冲（@keyframes）醒目高亮。
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  AlertTriangle,
  CalendarCheck2,
  CheckCircle2,
  Clock,
  X,
} from '@lucide/vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import type { PkArrangement, PkCourseDetail } from '@/site/types/pk'
import { conflictBaseOf } from '@/site/utils/pkConflict'
import { weeksOverlap } from '@/site/utils/pkArrange'

const props = withDefaults(
  defineProps<{
    open: boolean
    anchorEl?: HTMLElement | null
    stagedDetail?: PkCourseDetail | null
    courseName?: string
    isReviewOpen?: boolean
  }>(),
  {
    anchorEl: null,
    stagedDetail: null,
    courseName: '',
    isReviewOpen: false,
  },
)

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const store = useScheduleStore()

const popoverEl = ref<HTMLElement | null>(null)
const isMobile = ref(typeof window !== 'undefined' ? window.innerWidth < 768 : false)
const isPaused = ref(false)
const pos = ref({ top: 0, left: 0, placement: 'bottom' as 'top' | 'bottom' })
const arrowLeft = ref(160)

let autoCloseTimer: ReturnType<typeof setTimeout> | undefined
let rafId: number | null = null

/** 周几缩写（对应 1..7）。 */
const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const
const weekdayNames = computed(() => WEEKDAY_KEYS.map(day => t(`schedule.weekdays.${day}`)))

/** 课表总行数（通常 11 或 12 行）。 */
const maxRows = computed(() => store.readTimeTableRows())

/** 本次加入课程占用的 (day, period) 键集合，格式 "day_period"（day: 1..7, period: 1..maxRows）。 */
const stagedSlotSet = computed(() => {
  const set = new Set<string>()
  if (!props.stagedDetail) return set
  for (const arr of props.stagedDetail.arrangementInfo ?? []) {
    for (const time of arr.occupyTime ?? []) {
      set.add(`${arr.occupyDay}_${time}`)
    }
  }
  return set
})

/** 本次加入课程有冲突的时段集合。 */
const conflictedSlotSet = computed(() => {
  const set = new Set<string>()
  if (!props.stagedDetail) return set
  const base = conflictBaseOf(props.stagedDetail.code)
  for (const arr of props.stagedDetail.arrangementInfo ?? []) {
    for (const time of arr.occupyTime ?? []) {
      const cell = store.state.occupied[time - 1]?.[arr.occupyDay - 1] ?? []
      const hasConflict = cell.some(
        (item) => conflictBaseOf(item.code) !== base && weeksOverlap(arr.occupyWeek, item.occupyWeek),
      )
      if (hasConflict) {
        set.add(`${arr.occupyDay}_${time}`)
      }
    }
  }
  return set
})

/** 是否存在冲突。 */
const hasConflict = computed(() => conflictedSlotSet.value.size > 0)

/** 被课表中已有其他课程占用的 (day, period) 键集合（预计算 Set 实现 O(1) 检索，避免模板内反复循环）。 */
const existingSlotSet = computed(() => {
  const set = new Set<string>()
  const base = props.stagedDetail ? conflictBaseOf(props.stagedDetail.code) : ''
  for (let r = 0; r < maxRows.value; r++) {
    for (let d = 0; d < 7; d++) {
      const cell = store.state.occupied[r]?.[d] ?? []
      if (cell.some((item) => conflictBaseOf(item.code) !== base)) {
        set.add(`${d + 1}_${r + 1}`)
      }
    }
  }
  return set
})

/** 本次加入课程的安排摘要（如：周二 第3-4节 · 周四 第1-2节）。 */
const arrangementSummary = computed(() => {
  if (!props.stagedDetail?.arrangementInfo?.length) return ''
  return props.stagedDetail.arrangementInfo
    .map((arr) => {
      const day = weekdayNames.value[arr.occupyDay - 1] ?? ''
      const sections = arr.occupyTime?.length
        ? t('schedule.previewPeriods', { periods: arr.occupyTime.join(', ') })
        : ''
      return `${day} ${sections}`.trim()
    })
    .filter(Boolean)
    .join(' · ')
})

/** 本次课程占用的周几集合（用于高亮列头）。 */
const stagedDays = computed(() => {
  const days = new Set<number>()
  if (!props.stagedDetail) return days
  for (const arr of props.stagedDetail.arrangementInfo ?? []) {
    days.add(arr.occupyDay)
  }
  return days
})

// ---- 定位与视口避让（带 RAF 节流，避免 DOM 重排风暴）----

function updatePosition() {
  if (!props.open) return
  if (typeof window === 'undefined') return

  isMobile.value = window.innerWidth < 768
  if (isMobile.value) return

  if (!props.anchorEl) return
  const rect = props.anchorEl.getBoundingClientRect()

  // 仅在已有实际排版尺寸（非 headless/测试环境未布局）且滚出视口过远时静默关闭
  const hasLayout = rect.width > 0 || rect.height > 0 || rect.bottom > 0 || rect.top > 0
  if (hasLayout && (rect.bottom < 40 || rect.top > window.innerHeight - 40)) {
    emit('close')
    return
  }

  const popWidth = 320
  const popHeight = popoverEl.value?.offsetHeight || 300

  // 水平定位：
  // 教学班卡片上的加入按钮位于左栏右侧。
  // 右对齐按钮右边缘向左展开（left = rect.right - popWidth），
  // 在展开课评面板（桌面端右侧 380px）时绝对不会遮挡课评面板，也不会超出视口边界。
  let left = rect.right - popWidth
  if (left < 16) left = 16
  if (left + popWidth > window.innerWidth - 16) {
    left = Math.max(16, window.innerWidth - popWidth - 16)
  }

  // 垂直定位：优先向下，空间不足时向上
  const spaceBelow = window.innerHeight - rect.bottom - 12
  const spaceAbove = rect.top - 12
  let top = 0
  let placement: 'top' | 'bottom' = 'bottom'

  if (spaceBelow >= popHeight || spaceBelow >= spaceAbove) {
    top = rect.bottom + 8
    placement = 'bottom'
    if (top + popHeight > window.innerHeight - 16) {
      top = Math.max(16, window.innerHeight - popHeight - 16)
    }
  } else {
    top = rect.top - popHeight - 8
    placement = 'top'
    if (top < 16) top = 16
  }

  // 箭头居中对齐按钮中心
  const buttonCenterX = rect.left + rect.width / 2
  arrowLeft.value = Math.max(20, Math.min(popWidth - 20, buttonCenterX - left))

  pos.value = { top, left, placement }
}

/** 视口滚动/尺寸变化节流处理：通过 requestAnimationFrame 保证 60fps/120fps 平滑调度。 */
function onViewportChanged() {
  if (rafId !== null) return
  rafId = requestAnimationFrame(() => {
    rafId = null
    updatePosition()
  })
}

const desktopStyle = computed(() => ({
  top: `${pos.value.top}px`,
  left: `${pos.value.left}px`,
}))

// ---- 倒计时与交互暂停 ----

function startAutoClose(duration = 4000) {
  clearAutoClose()
  autoCloseTimer = setTimeout(() => {
    emit('close')
  }, duration)
}

function clearAutoClose() {
  if (autoCloseTimer) {
    clearTimeout(autoCloseTimer)
    autoCloseTimer = undefined
  }
}

function onMouseEnter() {
  isPaused.value = true
  clearAutoClose()
}

function onMouseLeave() {
  isPaused.value = false
  startAutoClose(2000) // 移出后给予 2 秒宽限时间
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.open) {
    e.stopPropagation()
    emit('close')
  }
}

/** 点击气泡外部即时退场（非阻塞：不拦截事件传播，保证外部按钮正常响应）。 */
function onDocumentPointerDown(event: PointerEvent) {
  if (!props.open) return
  const target = event.target as Node | null
  if (popoverEl.value?.contains(target) || props.anchorEl?.contains(target)) {
    return
  }
  emit('close')
}

function addListeners() {
  window.addEventListener('resize', onViewportChanged)
  window.addEventListener('scroll', onViewportChanged, true)
  window.addEventListener('keydown', onKeydown, true)
  document.addEventListener('pointerdown', onDocumentPointerDown, true)
}

function removeListeners() {
  if (rafId !== null) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
  window.removeEventListener('resize', onViewportChanged)
  window.removeEventListener('scroll', onViewportChanged, true)
  window.removeEventListener('keydown', onKeydown, true)
  document.removeEventListener('pointerdown', onDocumentPointerDown, true)
}

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      isPaused.value = false
      addListeners()
      nextTick(() => {
        updatePosition()
        startAutoClose(4000)
      })
    } else {
      clearAutoClose()
      removeListeners()
    }
  },
  { immediate: true },
)

watch(
  () => props.anchorEl,
  () => {
    if (props.open) {
      nextTick(updatePosition)
    }
  },
)

watch(
  () => props.isReviewOpen,
  () => {
    if (props.open) {
      nextTick(updatePosition)
      // 课评面板 300ms CSS 过渡期间重新对齐，避免动画过程中定位脱节
      setTimeout(updatePosition, 160)
      setTimeout(updatePosition, 320)
    }
  },
)

onMounted(() => {
  if (props.open) {
    addListeners()
  }
})

onBeforeUnmount(() => {
  clearAutoClose()
  removeListeners()
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 scale-95 translate-y-1"
      enter-to-class="opacity-100 scale-100 translate-y-0"
      leave-active-class="transition duration-150 ease-in pointer-events-none"
      leave-from-class="opacity-100 scale-100 translate-y-0"
      leave-to-class="opacity-0 scale-95 translate-y-1"
    >
      <div
        v-if="open"
        ref="popoverEl"
        role="status"
        aria-live="polite"
        data-testid="schedule-mini-preview-popover"
        class="fixed z-[2300] select-none touch-manipulation rounded-2xl border border-line/80 bg-base-100/98 p-3.5 shadow-2xl backdrop-blur-md dark:shadow-black/70 focus:outline-none"
        :class="[
          isMobile
            ? 'bottom-4 inset-x-3 max-w-sm mx-auto'
            : 'w-[320px]'
        ]"
        :style="isMobile ? {} : desktopStyle"
        @mouseenter="onMouseEnter"
        @mouseleave="onMouseLeave"
        @touchstart.passive="onMouseEnter"
        @touchend.passive="onMouseLeave"
      >
        <!-- 桌面端小三角箭头指示器（指向加入按钮） -->
        <div
          v-if="!isMobile"
          class="pointer-events-none absolute h-2.5 w-2.5 rotate-45 border border-line/80 bg-base-100"
          :class="pos.placement === 'bottom' ? '-top-[6px] border-b-0 border-r-0' : '-bottom-[6px] border-t-0 border-l-0'"
          :style="{ left: `${arrowLeft}px` }"
        />

        <!-- 气泡顶栏：标题 + 状态胶囊 + 关闭按钮 -->
        <div class="relative z-10 flex items-center justify-between gap-2 border-b border-line/50 pb-2.5">
          <div class="flex items-center gap-1.5 min-w-0">
            <span
              class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full"
              :class="hasConflict ? 'bg-error/15 text-error' : 'bg-primary/15 text-primary'"
            >
              <AlertTriangle v-if="hasConflict" class="h-3 w-3 stroke-[2.5]" />
              <CalendarCheck2 v-else class="h-3 w-3 stroke-[2.5]" />
            </span>
            <span class="truncate text-xs font-bold text-base-content">
              {{ t('schedule.previewPopoverTitle') }}
            </span>
          </div>

          <div class="flex items-center gap-1.5 shrink-0">
            <span
              class="inline-flex items-center rounded-md px-1.5 py-0.5 text-[10px] font-semibold"
              :class="hasConflict ? 'bg-error/15 text-error' : 'bg-success/15 text-success'"
            >
              {{ hasConflict ? t('schedule.previewPopoverConflict') : t('schedule.previewPopoverNoConflict') }}
            </span>

            <button
              type="button"
              class="gf-icon-button -mr-1 h-6 w-6 rounded-lg text-base-content/60 hover:bg-base-200 hover:text-base-content active:scale-[0.96] touch-manipulation"
              :aria-label="t('common.close')"
              @click="emit('close')"
            >
              <X class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>

        <!-- 课程信息与排课时段摘要 -->
        <div class="relative z-10 mt-2 mb-2.5">
          <div class="flex items-center justify-between gap-2 text-xs">
            <span class="truncate font-bold text-base-content" :title="courseName || stagedDetail?.code">
              {{ courseName || stagedDetail?.code }}
            </span>
            <span v-if="stagedDetail?.code" class="shrink-0 rounded bg-base-200 px-1 py-0.2 text-[10px] font-mono font-medium text-base-content/70">
              {{ stagedDetail.code }}
            </span>
          </div>
          <p v-if="arrangementSummary" class="mt-0.5 text-[11px] text-base-content/65 flex items-center gap-1">
            <Clock class="h-3 w-3 shrink-0 text-base-content/40" />
            <span class="truncate">{{ arrangementSummary }}</span>
          </p>
        </div>

        <!-- 缩略课表网格（7列代表周一至周日，11/12行代表各节次） -->
        <div class="relative z-10 rounded-xl border border-line/60 bg-base-200/30 p-2">
          <!-- 星期列头 -->
          <div class="grid grid-cols-[14px_repeat(7,1fr)] gap-1 text-center mb-1">
            <span class="text-[9px] text-base-content/30">#</span>
            <span
              v-for="(dayName, idx) in weekdayNames"
              :key="idx"
              class="text-[10px] font-medium transition-colors"
              :class="stagedDays.has(idx + 1) ? 'font-bold text-primary' : 'text-base-content/50'"
            >
              {{ dayName }}
            </span>
          </div>

          <!-- 节次网格行 -->
          <div class="space-y-1">
            <div
              v-for="period in maxRows"
              :key="period"
              class="grid grid-cols-[14px_repeat(7,1fr)] gap-1 items-center"
            >
              <!-- 节次序号 -->
              <span class="text-center font-mono text-[9px] tabular-nums text-base-content/40">
                {{ period }}
              </span>

              <!-- 7 天对应的微型格子 -->
              <div
                v-for="day in 7"
                :key="day"
                class="relative h-3 rounded-[3px] transition-all"
                :class="[
                  stagedSlotSet.has(`${day}_${period}`)
                    ? (conflictedSlotSet.has(`${day}_${period}`)
                        ? 'bg-error text-error-content shadow-xs ring-1 ring-error/60 animate-slot-pulse'
                        : 'bg-primary text-primary-content shadow-xs ring-1 ring-primary/60 animate-slot-pulse')
                    : existingSlotSet.has(`${day}_${period}`)
                      ? 'bg-base-content/25 dark:bg-base-content/30 border border-base-content/10'
                      : 'bg-base-100/70 dark:bg-base-100/40 border border-line/40'
                ]"
                :title="
                  stagedSlotSet.has(`${day}_${period}`)
                    ? `${weekdayNames[day - 1]} ${t('schedule.previewPeriods', { periods: period })}: ${t('schedule.previewPopoverNew')}`
                    : existingSlotSet.has(`${day}_${period}`)
                      ? `${weekdayNames[day - 1]} ${t('schedule.previewPeriods', { periods: period })}: ${t('schedule.previewPopoverExisting')}`
                      : `${weekdayNames[day - 1]} ${t('schedule.previewPeriods', { periods: period })}`
                "
              />
            </div>
          </div>
        </div>

        <!-- 底部图例与自动退场提示 -->
        <div class="relative z-10 mt-2.5 flex items-center justify-between gap-2 text-[10px] text-base-content/60">
          <div class="flex items-center gap-2.5">
            <span class="inline-flex items-center gap-1 font-medium">
              <span class="h-2 w-2 rounded-[2px] bg-primary" />
              <span>{{ t('schedule.previewPopoverNew') }}</span>
            </span>
            <span class="inline-flex items-center gap-1 font-medium">
              <span class="h-2 w-2 rounded-[2px] bg-base-content/30" />
              <span>{{ t('schedule.previewPopoverExisting') }}</span>
            </span>
            <span v-if="hasConflict" class="inline-flex items-center gap-1 font-medium text-error">
              <span class="h-2 w-2 rounded-[2px] bg-error" />
              <span>{{ t('schedule.previewPopoverConflict') }}</span>
            </span>
          </div>

          <span class="text-[9px] text-base-content/40 tabular-nums">
            {{ isPaused ? t('schedule.previewPopoverAutoClose') : t('schedule.previewPopoverAutoClose') }}
          </span>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
@keyframes mini-slot-pulse {
  0%, 100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.18);
    opacity: 0.85;
    box-shadow: 0 0 0 3px color-mix(in oklab, currentColor 40%, transparent);
  }
}

.animate-slot-pulse {
  animation: mini-slot-pulse 0.6s ease-in-out 3;
  z-index: 20;
}
</style>
