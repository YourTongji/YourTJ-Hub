<script setup lang="ts">
// 课表网格（v2：周次视图 + 容忍式冲突标注 + 自定义占位 + 导出图片）。
// - calendarId>=120 新制为 11 行；同列节次区间聚类（相交/包含/部分重叠同格渲染），
//   单块 rowspan 只吞自己簇覆盖的行——部分重叠的冲突课必须同格可见。
// - 周次视图：weekView.week 为 null → 全部周次；指定周 → 按 occupyWeek 过滤
//   后显示（聚类按过滤后集合重算）。
// - 同格多课（单双周同位共存 / 容忍式冲突）竖向堆叠渲染，每块显示
//   课名+周次（单双周可辨）；周次无交集不判冲突（weeksOverlap 判据）。
// - 冲突标注：deriveConflicts 统一判据（同天+同节+周次交集），⚠ 角标。
// - 自定义占位（custom: 伪课号）渲染为灰块，不进课程详情；导出 PNG（html-to-image）。
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertCircle, AlertTriangle, BookOpen, CalendarCog, Clock, Download, Info, MapPin, User } from '@lucide/vue'
import {
  HoverCardArrow,
  HoverCardContent,
  HoverCardPortal,
  HoverCardRoot,
  HoverCardTrigger,
} from 'reka-ui'
import EmptyState from '@/site/components/EmptyState.vue'
import SiteSelect from '@/site/components/SiteSelect.vue'
import ScheduleConflictWarningDialog from '@/site/components/schedule/ScheduleConflictWarningDialog.vue'
import ScheduleExportDialog from '@/site/components/schedule/ScheduleExportDialog.vue'
import ScheduleExternalToolsTip from '@/site/components/schedule/ScheduleExternalToolsTip.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { courseColorSlotFor, courseContentVar, courseSlotVar } from '@/site/utils/courseColors'
import {
  clusterBySections,
  consolidateSameClassArrangements,
  currentWeekForDate,
  formatWeeksText,
  MAX_WEEK,
} from '@/site/utils/pkArrange'
import { conflictBaseOf, deriveConflicts, CUSTOM_EVENT_CODE_PREFIX, type PkConflictItem } from '@/site/utils/pkConflict'
import { sectionTimesFor } from '@/site/utils/sectionTimes'
import {
  cardMinHeightFor,
  cellInnerHeightFor,
  compactTeacherName as compactTeacherNames,
  computeRowHeights,
  dayPartLabelForRow,
  formatDisplayWeeks as formatDisplayWeeksWith,
  interactiveRowMetrics,
  teacherName as courseTeacherName,
  weekParityLabel,
} from '@/site/utils/timetableGrid'
import type { PkCourseOnTable } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

/** 周几 i18n key（与 locales schedule.weekdays.* 对齐）。 */
const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const
/** 「全部周次」下拉哨兵值（reka-ui SelectItem 不允许空字符串 value）。 */
const WEEK_ALL = 'all'

const cellCourses = ref<PkCourseOnTable[][][]>([])
const cellSpans = ref<number[][]>([])
const occupiedGrid = ref<boolean[][]>([])

const emit = defineEmits<{
  openDetail: [course: PkCourseOnTable]
  cellClick: [day: number, section: number]
  customize: []
  openPicker: []
}>()

// ---- 移动端长按检测 ----
const isMobile = ref(false)
let pressTimer: ReturnType<typeof setTimeout> | undefined
let pressStartX = 0
let pressStartY = 0

function onPressStart(course: PkCourseOnTable, event: Event) {
  if (!isMobile.value) return
  onPressCancel()
  const pointer = (event as TouchEvent).touches?.[0] ?? (event as MouseEvent)
  pressStartX = Number(pointer?.clientX ?? 0)
  pressStartY = Number(pointer?.clientY ?? 0)
  pressTimer = setTimeout(() => {
    pressTimer = undefined
    openCourseDetail(course)
  }, 420)
}

function onPressMove(event: Event) {
  if (!isMobile.value || !pressTimer) return
  const pointer = (event as TouchEvent).touches?.[0]
  if (!pointer) return
  const dx = Math.abs(Number(pointer.clientX) - pressStartX)
  const dy = Math.abs(Number(pointer.clientY) - pressStartY)
  if (dx > 10 || dy > 10) onPressCancel()
}

function onPressCancel() {
  if (pressTimer) clearTimeout(pressTimer)
  pressTimer = undefined
}

function setupMobileDetection() {
  const query = window.matchMedia('(max-width: 767px)')
  const apply = () => {
    isMobile.value = query.matches
  }
  apply()
  query.addEventListener('change', apply)
  return () => query.removeEventListener('change', apply)
}

// ---- 学期日期与周次定位 ----

/** 当前学期的起止日期（store.calendars 匹配 calendarId；无数据为 null）。 */
const activeCalendar = computed(
  () => store.state.calendars.find((cal) => cal.calendarId === store.state.majorSelected.calendarId) ?? null,
)

/** 由学期起始日期定位的当前周（学期外/无日期为 null，「当前周次」开关禁用）。 */
const currentWeek = computed(() => {
  const cal = activeCalendar.value
  if (!cal?.startDate) return null
  // endDate 之后（学期已结束）返回 null：开关禁用，不停留在最后一周。
  return currentWeekForDate(cal.startDate, new Date(), cal.endDate ?? undefined)
})

/** 「当前周次」开关可用性：需要学期起始日期且今天在学期内。 */
const canUseCurrentWeek = computed(() => currentWeek.value !== null)

/** 周次下拉选项：全部周次（哨兵 all，reka-ui SelectItem 禁空串 value）+ 第 1..16 周。 */
const weekOptions = computed(() => [
  { value: WEEK_ALL, label: t('schedule.weekAll') },
  ...Array.from({ length: MAX_WEEK }, (_, i) => ({
    value: String(i + 1),
    label: t('schedule.weekN', { n: i + 1 }),
  })),
])

