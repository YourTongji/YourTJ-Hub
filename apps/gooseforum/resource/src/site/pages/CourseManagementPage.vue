<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen, Loader2, Pencil, Plus, RefreshCw, Search, Trash2, X } from '@lucide/vue'
import {
  approveCourseRelation,
  createAdminCourse,
  createCourseRelation,
  deleteAdminCourse,
  deleteAdminReview,
  fetchAdminCourses,
  fetchAdminReviews,
  fetchCourseRelations,
  getCourseDetail,
  ignoreCourseRelation,
  mergeCourseRelation,
  moderationCourseReviewStatus,
  resetCourseRelation,
  rebuildCourseStats,
  undoMergeCourseRelation,
  updateAdminCourse,
  updateAdminReview,
  type AdminCourseItem,
  type AdminCourseUpdateInput,
  type AdminReviewItem,
  type CourseMergeResult,
  type CourseRelationItem,
} from '@/runtime/api'
import EmptyState from '@/site/components/EmptyState.vue'
import InfiniteScrollFooter from '@/site/components/InfiniteScrollFooter.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import type { CourseManagementPageProps, LayoutPayload } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: CourseManagementPageProps
}>()
const { t } = useI18n()

const activeTab = ref<'courses' | 'reviews' | 'relations'>('courses')
const pageError = ref('')

// ---- 课程管理 ----
const courseKeyword = ref('')
const coursePage = ref(1)
const courseItems = ref<AdminCourseItem[]>([])
const courseHasNext = ref(true)
const courseLoading = ref(false)
const courseLoaded = ref(false)
const courseBusyIds = ref<number[]>([])
// 加载更多失败状态（区别于共享的 pageError：后者还被删除/新建等操作写入，
// 直接传入会展示无关错误）。错误态交给 InfiniteScrollFooter 就地展示 + 停止自动触发。
const courseLoadMoreError = ref('')

async function loadCourses(reset = false) {
  if (courseLoading.value) return
  courseLoading.value = true
  if (reset) pageError.value = ''
  else courseLoadMoreError.value = ''
  try {
    const payload = await fetchAdminCourses(courseKeyword.value.trim(), '', reset ? 1 : coursePage.value, 20)
    courseItems.value = reset ? payload.list : [...courseItems.value, ...payload.list]
    coursePage.value = payload.page + 1
    courseHasNext.value = payload.hasNext
    courseLoaded.value = true
    courseLoadMoreError.value = ''
  } catch (error) {
    const message = error instanceof Error ? error.message : t('api.adminCourseListFailed')
    if (reset) pageError.value = message
    else courseLoadMoreError.value = message
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
  reviewScope: string
  teamKey: string
}

const courseFormOpen = ref(false)
const courseFormEditingId = ref(0)
const courseForm = ref<CourseForm>(emptyCourseForm())
const courseFormOriginal = ref<CourseForm>(emptyCourseForm())
const courseFormSubmitting = ref(false)
const courseFormError = ref('')

function emptyCourseForm(): CourseForm {
  return { primaryCode: '', name: '', department: '', credit: '', aliases: '', instructors: '', reviewScope: 'teacher', teamKey: '' }
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
    reviewScope: 'teacher',
    teamKey: '',
  }
  courseFormOriginal.value = { ...courseForm.value }
  courseFormError.value = ''
  courseFormOpen.value = true
  // 管理列表不含 reviewScope/teamKey，异步取详情预填；隐藏课程详情 404 时保持默认值。
  void loadCourseDetailForForm(item.id)
}

