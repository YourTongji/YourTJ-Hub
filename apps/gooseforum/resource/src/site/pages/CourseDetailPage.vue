<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, BookOpen, Building2, CalendarDays, ChevronDown, FileText, Flag, Loader2, MessageSquareText, Pencil, Star, ThumbsUp, Trash2, UsersRound, X } from '@lucide/vue'
import {
  createCourseReview,
  deleteCourseReview,
  getCourseRelated,
  listCourseReviews,
  reportCourseReview,
  setReviewHelpful,
  updateCourseReview,
  type CourseRelatedResult,
  type ReviewPayload,
} from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
import { useFlashMessages } from '@/runtime/flash-message'
import CourseReviewTemplateSelector from '@/site/components/CourseReviewTemplateSelector.vue'
import AISummaryCard from '@/site/components/AISummaryCard.vue'
import EmptyState from '@/site/components/EmptyState.vue'
import InfiniteScrollFooter from '@/site/components/InfiniteScrollFooter.vue'
import { COURSE_REVIEW_TEMPLATES } from '@/site/utils/course-review-templates'
import {
  nextReviewTotalOnCreate,
  nextReviewTotalOnDelete,
  resolveStatsReviewCount,
} from '@/site/utils/course-review-count'
import { createReviewPageLoader } from '@/site/utils/course-review-loader'
import PageHeader from '@/site/components/PageHeader.vue'
import type { CourseDetailPageProps, LayoutPayload } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: CourseDetailPageProps
}>()
const { t } = useI18n()
const { push: pushFlash } = useFlashMessages()

const loginHref = computed(() => {
  const next = encodeURIComponent(window.location.pathname + window.location.search + window.location.hash)
  return `/login?next=${next}`
})

function formatCredit(creditX10: number) {
  if (!creditX10) return ''
  return (creditX10 / 10).toFixed(1).replace(/\.0$/, '')
}

function offeringLabel(id: number) {
  const offering = page.props.course.offerings?.find((item) => item.id === id)
  if (!offering) return `#${id}`
  const classLabel = offering.className || offering.classCode || ''
  return [offering.termCode, classLabel, offering.campus, offering.instructors?.join('、')].filter(Boolean).join(' · ')
}

function formatRating(avg: number) {
  return avg > 0 ? avg.toFixed(1) : '—'
}

// ---- 顶部统计区（B1：ratingAvg / reviewCount / ratingDistribution）----
// ratingDistribution 为 [1星, 2星, 3星, 4星, 5星] 各档计数（index 0 = 1 星）。
const ratingAvg = computed(() => page.props.course.ratingAvg ?? null)
const reviewCount = computed(() => page.props.course.reviewCount ?? 0)
const ratingDistribution = computed(() => page.props.course.ratingDistribution ?? [0, 0, 0, 0, 0])
// 分布行按 5 星 → 1 星降序展示；基准取最大值（至少 1，避免除零）。
const distributionRows = computed(() =>
  [5, 4, 3, 2, 1].map((star) => ({ star, count: ratingDistribution.value[star - 1] ?? 0 })),
)
const distributionMax = computed(() => Math.max(...ratingDistribution.value, 1))

// ---- 相关课程（同教师其他课 / 同课程其他教师）----
const related = ref<CourseRelatedResult | null>(null)
// 初始为 true：避免首屏未加载完成时错误闪现"暂无相关内容"。
const relatedLoading = ref(true)
const relatedError = ref('')
const relatedMobileExpanded = ref(false)

async function loadRelated() {
  relatedLoading.value = true
  relatedError.value = ''
  try {
    related.value = await getCourseRelated(page.props.course.id)
  } catch (error) {
    relatedError.value = error instanceof Error ? error.message : t('courseDetailPage.relatedLoadFailed')
  } finally {
    relatedLoading.value = false
  }
}

// authorLabel 作者展示名：member 用服务端回填的用户名；anonymous/legacy 走本地 i18n，
// 避免非中文界面原样渲染服务端硬编码的中文标签。
function authorLabel(author: ReviewPayload['author']) {
  if (author.kind === 'member') return author.label
  if (author.kind === 'legacy') return t('courseDetailPage.authorLegacy')
  return t('courseDetailPage.authorAnonymous')
}

