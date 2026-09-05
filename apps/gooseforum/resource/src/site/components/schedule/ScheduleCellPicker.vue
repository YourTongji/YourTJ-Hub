<script setup lang="ts">
// 课表时段选课 / 同时段课程替换弹窗：
// 1. 已选备选课程：从 stagedCourses 筛选出该时段（天 + 节次）有安排的教学班
// 2. 该时段全校课程：调用 getPkCoursesByTime 获取该时段全校课程，并支持展开班级直接加入
// 3. 同时段替换模式：传入 replacingCourse 时，点击目标班级将自动移除原课程并加入新课程
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowLeftRight,
  ChevronDown,
  ChevronRight,
  Loader2,
  Plus,
  RefreshCw,
  Search,
  X,
} from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { getPkCourseDetails, getPkCoursesByTime } from '@/runtime/pk-api'
import { getCourseBaseCode, type PkConflictItem } from '@/site/utils/pkConflict'
import { sortPlannedCoursesFirst } from '@/site/utils/pkCourseOrder'
import { getRowSection, getSectionRangeText } from '@/site/utils/timetable'
import type { PkCourse, PkCourseDetail, PkCourseOnTable, PkStagedCourse } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const props = defineProps<{
  open: boolean
  day: number | null
  section: number | null
  replacingCourse?: PkCourseOnTable | null
}>()

const emit = defineEmits<{
  close: []
  conflict: [detail: PkCourseDetail, conflicts: PkConflictItem[]]
  staged: []
  replaced: [fromCourseName: string, toCourseName: string]
}>()

// 弹窗无障碍（reka-ui Dialog）：焦点移入/Tab 圈禁/Esc 关闭/焦点恢复由 Dialog 内置处理。
const dialogOpen = computed({
  get: () => props.open,
  set: (open: boolean) => {
    if (!open) emit('close')
  },
})

const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const

const slotTitle = computed(() => {
  if (props.day === null || props.section === null) return ''
  const weekday = t(`schedule.weekdays.${WEEKDAY_KEYS[props.day - 1]}`)
  const calendarId = store.state.majorSelected.calendarId ?? 0
  const rowSec = getRowSection(props.section, calendarId)
  const range = getSectionRangeText(rowSec, calendarId)
  return `${weekday} ${t('schedule.sectionLabel', { section: props.section })}（${range}）`
})

const replacingTargetName = computed(() => {
  if (!props.replacingCourse) return ''
  return props.replacingCourse.courseName || props.replacingCourse.code
})

interface CellCandidate {
  courseName: string
  courseCode: string
  credit: number
  detail: PkCourseDetail
}

/** 搜索关键词 */
const searchKeyword = ref('')

/** 备选课程中该时段教学班（排除正在替换的班级本身） */
const stagedCandidates = computed<CellCandidate[]>(() => {
  if (props.day === null || props.section === null) return []
  const list: CellCandidate[] = []
  const kw = searchKeyword.value.trim().toLowerCase()
  const replacingClassCode = props.replacingCourse?.code

  for (const course of store.state.commonLists.stagedCourses) {
    for (const detail of course.courseDetail) {
      if (replacingClassCode && detail.code === replacingClassCode) {
        // 排除当前正在替换的原班级
        continue
      }
      const hits = detail.arrangementInfo.some(
        (arr) => arr.occupyDay === props.day && arr.occupyTime.includes(props.section!),
      )
      if (hits) {
        const name = course.courseNameReserved || course.courseName || course.courseCode
        if (kw) {
          const teacherMatches = detail.teachers.some((tch) => tch.teacherName.toLowerCase().includes(kw))
          if (
            !name.toLowerCase().includes(kw) &&
            !course.courseCode.toLowerCase().includes(kw) &&
            !detail.code.toLowerCase().includes(kw) &&
            !teacherMatches
          ) {
            continue
          }
        }
        list.push({
          courseName: name,
          courseCode: course.courseCode,
          credit: course.credit,
          detail,
        })
      }
    }
  }
  return sortPlannedCoursesFirst(list, store.state.commonLists.compulsoryCourses)
})

// ---- 该时段全校课程（P10 getPkCoursesByTime）----
const slotElectives = ref<PkCourse[]>([])
const loadingSlotCourses = ref(false)
const slotCoursesError = ref('')
let slotRequestSeq = 0

/** 课程班级详情缓存：courseCode -> PkCourseDetail[] */
const classCache = ref<Map<string, PkCourseDetail[]>>(new Map())
const expandedCourseCodes = ref<Set<string>>(new Set())
const loadingClassesCourseCodes = ref<Set<string>>(new Set())

