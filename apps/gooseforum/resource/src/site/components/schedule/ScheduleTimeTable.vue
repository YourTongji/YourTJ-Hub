<script setup lang="ts">
// 课表网格（v2：周次视图 + 容忍式冲突标注 + 自定义占位 + 导出图片）。
// - calendarId>=120 新制为 11 行；同列节次区间聚类（相交/包含/部分重叠同格渲染），
//   单块 rowspan 只吞自己簇覆盖的行——部分重叠的冲突课必须同格可见。
// - 周次视图：weekView.week 为 null → 全部周次（同格多课并排细条）；
//   指定周 → 按 occupyWeek 过滤后整宽显示（聚类按过滤后集合重算）。
// - 冲突标注：deriveConflicts 统一判据（同天+同节+周次交集），⚠ 角标。
// - 自定义占位（custom: 伪课号）渲染为灰块，不进课程详情；导出 PNG（html-to-image）。
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toPng } from 'html-to-image'
import { AlertTriangle, BookOpen, CalendarCog, Download } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import SiteSelect from '@/site/components/SiteSelect.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { queueFlashMessage } from '@/runtime/flash-message'
import { courseColorSlotFor, courseContentVar, courseSlotVar } from '@/site/utils/courseColors'
import { clusterBySections, currentWeekForDate, formatWeeksText, MAX_WEEK } from '@/site/utils/pkArrange'
import { conflictBaseOf, deriveConflicts, CUSTOM_EVENT_CODE_PREFIX } from '@/site/utils/pkConflict'
import { dayPartBoundaries, sectionTimesFor, type DayPart } from '@/site/utils/sectionTimes'
import type { PkCourseOnTable } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

/** 周几 i18n key（与 locales schedule.weekdays.* 对齐）。 */
const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const

const cellCourses = ref<PkCourseOnTable[][][]>([])
const cellSpans = ref<number[][]>([])
const occupiedGrid = ref<boolean[][]>([])

const emit = defineEmits<{
  openDetail: [course: PkCourseOnTable]
  cellClick: [day: number, section: number]
  customize: []
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

/** 周次下拉选项：全部周次 + 第 1..16 周。 */
const weekOptions = computed(() => [
  { value: '', label: t('schedule.weekAll') },
  ...Array.from({ length: MAX_WEEK }, (_, i) => ({
    value: String(i + 1),
    label: t('schedule.weekN', { n: i + 1 }),
  })),
])

const weekValue = computed({
  get: () => (store.state.weekView.week === null ? '' : String(store.state.weekView.week)),
  set: (value: string) => {
    store.setWeekView({ week: value ? Number(value) : null, useCurrent: false })
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
  if (isCustomEvent(course)) return {}
  const seed = course.code || course.courseName || course.showText || 'course'
  const slot = courseColorSlotFor(seed)
  const bgVar = courseSlotVar(slot)
  const contentVar = courseContentVar(slot)
  return {
    // 先给单值底做回退（不支持 color-mix 的浏览器），再覆盖渐变
    background: `var(${bgVar})`,
    backgroundImage: `linear-gradient(135deg, var(${bgVar}), color-mix(in oklab, var(${bgVar}) 80%, black))`,
    color: `var(${contentVar})`,
    borderColor: `var(${contentVar})`,
  }
}

/** 左侧色条颜色（内容色；custom 事件无色条）。 */
function accentColor(course: PkCourseOnTable): string {
  if (isCustomEvent(course)) return 'transparent'
  return `var(${courseContentVar(courseColorSlotFor(course.code || course.courseName || 'course'))})`
}

function compactName(name: string): string {
  const cleaned = String(name || '')
    .replace(/[（(][^()（）]*[）)]/g, '')
    .replace(/\s+/g, '')
    .trim()
  if (!cleaned) return t('schedule.courseFallback')
  const chars = Array.from(cleaned)
  return chars.length > 7 ? `${chars.slice(0, 6).join('')}…` : cleaned
}

/** 教师名（teacherAndCode "张三(T001)" → "张三"）。 */
function teacherName(course: PkCourseOnTable): string {
  return String(course.teacherAndCode || '').replace(/\([^)]*\)$/g, '').trim()
}

/** 周次节次教室行（结构化字段；缺字段时回退 arrangementText/showText）。 */
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
      const row = cluster.start - 1
      spans[row][day] = cluster.end - row
      coursesGrid[row][day] = cluster.items
      for (let r = row + 1; r < row + spans[row][day]; r++) {
        if (r < maxRows) covered[r][day] = true
      }
    }
  }

  cellSpans.value = spans
  cellCourses.value = coursesGrid
  occupiedGrid.value = covered
}

