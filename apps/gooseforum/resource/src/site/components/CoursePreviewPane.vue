<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import {
  ArrowUpRight,
  Bookmark,
  ChevronDown,
  GraduationCap,
  PenLine,
  Pin,
  Star,
  UsersRound,
  X,
} from '@lucide/vue'
import AISummaryCard from '@/site/components/AISummaryCard.vue'
import { bookmarkCourse, getCourseRelated, listCourseReviews, type CourseRelatedResult, type ReviewPage, type ReviewPayload } from '@/runtime/api'
import { reviewAvatarSrc } from '@/site/utils/course-review-share'
import { useFlashMessages } from '@/runtime/flash-message'
import { formatDateTime } from '@/runtime/format'
import { shortTerm, sortedRecentTerms } from '@/site/utils/term'
import type { CourseSummaryPayload } from '@gooseforum/client'

const props = defineProps<{
  course: CourseSummaryPayload | null
  isAuthenticated: boolean
  bookmarkedCourseIds: number[]
}>()

const emit = defineEmits<{
  close: []
  'bookmark-toggle': []
  'select-course': [courseId: number]
}>()

const { t } = useI18n()
const { push: pushFlash } = useFlashMessages()

// 课程预览：点击课程后异步加载相关课程与评价（AI 总结由 AISummaryCard 自行懒加载）。
const related = ref<CourseRelatedResult | null>(null)
const reviews = ref<ReviewPage | null>(null)
const loadingRelated = ref(false)
const loadingReviews = ref(false)

// —— 收纳：会话级暂存浏览过的课程（替代被移除的「收起 mini 视图」）——
// 新收纳项置顶（最近收纳在前），支持单项移除与「清除除本页外」；sessionStorage 持久化。
type PinnedCourse = {
  id: number
  name: string
  primaryCode: string
  teacher: string
}
const PINNED_STORAGE_KEY = 'goose:coursePreviewPinned'
const pinnedCourses = ref<PinnedCourse[]>([])
const pinButtonEl = ref<HTMLButtonElement | null>(null)
// 收纳仓展开态：默认折叠（仅占一行），常驻标题区下方，展开后列表限高内部滚动。
const pinnedOpen = ref(false)

const isPinned = computed(() =>
  props.course != null && pinnedCourses.value.some((item) => item.id === props.course?.id),
)

function persistPinned() {
  try {
    sessionStorage.setItem(PINNED_STORAGE_KEY, JSON.stringify(pinnedCourses.value))
  } catch {
    // sessionStorage 不可用（隐私模式/容量）时静默降级：仅当前会话内有效。
  }
}

function restorePinned() {
  try {
    const raw = sessionStorage.getItem(PINNED_STORAGE_KEY)
    if (!raw) return
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return
    pinnedCourses.value = parsed.filter(
      (item): item is PinnedCourse =>
        typeof item === 'object' &&
        item != null &&
        typeof (item as PinnedCourse).id === 'number' &&
        typeof (item as PinnedCourse).name === 'string',
    )
  } catch {
    pinnedCourses.value = []
  }
}

function togglePin() {
  if (!props.course) return
  if (isPinned.value) {
    const currentId = props.course.id
    pinnedCourses.value = pinnedCourses.value.filter((item) => item.id !== currentId)
  } else {
    const snapshot: PinnedCourse = {
      id: props.course.id,
      name: props.course.name,
      primaryCode: props.course.primaryCode,
      teacher: props.course.teacherName || props.course.instructors?.join('、') || t('coursesPage.noTeacher'),
    }
    pinnedCourses.value = [snapshot, ...pinnedCourses.value.filter((item) => item.id !== snapshot.id)]
  }
  persistPinned()
  void nextTick(() => pinButtonEl.value?.focus())
}

// 移除单个收纳项后，焦点回退到相邻项或「收纳」按钮，保持键盘用户上下文。
function removePinned(courseId: number, event: MouseEvent | KeyboardEvent) {
  const itemEl = (event.currentTarget as HTMLElement).closest('li')
  const neighbour = itemEl?.nextElementSibling ?? itemEl?.previousElementSibling
  pinnedCourses.value = pinnedCourses.value.filter((item) => item.id !== courseId)
  persistPinned()
  void nextTick(() => {
    const target = neighbour?.querySelector<HTMLButtonElement>('button') ?? pinButtonEl.value
    target?.focus()
  })
}

