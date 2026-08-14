<script setup lang="ts">
// 12×7 课表网格（calendarId>=120 新制为 11 行）。长课合并单元格，短课叠进长课格。
// 对齐上游 TimeTable.vue 的渲染算法与交互：点击空格查时段课程、长按课程块看详情。
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import type { PkCourseOnTable } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

/** 周几 i18n key（与 locales schedule.weekdays.* 对齐）。 */
const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const

const timeTable = ref<PkCourseOnTable[][][]>([])
const maxSpans = ref<number[][]>([])
const occupiedGrid = ref<boolean[][]>([])

interface CourseLineInfo {
  title: string
  mobileTitle: string
  mobileMeta: string
  sub: string
  meta: string
}

const emit = defineEmits<{
  openDetail: [course: PkCourseOnTable]
  cellClick: [day: number, section: number]
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
    emit('openDetail', course)
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

// ---- 课程块展示文本解析 ----
function hashColor(input: string): number {
  let h = 0
  for (let i = 0; i < input.length; i++) h = (h * 31 + input.charCodeAt(i)) >>> 0
  return h
}

function courseCardStyle(course: PkCourseOnTable): Record<string, string> {
  const seed = hashColor(course.code || course.courseName || course.showText || 'course')
  const hue = seed % 360
  if (isMobile.value) {
    return { background: `linear-gradient(135deg, hsl(${hue}, 82%, 52%), hsl(${(hue + 24) % 360}, 82%, 42%))` }
  }
  return { background: 'linear-gradient(135deg, #5d57e8, #4b3fd9)' }
}

function compactName(name: string): string {
  const cleaned = String(name || '')
    .replace(/[（(][^()（）]*[）)]/g, '')
    .replace(/\s+/g, '')
    .trim()
  if (!cleaned) return '课程'
  const chars = Array.from(cleaned)
  return chars.length > 7 ? `${chars.slice(0, 6).join('')}…` : cleaned
}

function compactMeta(teacher: string, room: string): string {
  const teacherText = String(teacher || '').replace(/[A-Z0-9]+$/i, '').trim()
  const roomText = String(room || '').replace(/校区/g, '').replace(/教学楼|学院楼|综合楼/g, '').replace(/\s+/g, '').trim()
  return [teacherText, roomText].filter(Boolean).join(' · ')
}

function formatCourseLines(course: PkCourseOnTable): CourseLineInfo {
  const raw = String(course.showText || '').trim()
  // "教师 课程名(班号) 周X A-B节 [周次] 教室"
  const match = /^(\S+)\s+(.+?)\(([^)]+)\)\s+(.+)$/.exec(raw)
  if (match) {
    const teacher = match[1]
    const name = match[2].trim()
    const code = match[3].trim()
    const rest = match[4].trim()
    const dayMatch = rest.match(/(星期[一二三四五六日])([0-9]{1,2}-[0-9]{1,2})节/)
    const weekMatch = rest.match(/\[([^\]]+)\]/)
    const roomMatch = rest.match(/\]\s*(.+)$/)
    const shortDay = dayMatch ? `${dayMatch[1].replace('星期', '周')}${dayMatch[2]}` : ''
    const weekText = weekMatch ? weekMatch[1] : ''
    const room = roomMatch ? roomMatch[1].trim() : ''
    if (isMobile.value) {
      return {
        title: name,
        mobileTitle: compactName(name),
        mobileMeta: compactMeta(teacher, room),
        sub: '',
        meta: '',
      }
    }
    return {
      title: `${teacher} ${name}(${code})`,
      mobileTitle: compactName(name),
      mobileMeta: compactMeta(teacher, room),
      sub: [shortDay, weekText, room].filter(Boolean).join(' '),
      meta: '',
    }
  }
  return {
    title: course.courseName || course.code || '课程',
    mobileTitle: compactName(course.courseName || course.code || '课程'),
    mobileMeta: course.code || '',
    sub: raw,
    meta: course.code || '',
  }
}