// ---- 评价列表 ----
const reviews = ref<ReviewPayload[]>([])
const reviewTotal = ref(0)
const reviewNextCursor = ref('')
const reviewLoadingMore = ref(false)
const reviewLoading = ref(false)
// 加载更多失败状态：就地展示于 InfiniteScrollFooter（错误态停止自动触发 + 手动重试），
// 区别于首屏 reviewError 的顶部 banner 展示。
const reviewLoadMoreError = ref('')
// reviewsLoadSeq 列表加载代际：写操作（创建/编辑/删除）成功后递增，使 in-flight 的
// 旧列表响应失效——否则初次 loadReviews 的旧快照会在写成功后返回并覆盖刚发布的
// 评价（unshift 内容消失、计数回退，直到刷新）。
let reviewsLoadSeq = 0

// invalidateReviews 写操作成功后调用：使 in-flight 列表加载失效；若仍有加载在途或
// 列表从未成功加载，触发一次静默重拉对账（此时服务端已包含本次写入），保证列表完整
// （丢弃旧快照后不重拉会只剩本地写入的一条、旧评价丢失）。
function invalidateReviews() {
  reviewsLoadSeq += 1
  if (reviewLoading.value || !reviewLoaded.value) {
    void loadReviews()
  }
}
const reviewError = ref('')
const reviewLoaded = ref(false)
const helpfulBusyIds = ref<number[]>([])

// 统计卡评论数：评价列表加载完成后以 reviewTotal（客户端、删除/创建后实时更新）为准，
// 未加载时回退 SSR 的 reviewCount，避免顶部统计卡与评价区计数口径分叉。
const statsReviewCount = computed(() => resolveStatsReviewCount(reviewLoaded.value, reviewTotal.value, reviewCount.value))

// 列表加载协调器：用请求版本号避免 onMounted 的首屏 GET 在创建/删除后返回旧快照
// 覆盖本地状态（issue #178 review P1 竞态）。
const reviewLoader = createReviewPageLoader((offeringId, cursor) =>
  listCourseReviews(page.props.course.id, offeringId, cursor),
)

async function loadReviews() {
  const seq = ++reviewsLoadSeq
  reviewLoading.value = true
  reviewError.value = ''
  try {
    const reviewPage = await reviewLoader.load(0, '')
    if (reviewPage === null) return // 过期响应：期间发生写操作，丢弃以保留本地状态
    // 代际守卫：期间有更新加载（写操作触发的重拉）发起，本结果作废。
    if (seq !== reviewsLoadSeq) return
    reviews.value = reviewPage.list
    reviewTotal.value = reviewPage.total
    reviewNextCursor.value = reviewPage.nextCursor ?? ''
  } catch (error) {
    // 旧代际的失败不覆盖新状态（避免已成功的写操作被错误提示淹没）。
    if (seq !== reviewsLoadSeq) return
    reviewError.value = error instanceof Error ? error.message : t('courseDetailPage.reviewsLoadFailed')
  } finally {
    // 仅当前代际可动 loading/loaded：旧代际完成时新代际仍在途，保留其 loading 状态。
    if (seq === reviewsLoadSeq) {
      reviewLoading.value = false
      reviewLoaded.value = true
    }
  }
}

// 加载更多（B2 cursor 分页，issue #174）
async function loadMoreReviews() {
  if (!reviewNextCursor.value || reviewLoadingMore.value) return
  const seq = reviewsLoadSeq
  reviewLoadingMore.value = true
  reviewLoadMoreError.value = ''
  try {
    const reviewPage = await reviewLoader.load(0, reviewNextCursor.value)
    if (reviewPage === null) return // 过期响应：丢弃
    // 代际守卫：写操作已失效本代（旧 cursor 数据可能含已删除/旧内容），丢弃不 concat。
    if (seq !== reviewsLoadSeq) return
    reviews.value = reviews.value.concat(reviewPage.list)
    reviewNextCursor.value = reviewPage.nextCursor ?? ''
  } catch (error) {
    if (seq !== reviewsLoadSeq) return
    reviewLoadMoreError.value = error instanceof Error ? error.message : t('courseDetailPage.reviewsLoadFailed')
  } finally {
    reviewLoadingMore.value = false
  }
}

function replaceReview(updated: ReviewPayload) {
  const index = reviews.value.findIndex((item) => item.id === updated.id)
  if (index >= 0) reviews.value[index] = updated
}

// ---- 写评 / 编辑表单 ----
const formVisible = ref(false)
const formOfferingId = ref<number>(0)
const formRating = ref(0)
const formContent = ref('')
const formAnonymous = ref(true)
const formSubmitting = ref(false)
const formError = ref('')
const editingReviewId = ref<number | null>(null)
const templateSelectorOpen = ref(false)
const formTemplateId = ref('')

