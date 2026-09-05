<script setup lang="ts">
// 已选课程列表（v2 富卡片）：课名+班次数、冲突红标、课号、学院·学分、教师、
// 排课摘要、退课按钮；顶部「搜索课程名/课号」纯前端过滤。
// 点击课程行写入 clickedCourseInfo 并 emit('openDetail')，由父级弹出
// 浮动「选择教学班」弹窗（左右栏高度不再受内联班级列影响）。
import { computed, ref } from 'vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import { BookOpen, Save, Search, X } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import { queueFlashMessage } from '@/runtime/flash-message'
import { deriveConflicts } from '@/site/utils/pkConflict'
import { COURSE_STATUS, useScheduleStore } from '@/site/composables/useScheduleStore'
import type { PkStagedCourse } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

/** 周几 i18n key（与 locales schedule.weekdays.* 对齐）。 */
const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const
const pendingDrop = ref<PkStagedCourse | null>(null)
const keyword = ref('')

const emit = defineEmits<{
  openPicker: []
  openDetail: []
}>()

const dropDialogOpen = computed({
  get: () => pendingDrop.value !== null,
  set: (open: boolean) => {
    if (!open) pendingDrop.value = null
  },
})

// ---- 冲突派生（与课表/统计同判据）----
const conflicts = computed(() => deriveConflicts(store.state.occupied))

function courseConflicts(course: PkStagedCourse) {
  return conflicts.value.get(course.courseCode) ?? []
}

// ---- 搜索过滤（课程名/课号，纯前端）----
const filteredCourses = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  if (!query) return store.state.commonLists.stagedCourses
  return store.state.commonLists.stagedCourses.filter(
    (course) =>
      (course.courseNameReserved || course.courseName || '').toLowerCase().includes(query) ||
      course.courseCode.toLowerCase().includes(query),
  )
})

function statusLabel(course: PkStagedCourse): string {
  if (course.status === 2) return t('schedule.statusSelected')
  if (course.status === 1) return t('schedule.statusStaged')
  return t('schedule.statusUnselected')
}

function statusClass(course: PkStagedCourse): string {
  if (course.status === 2) return 'gf-badge gf-badge-success'
  if (course.status === 1) return 'gf-badge gf-badge-warning'
  return 'gf-badge gf-badge-muted'
}

function selectCourse(course: PkStagedCourse) {
  store.setClickedCourseInfo({
    courseCode: course.courseCode,
    courseName: course.courseNameReserved,
  })
  emit('openDetail')
}

function dropCourse(course: PkStagedCourse) {
  pendingDrop.value = course
}

function clearCourseClass(course: PkStagedCourse) {
  if (course.status === COURSE_STATUS.UNSELECTED) {
    store.popStagedCourse(course.courseCode)
  } else {
    store.clearStagedCourseClass(course.courseCode)
  }
  store.solidify()
}

function confirmDrop() {
  if (!pendingDrop.value) return
  store.popStagedCourse(pendingDrop.value.courseCode)
  // 退课必须立即持久化，否则刷新/重进页面时 loadSolidify 会用旧的
  // localStorage 覆盖内存，刚退掉的课程会"复活"（表现为无法退课）。
  store.solidify()
  pendingDrop.value = null
}

function saveTimetable() {
  try {
    store.saveSelectedCourses()
    store.solidify()
    queueFlashMessage(t('schedule.saveSuccess'), 'success')
  } catch {
    queueFlashMessage(t('schedule.saveFailed'), 'error')
  }
}

/** 课表里已排入的班级数（用于展示）。 */
function arrangedClassCount(course: PkStagedCourse): number {
  return course.courseDetail.filter((d) => d.status === 2).length
}

/** 排课摘要：已排班级的「周次: 天(节)」串联。 */
function arrangementSummary(course: PkStagedCourse): string {
  const parts: string[] = []
  for (const detail of course.courseDetail) {
    if (detail.status !== 1 && detail.status !== 2) continue
    for (const arr of detail.arrangementInfo) {
      const span = arr.occupyTime.length === 1
        ? `${arr.occupyTime[0]}`
        : `${arr.occupyTime[0]}-${arr.occupyTime[arr.occupyTime.length - 1]}`
      const weekday = WEEKDAY_KEYS[arr.occupyDay - 1]
      const dayLabel = weekday ? t(`schedule.weekdays.${weekday}`) : String(arr.occupyDay)
      parts.push(t('schedule.arrangementBrief', { day: dayLabel, sections: span, room: arr.occupyRoom || '' }))
    }
  }
  return parts.slice(0, 3).join('；') + (parts.length > 3 ? '…' : '')
}

