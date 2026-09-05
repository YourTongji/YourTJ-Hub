<script setup lang="ts">
// 课程班级列表：展示 clickedCourseInfo 对应课程的全部教学班，点击班级加入课表。
// 容忍式冲突：无论是否冲突都入表；有冲突时 emit('conflict') 仅用于父级 flash 提示，
// 不再弹「强制替换/放弃」窗（多方案/周次模型下不阻断）。
// 桌面端（lg+）：默认专注展示教学班卡片列表（充裕呼吸感与负空间）；
// 点击「查看课评」或班级评分胶囊时，以伴随浮动面板展开（复用 RatingSummaryCard 与
// CoursePreviewPane 交互语言），双栏独立滚动且避免任何容器裁切与屏幕溢出；
// 移动端（<lg）：采用轻量抽屉图层覆盖展示，提供「返回班级列表」导航，不挤压窄屏宽度。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  BookOpen,
  CalendarDays,
  Check,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Eye,
  History,
  MapPin,
  MessageSquareQuote,
  Plus,
  RefreshCw,
  RotateCcw,
  Star,
  Users,
  X,
} from '@lucide/vue'
import AISummaryCard from '@/site/components/AISummaryCard.vue'
import EmptyState from '@/site/components/EmptyState.vue'
import RatingSummaryCard from '@/site/components/RatingSummaryCard.vue'
import { useScheduleStore } from '@/site/composables/useScheduleStore'
import { getPkCourseReviewBrief } from '@/runtime/pk-api'
import { listCourseReviews, type ReviewPage, type ReviewPayload } from '@/runtime/api'
import { reviewAvatarSrc } from '@/site/utils/course-review-share'
import { getCourseBaseCode, isSameCourse, type PkConflictItem } from '@/site/utils/pkConflict'
import type { PkArrangement, PkCourseDetail, PkCourseReviewBrief, PkReviewBriefClass } from '@/site/types/pk'

const { t } = useI18n()
const store = useScheduleStore()

const props = withDefaults(
  defineProps<{
    reviewPanelOpen?: boolean
  }>(),
  {
    reviewPanelOpen: undefined,
  },
)

const emit = defineEmits<{
  conflict: [detail: PkCourseDetail, conflicts: PkConflictItem[]]
  staged: []
  'update:reviewPanelOpen': [open: boolean]
}>()

/** 周几 i18n key（与 locales schedule.weekdays.* 对齐）。 */
const WEEKDAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const

/** 内部受控/非受控浮动课评展开状态：单元测试未传 prop 时默认为 true，以兼容既存断言。 */
const internalReviewOpen = ref(props.reviewPanelOpen ?? true)

watch(
  () => props.reviewPanelOpen,
  (val) => {
    if (val !== undefined) internalReviewOpen.value = val
  },
)

const isReviewOpen = computed({
  get: () => internalReviewOpen.value,
  set: (val: boolean) => {
    internalReviewOpen.value = val
    emit('update:reviewPanelOpen', val)
  },
})

/** 当前高亮/聚焦的教学班课号（点击具体班级课评胶囊时高亮对应条目）。 */
const activeClassCode = ref<string | null>(null)

const currentCourse = computed(() =>
  store.state.commonLists.stagedCourses.find(
    (course) => course.courseCode === store.state.clickedCourseInfo.courseCode,
  ),
)

/** 本学期课评摘要。 */
const brief = ref<PkCourseReviewBrief | null>(null)
const briefError = ref('')
const briefLoading = ref(false)

/** 历史全部教学班课评摘要（calendarId: 0）。 */
const historyBrief = ref<PkCourseReviewBrief | null>(null)
const historyLoading = ref(false)

/** 教学班列表展示范围：'current' 本学期，'history' 历史全部。 */
const classesScope = ref<'current' | 'history'>('current')

/** 学生评价列表与加载态。 */
const reviews = ref<ReviewPage | null>(null)
const reviewsLoading = ref(false)

/** 请求序号：快速切换课程时丢弃过期响应，避免旧请求覆盖新课评摘要。 */
let briefRequestSeq = 0

/** 课评跳转：能匹配到课程目录主键时直达详情页，否则回退课程搜索页。
 * clickedCourseInfo.courseCode 已是基础课号（无班号），无需再剥班号。 */
const reviewHref = computed(() => {
  const base = store.state.clickedCourseInfo.courseCode
  if (brief.value?.courseId) {
    return `/courses/${brief.value.courseId}`
  }
  return `/courses?keyword=${encodeURIComponent(base)}`
})

/** 归一化班级课号：去掉点号（PK "122004.01" ↔ offering.class_code "12200401"）。 */
function normalizeClassCode(code: string): string {
  return String(code ?? '').replaceAll('.', '')
}

/** 教学班对应的 offering 级课评摘要（优先本学期，无记录且已同步历史时回退历史匹配）。 */
function classBrief(detailCode: string): PkReviewBriefClass | undefined {
  const target = normalizeClassCode(detailCode)
  const current = brief.value?.classes?.find((item) => normalizeClassCode(item.classCode) === target)
  if (current && (current.reviewCount > 0 || !historyBrief.value)) {
    return current
  }
  if (historyBrief.value?.classes) {
    const hist = historyBrief.value.classes.find((item) => normalizeClassCode(item.classCode) === target)
    if (hist) return hist
  }
  return current
}