function campusText(campus?: string | string[]): string {
  if (!campus) return ''
  return Array.isArray(campus) ? campus.join('、') : campus
}

const filteredSlotElectives = computed(() => {
  const kw = searchKeyword.value.trim().toLowerCase()
  const filtered = !kw
    ? slotElectives.value
    : slotElectives.value.filter((course) => {
        return (
          course.courseName.toLowerCase().includes(kw) ||
          course.courseCode.toLowerCase().includes(kw) ||
          (course.faculty && course.faculty.toLowerCase().includes(kw)) ||
          campusText(course.campus).toLowerCase().includes(kw)
        )
      })
  return sortPlannedCoursesFirst(filtered, store.state.commonLists.compulsoryCourses)
})

async function fetchSlotCourses() {
  if (!props.open || props.day === null || props.section === null) return
  const calendarId = store.state.majorSelected.calendarId ?? 0
  const rowSec = getRowSection(props.section, calendarId)
  const seq = ++slotRequestSeq

  loadingSlotCourses.value = true
  slotCoursesError.value = ''
  try {
    const res = await getPkCoursesByTime(calendarId, props.day, rowSec)
    if (seq !== slotRequestSeq) return
    slotElectives.value = res.courses || []
  } catch (err) {
    if (seq !== slotRequestSeq) return
    slotCoursesError.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  } finally {
    if (seq === slotRequestSeq) {
      loadingSlotCourses.value = false
    }
  }
}

watch(
  [() => props.open, () => props.day, () => props.section],
  ([open, day, section]) => {
    if (open && day !== null && section !== null) {
      searchKeyword.value = ''
      expandedCourseCodes.value = new Set()
      void fetchSlotCourses()
    } else {
      slotElectives.value = []
      loadingSlotCourses.value = false
      slotCoursesError.value = ''
      searchKeyword.value = ''
    }
  },
  { immediate: true },
)

async function toggleExpandCourse(course: PkCourse) {
  const code = course.courseCode
  if (expandedCourseCodes.value.has(code)) {
    const updated = new Set(expandedCourseCodes.value)
    updated.delete(code)
    expandedCourseCodes.value = updated
    return
  }

  // 展开并获取班级
  const updated = new Set(expandedCourseCodes.value)
  updated.add(code)
  expandedCourseCodes.value = updated

  if (!classCache.value.has(code)) {
    const calendarId = store.state.majorSelected.calendarId ?? 0
    loadingClassesCourseCodes.value.add(code)
    try {
      const detailsMap = await getPkCourseDetails(calendarId, [code])
      const list = detailsMap[code] || []
      classCache.value.set(code, list)
    } catch {
      classCache.value.set(code, [])
    } finally {
      loadingClassesCourseCodes.value.delete(code)
    }
  }
}

function getCourseClasses(code: string): PkCourseDetail[] {
  return classCache.value.get(code) || []
}

function isSlotMatch(detail: PkCourseDetail): boolean {
  if (props.day === null || props.section === null) return false
  return detail.arrangementInfo.some(
    (arr) => arr.occupyDay === props.day && arr.occupyTime.includes(props.section!),
  )
}

function arrangementText(detail: PkCourseDetail): string {
  return detail.arrangementInfo.map((arr) => arr.arrangementText).join('；')
}

function teacherText(detail: PkCourseDetail): string {
  return detail.teachers.map((teacher) => teacher.teacherName).filter(Boolean).join('、')
}

function buildStagedCourseFromPkCourse(course: PkCourse, details: PkCourseDetail[]): PkStagedCourse {
  return {
    courseCode: course.courseCode,
    courseName: `${course.courseName}(${course.courseCode})`,
    courseNameReserved: course.courseName,
    credit: course.credit,
    courseType: '选',
    courseNature: course.courseNature,
    teacher: [],
    status: 0,
    courseDetail: details.map((detail) => ({
      ...detail,
      status: detail.status ?? 0,
    })),
  }
}