// ---- 网格渲染（对齐上游 updateTimeTable） ----
function updateTimeTable() {
  const maxRows = store.readTimeTableRows()
  const newTimeTable = Array.from({ length: maxRows }, () =>
    Array.from({ length: 7 }, () => [] as PkCourseOnTable[]),
  )
  const newMaxSpans = Array.from({ length: maxRows }, () => Array(7).fill(1) as number[])
  const newOccupied = Array.from({ length: maxRows }, () => Array(7).fill(false) as boolean[])

  const safeCourses = store.state.timeTableData.filter(
    (course) =>
      Array.isArray(course?.occupyTime) &&
      course.occupyTime.length > 0 &&
      typeof course?.occupyDay === 'number' &&
      course.occupyDay >= 1 &&
      course.occupyDay <= 7 &&
      course.occupyTime.every((slot) => slot >= 1 && slot <= maxRows),
  )

  const sortedCourses = [...safeCourses].sort((a, b) => b.occupyTime.length - a.occupyTime.length)

  interface CellRange {
    startTime: number
    endTime: number
    courses: PkCourseOnTable[]
  }
  const cellRanges: (CellRange | null)[][] = Array.from({ length: maxRows }, () => Array(7).fill(null))

  for (const course of sortedCourses) {
    const startRow = course.occupyTime[0] - 1
    const dayIndex = course.occupyDay - 1
    let merged = false

    for (let checkRow = 0; checkRow <= startRow; checkRow++) {
      const existingRange = cellRanges[checkRow][dayIndex]
      if (
        existingRange &&
        existingRange.startTime <= course.occupyTime[0] &&
        existingRange.endTime >= course.occupyTime[course.occupyTime.length - 1]
      ) {
        existingRange.courses.push(course)
        newTimeTable[checkRow][dayIndex].push(course)
        merged = true
        break
      }
    }

    if (!merged) {
      newTimeTable[startRow][dayIndex].push(course)
      cellRanges[startRow][dayIndex] = {
        startTime: course.occupyTime[0],
        endTime: course.occupyTime[course.occupyTime.length - 1],
        courses: [course],
      }
    }
  }

  for (let row = 0; row < maxRows; row++) {
    for (let col = 0; col < 7; col++) {
      const courses = newTimeTable[row][col]
      if (courses.length > 0) {
        newMaxSpans[row][col] = Math.max(...courses.map((c) => c.occupyTime.length))
      }
    }
  }

  for (let row = 0; row < maxRows; row++) {
    for (let col = 0; col < 7; col++) {
      const span = newMaxSpans[row][col]
      if (span > 1) {
        for (let i = 1; i < span; i++) {
          if (row + i < maxRows) newOccupied[row + i][col] = true
        }
      }
    }
  }

  timeTable.value = newTimeTable
  maxSpans.value = newMaxSpans
  occupiedGrid.value = newOccupied
}

// ---- 学分统计 ----
const creditSummary = computed(() => {
  if (!store.isMajorSelected()) return null
  return store.creditSummary()
})

/** 课表是否已有课程（决定渲染网格还是空态引导）。 */
const hasCourses = computed(() => timeTable.value.some((row) => row.some((cell) => cell.length > 0)))

function handleCellClick(dayIndex: number, rowIndex: number) {
  if (!store.isMajorSelected()) return
  if ((store.state.occupied?.[rowIndex]?.[dayIndex] ?? []).length > 0) return
  emit('cellClick', dayIndex + 1, rowIndex + 1)
}