// 清除除当前预览课程外的全部收纳项；当前未收纳时等价于整体清空。
function clearPinnedOthers() {
  const currentId = props.course?.id
  pinnedCourses.value =
    currentId == null ? [] : pinnedCourses.value.filter((item) => item.id === currentId)
  persistPinned()
  void nextTick(() => pinButtonEl.value?.focus())
}

// 可清除的「其他项」数量：为 0 时禁用批量清除入口。
const clearablePinnedCount = computed(() => {
  const currentId = props.course?.id
  if (currentId == null) return pinnedCourses.value.length
  return pinnedCourses.value.filter((item) => item.id !== currentId).length
})

function selectPinned(courseId: number) {
  emit('select-course', courseId)
}

// 移动端(<lg)把预览面板作为 modal dialog（底部抽屉）处理：
// 打开时锁背景滚动、焦点移入面板，Esc/backdrop 可关闭；桌面端保持 complementary 非模态不变。
const isMobile = useMediaQuery('(max-width: 1023.98px)')
const panelEl = ref<HTMLElement | null>(null)

watch(
  [() => props.course?.id, isMobile],
  ([courseId, mobile]) => {
    if (typeof document === 'undefined') return
    if (courseId != null && mobile) {
      document.body.style.overflow = 'hidden'
      void nextTick(() => {
        panelEl.value?.querySelector<HTMLElement>('button')?.focus()
      })
    } else {
      document.body.style.overflow = ''
    }
  },
  { immediate: true },
)

// <lg 焦点陷阱：Tab 循环限制在面板内（桌面端非模态，不做陷阱）。
function onPanelKeydown(event: KeyboardEvent) {
  if (!isMobile.value || !props.course || !panelEl.value) return
  if (event.key !== 'Tab') return
  const focusables = Array.from(
    panelEl.value.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  ).filter((el) => el.offsetParent !== null)
  if (focusables.length === 0) return
  const first = focusables[0]
  const last = focusables[focusables.length - 1]
  const active = document.activeElement
  if (event.shiftKey) {
    if (active === first || !panelEl.value.contains(active)) {
      event.preventDefault()
      last.focus()
    }
  } else if (active === last || !panelEl.value.contains(active)) {
    event.preventDefault()
    first.focus()
  }
}

// Esc 关闭（仅移动端 dialog 语义时生效；桌面端行为不变）。
function onDocumentKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  if (props.course && isMobile.value) emit('close')
}

onMounted(() => {
  document.addEventListener('keydown', onDocumentKeydown)
  restorePinned()
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onDocumentKeydown)
  document.body.style.overflow = ''
})

// 预览加载代际：切换课程时递增，使 in-flight 的旧课程响应失效——否则快速点选
// A→B 后 A 的晚到响应会覆盖 B 的相关课程/评价（对齐详情页 reviewsLoadSeq 守卫）。
let previewLoadSeq = 0

watch(
  () => props.course?.id,
  (courseId) => {
    previewLoadSeq += 1
    related.value = null
    reviews.value = null
    if (!courseId) return
    void loadRelated(courseId)
    void loadReviews(courseId)
  },
  { immediate: true },
)

async function loadRelated(courseId: number) {
  const seq = previewLoadSeq
  loadingRelated.value = true
  try {
    const result = await getCourseRelated(courseId)
    // 代际守卫：期间已切换课程，本结果作废，不覆盖新课程状态。
    if (seq !== previewLoadSeq) return
    related.value = result
  } catch {
    if (seq !== previewLoadSeq) return
    related.value = null
  } finally {
    // 仅当前代际可动 loading：旧代际完成时新代际仍在途，保留其 loading 状态。
    if (seq === previewLoadSeq) loadingRelated.value = false
  }
}

async function loadReviews(courseId: number) {
  const seq = previewLoadSeq
  loadingReviews.value = true
  try {
    const result = await listCourseReviews(courseId, 0, '', 3)
    if (seq !== previewLoadSeq) return
    reviews.value = result
  } catch {
    if (seq !== previewLoadSeq) return
    reviews.value = null
  } finally {
    if (seq === previewLoadSeq) loadingReviews.value = false
  }
}