/** 课表是否已有课程（决定渲染网格还是空态引导，issue #229）。 */
const hasCourses = computed(() => cellCourses.value.some((row) => row.some((cell) => cell.length > 0)))

// ---- 节次时间与分组 ----

const sectionTimes = computed(() => sectionTimesFor(store.readTimeTableRows()))

/** 每行节次的起止时间（无数据返回空串）。 */
function sectionTimeText(index: number): string {
  const item = sectionTimes.value[index]
  return item ? `${item.start}-${item.end}` : ''
}

/** 上午/下午/晚上分组边界（每段首个节次，1-based；0 表示该行不分组）。 */
const dayPartStarts = computed(() => {
  const boundaries = dayPartBoundaries(sectionTimes.value)
  return boundaries
})

/** 该行是否为某时段分组首行（渲染分组标签）。 */
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

// ---- 导出图片（html-to-image）----

const exportTarget = ref<HTMLElement | null>(null)
const exporting = ref(false)

async function exportPng() {
  const node = exportTarget.value
  if (!node || exporting.value) return
  exporting.value = true
  try {
    const dataUrl = await toPng(node, { pixelRatio: 2, backgroundColor: getComputedStyle(node).backgroundColor || '#ffffff' })
    const link = document.createElement('a')
    link.download = 'yourtj-schedule.png'
    link.href = dataUrl
    link.click()
  } catch {
    queueFlashMessage(t('schedule.exportImageFailed'), 'error')
  } finally {
    exporting.value = false
  }
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
    <!-- 工具条：周次控制 + 学期日期 + 自定义 + 导出图片 -->
    <div class="mb-2 flex flex-wrap items-center gap-2">
      <label
        class="flex items-center gap-1.5 text-[12px] text-base-content/70"
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
      <div class="w-28">
        <SiteSelect
          v-model="weekValue"
          :options="weekOptions"
          :label="t('schedule.weekView')"
          :aria-label="t('schedule.weekView')"
        />
      </div>
      <span v-if="semesterDateText" class="hidden text-[11px] text-base-content/55 sm:inline">{{ semesterDateText }}</span>
      <span class="flex-1"></span>
      <button type="button" class="gf-button gf-button-sm gf-button-outline" @click="emit('customize')">
        <CalendarCog class="h-3.5 w-3.5" />
        {{ t('schedule.customize') }}
      </button>
      <button
        type="button"
        class="gf-button gf-button-sm gf-button-primary"
        :disabled="exporting"
        @click="exportPng"
      >
        <Download class="h-3.5 w-3.5" />
        {{ t('schedule.exportImage') }}
      </button>
    </div>

    <div v-if="isMobile" class="px-1 pb-2 text-[11px] text-base-content/55">
      {{ t('schedule.longPressHint') }}
    </div>

    <div
      ref="exportTarget"
      class="overflow-x-auto rounded-2xl border border-line/70 bg-base-100 shadow-sm"
    >
      <EmptyState
        v-if="cellCourses.length > 0 && !hasCourses"
        class="border-b border-line/60"
        :icon="BookOpen"
        :title="t('schedule.timetableEmptyTitle')"
        :description="t('schedule.timetableEmptyHint')"
      />
      <EmptyState v-else-if="cellCourses.length === 0" :icon="BookOpen" :title="t('schedule.selectMajorFirst')" />
      <table
        v-if="hasCourses || cellCourses.length === 0"
        class="w-full border-collapse table-fixed"
        :class="isMobile ? 'min-w-[400px]' : ''"
      >
        <thead>
          <tr class="bg-base-200/60">
            <th class="w-[42px] border border-line/70 p-1 text-[11px] font-semibold text-base-content/70 md:w-[86px] md:p-2 md:text-xs">
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
            :class="[index === cellCourses.length - 1 ? 'bg-base-200/50' : index % 2 === 0 ? 'bg-base-100' : 'bg-base-200/30']"
          >
            <td
              class="border border-line/70 p-1 text-center text-[11px] font-semibold text-base-content/70 md:p-2 md:text-xs"
            >
              <span v-if="dayPartLabelAt(index)" class="mb-0.5 block text-[10px] font-bold text-primary/80">
                {{ dayPartLabelAt(index) }}
              </span>
              {{ t('schedule.sectionLabel', { section: index + 1 }) }}
              <span v-if="sectionTimeText(index)" class="block whitespace-nowrap text-[9px] font-normal text-base-content/45">
                {{ sectionTimeText(index) }}
              </span>
            </td>
            <template v-for="(courses, dayIndex) in row" :key="dayIndex">
              <!-- rowspan 已占用的列槽不渲染 td，否则整行多出一列导致错位 -->
              <td
                v-if="!occupiedGrid[index][dayIndex]"
                class="border border-line/70 p-[2px] align-top text-center md:p-1"
                :rowspan="cellSpans[index][dayIndex]"
                :tabindex="courses.length > 0 ? undefined : 0"
                :role="courses.length > 0 ? undefined : 'button'"
                :aria-label="courses.length > 0 ? undefined : t('schedule.emptyCell')"
                @click="handleCellClick(dayIndex, index)"
                @keydown.enter.prevent="courses.length === 0 && handleCellClick(dayIndex, index)"
                @keydown.space.prevent="courses.length === 0 && handleCellClick(dayIndex, index)"
              >
                <div
                  v-if="courses.length > 0"
                  class="flex h-full overflow-hidden rounded-xl"
                  :class="store.state.weekView.week === null && courses.length > 1 ? 'flex-row gap-[2px]' : 'flex-col'"
                  :style="{ height: cellSpans[index][dayIndex] * (isMobile ? 44 : 54) + 'px' }"
                >
                  <div
                    v-for="(course, courseIndex) in courses"
                    :key="course.code + '_' + courseIndex"
                    class="relative flex min-h-0 min-w-0 flex-1 flex-col justify-center overflow-y-auto overscroll-contain px-1 py-1 text-[11px] leading-tight md:px-2 md:py-2 md:text-xs"
                    :class="[
                      isMobile ? 'text-center' : 'text-left',
                      store.state.weekView.week === null && courses.length > 1 ? '' : courseIndex !== courses.length - 1 ? 'border-b border-dashed' : '',
                      isCustomEvent(course) ? 'border-l-[3px]' : 'border-l-[3px]',
                    ]"
                    :style="{ ...courseCardStyle(course), borderLeftColor: accentColor(course) }"
                    @touchstart.stop="onPressStart(course, $event)"
                    @touchmove.stop="onPressMove($event)"
                    @touchend.stop="onPressCancel()"
                    @touchcancel.stop="onPressCancel()"
                    @mousedown.stop="onPressStart(course, $event)"
                    @mouseup.stop="onPressCancel()"
                    @mouseleave.stop="onPressCancel()"
                    tabindex="0"
                    role="button"
                    :aria-label="course.courseName || course.code"
                    @click.stop="openCourseDetail(course)"
                    @keydown.enter.stop.prevent="openCourseDetail(course)"
                    @keydown.space.stop.prevent="openCourseDetail(course)"
                  >
                    <span
                      v-if="isConflicted(course) && !isCustomEvent(course)"
                      class="absolute right-0.5 top-0.5 z-10 flex items-center justify-center rounded-full bg-black/25 text-[10px] leading-none"
                      :title="t('schedule.conflictBadge')"
                      :aria-label="t('schedule.conflictBadge')"
                    >
                      <AlertTriangle class="h-3 w-3" />
                    </span>
                    <div class="my-auto w-full min-w-0">
                      <template v-if="isMobile">
                        <span class="max-w-full truncate text-[11px] font-extrabold leading-tight">{{ compactName(course.courseName) }}</span>
                        <span v-if="courseSubline(course)" class="mt-0.5 max-w-full truncate text-[10px] opacity-85">{{ courseSubline(course) }}</span>
                      </template>
                      <template v-else>
                        <!-- 全部周次模式下同格多课并排细条：只显示课名+周次 -->
                        <template v-if="store.state.weekView.week === null && courses.length > 1">
                          <span class="block truncate text-[11px] font-extrabold leading-tight">{{ compactName(course.courseName) }}</span>
                          <span v-if="formatWeeksText(course.occupyWeek)" class="block truncate text-[10px] opacity-85">
                            {{ t('schedule.weeksN', { range: formatWeeksText(course.occupyWeek) }) }}
                          </span>
                        </template>
                        <template v-else>
                          <!-- 分层课程块：课名 / 班号 / 教师 / 周次节次教室 -->
                          <span class="block break-words font-extrabold tracking-tight">{{ course.courseName || course.code }}</span>
                          <span v-if="course.code && !isCustomEvent(course)" class="mt-0.5 block break-words text-[10px] opacity-85">{{ course.code }}</span>
                          <span v-if="teacherName(course) && !isCustomEvent(course)" class="mt-0.5 block break-words text-[10px] opacity-85">{{ teacherName(course) }}</span>
                          <span v-if="courseSubline(course)" class="mt-1 block break-words opacity-95">{{ courseSubline(course) }}</span>
                        </template>
                      </template>
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
</template>