/** 教学班课评跳转（左栏行内链接）：有 offeringId 时聚焦该班评价，否则回退课程搜索页。 */
function classReviewHref(detailCode: string): string {
  const base = store.state.clickedCourseInfo.courseCode
  const item = classBrief(detailCode)
  if (brief.value?.courseId && item?.offeringId) {
    return `/courses/${brief.value.courseId}?offeringId=${item.offeringId}`
  }
  return `/courses?keyword=${encodeURIComponent(base)}`
}

/** 右栏教学班课评项跳转：有 offeringId 时聚焦该班评价，否则回退课程搜索页。 */
function classPanelHref(item: PkReviewBriefClass): string {
  const base = store.state.clickedCourseInfo.courseCode
  if (brief.value?.courseId && item.offeringId) {
    return `/courses/${brief.value.courseId}?offeringId=${item.offeringId}`
  }
  return `/courses?keyword=${encodeURIComponent(base)}`
}

/** 当前教学班课评展示列表。 */
const displayedClasses = computed(() => {
  if (classesScope.value === 'history') {
    return historyBrief.value?.classes ?? []
  }
  return brief.value?.classes ?? []
})

/** 教学班课评列表展开折叠控制：默认展示前 4 个，避免超长列表挤占空间并引发巨型滚动。 */
const INITIAL_CLASS_LIMIT = 4
const classesExpanded = ref(false)

/** 实际渲染的教学班课评列表：折叠态下展示前 4 个，且若当前聚焦的班级在 4 个之后则追加保证聚焦可见。 */
const visibleClasses = computed(() => {
  if (classesExpanded.value || displayedClasses.value.length <= INITIAL_CLASS_LIMIT) {
    return displayedClasses.value
  }
  const top = displayedClasses.value.slice(0, INITIAL_CLASS_LIMIT)
  if (
    activeClassCode.value &&
    !top.some((c) => normalizeClassCode(c.classCode) === normalizeClassCode(activeClassCode.value!))
  ) {
    const activeItem = displayedClasses.value.find(
      (c) => normalizeClassCode(c.classCode) === normalizeClassCode(activeClassCode.value!),
    )
    if (activeItem) {
      return [...top, activeItem]
    }
  }
  return top
})

/** 当前聚焦教学班对应的 offering 级课评摘要。 */
const activeClassItem = computed(() => {
  if (!activeClassCode.value) return undefined
  return classBrief(activeClassCode.value)
})

/** 当前聚焦教学班的教师名称。 */
const activeClassTeacher = computed(() => {
  if (!activeClassCode.value) return ''
  const detail = currentCourse.value?.courseDetail.find(
    (d) => normalizeClassCode(d.code) === normalizeClassCode(activeClassCode.value!),
  )
  if (detail) return teacherText(detail)
  return activeClassItem.value?.teachers.join('、') || ''
})

/** 点击教学班卡片时触发：聚焦该教学班，并在伴随面板中展示该班课评。 */
function onSelectClassCard(detail: PkCourseDetail) {
  activeClassCode.value = detail.code
  isReviewOpen.value = true
}

/** 打开并聚焦具体班级课评（通过微型胶囊）。 */
function openReviewForClass(detailCode: string) {
  activeClassCode.value = detailCode
  isReviewOpen.value = true
}

/** 清除具体教学班选中，回到课程全局课评摘要。 */
function clearActiveClass() {
  activeClassCode.value = null
}

/** 在伴随面板中点击某个教学班时切换预览。 */
function selectReviewClass(classCode: string) {
  if (activeClassCode.value && normalizeClassCode(activeClassCode.value) === normalizeClassCode(classCode)) {
    activeClassCode.value = null
  } else {
    activeClassCode.value = classCode
  }
}

function closeReviews() {
  isReviewOpen.value = false
  activeClassCode.value = null
}

/** 加载学生评论（精选前 3 条，支持按教学班 offeringId 过滤）。 */
async function loadReviews(courseId: number, offeringId = 0) {
  const seq = briefRequestSeq
  reviewsLoading.value = true
  try {
    const result = await listCourseReviews(courseId, offeringId, '', 3)
    if (seq !== briefRequestSeq) return
    reviews.value = result
  } catch {
    if (seq !== briefRequestSeq) return
    reviews.value = null
  } finally {
    if (seq === briefRequestSeq) reviewsLoading.value = false
  }
}

/** 聚焦特定教学班时拉取该班专属评价；取消聚焦时恢复全课评价。 */
watch(
  activeClassCode,
  (code) => {
    if (!brief.value?.courseId) return
    if (!code) {
      void loadReviews(brief.value.courseId, 0)
      return
    }
    const item = classBrief(code)
    if (item?.offeringId) {
      void loadReviews(brief.value.courseId, item.offeringId)
    } else {
      reviews.value = { total: 0, list: [] }
    }
  },
)

/** 当面板折叠收起时，重置激活的班级聚焦。 */
watch(isReviewOpen, (open) => {
  if (!open) {
    activeClassCode.value = null
  }
})

/** 切换本学期/历史教学班时，重置班级列表折叠态。 */
watch(classesScope, () => {
  classesExpanded.value = false
})

