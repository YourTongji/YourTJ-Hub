<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Building2, CalendarDays, Flag, Loader2, MessageSquareText, Pencil, Star, ThumbsUp, Trash2, UsersRound, X } from '@lucide/vue'
import {
  createCourseReview,
  deleteCourseReview,
  listCourseReviews,
  reportCourseReview,
  setReviewHelpful,
  updateCourseReview,
  type ReviewPayload,
} from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
import EmptyState from '@/site/components/EmptyState.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import type { CourseDetailPageProps, LayoutPayload } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: CourseDetailPageProps
}>()
const { t } = useI18n()

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
  return [offering.termCode, offering.campus, offering.instructors?.join('、')].filter(Boolean).join(' · ')
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
const reviewLoading = ref(false)
const reviewError = ref('')
const reviewLoaded = ref(false)
const helpfulBusyIds = ref<number[]>([])

async function loadReviews() {
  reviewLoading.value = true
  reviewError.value = ''
  try {
    reviews.value = await listCourseReviews(page.props.course.id)
  } catch (error) {
    reviewError.value = error instanceof Error ? error.message : t('courseDetailPage.reviewsLoadFailed')
  } finally {
    reviewLoading.value = false
    reviewLoaded.value = true
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

function openCreateForm() {
  editingReviewId.value = null
  formOfferingId.value = page.props.course.offerings?.[0]?.id ?? 0
  formRating.value = 0
  formContent.value = ''
  formAnonymous.value = true
  formError.value = ''
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
  formVisible.value = true
}

function cancelForm() {
  formVisible.value = false
  editingReviewId.value = null
  formError.value = ''
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
      reviews.value.unshift(created)
    }
    formVisible.value = false
    editingReviewId.value = null
  } catch (error) {
    formError.value = error instanceof Error ? error.message : t('courseDetailPage.reviewSaveFailed')
  } finally {
    formSubmitting.value = false
  }
}

async function removeReview(review: ReviewPayload) {
  if (!window.confirm(t('courseDetailPage.confirmDeleteReview'))) return
  try {
    await deleteCourseReview(review.id)
    reviews.value = reviews.value.filter((item) => item.id !== review.id)
  } catch (error) {
    window.alert(error instanceof Error ? error.message : t('courseDetailPage.reviewDeleteFailed'))
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
    window.alert(error instanceof Error ? error.message : t('courseDetailPage.reviewHelpfulFailed'))
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

onMounted(loadReviews)
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

    <section class="mt-6">
      <div class="mb-3 flex items-center justify-between gap-2">
        <h2 class="text-base font-semibold text-base-content">
          {{ t('courseDetailPage.reviewsTitle') }}
          <span v-if="reviews.length" class="ml-1 text-[13px] font-normal text-base-content/45">{{ reviews.length }}</span>
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
              @click="removeReview(review)"
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
    </section>

    <!-- 举报弹窗 -->
    <Teleport to="body">
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
    </Teleport>
  </div>
</template>