const loginHref = computed(() => {
  const next = encodeURIComponent(window.location.pathname + window.location.search + window.location.hash)
  return `/login?next=${next}`
})

function formatCredit(creditX10: number) {
  return (creditX10 / 10).toFixed(1).replace(/\.0$/, '')
}

// 作者展示名：member 用服务端回填的用户名；anonymous/legacy 走本地 i18n，
// 与详情页/分享卡保持一致（避免非中文界面原样渲染硬编码中文标签）。
function authorLabel(author: ReviewPayload['author']) {
  if (author.kind === 'member') return author.label
  if (author.kind === 'legacy') return t('courseDetailPage.authorLegacy')
  return t('courseDetailPage.authorAnonymous')
}

// 学期标签：最近学期 + 其余折叠（对齐参考稿「最近 26春 +3」），用 shortTerm 缩写。
// 标准学期码按时间降序，非标准码（「其他」）恒置末尾；「最近」取排序后首个。
const sortedTerms = computed(() => sortedRecentTerms(props.course?.recentTerms))
const recentTerm = computed(() => {
  const term = sortedTerms.value[0] ?? ''
  return term ? shortTerm(term) : ''
})
const otherTerms = computed(() => sortedTerms.value.slice(1).map(shortTerm))
const showOtherTerms = ref(false)

const isBookmarked = computed(() =>
  props.course != null && props.bookmarkedCourseIds.includes(props.course.id),
)

async function toggleBookmark() {
  if (!props.course) return
  if (!props.isAuthenticated) {
    window.location.href = loginHref.value
    return
  }
  try {
    await bookmarkCourse(props.course.id, isBookmarked.value ? 2 : 1)
    emit('bookmark-toggle')
  } catch {
    pushFlash(t('api.bookmarkFailed'), 'error')
  }
}

function courseStars(ratingAvg: number | undefined) {
  if (ratingAvg == null) return '-'
  return ratingAvg.toFixed(1)
}
</script>