async function loadCourseDetailForForm(courseId: number) {
  try {
    const detail = await getCourseDetail(courseId)
    if (courseFormEditingId.value !== courseId) return
    courseForm.value.reviewScope = detail.reviewScope || 'teacher'
    courseForm.value.teamKey = detail.teamKey || ''
    courseFormOriginal.value.reviewScope = courseForm.value.reviewScope
    courseFormOriginal.value.teamKey = courseForm.value.teamKey
  } catch {
    // 隐藏课程详情不可读（404）：保留默认 teacher/空，允许直接保存其它字段。
  }
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
      if (form.reviewScope !== original.reviewScope) payload.reviewScope = form.reviewScope
      if (form.teamKey !== original.teamKey) payload.teamKey = form.teamKey
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

// ---- 课程沿革审核 ----
const relationStatus = ref('pending')
const relationType = ref('')
const relationPage = ref(1)
const relationItems = ref<CourseRelationItem[]>([])
const relationHasNext = ref(true)
const relationLoading = ref(false)
const relationLoaded = ref(false)
const relationBusyIds = ref<number[]>([])
const relationLoadMoreError = ref('')
const relationMessage = ref('')
// 请求代际：筛选切换（reset）时递增，使在途旧请求的结果失效——旧筛选的响应
// 不得覆盖新筛选列表（review P2：stale response 会把错误筛选的行写入刚清空的列表）。
let relationRequestSeq = 0

const relationTypeFilterOptions = [
  { key: '', label: 'courseManagement.relationType.all' },
  { key: 'EQUIVALENT', label: 'courseManagement.relationType.EQUIVALENT' },
  { key: 'RENAMED_FROM', label: 'courseManagement.relationType.RENAMED_FROM' },
  { key: 'SPLIT_FROM', label: 'courseManagement.relationType.SPLIT_FROM' },
  { key: 'MERGED_FROM', label: 'courseManagement.relationType.MERGED_FROM' },
  { key: 'RELATED', label: 'courseManagement.relationType.RELATED' },
]

const relationStatusOptions = [
  { key: 'pending', label: 'courseManagement.relationTabs.pending' },
  { key: 'approved', label: 'courseManagement.relationTabs.approved' },
  { key: 'ignored', label: 'courseManagement.relationTabs.ignored' },
  { key: 'merged', label: 'courseManagement.relationTabs.merged' },
  { key: '', label: 'courseManagement.relationTabs.all' },
]

const relationTypeOptions = ['EQUIVALENT', 'RENAMED_FROM', 'SPLIT_FROM', 'MERGED_FROM', 'RELATED']

// 沿革列表项自带 from/to 课程摘要（后端附带课号/课程名/教师），直接展示；
// 课程已删除（摘要缺省）时回退 #id。
function relationSideName(item: CourseRelationItem, side: 'fromCourse' | 'toCourse'): string {
  const brief = item[side]
  const id = side === 'fromCourse' ? item.fromCourseId : item.toCourseId
  return brief?.name ? brief.name : `#${id}`
}

function relationSideMeta(item: CourseRelationItem, side: 'fromCourse' | 'toCourse'): string {
  const brief = item[side]
  if (!brief) return ''
  const parts: string[] = []
  if (brief.primaryCode) parts.push(brief.primaryCode)
  if (brief.teacherName) parts.push(brief.teacherName)
  if (brief.status === 1) parts.push(t('courseManagement.relationCourseHidden'))
  return parts.join(' · ')
}

function relationTypeLabel(type: string): string {
  return t(`courseManagement.relationType.${type}`)
}

function relationSourceLabel(source: string): string {
  return t(`courseManagement.relationSource.${source}`)
}

function relationStatusLabel(status: string): string {
  return t(`courseManagement.relationTabs.${status}`)
}

function relationStatusBadgeClass(status: string): string {
  if (status === 'pending') return 'gf-badge-warning'
  if (status === 'approved') return 'gf-badge-success'
  if (status === 'merged') return 'gf-badge-info'
  return 'gf-badge-muted'
}

function isMergeableType(type: string): boolean {
  return type === 'EQUIVALENT' || type === 'RENAMED_FROM'
}

function formatConfidence(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '—'
  return `${Math.round(value * 100)}%`
}

async function loadRelations(reset = false) {
  if (!reset && relationLoading.value) return
  const seq = ++relationRequestSeq
  relationLoading.value = true
  if (reset) pageError.value = ''
  else relationLoadMoreError.value = ''
  try {
    const payload = await fetchCourseRelations(relationStatus.value, relationType.value, reset ? 1 : relationPage.value, 20)
    if (seq !== relationRequestSeq) return // 已有更新的请求在途：丢弃本次结果
    relationItems.value = reset ? payload.list : [...relationItems.value, ...payload.list]
    relationPage.value = payload.page + 1
    relationHasNext.value = payload.hasNext
    relationLoaded.value = true
    relationLoadMoreError.value = ''
  } catch (error) {
    if (seq !== relationRequestSeq) return
    const message = error instanceof Error ? error.message : t('api.adminCourseRelationListFailed')
    if (reset) pageError.value = message
    else relationLoadMoreError.value = message
  } finally {
    if (seq === relationRequestSeq) relationLoading.value = false
  }
}

function resetRelationList() {
  relationPage.value = 1
  relationItems.value = []
  relationHasNext.value = true
  relationLoaded.value = false
  void loadRelations(true)
}

function switchRelationStatus(status: string) {
  if (relationStatus.value === status) return
  relationStatus.value = status
  resetRelationList()
}

function switchRelationType(event: Event) {
  const type = (event.target as HTMLSelectElement).value
  if (relationType.value === type) return
  relationType.value = type
  resetRelationList()
}

async function revertRelation(item: CourseRelationItem) {
  if (relationBusy(item.id)) return
  relationBusyIds.value = [...relationBusyIds.value, item.id]
  relationMessage.value = ''
  pageError.value = ''
  try {
    await resetCourseRelation(item.id)
    relationMessage.value = t('courseManagement.relationReverted')
    await loadRelations(true)
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : t('api.adminCourseRelationOpFailed')
  } finally {
    relationBusyIds.value = relationBusyIds.value.filter((id) => id !== item.id)
  }
}


function relationBusy(id: number) {
  return relationBusyIds.value.includes(id)
}

async function approveRelation(item: CourseRelationItem) {
  if (relationBusy(item.id)) return
  relationBusyIds.value = [...relationBusyIds.value, item.id]
  relationMessage.value = ''
  pageError.value = ''
  try {
    await approveCourseRelation(item.id)
    relationMessage.value = t('courseManagement.relationApproved')
    await loadRelations(true)
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : t('api.adminCourseRelationOpFailed')
  } finally {
    relationBusyIds.value = relationBusyIds.value.filter((id) => id !== item.id)
  }
}

async function ignoreRelation(item: CourseRelationItem) {
  if (relationBusy(item.id)) return
  relationBusyIds.value = [...relationBusyIds.value, item.id]
  relationMessage.value = ''
  pageError.value = ''
  try {
    await ignoreCourseRelation(item.id)
    relationMessage.value = t('courseManagement.relationIgnored')
    await loadRelations(true)
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : t('api.adminCourseRelationOpFailed')
  } finally {
    relationBusyIds.value = relationBusyIds.value.filter((id) => id !== item.id)
  }
}

// ---- 沿革合并 / 撤销合并（确认弹窗） ----
const relationConfirmTarget = ref<CourseRelationItem | null>(null)
const relationConfirmAction = ref<'merge' | 'undo'>('merge')
const relationConfirmSubmitting = ref(false)

function openMergeRelation(item: CourseRelationItem) {
  relationConfirmAction.value = 'merge'
  relationConfirmTarget.value = item
}

function openUndoMergeRelation(item: CourseRelationItem) {
  relationConfirmAction.value = 'undo'
  relationConfirmTarget.value = item
}

function closeRelationConfirm() {
  relationConfirmTarget.value = null
  relationConfirmSubmitting.value = false
}

async function confirmRelationAction() {
  const item = relationConfirmTarget.value
  if (!item) return
  relationConfirmSubmitting.value = true
  relationMessage.value = ''
  pageError.value = ''
  try {
    if (relationConfirmAction.value === 'merge') {
      const result: CourseMergeResult = await mergeCourseRelation(item.id)
      relationMessage.value = t('courseManagement.relationMergeDone', {
        from: result.fromName || `#${result.fromCourseId}`,
        to: result.toName || `#${result.toCourseId}`,
        movedOfferings: result.movedOfferings,
        migratedAliases: result.migratedAliases,
      })
    } else {
      await undoMergeCourseRelation(item.id)
      relationMessage.value = t('courseManagement.relationMergeUndoDone')
    }
    closeRelationConfirm()
    await loadRelations(true)
    // 合并会隐藏旧卡、撤销会恢复旧卡，刷新课程列表保持一致。
    await loadCourses(true)
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : t('api.courseMergeFailed')
    closeRelationConfirm()
  } finally {
    relationConfirmSubmitting.value = false
  }
}

// ---- 手动建关系 ----
interface RelationCreateForm {
  fromCourseId: string
  toCourseId: string
  relationType: string
  evidence: string
  confidence: string
}

const relationCreateOpen = ref(false)
const relationCreateForm = ref<RelationCreateForm>(emptyRelationCreateForm())
const relationCreateSubmitting = ref(false)
const relationCreateError = ref('')

function emptyRelationCreateForm(): RelationCreateForm {
  return { fromCourseId: '', toCourseId: '', relationType: 'EQUIVALENT', evidence: '', confidence: '' }
}

async function submitRelationCreate() {
  const form = relationCreateForm.value
  const fromCourseId = Number.parseInt(form.fromCourseId, 10)
  const toCourseId = Number.parseInt(form.toCourseId, 10)
  if (!Number.isInteger(fromCourseId) || fromCourseId <= 0) {
    relationCreateError.value = t('courseManagement.relationCreateFromRequired')
    return
  }
  if (!Number.isInteger(toCourseId) || toCourseId <= 0) {
    relationCreateError.value = t('courseManagement.relationCreateToRequired')
    return
  }
  if (fromCourseId === toCourseId) {
    relationCreateError.value = t('courseManagement.relationCreateSame')
    return
  }
  const confidence = Number.parseFloat(form.confidence)
  relationCreateSubmitting.value = true
  relationCreateError.value = ''
  try {
    await createCourseRelation({
      fromCourseId,
      toCourseId,
      relationType: form.relationType,
      evidence: form.evidence.trim() || undefined,
      confidence: Number.isFinite(confidence) && confidence >= 0 ? confidence : undefined,
    })
    relationCreateForm.value = emptyRelationCreateForm()
    relationMessage.value = t('courseManagement.relationCreated')
    await loadRelations(true)
  } catch (error) {
    relationCreateError.value = error instanceof Error ? error.message : t('api.adminCourseRelationCreateFailed')
  } finally {
    relationCreateSubmitting.value = false
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
const reviewLoadMoreError = ref('')

async function loadReviews(reset = false) {
  if (reviewLoading.value) return
  reviewLoading.value = true
  if (reset) pageError.value = ''
  else reviewLoadMoreError.value = ''
  try {
    const payload = await fetchAdminReviews(reviewKeyword.value.trim(), reviewStatus.value, reset ? 0 : reviewCursor.value, 20)
    reviewItems.value = reset ? payload.items : [...reviewItems.value, ...payload.items]
    reviewCursor.value = payload.nextCursor
    reviewHasNext.value = payload.hasNext
    reviewLoaded.value = true
    reviewLoadMoreError.value = ''
  } catch (error) {
    const message = error instanceof Error ? error.message : t('api.adminReviewListFailed')
    if (reset) pageError.value = message
    else reviewLoadMoreError.value = message
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

// 首次切到「评价管理」/「课程沿革」tab 时若尚未加载，触发一次拉取，避免显示空态。
watch(activeTab, (tab) => {
  if (tab === 'reviews' && !reviewLoaded.value) {
    void loadReviews(true)
  }
  if (tab === 'relations' && !relationLoaded.value) {
    void loadRelations(true)
  }
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
        <button
          type="button"
          class="gf-tab"
          :class="activeTab === 'relations' ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
          @click="activeTab = 'relations'"
        >
          {{ t('courseManagement.tabs.relations') }}
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

        <InfiniteScrollFooter
          v-if="courseLoaded && (courseItems.length || courseHasNext)"
          :has-next="courseHasNext"
          :loading="courseLoading"
          :error="courseLoadMoreError"
          :has-items="courseItems.length > 0"
          @load-more="loadCourses(false)"
        />
      </div>
    </section>

    <!-- 评价管理 tab -->
    <section v-else-if="activeTab === 'reviews'" class="space-y-3">
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

        <InfiniteScrollFooter
          v-if="reviewLoaded && (reviewItems.length || reviewHasNext)"
          :has-next="reviewHasNext"
          :loading="reviewLoading"
          :error="reviewLoadMoreError"
          :has-items="reviewItems.length > 0"
          @load-more="loadReviews(false)"
        />
      </div>
    </section>

    <!-- 课程沿革 tab -->
    <section v-else-if="activeTab === 'relations'" class="space-y-3">
      <div class="gf-card overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-2 border-b border-line bg-base-200/60 p-2">
          <div class="flex flex-wrap items-center gap-1">
            <button
              v-for="option in relationStatusOptions"
              :key="option.key"
              type="button"
              class="gf-tab"
              :class="relationStatus === option.key ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
              @click="switchRelationStatus(option.key)"
            >
              {{ t(option.label) }}
            </button>
            <select
              :value="relationType"
              class="gf-input h-8 w-auto min-w-[7.5rem] shrink-0 px-2 text-xs font-semibold"
              :title="t('courseManagement.relationType.all')"
              @change="switchRelationType"
            >
              <option v-for="option in relationTypeFilterOptions" :key="option.key" :value="option.key">
                {{ t(option.label) }}
              </option>
            </select>
          </div>
          <button type="button" class="gf-button gf-button-sm gf-button-primary shrink-0" @click="relationCreateOpen = true">
            <Plus class="h-4 w-4" />
            {{ t('courseManagement.relationCreateTitle') }}
          </button>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full min-w-[900px] text-left text-sm">
            <colgroup>
              <col class="w-[16%]" />
              <col class="w-[16%]" />
              <col class="w-[11%]" />
              <col class="w-[9%]" />
              <col class="w-[9%]" />
              <col class="w-[14%]" />
              <col class="w-[9%]" />
              <col class="w-[16%]" />
            </colgroup>
            <thead class="border-b border-line/70 bg-base-200/40 text-xs text-base-content/55">
              <tr>
                <th class="px-3 py-3 font-medium">{{ t('courseManagement.relationColFrom') }}</th>
                <th class="px-3 py-3 font-medium">{{ t('courseManagement.relationColTo') }}</th>
                <th class="px-3 py-3 font-medium">{{ t('courseManagement.relationColType') }}</th>
                <th class="px-3 py-3 font-medium">{{ t('courseManagement.relationColSource') }}</th>
                <th class="px-3 py-3 font-medium">{{ t('courseManagement.relationColConfidence') }}</th>
                <th class="px-3 py-3 font-medium">{{ t('courseManagement.relationColEvidence') }}</th>
                <th class="px-3 py-3 font-medium">{{ t('courseManagement.relationColStatus') }}</th>
                <th class="px-3 py-3 text-right font-medium">{{ t('courseManagement.columns.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-line/60">
              <tr v-for="item in relationItems" :key="item.id" class="align-top transition hover:bg-base-200/40">
                <td class="px-3 py-3">
                  <span class="block truncate font-medium text-base-content" :title="relationSideName(item, 'fromCourse')">{{ relationSideName(item, 'fromCourse') }}</span>
                  <span v-if="relationSideMeta(item, 'fromCourse')" class="block truncate text-xs text-base-content/55">{{ relationSideMeta(item, 'fromCourse') }}</span>
                </td>
                <td class="px-3 py-3">
                  <span class="block truncate font-medium text-base-content" :title="relationSideName(item, 'toCourse')">{{ relationSideName(item, 'toCourse') }}</span>
                  <span v-if="relationSideMeta(item, 'toCourse')" class="block truncate text-xs text-base-content/55">{{ relationSideMeta(item, 'toCourse') }}</span>
                </td>
                <td class="px-3 py-3"><span class="gf-badge gf-badge-ghost text-[11px]">{{ relationTypeLabel(item.relationType) }}</span></td>
                <td class="px-3 py-3 text-base-content/60">{{ relationSourceLabel(item.source) }}</td>
                <td class="px-3 py-3 tabular-nums text-base-content/60">{{ formatConfidence(item.confidence) }}</td>
                <td class="max-w-[180px] px-3 py-3">
                  <details v-if="item.evidenceJson" class="group">
                    <summary class="cursor-pointer select-none text-xs text-primary/80 hover:text-primary">{{ t('courseManagement.relationEvidenceToggle') }}</summary>
                    <pre class="mt-1 whitespace-pre-wrap break-all rounded bg-base-200/70 p-2 font-mono text-[11px] leading-4 text-base-content/65">{{ item.evidenceJson }}</pre>
                  </details>
                  <span v-else class="text-base-content/35">—</span>
                </td>
                <td class="px-3 py-3"><span class="gf-badge text-[11px]" :class="relationStatusBadgeClass(item.status)">{{ relationStatusLabel(item.status) }}</span></td>
                <td class="px-3 py-3">
                  <div class="flex flex-wrap items-center justify-end gap-1.5">
                    <button
                      v-if="item.status === 'pending' && isMergeableType(item.relationType)"
                      type="button"
                      class="gf-button gf-button-sm gf-button-primary shrink-0"
                      :disabled="relationBusy(item.id)"
                      @click="openMergeRelation(item)"
                    >
                      {{ t('courseManagement.relationMerge') }}
                    </button>
                    <button
                      v-if="item.status === 'pending' && !isMergeableType(item.relationType)"
                      type="button"
                      class="gf-button gf-button-sm gf-button-outline shrink-0"
                      :disabled="relationBusy(item.id)"
                      @click="approveRelation(item)"
                    >
                      {{ t('courseManagement.relationApprove') }}
                    </button>
                    <button
                      v-if="item.status === 'pending'"
                      type="button"
                      class="gf-button gf-button-sm gf-button-ghost shrink-0"
                      :disabled="relationBusy(item.id)"
                      @click="ignoreRelation(item)"
                    >
                      {{ t('courseManagement.relationIgnore') }}
                    </button>
                    <button
                      v-if="item.status === 'approved' || item.status === 'ignored'"
                      type="button"
                      class="gf-button gf-button-sm gf-button-outline shrink-0"
                      :disabled="relationBusy(item.id)"
                      @click="revertRelation(item)"
                    >
                      {{ t('courseManagement.relationRevert') }}
                    </button>
                    <button
                      v-if="item.status === 'merged'"
                      type="button"
                      class="gf-button gf-button-sm gf-button-outline shrink-0"
                      :disabled="relationBusy(item.id)"
                      @click="openUndoMergeRelation(item)"
                    >
                      {{ t('courseManagement.relationMergeUndo') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <EmptyState
          v-if="!relationItems.length && relationLoading"
          :icon="BookOpen"
          :title="t('courseManagement.loading')"
          loading
        />
        <EmptyState
          v-else-if="!relationItems.length"
          :icon="BookOpen"
          :title="t('courseManagement.relationsEmpty')"
          :description="t('courseManagement.relationsEmptyDesc')"
        />

        <InfiniteScrollFooter
          v-if="relationLoaded && (relationItems.length || relationHasNext)"
          :has-next="relationHasNext"
          :loading="relationLoading"
          :error="relationLoadMoreError"
          :has-items="relationItems.length > 0"
          @load-more="loadRelations(false)"
        />
      </div>
    </section>

    <!-- 手动建关系弹窗 -->
    <Teleport to="body">
      <div
        v-if="relationCreateOpen"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        @click.self="relationCreateOpen = false"
      >
        <div class="w-full max-w-lg rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <div class="flex items-start justify-between gap-3">
            <h2 class="text-base font-bold text-base-content">{{ t('courseManagement.relationCreateTitle') }}</h2>
            <button
              type="button"
              class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75"
              @click="relationCreateOpen = false"
            >
              <X class="h-4 w-4" />
            </button>
          </div>

          <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.relationCreateFrom') }}</span>
              <input v-model="relationCreateForm.fromCourseId" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.relationCreateIdPlaceholder')" />
            </label>
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.relationCreateTo') }}</span>
              <input v-model="relationCreateForm.toCourseId" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.relationCreateIdPlaceholder')" />
            </label>
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.relationCreateType') }}</span>
              <select v-model="relationCreateForm.relationType" class="gf-input gf-input-md w-full">
                <option v-for="type in relationTypeOptions" :key="type" :value="type">{{ relationTypeLabel(type) }}</option>
              </select>
            </label>
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.relationCreateConfidence') }}</span>
              <input v-model="relationCreateForm.confidence" type="number" min="0" max="1" step="0.05" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.relationCreateConfidencePlaceholder')" />
            </label>
            <label class="block sm:col-span-2">
              <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.relationCreateEvidence') }}</span>
              <textarea v-model="relationCreateForm.evidence" class="gf-textarea min-h-24 w-full" :placeholder="t('courseManagement.relationCreateEvidencePlaceholder')" />
            </label>
          </div>

          <p v-if="relationCreateError" class="mt-3 text-sm text-error">{{ relationCreateError }}</p>

          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-md gf-button-muted" :disabled="relationCreateSubmitting" @click="relationCreateOpen = false">
              {{ t('courseManagement.cancel') }}
            </button>
            <button type="button" class="gf-button gf-button-md gf-button-primary" :disabled="relationCreateSubmitting" @click="submitRelationCreate">
              <Loader2 v-if="relationCreateSubmitting" class="h-4 w-4 animate-spin" />
              {{ relationCreateSubmitting ? t('common.loadingShort') : t('courseManagement.relationCreateSubmit') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 沿革合并 / 撤销合并确认弹窗 -->
    <Teleport to="body">
      <div
        v-if="relationConfirmTarget"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        @click.self="closeRelationConfirm"
      >
        <div class="w-full max-w-md rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <h2 class="text-base font-bold text-base-content">
            {{ relationConfirmAction === 'merge' ? t('courseManagement.relationMerge') : t('courseManagement.relationMergeUndo') }}
          </h2>
          <p class="mt-2 text-[13px] leading-5 text-base-content/60">
            {{
              relationConfirmAction === 'merge'
                ? t('courseManagement.relationMergeConfirm', {
                    from: relationSideName(relationConfirmTarget, 'fromCourse'),
                    to: relationSideName(relationConfirmTarget, 'toCourse'),
                  })
                : t('courseManagement.relationMergeUndoConfirm', {
                    from: relationSideName(relationConfirmTarget, 'fromCourse'),
                    to: relationSideName(relationConfirmTarget, 'toCourse'),
                  })
            }}
          </p>
          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-md gf-button-muted" :disabled="relationConfirmSubmitting" @click="closeRelationConfirm">
              {{ t('courseManagement.cancel') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-md"
              :class="relationConfirmAction === 'merge' ? 'gf-button-primary' : 'gf-button-danger'"
              :disabled="relationConfirmSubmitting"
              @click="confirmRelationAction"
            >
              <Loader2 v-if="relationConfirmSubmitting" class="h-4 w-4 animate-spin" />
              {{ relationConfirmSubmitting ? t('common.loadingShort') : (relationConfirmAction === 'merge' ? t('courseManagement.relationMerge') : t('courseManagement.relationMergeUndo')) }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

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
            <template v-if="courseFormEditingId !== 0">
              <label class="block">
                <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.courseForm.reviewScope') }}</span>
                <select v-model="courseForm.reviewScope" class="gf-input gf-input-md w-full">
                  <option value="teacher">{{ t('courseManagement.courseForm.reviewScopeTeacher') }}</option>
                  <option value="team">{{ t('courseManagement.courseForm.reviewScopeTeam') }}</option>
                  <option value="course">{{ t('courseManagement.courseForm.reviewScopeCourse') }}</option>
                </select>
              </label>
              <label class="block">
                <span class="mb-1 block text-xs font-medium text-base-content/60">{{ t('courseManagement.courseForm.teamKey') }}</span>
                <input v-model="courseForm.teamKey" class="gf-input gf-input-md w-full" :placeholder="t('courseManagement.courseForm.teamKeyPlaceholder')" />
              </label>
            </template>
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
