<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Bookmark,
  BookOpen,
  Building2,
  CalendarDays,
  ChevronDown,
  Compass,
  Eraser,
  Eye,
  Lightbulb,
  MapPin,
  Search,
  Share2,
  SlidersHorizontal,
  Star,
  UsersRound,
  X,
} from '@lucide/vue'
import CourseCatalogMultiSelect from '@/site/components/CourseCatalogMultiSelect.vue'
import CoursePreviewPane from '@/site/components/CoursePreviewPane.vue'
import EmptyState from '@/site/components/EmptyState.vue'
import InfiniteScrollFooter from '@/site/components/InfiniteScrollFooter.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import { bookmarkCourse, listCourses } from '@/runtime/api'
import { useFlashMessages } from '@/runtime/flash-message'
import { mergeCourses } from '@/site/utils/course-merge'
import { rememberCourseCatalogUrl } from '@/site/utils/course-catalog-return'
import { shortTerm, sortedRecentTerms } from '@/site/utils/term'
import type { CourseCatalogPageProps, CourseSummaryPayload, LayoutPayload } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: CourseCatalogPageProps
  pageUrl: string
}>()

const { t } = useI18n()
const { push: pushFlash } = useFlashMessages()

const viewer = computed(() => page.layout.viewer)
const isAuthenticated = computed(() => viewer.value.isAuthenticated)

// URL 中可能残留空值（如空 instructor=、空的 department=），统一去除，
// 避免生成空筛选项、或在计数/激活态上误判。
function cleanMulti(values: string[] | undefined): string[] {
  return (values ?? []).map((v) => v.trim()).filter(Boolean)
}

const query = computed(() => {
  const q = page.props.query
  return {
    ...q,
    department: cleanMulti(q.department),
    term: cleanMulti(q.term),
    campus: cleanMulti(q.campus),
    instructor: cleanMulti(q.instructor),
  }
})

const hasActiveFilters = computed(() => {
  const q = query.value
  return Boolean(
    q.keyword?.trim() ||
      (q.department?.length ?? 0) ||
      (q.term?.length ?? 0) ||
      (q.campus?.length ?? 0) ||
      (q.instructor?.length ?? 0) ||
      q.onlyWithReviews ||
      q.sortBy,
  )
})

// 表格区「更多筛选」按钮的激活计数 badge。
const activeFilterCount = computed(() => {
  const q = query.value
  let count = 0
  if (q.department?.length) count++
  if (q.term?.length) count++
  if (q.campus?.length) count++
  if (q.instructor?.length) count++
  if (q.onlyWithReviews) count++
  if (q.sortBy) count++
  return count
})

const moreFiltersOpen = ref(false)
// 排序是单值的两项选择，用分段切换器表达（与自建多选控件同族）；
// 选中值经隐藏 input 随「应用筛选」提交，与其余字段保持"先选后应用"的一致语义。
const draftSortBy = ref(page.props.query.sortBy ?? '')

// 顶部筛选栏单行回显「更多筛选」中已选中的条件；未选中时不占空间。
type FilterChip = { key: string; value: string; label: string }
const selectedFilterChips = computed<FilterChip[]>(() => {
  const q = query.value
  const chips: FilterChip[] = []
  for (const dep of q.department ?? []) chips.push({ key: 'department', value: dep, label: dep })
  for (const term of q.term ?? []) {
    const termOption = page.props.terms.find((item) => item.value === term)
    chips.push({ key: 'term', value: term, label: termOption?.label ?? term })
  }
  for (const campus of q.campus ?? []) chips.push({ key: 'campus', value: campus, label: campus })
  for (const instructor of q.instructor ?? []) chips.push({ key: 'instructor', value: instructor, label: instructor })
  if (q.onlyWithReviews) chips.push({ key: 'onlyWithReviews', value: '1', label: t('coursesPage.onlyWithReviews') })
  if (q.sortBy === 'rating') chips.push({ key: 'sortBy', value: 'rating', label: t('coursesPage.sortByRating') })
  return chips
})

// 移除某个已选条件（数组维度只去掉这一项，标量维度整体移除）。
function removeChipHref(chip: FilterChip): string {
  const q = query.value
  if (chip.key === 'onlyWithReviews' || chip.key === 'sortBy') {
    return buildQuery({ [chip.key]: undefined })
  }
  const arr = (q as unknown as Record<string, string[] | undefined>)[chip.key] ?? []
  return buildQuery({ [chip.key]: arr.filter((item) => item !== chip.value) })
}

const loginHref = computed(() => {
  const next = encodeURIComponent(window.location.pathname + window.location.search + window.location.hash)
  return `/login?next=${next}`
})

// 快捷标签 / 课程表格「全部 + 各学院」共用的查询串构造。
// department/term/campus/instructor 支持 string[] 多值（序列化为重复参数）；
// 传空数组或 undefined 表示移除该维度。
function buildQuery(overrides: Record<string, string | string[] | undefined>): string {
  // SSR 阶段没有 window，无法读取当前查询串；返回路径占位，href 会在客户端水合后补全。
  if (typeof window === 'undefined') return '/courses'
  const params = new URLSearchParams()
  const current = new URLSearchParams(window.location.search)
  const merged: Record<string, string | string[] | undefined> = {
    keyword: current.get('keyword') ?? undefined,
    department: current.getAll('department'),
    term: current.getAll('term'),
    campus: current.getAll('campus'),
    instructor: current.getAll('instructor'),
    onlyWithReviews: current.get('onlyWithReviews') ?? undefined,
    sortBy: current.get('sortBy') ?? undefined,
    ...overrides,
  }
  for (const [key, value] of Object.entries(merged)) {
    if (Array.isArray(value)) {
      for (const item of value) {
        if (item) params.append(key, item)
      }
    } else if (value != null && value !== '') {
      params.set(key, value)
    }
  }
  const qs = params.toString()
  return qs ? `/courses?${qs}` : '/courses'
}