/** 教师名串联（优先 stagedCourse.teacher，空则取已排班级教师）。 */
function teacherSummary(course: PkStagedCourse): string {
  const named = course.teacher.map((item) => item.teacherName).filter(Boolean)
  if (named.length > 0) return named.join('、')
  for (const detail of course.courseDetail) {
    if (detail.status !== 1 && detail.status !== 2) continue
    const teachers = detail.teachers.map((item) => item.teacherName).filter(Boolean)
    if (teachers.length > 0) return teachers.join('、')
  }
  return ''
}
</script>

<template>
  <div class="flex min-h-0 flex-col gap-3">
    <!-- 顶部操作区：两行解耦，彻底避免 352px 侧栏内搜索框被挤压至 108px 导致占位文本截断 -->
    <div class="flex flex-col gap-2">
      <!-- 行 1：全宽搜索栏 + 左侧 Search 图标 + 右侧一键清空 X 按钮 -->
      <div class="relative w-full">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
        <input
          v-model="keyword"
          type="search"
          class="gf-input gf-input-md w-full pl-9 pr-8 rounded-xl transition-all"
          :placeholder="t('schedule.listSearchPlaceholder')"
          :aria-label="t('schedule.listSearchPlaceholder')"
        />
        <button
          v-if="keyword"
          type="button"
          class="absolute right-2.5 top-1/2 -translate-y-1/2 rounded-full p-1 text-base-content/40 hover:text-base-content hover:bg-base-200 focus-visible:outline-none transition-colors"
          :title="t('schedule.clearSearch')"
          :aria-label="t('schedule.clearSearch')"
          @click="keyword = ''"
        >
          <X class="h-3.5 w-3.5" />
        </button>
      </div>

      <!-- 行 2：主副动作分级排布（主动作「选择课程」占据主体比例，辅助动作「保存课表」作为紧凑工具按钮），视觉重心平稳自洽 -->
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="gf-button gf-button-primary flex-1 h-10 justify-center gap-2 rounded-xl text-sm font-semibold shadow-xs transition-transform active:scale-[0.96]"
          @click="emit('openPicker')"
        >
          <BookOpen class="h-4 w-4" />
          <span>{{ t('schedule.openPicker') }}</span>
        </button>
        <button
          type="button"
          class="gf-button gf-button-secondary shrink-0 h-10 justify-center gap-1.5 rounded-xl border border-line/80 bg-base-100 px-3.5 text-sm font-medium text-base-content/80 shadow-xs transition-all hover:bg-base-200 hover:text-base-content active:scale-[0.96]"
          :title="t('schedule.saveTimetable')"
          @click="saveTimetable"
        >
          <Save class="h-4 w-4 text-base-content/60" />
          <span>{{ t('schedule.saveTimetable') }}</span>
        </button>
      </div>
    </div>

    <!-- 搜索结果计数反馈栏 -->
    <div
      v-if="keyword.trim()"
      class="flex items-center justify-between px-1 text-xs text-base-content/60"
    >
      <span>{{ t('schedule.searchResultsCount', { count: filteredCourses.length }) }}</span>
      <button
        type="button"
        class="text-xs text-primary hover:underline"
        @click="keyword = ''"
      >
        {{ t('schedule.clearSearch') }}
      </button>
    </div>

    <EmptyState
      v-if="!store.state.commonLists.stagedCourses.length"
      class="gf-panel rounded-2xl border border-line/70 shadow-[0_2px_10px_-4px_rgba(0,0,0,0.05)] flex-1 p-6"
      :icon="BookOpen"
      :title="t('schedule.emptyStagedTitle')"
      :description="t('schedule.emptyStagedHint')"
    >
      <button
        type="button"
        class="gf-button gf-button-sm gf-button-primary mt-1 shadow-sm transition-transform active:scale-[0.96]"
        @click="emit('openPicker')"
      >
        <BookOpen class="h-3.5 w-3.5" />
        {{ t('schedule.emptyStagedAction') }}
      </button>
    </EmptyState>
    <EmptyState
      v-else-if="!filteredCourses.length"
      class="gf-panel rounded-2xl border border-line/70 shadow-[0_2px_10px_-4px_rgba(0,0,0,0.05)] flex-1 p-6"
      :icon="Search"
      :title="t('schedule.listSearchEmpty')"
      :description="t('schedule.emptySearchHint')"
    >
      <button
        type="button"
        class="gf-button gf-button-sm gf-button-ghost mt-1 border border-line/70 transition-transform active:scale-[0.96]"
        @click="keyword = ''"
      >
        <X class="h-3.5 w-3.5" />
        {{ t('schedule.clearSearch') }}
      </button>
    </EmptyState>

    <ul v-else class="gf-panel rounded-2xl border border-line/70 shadow-[0_2px_10px_-4px_rgba(0,0,0,0.05)] gf-scrollbar-thin flex-1 divide-y divide-line/60 overflow-y-auto overscroll-contain">
      <li
        v-for="course in filteredCourses"
        :key="course.courseCode"
        class="px-3 py-2.5 transition-colors duration-150 hover:bg-base-200/40"
        :class="courseConflicts(course).length > 0 ? 'bg-error/5' : ''"
      >
        <div class="flex items-start gap-2">
          <button
            type="button"
            class="group min-w-0 flex-1 text-left"
            @click="selectCourse(course)"
          >
            <div class="flex flex-wrap items-center gap-1.5">
              <span
                class="truncate text-[13px] font-semibold text-base-content group-hover:text-primary transition-colors duration-150"
                :title="course.courseNameReserved"
              >
                {{ course.courseNameReserved }}
              </span>
              <span
                v-if="courseConflicts(course).length > 0"
                class="gf-badge gf-badge-error"
                :title="courseConflicts(course).map((item) => item.courseName).join('、')"
              >
                {{ t('schedule.conflictBadge') }}
              </span>
              <span :class="statusClass(course)">{{ statusLabel(course) }}</span>
            </div>
            <p class="mt-0.5 truncate text-[11px] text-primary/75">
              {{ course.courseCode }}
              <span class="text-base-content/50"> · {{ t('schedule.credit', { credit: course.credit }) }}</span>
              <template v-if="arrangedClassCount(course) > 0">
                <span class="text-base-content/50"> · </span>{{ t('schedule.classCount', { count: course.courseDetail.length }) }}
              </template>
            </p>
            <p v-if="teacherSummary(course)" class="mt-0.5 truncate text-[11px] text-base-content/55">
              {{ teacherSummary(course) }}
            </p>
            <p v-if="arrangementSummary(course)" class="mt-0.5 line-clamp-2 text-[11px] text-base-content/50">
              {{ arrangementSummary(course) }}
            </p>
          </button>
          <button
            type="button"
            class="gf-button gf-button-sm gf-button-ghost shrink-0 transition-transform active:scale-[0.96]"
            @click="course.status === 2 ? dropCourse(course) : clearCourseClass(course)"
          >
            {{ course.status === 2 ? t('schedule.dropCourse') : t('schedule.clear') }}
          </button>
        </div>
      </li>
    </ul>

    <!-- 退课确认弹窗 -->
    <DialogRoot v-model:open="dropDialogOpen">
      <DialogPortal>
        <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40" />
        <DialogContent
          class="fixed left-1/2 top-1/2 z-[2100] w-[88vw] max-w-[360px] -translate-x-1/2 -translate-y-1/2 outline-none"
        >
          <div class="rounded-2xl border border-line/70 bg-base-100 p-5 shadow-lg">
            <DialogTitle class="text-sm font-bold text-base-content">{{ t('schedule.dropCourse') }}</DialogTitle>
            <DialogDescription class="mt-2 text-[13px] text-base-content/70">
              {{ pendingDrop?.courseNameReserved }}（{{ pendingDrop?.courseCode }}）
            </DialogDescription>
            <div class="mt-4 flex justify-end gap-2">
              <button type="button" class="gf-button gf-button-md gf-button-ghost" @click="pendingDrop = null">
                {{ t('schedule.cancel') }}
              </button>
              <button type="button" class="gf-button gf-button-md gf-button-danger" @click="confirmDrop">
                {{ t('schedule.dropCourse') }}
              </button>
            </div>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </div>
</template>
