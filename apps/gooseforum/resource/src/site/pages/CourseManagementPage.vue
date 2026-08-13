<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen, Loader2, Pencil, Plus, RefreshCw, Search, Trash2, X } from '@lucide/vue'
import {
  createAdminCourse,
  deleteAdminCourse,
  deleteAdminReview,
  fetchAdminCourses,
  fetchAdminReviews,
  moderationCourseReviewStatus,
  rebuildCourseStats,
  updateAdminCourse,
  updateAdminReview,
  type AdminCourseItem,
  type AdminCourseUpdateInput,
  type AdminReviewItem,
} from '@/runtime/api'
import EmptyState from '@/site/components/EmptyState.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import type { CourseManagementPageProps, LayoutPayload } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: CourseManagementPageProps
}>()
const { t } = useI18n()

const activeTab = ref<'courses' | 'reviews'>('courses')
const pageError = ref('')

// ---- 课程管理 ----
const courseKeyword = ref('')
const coursePage = ref(1)
const courseItems = ref<AdminCourseItem[]>([])
const courseHasNext = ref(true)
const courseLoading = ref(false)
const courseLoaded = ref(false)
const courseBusyIds = ref<number[]>([])

async function loadCourses(reset = false) {
  if (courseLoading.value) return
  courseLoading.value = true
  pageError.value = ''
  try {
    const payload = await fetchAdminCourses(courseKeyword.value.trim(), '', reset ? 1 : coursePage.value, 20)
    courseItems.value = reset ? payload.list : [...courseItems.value, ...payload.list]
    coursePage.value = payload.page + 1
    courseHasNext.value = payload.hasNext
    courseLoaded.value = true
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : t('api.adminCourseListFailed')
  } finally {
    courseLoading.value = false
  }
}

function searchCourses() {
  coursePage.value = 1
  courseItems.value = []
  courseHasNext.value = true
  courseLoaded.value = false
  void loadCourses(true)
}

function courseBusy(id: number) {
  return courseBusyIds.value.includes(id)
}

// ---- 课程新增/编辑表单 ----
interface CourseForm {
  primaryCode: string
  name: string
  department: string
  credit: string
  aliases: string
  instructors: string
}

const courseFormOpen = ref(false)
const courseFormEditingId = ref(0)
const courseForm = ref<CourseForm>(emptyCourseForm())
const courseFormOriginal = ref<CourseForm>(emptyCourseForm())
const courseFormSubmitting = ref(false)
const courseFormError = ref('')

function emptyCourseForm(): CourseForm {
  return { primaryCode: '', name: '', department: '', credit: '', aliases: '', instructors: '' }
}

function openCreateCourse() {
  courseFormEditingId.value = 0
  courseForm.value = emptyCourseForm()
  courseFormOriginal.value = emptyCourseForm()
  courseFormError.value = ''
  courseFormOpen.value = true
}

function openEditCourse(item: AdminCourseItem) {
  courseFormEditingId.value = item.id
  courseForm.value = {
    primaryCode: item.primaryCode,
    name: item.name,
    department: item.department,
    credit: formatCredit(item.creditX10),
    aliases: item.aliases.join(', '),
    instructors: item.instructors.join(', '),
  }
  courseFormOriginal.value = { ...courseForm.value }
  courseFormError.value = ''
  courseFormOpen.value = true
}

function closeCourseForm() {
  courseFormOpen.value = false
  courseFormSubmitting.value = false
}

function parseCommaList(text: string): string[] {
  return text.split(/[,，、\s]+/).map((s) => s.trim()).filter((s) => s.length > 0)
}

function parseCredit(text: string): number | null {
  const value = Number.parseFloat(text)
  if (!Number.isFinite(value) || value < 0) return null
  return Math.round(value * 10)
}