function openCreateForm() {
  editingReviewId.value = null
  formOfferingId.value = page.props.course.offerings?.[0]?.id ?? 0
  formRating.value = 0
  formContent.value = ''
  formAnonymous.value = true
  formError.value = ''
  formTemplateId.value = ''
  templateSelectorOpen.value = false
  formVisible.value = true
}

function startEdit(review: ReviewPayload) {
  editingReviewId.value = review.id
  formOfferingId.value = review.offeringId
  formRating.value = review.rating ?? 0
  // 预填原始 Markdown 正文：列表 DTO 携带 content 字段（服务端返回），
  // 用户只改评分/匿名时不会因正文为空而被迫重写或不可逆覆盖原文。
  formContent.value = review.content ?? ''
  formAnonymous.value = review.author.kind === 'anonymous'
  formError.value = ''
  formTemplateId.value = ''
  templateSelectorOpen.value = false
  formVisible.value = true
}

function cancelForm() {
  formVisible.value = false
  editingReviewId.value = null
  formError.value = ''
  templateSelectorOpen.value = false
}

function applyTemplate(id: string, content: string) {
  // 保护：模板只应用于空正文，避免静默覆盖用户已输入/编辑中的内容。
  if (formContent.value.trim()) {
    pushFlash(t('courseDetailPage.templateContentNotEmpty'), 'warning')
    return
  }
  formTemplateId.value = id
  formContent.value = content
  templateSelectorOpen.value = false
}

function templateName(id: string) {
  const template = COURSE_REVIEW_TEMPLATES.find((item) => item.id === id)
  return template ? t(template.nameKey) : ''
}

async function submitForm() {
  formError.value = ''
  if (!formOfferingId.value) {
    formError.value = t('courseDetailPage.selectOfferingRequired')
    return
  }
  if (formRating.value < 1 || formRating.value > 5) {
    formError.value = t('courseDetailPage.ratingRequired')
    return
  }
  if (!formContent.value.trim()) {
    formError.value = t('courseDetailPage.contentRequired')
    return
  }
  formSubmitting.value = true
  try {
    if (editingReviewId.value) {
      const updated = await updateCourseReview(editingReviewId.value, {
        rating: formRating.value,
        content: formContent.value,
        isAnonymous: formAnonymous.value,
      })
      replaceReview(updated)
    } else {
      const created = await createCourseReview({
        offeringId: formOfferingId.value,
        rating: formRating.value,
        content: formContent.value,
        isAnonymous: formAnonymous.value,
      })
      reviewLoader.invalidate() // 使进行中的首屏 GET 过期，避免旧快照覆盖刚创建的评价
      reviews.value.unshift(created)
      // 同步计数：创建后 +1（与下方删除路径递减口径一致），否则统计卡/评价区标题
      // 会一直显示旧值直到刷新。能提交评价说明列表已加载或已具备客户端最新态，
      // 兜底置 reviewLoaded 为 true，确保统计卡走 reviewTotal 而非回退 SSR 旧值。
      reviewTotal.value = nextReviewTotalOnCreate(reviewTotal.value)
      reviewLoaded.value = true
    }
    invalidateReviews()
    formVisible.value = false
    editingReviewId.value = null
  } catch (error) {
    formError.value = error instanceof Error ? error.message : t('courseDetailPage.reviewSaveFailed')
  } finally {
    formSubmitting.value = false
  }
}

// ---- 删除确认（受控 Dialog，替代 window.confirm）----
const pendingDelete = ref<ReviewPayload | null>(null)
// deleting 防 in-flight 双击：删除请求未返回前禁止重复提交/取消，
// 避免 pendingDelete 被中途置 null 导致"删 A 后误关 B 的 Dialog"竞态。
const deleting = ref(false)

function askRemoveReview(review: ReviewPayload) {
  if (deleting.value) return
  pendingDelete.value = review
}

function cancelRemoveReview() {
  if (deleting.value) return
  pendingDelete.value = null
}

async function confirmRemoveReview() {
  const review = pendingDelete.value
  if (!review || deleting.value) return
  deleting.value = true
  try {
    await deleteCourseReview(review.id)
    reviewLoader.invalidate() // 使进行中的 GET 过期，避免旧快照覆盖删除后的状态
    reviews.value = reviews.value.filter((item) => item.id !== review.id)
    reviewTotal.value = nextReviewTotalOnDelete(reviewTotal.value)
    pendingDelete.value = null
    pushFlash(t('courseDetailPage.reviewDeleted'), 'success')
    invalidateReviews()
  } catch (error) {
    pendingDelete.value = null
    pushFlash(error instanceof Error ? error.message : t('courseDetailPage.reviewDeleteFailed'), 'error')
  } finally {
    deleting.value = false
  }
}