<template>
  <!-- 移动端(<lg)底部抽屉的半透明 backdrop：点击关闭，淡入淡出。 -->
  <Transition name="fade">
    <div
      v-if="course"
      class="fixed inset-0 z-30 bg-black/30 lg:hidden"
      aria-hidden="true"
      @click="emit('close')"
    />
  </Transition>

  <Transition name="preview">
  <aside
    v-if="course"
    ref="panelEl"
    tabindex="-1"
    :role="isMobile ? 'dialog' : 'complementary'"
    :aria-modal="isMobile || undefined"
    :aria-label="course.name"
    class="fixed inset-x-0 bottom-0 z-40 flex max-h-[85dvh] flex-col overflow-clip rounded-t-2xl border border-line/70 bg-base-100 shadow-xl lg:inset-x-auto lg:left-auto lg:right-4 lg:top-16 lg:bottom-4 lg:max-h-none lg:w-[380px] lg:rounded-2xl"
    @keydown="onPanelKeydown"
  >
    <!-- 移动端拖拽把手（纯视觉+点按关闭辅助，桌面端隐藏） -->
    <div class="mx-auto mt-2 h-1 w-10 shrink-0 rounded-full bg-base-content/20 lg:hidden" @click="emit('close')" />
    <!-- 顶栏：左=进入课程主页（导航），右=收起/展开 + 退出预览（面板操作，集中靠右） -->
    <div class="flex items-center justify-between gap-2 border-b border-line/70 px-4 py-2.5">
      <a
        :href="`/courses/${course.id}`"
        class="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-lg border border-primary/25 bg-info/10 px-3 text-xs font-semibold text-primary shadow-sm transition hover:border-primary/45 hover:bg-info/15 active:scale-[0.97] focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
      >
        <ArrowUpRight class="h-4 w-4 shrink-0" />
        <span class="truncate">{{ t('coursesPage.enterCourse') }}</span>
      </a>
      <div class="flex shrink-0 items-center gap-1.5">
        <button
          ref="pinButtonEl"
          type="button"
          class="inline-flex h-9 items-center gap-1 rounded-lg border px-2.5 text-xs font-medium shadow-sm transition hover:border-primary/40 active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
          :class="isPinned ? 'border-primary/45 bg-primary/10 text-primary' : 'border-line bg-base-100 text-base-content/70 hover:text-primary'"
          :aria-pressed="isPinned"
          :aria-label="t('coursesPage.pinCourseLabel')"
          :title="t('coursesPage.pinCourseLabel')"
          @click="togglePin"
        >
          <Pin class="h-4 w-4" :class="{ 'fill-primary/25': isPinned }" />
          <span>{{ t('coursesPage.pinCourse') }}</span>
        </button>
        <button
          type="button"
          class="inline-flex h-9 items-center gap-1 rounded-lg px-2 text-xs font-medium text-base-content/50 transition hover:bg-base-200 hover:text-base-content active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
          :aria-label="t('coursesPage.exitPreview')"
          @click="emit('close')"
        >
          <X class="h-4 w-4" />
          <span>{{ t('coursesPage.exitPreview') }}</span>
        </button>
      </div>
    </div>

    <!-- 内容区（标题 + 滚动）：切换课程时以 course.id 为 key 整体淡入淡出，
         旧内容先轻淡出、新内容淡入并轻微上浮；顶栏与底部收藏保持稳定。 -->
    <Transition name="course-swap" mode="out-in">
    <div :key="course.id" class="flex min-h-0 flex-1 flex-col">
    <!-- 标题区：常驻完整展示（课程码 + 名称 + 教师/评分 + 统计条 + 学期标签） -->
    <div class="border-b border-line/70 px-4 py-3">
      <div
        class="mb-2 inline-flex items-center gap-1.5 rounded-full border border-info/30 bg-info/10 px-3 py-1 text-xs font-medium text-info"
      >
        <span class="h-1.5 w-1.5 rounded-full bg-info" />
        {{ course.primaryCode }}
      </div>
      <h2 class="truncate text-xl font-bold text-base-content">
        {{ course.name }}
      </h2>
      <div class="mt-1 text-sm">
        <div class="flex items-center gap-2">
          <span class="font-medium text-base-content/75">{{ course.teacherName || course.instructors?.join('、') || t('coursesPage.noTeacher') }}</span>
          <span v-if="course.ratingAvg != null" class="inline-flex items-center gap-1 text-amber-500">
            <Star class="h-3.5 w-3.5 fill-warning text-warning" />
            <span class="tabular-nums font-bold text-base-content">{{ courseStars(course.ratingAvg) }}</span>
          </span>
        </div>
      </div>

      <div>
        <p class="mt-1 text-sm text-base-content/55">{{ course.department }}</p>

        <!-- 统计条：教师 / 综合评分 / 学分（一行三列紧凑呈现，替代原纵向大卡） -->
        <div class="mt-3 grid grid-cols-3 divide-x divide-line/60 rounded-xl border border-line/70 bg-base-200/40">
          <div class="min-w-0 px-2 py-2.5 text-center">
            <p class="flex items-center justify-center gap-1 text-[11px] text-base-content/50">
              <UsersRound class="h-3.5 w-3.5 shrink-0" />
              {{ t('coursesPage.teacher') }}
            </p>
            <p
              class="mt-1 truncate text-xs font-bold text-base-content"
              :title="course.teacherName || course.instructors?.join('、') || t('coursesPage.noTeacher')"
            >
              {{ course.teacherName || course.instructors?.join('、') || t('coursesPage.noTeacher') }}
            </p>
          </div>
          <div class="min-w-0 px-2 py-2.5 text-center">
            <p class="flex items-center justify-center gap-1 text-[11px] text-base-content/50">
              <Star class="h-3.5 w-3.5 shrink-0 fill-warning/50 text-warning/50" />
              {{ t('coursesPage.columnRating') }}
            </p>
            <p class="mt-1 text-xs font-bold tabular-nums text-base-content">
              {{ course.ratingAvg != null ? course.ratingAvg.toFixed(1) : '—' }} / 5.0
            </p>
          </div>
          <div class="min-w-0 px-2 py-2.5 text-center">
            <p class="flex items-center justify-center gap-1 text-[11px] text-base-content/50">
              <GraduationCap class="h-3.5 w-3.5 shrink-0" />
              {{ t('coursesPage.columnCredit') }}
            </p>
            <p class="mt-1 text-xs font-bold tabular-nums text-base-content">
              {{ formatCredit(course.creditX10) }} {{ t('coursesPage.creditUnit') }}
            </p>
          </div>
        </div>

        <!-- 学期标签：最近 + 其余折叠 -->
        <div v-if="recentTerm" class="mt-3 flex flex-wrap items-center gap-2">
          <span class="inline-flex items-center rounded-full border border-info/30 bg-info/10 px-3 py-1 text-xs font-medium text-info">
            {{ t('coursesPage.recent') }} {{ recentTerm }}
          </span>
          <button
            v-if="otherTerms.length"
            type="button"
            class="rounded-full border border-line bg-base-200/60 px-3 py-1 text-xs font-bold text-base-content/70 hover:bg-base-200"
            @click="showOtherTerms = !showOtherTerms"
          >
            + {{ otherTerms.length }}
          </button>
        </div>
        <div v-if="showOtherTerms" class="mt-2 flex flex-wrap gap-2">
          <span v-for="term in otherTerms" :key="term" class="rounded-full border border-line/70 bg-base-200/50 px-3 py-1 text-xs text-base-content/60">
            {{ term }}
          </span>
        </div>
      </div>
    </div>

    <!-- 收纳仓（二级导航层）：常驻折叠条位于「身份信息」与「决策内容」之间，
         默认仅一行不挤占滚动区；展开时列表限高内部滚动，可快速回看/切换，始终可见。 -->
    <section v-if="pinnedCourses.length" class="border-b border-line/70">
      <div class="flex items-center justify-between gap-2 px-4 py-1.5">
        <button
          type="button"
          class="flex min-w-0 flex-1 items-center gap-1.5 py-1 text-left text-xs font-bold text-base-content/75 transition hover:text-primary focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
          :aria-expanded="pinnedOpen"
          aria-controls="pinned-courses-panel"
          @click="pinnedOpen = !pinnedOpen"
        >
          <Pin class="h-3.5 w-3.5 shrink-0" />
          <span class="truncate">{{ t('coursesPage.pinnedCourses') }}</span>
          <span class="rounded-full bg-base-200 px-1.5 py-0.5 text-[11px] font-medium tabular-nums text-base-content/60">
            {{ pinnedCourses.length }}
          </span>
          <ChevronDown
            class="h-3.5 w-3.5 shrink-0 transition-transform motion-safe:duration-200"
            :class="{ 'rotate-180': pinnedOpen }"
            aria-hidden="true"
          />
        </button>
        <button
          type="button"
          class="shrink-0 px-1 py-1 text-[11px] font-medium text-base-content/45 transition hover:text-primary disabled:cursor-not-allowed disabled:opacity-40 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
          :disabled="clearablePinnedCount === 0"
          @click="clearPinnedOthers"
        >
          {{ t('coursesPage.clearPinnedOthers') }}
        </button>
      </div>

      <Transition name="pinned-panel">
        <ul
          v-if="pinnedOpen"
          id="pinned-courses-panel"
          tabindex="0"
          aria-label="已收纳课程"
          class="max-h-56 overflow-y-auto overscroll-contain divide-y divide-line/60 border-t border-line/70 bg-base-100"
        >
          <li v-for="item in pinnedCourses" :key="item.id" class="flex items-center gap-1 px-4 py-2">
            <button
              type="button"
              class="min-w-0 flex-1 rounded-lg px-1 py-1 text-left transition hover:bg-base-200/60 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
              :aria-current="item.id === course.id ? 'true' : undefined"
              @click="selectPinned(item.id)"
            >
              <span class="block truncate text-sm font-semibold text-base-content">{{ item.name }}</span>
              <span class="block truncate text-xs text-base-content/55">{{ item.primaryCode }} · {{ item.teacher }}</span>
            </button>
            <button
              type="button"
              class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-base-content/40 transition hover:bg-base-200 hover:text-base-content active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
              :aria-label="`${t('coursesPage.removePinned')} ${item.name}`"
              @click="removePinned(item.id, $event)"
            >
              <X class="h-4 w-4" />
            </button>
          </li>
        </ul>
      </Transition>
    </section>

    <!-- 可滚动内容区 -->
    <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain bg-base-200/30 px-4 py-4">
      <!-- 撰写评价（跳详情页并直达写评表单，置于滚动区顶部作为主动作） -->
      <a
        :href="`/courses/${course.id}?writeReview=1`"
        class="mb-6 flex w-full items-center justify-center gap-2 rounded-xl bg-neutral py-3 text-sm font-medium text-neutral-content shadow-sm transition hover:bg-neutral/90"
      >
        <PenLine class="h-4 w-4" />
        {{ t('coursesPage.writeReview') }}
      </a>

      <!-- AI 评课总结（复用完整卡片，默认折叠、展开才懒加载） -->
      <AISummaryCard :course-id="course.id" class="mb-6" />

      <!-- 相关课程/评价加载中：骨架占位，平滑衔接顶部内容（纯装饰，aria-hidden） -->
      <div v-if="loadingRelated || loadingReviews" aria-hidden="true" class="mb-6 space-y-3">
        <div class="animate-pulse space-y-2 rounded-xl border border-line/70 bg-base-100 p-4">
          <div class="h-3.5 w-1/3 rounded bg-base-200/80" />
          <div class="h-3 w-full rounded bg-base-200/60" />
          <div class="h-3 w-2/3 rounded bg-base-200/60" />
        </div>
        <div class="animate-pulse space-y-2 rounded-xl border border-line/70 bg-base-100 p-4">
          <div class="h-3.5 w-1/4 rounded bg-base-200/80" />
          <div class="h-3 w-full rounded bg-base-200/60" />
          <div class="h-3 w-full rounded bg-base-200/60" />
        </div>
      </div>

      <!-- 该老师的其他课程 -->
      <section v-if="related" class="mb-6">
        <div class="mb-3 flex items-center gap-2">
          <span class="h-1.5 w-1.5 rounded-full bg-info" />
          <h3 class="text-sm font-bold text-base-content">{{ t('coursesPage.teacherOtherCourses') }}</h3>
        </div>
        <div v-if="related.teacherOtherCourses.length" class="divide-y divide-line/60 rounded-xl border border-line/70 bg-base-100">
          <a
            v-for="item in related.teacherOtherCourses"
            :key="item.id"
            :href="`/courses/${item.id}`"
            class="group flex items-center justify-between p-4 transition hover:bg-base-200/50"
          >
            <div class="min-w-0">
              <h4 class="truncate text-sm font-bold text-base-content group-hover:text-primary">{{ item.name }}</h4>
              <p class="text-xs text-base-content/50">{{ item.primaryCode }}</p>
            </div>
            <div class="shrink-0 text-right">
              <p class="text-sm font-bold text-amber-500">{{ item.ratingAvg > 0 ? item.ratingAvg.toFixed(1) : '—' }}</p>
              <p class="text-xs tabular-nums text-base-content/50">{{ item.reviewCount }} {{ t('coursesPage.columnReviewCount') }}</p>
            </div>
          </a>
        </div>
        <p v-else class="text-xs text-base-content/50">{{ t('coursesPage.relatedEmpty') }}</p>
      </section>

      <!-- 该课程的其他老师 -->
      <section v-if="related" class="mb-6">
        <div class="mb-3 flex items-center gap-2">
          <span class="h-1.5 w-1.5 rounded-full bg-amber-500" />
          <h3 class="text-sm font-bold text-base-content">{{ t('coursesPage.otherTeachers') }}</h3>
        </div>
        <div v-if="related.sameCourseOtherTeachers.length" class="divide-y divide-line/60 rounded-xl border border-line/70 bg-base-100">
          <a
            v-for="item in related.sameCourseOtherTeachers"
            :key="item.id"
            :href="`/courses/${item.id}`"
            class="group flex items-center justify-between p-4 transition hover:bg-base-200/50"
          >
            <div class="min-w-0">
              <h4 class="truncate text-sm font-bold text-base-content group-hover:text-primary">{{ item.instructors?.join('、') || t('coursesPage.noTeacher') }}</h4>
              <p class="text-xs text-base-content/50">{{ item.name }}</p>
            </div>
            <div class="shrink-0 text-right">
              <p class="text-sm font-bold text-amber-500">{{ item.ratingAvg > 0 ? item.ratingAvg.toFixed(1) : '—' }}</p>
              <p class="text-xs tabular-nums text-base-content/50">{{ item.reviewCount }} {{ t('coursesPage.columnReviewCount') }}</p>
            </div>
          </a>
        </div>
        <p v-else class="text-xs text-base-content/50">{{ t('coursesPage.relatedEmpty') }}</p>
      </section>

      <!-- 用户评价 -->
      <section v-if="reviews" class="mb-6">
        <div class="mb-3 flex items-center justify-between">
          <h3 class="text-sm font-bold text-base-content">{{ t('coursesPage.reviews') }}</h3>
        </div>
        <div v-if="reviews.list.length" class="space-y-3">
          <div v-for="review in reviews.list" :key="review.id" class="rounded-xl border border-line/70 bg-base-100 p-4">
            <div class="mb-2 flex items-start justify-between gap-2">
              <div class="flex min-w-0 items-center gap-2.5">
                <img
                  :src="reviewAvatarSrc(review.author, review.id, 36)"
                  :alt="authorLabel(review.author)"
                  class="h-9 w-9 shrink-0 rounded-full object-cover ring-1 ring-line/60"
                  loading="lazy"
                />
                <div class="min-w-0">
                  <p class="truncate text-sm font-bold text-base-content">{{ authorLabel(review.author) }}</p>
                  <p class="text-xs text-base-content/50">{{ formatDateTime(review.createdAt) }}</p>
                </div>
              </div>
              <div v-if="review.rating" class="flex shrink-0 gap-0.5 text-amber-400">
                <Star v-for="n in review.rating" :key="n" class="h-3.5 w-3.5 fill-current" />
              </div>
            </div>
            <div
              v-if="review.contentHtml"
              v-code-highlight
              v-math-render
              class="gf-prose gf-prose-post gf-prose-preview mt-2 text-base-content/80"
              v-html="review.contentHtml"
            />
            <p v-else class="text-sm leading-relaxed text-base-content/80">{{ review.content }}</p>
          </div>
        </div>
        <p v-else class="text-xs text-base-content/50">{{ t('coursesPage.reviewsEmpty') }}</p>
      </section>
    </div>
    </div>
    </Transition>

    <!-- 底部：收藏 -->
    <div class="flex items-center justify-between gap-2 border-t border-line/70 bg-base-100 px-4 py-3">
      <button
        type="button"
        class="inline-flex items-center gap-2 rounded-lg border border-line px-4 py-2 text-sm font-medium text-base-content/75 transition hover:border-primary hover:text-primary"
        @click="toggleBookmark"
      >
        <Bookmark class="h-4 w-4" :class="{ 'fill-primary text-primary': isBookmarked }" />
        {{ isBookmarked ? t('api.bookmarked') : t('coursesPage.bookmarkCourse') }}
      </button>
      <span class="text-xs text-base-content/50">{{ course.reviewCount ?? 0 }} {{ t('coursesPage.columnReviewCount') }}</span>
    </div>
  </aside>
  </Transition>