// 快捷标签激活态判断（多值维度按包含判断）。
function quickActive(override: Record<string, string | undefined>): boolean {
  const q = query.value
  return Object.entries(override).every(([key, value]) => {
    const expected = value ?? ''
    if (key === 'onlyWithReviews') {
      return expected === '1' ? Boolean(q.onlyWithReviews) : !q.onlyWithReviews
    }
    const actual = (q as unknown as Record<string, string | string[] | undefined>)[key]
    if (Array.isArray(actual)) return actual.includes(expected)
    return (actual ?? '') === expected
  })
}

// 「常用筛选」快捷标签（真实派生：评分 / 仅看有评价 / 本学期）。
const quickFilters = computed(() => {
  const items: { key: string; href: string; label: string; active: boolean }[] = []
  items.push({ key: 'rating', href: buildQuery({ sortBy: 'rating' }), label: t('coursesPage.quickHighRated'), active: quickActive({ sortBy: 'rating' }) })
  items.push({ key: 'reviews', href: buildQuery({ onlyWithReviews: '1' }), label: t('coursesPage.quickWithReviews'), active: quickActive({ onlyWithReviews: '1' }) })
  if (page.props.terms.length) {
    items.push({ key: 'term', href: buildQuery({ term: page.props.terms[0].value }), label: t('coursesPage.quickThisTerm'), active: quickActive({ term: page.props.terms[0].value }) })
  }
  return items
})

// —— 搜索框：打字机 placeholder + 清空搜索 ——
// 占位符按常见课程名循环打字；尊重 prefers-reduced-motion（静止显示首个示例）。
const searchExamples = ['高等数学', '线性代数', '大学物理', '概率论与数理统计', '计算机组成原理']
const keywordInput = ref(page.props.query.keyword ?? '')
const keywordInputEl = ref<HTMLInputElement | null>(null)
const placeholderText = ref(searchExamples[0])

let exampleIndex = 0
let charCount = searchExamples[0].length
let phase: 'hold' | 'typing' | 'deleting' = 'hold'
let typeTimer: ReturnType<typeof setTimeout> | undefined

function typeStep() {
  const current = searchExamples[exampleIndex]
  if (phase === 'hold') {
    phase = 'deleting'
    typeTimer = setTimeout(typeStep, 1800)
    return
  }
  if (phase === 'deleting') {
    charCount = Math.max(0, charCount - 1)
    placeholderText.value = current.slice(0, charCount)
    if (charCount === 0) {
      phase = 'typing'
      exampleIndex = (exampleIndex + 1) % searchExamples.length
      typeTimer = setTimeout(typeStep, 400)
    } else {
      typeTimer = setTimeout(typeStep, 70)
    }
    return
  }
  charCount = Math.min(searchExamples[exampleIndex].length, charCount + 1)
  placeholderText.value = searchExamples[exampleIndex].slice(0, charCount)
  if (charCount === searchExamples[exampleIndex].length) {
    phase = 'hold'
    typeTimer = setTimeout(typeStep, 1800)
  } else {
    typeTimer = setTimeout(typeStep, 150)
  }
}

onMounted(() => {
  // 从目录页进入课程详情的任意入口（列表行课程名、预览面板「进入课程主页」）
  // 都在 click 捕获阶段记录当前目录 URL，供详情页「返回课程目录」恢复搜索/筛选态。
  document.addEventListener('click', onCourseDetailClick, true)
  if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return
  typeTimer = setTimeout(typeStep, 1400)
})
onBeforeUnmount(() => {
  clearTimeout(typeTimer)
  document.removeEventListener('click', onCourseDetailClick, true)
})

// 进入课程详情（/courses/{id}）时记录目录返回态；Ctrl/Cmd 新标签打开则无返回诉求，跳过。
function onCourseDetailClick(event: MouseEvent) {
  if (event.metaKey || event.ctrlKey) return
  const target = event.target as Element | null
  const anchor = target?.closest?.('a[href^="/courses/"]')
  if (!anchor) return
  const href = (anchor as HTMLAnchorElement).getAttribute('href') ?? ''
  if (!/^\/courses\/\d+/.test(href)) return
  rememberCourseCatalogUrl(window.location.href)
}

function clearSearch() {
  const wasSearching = Boolean(query.value.keyword?.trim())
  keywordInput.value = ''
  if (wasSearching) {
    // 已提交过关键词：清空并回到未过滤状态（与页面现有的整页导航一致）。
    window.location.href = buildQuery({ keyword: undefined })
  } else {
    nextTick(() => keywordInputEl.value?.focus())
  }
}

// 「开课学期」列收纳：仅显示最新学期，简写 26春/26秋；其余折叠为 +n 展开。
const courseTermList = computed<Record<number, string[]>>(() => {
  const map: Record<number, string[]> = {}
  for (const c of courses.value) map[c.id] = sortedRecentTerms(c.recentTerms)
  return map
})
const expandedTermRows = ref<Set<number>>(new Set())
function toggleTermExpand(courseId: number) {
  if (expandedTermRows.value.has(courseId)) expandedTermRows.value.delete(courseId)
  else expandedTermRows.value.add(courseId)
}

// 课程预览侧边栏：默认不展开（搜索/筛选为整页导航，会重新挂载导致自动选第一条，
// 因此保持 null）；仅在用户刻意点击课程行时出现并切换。
const previewCourseId = ref<number | null>(null)
const previewCourse = computed<CourseSummaryPayload | null>(() => {
  if (previewCourseId.value == null) return null
  return courses.value.find((c) => c.id === previewCourseId.value) ?? null
})

// 打开预览的触发元素（表格行 / 移动端卡片）：关闭预览后焦点归还，
// 保证键盘与读屏用户不会因面板出现而丢失上下文。
const previewTriggerEl = ref<HTMLElement | null>(null)