async function submitCourseForm() {
  const form = courseForm.value
  if (!form.primaryCode.trim()) {
    courseFormError.value = t('courseManagement.courseForm.codeRequired')
    return
  }
  if (!form.name.trim()) {
    courseFormError.value = t('courseManagement.courseForm.nameRequired')
    return
  }
  const creditX10 = parseCredit(form.credit)
  if (creditX10 === null) {
    courseFormError.value = t('courseManagement.courseForm.creditInvalid')
    return
  }
  courseFormSubmitting.value = true
  courseFormError.value = ''
  try {
    if (courseFormEditingId.value === 0) {
      await createAdminCourse({
        primaryCode: form.primaryCode.trim(),
        name: form.name.trim(),
        department: form.department.trim(),
        creditX10,
        aliases: parseCommaList(form.aliases),
        instructors: parseCommaList(form.instructors),
      })
    } else {
      const id = courseFormEditingId.value
      const original = courseFormOriginal.value
      const payload: AdminCourseUpdateInput = {}
      if (form.primaryCode.trim() !== original.primaryCode) payload.primaryCode = form.primaryCode.trim()
      if (form.name.trim() !== original.name) payload.name = form.name.trim()
      if (form.department.trim() !== original.department) payload.department = form.department.trim()
      if (parseCredit(form.credit) !== parseCredit(original.credit)) payload.creditX10 = creditX10
      if (form.aliases !== original.aliases) payload.aliases = parseCommaList(form.aliases)
      if (form.instructors !== original.instructors) payload.instructors = parseCommaList(form.instructors)
      await updateAdminCourse(id, payload)
    }
    closeCourseForm()
    await loadCourses(true)
  } catch (error) {
    courseFormError.value = error instanceof Error ? error.message : t('api.adminCourseUpdateFailed')
  } finally {
    courseFormSubmitting.value = false
  }
}

// ---- 删除课程（级联确认） ----
const courseDeleteTarget = ref<AdminCourseItem | null>(null)
const courseDeleteSubmitting = ref(false)

function openDeleteCourse(item: AdminCourseItem) {
  courseDeleteTarget.value = item
}

function closeDeleteCourse() {
  courseDeleteTarget.value = null
}

async function confirmDeleteCourse() {
  if (!courseDeleteTarget.value) return
  courseDeleteSubmitting.value = true
  pageError.value = ''
  try {
    await deleteAdminCourse(courseDeleteTarget.value.id)
    closeDeleteCourse()
    await loadCourses(true)
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : t('api.adminCourseDeleteFailed')
  } finally {
    courseDeleteSubmitting.value = false
  }
}

// ---- 评价管理 ----
const reviewKeyword = ref('')
const reviewStatus = ref(-1)
const reviewCursor = ref(0)
const reviewItems = ref<AdminReviewItem[]>([])
const reviewHasNext = ref(true)
const reviewLoading = ref(false)
const reviewLoaded = ref(false)
const reviewBusyIds = ref<number[]>([])

async function loadReviews(reset = false) {
  if (reviewLoading.value) return
  reviewLoading.value = true
  pageError.value = ''
  try {
    const payload = await fetchAdminReviews(reviewKeyword.value.trim(), reviewStatus.value, reset ? 0 : reviewCursor.value, 20)
    reviewItems.value = reset ? payload.items : [...reviewItems.value, ...payload.items]
    reviewCursor.value = payload.nextCursor
    reviewHasNext.value = payload.hasNext
    reviewLoaded.value = true
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : t('api.adminReviewListFailed')
  } finally {
    reviewLoading.value = false
  }
}

function searchReviews() {
  reviewCursor.value = 0
  reviewItems.value = []
  reviewHasNext.value = true
  reviewLoaded.value = false
  void loadReviews(true)
}

function switchReviewStatus(status: number) {
  if (reviewStatus.value === status) return
  reviewStatus.value = status
  reviewCursor.value = 0
  reviewItems.value = []
  reviewHasNext.value = true
  reviewLoaded.value = false
  void loadReviews(true)
}

function reviewBusy(id: number) {
  return reviewBusyIds.value.includes(id)
}

