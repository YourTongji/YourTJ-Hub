<script setup lang="ts">
// 独立课表图片导出生成器：
// - 柔和无边框暗场毛玻璃悬浮浮层 (Borderless Floating Stage)；
// - 移植 Jakub Antalik 的 img-fx (https://github.com/Jakubantalik/img-fx) 像素马赛克流光与对角粒子消融显现动效；
// - 课表下方独立悬浮交互按钮坞（关闭、重放动效、复制、下载）；
// - 固定 1140px 高清画幅，双向动态等比缩放自适应任何屏幕尺寸，杜绝视口容器切割与内容截断；
// - 品牌标识与主标题在水平对位轴上严密对称平衡，彻底杜绝 AI slop 模板感。
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toPng } from 'html-to-image'
import { Check, Clock, Copy, Download, Loader2, MapPin, User, X } from '@lucide/vue'
import {
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'
import logoUrl from '@/site/assets/logo.svg?url'
import { queueFlashMessage } from '@/runtime/flash-message'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { courseColorSlotFor, courseSlotVar } from '@/site/utils/courseColors'
import {
  detectWeekParity,
  formatWeeksText,
} from '@/site/utils/pkArrange'
import { conflictBaseOf, CUSTOM_EVENT_CODE_PREFIX } from '@/site/utils/pkConflict'
import { dayPartBoundaries, sectionTimesFor, type DayPart } from '@/site/utils/sectionTimes'
import { ImgFxController } from '@/site/utils/imgFxCanvas'
import type { PkCalendar, PkCourseOnTable } from '@/site/types/pk'

const CANVAS_WIDTH = 1140
const CANVAS_ESTIMATED_HEIGHT = 880

const { t } = useI18n()
const store = useScheduleStore()

const props = defineProps<{
  open: boolean
  cellCourses: PkCourseOnTable[][][]
  cellSpans: number[][]
  occupiedGrid: boolean[][]
  activeCalendar: PkCalendar | null
  activePlanName: string
  weekViewLabel: string
  exportSummaryText: string
}>()

const emit = defineEmits<{
  close: []
}>()

const dialogOpen = computed({
  get: () => props.open,
  set: (val: boolean) => {
    if (!val) emit('close')
  },
})

const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const
const WEEKDAY_EN = ['MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT', 'SUN'] as const

const posterCanvasRef = ref<HTMLElement | null>(null)
const posterStageRef = ref<HTMLElement | null>(null)
const dockRef = ref<HTMLElement | null>(null)
const imgFxCanvasRef = ref<HTMLCanvasElement | null>(null)
const imgFxController = ref<ImgFxController | null>(null)

let pointerDownTarget: Node | null = null

function handlePointerDown(e: PointerEvent) {
  pointerDownTarget = e.target as Node | null
}

function handleBackdropClick(e: MouseEvent) {
  const target = e.target as Node | null
  if (!target) return
  // 如果交互起始或结束于海报卡片或操作坞，不触发退出（防止拖拽划选或误触）
  if (posterStageRef.value?.contains(pointerDownTarget)) return
  if (dockRef.value?.contains(pointerDownTarget)) return
  if (posterStageRef.value?.contains(target)) return
  if (dockRef.value?.contains(target)) return

  dialogOpen.value = false
}

const fitScale = ref(0.68)
const actualHeight = ref(CANVAS_ESTIMATED_HEIGHT)
const generating = ref(false)
const copied = ref(false)
const isRevealed = ref(false)
const isCanvasReady = ref(false)

// ---- 视口防切割自适应缩放计算 ----
function updateFitScale() {
  if (typeof window === 'undefined') return
  const vw = window.innerWidth
  const vh = window.innerHeight
  // 留足上下浮动栏高度（操作坞高度与呼吸边距约 140px），以及左右边距
  const availW = vw * 0.92
  const availH = vh - 140
  const scaleW = availW / CANVAS_WIDTH
  const h = posterCanvasRef.value?.offsetHeight || actualHeight.value
  const scaleH = availH / h
  const target = Math.min(1, Math.max(0.24, Math.min(scaleW, scaleH)))
  fitScale.value = Number(target.toFixed(3))
}

// ---- 图片导出核心生成逻辑 ----
async function generateImageDataUrl(): Promise<string | null> {
  const el = posterCanvasRef.value
  if (!el) return null
  return await toPng(el, {
    pixelRatio: 2,
    backgroundColor: '#ffffff',
    cacheBust: true,
  })
}

// ---- img-fx 流光动效与显像流水线 ----
function runRevealAnimation() {
  updateFitScale()

  if (posterCanvasRef.value && imgFxCanvasRef.value) {
    const w = CANVAS_WIDTH
    const h = posterCanvasRef.value.offsetHeight || CANVAS_ESTIMATED_HEIGHT
    actualHeight.value = h

    if (!imgFxController.value) {
      imgFxController.value = new ImgFxController(imgFxCanvasRef.value, {
        cellSize: 22,
        gap: 0.5,
        holdDurationMs: 1200,
        sweepDurationMs: 2400,
      })
    }
    imgFxController.value.resize(w, h)
    imgFxController.value.play(() => {
      isRevealed.value = true
    })
    isCanvasReady.value = true
  } else {
    isRevealed.value = true
    isCanvasReady.value = true
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      copied.value = false
      isRevealed.value = false
      isCanvasReady.value = false
      nextTick(() => {
        updateFitScale()
        runRevealAnimation()
      })
    } else {
      isCanvasReady.value = false
      if (imgFxController.value) {
        imgFxController.value.destroy()
        imgFxController.value = null
      }
    }
  },
)