function openPreview(courseId: number, event?: MouseEvent | KeyboardEvent) {
  previewCourseId.value = courseId
  previewTriggerEl.value = (event?.currentTarget as HTMLElement | null) ?? null
}

// 收纳项切换：课程仍在当前列表则直接切预览；已被筛选移除则进入课程详情页兜底。
function onPreviewSelectCourse(courseId: number) {
  if (courses.value.some((course) => course.id === courseId)) {
    openPreview(courseId)
  } else {
    window.location.href = `/courses/${courseId}`
  }
}

// 整行/整卡可点容易误触：点击行/卡内的交互控件（课程名链接、收藏按钮、学期 +n、评分等）
// 只触发该控件本身，不因冒泡而打开预览面板；卡片本体带 role="button"，需要排除自身命中。
function onCourseRowClick(courseId: number, event: MouseEvent) {
  const target = event.target as HTMLElement
  const trigger = event.currentTarget as Element | null
  const interactive = target.closest('a, button, input, select, label, [role="button"]')
  if (interactive && interactive !== trigger) return
  openPreview(courseId, event)
}

// 移动端卡片键控激活（与 role="button" 语义配套）；仅卡片本体聚焦时响应，
// 避免与卡内链接/按钮的原生键盘激活冲突。
function onCourseCardKeydown(courseId: number, event: KeyboardEvent) {
  if (event.target !== event.currentTarget) return
  event.preventDefault()
  openPreview(courseId, event)
}

function closePreview() {
  previewCourseId.value = null
  const trigger = previewTriggerEl.value
  previewTriggerEl.value = null
  if (!trigger) return
  nextTick(() => {
    if (document.contains(trigger)) trigger.focus()
  })
}

// KeepAlive 缓存路径：SPA 客户端导航（如预览面板「撰写评价」→ 详情页）时本页
// 不卸载而走 onDeactivated，onBeforeUnmount 不会触发——若预览面板正打开（移动端
// dialog 锁定了 body 滚动），滚动锁会残留到目标页导致无法滑动。此处显式关闭预览
// 面板以触发 body 解锁；返回本页时面板保持关闭（与整页导航后的行为一致）。
onDeactivated(() => {
  if (previewCourseId.value != null) closePreview()
  // 必须一并摘除 document 级捕获监听器：它挂在 document 上，不会随组件停用而消失。
  // 残留时在详情页点击任意 /courses/{id} 链接（如右侧「相关课程」）仍会触发 handler，
  // 把当时的详情页 URL 写入 sessionStorage，覆盖真正保存的目录 URL，
  // 导致「返回课程目录」退化为裸 /courses（搜索/筛选态丢失）。
  clearTimeout(typeTimer)
  document.removeEventListener('click', onCourseDetailClick, true)
})
onActivated(() => {
  // 从缓存恢复时若 body 仍处于锁定状态（异常残留），兜底解锁。
  if (typeof document !== 'undefined') document.body.style.overflow = ''
  // 重新挂载捕获监听器（onDeactivated 已摘除），否则返回目录页后不再记录返回态。
  document.addEventListener('click', onCourseDetailClick, true)
})

// 收藏状态单一来源：以服务端初始收藏为镜像，本地乐观增删。
// 取消收藏时直接从该列表移除，避免「服务端集合残留 id」导致二次点击无法取消/UI 恒显示已收藏。
const bookmarkedIds = ref<number[]>(page.props.bookmarkedCourseIDs ?? [])
const bookmarkedSet = computed(() => new Set(bookmarkedIds.value))

function isBookmarked(courseId: number): boolean {
  return bookmarkedSet.value.has(courseId)
}

// 预览侧边栏内收藏成功（组件内已调用 bookmarkCourse），仅同步本地状态。
function onPreviewBookmarkToggle() {
  if (previewCourseId.value == null) return
  const current = new Set(bookmarkedIds.value)
  if (current.has(previewCourseId.value)) current.delete(previewCourseId.value)
  else current.add(previewCourseId.value)
  bookmarkedIds.value = [...current]
}

// 表格收藏（乐观更新单一状态源；失败回滚并提示）。
async function toggleBookmark(courseId: number) {
  if (!isAuthenticated.value) {
    window.location.href = loginHref.value
    return
  }
  const next = !isBookmarked(courseId)
  const current = new Set(bookmarkedIds.value)
  if (next) current.add(courseId)
  else current.delete(courseId)
  bookmarkedIds.value = [...current]
  try {
    await bookmarkCourse(courseId, next ? 1 : 2)
  } catch {
    const revert = new Set(bookmarkedIds.value)
    if (next) revert.delete(courseId)
    else revert.add(courseId)
    bookmarkedIds.value = [...revert]
    pushFlash(t('api.bookmarkFailed'), 'error')
  }
}

// 无限滚动（对齐首页帖子流）：SSR 首屏 props 复制到本地 ref，滚动到底自动加载下一页。
const courses = ref<CourseSummaryPayload[]>([])
const pagination = ref(page.props.pagination)
const loadingMore = ref(false)
const loadError = ref('')

watch(
  () => page.pageUrl,
  () => {
    courses.value = page.props.courses
    pagination.value = page.props.pagination
    loadingMore.value = false
    loadError.value = ''
    keywordInput.value = query.value.keyword ?? ''
  },
  { immediate: true },
)