function reviewStatusLabel(status: number): string {
  if (status === 1) return t('courseManagement.statusHidden')
  if (status === 2) return t('courseManagement.reviewStatusTabs.deleted')
  return t('courseManagement.statusVisible')
}

async function toggleReviewVisibility(item: AdminReviewItem) {
  if (reviewBusy(item.id)) return
  reviewBusyIds.value = [...reviewBusyIds.value, item.id]
  pageError.value = ''
  try {
    await moderationCourseReviewStatus(item.id, item.status === 1 ? 'show' : 'hide')
    await loadReviews(true)
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : t('api.moderationActionFailed')
  } finally {
    reviewBusyIds.value = reviewBusyIds.value.filter((id) => id !== item.id)
  }
}

// ---- 编辑评价 ----
const reviewEditTarget = ref<AdminReviewItem | null>(null)
const reviewEditRating = ref(5)
const reviewEditContent = ref('')
const reviewEditSubmitting = ref(false)
const reviewEditError = ref('')

function openEditReview(item: AdminReviewItem) {
  reviewEditTarget.value = item
  reviewEditRating.value = item.rating ?? 5
  reviewEditContent.value = item.content
  reviewEditError.value = ''
}

function closeEditReview() {
  reviewEditTarget.value = null
  reviewEditSubmitting.value = false
}

async function submitEditReview() {
  if (!reviewEditTarget.value) return
  reviewEditSubmitting.value = true
  reviewEditError.value = ''
  try {
    await updateAdminReview(reviewEditTarget.value.id, {
      rating: reviewEditRating.value,
      content: reviewEditContent.value,
    })
    closeEditReview()
    await loadReviews(true)
  } catch (error) {
    reviewEditError.value = error instanceof Error ? error.message : t('api.adminReviewUpdateFailed')
  } finally {
    reviewEditSubmitting.value = false
  }
}

// ---- 删除评价 ----
const reviewDeleteTarget = ref<AdminReviewItem | null>(null)
const reviewDeleteSubmitting = ref(false)

function openDeleteReview(item: AdminReviewItem) {
  reviewDeleteTarget.value = item
}

function closeDeleteReview() {
  reviewDeleteTarget.value = null
}

async function confirmDeleteReview() {
  if (!reviewDeleteTarget.value) return
  reviewDeleteSubmitting.value = true
  pageError.value = ''
  try {
    await deleteAdminReview(reviewDeleteTarget.value.id)
    closeDeleteReview()
    await loadReviews(true)
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : t('api.adminReviewDeleteFailed')
  } finally {
    reviewDeleteSubmitting.value = false
  }
}

// ---- 统计重建 ----
const rebuildSubmitting = ref(false)
const rebuildConfirmOpen = ref(false)
const rebuildMessage = ref('')

function openRebuild() {
  rebuildConfirmOpen.value = true
}

function closeRebuild() {
  rebuildConfirmOpen.value = false
}

async function confirmRebuild() {
  rebuildSubmitting.value = true
  rebuildMessage.value = ''
  try {
    await rebuildCourseStats()
    rebuildMessage.value = t('courseManagement.rebuildQueued')
    closeRebuild()
  } catch (error) {
    rebuildMessage.value = error instanceof Error ? error.message : t('api.adminCourseStatsRebuildFailed')
  } finally {
    rebuildSubmitting.value = false
  }
}

function formatCredit(creditX10: number): string {
  return (creditX10 / 10).toString()
}

onMounted(() => {
  void loadCourses(true)
})
</script>