onMounted(() => {
  window.addEventListener('resize', updateFitScale)
})

onUnmounted(() => {
  window.removeEventListener('resize', updateFitScale)
  if (imgFxController.value) {
    imgFxController.value.destroy()
    imgFxController.value = null
  }
})

// ---- 课程与节次辅助方法 ----
function isCustomEvent(course: PkCourseOnTable): boolean {
  return course.code.startsWith(CUSTOM_EVENT_CODE_PREFIX)
}

function courseCardStyle(course: PkCourseOnTable): Record<string, string> {
  if (isCustomEvent(course)) {
    return {
      '--card-accent': '#475569',
      '--card-bg': '#f8fafc',
      '--card-border': '#94a3b8',
      '--card-title': '#090d16',
      '--card-sub': '#1e293b',
      backgroundColor: '#f8fafc',
      borderColor: '#94a3b8',
      color: '#090d16',
    }
  }
  const seed = course.code || course.courseName || course.showText || 'course'
  const slot = courseColorSlotFor(seed)
  const slotVar = courseSlotVar(slot)
  return {
    '--card-accent': `var(${slotVar})`,
    '--card-bg': `color-mix(in oklab, var(${slotVar}) 16%, #ffffff)`,
    '--card-border': `color-mix(in oklab, var(${slotVar}) 32%, #94a3b8)`,
    '--card-title': '#090d16',
    '--card-sub': '#1e293b',
    backgroundColor: `color-mix(in oklab, var(${slotVar}) 16%, #ffffff)`,
    borderColor: `color-mix(in oklab, var(${slotVar}) 32%, #94a3b8)`,
    color: '#090d16',
  }
}

function teacherName(course: PkCourseOnTable): string {
  return String(course.teacherAndCode || '').replace(/\([^)]*\)$/g, '').trim()
}

function compactTeacherName(raw: string): string {
  if (!raw) return ''
  const teachers = raw.split(/[,，、]/).map((s) => s.trim()).filter(Boolean)
  if (teachers.length <= 3) return teachers.join('、')
  return `${teachers.slice(0, 2).join('、')} 等`
}

function weekParityBadge(weeks: readonly number[] | undefined): string {
  if (!weeks || weeks.length === 0) return ''
  const parity = detectWeekParity(weeks)
  if (parity === 'odd') return t('schedule.parityOdd')
  if (parity === 'even') return t('schedule.parityEven')
  return ''
}