async function loadMore() {
  if (loadingMore.value || !pagination.value.hasNext) return
  loadingMore.value = true
  loadError.value = ''
  try {
    // 翻页走课程 JSON API（GET /api/forum/courses）而非 SSR 页面路由：
    // SSR 首屏才需要 departments/terms/campuses 等静态筛选项与收藏集合，
    // 翻页重复请求 SSR 会把这些查询按滚动页数线性放大，而前端只取课程列表。
    const result = await listCourses({
      keyword: query.value.keyword,
      department: query.value.department,
      term: query.value.term,
      campus: query.value.campus,
      instructor: query.value.instructor,
      onlyWithReviews: query.value.onlyWithReviews,
      sortBy: query.value.sortBy,
      page: pagination.value.page + 1,
      size: page.props.query.size,
    })
    courses.value = mergeCourses(courses.value, result.list)
    pagination.value = {
      page: result.page,
      nextPage: result.page + 1,
      hasNext: result.hasNext,
      // JSON API 翻页按页码推进，不再依赖 SSR 生成的 nextUrl。
      nextUrl: '',
    }
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : t('common.loadFailed')
  } finally {
    loadingMore.value = false
  }
}
</script>

<template>
  <div class="pb-12">
    <PageHeader :title="t('coursesPage.title')" :description="t('coursesPage.subtitle')" />

    <div class="flex flex-col items-stretch gap-6 lg:flex-row lg:items-start lg:gap-8">
      <!-- 主列 -->
      <div class="min-w-0 flex-1">
        <!-- 搜索横幅：编辑式「课程寻宝」hero（品牌浅染 + 角落柔光 + 一体化搜索 pill + 常用筛选） -->
        <section class="gf-card relative mb-8 overflow-hidden p-5 sm:p-8 lg:p-10">
          <div
            aria-hidden="true"
            class="pointer-events-none absolute inset-0 bg-gradient-to-br from-info/10 via-transparent to-base-200/30"
          ></div>
          <div aria-hidden="true" class="pointer-events-none absolute -right-16 -top-24 h-72 w-72 rounded-full bg-primary/10 blur-3xl"></div>
          <!-- 左下角圆点纹理：currentColor + 环形渐隐，随主题明暗自适应 -->
          <div
            aria-hidden="true"
            class="pointer-events-none absolute -bottom-16 -left-14 h-60 w-60 text-base-content opacity-[0.15] dark:opacity-[0.12]"
            style="
              background-image: radial-gradient(currentColor 1.5px, transparent 1.5px);
              background-size: 15px 15px;
              mask-image: radial-gradient(circle, black 30%, transparent 72%);
              -webkit-mask-image: radial-gradient(circle, black 30%, transparent 72%);
            "
          ></div>

          <div class="relative z-10">
            <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
              <div class="max-w-2xl">
                <h2 class="text-balance text-3xl font-bold leading-[1.1] tracking-tight text-base-content sm:text-4xl lg:text-[2.75rem]">
                  {{ t('coursesPage.bannerTitleLeading') }}
                  <span class="bg-gradient-to-r from-sky-600 via-primary to-cyan-600 bg-clip-text text-transparent dark:from-sky-300 dark:via-primary dark:to-cyan-300">{{
                    t('coursesPage.bannerTitleAccent')
                  }}</span>
                </h2>
                <!-- 三词条愿景：icon + 词组并排，弱化文字重量换取节奏感 -->
                <p
                  class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-base-content/75 sm:text-[15px]"
                  :aria-label="t('coursesPage.bannerSubtitle')"
                >
                  <span class="inline-flex items-center gap-1.5 font-medium">
                    <Compass class="h-4 w-4 shrink-0 text-primary/75" aria-hidden="true" />
                    {{ t('coursesPage.bannerSubtitleItems.discover') }}
                  </span>
                  <span class="inline-flex items-center gap-1.5 font-medium">
                    <Lightbulb class="h-4 w-4 shrink-0 text-primary/75" aria-hidden="true" />
                    {{ t('coursesPage.bannerSubtitleItems.seek') }}
                  </span>
                  <span class="inline-flex items-center gap-1.5 font-medium">
                    <Share2 class="h-4 w-4 shrink-0 text-primary/75" aria-hidden="true" />
                    {{ t('coursesPage.bannerSubtitleItems.share') }}
                  </span>
                </p>
              </div>

              <!-- 桌面端右侧：一排模糊的迷你「课程卡片」，营造选课目录的生机（纯装饰，aria-hidden） -->
              <div aria-hidden="true" class="hidden shrink-0 items-end lg:flex">
                <div class="flex w-16 -rotate-6 flex-col gap-1 rounded-lg border border-line/50 bg-base-100/60 p-2 opacity-70 blur-[1px] shadow-sm">
                  <div class="h-1.5 w-6 rounded-full bg-primary/40"></div>
                  <div class="h-1.5 w-full rounded-full bg-base-200"></div>
                  <div class="mt-0.5 flex items-center gap-1">
                    <Star class="h-3 w-3 fill-warning/60 text-warning/60" />
                    <div class="h-1 w-6 rounded-full bg-warning/30"></div>
                  </div>
                </div>
                <div class="z-10 -ml-2 flex w-20 -translate-y-3 rotate-2 flex-col gap-1 rounded-lg border border-line/70 bg-base-100/95 p-2.5 shadow-md">
                  <div class="flex items-center gap-1.5">
                    <div class="h-3.5 w-3.5 shrink-0 rounded-md bg-info/30"></div>
                    <div class="h-1.5 w-full rounded-full bg-base-200"></div>
                  </div>
                  <div class="h-1.5 w-4/5 rounded-full bg-base-200"></div>
                  <div class="mt-1 flex items-center justify-between">
                    <div class="flex items-center gap-1">
                      <Star class="h-3 w-3 fill-warning text-warning" />
                      <div class="h-1 w-7 rounded-full bg-warning/40"></div>
                    </div>
                    <div class="h-1 w-4 rounded-full bg-base-200"></div>
                  </div>
                </div>
                <div class="-ml-2 flex w-16 rotate-6 flex-col gap-1 rounded-lg border border-line/50 bg-base-100/60 p-2 opacity-70 blur-[1px] shadow-sm">
                  <div class="h-1.5 w-6 rounded-full bg-primary/40"></div>
                  <div class="h-1.5 w-full rounded-full bg-base-200"></div>
                  <div class="mt-0.5 flex items-center gap-1">
                    <Star class="h-3 w-3 fill-warning/60 text-warning/60" />
                    <div class="h-1 w-6 rounded-full bg-warning/30"></div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 一体化圆角搜索 pill（打字机占位 + 清空按钮） -->
            <form
              class="mt-6 flex max-w-xl items-center gap-1 rounded-full border border-line/80 bg-base-100/90 p-1.5 pl-2 shadow-sm transition focus-within:border-primary/50 focus-within:ring-2 focus-within:ring-primary/15 lg:mt-8"
              action="/courses"
              method="get"
              role="search"
            >
              <label class="sr-only" for="course-keyword">{{ t('coursesPage.search') }}</label>
              <Search class="ml-3 h-4 w-4 shrink-0 text-base-content/40" />
              <input
                id="course-keyword"
                ref="keywordInputEl"
                name="keyword"
                type="search"
                v-model="keywordInput"
                :placeholder="placeholderText"
                class="course-search-input min-w-0 flex-1 bg-transparent py-2.5 text-base text-base-content outline-none placeholder:text-base-content/45 sm:text-sm"
                @keydown.esc="clearSearch"
              />
              <button
                v-if="keywordInput"
                type="button"
                class="shrink-0 rounded-full p-1.5 text-base-content/45 transition hover:bg-base-200 hover:text-base-content focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
                :aria-label="t('coursesPage.clearSearch')"
                @click="clearSearch"
              >
                <X class="h-4 w-4" />
              </button>
              <button type="submit" class="gf-button gf-button-md gf-button-primary shrink-0 rounded-full">
                {{ t('coursesPage.search') }}
              </button>
            </form>

            <!-- 常用筛选快捷标签 -->
            <div class="mt-6 flex flex-wrap items-center gap-2" role="group">
              <span class="mr-1 text-xs font-medium uppercase tracking-wide text-base-content/50">
                {{ t('coursesPage.intelligent') }}
              </span>
              <a
                v-for="chip in quickFilters"
                :key="chip.key"
                :href="chip.href"
                :aria-current="chip.active ? 'true' : undefined"
                class="rounded-full border px-3.5 py-1.5 text-sm font-medium motion-safe:transition-[transform,background-color,border-color,color] duration-200 hover:-translate-y-[1px] active:scale-[0.96]"
                :class="
                  chip.active
                    ? 'border-primary/50 bg-primary/10 text-primary'
                    : 'border-line bg-base-100/70 text-base-content/75 hover:border-primary/40 hover:text-primary'
                "
              >
                {{ chip.label }}
              </a>
            </div>
          </div>
        </section>

        <!-- 课程列表（表格） -->
        <section class="gf-card overflow-hidden">
          <header class="flex items-center justify-between gap-3 border-b border-line/70 px-4 py-3">
            <h2 class="text-base font-bold text-base-content">{{ t('coursesPage.courseList') }}</h2>
          </header>

          <!-- 筛选栏：单行，仅回显「更多筛选」中已选中的条件（院系全列表收敛到更多筛选面板） -->
          <div class="flex items-center gap-2 border-b border-line/70 px-4 py-3">
            <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2 lg:flex-nowrap lg:overflow-x-auto">
              <a
                href="/courses"
                class="shrink-0 rounded-md px-3 py-1.5 text-sm font-medium"
                :class="hasActiveFilters ? 'bg-base-200 text-base-content/60 hover:bg-base-200/80' : 'bg-primary text-primary-content'"
              >
                {{ t('coursesPage.allDepartments') }}
              </a>
              <template v-for="chip in selectedFilterChips" :key="`${chip.key}:${chip.value}`">
                <a
                  :href="removeChipHref(chip)"
                  class="group inline-flex shrink-0 items-center gap-1 rounded-md border border-primary/40 bg-info/10 px-3 py-1.5 text-sm text-primary"
                >
                  <span class="max-w-[12rem] truncate">{{ chip.label }}</span>
                  <X class="h-3.5 w-3.5 shrink-0 text-primary/70 transition group-hover:text-primary" />
                </a>
              </template>
            </div>
            <button
              type="button"
              class="flex shrink-0 items-center gap-1 rounded-md px-2.5 py-1.5 text-sm text-base-content/60 transition hover:text-base-content"
              :aria-expanded="moreFiltersOpen"
              aria-controls="course-more-filters"
              @click="moreFiltersOpen = !moreFiltersOpen"
            >
              <SlidersHorizontal class="h-4 w-4" />
              {{ t('coursesPage.moreFilters') }}
              <span v-if="activeFilterCount" class="rounded-full bg-primary px-1.5 py-0.5 text-xs font-medium tabular-nums text-primary-content">+{{ activeFilterCount }}</span>
              <ChevronDown class="h-3.5 w-3.5 transition-transform" :class="{ 'rotate-180': moreFiltersOpen }" />
            </button>
          </div>

          <!-- 更多筛选展开面板（院系/学期/校区/教师/仅看有评价/排序）：
               第二行三列统一为 h-10 字段（输入 / 复选 / 排序），图标均贴左内缘，共享对齐边；底部「清除全部筛选 ⇄ 应用筛选」两端布局。 -->
          <form v-if="moreFiltersOpen" id="course-more-filters" class="border-b border-line/70 bg-base-200/30 p-4" action="/courses" method="get" role="search">
            <div class="grid gap-2 sm:grid-cols-3">
              <CourseCatalogMultiSelect
                model-name="department"
                :label="t('coursesPage.allDepartments')"
                :options="page.props.departments.map((d) => ({ value: d, label: d }))"
                :selected="page.props.query.department ?? []"
                :placeholder="t('common.search')"
                :icon="Building2"
              />
              <CourseCatalogMultiSelect
                model-name="term"
                :label="t('coursesPage.allTerms')"
                :options="page.props.terms.map((term) => ({ value: term.value, label: term.label }))"
                :selected="page.props.query.term ?? []"
                :placeholder="t('common.search')"
                :icon="CalendarDays"
              />
              <CourseCatalogMultiSelect
                model-name="campus"
                :label="t('coursesPage.allCampuses')"
                :options="page.props.campuses.map((c) => ({ value: c, label: c }))"
                :selected="page.props.query.campus ?? []"
                :placeholder="t('common.search')"
                :icon="MapPin"
              />
            </div>
            <div class="mt-2 grid gap-2 sm:grid-cols-3">
              <div class="relative">
                <UsersRound class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
                <input id="course-instructor" name="instructor" type="text" :value="(page.props.query.instructor ?? []).join(', ')" :placeholder="t('coursesPage.teacher')" class="gf-input gf-input-md w-full pl-9" />
              </div>
              <!-- 「仅看有评价」：布尔开关（switch），第一性上不是输入字段——
                   设置行样式（左文案右开关）而非大字段壳；选中态轨道转 primary，一眼可辨。 -->
              <label class="flex h-10 w-full cursor-pointer items-center justify-between gap-2.5 px-1 text-sm text-base-content/75 transition-colors hover:text-base-content">
                <span class="truncate">{{ t('coursesPage.onlyWithReviews') }}</span>
                <input name="onlyWithReviews" type="checkbox" value="1" :checked="page.props.query.onlyWithReviews" class="peer sr-only" />
                <span
                  class="relative h-5 w-9 shrink-0 rounded-full bg-base-300 transition-colors duration-200 after:absolute after:left-0.5 after:top-0.5 after:h-4 after:w-4 after:rounded-full after:bg-white after:shadow-sm after:transition-transform after:duration-200 peer-focus-visible:ring-2 peer-focus-visible:ring-primary/40 peer-checked:bg-primary peer-checked:after:translate-x-4"
                  aria-hidden="true"
                />
              </label>
              <!-- 排序：两项分段切换器（自建控件族，替代浏览器原生下拉）；选中值经隐藏 input 随表单提交 -->
              <div
                class="flex h-10 w-full items-center gap-1 rounded-[var(--gf-radius-field)] border border-line/70 bg-base-100 p-1"
                role="group"
                :aria-label="t('coursesPage.sortLabel')"
              >
                <button
                  type="button"
                  class="flex h-8 flex-1 items-center justify-center rounded-md text-[13px] font-medium transition-colors duration-150"
                  :class="draftSortBy === '' ? 'bg-primary text-primary-content shadow-sm' : 'text-base-content/60 hover:bg-base-200 hover:text-base-content'"
                  :aria-pressed="draftSortBy === ''"
                  @click="draftSortBy = ''"
                >
                  {{ t('coursesPage.sortDefault') }}
                </button>
                <button
                  type="button"
                  class="flex h-8 flex-1 items-center justify-center gap-1 rounded-md text-[13px] font-medium transition-colors duration-150"
                  :class="draftSortBy === 'rating' ? 'bg-primary text-primary-content shadow-sm' : 'text-base-content/60 hover:bg-base-200 hover:text-base-content'"
                  :aria-pressed="draftSortBy === 'rating'"
                  @click="draftSortBy = 'rating'"
                >
                  <Star class="h-3.5 w-3.5" aria-hidden="true" />
                  {{ t('coursesPage.sortByRating') }}
                </button>
              </div>
              <input type="hidden" name="sortBy" :value="draftSortBy" />
            </div>
            <div class="mt-3 flex items-center gap-2">
              <a v-if="hasActiveFilters" href="/courses" class="gf-button gf-button-md gf-button-ghost">
                <Eraser class="h-4 w-4" />
                {{ t('coursesPage.clearFilters') }}
              </a>
              <button type="submit" class="gf-button gf-button-md gf-button-primary ml-auto">
                {{ t('coursesPage.applyFilters') }}
              </button>
            </div>
          </form>

          <EmptyState
            v-if="!courses.length"
            class="m-4 border border-line/60"
            :icon="BookOpen"
            :title="hasActiveFilters ? t('coursesPage.noFilterResults') : t('coursesPage.emptyTitle')"
            :description="hasActiveFilters ? t('coursesPage.noFilterResultsDesc') : t('coursesPage.emptyDescription')"
          >
            <a v-if="hasActiveFilters" href="/courses" class="gf-button gf-button-md gf-button-outline">
              {{ t('coursesPage.clearFilters') }}
            </a>
          </EmptyState>

          <template v-else>
            <!-- 移动端（<lg）课程卡片列表：与表格共用 courses 数据源，CSS 断点切换，SSR 首帧与水合一致 -->
            <div class="lg:hidden">
              <ul class="flex flex-col gap-2 p-3 sm:p-4" :aria-label="t('coursesPage.courseList')">
                <li v-for="course in courses" :key="course.id">
                  <div
                    role="button"
                    tabindex="0"
                    class="relative flex w-full cursor-pointer items-start gap-3 rounded-xl border border-line/70 bg-base-100 p-3 text-left shadow-sm transition-colors duration-150 hover:border-primary/30 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
                    :class="{ 'border-primary/30 bg-info/5': previewCourseId === course.id }"
                    :aria-label="course.name"
                    :aria-current="previewCourseId === course.id ? 'true' : undefined"
                    @click="onCourseRowClick(course.id, $event)"
                    @keydown.enter="onCourseCardKeydown(course.id, $event)"
                    @keydown.space="onCourseCardKeydown(course.id, $event)"
                  >
                    <div class="min-w-0 flex-1">
                      <div class="flex items-start gap-2">
                        <a :href="`/courses/${course.id}`" class="min-w-0 flex-1 truncate font-medium text-base-content underline-offset-2 hover:text-primary hover:underline decoration-primary/60 decoration-2" :title="course.name">{{ course.name }}</a>
                        <span class="shrink-0 rounded-md bg-base-200/80 px-1.5 py-0.5 text-[11px] leading-5 tabular-nums text-base-content/55" :title="course.primaryCode">{{ course.primaryCode }}</span>
                      </div>
                      <p class="mt-1 truncate text-xs text-base-content/60" :title="course.teacherName || course.instructors?.join('、') || t('coursesPage.noTeacher')">
                        {{ course.teacherName || course.instructors?.join('、') || t('coursesPage.noTeacher') }} · {{ course.department }}
                      </p>
                      <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs">
                        <span v-if="course.ratingAvg != null" class="inline-flex items-center gap-1">
                          <Star class="h-3.5 w-3.5 shrink-0 fill-warning text-warning" />
                          <span class="tabular-nums font-medium text-base-content">{{ course.ratingAvg.toFixed(1) }}</span>
                          <span class="tabular-nums text-base-content/50">({{ course.reviewCount ?? 0 }})</span>
                        </span>
                        <span v-else-if="course.reviewCount" class="text-base-content/50">{{ t('coursesPage.noRating') }}</span>
                        <span class="tabular-nums text-base-content/70">{{ t('coursesPage.creditUnit') }} {{ course.creditX10 ? (course.creditX10 / 10).toFixed(1).replace(/\.0$/, '') : '—' }}</span>
                        <span class="inline-flex flex-wrap items-center gap-1.5">
                          <template v-if="courseTermList[course.id]?.length">
                            <button
                              v-if="courseTermList[course.id].length > 1"
                              type="button"
                              class="inline-flex h-6 min-w-8 items-center justify-center gap-1 whitespace-nowrap rounded-full border border-line/60 bg-base-200/50 px-1.5 text-xs font-medium text-primary transition-colors duration-150 hover:bg-base-200/80"
                              :aria-expanded="expandedTermRows.has(course.id)"
                              @click.stop="toggleTermExpand(course.id)"
                            >
                              <span>{{ shortTerm(courseTermList[course.id][0]) }}</span>
                              <span class="tabular-nums text-base-content/50">
                                {{ expandedTermRows.has(course.id) ? `−${courseTermList[course.id].length - 1}` : `+${courseTermList[course.id].length - 1}` }}
                              </span>
                            </button>
                            <span v-else class="inline-flex h-6 items-center whitespace-nowrap rounded-full border border-line/60 bg-base-200/50 px-2 text-xs text-base-content/70">
                              {{ shortTerm(courseTermList[course.id][0]) }}
                            </span>
                            <template v-if="expandedTermRows.has(course.id) && courseTermList[course.id].length > 1">
                              <span
                                v-for="term in courseTermList[course.id].slice(1)"
                                :key="term"
                                class="inline-flex h-6 items-center whitespace-nowrap rounded-full border border-line/60 bg-base-200/50 px-2 text-xs text-base-content/70"
                              >
                                {{ shortTerm(term) }}
                              </span>
                            </template>
                          </template>
                          <span v-else class="text-base-content/40">—</span>
                        </span>
                      </div>
                    </div>
                    <button
                      type="button"
                      class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-lg p-2 text-sm transition-colors duration-150 active:scale-[0.96]"
                      :class="isBookmarked(course.id) ? 'text-primary' : 'text-base-content/45 hover:text-primary'"
                      :aria-pressed="isBookmarked(course.id)"
                      :aria-label="isBookmarked(course.id) ? t('api.bookmarked') : t('api.bookmark')"
                      :title="isBookmarked(course.id) ? t('api.bookmarked') : t('api.bookmark')"
                      @click.stop="toggleBookmark(course.id)"
                    >
                      <Bookmark class="h-5 w-5" :class="{ 'fill-primary': isBookmarked(course.id) }" aria-hidden="true" />
                    </button>
                  </div>
                </li>
              </ul>
            </div>

            <!-- 桌面端（≥lg）原表格：类名与数据绑定保持原样 -->
            <div class="hidden overflow-x-auto lg:block">
              <table class="w-full min-w-[760px] table-fixed text-left text-sm">
                <colgroup>
                  <col class="w-[21%]" />
                  <col class="w-[11%]" />
                  <col class="w-[13%]" />
                  <col class="w-[13%]" />
                  <col class="w-[9%]" />
                  <col class="w-[7%]" />
                  <col class="w-[7%]" />
                  <col class="w-[11%]" />
                  <col class="w-[8%]" />
                </colgroup>
                <thead class="border-b border-line/70 bg-base-200/40 text-xs text-base-content/55">
                  <tr>
                    <th class="px-3 py-3 font-medium">{{ t('coursesPage.columnCourse') }}</th>
                    <th class="px-3 py-3 font-medium">{{ t('coursesPage.columnCode') }}</th>
                    <th class="px-3 py-3 font-medium">{{ t('coursesPage.columnTeacher') }}</th>
                    <th class="px-3 py-3 font-medium">{{ t('coursesPage.columnDepartment') }}</th>
                    <th class="px-3 py-3 font-medium">{{ t('coursesPage.columnRating') }}</th>
                    <th class="px-3 py-3 font-medium">{{ t('coursesPage.columnReviewCount') }}</th>
                    <th class="px-3 py-3 font-medium">{{ t('coursesPage.columnCredit') }}</th>
                    <th class="px-3 py-3 font-medium">{{ t('coursesPage.columnTerms') }}</th>
                    <th class="px-3 py-3 text-right font-medium">{{ t('coursesPage.columnActions') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-line/60">
                  <tr
                    v-for="course in courses"
                    :key="course.id"
                    class="cursor-pointer transition hover:bg-base-200/40 focus-within:bg-base-200/40"
                    :class="{ 'bg-info/5': previewCourseId === course.id }"
                    :aria-current="previewCourseId === course.id ? 'true' : undefined"
                    :title="t('coursesPage.rowHint')"
                    tabindex="-1"
                    @click="onCourseRowClick(course.id, $event)"
                  >
                    <td class="min-w-0 px-3 py-3">
                      <a
                        :href="`/courses/${course.id}`"
                        class="block truncate font-medium text-base-content underline-offset-2 hover:text-primary hover:underline decoration-primary/60 decoration-2"
                        :title="course.name"
                        @click.stop
                      >{{ course.name }}</a>
                    </td>
                    <td class="px-3 py-3 text-base-content/55"><span class="block truncate" :title="course.primaryCode">{{ course.primaryCode }}</span></td>
                    <td class="min-w-0 px-3 py-3 text-base-content/75">
                      <span class="block truncate" :title="course.teacherName || course.instructors?.join('、') || t('coursesPage.noTeacher')">
                        {{ course.teacherName || course.instructors?.join('、') || t('coursesPage.noTeacher') }}
                      </span>
                    </td>
                    <td class="min-w-0 px-3 py-3 text-base-content/55">
                      <span class="block truncate" :title="course.department">{{ course.department }}</span>
                    </td>
                    <td class="px-3 py-3">
                      <span v-if="course.ratingAvg != null" class="inline-flex items-center gap-1 text-amber-500">
                        <Star class="h-3.5 w-3.5 fill-warning text-warning" />
                        <span class="tabular-nums font-medium text-base-content">{{ course.ratingAvg.toFixed(1) }}</span>
                      </span>
                      <span v-else-if="course.reviewCount" class="text-base-content/50">{{ t('coursesPage.noRating') }}</span>
                    </td>
                    <td class="px-3 py-3 text-base-content/55"><span class="tabular-nums">({{ course.reviewCount ?? 0 }})</span></td>
                    <td class="px-3 py-3 text-base-content/75 tabular-nums">{{ course.creditX10 ? (course.creditX10 / 10).toFixed(1).replace(/\.0$/, '') : '—' }}</td>
                    <td class="px-3 py-3 text-base-content/55">
                      <div v-if="courseTermList[course.id]?.length" class="flex flex-col items-start gap-1">
                        <button
                          v-if="courseTermList[course.id].length > 1"
                          type="button"
                          class="inline-flex items-center gap-1 text-left font-medium text-primary transition hover:text-primary/80"
                          :aria-expanded="expandedTermRows.has(course.id)"
                          @click.stop="toggleTermExpand(course.id)"
                        >
                          <span>{{ shortTerm(courseTermList[course.id][0]) }}</span>
                          <span class="rounded-full bg-base-200/80 px-1.5 py-0.5 text-xs leading-none tabular-nums text-base-content/60">
                            {{ expandedTermRows.has(course.id) ? `−${courseTermList[course.id].length - 1}` : `+${courseTermList[course.id].length - 1}` }}
                          </span>
                        </button>
                        <span v-else>{{ shortTerm(courseTermList[course.id][0]) }}</span>
                        <div
                          v-if="expandedTermRows.has(course.id) && courseTermList[course.id].length > 1"
                          class="flex flex-wrap gap-x-2 gap-y-0.5 text-xs text-base-content/50"
                        >
                          <span v-for="term in courseTermList[course.id].slice(1)" :key="term">{{ shortTerm(term) }}</span>
                        </div>
                      </div>
                      <span v-else>—</span>
                    </td>
                    <td class="px-3 py-3 text-right">
                      <div class="inline-flex items-center justify-end gap-0.5">
                        <button
                          type="button"
                          class="inline-flex h-7 w-7 items-center justify-center rounded-md transition active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
                          :class="
                            previewCourseId === course.id
                              ? 'bg-primary/10 text-primary'
                              : 'text-base-content/40 hover:bg-base-200 hover:text-primary'
                          "
                          :aria-label="t('coursesPage.previewCourse')"
                          :title="t('coursesPage.previewCourse')"
                          @click.stop="openPreview(course.id, $event)"
                        >
                          <Eye class="h-4 w-4" aria-hidden="true" />
                        </button>
                        <button
                          type="button"
                          class="inline-flex h-7 w-7 items-center justify-center rounded-md transition active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
                          :class="isBookmarked(course.id) ? 'text-primary' : 'text-base-content/40 hover:bg-base-200 hover:text-primary'"
                          :aria-pressed="isBookmarked(course.id)"
                          :aria-label="isBookmarked(course.id) ? t('api.bookmarked') : t('api.bookmark')"
                          :title="isBookmarked(course.id) ? t('api.bookmarked') : t('api.bookmark')"
                          @click.stop="toggleBookmark(course.id)"
                        >
                          <Bookmark class="h-4 w-4" :class="{ 'fill-primary': isBookmarked(course.id) }" aria-hidden="true" />
                        </button>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </template>

          <InfiniteScrollFooter
            :has-next="pagination.hasNext"
            :loading="loadingMore"
            :error="loadError"
            :has-items="courses.length > 0"
            @load-more="loadMore"
          />
        </section>
      </div>

      <!-- 课程预览侧边栏 -->
      <CoursePreviewPane
        :course="previewCourse"
        :is-authenticated="isAuthenticated"
        :bookmarked-course-ids="bookmarkedIds"
        @close="closePreview"
        @bookmark-toggle="onPreviewBookmarkToggle"
        @select-course="onPreviewSelectCourse"
      />
    </div>
  </div>
</template>

<style scoped>
/* 隐藏 type="search" 浏览器自带的清除按钮，避免与自定义清空按钮重复。 */
.course-search-input::-webkit-search-cancel-button,
.course-search-input::-webkit-search-decoration {
  -webkit-appearance: none;
  appearance: none;
}
</style>