const weekValue = computed({
  get: () => (store.state.weekView.week === null ? WEEK_ALL : String(store.state.weekView.week)),
  set: (value: string) => {
    store.setWeekView({ week: value === WEEK_ALL ? null : Number(value), useCurrent: false })
  },
})

/** 「当前周次」开关：勾选定位当前周，取消回到全部周次。 */
function toggleCurrentWeek(event: Event) {
  const checked = event.target instanceof HTMLInputElement && event.target.checked
  store.setWeekView({ week: checked ? currentWeek.value : null, useCurrent: checked })
}

/** 学期日期条文本（有任一端日期时展示）。 */
const semesterDateText = computed(() => {
  const cal = activeCalendar.value
  if (!cal?.startDate && !cal?.endDate) return ''
  return [cal.startDate, cal.endDate].filter(Boolean).join(' ~ ')
})

// ---- 冲突派生（与占用表同判据）----
const conflicts = computed(() => deriveConflicts(store.state.occupied))

/** 课程块是否冲突（conflictBaseOf 归一：custom 原样、真实课号取基础课号，兼容无点班号）。 */
function isConflicted(course: PkCourseOnTable): boolean {
  return (conflicts.value.get(conflictBaseOf(course.code))?.length ?? 0) > 0
}

// ---- 课程块渲染（v2：结构化字段分层，不再从 showText 反解）----

function isCustomEvent(course: PkCourseOnTable): boolean {
  return course.code.startsWith(CUSTOM_EVENT_CODE_PREFIX)
}

function courseCardStyle(course: PkCourseOnTable): Record<string, string> {
  if (isCustomEvent(course)) {
    return {
      '--card-accent': 'var(--gf-color-base-content)',
      '--card-bg': 'color-mix(in oklab, var(--gf-color-base-200) 80%, var(--gf-color-base-100))',
      '--card-bg-hover': 'var(--gf-color-base-200)',
      '--card-border': 'var(--gf-color-line)',
      '--card-title': 'var(--gf-color-base-content)',
      '--card-sub': 'color-mix(in oklab, var(--gf-color-base-content) 70%, transparent)',
      '--card-shadow-hover': '0 2px 8px -2px rgba(0, 0, 0, 0.08), 0 1px 3px -1px rgba(0, 0, 0, 0.04)',
      backgroundColor: 'var(--card-bg)',
      borderColor: 'var(--card-border)',
      color: 'var(--card-title)',
    }
  }
  const seed = course.code || course.courseName || course.showText || 'course'
  const slot = courseColorSlotFor(seed)
  const slotVar = courseSlotVar(slot)
  return {
    '--card-accent': `var(${slotVar})`,
    // 借鉴参考图的柔和莫兰迪/马卡龙粉彩色底（11% 槽位色轻盈融合）
    '--card-bg': `color-mix(in oklab, var(${slotVar}) 11%, var(--gf-color-base-100))`,
    '--card-bg-hover': `color-mix(in oklab, var(${slotVar}) 17%, var(--gf-color-base-100))`,
    // 极轻微的同色系半透细边框，呈现「不包裹」的自然悬浮感
    '--card-border': `color-mix(in oklab, var(${slotVar}) 18%, transparent)`,
    // 标题文字：以 base-content 为底混入 45% 槽位色，确保与浅色/深色底对比度均 ≥8:1
    '--card-title': `color-mix(in oklab, var(${slotVar}) 45%, var(--gf-color-base-content))`,
    // 次级文字（教师、周次）
    '--card-sub': `color-mix(in oklab, var(${slotVar}) 25%, var(--gf-color-base-content))`,
    '--card-badge-bg': `color-mix(in oklab, var(${slotVar}) 12%, transparent)`,
    // 自然柔和环境光沉降阴影
    '--card-shadow-hover': '0 3px 10px -2px rgba(0, 0, 0, 0.08), 0 1px 3px -1px rgba(0, 0, 0, 0.04)',
    backgroundColor: 'var(--card-bg)',
    borderColor: 'var(--card-border)',
    color: 'var(--card-title)',
  }
}

function teacherName(course: PkCourseOnTable): string {
  return courseTeacherName(course)
}

/** 课程卡片内紧凑展示教师名（最多展示 2 位，超量显示「首位 等」，防多位教师撑满空间）。 */
function compactTeacherName(raw: string): string {
  return compactTeacherNames(raw, 2)
}

/** 课程学分（从已加课程列表中匹配）。 */
function creditOfCourse(course: PkCourseOnTable): number | null {
  const base = conflictBaseOf(course.code)
  const staged = store.state.commonLists.stagedCourses.find((c) => c.courseCode === base)
  return staged?.credit ?? null
}

/** 该课程的时间冲突条目。 */
function courseConflictItems(course: PkCourseOnTable): PkConflictItem[] {
  return conflicts.value.get(conflictBaseOf(course.code)) ?? []
}

/** 节次区间文本（"第 3-4 节"）。 */
function sectionSpanText(sections: number[]): string {
  if (!sections || sections.length === 0) return ''
  const span = sections.length === 1 ? `${sections[0]}` : `${sections[0]}-${sections[sections.length - 1]}`
  return t('schedule.sectionsN', { range: span })
}

/** 周次格式精简（共享实现见 timetableGrid，如单周 1-15 → "1-15周(单周)"）。 */
function formatDisplayWeeks(weeks: readonly number[] | undefined): string {
  return formatDisplayWeeksWith(weeks, t)
}