function formatDisplayWeeks(weeks: readonly number[] | undefined): string {
  if (!weeks || weeks.length === 0) return ''
  const parity = detectWeekParity(weeks)
  const sorted = [...new Set(weeks)].sort((a, b) => a - b)
  if (parity === 'odd' && sorted.length >= 3) {
    const isRegularStep = sorted.every((w, i) => i === 0 || w === sorted[i - 1] + 2)
    if (isRegularStep) return `${sorted[0]}-${sorted[sorted.length - 1]}周(单)`
  }
  if (parity === 'even' && sorted.length >= 3) {
    const isRegularStep = sorted.every((w, i) => i === 0 || w === sorted[i - 1] + 2)
    if (isRegularStep) return `${sorted[0]}-${sorted[sorted.length - 1]}周(双)`
  }
  return t('schedule.weeksN', { range: formatWeeksText(weeks) })
}

/**
 * 动态计算每行的基准与扩展高度：
 * 当同一行/节次跨度中有某天存在单双周多门课纵向堆叠（如周五两门课）时，
 * 该节次行会自动增高；此处统筹整网格所有单元格的最小空间需求，
 * 使得同行跨同样节次的单门课（如周一、周三）也能自动对齐均分/撑满整个扩展后的行高，
 * 绝不会出现单双周两门课很高而单门课下面留出大片空白的割裂现象。
 *
 * 排序策略：同 span 内，多门课格子（count > 1）先处理，确保行高先被真实内容需求撑高。
 */
const computedRowHeights = computed<number[]>(() => {
  const rowCount = props.cellCourses.length
  if (rowCount === 0) return []
  const rowHeights = new Array(rowCount).fill(76)

  interface CellInfo {
    span: number
    rIndex: number
    dayIndex: number
    count: number
  }

  const cells: CellInfo[] = []
  for (let r = 0; r < rowCount; r++) {
    const row = props.cellCourses[r]
    if (!row) continue
    for (let d = 0; d < 7; d++) {
      if (!props.occupiedGrid?.[r]?.[d]) {
        const span = props.cellSpans?.[r]?.[d] || 1
        const count = row[d]?.length || 0
        cells.push({ span, rIndex: r, dayIndex: d, count })
      }
    }
  }

  // 主排序：span 升序（小跨度先确定基准），次排序：count 降序（同 span 内多门课先撑高行高）
  cells.sort((a, b) => a.span - b.span || b.count - a.count)

  for (const { span, rIndex, count } of cells) {
    if (count === 0) continue
    let reqInner = 0
    if (count === 1) {
      // 单门课只需填满当前行高，不主动拉伸（行高由多门课决定）
      reqInner = Math.max(span * 76 - 8, 68)
    } else {
      // 多门课纵向堆叠（如单双周），紧凑卡片高度约 90px，间距 4px
      reqInner = count * 90 + (count - 1) * 4
    }
    const reqTotal = reqInner + 8 // 8px 为 td 上下 padding
    let curTotal = 0
    for (let i = 0; i < span; i++) {
      curTotal += rowHeights[rIndex + i] || 76
    }
    if (reqTotal > curTotal) {
      const diff = reqTotal - curTotal
      const perRow = Math.ceil(diff / span)
      for (let i = 0; i < span; i++) {
        const idx = rIndex + i
        if (idx < rowCount) {
          rowHeights[idx] += perRow
        }
      }
    }
  }

  return rowHeights
})

function cellInnerHeight(rIndex: number, dayIndex: number): number {
  const span = props.cellSpans?.[rIndex]?.[dayIndex] || 1
  let totalH = 0
  for (let i = 0; i < span; i++) {
    totalH += computedRowHeights.value[rIndex + i] || 76
  }
  return Math.max(68, totalH - 8)
}

function cardMinHeight(rIndex: number, dayIndex: number, courseCount: number): number {
  const innerH = cellInnerHeight(rIndex, dayIndex)
  const count = Math.max(1, courseCount)
  // 单门课：直接返回格子全高，彻底撑满，消除下半截空白
  if (count === 1) {
    return Math.max(68, innerH)
  }
  // 多门叠放：均分可用高度（扣除各卡片间 gap-1 = 4px）
  const available = innerH - (count - 1) * 4
  return Math.max(68, Math.floor(available / count))
}

const sectionTimes = computed(() => sectionTimesFor(store.readTimeTableRows(), store.state.sectionTimeOverrides))

function sectionTimeText(index: number): string {
  const times = sectionTimes.value[index]
  if (!times) return ''
  return `${times.start}-${times.end}`
}

const dayPartStarts = computed(() => dayPartBoundaries(sectionTimes.value))