// ---- helpful ----
async function toggleHelpful(review: ReviewPayload) {
  if (!page.layout.viewer.isAuthenticated) {
    window.location.href = loginHref.value
    return
  }
  if (helpfulBusyIds.value.includes(review.id)) return
  helpfulBusyIds.value.push(review.id)
  try {
    const next = !review.viewer.isHelpful
    await setReviewHelpful(review.id, next)
    review.viewer.isHelpful = next
    review.helpfulCount += next ? 1 : -1
  } catch (error) {
    pushFlash(error instanceof Error ? error.message : t('courseDetailPage.reviewHelpfulFailed'), 'error')
  } finally {
    helpfulBusyIds.value = helpfulBusyIds.value.filter((id) => id !== review.id)
  }
}

// ---- 举报 ----
const reportReasons = ['spam', 'abuse', 'illegal', 'irrelevant', 'other']
const pendingReport = ref<ReviewPayload | null>(null)
const reportReason = ref('spam')
const reportNote = ref('')
const reportSubmitting = ref(false)
const reportError = ref('')

function openReport(review: ReviewPayload) {
  pendingReport.value = review
  reportReason.value = 'spam'
  reportNote.value = ''
  reportError.value = ''
}

function closeReport() {
  pendingReport.value = null
  reportError.value = ''
}

async function submitReport() {
  if (!pendingReport.value) return
  reportSubmitting.value = true
  reportError.value = ''
  try {
    await reportCourseReview(pendingReport.value.id, reportReason.value, reportNote.value)
    closeReport()
  } catch (error) {
    reportError.value = error instanceof Error ? error.message : t('courseDetailPage.reviewReportFailed')
  } finally {
    reportSubmitting.value = false
  }
}

onMounted(() => {
  loadReviews()
  loadRelated()
})
</script>