/** 周次节次教室行（回退/辅助文本）。 */
function courseSubline(course: PkCourseOnTable): string {
  const parts: string[] = []
  const weeks = formatWeeksText(course.occupyWeek)
  if (weeks) parts.push(t('schedule.weeksN', { range: weeks }))
  const sections = course.occupyTime
  if (sections.length > 0) {
    const span = sections.length === 1 ? `${sections[0]}` : `${sections[0]}-${sections[sections.length - 1]}`
    parts.push(t('schedule.sectionsN', { range: span }))
  }
  if (course.occupyRoom) parts.push(course.occupyRoom)
  if (parts.length > 0) return parts.join(' ')
  return course.arrangementText || course.showText
}

/**
 * 动态计算每行的基准与扩展高度（共享实现见 timetableGrid）：
 * 单双周多门课纵向堆叠时该行自动增高，同行单门课均分撑满扩展后的行高。
 * 交互网格按移动/桌面切换行高度量。
 */
const computedRowHeights = computed<number[]>(() =>
  computeRowHeights(
    {
      cellCourses: cellCourses.value,
      cellSpans: cellSpans.value,
      occupiedGrid: occupiedGrid.value,
    },
    interactiveRowMetrics(isMobile.value),
  ),
)

function cellInnerHeight(rIndex: number, dayIndex: number): number {
  return cellInnerHeightFor(
    cellSpans.value?.[rIndex]?.[dayIndex] || 1,
    computedRowHeights.value,
    interactiveRowMetrics(isMobile.value),
    rIndex,
  )
}

function cardMinHeight(rIndex: number, dayIndex: number, courseCount: number): number {
  return cardMinHeightFor(
    cellSpans.value?.[rIndex]?.[dayIndex] || 1,
    computedRowHeights.value,
    courseCount,
    interactiveRowMetrics(isMobile.value),
    rIndex,
  )
}

// ---- 桌面端课程块浮动预览微卡片 ----
const hoveredCourse = ref<PkCourseOnTable | null>(null)
const hoverCardPos = ref<{ top: number; left: number } | null>(null)
let hoverTimer: ReturnType<typeof setTimeout> | undefined

function onCourseMouseEnter(course: PkCourseOnTable, event: MouseEvent) {
  if (isMobile.value) return
  if (hoverTimer) clearTimeout(hoverTimer)
  const target = event.currentTarget as HTMLElement | null
  if (!target) return
  const rect = target.getBoundingClientRect()
  // 计算卡片位置：默认显示在右侧，若超出视口则显示在左侧
  const cardWidth = 270
  let left = rect.right + 10
  if (typeof window !== 'undefined' && left + cardWidth > window.innerWidth - 16) {
    left = Math.max(16, rect.left - cardWidth - 10)
  }
  let top = rect.top
  if (typeof window !== 'undefined' && top + 280 > window.innerHeight - 16) {
    top = Math.max(16, window.innerHeight - 296)
  }
  hoverCardPos.value = { top, left }
  hoverTimer = setTimeout(() => {
    hoveredCourse.value = course
  }, 120)
}

function onCourseMouseLeave() {
  if (hoverTimer) clearTimeout(hoverTimer)
  hoverTimer = undefined
  hoveredCourse.value = null
  hoverCardPos.value = null
}

const hoverCardStyle = computed(() => {
  if (!hoverCardPos.value) return {}
  return {
    top: `${hoverCardPos.value.top}px`,
    left: `${hoverCardPos.value.left}px`,
  }
})

/** 课程无障碍描述（屏幕阅读器等辅助技术可用；杜绝原生 title 属性，避免触发浏览器黑色原生 tooltip 遮盖浮动预览面板）。 */
function courseAriaLabel(course: PkCourseOnTable): string {
  const parts: string[] = [course.courseName || course.code]
  if (course.code) parts.push(`(${course.code})`)
  if (course.occupyRoom) parts.push(`地点: ${course.occupyRoom}`)
  if (teacherName(course)) parts.push(`教师: ${teacherName(course)}`)
  const weeks = formatDisplayWeeks(course.occupyWeek)
  if (weeks) parts.push(`周次: ${weeks}`)
  const sections = sectionSpanText(course.occupyTime)
  if (sections) parts.push(`节次: ${sections}`)
  if (isConflicted(course)) parts.push(`[${t('schedule.conflictBadge')}]`)
  return parts.join(' ')
}

// ---- 网格渲染（对齐上游 updateTimeTable；单周模式按过滤后集合重算）----

/** 周次过滤后的课表行（week=null 全量；指定周按 occupyWeek 命中）。 */
function filteredCourses(): PkCourseOnTable[] {
  const week = store.state.weekView.week
  const all = store.state.timeTableData
  if (week === null) return all
  return all.filter((course) => (course.occupyWeek ?? []).includes(week))
}

function updateTimeTable() {
  const maxRows = store.readTimeTableRows()
  const spans = Array.from({ length: maxRows }, () => Array(7).fill(1) as number[])
  const covered = Array.from({ length: maxRows }, () => Array(7).fill(false) as boolean[])
  const coursesGrid = Array.from({ length: maxRows }, () =>
    Array.from({ length: 7 }, () => [] as PkCourseOnTable[]),
  )

  const safeCourses = filteredCourses().filter(
    (course) =>
      Array.isArray(course?.occupyTime) &&
      course.occupyTime.length > 0 &&
      typeof course?.occupyDay === 'number' &&
      course.occupyDay >= 1 &&
      course.occupyDay <= 7 &&
      course.occupyTime.every((slot) => slot >= 1 && slot <= maxRows),
  )

  const byDay: PkCourseOnTable[][] = Array.from({ length: 7 }, () => [])
  for (const course of safeCourses) byDay[course.occupyDay - 1].push(course)

  // 节次区间聚类：相交（含部分重叠/包含）的课程同格渲染，
  // 避免一块的 rowspan 吞掉部分重叠的另一块（容忍式冲突必须可见）。
  for (let day = 0; day < 7; day++) {
    for (const cluster of clusterBySections(byDay[day])) {
      const consolidatedItems = consolidateSameClassArrangements(cluster.items)
      const row = cluster.start - 1
      spans[row][day] = cluster.end - row
      coursesGrid[row][day] = consolidatedItems
      for (let r = row + 1; r < row + spans[row][day]; r++) {
        if (r < maxRows) covered[r][day] = true
      }
    }
  }

  cellSpans.value = spans
  cellCourses.value = coursesGrid
  occupiedGrid.value = covered
}