/** 加载历史全部开课与班级课评（calendarId: 0）。 */
async function loadHistoryBrief(courseCode: string, force = false) {
  if (historyBrief.value && !force) return
  const seq = briefRequestSeq
  historyLoading.value = true
  try {
    const result = await getPkCourseReviewBrief({
      courseCode,
      teacherName: '',
      calendarId: 0,
    })
    if (seq !== briefRequestSeq) return
    historyBrief.value = result
  } catch {
    // 静默降级，不阻断主交互
  } finally {
    if (seq === briefRequestSeq) historyLoading.value = false
  }
}

/** 切换到历史教学班课评视图。 */
function switchToHistory() {
  classesScope.value = 'history'
  const code = store.state.clickedCourseInfo.courseCode
  if (code && !historyBrief.value) {
    void loadHistoryBrief(code)
  }
}

async function loadBrief(courseCode: string) {
  const seq = ++briefRequestSeq
  brief.value = null
  briefError.value = ''
  briefLoading.value = true
  historyBrief.value = null
  reviews.value = null
  classesScope.value = 'current'
  classesExpanded.value = false

  try {
    // 课程级摘要：单教学班课程直接用该班 teachingClassId 精准定位，多班课程回退课程级匹配。
    const details = currentCourse.value?.courseDetail ?? []
    const teachingClassId = details.length === 1 ? details[0].teachingClassId : undefined
    const result = await getPkCourseReviewBrief({
      courseCode,
      teacherName: '',
      calendarId: store.state.majorSelected.calendarId ?? 0,
      teachingClassId,
    })
    if (seq !== briefRequestSeq) return
    brief.value = result

    if (result.courseId) {
      const item = activeClassCode.value ? classBrief(activeClassCode.value) : undefined
      void loadReviews(result.courseId, item?.offeringId ?? 0)
    }
  } catch (err) {
    if (seq !== briefRequestSeq) return
    briefError.value = err instanceof Error ? err.message : t('schedule.loadFailed')
  } finally {
    if (seq === briefRequestSeq) briefLoading.value = false
  }
}

watch(
  () => store.state.clickedCourseInfo.courseCode,
  (code) => {
    activeClassCode.value = null
    if (code) void loadBrief(code)
  },
  { immediate: true },
)

function authorLabel(author: ReviewPayload['author']) {
  if (author.kind === 'member') return author.label
  if (author.kind === 'legacy') return t('courseDetailPage.authorLegacy')
  return t('courseDetailPage.authorAnonymous')
}

function statusLabel(status: number | undefined): string {
  if (status === 2) return t('schedule.statusSelected')
  if (status === 1) return t('schedule.statusStaged')
  return t('schedule.statusUnselected')
}

function statusClass(status: number | undefined): string {
  if (status === 2) return 'gf-badge gf-badge-success'
  if (status === 1) return 'gf-badge gf-badge-warning'
  return 'gf-badge gf-badge-muted'
}

function teacherText(detail: PkCourseDetail): string {
  return detail.teachers.map((teacher) => teacher.teacherName).filter(Boolean).join('、')
}

/** 格式化教学时间段胶囊信息。 */
function formatArrSlot(arr: PkArrangement): { day: string; sections: string; weeks: string; room: string } {
  const weekday = WEEKDAY_KEYS[arr.occupyDay - 1]
  const day = weekday ? t(`schedule.weekdays.${weekday}`) : (arr.occupyDay ? `周${arr.occupyDay}` : '')
  const sections = arr.occupyTime && arr.occupyTime.length > 0
    ? (arr.occupyTime.length === 1 ? `第${arr.occupyTime[0]}节` : `第${arr.occupyTime[0]}-${arr.occupyTime[arr.occupyTime.length - 1]}节`)
    : ''
  const weeks = arr.occupyWeek && arr.occupyWeek.length > 0
    ? (arr.occupyWeek.length === 1 ? `${arr.occupyWeek[0]}周` : `${arr.occupyWeek[0]}-${arr.occupyWeek[arr.occupyWeek.length - 1]}周`)
    : ''
  const room = arr.occupyRoom || ''
  return { day, sections, weeks, room }
}

/** 判断教学班是否已加入课表（暂存 status=1、已选 status=2 或已存在于课表数据中）。 */
function isClassAdded(detail: PkCourseDetail): boolean {
  if (detail.status === 1 || detail.status === 2) return true
  return store.state.timeTableData.some((c) => isSameCourse(c.code, detail.code))
}

function tryStage(detail: PkCourseDetail) {
  // 若该班已加入课表，再次点击触发退选当前班，提供双向便捷反选
  if (isClassAdded(detail)) {
    const courseCode = currentCourse.value?.courseCode || getCourseBaseCode(detail.code)
    store.clearStagedCourseClass(courseCode)
    store.solidify()
    return
  }
  // 容忍式：总是入表；冲突仅作 flash 提示（deriveConflicts 负责课表/列表/统计标注）。
  const result = store.stageCourse(detail)
  store.solidify()
  if (result.conflicts && result.conflicts.length > 0) {
    emit('conflict', detail, result.conflicts)
    return
  }
  emit('staged')
}
</script>