function dayPartLabelAt(index: number): string | null {
  const row = index + 1
  for (const [part, start] of Object.entries(dayPartStarts.value)) {
    if (start === row) {
      const key = part as DayPart
      return t(`schedule.dayPart.${key}`)
    }
  }
  return null
}

/** 格式化导出时间戳 */
const exportTimestamp = computed(() => {
  const now = new Date()
  const yyyy = now.getFullYear()
  const mm = String(now.getMonth() + 1).padStart(2, '0')
  const dd = String(now.getDate()).padStart(2, '0')
  const hh = String(now.getHours()).padStart(2, '0')
  const mi = String(now.getMinutes()).padStart(2, '0')
  return `${yyyy}.${mm}.${dd} ${hh}:${mi}`
})

async function handleDownload() {
  if (generating.value) return
  generating.value = true
  try {
    const dataUrl = await generateImageDataUrl()
    if (!dataUrl) throw new Error('Render failed')
    const filename = `YourTJ_Schedule_${props.activeCalendar?.calendarName || 'Timetable'}_${props.activePlanName}.png`
    const link = document.createElement('a')
    link.download = filename
    link.href = dataUrl
    link.click()
  } catch {
    queueFlashMessage(t('schedule.exportImageFailed'), 'error')
  } finally {
    generating.value = false
  }
}

async function handleCopy() {
  if (generating.value) return
  generating.value = true
  try {
    const dataUrl = await generateImageDataUrl()
    if (!dataUrl) throw new Error('Render failed')
    const res = await fetch(dataUrl)
    const blob = await res.blob()
    await navigator.clipboard.write([
      new ClipboardItem({ 'image/png': blob }),
    ])
    copied.value = true
    queueFlashMessage(t('schedule.exportCopySuccess'), 'success')
    setTimeout(() => {
      copied.value = false
    }, 2500)
  } catch {
    queueFlashMessage(t('schedule.exportCopyFailed'), 'error')
  } finally {
    generating.value = false
  }
}
</script>