/** 单双周标识提取（多课/紧凑展示用；共享实现见 timetableGrid）。 */
function weekParityBadge(weeks: readonly number[] | undefined): string | null {
  return weekParityLabel(weeks, t)
}

/** 课表是否已有课程（决定渲染网格还是空态引导，issue #229）。 */
const hasCourses = computed(() => cellCourses.value.some((row) => row.some((cell) => cell.length > 0)))

const sectionTimes = computed(() => sectionTimesFor(store.readTimeTableRows(), store.state.sectionTimeOverrides))

/** 每行节次的起止时间（无数据返回空串）。 */
function sectionTimeText(index: number): string {
  const item = sectionTimes.value[index]
  return item ? `${item.start}-${item.end}` : ''
}

/** 该行是否为某时段分组首行（渲染分组标签；共享实现见 timetableGrid）。 */
function dayPartLabelAt(index: number): string | null {
  return dayPartLabelForRow(index + 1, sectionTimes.value, t)
}

/** 该格在当前周次视图下是否已被占用（单周模式按周过滤，空格可点选加课）。 */
function cellOccupiedForView(dayIndex: number, rowIndex: number): boolean {
  const items = store.state.occupied?.[rowIndex]?.[dayIndex] ?? []
  const week = store.state.weekView.week
  if (week === null) return items.length > 0
  return items.some((item) => (item.occupyWeek ?? []).includes(week))
}

function handleCellClick(dayIndex: number, rowIndex: number) {
  if (!store.isMajorSelected()) return
  if (cellOccupiedForView(dayIndex, rowIndex)) return
  emit('cellClick', dayIndex + 1, rowIndex + 1)
}

/** 课程块激活（点击/长按/回车）：custom 占位不进课程详情（避免伪课号触发课评请求）。 */
function openCourseDetail(course: PkCourseOnTable) {
  if (isCustomEvent(course)) return
  emit('openDetail', course)
}

// ---- 导出图片与排版元数据（html-to-image）----

const activePlan = computed(() =>
  store.state.plans.find((p) => p.id === store.state.activePlanId),
)

const activePlanName = computed(
  () => activePlan.value?.name || t('schedule.planDefaultName', { n: 1 }),
)

const majorDisplayName = computed(() => {
  const sel = store.state.majorSelected
  return sel.majorName || sel.major || ''
})

const weekViewLabel = computed(() => {
  if (store.state.weekView.week === null) return t('schedule.weekAll')
  return t('schedule.weekN', { n: store.state.weekView.week })
})

const exportSummaryText = computed(() => {
  const stats = store.stats()
  const parts: string[] = []
  if (majorDisplayName.value) parts.push(majorDisplayName.value)
  parts.push(t('schedule.courseCountAndCredits', { count: stats.courseCount, credits: stats.totalCredit }))
  if (semesterDateText.value) parts.push(semesterDateText.value)
  return parts.join(' · ')
})

const exportDateText = computed(() => {
  try {
    return new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
  } catch {
    return ''
  }
})

const showConflictWarning = ref(false)
const showExportDialog = ref(false)

function handleExportClick() {
  if (conflicts.value.size > 0) {
    showConflictWarning.value = true
  } else {
    showExportDialog.value = true
  }
}

function handleResolveConflicts() {
  showConflictWarning.value = false
}

function handleContinueExport() {
  showConflictWarning.value = false
  showExportDialog.value = true
}

watch(
  () => [store.state.timeTableData, store.state.weekView.week],
  () => updateTimeTable(),
  { deep: true, immediate: true },
)

let cleanupMobile: (() => void) | undefined
onMounted(() => {
  if (typeof window !== 'undefined') {
    cleanupMobile = setupMobileDetection()
  }
})
onBeforeUnmount(() => {
  cleanupMobile?.()
  onPressCancel()
})
</script>