watch(
  () => store.state.timeTableData,
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
    <div v-if="isMobile" class="px-1 pb-2 text-xs text-base-content/55">
      {{ t('schedule.longPressHint') }}
    </div>

    <div
      class="overflow-hidden rounded-2xl border border-line/70 bg-base-100 shadow-sm"
    >
      <div
        v-if="creditSummary"
        class="relative border-b border-line/70 bg-base-200/45 px-3 py-2"
      >
        <span
          class="absolute right-2 top-2 flex h-5 w-5 cursor-help items-center justify-center rounded-full bg-warning text-xs font-black leading-none text-base-100"
          :title="t('schedule.creditNote')"
        >
          !
        </span>
        <div class="flex flex-wrap items-center gap-3 text-xs text-base-content/80 md:text-xs">
          <span class="font-bold text-base-content">{{ t('schedule.creditSummary') }}</span>
          <span>{{ t('schedule.selectedCredit', { value: creditSummary.selectedTotal.toFixed(1) }) }}</span>
          <span>{{ t('schedule.majorCredit', { value: creditSummary.selectedMajor.toFixed(1) }) }}</span>
          <span>{{ t('schedule.generalCredit', { value: creditSummary.selectedGeneral.toFixed(1) }) }}</span>
        </div>
      </div>

      <EmptyState
        v-if="timeTable.length > 0 && !hasCourses"
        class="border-b border-line/60"
        :icon="BookOpen"
        :title="t('schedule.timetableEmptyTitle')"
        :description="t('schedule.timetableEmptyHint')"
      />
      <EmptyState
        v-else-if="timeTable.length === 0"
        :icon="BookOpen"
        :title="t('schedule.selectMajorFirst')"
      />

      <!-- 移动端 7 列固定 42px+（~45px/列）导致课程块标题截断到 1-2 字、元信息
           8px 不可读。小屏改为横向滚动：列宽 min-w-[60px]，整体宽出容器，可读性优先。 -->
      <div :class="hasCourses ? (isMobile ? 'overflow-x-auto gf-scrollbar-none' : '') : 'hidden'">
      <table class="w-full border-collapse table-fixed" :class="isMobile ? 'min-w-[466px]' : ''">
        <thead>
          <tr class="bg-base-200/60">
            <th class="w-[42px] border border-line/70 p-1 text-xs font-semibold text-base-content/70 md:w-[78px] md:p-2 md:text-xs">
              {{ t('schedule.arrangement') }}
            </th>
            <th
              v-for="day in WEEKDAY_KEYS"
              :key="day"
              class="min-w-[60px] border border-line/70 p-1 text-xs font-semibold text-base-content/70 md:p-2 md:text-xs"
            >
              {{ t(`schedule.weekdays.${day}`) }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(row, index) in timeTable"
            :key="index"
            :class="[index === timeTable.length - 1 ? 'bg-error/5' : index % 2 === 0 ? 'bg-base-100' : 'bg-base-200/30']"
          >
            <td
              class="border border-line/70 p-1 text-center text-xs font-semibold text-base-content/70 md:p-2 md:text-xs"
            >
              {{ t('schedule.sectionLabel', { section: index + 1 }) }}
            </td>
            <template v-for="(courses, dayIndex) in row" :key="dayIndex">
              <!-- rowspan 已占用的列槽不渲染 td，否则整行多出一列导致错位 -->
              <td
                v-if="!occupiedGrid[index][dayIndex]"
                class="border border-line/70 p-[2px] align-top text-center md:p-1"
                :rowspan="maxSpans[index][dayIndex]"
                @click="handleCellClick(dayIndex, index)"
              >
                <div
                  v-if="courses.length > 0"
                  class="h-full rounded-xl"
                  :class="isMobile ? 'min-h-[44px]' : 'min-h-[54px]'"
                >
                  <div
                    v-for="(course, courseIndex) in courses"
                    :key="course.code + '_' + courseIndex"
                    class="flex min-h-0 flex-1 flex-col justify-center overflow-hidden px-1 py-1 text-xs leading-tight text-white md:px-2 md:py-2 md:text-xs"
                    :class="[isMobile ? 'min-h-[44px] text-center' : 'text-left', courseIndex !== courses.length - 1 ? 'border-b border-dashed border-white/60' : '']"
                    :style="courseCardStyle(course)"
                    @touchstart.stop="onPressStart(course, $event)"
                    @touchmove.stop="onPressMove($event)"
                    @touchend.stop="onPressCancel()"
                    @touchcancel.stop="onPressCancel()"
                    @mousedown.stop="onPressStart(course, $event)"
                    @mouseup.stop="onPressCancel()"
                    @mouseleave.stop="onPressCancel()"
                    @click.stop="emit('openDetail', course)"
                  >
                    <template v-if="isMobile">
                      <span class="max-w-full truncate text-xs font-extrabold leading-tight">{{ formatCourseLines(course).mobileTitle }}</span>
                      <span v-if="formatCourseLines(course).mobileMeta" class="mt-0.5 max-w-full truncate text-[10px] opacity-85">{{ formatCourseLines(course).mobileMeta }}</span>
                    </template>
                    <template v-else>
                      <span class="break-words font-extrabold tracking-tight">{{ formatCourseLines(course).title }}</span>
                      <span v-if="formatCourseLines(course).sub" class="mt-1 break-words whitespace-pre-line opacity-95">{{ formatCourseLines(course).sub }}</span>
                    </template>
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
</template>