/** 执行加入/替换 */
function applySelect(
  courseInfo: { courseCode: string; courseName: string; credit?: number; rawCourse?: PkCourse },
  detail: PkCourseDetail,
) {
  const isReplacing = Boolean(props.replacingCourse)
  const oldCourseName = props.replacingCourse?.courseName || props.replacingCourse?.code || ''
  const oldClassCode = props.replacingCourse?.code || ''
  const oldBaseCode = getCourseBaseCode(oldClassCode)

  if (isReplacing) {
    // 若替换为不同课程，先从备选/课表中移除原课程
    if (oldBaseCode && oldBaseCode !== courseInfo.courseCode) {
      store.popStagedCourse(oldClassCode)
    }
  }

  // 确保新课程已在 stagedCourses 中
  const existsInStaged = store.state.commonLists.stagedCourses.some(
    (c) => c.courseCode === courseInfo.courseCode,
  )
  if (!existsInStaged) {
    const allDetails = classCache.value.get(courseInfo.courseCode) || [detail]
    if (courseInfo.rawCourse) {
      store.pushStagedCourse(buildStagedCourseFromPkCourse(courseInfo.rawCourse, allDetails))
    } else {
      store.pushStagedCourse({
        courseCode: courseInfo.courseCode,
        courseName: `${courseInfo.courseName}(${courseInfo.courseCode})`,
        courseNameReserved: courseInfo.courseName,
        credit: courseInfo.credit ?? 0,
        courseType: '选',
        teacher: [],
        status: 0,
        courseDetail: allDetails.map((d) => ({ ...d, status: d.status ?? 0 })),
      })
    }
  }

  // 设置点击上下文并入表
  store.setClickedCourseInfo({
    courseCode: courseInfo.courseCode,
    courseName: courseInfo.courseName,
  })
  const result = store.stageCourse(detail)
  store.solidify()

  if (isReplacing) {
    emit('replaced', oldCourseName, courseInfo.courseName)
  } else {
    if (result.conflicts && result.conflicts.length > 0) {
      emit('conflict', detail, result.conflicts)
    } else {
      emit('staged')
    }
  }
  emit('close')
}

function tryStageStagedCandidate(candidate: CellCandidate) {
  applySelect(
    {
      courseCode: candidate.courseCode,
      courseName: candidate.courseName,
      credit: candidate.credit,
    },
    candidate.detail,
  )
}
</script>