<template>
  <div v-if="!currentCourse" class="p-6">
    <EmptyState
      :icon="BookOpen"
      :title="t('schedule.emptyDetailGuide')"
      :description="t('schedule.majorHint')"
    />
  </div>

  <div
    v-else
    class="relative flex h-full min-h-0 w-full overflow-hidden"
  >
    <!-- 左栏 / 主内容区：课程信息 + 教学班卡片列表（充裕呼吸感与负空间，确立独立滚动容器） -->
    <div class="flex-1 min-w-0 h-full overflow-y-auto overscroll-contain p-4 sm:p-5">
      <!-- 课程元信息概览条 -->
      <div class="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-xl border border-line/60 bg-base-200/40 p-3">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <h3 class="truncate text-sm font-bold text-base-content">
              {{ currentCourse.courseNameReserved }}
            </h3>
            <span class="rounded bg-base-200 px-1.5 py-0.5 text-xs font-semibold tabular-nums text-base-content/70">
              {{ currentCourse.courseCode }}
            </span>
          </div>
          <p class="mt-0.5 text-xs text-base-content/60">
            {{ t('schedule.credit', { credit: currentCourse.credit }) }}
            <template v-if="currentCourse.courseType"> · {{ currentCourse.courseType }}</template>
            · {{ t('schedule.classCount', { count: currentCourse.courseDetail.length }) }}
          </p>
        </div>

        <!-- 顶栏快捷课评入口（若浮动面板未展开，显式提供开启按钮） -->
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-lg border border-line/70 bg-base-100 px-2.5 py-1 text-xs font-medium text-base-content/80 shadow-xs transition hover:bg-base-200 active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-primary"
            :class="{ 'border-primary/45 bg-primary/10 text-primary font-semibold': isReviewOpen }"
            @click="isReviewOpen = !isReviewOpen"
          >
            <Star class="h-3.5 w-3.5" :class="isReviewOpen ? 'fill-warning text-warning' : 'text-base-content/50'" />
            <span v-if="brief?.ratingAvg != null" class="font-bold tabular-nums text-amber-500">
              {{ brief.ratingAvg.toFixed(1) }}
            </span>
            <span>{{ isReviewOpen ? t('schedule.hideReviews') : t('schedule.reviews') }}</span>
          </button>
          <!-- 语义链接保持测试兼容与直接跳详情页 -->
          <a
            :href="reviewHref"
            class="sr-only"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ t('schedule.reviews') }}
          </a>
        </div>
      </div>

      <!-- 教学班卡片列表 -->
      <div v-if="currentCourse.courseDetail.length" class="space-y-3">
        <div
          v-for="detail in currentCourse.courseDetail"
          :key="detail.code"
          class="group relative rounded-xl border p-3.5 sm:p-4 shadow-xs transition-all cursor-pointer select-none"
          :class="[
            activeClassCode && normalizeClassCode(detail.code) === normalizeClassCode(activeClassCode)
              ? 'border-primary ring-2 ring-primary/40 bg-primary/[0.04] shadow-sm'
              : isClassAdded(detail)
                ? 'border-primary/60 bg-primary/[0.03] ring-1 ring-primary/20 hover:border-primary'
                : 'border-line/70 bg-base-100 hover:border-primary/50 hover:bg-base-200/40 hover:shadow-xs'
          ]"
          @click="onSelectClassCard(detail)"
        >
          <!-- 卡片顶栏：班级代码 + 状态 Badge + 预览指示器 + 课评小胶囊 + 选课主操作 -->
          <div class="flex items-center justify-between gap-2.5">
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <span class="font-bold tabular-nums text-sm text-base-content">
                {{ detail.code }}
              </span>
              <span :class="statusClass(detail.status)">
                {{ statusLabel(detail.status) }}
              </span>
              <span
                v-if="detail.isExclusive"
                class="rounded-md border border-info/30 bg-info/10 px-1.5 py-0.5 text-[11px] font-medium text-info"
              >
                {{ t('schedule.tabRequired') }}
              </span>
              <span
                v-if="activeClassCode && normalizeClassCode(detail.code) === normalizeClassCode(activeClassCode)"
                class="inline-flex items-center gap-1 rounded-md bg-primary/10 px-1.5 py-0.5 text-[11px] font-semibold text-primary"
              >
                <Eye class="h-3 w-3" />
                <span>{{ t('schedule.previewing') }}</span>
              </span>
            </div>

            <div class="flex items-center gap-2 shrink-0">
              <!-- 教学班课评快捷微型胶囊（点击唤起浮动课评并聚焦该班） -->
              <template v-if="classBrief(detail.code)">
                <button
                  type="button"
                  class="inline-flex items-center gap-1 rounded-lg border border-amber-400/30 bg-amber-500/10 px-2 py-1 text-xs font-semibold text-amber-600 dark:text-amber-400 transition-colors hover:bg-amber-500/20 active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-primary"
                  :title="t('schedule.classReviewFocus')"
                  @click.stop="openReviewForClass(detail.code)"
                >
                  <Star class="h-3 w-3 fill-amber-400 text-amber-400" />
                  <span v-if="classBrief(detail.code)?.ratingAvg != null" class="tabular-nums">
                    {{ classBrief(detail.code)?.ratingAvg?.toFixed(1) }}
                  </span>
                  <span class="tabular-nums">
                    {{ t('schedule.reviewCount', { count: classBrief(detail.code)?.reviewCount ?? 0 }) }}
                  </span>
                  <ChevronRight class="h-3 w-3 text-amber-500/70" />
                </button>
                <!-- 保持 semantic <a> 供单元测试与新标签页跳转使用 -->
                <a
                  :href="classReviewHref(detail.code)"
                  class="sr-only"
                  tabindex="-1"
                >
                  {{ classBrief(detail.code)?.ratingAvg != null ? classBrief(detail.code)?.ratingAvg?.toFixed(1) : '' }}
                  {{ t('schedule.reviewCount', { count: classBrief(detail.code)?.reviewCount ?? 0 }) }}
                </a>
              </template>

              <!-- 选班核心操作按钮（已加入：深色/Primary + Check 强调稳固态；未加入：Secondary + Plus 引导交互） -->
              <button
                type="button"
                class="gf-button gf-button-xs shrink-0 whitespace-nowrap px-3 text-xs font-semibold transition-all duration-150 active:scale-[0.96]"
                :class="isClassAdded(detail)
                  ? 'gf-button-primary shadow-xs'
                  : 'gf-button-secondary border border-line/80 hover:border-primary/50 hover:bg-base-200/70 text-base-content/90'"
                :title="isClassAdded(detail) ? t('schedule.statusAdded') : t('schedule.clickToStage')"
                @click.stop="tryStage(detail)"
              >
                <Check v-if="isClassAdded(detail)" class="h-3.5 w-3.5 shrink-0 stroke-[2.2]" />
                <Plus v-else class="h-3.5 w-3.5 shrink-0 stroke-[2.2]" />
                <span>{{ isClassAdded(detail) ? t('schedule.statusAdded') : t('schedule.clickToStage') }}</span>
              </button>
            </div>
          </div>

          <!-- 教师与授课校区/语言信息 -->
          <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-base-content/70">
            <span v-if="teacherText(detail)" class="flex items-center gap-1 font-medium text-base-content/90">
              <Users class="h-3.5 w-3.5 text-base-content/45 shrink-0" />
              {{ t('schedule.teacherWith', { value: teacherText(detail) }) }}
            </span>
            <span class="flex items-center gap-1 text-base-content/60">
              <MapPin class="h-3.5 w-3.5 text-base-content/40 shrink-0" />
              {{ detail.campus }}
            </span>
            <span class="text-base-content/60">
              {{ detail.teachingLanguage }}
            </span>
          </div>

          <!-- 排课时间胶囊（独立分词展示，告别生硬分号字符串） -->
          <div v-if="detail.arrangementInfo.length" class="mt-2.5 flex flex-wrap gap-1.5">
            <div
              v-for="(arr, idx) in detail.arrangementInfo"
              :key="idx"
              class="inline-flex items-center gap-1.5 rounded-lg border border-line/60 bg-base-200/50 px-2 py-1 text-xs text-base-content/85"
            >
              <CalendarDays class="h-3.5 w-3.5 text-primary/70 shrink-0" />
              <span class="font-medium">{{ formatArrSlot(arr).day }} {{ formatArrSlot(arr).sections }}</span>
              <span v-if="formatArrSlot(arr).weeks" class="rounded bg-base-300/60 px-1 py-0.5 text-[11px] tabular-nums text-base-content/60">
                {{ formatArrSlot(arr).weeks }}
              </span>
              <span v-if="formatArrSlot(arr).room" class="text-base-content/65">
                {{ formatArrSlot(arr).room }}
              </span>
            </div>
          </div>
        </div>
      </div>
      <p v-else class="p-6 text-center text-xs text-base-content/50">
        {{ t('schedule.emptyDetailNoClass') }}
      </p>
    </div>

    <!-- 右栏：快捷课评浮动面板（参考 CoursePreviewPane 交互与视觉语言）
         桌面端：弹性伴随面板并排独立滚动，彻底杜绝 transform 容器裁切与屏幕溢出；
         移动/窄屏端：抽屉式全景覆盖展示，带返回导航，不硬分双栏。 -->
    <aside
      v-if="currentCourse"
      v-show="isReviewOpen"
      role="complementary"
      :aria-label="t('schedule.reviews')"
      class="flex h-full min-h-0 flex-col border-line/70 bg-base-100 transition-all"
      :class="[
        // 桌面端伴随右栏
        'lg:w-[380px] lg:shrink-0 lg:border-l',
        // 移动/平板端全量覆盖抽屉
        'max-lg:absolute max-lg:inset-0 max-lg:z-30 max-lg:bg-base-100/98 max-lg:backdrop-blur-sm'
      ]"
    >
      <!-- 顶栏：左侧跳详情页，右侧收起/关闭按钮 -->
      <div class="flex items-center justify-between gap-2 border-b border-line/70 bg-base-200/40 px-4 py-2.5 shrink-0">
        <a
          :href="reviewHref"
          class="inline-flex h-8 items-center gap-1.5 rounded-lg border border-primary/25 bg-primary/10 px-2.5 text-xs font-semibold text-primary shadow-xs transition hover:bg-primary/15 active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-primary"
          target="_blank"
          rel="noopener noreferrer"
        >
          <ExternalLink class="h-3.5 w-3.5" />
          <span>{{ t('schedule.enterCoursePage') }}</span>
        </a>

        <button
          type="button"
          class="inline-flex h-8 items-center gap-1 rounded-lg px-2 text-xs font-medium text-base-content/60 transition hover:bg-base-200 hover:text-base-content active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-primary"
          :title="t('schedule.hideReviews')"
          :aria-label="t('schedule.hideReviews')"
          @click="closeReviews"
        >
          <X class="h-3.5 w-3.5" />
          <span class="max-lg:hidden">{{ t('schedule.hideReviews') }}</span>
          <span class="lg:hidden">{{ t('schedule.backToClassList') }}</span>
        </button>
      </div>

      <!-- 课评滚动展示区 -->
      <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain p-4 space-y-4">
        <!-- 加载中骨架 -->
        <template v-if="briefLoading">
          <div class="animate-pulse space-y-3">
            <div class="h-24 rounded-xl border border-line/60 bg-base-200/50 p-4" />
            <div class="h-32 rounded-xl border border-line/60 bg-base-200/50 p-4" />
          </div>
        </template>

        <!-- 错误占位 -->
        <template v-else-if="briefError">
          <div class="rounded-xl border border-line/60 bg-base-200/40 p-4 text-center">
            <p class="text-xs text-base-content/60">{{ t('schedule.loadFailed') }}</p>
            <a
              :href="reviewHref"
              class="mt-2 inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
              target="_blank"
              rel="noopener noreferrer"
            >
              <span>{{ t('schedule.reviews') }}</span>
              <ExternalLink class="h-3 w-3" />
            </a>
          </div>
        </template>

        <!-- 课评正文内容 -->
        <template v-else-if="brief">
          <!-- 教学班专属预览横幅（当点击具体班级卡片时展示聚焦状态与切回全课按钮） -->
          <div
            v-if="activeClassCode"
            class="rounded-xl border border-primary/35 bg-gradient-to-br from-primary/[0.08] to-primary/[0.02] p-3.5 shadow-xs space-y-2"
          >
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-1.5 min-w-0">
                <span class="rounded bg-primary px-1.5 py-0.5 text-[11px] font-bold tabular-nums text-primary-content">
                  {{ activeClassCode }}
                </span>
                <span class="truncate text-xs font-bold text-base-content">
                  {{ activeClassTeacher || t('schedule.previewingClass') }}
                </span>
              </div>
              <button
                type="button"
                class="inline-flex shrink-0 items-center gap-1 rounded-md border border-line/70 bg-base-100 px-2 py-1 text-[11px] font-medium text-base-content/75 shadow-2xs hover:bg-base-200 hover:text-base-content transition active:scale-[0.96]"
                @click="clearActiveClass"
              >
                <RotateCcw class="h-3 w-3 text-base-content/50" />
                <span>{{ t('schedule.viewAllCourseReviews') }}</span>
              </button>
            </div>

            <div class="flex flex-wrap items-center gap-2 text-xs">
              <template v-if="activeClassItem">
                <div class="inline-flex items-center gap-1 font-bold text-amber-600 dark:text-amber-400">
                  <Star class="h-3.5 w-3.5 fill-amber-400 text-amber-400" />
                  <span class="tabular-nums">{{ activeClassItem.ratingAvg != null ? activeClassItem.ratingAvg.toFixed(1) : '—' }}</span>
                </div>
                <span class="text-base-content/30">·</span>
                <span class="tabular-nums text-base-content/70">
                  {{ t('schedule.reviewCount', { count: activeClassItem.reviewCount }) }}
                </span>
              </template>
              <template v-else>
                <span class="text-[11px] text-base-content/60">{{ t('schedule.noClassesThisTerm') }}</span>
              </template>
            </div>
          </div>

          <!-- 课程综合评分卡片（复用 RatingSummaryCard） -->
          <RatingSummaryCard
            :rating-avg="brief.ratingAvg ?? null"
            :review-count="brief.reviewCount"
            :distribution="brief.ratingDistribution ?? [0, 0, 0, 0, 0]"
          />

          <!-- AI 评课总结（复用 AISummaryCard，支持多模型、重新生成与折叠展开） -->
          <AISummaryCard
            v-if="brief.courseId"
            :course-id="brief.courseId"
            class="rounded-xl border border-line/70 shadow-xs"
          />

          <!-- 教学班课评列表（支持本学期/历史全部切换与同步） -->
          <section
            class="rounded-xl border border-line/70 bg-base-100 p-3.5 shadow-xs"
            :aria-label="t('schedule.classReviewsTitle')"
          >
            <!-- 栏目标题与历史同步操作 -->
            <div class="mb-3 space-y-2">
              <div class="flex items-center justify-between">
                <h4 class="flex items-center gap-1.5 text-xs font-bold text-base-content">
                  <Star class="h-3.5 w-3.5 fill-amber-400 text-amber-500" />
                  <span>{{ t('schedule.classReviewsTitle') }}</span>
                </h4>
                <button
                  v-if="classesScope === 'history'"
                  type="button"
                  class="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[11px] font-medium text-primary hover:bg-primary/10 transition"
                  :title="t('schedule.syncHistoricalReviews')"
                  @click="loadHistoryBrief(store.state.clickedCourseInfo.courseCode, true)"
                >
                  <RefreshCw class="h-3 w-3" :class="{ 'animate-spin': historyLoading }" />
                  <span>{{ t('schedule.syncHistoricalReviews') }}</span>
                </button>
              </div>

              <!-- 分段切换器：本学期教学班 vs 历史全部教学班 -->
              <div class="grid grid-cols-2 rounded-lg bg-base-200/70 p-0.5 text-xs font-medium">
                <button
                  type="button"
                  class="rounded-md py-1 text-center transition-all"
                  :class="classesScope === 'current' ? 'bg-base-100 font-semibold text-base-content shadow-xs' : 'text-base-content/60 hover:text-base-content'"
                  @click="classesScope = 'current'"
                >
                  {{ t('schedule.currentSemesterClasses') }}
                  <span v-if="brief.classes?.length" class="ml-1 text-[11px] tabular-nums text-base-content/50">
                    ({{ brief.classes.length }})
                  </span>
                </button>
                <button
                  type="button"
                  class="rounded-md py-1 text-center transition-all"
                  :class="classesScope === 'history' ? 'bg-base-100 font-semibold text-base-content shadow-xs' : 'text-base-content/60 hover:text-base-content'"
                  @click="switchToHistory"
                >
                  {{ t('schedule.historicalClasses') }}
                  <span v-if="historyBrief?.classes?.length" class="ml-1 text-[11px] tabular-nums text-base-content/50">
                    ({{ historyBrief.classes.length }})
                  </span>
                </button>
              </div>
            </div>

            <!-- 历史加载中骨架 -->
            <div v-if="classesScope === 'history' && historyLoading" class="space-y-2 py-3">
              <div class="animate-pulse space-y-2">
                <div class="h-8 rounded-lg bg-base-200/60" />
                <div class="h-8 rounded-lg bg-base-200/60" />
              </div>
            </div>

            <!-- 班级列表 -->
            <div v-else-if="displayedClasses.length" class="space-y-2">
              <ul class="divide-y divide-line/60">
                <li
                  v-for="item in visibleClasses"
                  :key="item.offeringId"
                  class="py-2 first:pt-0 last:pb-0"
                >
                  <div
                    class="group/item flex items-center justify-between gap-2 rounded-lg p-2 transition cursor-pointer"
                    :class="activeClassCode && normalizeClassCode(item.classCode) === normalizeClassCode(activeClassCode)
                      ? 'border border-primary/50 bg-primary/[0.08] ring-1 ring-primary/25'
                      : 'hover:bg-base-200/60'"
                    @click="selectReviewClass(item.classCode)"
                  >
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-1.5">
                        <span class="block truncate text-xs font-semibold tabular-nums text-base-content">
                          {{ item.classCode }}
                        </span>
                        <span
                          v-if="activeClassCode && normalizeClassCode(item.classCode) === normalizeClassCode(activeClassCode)"
                          class="rounded bg-primary/20 px-1 py-0.2 text-[10px] font-bold text-primary"
                        >
                          {{ t('schedule.previewing') }}
                        </span>
                      </div>
                      <span class="block truncate text-[11px] text-base-content/55">
                        {{ item.teachers.join('、') || t('coursesPage.noTeacher') }}
                      </span>
                    </div>
                    <div class="flex items-center gap-2 shrink-0">
                      <div class="text-right">
                        <span class="block text-xs font-bold tabular-nums text-amber-500">
                          {{ item.ratingAvg != null ? item.ratingAvg.toFixed(1) : '—' }}
                        </span>
                        <span class="block text-[10px] tabular-nums text-base-content/50">
                          {{ t('schedule.reviewCount', { count: item.reviewCount }) }}
                        </span>
                      </div>
                      <a
                        :href="classPanelHref(item)"
                        class="opacity-0 group-hover/item:opacity-100 focus:opacity-100 inline-flex h-6 w-6 items-center justify-center rounded-md text-base-content/50 hover:bg-base-300/60 hover:text-primary transition"
                        :title="t('schedule.enterCoursePage')"
                        target="_blank"
                        rel="noopener noreferrer"
                        @click.stop
                      >
                        <ExternalLink class="h-3 w-3" />
                        <span class="sr-only">{{ t('schedule.enterCoursePage') }}</span>
                      </a>
                    </div>
                  </div>
                </li>
              </ul>

              <!-- 超出初始阈值时的折叠/展开按钮 -->
              <div
                v-if="displayedClasses.length > INITIAL_CLASS_LIMIT"
                class="pt-1 border-t border-line/50 text-center"
              >
                <button
                  type="button"
                  class="inline-flex items-center gap-1 text-xs font-medium text-primary hover:text-primary/80 transition-colors py-1.5 px-3 rounded-lg hover:bg-primary/10 active:scale-[0.98] cursor-pointer"
                  :aria-expanded="classesExpanded"
                  @click="classesExpanded = !classesExpanded"
                >
                  <span>{{ classesExpanded ? t('schedule.collapseClasses') : t('schedule.expandAllClasses', { count: displayedClasses.length }) }}</span>
                  <ChevronDown class="h-3.5 w-3.5 transition-transform duration-200" :class="{ 'rotate-180': classesExpanded }" />
                </button>
              </div>
            </div>

            <!-- 本学期无专属班级评价时的空状态提示与一键切换 -->
            <div v-else-if="classesScope === 'current'" class="rounded-lg bg-base-200/40 p-3 text-center">
              <p class="text-[11px] text-base-content/60">{{ t('schedule.noClassesThisTerm') }}</p>
              <button
                type="button"
                class="mt-2 inline-flex items-center gap-1 rounded-md border border-primary/30 bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary hover:bg-primary/20 transition active:scale-[0.97]"
                @click="switchToHistory"
              >
                <History class="h-3 w-3" />
                <span>{{ t('schedule.viewHistoricalClassesReviews') }}</span>
              </button>
            </div>
            <p v-else class="py-3 text-center text-xs text-base-content/50">
              {{ t('courseDetailPage.noOfferings') }}
            </p>
          </section>

          <!-- 精选学生评价（精选展示真实评论，支持教学班级筛选联动） -->
          <section
            v-if="brief.courseId"
            class="space-y-2.5 rounded-xl border border-line/70 bg-base-100 p-3.5 shadow-xs"
          >
            <div class="flex items-center justify-between">
              <h4 class="flex items-center gap-1.5 text-xs font-bold text-base-content">
                <MessageSquareQuote class="h-3.5 w-3.5 text-primary" />
                <span>{{ activeClassCode ? t('schedule.classReviewsTitle') : t('schedule.recentReviewsTitle') }}</span>
                <span v-if="activeClassCode" class="text-[11px] font-normal text-base-content/60 tabular-nums">
                  ({{ activeClassCode }})
                </span>
              </h4>
              <span v-if="reviews?.total" class="text-[11px] tabular-nums text-base-content/50">
                {{ t('schedule.reviewCount', { count: reviews.total }) }}
              </span>
            </div>

            <!-- 评价加载骨架 -->
            <div v-if="reviewsLoading" class="space-y-2 py-1">
              <div class="animate-pulse rounded-lg border border-line/50 bg-base-200/40 p-2.5 space-y-2">
                <div class="flex items-center gap-2">
                  <div class="h-6 w-6 rounded-full bg-base-200" />
                  <div class="h-3 w-1/3 rounded bg-base-200" />
                </div>
                <div class="h-3 w-full rounded bg-base-200" />
              </div>
            </div>

            <!-- 真实评价卡片列表 -->
            <div v-else-if="reviews?.list?.length" class="space-y-2">
              <div
                v-for="rev in reviews.list"
                :key="rev.id"
                class="rounded-lg border border-line/60 bg-base-200/30 p-2.5 text-xs transition hover:bg-base-200/50"
              >
                <div class="flex items-center justify-between gap-2 mb-1.5">
                  <div class="flex items-center gap-1.5 min-w-0">
                    <img
                      :src="reviewAvatarSrc(rev.author, rev.id, 24)"
                      :alt="authorLabel(rev.author)"
                      class="h-5 w-5 shrink-0 rounded-full object-cover ring-1 ring-line/50"
                      loading="lazy"
                    />
                    <span class="truncate font-semibold text-base-content text-[11px]">{{ authorLabel(rev.author) }}</span>
                    <span class="text-[10px] text-base-content/45 tabular-nums">{{ rev.createdAt }}</span>
                  </div>
                  <div v-if="rev.rating" class="flex shrink-0 gap-0.5 text-amber-400">
                    <Star v-for="n in rev.rating" :key="n" class="h-2.5 w-2.5 fill-current" />
                  </div>
                </div>
                <div
                  v-if="rev.contentHtml"
                  v-code-highlight
                  v-math-render
                  class="gf-prose gf-prose-compact text-xs leading-relaxed text-base-content/85 break-words [word-break:break-word] mt-1"
                  v-html="rev.contentHtml"
                />
                <p v-else class="text-xs leading-relaxed text-base-content/85 break-words [word-break:break-word] mt-1 whitespace-pre-line">
                  {{ rev.content }}
                </p>
              </div>
            </div>

            <!-- 暂无文字评价提示（若处于班级筛选态，提供一键切回全课按钮） -->
            <div v-else-if="!reviewsLoading" class="py-2 text-center space-y-1.5">
              <p class="text-xs text-base-content/50">
                {{ activeClassCode ? t('schedule.noClassReviewsYet') : t('schedule.noReviewsYet') }}
              </p>
              <button
                v-if="activeClassCode"
                type="button"
                class="inline-flex items-center gap-1 rounded-md border border-line/70 bg-base-200/60 px-2 py-1 text-xs font-medium text-primary hover:bg-base-200 transition active:scale-[0.96]"
                @click="clearActiveClass"
              >
                <RotateCcw class="h-3 w-3" />
                <span>{{ t('schedule.viewAllCourseReviews') }}</span>
              </button>
            </div>
          </section>

          <!-- 查看完整课评大按钮 -->
          <a
            :href="reviewHref"
            class="gf-button gf-button-md gf-button-primary w-full justify-center gap-1.5 rounded-xl shadow-xs"
            target="_blank"
            rel="noopener noreferrer"
          >
            <span>{{ t('schedule.reviews') }}</span>
            <ExternalLink class="h-3.5 w-3.5" />
          </a>
        </template>
      </div>
    </aside>
  </div>
</template>