<template>
  <main class="min-w-0 pb-8">
    <PageHeader
      :title="t('courseManagement.title')"
      :description="t('courseManagement.description')"
      compact
      class="border-b-0 !mb-2 sm:!mb-2 !pb-2 sm:!pb-2"
    />

    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-1 rounded-[var(--gf-radius-box)] bg-base-200/60 p-1">
        <button
          type="button"
          class="gf-tab"
          :class="activeTab === 'courses' ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
          @click="activeTab = 'courses'"
        >
          {{ t('courseManagement.tabs.courses') }}
        </button>
        <button
          type="button"
          class="gf-tab"
          :class="activeTab === 'reviews' ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
          @click="activeTab = 'reviews'"
        >
          {{ t('courseManagement.tabs.reviews') }}
        </button>
      </div>
      <button
        type="button"
        class="gf-button gf-button-sm gf-button-outline shrink-0"
        :disabled="rebuildSubmitting"
        @click="openRebuild"
      >
        <RefreshCw class="h-4 w-4" />
        {{ t('courseManagement.rebuild') }}
      </button>
    </div>

    <p v-if="pageError" class="mb-3 rounded border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">{{ pageError }}</p>
    <p v-if="rebuildMessage" class="mb-3 rounded border border-success/25 bg-success/10 px-3 py-2 text-sm text-base-content/75">{{ rebuildMessage }}</p>

    <!-- 课程管理 tab -->
    <section v-if="activeTab === 'courses'" class="space-y-3">
      <div class="flex flex-wrap items-center gap-2">
        <div class="relative min-w-0 flex-1">
          <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
          <input
            v-model="courseKeyword"
            class="gf-input gf-input-md w-full pl-9"
            :placeholder="t('courseManagement.searchPlaceholder')"
            @keyup.enter="searchCourses"
          />
        </div>
        <button type="button" class="gf-button gf-button-sm gf-button-ghost" @click="searchCourses">
          {{ t('common.search') }}
        </button>
        <button type="button" class="gf-button gf-button-sm gf-button-primary" @click="openCreateCourse">
          <Plus class="h-4 w-4" />
          {{ t('courseManagement.add') }}
        </button>
      </div>

      <div class="gf-card overflow-hidden">
        <div v-if="courseItems.length" class="divide-y divide-line">
          <article
            v-for="item in courseItems"
            :key="item.id"
            class="flex flex-col gap-2 px-3 py-3 transition hover:bg-base-200/70 lg:grid lg:grid-cols-[130px_minmax(0,1fr)_140px_90px_130px_100px] lg:items-center lg:gap-4"
          >
            <div class="flex items-center gap-2">
              <span class="font-mono text-[13px] font-semibold text-base-content/80">{{ item.primaryCode }}</span>
              <span v-if="item.status === 1" class="gf-badge gf-badge-ghost text-[11px]">{{ t('courseManagement.statusHidden') }}</span>
            </div>
            <div class="min-w-0">
              <p class="truncate text-[15px] font-medium text-base-content">{{ item.name }}</p>
              <p v-if="item.instructors.length" class="truncate text-[12px] text-base-content/50">{{ item.instructors.join('、') }}</p>
            </div>
            <div class="truncate text-[13px] text-base-content/55">{{ item.department || '—' }}</div>
            <div class="text-[13px] tabular-nums text-base-content/55">{{ formatCredit(item.creditX10) }}</div>
            <div class="text-[13px] tabular-nums text-base-content/55">
              {{ item.reviewCount }}<template v-if="item.ratingAvg !== undefined"> · {{ item.ratingAvg.toFixed(1) }}★</template>
            </div>
            <div class="flex items-center gap-1.5 lg:justify-end">
              <button
                type="button"
                class="gf-button gf-button-sm gf-button-outline shrink-0"
                :disabled="courseBusy(item.id)"
                @click="openEditCourse(item)"
              >
                <Pencil class="h-4 w-4" />
                {{ t('courseManagement.edit') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-sm gf-button-danger shrink-0"
                :disabled="courseBusy(item.id)"
                @click="openDeleteCourse(item)"
              >
                <Trash2 class="h-4 w-4" />
                {{ t('courseManagement.delete') }}
              </button>
            </div>
          </article>
        </div>

        <EmptyState
          v-else-if="courseLoading"
          :icon="BookOpen"
          :title="t('courseManagement.loading')"
          loading
        />
        <EmptyState
          v-else
          :icon="BookOpen"
          :title="t('courseManagement.coursesEmpty')"
          :description="t('courseManagement.coursesEmptyDesc')"
        />

        <footer v-if="courseLoaded && (courseItems.length || courseHasNext)" class="border-t border-line px-4 py-3 text-center">
          <button
            v-if="courseHasNext"
            type="button"
            class="gf-button gf-button-sm gf-button-ghost"
            :disabled="courseLoading"
            @click="loadCourses(false)"
          >
            {{ courseLoading ? t('courseManagement.loading') : t('courseManagement.loadMore') }}
          </button>
          <span v-else-if="courseItems.length" class="text-xs text-base-content/45">{{ t('courseManagement.noMore') }}</span>
        </footer>
      </div>
    </section>

    <!-- 评价管理 tab -->
    <section v-else class="space-y-3">
      <div class="flex flex-wrap items-center gap-2">
        <div class="relative min-w-0 flex-1">
          <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
          <input
            v-model="reviewKeyword"
            class="gf-input gf-input-md w-full pl-9"
            :placeholder="t('courseManagement.reviewSearchPlaceholder')"
            @keyup.enter="searchReviews"
          />
        </div>
        <button type="button" class="gf-button gf-button-sm gf-button-ghost" @click="searchReviews">
          {{ t('common.search') }}
        </button>
      </div>

      <div class="gf-card overflow-hidden">
        <div class="flex items-center gap-1 border-b border-line bg-base-200/60 p-2">
          <button
            v-for="status in [
              { key: -1, label: t('courseManagement.reviewStatusTabs.all') },
              { key: 0, label: t('courseManagement.reviewStatusTabs.visible') },
              { key: 1, label: t('courseManagement.reviewStatusTabs.hidden') },
              { key: 2, label: t('courseManagement.reviewStatusTabs.deleted') },
            ]"
            :key="status.key"
            type="button"
            class="gf-tab"
            :class="reviewStatus === status.key ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
            @click="switchReviewStatus(status.key)"
          >
            {{ status.label }}
          </button>
        </div>

        <div v-if="reviewItems.length" class="divide-y divide-line">
          <article
            v-for="item in reviewItems"
            :key="item.id"
            class="flex flex-col gap-2 px-3 py-3 transition hover:bg-base-200/70"
          >
            <div class="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-xs text-base-content/50">
              <span class="shrink-0 font-semibold text-base-content/70">{{ t('courseManagement.reviewCourseLabel') }}</span>
              <span class="font-mono">{{ item.courseCode }}</span>
              <span class="min-w-0 truncate text-base-content/65">{{ item.courseName }}</span>
              <span class="shrink-0">#{{ item.id }}</span>
              <span class="gf-badge gf-badge-ghost shrink-0 text-[11px]">{{ reviewStatusLabel(item.status) }}</span>
              <span v-if="item.rating" class="shrink-0 text-warning">{{ '★'.repeat(item.rating) }}</span>
            </div>
            <p class="line-clamp-3 whitespace-pre-wrap text-[13px] leading-5 text-base-content/70">{{ item.content }}</p>
            <div class="flex items-center gap-2">
              <button
                v-if="item.status !== 2"
                type="button"
                class="gf-button gf-button-sm gf-button-outline shrink-0"
                :disabled="reviewBusy(item.id)"
                @click="toggleReviewVisibility(item)"
              >
                {{ item.status === 1 ? t('courseManagement.show') : t('courseManagement.hide') }}
              </button>
              <button
                v-if="item.status !== 2"
                type="button"
                class="gf-button gf-button-sm gf-button-ghost shrink-0"
                :disabled="reviewBusy(item.id)"
                @click="openEditReview(item)"
              >
                <Pencil class="h-4 w-4" />
                {{ t('courseManagement.edit') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-sm gf-button-danger shrink-0"
                :disabled="reviewBusy(item.id)"
                @click="openDeleteReview(item)"
              >
                <Trash2 class="h-4 w-4" />
                {{ t('courseManagement.delete') }}
              </button>
            </div>
          </article>
        </div>

        <EmptyState
          v-else-if="reviewLoading"
          :icon="BookOpen"
          :title="t('courseManagement.loading')"
          loading
        />
        <EmptyState
          v-else
          :icon="BookOpen"
          :title="t('courseManagement.reviewsEmpty')"
          :description="t('courseManagement.reviewsEmptyDesc')"
        />

        <footer v-if="reviewLoaded && (reviewItems.length || reviewHasNext)" class="border-t border-line px-4 py-3 text-center">
          <button
            v-if="reviewHasNext"
            type="button"
            class="gf-button gf-button-sm gf-button-ghost"
            :disabled="reviewLoading"
            @click="loadReviews(false)"
          >
            {{ reviewLoading ? t('courseManagement.loading') : t('courseManagement.loadMore') }}
          </button>
          <span v-else-if="reviewItems.length" class="text-xs text-base-content/45">{{ t('courseManagement.noMore') }}</span>
        </footer>
      </div>
    </section>

    <!-- 课程表单弹窗 -->
    <Teleport to="body">
      <div
        v-if="courseFormOpen"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        @click.self="closeCourseForm"
      >
        <div class="w-full max-w-lg rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <div class="flex items-start justify-between gap-3">
            <h2 class="text-base font-bold text-base-content">
              {{ courseFormEditingId === 0 ? t('courseManagement.courseForm.createTitle') : t('courseManagement.courseForm.editTitle') }}
            </h2>
            <button
              type="button"
              class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75"
              @click="closeCourseForm"
            >
              <X class="h-4 w-4" />
            </button>
          </div>

          <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.courseForm.code') }}</span>
              <input v-model="courseForm.primaryCode" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.courseForm.codePlaceholder')" />
            </label>
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.courseForm.name') }}</span>
              <input v-model="courseForm.name" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.courseForm.namePlaceholder')" />
            </label>
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.courseForm.department') }}</span>
              <input v-model="courseForm.department" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.courseForm.departmentPlaceholder')" />
            </label>
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.courseForm.credit') }}</span>
              <input v-model="courseForm.credit" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.courseForm.creditPlaceholder')" />
            </label>
            <label class="block sm:col-span-2">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.courseForm.aliases') }}</span>
              <input v-model="courseForm.aliases" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.courseForm.aliasesPlaceholder')" />
            </label>
            <label class="block sm:col-span-2">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.courseForm.instructors') }}</span>
              <input v-model="courseForm.instructors" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.courseForm.instructorsPlaceholder')" />
            </label>
          </div>

          <p v-if="courseFormError" class="mt-3 text-sm text-error">{{ courseFormError }}</p>

          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-md gf-button-muted" :disabled="courseFormSubmitting" @click="closeCourseForm">
              {{ t('courseManagement.cancel') }}
            </button>
            <button type="button" class="gf-button gf-button-md gf-button-primary" :disabled="courseFormSubmitting" @click="submitCourseForm">
              <Loader2 v-if="courseFormSubmitting" class="h-4 w-4 animate-spin" />
              {{ courseFormSubmitting ? t('common.loadingShort') : t('courseManagement.save') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 删除课程确认弹窗 -->
    <Teleport to="body">
      <div
        v-if="courseDeleteTarget"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        @click.self="closeDeleteCourse"
      >
        <div class="w-full max-w-md rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <h2 class="text-base font-bold text-base-content">{{ t('courseManagement.confirmDelete') }}</h2>
          <p class="mt-2 text-[13px] leading-5 text-base-content/60">
            {{ t('courseManagement.deleteCourseConfirm', { name: courseDeleteTarget.name, count: courseDeleteTarget.reviewCount }) }}
          </p>
          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-md gf-button-muted" :disabled="courseDeleteSubmitting" @click="closeDeleteCourse">
              {{ t('courseManagement.cancel') }}
            </button>
            <button type="button" class="gf-button gf-button-md gf-button-danger" :disabled="courseDeleteSubmitting" @click="confirmDeleteCourse">
              <Loader2 v-if="courseDeleteSubmitting" class="h-4 w-4 animate-spin" />
              {{ courseDeleteSubmitting ? t('common.loadingShort') : t('courseManagement.confirmDelete') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 编辑评价弹窗 -->
    <Teleport to="body">
      <div
        v-if="reviewEditTarget"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        @click.self="closeEditReview"
      >
        <div class="w-full max-w-lg rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <div class="flex items-start justify-between gap-3">
            <h2 class="text-base font-bold text-base-content">{{ t('courseManagement.reviewEditTitle') }}</h2>
            <button type="button" class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75" @click="closeEditReview">
              <X class="h-4 w-4" />
            </button>
          </div>
          <div class="mt-4 space-y-3">
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.reviewRating') }}</span>
              <select v-model.number="reviewEditRating" class="gf-input gf-input-md w-full">
                <option v-for="n in 5" :key="n" :value="n">{{ n }}</option>
              </select>
            </label>
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.reviewContent') }}</span>
              <textarea v-model="reviewEditContent" class="gf-textarea min-h-40 w-full" :placeholder="t('courseManagement.reviewContentPlaceholder')" />
            </label>
          </div>
          <p v-if="reviewEditError" class="mt-3 text-sm text-error">{{ reviewEditError }}</p>
          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-md gf-button-muted" :disabled="reviewEditSubmitting" @click="closeEditReview">
              {{ t('courseManagement.cancel') }}
            </button>
            <button type="button" class="gf-button gf-button-md gf-button-primary" :disabled="reviewEditSubmitting" @click="submitEditReview">
              <Loader2 v-if="reviewEditSubmitting" class="h-4 w-4 animate-spin" />
              {{ reviewEditSubmitting ? t('common.loadingShort') : t('courseManagement.save') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 删除评价确认弹窗 -->
    <Teleport to="body">
      <div
        v-if="reviewDeleteTarget"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        @click.self="closeDeleteReview"
      >
        <div class="w-full max-w-md rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <h2 class="text-base font-bold text-base-content">{{ t('courseManagement.confirmDelete') }}</h2>
          <p class="mt-2 text-[13px] leading-5 text-base-content/60">{{ t('courseManagement.reviewDeleteConfirm') }}</p>
          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-md gf-button-muted" :disabled="reviewDeleteSubmitting" @click="closeDeleteReview">
              {{ t('courseManagement.cancel') }}
            </button>
            <button type="button" class="gf-button gf-button-md gf-button-danger" :disabled="reviewDeleteSubmitting" @click="confirmDeleteReview">
              <Loader2 v-if="reviewDeleteSubmitting" class="h-4 w-4 animate-spin" />
              {{ reviewDeleteSubmitting ? t('common.loadingShort') : t('courseManagement.confirmDelete') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 统计重建确认弹窗 -->
    <Teleport to="body">
      <div
        v-if="rebuildConfirmOpen"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        @click.self="closeRebuild"
      >
        <div class="w-full max-w-md rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <h2 class="text-base font-bold text-base-content">{{ t('courseManagement.rebuild') }}</h2>
          <p class="mt-2 text-[13px] leading-5 text-base-content/60">{{ t('courseManagement.rebuildConfirm') }}</p>
          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-md gf-button-muted" :disabled="rebuildSubmitting" @click="closeRebuild">
              {{ t('courseManagement.cancel') }}
            </button>
            <button type="button" class="gf-button gf-button-md gf-button-primary" :disabled="rebuildSubmitting" @click="confirmRebuild">
              <Loader2 v-if="rebuildSubmitting" class="h-4 w-4 animate-spin" />
              {{ rebuildSubmitting ? t('common.loadingShort') : t('courseManagement.rebuild') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </main>
</template>
