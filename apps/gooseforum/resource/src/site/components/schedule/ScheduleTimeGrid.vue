<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, MapPin, User } from '@lucide/vue'
import { CUSTOM_EVENT_CODE_PREFIX } from '@/site/utils/pkConflict'
import { formatWeeksText } from '@/site/utils/pkArrange'
import { timetableCourseStyle } from '@/site/utils/timetableCourseStyle'
import { cardMinHeightFor, cellInnerHeightFor, compactTeacherName as compactTeachers, computeRowHeights, dayPartLabelForRow, formatDisplayWeeks as displayWeeks, interactiveRowMetrics, teacherName, weekParityLabel, type GridLayout } from '@/site/utils/timetableGrid'
import type { PkCourseOnTable } from '@/site/types/pk'
import type { SectionTime } from '@/site/utils/sectionTimes'

// No store, persistence or fetching here: official records stay in page memory.
const props = withDefaults(defineProps<{
  grid: GridLayout
  sectionTimes: SectionTime[]
  isMobile: boolean
  interactive?: boolean
  showCourseCodes?: boolean
  activeDay?: number
  isConflicted?: (course: PkCourseOnTable) => boolean
  courseLabel?: (course: PkCourseOnTable) => string
}>(), { interactive: true, showCourseCodes: true, activeDay: 0, isConflicted: () => false })
const emit = defineEmits<{
  cellClick: [dayIndex: number, rowIndex: number]
  openDetail: [course: PkCourseOnTable]
  pressStart: [course: PkCourseOnTable, event: Event]
  pressMove: [event: Event]
  pressCancel: []
  courseEnter: [course: PkCourseOnTable, event: MouseEvent]
  courseLeave: []
}>()
const { t } = useI18n()
const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const
const computedRowHeights = computed(() => computeRowHeights(props.grid, interactiveRowMetrics(props.isMobile)))
function isCustomEvent(course: PkCourseOnTable) { return course.code.startsWith(CUSTOM_EVENT_CODE_PREFIX) }
function courseCardStyle(course: PkCourseOnTable) { return timetableCourseStyle(course.code || course.courseName || course.showText || 'course', isCustomEvent(course)) }
function courseAriaLabel(course: PkCourseOnTable) { return props.courseLabel?.(course) ?? [course.courseName, course.occupyRoom, teacherName(course), formatDisplayWeeks(course.occupyWeek)].filter(Boolean).join(' · ') }
function compactTeacherName(raw: string) { return compactTeachers(raw, 2) }
function formatDisplayWeeks(weeks: readonly number[] | undefined) { return displayWeeks(weeks, t) }
function weekParityBadge(weeks: readonly number[] | undefined) { return weekParityLabel(weeks, t) }
function sectionTimeText(index: number) { const item = props.sectionTimes[index]; return item ? `${item.start}-${item.end}` : '' }
function dayPartLabelAt(index: number) { return dayPartLabelForRow(index + 1, props.sectionTimes, t) }
function cellInnerHeight(row: number, day: number) { return cellInnerHeightFor(props.grid.cellSpans[row]?.[day] || 1, computedRowHeights.value, interactiveRowMetrics(props.isMobile), row) }
function cardMinHeight(row: number, day: number, count: number) { return cardMinHeightFor(props.grid.cellSpans[row]?.[day] || 1, computedRowHeights.value, count, interactiveRowMetrics(props.isMobile), row) }
</script>

<template>
  <table
    class="w-full border-collapse table-fixed"
    :class="isMobile ? 'min-w-[530px]' : 'min-w-[640px]'"
  >
    <thead>
      <tr class="bg-base-200/60 h-9 md:h-10">
        <th class="w-[50px] border border-line/70 p-1 text-[11px] font-semibold text-base-content/70 sm:w-[60px] md:w-[86px] md:p-2 md:text-xs">
          {{ t('schedule.arrangement') }}
        </th>
        <th
          v-for="(day, dayIndex) in WEEKDAY_KEYS"
          :key="day"
          class="border border-line/70 p-1 text-[11px] font-semibold text-base-content/70 md:p-2 md:text-xs"
        >
          <span :class="activeDay === dayIndex + 1 ? 'text-primary' : ''">{{ t(`schedule.weekdays.${day}`) }}</span>
        </th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="(row, index) in grid.cellCourses"
        :key="index"
        class="border-b border-line/70"
        :style="{ height: `${computedRowHeights[index]}px` }"
        :class="[index === grid.cellCourses.length - 1 ? 'bg-base-200/50' : index % 2 === 0 ? 'bg-base-100' : 'bg-base-200/30']"
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
            v-if="!grid.occupiedGrid[index][dayIndex]"
            class="h-px border border-line/70 p-[2px] align-top text-left md:p-1"
            :rowspan="grid.cellSpans[index][dayIndex]"
            :tabindex="interactive && courses.length === 0 ? 0 : undefined"
            :role="interactive && courses.length === 0 ? 'button' : undefined"
            :aria-label="interactive && courses.length === 0 ? t('schedule.emptyCell') : undefined"
            @click="interactive && emit('cellClick', dayIndex, index)"
            @keydown.enter.prevent="courses.length === 0 && interactive && emit('cellClick', dayIndex, index)"
            @keydown.space.prevent="courses.length === 0 && interactive && emit('cellClick', dayIndex, index)"
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
                class="schedule-course-card relative flex min-h-0 min-w-0 flex-1 flex-col justify-between overflow-hidden rounded-xl border select-none text-left transition-all outline-none focus-visible:ring-2 focus-visible:ring-primary/60 focus-visible:ring-offset-1"
                :class="[
                  interactive ? 'cursor-pointer' : '',
                  courses.length > 1 || grid.cellSpans[index][dayIndex] === 1
                    ? 'p-1 md:p-1.5'
                    : 'p-1 sm:p-1.5 md:p-2',
                ]"
                :style="[courseCardStyle(course), { minHeight: `${cardMinHeight(index, dayIndex, courses.length)}px` }]"
                :tabindex="interactive ? 0 : undefined"
                :role="interactive ? 'button' : 'group'"
                :aria-label="courseAriaLabel(course)"
                @click.stop="interactive && emit('openDetail', course)"
                @keydown.enter.stop.prevent="interactive && emit('openDetail', course)"
                @keydown.space.stop.prevent="interactive && emit('openDetail', course)"
                @touchstart.stop="interactive && emit('pressStart', course, $event)"
                @touchmove.stop="interactive && emit('pressMove', $event)"
                @touchend.stop="emit('pressCancel')"
                @touchcancel.stop="emit('pressCancel')"
                @mousedown.stop="interactive && emit('pressStart', course, $event)"
                @mouseup.stop="emit('pressCancel')"
                @mouseleave.stop="emit('pressCancel')"
                @mouseenter="interactive && emit('courseEnter', course, $event)"
                @mouseleave="emit('courseLeave')"
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
                <template v-if="courses.length > 1 || grid.cellSpans[index][dayIndex] === 1">
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
                          v-if="showCourseCodes && course.code && !isCustomEvent(course)"
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
                        v-if="showCourseCodes && course.code && !isCustomEvent(course)"
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

.schedule-course-card[role="button"]:active {
  transform: scale(0.985);
}
</style>