<template>
  <div class="pb-12">
    <a href="/courses" class="mb-3 inline-flex items-center gap-1 text-[13px] text-base-content/55 hover:text-primary">
      <ArrowLeft class="h-3.5 w-3.5" />
      {{ t('courseDetailPage.backToList') }}
    </a>

    <PageHeader :title="props.course.name">
      <template #badge>
        <span class="gf-badge gf-badge-muted">{{ props.course.primaryCode }}</span>
      </template>
      <template #meta>
        <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-[13px] text-base-content/60">
          <span class="inline-flex items-center gap-1">
            <Building2 class="h-3.5 w-3.5" />
            {{ props.course.department }}
          </span>
          <span class="inline-flex items-center gap-1">
            <UsersRound class="h-3.5 w-3.5" />
            {{ props.course.teacherName || t('courseDetailPage.noTeacher') }}
          </span>
          <span v-if="formatCredit(props.course.creditX10)" class="inline-flex items-center gap-1">
            <span class="h-1 w-1 rounded-full bg-base-content/30" />
            {{ t('courseDetailPage.credit') }}：{{ formatCredit(props.course.creditX10) }}
          </span>
        </div>
      </template>
    </PageHeader>

    <div v-if="props.course.aliases?.length" class="mb-4 flex flex-wrap items-center gap-1.5">
      <span class="text-[12px] text-base-content/45">{{ t('courseDetailPage.aliases') }}：</span>
      <span v-for="alias in props.course.aliases" :key="alias" class="gf-badge gf-badge-ghost text-[11px]">
        {{ alias }}
      </span>
    </div>

    <!-- 内容区：桌面端（xl+）评价列表为主列，评分分布/开课记录/相关课程/AI 总结收纳右栏；移动端按 DOM 顺序纵向堆叠。 -->
    <div class="mt-6 grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,340px)] xl:items-start">
      <!-- 右栏（xl+ 排右侧；移动端先于评价列表显示，保持原顺序） -->
      <div class="min-w-0 space-y-4 xl:order-2 xl:sticky xl:top-6">
        <section class="gf-panel p-5">
          <div class="flex flex-col gap-5">
            <div class="flex flex-col items-center">
          <div class="text-4xl font-bold tracking-tight tabular-nums text-warning">
            {{ ratingAvg != null ? ratingAvg.toFixed(1) : '—' }}
          </div>
          <div class="mt-0.5 text-[12px] text-base-content/50">{{ t('courseDetailPage.ratingOutOf') }}</div>
          <div class="mt-1.5 text-[13px] text-base-content/60">
            {{ t('courseDetailPage.reviewCountLabel', { count: statsReviewCount }, statsReviewCount) }}
          </div>
        </div>

        <div class="min-w-0 flex-1 space-y-1.5">
          <div v-for="row in distributionRows" :key="row.star" class="flex items-center gap-2">
            <span class="w-8 shrink-0 text-right text-[12px] tabular-nums text-base-content/55">
              {{ row.star }} {{ t('courseDetailPage.stars') }}
            </span>
            <div class="h-2 flex-1 overflow-hidden rounded-full bg-base-300/60">
              <div
                class="h-full rounded-full bg-warning"
                :style="{ width: `${(row.count / distributionMax) * 100}%` }"
              />
            </div>
            <span class="w-6 shrink-0 text-[12px] tabular-nums text-base-content/45">{{ row.count }}</span>
          </div>
        </div>

      </div>
    </section>

    <section class="gf-panel p-4">
      <h2 class="mb-3 text-base font-semibold text-base-content">
        {{ t('courseDetailPage.offeringsTitle') }}
      </h2>
      <EmptyState
        v-if="!props.course.offerings?.length"
        :icon="CalendarDays"
        :title="t('courseDetailPage.noOfferings')"
      />
      <ul v-else class="space-y-3">
        <li
          v-for="offering in props.course.offerings"
          :key="offering.id"
          class="rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 sm:bg-base-100"
        >
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span class="gf-badge gf-badge-muted">{{ offering.termCode }}</span>
            <span v-if="offering.className" class="gf-badge gf-badge-info text-[11px]">
              {{ offering.className }}
            </span>
            <span v-else-if="offering.classCode" class="gf-badge gf-badge-info text-[11px]">
              {{ offering.classCode }}
            </span>
            <span v-if="offering.campus" class="text-[12px] text-base-content/55">{{ offering.campus }}</span>
            <span v-if="offering.faculty" class="text-[12px] text-base-content/55">{{ offering.faculty }}</span>
          </div>
          <div v-if="offering.instructors?.length" class="mt-2 flex items-center gap-1.5 text-[13px] text-base-content/75">
            <UsersRound class="h-3.5 w-3.5 text-base-content/45" />
            {{ t('courseDetailPage.instructors') }}：{{ offering.instructors.join('、') }}
          </div>
        </li>
      </ul>
    </section>

      <AISummaryCard :course-id="page.props.course.id" />

      <!-- 相关课程：桌面常显；移动端折叠展开 -->
      <section>
      <div class="mb-3 flex items-center justify-between gap-2">
        <h2 class="text-base font-semibold text-base-content">
          {{ t('courseDetailPage.relatedTitle') }}
        </h2>
        <button
          type="button"
          class="gf-button gf-button-sm gf-button-ghost sm:hidden"
          aria-controls="course-related-panel"
          :aria-expanded="relatedMobileExpanded"
          @click="relatedMobileExpanded = !relatedMobileExpanded"
        >
          <ChevronDown class="h-4 w-4 transition" :class="relatedMobileExpanded ? 'rotate-180' : ''" />
          {{ relatedMobileExpanded ? t('courseDetailPage.relatedCollapse') : t('courseDetailPage.relatedExpand') }}
        </button>
      </div>

      <p v-if="relatedError" class="mb-3 rounded border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">
        {{ relatedError }}
      </p>

      <div v-if="relatedLoading" class="gf-panel">
        <EmptyState :icon="BookOpen" :title="t('common.loading')" loading />
      </div>

      <div
        v-else-if="!relatedError"
        id="course-related-panel"
        :class="['space-y-4 sm:block', relatedMobileExpanded ? 'block' : 'hidden']"
      >
        <div class="gf-panel p-4">
          <h3 class="mb-3 text-sm font-semibold text-base-content">
            {{ t('courseDetailPage.relatedTeacherCoursesTitle') }}
          </h3>
          <EmptyState
            v-if="!related?.teacherOtherCourses.length"
            :icon="BookOpen"
            :title="t('courseDetailPage.relatedEmpty')"
          />
          <ul v-else class="space-y-2">
            <li v-for="item in related.teacherOtherCourses" :key="item.id">
              <a
                :href="`/courses/${item.id}`"
                class="flex items-center justify-between gap-3 rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 px-3 py-2 transition hover:bg-base-200/70"
              >
                <span class="min-w-0">
                  <span class="block truncate text-sm font-medium text-base-content">{{ item.name }}</span>
                  <span class="block truncate text-[12px] text-base-content/50">
                    {{ item.primaryCode }}<template v-if="item.instructors?.length"> · {{ item.instructors.join('、') }}</template>
                  </span>
                </span>
                <span class="shrink-0 text-right">
                  <span class="block text-sm font-semibold tabular-nums text-warning">{{ formatRating(item.ratingAvg) }}</span>
                  <span class="block text-[11px] tabular-nums text-base-content/45">
                    {{ t('courseDetailPage.relatedReviews', { count: item.reviewCount }, item.reviewCount) }}
                  </span>
                </span>
              </a>
            </li>
          </ul>
        </div>

        <div class="gf-panel p-4">
          <h3 class="mb-3 text-sm font-semibold text-base-content">
            {{ t('courseDetailPage.relatedOtherTeachersTitle') }}
          </h3>
          <EmptyState
            v-if="!related?.sameCourseOtherTeachers.length"
            :icon="UsersRound"
            :title="t('courseDetailPage.relatedEmpty')"
          />
          <ul v-else class="space-y-2">
            <li
              v-for="item in related.sameCourseOtherTeachers"
              :key="item.id"
            >
              <a
                :href="`/courses/${item.id}`"
                class="flex items-center justify-between gap-3 rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 px-3 py-2 transition hover:bg-base-200/70"
              >
                <span class="min-w-0">
                  <span class="block truncate text-sm font-medium text-base-content">{{ item.name }}</span>
                  <span class="block truncate text-[12px] text-base-content/50">
                    {{ item.primaryCode }}<template v-if="item.teacherName"> · {{ item.teacherName }}</template>
                  </span>
                </span>
                <span class="shrink-0 text-right">
                  <span class="block text-sm font-semibold tabular-nums text-warning">{{ formatRating(item.ratingAvg) }}</span>
                  <span class="block text-[11px] tabular-nums text-base-content/45">
                    {{ t('courseDetailPage.relatedReviews', { count: item.reviewCount }, item.reviewCount) }}
                  </span>
                </span>
              </a>
            </li>
          </ul>
        </div>
      </div>
      </section>

      </div>

    <section class="min-w-0 xl:order-1">
      <div class="mb-3 flex items-center justify-between gap-2">
        <h2 class="text-base font-semibold text-base-content">
          {{ t('courseDetailPage.reviewsTitle') }}
          <span v-if="reviewTotal" class="ml-1 text-[13px] font-normal text-base-content/45">{{ reviewTotal }}</span>
        </h2>
        <button
          v-if="page.layout.viewer.isAuthenticated && props.course.offerings?.length"
          type="button"
          class="gf-button gf-button-sm gf-button-primary"
          @click="openCreateForm"
        >
          <MessageSquareText class="h-4 w-4" />
          {{ t('courseDetailPage.writeReview') }}
        </button>
        <a
          v-else-if="!page.layout.viewer.isAuthenticated"
          :href="loginHref"
          class="gf-button gf-button-sm gf-button-outline"
        >
          {{ t('courseDetailPage.loginToReview') }}
        </a>
      </div>

      <p v-if="reviewError" class="mb-3 rounded border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">
        {{ reviewError }}
      </p>

      <!-- 写评 / 编辑表单 -->
      <Transition name="gf-local-expand">
      <form
        v-if="formVisible"
        class="mb-4 rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 sm:bg-base-100"
        @submit.prevent="submitForm"
      >
        <h3 class="mb-3 text-sm font-semibold text-base-content">
          {{ editingReviewId ? t('courseDetailPage.editReviewTitle') : t('courseDetailPage.writeReviewTitle') }}
        </h3>

        <div class="space-y-3">
          <fieldset>
            <legend class="mb-1.5 text-[13px] text-base-content/70">{{ t('courseDetailPage.selectOffering') }}</legend>
            <div class="max-h-44 space-y-1.5 overflow-y-auto pr-1">
              <label
                v-for="offering in props.course.offerings"
                :key="offering.id"
                class="flex cursor-pointer items-center gap-2 rounded border border-line/60 bg-base-100 px-3 py-2 text-[13px] text-base-content/75 transition has-[:checked]:border-primary/40 has-[:checked]:bg-info/10"
              >
                <input
                  v-model="formOfferingId"
                  type="radio"
                  name="review-offering"
                  class="radio radio-sm"
                  :value="offering.id"
                />
                <span class="min-w-0 truncate">{{ offeringLabel(offering.id) }}</span>
              </label>
            </div>
          </fieldset>

          <div>
            <span class="mb-1.5 block text-[13px] text-base-content/70">{{ t('courseDetailPage.rating') }}</span>
            <div class="flex items-center gap-1">
              <button
                v-for="star in 5"
                :key="star"
                type="button"
                class="rounded p-0.5 transition hover:scale-110"
                :aria-label="`${star} ${t('courseDetailPage.stars')}`"
                @click="formRating = star"
              >
                <Star
                  class="h-6 w-6"
                  :class="formRating >= star ? 'fill-warning text-warning' : 'text-base-content/25'"
                />
              </button>
              <span v-if="formRating" class="ml-2 text-sm font-semibold tabular-nums text-base-content/70">{{ formRating }}.0</span>
            </div>
          </div>

          <div>
            <label class="mb-1.5 block text-[13px] text-base-content/70" for="review-content">
              {{ t('courseDetailPage.content') }}
            </label>
            <textarea
              id="review-content"
              v-model="formContent"
              class="gf-textarea min-h-28"
              maxlength="50000"
              :placeholder="t('courseDetailPage.contentPlaceholder')"
            />
          </div>

          <div>
            <span class="mb-1.5 block text-[13px] text-base-content/70">{{ t('courseDetailPage.template') }}</span>
            <div class="flex flex-wrap items-center gap-2">
              <button
                type="button"
                class="gf-button gf-button-sm gf-button-ghost"
                @click="templateSelectorOpen = true"
              >
                <FileText class="h-4 w-4" />
                {{ formTemplateId ? t('courseDetailPage.templateChange') : t('courseDetailPage.chooseTemplate') }}
              </button>
              <span v-if="formTemplateId" class="gf-badge gf-badge-muted text-[11px]">
                {{ templateName(formTemplateId) }}
              </span>
            </div>
          </div>

          <label class="flex cursor-pointer items-center gap-2 text-[13px] text-base-content/75">
            <input v-model="formAnonymous" type="checkbox" class="checkbox checkbox-sm" />
            {{ t('courseDetailPage.anonymous') }}
          </label>
        </div>

        <p v-if="formError" class="mt-3 text-sm text-error">{{ formError }}</p>

        <div class="mt-4 flex justify-end gap-2">
          <button
            type="button"
            class="gf-button gf-button-md gf-button-muted"
            :disabled="formSubmitting"
            @click="cancelForm"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            class="gf-button gf-button-md gf-button-primary"
            :disabled="formSubmitting"
          >
            <Loader2 v-if="formSubmitting" class="h-4 w-4 animate-spin" />
            {{ formSubmitting ? t('common.loadingShort') : t('courseDetailPage.submitReview') }}
          </button>
        </div>
      </form>
      </Transition>

      <!-- 评价列表 -->
      <div v-if="reviewLoading" class="gf-panel">
        <EmptyState :icon="MessageSquareText" :title="t('courseDetailPage.reviewsLoading')" loading />
      </div>
      <EmptyState
        v-else-if="reviewLoaded && !reviews.length"
        class="gf-panel"
        :icon="MessageSquareText"
        :title="t('courseDetailPage.reviewsEmpty')"
        :description="t('courseDetailPage.reviewsEmptyDescription')"
      />
      <ul v-else class="space-y-3">
        <li
          v-for="review in reviews"
          :key="review.id"
          class="rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 sm:bg-base-100"
        >
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span class="gf-badge gf-badge-muted text-[11px]">{{ offeringLabel(review.offeringId) }}</span>
            <span v-if="review.rating" class="inline-flex items-center gap-0.5">
              <Star
                v-for="star in 5"
                :key="star"
                class="h-3.5 w-3.5"
                :class="review.rating && review.rating >= star ? 'fill-warning text-warning' : 'text-base-content/20'"
              />
            </span>
            <span v-else class="text-[12px] text-base-content/45">{{ t('courseDetailPage.noRating') }}</span>
            <span class="text-[12px] text-base-content/55">{{ authorLabel(review.author) }}</span>
            <time class="text-[12px] tabular-nums text-base-content/45">{{ formatDateTime(review.createdAt) }}</time>
          </div>

          <div
            v-if="review.contentHtml"
            v-code-highlight
            v-math-render
            class="gf-prose gf-prose-post mt-2 text-[14px] leading-6"
            v-html="review.contentHtml"
          />

          <div class="mt-3 flex flex-wrap items-center gap-2">
            <button
              type="button"
              class="gf-button gf-button-sm"
              :class="review.viewer.isHelpful ? 'gf-button-primary' : 'gf-button-ghost'"
              :disabled="helpfulBusyIds.includes(review.id)"
              @click="toggleHelpful(review)"
            >
              <ThumbsUp class="h-3.5 w-3.5" />
              <span v-if="review.helpfulCount" class="tabular-nums">{{ review.helpfulCount }}</span>
              {{ t('courseDetailPage.helpful') }}
            </button>
            <button
              v-if="review.viewer.canEdit"
              type="button"
              class="gf-button gf-button-sm gf-button-ghost"
              @click="startEdit(review)"
            >
              <Pencil class="h-3.5 w-3.5" />
              {{ t('courseDetailPage.edit') }}
            </button>
            <button
              v-if="review.viewer.canDelete"
              type="button"
              class="gf-button gf-button-sm gf-button-ghost text-error"
              @click="askRemoveReview(review)"
            >
              <Trash2 class="h-3.5 w-3.5" />
              {{ t('courseDetailPage.delete') }}
            </button>
            <button
              v-if="page.layout.viewer.isAuthenticated && !review.viewer.canEdit"
              type="button"
              class="gf-button gf-button-sm gf-button-ghost ml-auto"
              @click="openReport(review)"
            >
              <Flag class="h-3.5 w-3.5" />
              {{ t('courseDetailPage.report') }}
            </button>
          </div>
        </li>
      </ul>
      <InfiniteScrollFooter
        v-if="reviewLoaded && (reviews.length || reviewNextCursor)"
        :has-next="!!reviewNextCursor"
        :loading="reviewLoadingMore"
        :error="reviewLoadMoreError"
        :has-items="reviews.length > 0"
        :load-label="t('courseDetailPage.loadMoreReviews')"
        @load-more="loadMoreReviews"
      />
    </section>
    </div>

    <!-- 举报弹窗 -->
    <Teleport to="body">
      <Transition name="gf-modal">
      <div
        v-if="pendingReport"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        @click.self="closeReport"
      >
        <div class="w-full max-w-md rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <div class="flex items-start justify-between gap-3">
            <h2 class="text-base font-bold text-base-content">{{ t('courseDetailPage.reportTitle') }}</h2>
            <button
              type="button"
              class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75"
              @click="closeReport"
            >
              <X class="h-4 w-4" />
            </button>
          </div>

          <div class="mt-4 space-y-3">
            <label v-for="reason in reportReasons" :key="reason" class="flex cursor-pointer items-center gap-2 text-sm text-base-content/75">
              <input v-model="reportReason" class="radio radio-sm" type="radio" name="course-review-report-reason" :value="reason" />
              <span>{{ t(`courseDetailPage.reportReasons.${reason}`) }}</span>
            </label>
            <textarea
              v-model="reportNote"
              class="gf-textarea min-h-24"
              maxlength="300"
              :placeholder="t('courseDetailPage.reportNotePlaceholder')"
            />
          </div>

          <p v-if="reportError" class="mt-3 text-sm text-error">{{ reportError }}</p>

          <div class="mt-4 flex justify-end gap-2">
            <button
              type="button"
              class="gf-button gf-button-md gf-button-muted"
              :disabled="reportSubmitting"
              @click="closeReport"
            >
              {{ t('common.cancel') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-md gf-button-primary"
              :disabled="reportSubmitting"
              @click="submitReport"
            >
              <Loader2 v-if="reportSubmitting" class="h-4 w-4 animate-spin" />
              {{ reportSubmitting ? t('common.loadingShort') : t('courseDetailPage.submitReport') }}
            </button>
          </div>
        </div>
      </div>
      </Transition>
    </Teleport>

    <!-- 删除确认 Dialog（受控，替代 window.confirm） -->
    <Teleport to="body">
      <Transition name="gf-modal">
      <div
        v-if="pendingDelete"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="alertdialog"
        aria-modal="true"
        @click.self="cancelRemoveReview"
      >
        <div class="w-full max-w-sm rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <div class="flex items-start justify-between gap-3">
            <h2 class="text-base font-bold text-base-content">{{ t('courseDetailPage.confirmDeleteTitle') }}</h2>
            <button
              type="button"
              class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75"
              :aria-label="t('common.close')"
              @click="cancelRemoveReview"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
          <p class="mt-3 text-sm text-base-content/75">{{ t('courseDetailPage.confirmDeleteReview') }}</p>
          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-md gf-button-muted" :disabled="deleting" @click="cancelRemoveReview">
              {{ t('common.cancel') }}
            </button>
            <button type="button" class="gf-button gf-button-md gf-button-danger" :disabled="deleting" @click="confirmRemoveReview">
              <Loader2 v-if="deleting" class="h-4 w-4 animate-spin" />
              <Trash2 v-else class="h-4 w-4" />
              {{ t('courseDetailPage.delete') }}
            </button>
          </div>
        </div>
      </div>
      </Transition>
    </Teleport>

    <!-- 写评模板选择器 -->
    <CourseReviewTemplateSelector
      :open="templateSelectorOpen"
      @close="templateSelectorOpen = false"
      @select="applyTemplate($event.id, $event.content)"
    />
  </div>
</template>