<template>
  <DialogRoot v-model:open="dialogOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40 backdrop-blur-xs transition-opacity duration-200" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[2100] flex max-h-[85vh] h-[85vh] w-[92vw] max-w-[620px] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl border border-line/70 bg-base-100 shadow-2xl outline-none"
      >
        <!-- 弹窗顶栏 -->
        <div class="flex items-center justify-between gap-3 border-b border-line/60 bg-base-100 px-4 py-3.5 sm:px-5 shrink-0">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <DialogTitle class="truncate text-sm font-bold text-base-content sm:text-base">
                {{ props.replacingCourse ? t('schedule.replaceModalTitle') : t('schedule.cellPickTitle') }}
              </DialogTitle>
              <span
                v-if="props.replacingCourse"
                class="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-[11px] font-medium text-primary"
              >
                <ArrowLeftRight class="h-3 w-3" />
                {{ t('schedule.replaceCourseInSlot') }}
              </span>
            </div>
            <DialogDescription class="mt-0.5 truncate text-[11px] text-base-content/60 sm:text-xs">
              {{ slotTitle }}
              <template v-if="props.replacingCourse">
                · {{ t('schedule.replaceTarget', { name: replacingTargetName }) }}
              </template>
            </DialogDescription>
          </div>
          <button
            type="button"
            class="gf-icon-button shrink-0"
            :aria-label="t('common.close')"
            @click="emit('close')"
          >
            <X class="h-4 w-4" />
          </button>
        </div>

        <!-- 搜索栏 -->
        <div class="border-b border-line/60 bg-base-200/30 px-4 py-2.5 sm:px-5 shrink-0">
          <div class="relative flex items-center">
            <Search class="pointer-events-none absolute left-3 h-3.5 w-3.5 text-base-content/40" />
            <input
              v-model="searchKeyword"
              type="search"
              class="gf-input w-full pl-8.5 pr-8 py-1.5 text-xs rounded-xl bg-base-100"
              :placeholder="t('schedule.searchSlotCoursePlaceholder')"
            />
            <button
              v-if="searchKeyword"
              type="button"
              class="absolute right-2.5 p-0.5 text-base-content/40 hover:text-base-content rounded"
              @click="searchKeyword = ''"
            >
              <X class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>

        <!-- 内容区域 -->
        <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain p-4 sm:p-5 space-y-4 gf-scrollbar-thin">
          <!-- Section 1: 备选课程中该时段课程 -->
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold text-base-content/80 flex items-center gap-1.5">
                <span>{{ t('schedule.stagedSlotCourses') }}</span>
                <span v-if="stagedCandidates.length > 0" class="text-[11px] font-normal text-base-content/50">({{ stagedCandidates.length }})</span>
              </span>
            </div>

            <div v-if="stagedCandidates.length === 0" class="rounded-xl border border-dashed border-line/70 bg-base-200/20 px-4 py-4 text-center">
              <p class="text-xs text-base-content/50">{{ t('schedule.cellPickEmpty') }}</p>
            </div>

            <ul v-else class="divide-y divide-line/60 rounded-xl border border-line/70 bg-base-100 overflow-hidden shadow-xs">
              <li
                v-for="candidate in stagedCandidates"
                :key="candidate.detail.code"
                class="p-3 transition-colors hover:bg-base-200/40"
              >
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0 flex-1">
                    <div class="flex items-center gap-2">
                      <span class="text-[13px] font-medium text-base-content truncate">{{ candidate.courseName }}</span>
                      <span class="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                        {{ t('schedule.credit', { credit: candidate.credit }) }}
                      </span>
                    </div>
                    <span class="mt-0.5 block text-[11px] text-base-content/50">{{ candidate.detail.code }}</span>
                    <p v-if="teacherText(candidate.detail)" class="mt-0.5 text-[12px] text-base-content/70">
                      {{ t('schedule.teacherWith', { value: teacherText(candidate.detail) }) }}
                    </p>
                    <p class="mt-0.5 text-[12px] text-base-content/60 leading-snug">
                      {{ arrangementText(candidate.detail) }}
                    </p>
                  </div>
                  <button
                    type="button"
                    class="gf-button gf-button-xs gf-button-primary shrink-0 gap-1 rounded-lg self-center"
                    @click="tryStageStagedCandidate(candidate)"
                  >
                    <ArrowLeftRight v-if="props.replacingCourse" class="h-3.5 w-3.5" />
                    <Plus v-else class="h-3.5 w-3.5" />
                    <span>{{ props.replacingCourse ? t('schedule.addAndReplace') : t('schedule.addToStagedAndSchedule') }}</span>
                  </button>
                </div>
              </li>
            </ul>
          </div>

          <!-- Section 2: 该时段全校选修课 -->
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold text-base-content/80 flex items-center gap-1.5">
                <span>{{ t('schedule.slotElectiveCourses') }}</span>
                <span v-if="filteredSlotElectives.length > 0" class="text-[11px] font-normal text-base-content/50">({{ filteredSlotElectives.length }})</span>
              </span>
              <button
                v-if="!loadingSlotCourses"
                type="button"
                class="text-[11px] text-primary hover:underline flex items-center gap-1 cursor-pointer"
                @click="fetchSlotCourses"
              >
                <RefreshCw class="h-3 w-3" />
                <span>{{ t('common.refresh') }}</span>
              </button>
            </div>

            <!-- 加载中 -->
            <div
              v-if="loadingSlotCourses"
              class="flex items-center justify-center gap-2 rounded-xl border border-line/60 bg-base-200/20 py-8 text-xs text-base-content/55"
            >
              <Loader2 class="h-4 w-4 animate-spin text-primary" />
              <span>{{ t('schedule.loadingSlotCourses') }}</span>
            </div>

            <!-- 加载失败 -->
            <div
              v-else-if="slotCoursesError"
              class="rounded-xl border border-error/30 bg-error/10 p-4 text-center"
            >
              <p class="text-xs text-error mb-2">{{ slotCoursesError }}</p>
              <button
                type="button"
                class="gf-button gf-button-xs gf-button-secondary"
                @click="fetchSlotCourses"
              >
                {{ t('common.retry') }}
              </button>
            </div>

            <!-- 空态 -->
            <div
              v-else-if="filteredSlotElectives.length === 0"
              class="rounded-xl border border-dashed border-line/70 bg-base-200/20 p-4 text-center"
            >
              <p class="text-xs text-base-content/50">{{ t('schedule.noSlotCourses') }}</p>
            </div>

            <!-- 课程列表 -->
            <div v-else class="space-y-2.5">
              <div
                v-for="course in filteredSlotElectives"
                :key="course.courseCode"
                class="rounded-xl border border-line/70 bg-base-100 shadow-xs transition-all overflow-hidden"
              >
                <!-- 课程头部 -->
                <div
                  class="flex items-center justify-between gap-3 p-3.5 cursor-pointer hover:bg-base-200/30 transition-colors"
                  @click="toggleExpandCourse(course)"
                >
                  <div class="min-w-0 flex-1">
                    <div class="flex items-center gap-2 flex-wrap">
                      <span class="text-[13px] font-medium text-base-content">{{ course.courseName }}</span>
                      <span class="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                        {{ t('schedule.credit', { credit: course.credit }) }}
                      </span>
                      <span v-if="course.faculty" class="rounded bg-base-200 px-1.5 py-0.5 text-[10px] text-base-content/65">
                        {{ course.faculty }}
                      </span>
                      <span v-if="campusText(course.campus)" class="rounded bg-base-200 px-1.5 py-0.5 text-[10px] text-base-content/65">
                        {{ campusText(course.campus) }}
                      </span>
                    </div>
                    <span class="mt-0.5 block text-[11px] text-base-content/50 font-mono">{{ course.courseCode }}</span>
                  </div>

                  <div class="flex items-center gap-2 shrink-0">
                    <button
                      type="button"
                      class="gf-button gf-button-xs gf-button-secondary gap-1 rounded-lg"
                      :aria-expanded="expandedCourseCodes.has(course.courseCode)"
                      @click.stop="toggleExpandCourse(course)"
                    >
                      <span>{{ expandedCourseCodes.has(course.courseCode) ? t('schedule.hideClassesForSlot') : t('schedule.viewClassesForSlot') }}</span>
                      <ChevronDown v-if="expandedCourseCodes.has(course.courseCode)" class="h-3 w-3" />
                      <ChevronRight v-else class="h-3 w-3" />
                    </button>
                  </div>
                </div>

                <!-- 展开的班级列表 -->
                <div
                  v-if="expandedCourseCodes.has(course.courseCode)"
                  class="border-t border-line/60 bg-base-200/20 p-3 space-y-2"
                >
                  <!-- 班级加载中 -->
                  <div
                    v-if="loadingClassesCourseCodes.has(course.courseCode)"
                    class="flex items-center justify-center gap-2 py-4 text-xs text-base-content/55"
                  >
                    <Loader2 class="h-3.5 w-3.5 animate-spin text-primary" />
                    <span>{{ t('schedule.loading') }}</span>
                  </div>

                  <!-- 班级列表 -->
                  <template v-else>
                    <div
                      v-if="getCourseClasses(course.courseCode).length === 0"
                      class="py-3 text-center text-xs text-base-content/50"
                    >
                      {{ t('schedule.noClassesMatchingSlot') }}
                    </div>

                    <div
                      v-for="detail in getCourseClasses(course.courseCode)"
                      :key="detail.code"
                      class="rounded-lg border p-2.5 transition-all"
                      :class="isSlotMatch(detail) ? 'border-primary/40 bg-base-100 shadow-xs' : 'border-line/60 bg-base-100/70'"
                    >
                      <div class="flex items-start justify-between gap-2.5">
                        <div class="min-w-0 flex-1 space-y-1">
                          <div class="flex items-center gap-2 flex-wrap">
                            <span class="font-mono text-xs font-semibold text-base-content">{{ detail.code }}</span>
                            <span
                              v-if="isSlotMatch(detail)"
                              class="inline-flex items-center rounded bg-success/15 px-1.5 py-0.5 text-[10px] font-medium text-success"
                            >
                              {{ t('schedule.slotClassesMatching') }}
                            </span>
                            <span v-if="detail.campus" class="text-[11px] text-base-content/55">
                              {{ detail.campus }}
                            </span>
                          </div>

                          <p v-if="teacherText(detail)" class="text-xs text-base-content/75">
                            {{ t('schedule.teacherWith', { value: teacherText(detail) }) }}
                          </p>

                          <p class="text-xs text-base-content/65 leading-snug">
                            {{ arrangementText(detail) }}
                          </p>
                        </div>

                        <button
                          type="button"
                          class="gf-button gf-button-xs gf-button-primary shrink-0 gap-1 rounded-lg self-center"
                          @click="applySelect({ courseCode: course.courseCode, courseName: course.courseName, credit: course.credit, rawCourse: course }, detail)"
                        >
                          <ArrowLeftRight v-if="props.replacingCourse" class="h-3.5 w-3.5" />
                          <Plus v-else class="h-3.5 w-3.5" />
                          <span>{{ props.replacingCourse ? t('schedule.addAndReplace') : t('schedule.addToStagedAndSchedule') }}</span>
                        </button>
                      </div>
                    </div>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