</template>

<style scoped>
/* 移动端 backdrop：仅 opacity 淡入淡出。 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* 内容区跨课程切换：旧内容先快速轻淡出（0.14s），新内容淡入并轻微上浮（0.22s）。
   面板框架（顶栏/底部收藏）保持稳定；动画只在 opacity/transform 上，性能友好。 */
.course-swap-enter-active {
  transition:
    opacity 0.22s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.22s cubic-bezier(0.16, 1, 0.3, 1);
}
.course-swap-leave-active {
  transition:
    opacity 0.14s ease,
    transform 0.14s ease;
}
.course-swap-enter-from {
  opacity: 0;
  transform: translateY(6px);
}
.course-swap-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

/* 预览面板进入/退出：仅动画 transform+opacity。
   进入用 expo-out（快进慢出 + 轻微缩放回弹），退出稍快且收敛，避免生硬。 */
.preview-enter-active {
  transition:
    opacity 0.32s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.32s cubic-bezier(0.16, 1, 0.3, 1);
}
.preview-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s cubic-bezier(0.4, 0, 1, 1);
}
.preview-enter-from {
  opacity: 0;
  transform: translateX(44px) scale(0.98);
}
.preview-leave-to {
  opacity: 0;
  transform: translateX(36px);
}
/* 移动端(<lg)：底部抽屉改为上滑进入/下滑退出；桌面端仍保持右滑（上方基础规则）。 */
@media (max-width: 1023.98px) {
  .preview-enter-from,
  .preview-leave-to {
    transform: translateY(100%);
  }
}
@media (prefers-reduced-motion: reduce) {
  .preview-enter-active,
  .preview-leave-active {
    transition: opacity 0.16s ease;
  }
  .preview-enter-from,
  .preview-leave-to {
    transform: none;
  }
  .course-swap-enter-active,
  .course-swap-leave-active {
    transition: opacity 0.12s ease;
  }
  .course-swap-enter-from,
  .course-swap-leave-to {
    transform: none;
  }
}