<template>
  <DialogRoot v-model:open="dialogOpen">
    <DialogPortal>
      <!-- 柔和暗场毛玻璃背板（点击背景区域快速轻量退出 Light Dismiss） -->
      <DialogOverlay
        class="fixed inset-0 z-[2100] bg-slate-950/80 backdrop-blur-md transition-opacity duration-300 animate-in fade-in cursor-pointer"
        @click="dialogOpen = false"
      />

      <!-- 无边框轻盈浮层容器（点击海报焦点外空白区域退出预览） -->
      <DialogContent
        class="fixed inset-0 z-[2101] flex flex-col items-center justify-center p-3 sm:p-5 md:p-6 outline-none border-none bg-transparent shadow-none select-none animate-in fade-in zoom-in-95 duration-250 cursor-pointer"
        @pointerdown="handlePointerDown"
        @click="handleBackdropClick"
      >
        <DialogTitle class="sr-only">
          {{ t('schedule.exportPreviewTitle') }}
        </DialogTitle>
        <DialogDescription class="sr-only">
          {{ t('schedule.exportPreviewSubtitle') }}
        </DialogDescription>

        <!-- 居中课表海报舞台（动态双向等比缩放，严格杜绝容器切割与视口溢出） -->
        <div
          ref="posterStageRef"
          class="relative flex items-center justify-center cursor-default"
          :style="{
            width: `${CANVAS_WIDTH * fitScale}px`,
            height: `${actualHeight * fitScale}px`,
          }"
        >
          <!-- 缩放卡片实体 -->
          <div
            class="absolute origin-center transition-transform duration-200"
            :style="{
              width: `${CANVAS_WIDTH}px`,
              transform: `scale(${fitScale})`,
            }"
          >
            <!-- 实际 1140px 独立高清课表海报画布 -->
            <div
              ref="posterCanvasRef"
              class="relative select-none rounded-2xl bg-white p-6 text-slate-800 border border-slate-200/80 shadow-2xl shadow-black/45"
              :style="{
                width: `${CANVAS_WIDTH}px`,
                backgroundColor: '#ffffff',
                backgroundImage: 'radial-gradient(circle, #e2e8f0 1.2px, transparent 1.2px)',
                backgroundSize: '20px 20px',
                fontFamily: 'ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, \'Segoe UI\', Roboto, sans-serif',
              }"
            >
              <!-- 初始化防闪烁护盾（首帧渲染就绪前 100% 遮蔽底图，彻底杜绝闪现） -->
              <div
                v-if="!isCanvasReady"
                class="pointer-events-none absolute inset-0 z-30 rounded-2xl bg-[#f8fafc]"
              />

              <!-- img-fx 像素马赛克流光与消融显像层 -->
              <canvas
                ref="imgFxCanvasRef"
                class="pointer-events-none absolute inset-0 z-30 rounded-2xl transition-opacity duration-300"
              />

              <!-- 极简转角十字标 -->
              <div class="pointer-events-none absolute left-3 top-3 font-mono text-[11px] font-bold text-slate-300 select-none">＋</div>
              <div class="pointer-events-none absolute right-3 top-3 font-mono text-[11px] font-bold text-slate-300 select-none">＋</div>
              <div class="pointer-events-none absolute left-3 bottom-3 font-mono text-[11px] font-bold text-slate-300 select-none">＋</div>
              <div class="pointer-events-none absolute right-3 bottom-3 font-mono text-[11px] font-bold text-slate-300 select-none">＋</div>

              <!-- 海报顶部横幅：双层水平轴对称结构，实现品牌标识与各布局要素的完美重心平衡 (/better-layout & /better-ui) -->
              <header class="border-b border-slate-200/90 pb-3.5 space-y-2.5">
                <!-- 顶层主轴：左侧标题与学期，右侧品牌标识徽标，顶边缘基线完美对齐 -->
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-3">
                    <h1 class="text-2xl font-black text-slate-950 tracking-tight leading-none">
                      {{ t('schedule.timetable') }}
                    </h1>
                    <span
                      v-if="activeCalendar?.calendarName"
                      class="inline-flex items-center whitespace-nowrap rounded-lg bg-slate-100/90 border border-slate-200/90 px-2.5 py-1 font-mono text-xs font-bold text-slate-700"
                    >
                      {{ activeCalendar.calendarName }}
                    </span>
                  </div>

                  <!-- 品牌标识：作为第一行顶层右侧锚点，与主标题形成水平对位呼应，消除孤立浮空感 -->
                  <div class="flex shrink-0 items-center gap-2 rounded-xl border border-slate-200/90 bg-white px-3 py-1.5 shadow-2xs">
                    <img :src="logoUrl" alt="YourTJ Logo" class="h-4 w-4 object-contain shrink-0" />
                    <span class="text-xs font-bold text-slate-800 tracking-tight whitespace-nowrap">
                      {{ t('schedule.exportBrandTag') }}
                    </span>
                  </div>
                </div>

                <!-- 次层信息轴：左侧方案、周次与汇总统计，右侧生成时间戳，底边缘基线对齐 -->
                <div class="flex items-center justify-between text-xs">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="inline-flex items-center whitespace-nowrap rounded-lg bg-slate-100 border border-slate-300/80 px-2.5 py-1 font-bold text-slate-800">
                      {{ activePlanName }}
                    </span>
                    <span class="inline-flex items-center whitespace-nowrap rounded-lg bg-primary/10 border border-primary/30 px-2.5 py-1 font-bold text-primary">
                      {{ weekViewLabel }}
                    </span>
                    <span v-if="exportSummaryText" class="inline-flex items-center whitespace-nowrap text-slate-700 font-bold pl-1">
                      {{ exportSummaryText }}
                    </span>
                  </div>

                  <!-- 时间戳：作为次层右侧锚点，平衡右下方视觉重心 -->
                  <div class="flex items-center gap-1.5 font-mono text-[11px] font-semibold text-slate-500 tabular-nums">
                    <Clock class="h-3.5 w-3.5 text-slate-400 stroke-[2]" />
                    <span>{{ exportTimestamp }}</span>
                  </div>
                </div>
              </header>

              <!-- 课表网格结构 (固定 1140px 排版，列宽与行高舒展充足) -->
              <div class="mt-4 overflow-hidden rounded-xl border border-slate-300 bg-white/95 shadow-2xs">
                <table class="w-full table-fixed border-collapse">
                  <thead>
                    <tr class="h-10 bg-slate-100/90 border-b border-slate-300">
                      <th class="w-[88px] border-r border-slate-300 p-1 text-center font-mono text-xs font-black text-slate-700 uppercase tracking-wider">
                        {{ t('schedule.arrangement') }}
                      </th>
                      <th
                        v-for="(day, dIndex) in WEEKDAY_KEYS"
                        :key="day"
                        class="border-r last:border-r-0 border-slate-300 p-2 text-center"
                      >
                        <div class="font-black text-[13px] text-slate-900 tracking-tight">
                          {{ t(`schedule.weekdays.${day}`) }}
                        </div>
                        <div class="font-mono text-[10px] font-bold text-slate-600 tracking-wider">
                          {{ WEEKDAY_EN[dIndex] }}
                        </div>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="(row, rIndex) in cellCourses"
                      :key="rIndex"
                      class="border-b last:border-b-0 border-slate-200/80"
                      :style="{ height: `${computedRowHeights[rIndex]}px` }"
                      :class="rIndex % 2 === 0 ? 'bg-white' : 'bg-slate-50/40'"
                    >
                      <!-- 节次与时间侧栏 -->
                      <td class="h-px border-r border-slate-300 p-1 text-center align-middle bg-slate-100/40">
                        <span v-if="dayPartLabelAt(rIndex)" class="mb-0.5 block text-[10px] font-black text-primary">
                          {{ dayPartLabelAt(rIndex) }}
                        </span>
                        <div class="font-mono text-sm font-black text-slate-900 leading-tight">
                          {{ String(rIndex + 1).padStart(2, '0') }}
                        </div>
                        <span v-if="sectionTimeText(rIndex)" class="mt-0.5 block font-mono text-[9.5px] font-bold text-slate-600 tabular-nums leading-none">
                          {{ sectionTimeText(rIndex) }}
                        </span>
                      </td>

                      <!-- 7 列课程格 -->
                      <template v-for="(courses, dayIndex) in row" :key="dayIndex">
                        <td
                          v-if="!occupiedGrid[rIndex][dayIndex]"
                          :rowspan="cellSpans[rIndex][dayIndex]"
                          class="h-px border-r last:border-r-0 border-slate-200/80 p-1 align-top"
                        >
                          <div
                            v-if="courses.length > 0"
                            class="h-full w-full flex min-h-0 flex-col"
                            :class="courses.length > 1 ? 'gap-1' : ''"
                            :style="{ minHeight: `${cellInnerHeight(rIndex, dayIndex)}px` }"
                          >
                            <div
                              v-for="(course, cIdx) in courses"
                              :key="course.code + '_' + cIdx"
                              class="relative flex min-h-0 min-w-0 flex-1 flex-col justify-between rounded-xl border text-left select-none shadow-2xs transition-all"
                              :class="[
                                cellSpans[rIndex][dayIndex] === 1 || courses.length > 1 ? 'p-1.5 gap-1' : 'p-2.5 gap-1.5',
                              ]"
                              :style="[courseCardStyle(course), { minHeight: `${cardMinHeight(rIndex, dayIndex, courses.length)}px` }]"
                            >
                              <!-- 顶部：不包裹短胶囊条（呼应参考图） -->
                              <div class="min-w-0">
                                <div
                                  v-if="!isCustomEvent(course)"
                                  class="mx-auto rounded-full opacity-90"
                                  :class="cellSpans[rIndex][dayIndex] === 1 || courses.length > 1 ? 'mb-1 h-[2.5px] w-6' : 'mb-1.5 h-[3.5px] w-8'"
                                  :style="{ backgroundColor: 'var(--card-accent)' }"
                                  aria-hidden="true"
                                />
                                <h4
                                  class="font-black leading-snug break-words tracking-tight"
                                  :class="cellSpans[rIndex][dayIndex] === 1 || courses.length > 1 ? 'text-[11.5px]' : 'text-[13px]'"
                                  :style="{ color: 'var(--card-title)' }"
                                >
                                  {{ course.courseName || course.code }}
                                </h4>
                                <div
                                  v-if="course.code && !isCustomEvent(course)"
                                  class="font-mono text-[9.5px] font-bold text-slate-600 tabular-nums leading-none mt-0.5"
                                >
                                  #{{ course.code }}
                                </div>
                              </div>

                              <!-- 中部：教室地点（特制字重、主视觉强化，方便看图瞬间锁定） -->
                              <div v-if="course.occupyRoom" class="min-w-0">
                                <div
                                  class="flex items-center gap-1 font-black break-words tracking-tight"
                                  :class="cellSpans[rIndex][dayIndex] === 1 || courses.length > 1 ? 'text-[11px]' : 'text-[12px]'"
                                  :style="{ color: 'var(--card-title)' }"
                                >
                                  <MapPin class="h-3.5 w-3.5 shrink-0 text-primary stroke-[2.2]" />
                                  <span>{{ course.occupyRoom }}</span>
                                </div>
                              </div>

                              <!-- 底部：教师与周次信息（特制字重与高对比字色） -->
                              <div
                                class="min-w-0 leading-tight"
                                :class="cellSpans[rIndex][dayIndex] === 1 || courses.length > 1 ? 'space-y-0.5 text-[10px]' : 'space-y-1 text-[11px]'"
                                :style="{ color: 'var(--card-sub)' }"
                              >
                                <div v-if="teacherName(course) && !isCustomEvent(course)" class="flex items-center gap-1 font-bold break-words text-slate-800">
                                  <User class="h-3 w-3 shrink-0 text-slate-700 stroke-[2]" />
                                  <span>{{ compactTeacherName(teacherName(course)) }}</span>
                                </div>
                                <div class="flex flex-wrap items-center gap-1">
                                  <span
                                    v-if="weekParityBadge(course.occupyWeek)"
                                    class="rounded px-1.5 py-0.2 text-[8.5px] font-black bg-primary/15 text-primary border border-primary/30 shrink-0 whitespace-nowrap"
                                  >
                                    {{ weekParityBadge(course.occupyWeek) }}
                                  </span>
                                  <span class="font-bold font-mono tabular-nums whitespace-nowrap text-slate-800">
                                    {{ formatDisplayWeeks(course.occupyWeek) }}
                                  </span>
                                </div>
                              </div>
                            </div>
                          </div>
                        </td>
                      </template>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        <!-- 课表下方独立悬浮交互操作坞 (Floating Action Dock) -->
        <nav
          ref="dockRef"
          aria-label="Export Actions"
          class="mt-4 sm:mt-5 flex shrink-0 items-center gap-2 rounded-full border border-white/20 bg-slate-900/85 px-3.5 py-2 text-white shadow-2xl backdrop-blur-xl transition-all cursor-default"
        >
          <!-- 关闭操作 -->
          <button
            type="button"
            class="flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-semibold text-slate-300 hover:bg-white/10 hover:text-white transition-all active:scale-[0.96]"
            @click="dialogOpen = false"
          >
            <X class="h-3.5 w-3.5" />
            <span>{{ t('common.close') }}</span>
          </button>

          <div class="hidden sm:block h-3.5 w-px bg-white/20" aria-hidden="true" />

          <!-- 复制图片（桌面/平板展示，移动端直接使用保存下载） -->
          <button
            type="button"
            class="hidden sm:flex items-center gap-1.5 rounded-full px-3.5 py-1.5 text-xs font-semibold text-slate-100 hover:bg-white/15 transition-all active:scale-[0.96]"
            :disabled="generating"
            @click="handleCopy"
          >
            <Check v-if="copied" class="h-3.5 w-3.5 text-emerald-400" />
            <Copy v-else class="h-3.5 w-3.5" />
            <span>{{ copied ? t('common.codeCopied') : t('schedule.exportCopy') }}</span>
          </button>

          <!-- 下载高清图片 -->
          <button
            type="button"
            class="flex items-center gap-1.5 rounded-full bg-primary px-4 py-1.5 text-xs font-bold text-white shadow-md hover:brightness-110 transition-all active:scale-[0.96]"
            :disabled="generating"
            @click="handleDownload"
          >
            <Loader2 v-if="generating" class="h-3.5 w-3.5 animate-spin" />
            <Download v-else class="h-3.5 w-3.5" />
            <span>{{ t('schedule.exportDownload') }}</span>
          </button>
        </nav>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