<template>
  <div class="min-w-0">
    <!-- 工具条：周次控制 + 学期日期 + 误差提示 + 自定义 + 导出图片 -->
    <!-- 移动端（<md）结构化拆为两行平衡重心；桌面端（md+）通过 md:contents 扁平展开为原有单行，零视觉回归 -->
    <div class="mb-2 flex flex-col gap-1.5 md:flex-row md:items-center md:gap-2">
      <!-- 组 1：视图控制与辅助提示（移动端首行平衡展示，桌面端居左） -->
      <div class="flex items-center justify-between gap-1.5 overflow-x-auto no-scrollbar py-0.5 md:contents">
        <div class="flex items-center gap-1.5 shrink-0 md:shrink">
          <label
            class="flex items-center gap-1.5 text-[12px] text-base-content/70 select-none"
            :title="canUseCurrentWeek ? undefined : t('schedule.currentWeekDisabled')"
          >
            <input
              type="checkbox"
              class="checkbox checkbox-sm"
              :disabled="!canUseCurrentWeek"
              :checked="store.state.weekView.useCurrent"
              @change="toggleCurrentWeek"
            />
            {{ t('schedule.currentWeek') }}
          </label>
          <div class="w-24 sm:w-28 shrink-0">
            <SiteSelect
              v-model="weekValue"
              :options="weekOptions"
              :label="t('schedule.weekView')"
              :aria-label="t('schedule.weekView')"
            />
          </div>
          <span v-if="semesterDateText" class="hidden text-[11px] text-base-content/55 sm:inline">{{ semesterDateText }}</span>
        </div>

        <!-- 辅助提示胶囊组（专业课识别提示 + 外部工具气泡） -->
        <div class="flex items-center gap-1.5 shrink-0 md:contents">
          <!-- 专业课识别误差提示（精巧 Hover Tip） -->
          <HoverCardRoot :open-delay="100" :close-delay="150">
            <HoverCardTrigger as-child>
              <button
                type="button"
                class="group inline-flex items-center gap-1.5 rounded-full border border-warning/30 bg-warning/10 px-2.5 py-1 text-[11px] font-medium text-warning hover:bg-warning/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warning/50 transition-colors cursor-help active:scale-[0.96]"
                :aria-label="t('schedule.academicInfoNotice')"
              >
                <AlertCircle class="h-3.5 w-3.5 shrink-0 text-warning transition-transform duration-150 group-hover:scale-110" />
                <span class="text-[11px] font-semibold">{{ t('schedule.academicInfoBadge') }}</span>
              </button>
            </HoverCardTrigger>
            <HoverCardPortal>
              <HoverCardContent
                side="bottom"
                align="start"
                :side-offset="8"
                :collision-padding="16"
                class="z-[2200] w-[min(320px,calc(100vw-2rem))] rounded-2xl border border-line/80 bg-base-100 p-4 shadow-2xl backdrop-blur-xl outline-none text-xs text-base-content"
              >
                <div class="flex items-start gap-3">
                  <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-warning/15 text-warning shadow-xs">
                    <AlertCircle class="h-4.5 w-4.5" />
                  </div>
                  <div class="min-w-0 flex-1 space-y-1">
                    <h4 class="text-xs font-bold text-base-content leading-snug">
                      {{ t('schedule.academicInfoNotice') }}
                    </h4>
                    <p class="text-[11px] leading-relaxed text-base-content/75">
                      {{ t('schedule.academicInfoTooltip') }}
                    </p>
                  </div>
                </div>
                <HoverCardArrow class="fill-base-100 stroke-line/80" />
              </HoverCardContent>
            </HoverCardPortal>
          </HoverCardRoot>

          <!-- 外部选课神器气泡（小且醒目的 Popover 气泡） -->
          <ScheduleExternalToolsTip />
        </div>
      </div>

      <!-- 桌面端弹性推挤占位 -->
      <span class="hidden md:block md:flex-1"></span>

      <!-- 组 2：交互操作与指引（移动端次行左右平衡，桌面端按钮靠右） -->
      <div class="flex items-center justify-between gap-2 pt-0.5 md:pt-0 md:contents">
        <!-- 移动端长按手势引导（左侧，以精巧信息胶囊呼应右侧操作按钮，平衡视觉重心，完全呈现不打点截断） -->
        <div v-if="isMobile" class="min-w-0 flex-1 flex items-center md:hidden">
          <div
            class="inline-flex items-center gap-1.5 rounded-lg bg-base-200/80 border border-line/60 px-2 py-1 text-[10.5px] font-medium text-base-content/70 select-none shrink-0"
            :title="t('schedule.longPressHint')"
          >
            <Info class="h-3 w-3 shrink-0 text-primary/80" />
            <span class="whitespace-nowrap">{{ t('schedule.longPressHint') }}</span>
          </div>
        </div>

        <!-- 操作按钮组（移动端紧凑文案释放横向空间，桌面端完整文案） -->
        <div class="flex items-center gap-1.5 sm:gap-2 shrink-0 md:contents">
          <button
            type="button"
            class="gf-button gf-button-sm gf-button-outline transition-transform active:scale-[0.96] px-2.5 sm:px-3"
            @click="emit('customize')"
          >
            <CalendarCog class="h-3.5 w-3.5" />
            <span class="sm:hidden">{{ t('schedule.customizeShort') }}</span>
            <span class="hidden sm:inline">{{ t('schedule.customize') }}</span>
          </button>
          <button
            type="button"
            class="gf-button gf-button-sm gf-button-primary transition-transform active:scale-[0.96] px-2.5 sm:px-3"
            @click="handleExportClick"
          >
            <Download class="h-3.5 w-3.5" />
            <span class="sm:hidden">{{ t('schedule.export') }}</span>
            <span class="hidden sm:inline">{{ t('schedule.exportImage') }}</span>
          </button>
        </div>
      </div>
    </div>

    <div
      class="overflow-x-auto rounded-2xl border border-line/70 bg-base-100 shadow-sm"
    >
      <!-- 课表抬头横幅（包含学期、方案名、专业、周次模式与统计，在有课程时展现） -->
      <div v-if="hasCourses" class="border-b border-line/60 bg-base-100 px-4 py-3 sm:px-6">
        <div class="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <h2 class="text-base font-bold text-base-content sm:text-lg tracking-tight">
              {{ activeCalendar?.calendarName || t('schedule.timetable') }}
            </h2>
            <p v-if="exportSummaryText" class="mt-0.5 text-xs text-base-content/65">
              {{ exportSummaryText }}
            </p>
          </div>
          <div class="flex items-center gap-2 text-xs">
            <span class="rounded-lg bg-base-200/80 px-2.5 py-1 font-semibold text-base-content/80 border border-line/50">
              {{ activePlanName }}
            </span>
            <span class="rounded-lg bg-primary/10 px-2.5 py-1 font-semibold text-primary border border-primary/20">
              {{ weekViewLabel }}
            </span>
          </div>
        </div>
      </div>

      <EmptyState
        v-if="cellCourses.length > 0 && !hasCourses"
        class="border-b border-line/60 py-10"
        :icon="BookOpen"
        :title="t('schedule.timetableEmptyTitle')"
        :description="t('schedule.timetableEmptyHint')"
      >
        <div class="flex items-center justify-center gap-2">
          <button
            type="button"
            class="gf-button gf-button-sm gf-button-primary shadow-sm transition-transform active:scale-[0.96]"
            @click="emit('openPicker')"
          >
            <BookOpen class="h-3.5 w-3.5" />
            {{ t('schedule.emptyTimetableAction') }}
          </button>
          <button
            type="button"
            class="gf-button gf-button-sm gf-button-outline transition-transform active:scale-[0.96]"
            @click="emit('customize')"
          >
            <CalendarCog class="h-3.5 w-3.5" />
            {{ t('schedule.customize') }}
          </button>
        </div>
      </EmptyState>
      <EmptyState
        v-else-if="cellCourses.length === 0"
        class="py-10"
        :icon="BookOpen"
        :title="t('schedule.selectMajorFirst')"
      />
      <table
        v-if="hasCourses || cellCourses.length === 0"
        class="w-full border-collapse table-fixed"
        :class="isMobile ? 'min-w-[530px]' : 'min-w-[640px]'"
      >
        <thead>
          <tr class="bg-base-200/60 h-9 md:h-10">
            <th class="w-[50px] border border-line/70 p-1 text-[11px] font-semibold text-base-content/70 sm:w-[60px] md:w-[86px] md:p-2 md:text-xs">
              {{ t('schedule.arrangement') }}
            </th>
            <th
              v-for="day in WEEKDAY_KEYS"
              :key="day"
              class="border border-line/70 p-1 text-[11px] font-semibold text-base-content/70 md:p-2 md:text-xs"
            >
              {{ t(`schedule.weekdays.${day}`) }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(row, index) in cellCourses"
            :key="index"
            class="border-b border-line/70"
            :style="{ height: `${computedRowHeights[index]}px` }"
            :class="[index === cellCourses.length - 1 ? 'bg-base-200/50' : index % 2 === 0 ? 'bg-base-100' : 'bg-base-200/30']"
          >
            <td
              class="h-px border border-line/70 p-0.5 text-center text-[11px] font-semibold text-base-content/70 overflow-hidden md:p-2 md:text-xs"
            >
              <span v-if="dayPartLabelAt(index)" class="mb-0.5 block text-[10px] font-bold text-primary/80">
                {{ dayPartLabelAt(index) }}
              </span>
              {{ t('schedule.sectionLabel', { section: index + 1 }) }}
              <span v-if="sectionTimeText(index)" class="hidden md:block whitespace-nowrap text-[9px] font-normal text-base-content/45 tabular-nums">
                {{ sectionTimeText(index) }}
              </span>
              <div v-if="sectionTimes[index]" class="md:hidden mt-0.5 text-[8.5px] font-mono leading-[1.1] text-base-content/50 tabular-nums">
                <span class="block">{{ sectionTimes[index]?.start }}</span>
                <span class="block text-[8px] opacity-75">{{ sectionTimes[index]?.end }}</span>
              </div>
            </td>
            <template v-for="(courses, dayIndex) in row" :key="dayIndex">
              <!-- rowspan 已占用的列槽不渲染 td，否则整行多出一列导致错位 -->
              <td
                v-if="!occupiedGrid[index][dayIndex]"
                class="h-px border border-line/70 p-[2px] align-top text-left md:p-1"
                :rowspan="cellSpans[index][dayIndex]"
                :tabindex="courses.length > 0 ? undefined : 0"
                :role="courses.length > 0 ? undefined : 'button'"
                :aria-label="courses.length > 0 ? undefined : t('schedule.emptyCell')"
                @click="handleCellClick(dayIndex, index)"
                @keydown.enter.prevent="courses.length === 0 && handleCellClick(dayIndex, index)"
                @keydown.space.prevent="courses.length === 0 && handleCellClick(dayIndex, index)"
              >
                <!-- 课程卡片网格容器：h-full w-full flex flex-col -->
                <div
                  v-if="courses.length > 0"
                  class="h-full w-full flex min-h-0 flex-col"
                  :class="courses.length > 1 ? 'gap-1' : ''"
                  :style="{ minHeight: `${cellInnerHeight(index, dayIndex)}px` }"
                >
                  <div
                    v-for="(course, courseIndex) in courses"
                    :key="course.code + '_' + courseIndex"
                    class="schedule-course-card relative flex min-h-0 min-w-0 flex-1 flex-col justify-between overflow-hidden rounded-xl border select-none cursor-pointer text-left transition-all outline-none focus-visible:ring-2 focus-visible:ring-primary/60 focus-visible:ring-offset-1"
                    :class="[
                      courses.length > 1 || cellSpans[index][dayIndex] === 1
                        ? 'p-1 md:p-1.5'
                        : 'p-1 sm:p-1.5 md:p-2',
                    ]"
                    :style="[courseCardStyle(course), { minHeight: `${cardMinHeight(index, dayIndex, courses.length)}px` }]"
                    tabindex="0"
                    role="button"
                    :aria-label="courseAriaLabel(course)"
                    @click.stop="openCourseDetail(course)"
                    @keydown.enter.stop.prevent="openCourseDetail(course)"
                    @keydown.space.stop.prevent="openCourseDetail(course)"
                    @touchstart.stop="onPressStart(course, $event)"
                    @touchmove.stop="onPressMove($event)"
                    @touchend.stop="onPressCancel()"
                    @touchcancel.stop="onPressCancel()"
                    @mousedown.stop="onPressStart(course, $event)"
                    @mouseup.stop="onPressCancel()"
                    @mouseleave.stop="onPressCancel()"
                    @mouseenter="onCourseMouseEnter(course, $event)"
                    @mouseleave="onCourseMouseLeave"
                  >
                    <!-- 冲突角标：右上角轻盈半透警告徽标 -->
                    <span
                      v-if="isConflicted(course) && !isCustomEvent(course)"
                      class="absolute right-1 top-1 z-10 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-error/15 text-error border border-error/30 text-[9px] shadow-2xs transition-transform hover:scale-110"
                      :aria-label="t('schedule.conflictBadge')"
                    >
                      <AlertTriangle class="h-2 w-2" />
                    </span>

                    <!-- 紧凑/同格多课模式（span=1 或 courses.length > 1） -->
                    <template v-if="courses.length > 1 || cellSpans[index][dayIndex] === 1">
                      <div class="min-w-0 h-full flex-1 flex flex-col justify-between gap-1">
                        <div class="min-w-0">
                          <!-- 顶部不包裹短条（借鉴参考图） -->
                          <div
                            v-if="!isCustomEvent(course)"
                            class="mx-auto mb-1 h-[2.5px] w-5 rounded-full opacity-65 transition-opacity group-hover:opacity-90"
                            :style="{ backgroundColor: 'var(--card-accent)' }"
                            aria-hidden="true"
                          />
                          <div class="min-w-0">
                            <span
                              class="block truncate font-semibold text-[10.5px] sm:text-[11px] leading-tight text-[var(--card-title)]"
                            >
                              {{ course.courseName || course.code }}
                            </span>
                            <span
                              v-if="course.code && !isCustomEvent(course)"
                              class="hidden md:block font-mono text-[9px] opacity-60 tabular-nums truncate"
                            >
                              #{{ course.code }}
                            </span>
                          </div>
                        </div>

                        <div class="flex items-center justify-between gap-1 text-[9.5px] sm:text-[10px] min-w-0 leading-none">
                          <span
                            v-if="course.occupyRoom"
                            class="inline-flex items-center gap-0.5 sm:gap-1 min-w-0 font-medium text-[var(--card-title)] opacity-90"
                          >
                            <MapPin class="hidden md:inline-block h-2.5 w-2.5 shrink-0 opacity-60" />
                            <span class="break-all md:truncate">{{ course.occupyRoom }}</span>
                          </span>
                          <span
                            v-if="weekParityBadge(course.occupyWeek)"
                            class="rounded px-0.5 sm:px-1 py-0.2 text-[8px] sm:text-[8.5px] font-semibold bg-primary/10 text-primary border border-primary/20 shrink-0"
                          >
                            {{ weekParityBadge(course.occupyWeek) }}
                          </span>
                          <span
                            v-else-if="formatWeeksText(course.occupyWeek)"
                            class="text-[8.5px] sm:text-[9px] opacity-70 tabular-nums truncate font-mono"
                          >
                            {{ formatWeeksText(course.occupyWeek) }}
                          </span>
                        </div>
                      </div>
                    </template>

                    <!-- 标准舒展模式（span >= 2 且单门课） -->
                    <template v-else>
                      <div class="flex h-full min-h-0 w-full flex-col justify-between gap-1 md:gap-1.5">
                        <!-- 顶部：不包裹短条 + 课名 + 课号 -->
                        <div class="min-w-0">
                          <!-- 顶部居中短条：不包裹、自然悬浮、呼应课程色彩 -->
                          <div
                            v-if="!isCustomEvent(course)"
                            class="mx-auto mb-1 md:mb-1.5 h-[3px] w-7 rounded-full opacity-70 transition-opacity group-hover:opacity-95"
                            :style="{ backgroundColor: 'var(--card-accent)' }"
                            aria-hidden="true"
                          />
                          <h3
                            class="block font-semibold tracking-tight text-[11px] sm:text-xs md:text-[12.5px] leading-tight md:leading-snug line-clamp-2 break-all text-[var(--card-title)]"
                          >
                            {{ course.courseName || course.code }}
                          </h3>
                          <span
                            v-if="course.code && !isCustomEvent(course)"
                            class="hidden md:block mt-0.5 font-mono text-[9px] opacity-60 tabular-nums truncate"
                          >
                            #{{ course.code }}
                          </span>
                        </div>

                        <!-- 中部：教室（纯净教室名，移动端免除 MapPin 挤占空间，保证完整可读） -->
                        <div v-if="course.occupyRoom" class="my-auto py-0.5 min-w-0">
                          <div
                            class="flex items-center gap-1 md:gap-1.5 min-w-0 text-[10.5px] md:text-[11px] font-medium text-[var(--card-title)]"
                          >
                            <MapPin class="hidden md:inline-block h-3 w-3 shrink-0 opacity-65 text-primary" />
                            <span class="break-all md:truncate leading-tight">{{ course.occupyRoom }}</span>
                          </div>
                        </div>

                        <!-- 底部：教师与周次（清爽层级排版，移动端免除 User 图标以完整呈现教师姓名） -->
                        <div class="min-w-0 space-y-0.5 text-[9.5px] sm:text-[10px] md:text-[10.5px] leading-tight text-[var(--card-sub)]">
                          <!-- 教师：精炼为首位+等，防多位教师炸裂撑满空间 -->
                          <div
                            v-if="teacherName(course) && !isCustomEvent(course)"
                            class="flex items-center gap-1 md:gap-1.5 font-medium opacity-85"
                          >
                            <User class="hidden md:inline-block h-2.5 w-2.5 shrink-0 opacity-55" />
                            <span class="break-all md:truncate">{{ compactTeacherName(teacherName(course)) }}</span>
                          </div>

                          <!-- 周次：解析为 1-15周(单) 等优雅文本 -->
                          <div class="flex items-center gap-1 md:gap-1.5 text-[9px] sm:text-[9.5px] md:text-[10px] opacity-80">
                            <span
                              v-if="weekParityBadge(course.occupyWeek)"
                              class="rounded px-0.5 md:px-1 py-0.2 text-[8px] md:text-[8.5px] font-semibold bg-primary/10 text-primary border border-primary/20 shrink-0"
                            >
                              {{ weekParityBadge(course.occupyWeek) }}
                            </span>
                            <span class="truncate font-mono tabular-nums">
                              {{ formatDisplayWeeks(course.occupyWeek) }}
                            </span>
                          </div>
                        </div>
                      </div>
                    </template>
                  </div>
                </div>
              </td>
            </template>
          </tr>
        </tbody>
      </table>

      <!-- 课表导出底部水印 -->
      <div v-if="hasCourses" class="border-t border-line/50 bg-base-200/20 px-4 py-2 flex items-center justify-between text-[11px] text-base-content/45 sm:px-6">
        <span>YourTJ Hub · {{ t('schedule.exportWatermark') }}</span>
        <span v-if="exportDateText">{{ exportDateText }}</span>
      </div>
    </div>

    <!-- 桌面端即时浮动微卡片（无缝全局浮层） -->
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="hoveredCourse && !isMobile && hoverCardPos"
        class="pointer-events-none fixed z-[2200] w-[270px] rounded-2xl border border-line/80 bg-base-100/98 p-3.5 shadow-2xl backdrop-blur-xl outline-none text-xs text-base-content"
        :style="hoverCardStyle"
      >
        <div class="space-y-2.5">
          <!-- 头部：完整课名、代码与学分 -->
          <div class="flex items-start justify-between gap-2 border-b border-line/50 pb-2">
            <div class="min-w-0">
              <h4 class="font-bold text-sm text-base-content leading-snug">
                {{ hoveredCourse.courseName || hoveredCourse.code }}
              </h4>
              <div class="mt-0.5 flex items-center gap-1.5 text-[11px] text-base-content/60 font-mono">
                <span>{{ hoveredCourse.code }}</span>
                <span v-if="creditOfCourse(hoveredCourse)">· {{ t('schedule.credit', { credit: creditOfCourse(hoveredCourse) }) }}</span>
              </div>
            </div>
            <span
              v-if="isConflicted(hoveredCourse)"
              class="gf-badge gf-badge-error shrink-0"
            >
              {{ t('schedule.conflictBadge') }}
            </span>
          </div>

          <!-- 核心参数列表 -->
          <div class="space-y-1.5 text-[11px] text-base-content/85">
            <div v-if="hoveredCourse.occupyRoom" class="flex items-center gap-2">
              <MapPin class="h-3.5 w-3.5 shrink-0 text-primary" />
              <span class="font-medium text-base-content">{{ hoveredCourse.occupyRoom }}</span>
            </div>

            <div v-if="teacherName(hoveredCourse)" class="flex items-start gap-2">
              <User class="h-3.5 w-3.5 shrink-0 text-primary mt-0.5" />
              <span class="leading-relaxed">{{ teacherName(hoveredCourse) }}</span>
            </div>

            <div class="flex items-start gap-2">
              <Clock class="h-3.5 w-3.5 shrink-0 text-primary mt-0.5" />
              <div class="leading-relaxed">
                <div>{{ formatDisplayWeeks(hoveredCourse.occupyWeek) }}</div>
                <div v-if="sectionSpanText(hoveredCourse.occupyTime)" class="text-[10px] text-base-content/60">
                  {{ sectionSpanText(hoveredCourse.occupyTime) }}
                </div>
              </div>
            </div>
          </div>

          <!-- 冲突提示区 -->
          <div
            v-if="isConflicted(hoveredCourse) && courseConflictItems(hoveredCourse).length > 0"
            class="rounded-xl border border-error/30 bg-error/10 p-2 text-[10.5px] text-error"
          >
            <div class="font-bold mb-0.5 flex items-center gap-1">
              <AlertTriangle class="h-3 w-3" />
              <span>{{ t('schedule.cardConflictHint') }}</span>
            </div>
            <div class="text-[10px] leading-tight opacity-90">
              {{ courseConflictItems(hoveredCourse).map((c) => c.courseName).join('、') }}
            </div>
          </div>

          <!-- 交互引导 -->
          <div class="pt-1 text-[10px] text-base-content/50 flex items-center justify-between border-t border-line/40">
            <span>{{ t('schedule.cardClickHint') }}</span>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 课表冲突提示弹窗 -->
    <ScheduleConflictWarningDialog
      :open="showConflictWarning"
      :conflicts="conflicts"
      @close="showConflictWarning = false"
      @resolve="handleResolveConflicts"
      @continue-export="handleContinueExport"
    />

    <!-- 独立高清课表导出生成器弹窗 -->
    <ScheduleExportDialog
      :open="showExportDialog"
      :cell-courses="cellCourses"
      :cell-spans="cellSpans"
      :occupied-grid="occupiedGrid"
      :active-calendar="activeCalendar"
      :active-plan-name="activePlanName"
      :week-view-label="weekViewLabel"
      :export-summary-text="exportSummaryText"
      @close="showExportDialog = false"
    />
  </div>
</template>

<style scoped>
.schedule-course-card {
  transition-property: transform, box-shadow, background-color, border-color;
  transition-duration: 160ms;
  transition-timing-function: cubic-bezier(0.16, 1, 0.3, 1);
}

@media (hover: hover) {
  .schedule-course-card:hover {
    transform: translateY(-1px);
    box-shadow: var(--card-shadow-hover, 0 3px 10px -2px rgba(0, 0, 0, 0.08));
    background-color: var(--card-bg-hover, var(--card-bg)) !important;
  }
}

.schedule-course-card:active {
  transform: scale(0.985);
}
</style>