/* 收纳仓展开/收起：仅 opacity + 轻微位移，与面板整体动效一致；
   条目本身不再单独做交错（列表限高内部滚动，避免层级过多）。 */
.pinned-panel-enter-active {
  transition:
    opacity 0.18s ease,
    transform 0.18s cubic-bezier(0.16, 1, 0.3, 1);
}
.pinned-panel-enter-from {
  opacity: 0;
  transform: translateY(-4px);
}
.pinned-panel-leave-active {
  transition:
    opacity 0.12s ease,
    transform 0.12s ease;
}
.pinned-panel-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
@media (prefers-reduced-motion: reduce) {
  .pinned-panel-enter-active,
  .pinned-panel-leave-active {
    transition: opacity 0.1s ease;
  }
  .pinned-panel-enter-from,
  .pinned-panel-leave-to {
    transform: none;
  }
}

/* 预览面板内的 Markdown 排版：窄栏缩放标题、舒适正文、紧凑间距，
   避免全局 gf-prose-post 的大标题在 380px 面板里显得突兀。 */
.gf-prose-preview {
  font-size: 15px;
  line-height: 1.65;
}
.gf-prose-preview > :first-child {
  margin-top: 0;
}
.gf-prose-preview > :last-child {
  margin-bottom: 0;
}
.gf-prose-preview p {
  margin: 0.375rem 0;
  line-height: 1.65;
}
.gf-prose-preview h1,
.gf-prose-preview h2,
.gf-prose-preview h3,
.gf-prose-preview h4 {
  font-weight: 600;
  line-height: 1.3;
  color: var(--gf-color-base-content);
}
.gf-prose-preview h1 {
  margin: 0.85rem 0 0.35rem;
  font-size: 1.25em;
}
.gf-prose-preview h2 {
  margin: 0.8rem 0 0.35rem;
  font-size: 1.15em;
}
.gf-prose-preview h3 {
  margin: 0.7rem 0 0.3rem;
  font-size: 1.08em;
}
.gf-prose-preview h4 {
  margin: 0.6rem 0 0.25rem;
  font-size: 1em;
}
.gf-prose-preview h5,
.gf-prose-preview h6 {
  margin: 0.5rem 0 0.25rem;
  font-size: 0.95em;
}
.gf-prose-preview ul,
.gf-prose-preview ol {
  margin: 0.375rem 0;
  padding-left: 1.25rem;
}
.gf-prose-preview li {
  margin: 0.2rem 0;
}
.gf-prose-preview blockquote {
  margin: 0.5rem 0;
  padding: 0.4rem 0.85rem;
  font-size: 0.95em;
}
.gf-prose-preview pre {
  margin: 0.5rem 0;
  padding: 0.6rem 0.85rem;
  font-size: 0.85em;
}
.gf-prose-preview table {
  font-size: 12px;
}
.gf-prose-preview code {
  font-size: 0.9em;
}
</style>
