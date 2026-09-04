<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, BookOpen, Building2, CalendarDays, Check, Download, FileText, Flag, GraduationCap, Hash, Loader2, MessageSquareText, Pencil, Share2, Star, ThumbsDown, ThumbsUp, Trash2, UsersRound, X } from '@lucide/vue'
import {
  createCourseReview,
  deleteCourseReview,
  getCourseRelated,
  listCourseReviews,
  reportCourseReview,
  setReviewDislike,
  setReviewHelpful,
  updateCourseReview,
  uploadImage,
  type CourseLineageItem,
  type CourseRelatedResult,
  type ReviewPayload,
} from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
import { useFlashMessages } from '@/runtime/flash-message'
import { processImageFile, validateImageFile } from '@/runtime/image'
import CourseReviewTemplateSelector from '@/site/components/CourseReviewTemplateSelector.vue'
import VditorOfficial from '@/site/components/VditorOfficial.vue'
import AISummaryCard from '@/site/components/AISummaryCard.vue'
import RatingSummaryCard from '@/site/components/RatingSummaryCard.vue'
import EmptyState from '@/site/components/EmptyState.vue'
import InfiniteScrollFooter from '@/site/components/InfiniteScrollFooter.vue'
import { COURSE_REVIEW_TEMPLATES } from '@/site/utils/course-review-templates'
import {
  exportShareNode,
  inlineMarkdownImages,
  reviewAvatarSrc,
  reviewSqid,
  waitForImages,
} from '@/site/utils/course-review-share'
import {
  nextReviewTotalOnCreate,
  nextReviewTotalOnDelete,
  resolveStatsReviewCount,
} from '@/site/utils/course-review-count'
import { createReviewPageLoader } from '@/site/utils/course-review-loader'
import { readCourseCatalogReturn } from '@/site/utils/course-catalog-return'
import { shortTerm } from '@/site/utils/term'
import PageHeader from '@/site/components/PageHeader.vue'
import type { CourseDetailPageProps, LayoutPayload } from '@gooseforum/client'

// SSR 透传的新字段（后端 CourseDetail 扩展，仅显示层读取）：
// reviewScope 课评范围三档（teacher/team/course，缺省按 teacher 处理）、teamKey 团队键、
// teamInstructors 团队教师名单、legacyNames 原名标注。
type CourseDetailScopeFields = {
  reviewScope?: 'teacher' | 'team' | 'course'
  teamKey?: string
  teamInstructors?: string[]
  legacyNames?: string[]
}

const page = defineProps<{
  layout: LayoutPayload
  props: CourseDetailPageProps & { course: CourseDetailPageProps['course'] & CourseDetailScopeFields }
}>()
const { t } = useI18n()
const { push: pushFlash } = useFlashMessages()

const loginHref = computed(() => {
  const next = encodeURIComponent(window.location.pathname + window.location.search + window.location.hash)
  return `/login?next=${next}`
})

// 「返回课程目录」：优先恢复进入时的目录搜索/筛选态（列表页写入 sessionStorage）；
// SSR 首帧与无记录时回退 /courses。详情页内多级跳转后记录仍有效。
const catalogReturnHref = ref('/courses')
onMounted(() => {
  const saved = readCourseCatalogReturn(window.location.href)
  if (saved) catalogReturnHref.value = saved
})

function formatCredit(creditX10: number) {
  if (!creditX10) return ''
  return (creditX10 / 10).toFixed(1).replace(/\.0$/, '')
}

function offeringLabel(id: number) {
  const offering = page.props.course.offerings?.find((item) => item.id === id)
  if (!offering) return `#${id}`
  const classLabel = offering.className || offering.classCode || ''
  return [shortTerm(offering.termCode), classLabel, offering.campus, offering.instructors?.join('、')].filter(Boolean).join(' · ')
}

function formatRating(avg: number) {
  return avg > 0 ? avg.toFixed(1) : '-'
}

// ---- 评分卡评论数：SSR 回退值（聚焦教学班时保持课程级口径，见 statsReviewCount）----
const reviewCount = computed(() => page.props.course.reviewCount ?? 0)
// 均分：分享评价卡（share 快照）复用；评分仪表卡由 RatingSummaryCard 组件自身计算。
const ratingAvg = computed(() => page.props.course.ratingAvg ?? null)

// ---- 相关课程（同教师其他课 / 同课程其他教师）----
const related = ref<CourseRelatedResult | null>(null)
// 初始为 true：避免首屏未加载完成时错误闪现"暂无相关内容"。
const relatedLoading = ref(true)
const relatedError = ref('')

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
// ---- 课评范围（teacher/team/course）与原名标注（SSR 透传，显示层）----
// team 档教师名单：「教学团队 · 张三、李四等 N 位教师」；无名单回退单卡教师名。
const teamInstructorsLabel = computed(() => {
  const names = page.props.course.teamInstructors ?? []
  if (!names.length) return ''
  const joined = names.join('、')
  if (names.length <= 1) return `${t('courseDetailPage.teamInstructorsPrefix')}${joined}`
  return `${t('courseDetailPage.teamInstructorsPrefix')}${joined}${t('courseDetailPage.teamInstructorsCountSuffix', { count: names.length })}`
})

// ---- 课程沿革（related.lineage）----
const RELATION_LABEL_KEYS: Record<string, string> = {
  EQUIVALENT: 'courseDetailPage.relationEquivalent',
  RENAMED_FROM: 'courseDetailPage.relationRenamed',
  SPLIT_FROM: 'courseDetailPage.relationSplit',
  MERGED_FROM: 'courseDetailPage.relationMerged',
  RELATED: 'courseDetailPage.relationRelated',
}

function relationLabel(relationType: string): string {
  const key = RELATION_LABEL_KEYS[relationType]
  return key ? t(key) : relationType
}

// 沿革条目只链向「非本卡」一侧的目标卡；merged 后旧卡已隐藏，不可跳转。
function lineageFromHref(item: CourseLineageItem): string | undefined {
  if (item.direction !== 'to' || item.status === 'merged') return undefined
  return `/courses/${item.fromCourseId}`
}

function lineageToHref(item: CourseLineageItem): string | undefined {
  if (item.direction !== 'from' || item.status === 'merged') return undefined
  return `/courses/${item.toCourseId}`
}

// authorLabel 作者展示名：member 用服务端回填的用户名；anonymous/legacy 走本地 i18n，
// 避免非中文界面原样渲染服务端硬编码的中文标签。
function authorLabel(author: ReviewPayload['author']) {
  if (author.kind === 'member') return author.label
  if (author.kind === 'legacy') return t('courseDetailPage.authorLegacy')
  return t('courseDetailPage.authorAnonymous')
}


// ---- 教学班级课评聚焦（排课器跳转 /courses/:id?offeringId=:offeringId） ----
// 打开详情页时若带 offeringId 查询参数，评价列表只显示该教学班的评价。
const focusOfferingId = ref<number>(0)
const OFFERING_QUERY_KEY = 'offeringId'

function parseFocusOfferingId(): number {
  const raw = new URLSearchParams(window.location.search).get(OFFERING_QUERY_KEY)
  const value = Number(raw ?? '')
  if (!Number.isInteger(value) || value <= 0) return 0
  // 只接受当前课程可见开课实例中的 offeringId：
  // 防跨课程 offering（/courses/A?offeringId=<B 的班>）与隐藏 offering 泄露。
  return page.props.course.offerings?.some((item) => item.id === value) ? value : 0
}

function activeOfferingId(): number {
  return focusOfferingId.value || 0
}

function setOfferingFocus(offeringId: number) {
  focusOfferingId.value = offeringId
  // 切班聚焦后重新加载评价列表（offering 过滤），并回到顶部评价区。
  reviewLoaded.value = false
  void loadReviews()
  const el = document.querySelector('#course-reviews')
  el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
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

// 统计卡评论数：聚焦教学班时保持课程级口径（与 ratingAvg/分布一致，SSR 课程级值），
// 避免过滤后的 offering 级 total 与课程级均分/分布混搭；未聚焦时维持现有行为
// （加载后以客户端 reviewTotal 实时值为准，未加载回退 SSR reviewCount）。
const statsReviewCount = computed(() =>
  focusOfferingId.value
    ? reviewCount.value
    : resolveStatsReviewCount(reviewLoaded.value, reviewTotal.value, reviewCount.value),
)

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
    const reviewPage = await reviewLoader.load(activeOfferingId(), '')
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
    const reviewPage = await reviewLoader.load(activeOfferingId(), reviewNextCursor.value)
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
// 提交按钮三态机：idle=常规；submitting=请求中（spinner 防重复）；success=成功后短暂
// 展示勾选确认再关表单（用户可见反馈）；error=失败（按钮抖动后回 idle，错误就地提示）。
const formSubmitState = ref<'idle' | 'submitting' | 'success' | 'error'>('idle')
// 新建评价成功后定位高亮：表单收起后平滑滚动到该条评论并短暂标记，锚定用户视线。
const highlightedReviewId = ref<number | null>(null)
const formError = ref('')
const editingReviewId = ref<number | null>(null)
const templateSelectorOpen = ref(false)
const formTemplateId = ref('')

// —— 富文本编辑器（与帖子/回复同款 Vditor，紧凑工具栏）：异步就绪遮罩 + 图片上传 ——
const reviewEditor = ref<InstanceType<typeof VditorOfficial> | null>(null)
const reviewEditorReady = ref(false)
const reviewEditorFailed = ref(false)
const uploadingReviewImages = ref(false)

watch(
  () => [reviewEditor.value?.editorReady, reviewEditor.value?.editorFailed] as const,
  ([ready, failed]) => {
    reviewEditorReady.value = !!ready
    reviewEditorFailed.value = !!failed
  },
)

function reviewImageAlt(filename: string) {
  return filename.replace(/\.[^.]+$/, '').replace(/[[\]\n\r]/g, ' ').trim() || 'image'
}

// 粘贴/拖拽图片：与发布页同款流程（校验 → 压缩 → 上传 → 插入 Markdown）。
async function uploadReviewImages(files: File[]) {
  if (!files.length || uploadingReviewImages.value) return
  uploadingReviewImages.value = true
  const markdownImages: string[] = []
  const failed: string[] = []
  try {
    for (const file of files) {
      const validation = validateImageFile(file)
      if (validation) {
        failed.push(`${file.name}: ${validation}`)
        continue
      }
      try {
        const optimized = await processImageFile(file)
        const url = await uploadImage(optimized.file)
        markdownImages.push(`![${reviewImageAlt(file.name)}](${url})`)
      } catch (error) {
        failed.push(`${file.name}: ${error instanceof Error ? error.message : t('api.imageUploadFailed')}`)
      }
    }
    if (markdownImages.length) {
      reviewEditor.value?.insertMarkdown(markdownImages.join('\n'))
    }
    if (failed.length) {
      pushFlash(
        failed.slice(0, 3).join(t('punctuation.semicolon')) +
          (failed.length > 3 ? t('publish.moreImageFailures', { count: failed.length - 3 }) : ''),
        'error',
      )
    } else if (!markdownImages.length) {
      pushFlash(t('publish.noUploadableImages'), 'error')
    }
  } finally {
    uploadingReviewImages.value = false
  }
}

function onReviewEditorError(editorError: Error) {
  reviewEditorFailed.value = true
  pushFlash(editorError.message || t('common.loadFailed'), 'error')
}

function openCreateForm() {
  editingReviewId.value = null
  // 聚焦教学班时写评默认选该班，否则回退第一个开课实例。
  formOfferingId.value = activeOfferingId() || (page.props.course.offerings?.[0]?.id ?? 0)
  formRating.value = 0
  formContent.value = ''
  formAnonymous.value = true
  formError.value = ''
  formTemplateId.value = ''
  templateSelectorOpen.value = false
  formSubmitState.value = 'idle'
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
  formSubmitState.value = 'idle'
  formVisible.value = true
}

function cancelForm() {
  formVisible.value = false
  editingReviewId.value = null
  formError.value = ''
  templateSelectorOpen.value = false
  formSubmitState.value = 'idle'
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
  formSubmitState.value = 'submitting'
  let createdReviewId = 0
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
      createdReviewId = created.id
      reviewLoader.invalidate() // 使进行中的首屏 GET 过期，避免旧快照覆盖刚创建的评价
      // 创建的评价与当前过滤（聚焦教学班）一致时才本地插入/计数；
      // 不一致（用户改了表单中的教学班）只做失效重拉，避免把别的班的评价
      // 插进当前过滤列表（review P1）。
      if (!activeOfferingId() || created.offeringId === activeOfferingId()) {
        reviews.value.unshift(created)
        // 同步计数：创建后 +1（与删除路径递减口径一致），否则统计卡/评价区标题
        // 会一直显示旧值直到刷新。能提交评价说明列表已加载或已具备客户端最新态，
        // 兜底置 reviewLoaded 为 true，确保统计卡走 reviewTotal 而非回退 SSR 旧值。
        reviewTotal.value = nextReviewTotalOnCreate(reviewTotal.value)
      }
      reviewLoaded.value = true
    }
    invalidateReviews()
    // 成功确认态：按钮短暂显示「已提交/已更新」后再关闭表单（微交互，防"点了没反应"）。
    formSubmitState.value = 'success'
    await new Promise((resolve) => setTimeout(resolve, 900))
    formVisible.value = false
    editingReviewId.value = null
    // 新建评价：等表单退场后平滑滚动到该条并短暂高亮，锚定用户视线（避免"凭空出现在列表顶部"）。
    if (createdReviewId) {
      const scrollTarget = () => document.querySelector<HTMLElement>(`#course-review-${createdReviewId}`)
      await new Promise((resolve) => setTimeout(resolve, 260))
      highlightedReviewId.value = createdReviewId
      scrollTarget()?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // prefers-reduced-motion 下跳过两次长延时，直接即时滚动并报读屏（无需 DOM 高亮）。
      const reduceMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
      if (!reduceMotion) {
        setTimeout(() => {
          if (highlightedReviewId.value === createdReviewId) highlightedReviewId.value = null
        }, 2000)
      }
    }
  } catch (error) {
    formError.value = error instanceof Error ? error.message : t('courseDetailPage.reviewSaveFailed')
    // 失败反馈：按钮抖动提示后回 idle，错误信息就地位于表单下方。
    formSubmitState.value = 'error'
    setTimeout(() => {
      formSubmitState.value = 'idle'
    }, 450)
  } finally {
    formSubmitting.value = false
  }
}

// ---- 删除确认（受控 Dialog，替代 window.confirm）----
const pendingDelete = ref<ReviewPayload | null>(null)
// deleting 防 in-flight 双击：删除请求未返回前禁止重复提交/取消，
// 避免 pendingDelete 被中途置 null 导致"删 A 后误关 B 的 Dialog"竞态。
const deleting = ref(false)
// a11y（对齐举报弹窗）：Esc 关闭 + Tab 焦点陷阱 + 打开即聚焦，兑现 aria-modal 承诺。
const deleteDialogEl = ref<HTMLElement | null>(null)
const DELETE_TITLE_ID = 'course-review-delete-title'

function deleteFocusableEls(): HTMLElement[] {
  if (!deleteDialogEl.value) return []
  return Array.from(
    deleteDialogEl.value.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  )
}

function onDeleteKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    cancelRemoveReview()
    return
  }
  if (event.key !== 'Tab') return
  const focusable = deleteFocusableEls()
  if (focusable.length < 2) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

function askRemoveReview(review: ReviewPayload) {
  if (deleting.value) return
  pendingDelete.value = review
  // 弹窗挂载后移入焦点：aria-modal 声明下键盘/读屏用户不应滞留在触发按钮。
  nextTick(() => deleteFocusableEls()[0]?.focus())
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

// ---- helpful / dislike ----
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
    // 点赞与点踩互斥（与 serverless 一致）
    if (next && review.viewer.isDisliked) {
      review.viewer.isDisliked = false
      review.dislikeCount = Math.max(0, review.dislikeCount - 1)
    }
  } catch (error) {
    pushFlash(error instanceof Error ? error.message : t('courseDetailPage.reviewHelpfulFailed'), 'error')
  } finally {
    helpfulBusyIds.value = helpfulBusyIds.value.filter((id) => id !== review.id)
  }
}

const dislikeBusyIds = ref<number[]>([])

async function toggleDislike(review: ReviewPayload) {
  if (!page.layout.viewer.isAuthenticated) {
    window.location.href = loginHref.value
    return
  }
  if (dislikeBusyIds.value.includes(review.id)) return
  dislikeBusyIds.value.push(review.id)
  try {
    const next = !review.viewer.isDisliked
    await setReviewDislike(review.id, next)
    review.viewer.isDisliked = next
    review.dislikeCount += next ? 1 : -1
    // 点踩与点赞互斥（与 serverless 一致）
    if (next && review.viewer.isHelpful) {
      review.viewer.isHelpful = false
      review.helpfulCount = Math.max(0, review.helpfulCount - 1)
    }
  } catch (error) {
    pushFlash(error instanceof Error ? error.message : t('courseDetailPage.reviewDislikeFailed'), 'error')
  } finally {
    dislikeBusyIds.value = dislikeBusyIds.value.filter((id) => id !== review.id)
  }
}

// ---- 分享评论长图（对齐 serverless 分享卡：boring 头像 + 白卡预览 + html-to-image 导出）----
type SharePreviewState = {
  review: ReviewPayload
  avatarUrl: string
  markdownHtml: string
}

const sharePreview = ref<SharePreviewState | null>(null)
const shareBusyId = ref<number | null>(null)
const shareSaving = ref(false)
const shareExportEl = ref<HTMLElement | null>(null)

// 评论头像：member 用论坛用户头像（同名同头像）；匿名/历史评价用 beam 占位头像
// （seed 用评价 id + 展示名，同一评价头像稳定，跨页面一致）。
function reviewAvatar(review: ReviewPayload, size: number): string {
  return reviewAvatarSrc(review.author, review.id, size)
}

function reviewDateLabel(review: ReviewPayload): string {
  return formatDateTime(review.createdAt)
}

async function openShare(review: ReviewPayload) {
  if (shareBusyId.value != null) return
  shareBusyId.value = review.id
  try {
    const markdownHtml = await inlineMarkdownImages(review.contentHtml)
    sharePreview.value = {
      review,
      avatarUrl: reviewAvatar(review, 88),
      markdownHtml,
    }
    document.body.style.overflow = 'hidden'
  } catch (error) {
    pushFlash(error instanceof Error ? error.message : t('courseDetailPage.sharePrepareFailed'), 'error')
  } finally {
    shareBusyId.value = null
  }
}

function closeShare() {
  sharePreview.value = null
  shareSaving.value = false
  document.body.style.overflow = ''
  nextTick(() => {
    const trigger = shareTriggerEl.value
    if (trigger?.isConnected) trigger.focus()
    shareTriggerEl.value = null
  })
}

async function saveShare() {
  const node = shareExportEl.value
  if (!node || shareSaving.value) return
  shareSaving.value = true
  try {
    await waitForImages(node)
    let dataUrl = ''
    let extension: 'png' | 'jpg' = 'png'
    try {
      dataUrl = await exportShareNode(node, 'png')
    } catch {
      dataUrl = await exportShareNode(node, 'jpg')
      extension = 'jpg'
    }
    if (!dataUrl) throw new Error('export_failed')
    const fileName = `${page.props.course.primaryCode || 'yourtj'}-${reviewSqid(sharePreview.value!.review.id)}.${extension}`
    const link = document.createElement('a')
    link.href = dataUrl
    link.download = fileName
    link.click()
  } catch {
    pushFlash(t('courseDetailPage.shareExportFailed'), 'error')
  } finally {
    shareSaving.value = false
  }
}

// ---- 分享弹窗 a11y（同举报弹窗：Esc 关闭 + 焦点陷阱 + 焦点归还 + 滚动锁）----
const shareTriggerEl = ref<HTMLElement | null>(null)
const shareDialogEl = ref<HTMLElement | null>(null)
const SHARE_TITLE_ID = 'course-review-share-title'

function shareFocusableEls(): HTMLElement[] {
  if (!shareDialogEl.value) return []
  return Array.from(
    shareDialogEl.value.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  )
}

function onShareKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    closeShare()
    return
  }
  if (event.key !== 'Tab') return
  const focusable = shareFocusableEls()
  if (focusable.length < 2) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

async function openShareDialog(review: ReviewPayload, event?: MouseEvent) {
  shareTriggerEl.value = event?.currentTarget instanceof HTMLElement ? event.currentTarget : null
  // 先等 openShare 置 sharePreview 再聚焦：弹窗是 v-if="sharePreview"，异步图片
  // 内联完成前聚焦时 DOM 尚未挂载，querySelector 落空，焦点滞留在触发按钮上。
  await openShare(review)
  if (!sharePreview.value) return // 准备失败（flash 已提示）：无弹窗可聚焦
  void nextTick(() => {
    shareDialogEl.value?.querySelector<HTMLElement>('button')?.focus()
  })
}

// ---- 举报（a11y：焦点陷阱 + Esc 关闭 + 焦点归还触发按钮 + 背景滚动锁）----
const reportReasons = ['spam', 'abuse', 'illegal', 'irrelevant', 'other']
const pendingReport = ref<ReviewPayload | null>(null)
const reportReason = ref('spam')
const reportNote = ref('')
const reportSubmitting = ref(false)
const reportError = ref('')
// reportTriggerEl 记录打开弹窗时聚焦的触发按钮，关闭后归还焦点。
const reportTriggerEl = ref<HTMLElement | null>(null)
const reportDialogEl = ref<HTMLElement | null>(null)
const REPORT_TITLE_ID = 'course-report-title'

function reportFocusableEls(): HTMLElement[] {
  if (!reportDialogEl.value) return []
  return Array.from(
    reportDialogEl.value.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  )
}

function onReportKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    closeReport()
    return
  }
  if (event.key !== 'Tab') return
  const focusable = reportFocusableEls()
  if (focusable.length < 2) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

function openReport(review: ReviewPayload) {
  reportTriggerEl.value = document.activeElement instanceof HTMLElement ? document.activeElement : null
  pendingReport.value = review
  reportReason.value = 'spam'
  reportNote.value = ''
  reportError.value = ''
  document.body.style.overflow = 'hidden'
  nextTick(() => reportFocusableEls()[0]?.focus())
}

function closeReport() {
  pendingReport.value = null
  reportError.value = ''
  document.body.style.overflow = ''
  nextTick(() => {
    const trigger = reportTriggerEl.value
    if (trigger?.isConnected) trigger.focus()
    reportTriggerEl.value = null
  })
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
  focusOfferingId.value = parseFocusOfferingId()
  loadReviews()
  loadRelated()
  // 预览面板「撰写评价」直达：带 writeReview=1 进入时自动打开写评表单并滚动到评价区。
  const params = new URLSearchParams(window.location.search)
  if (params.has('writeReview') && page.layout.viewer.isAuthenticated && page.props.course.offerings?.length) {
    openCreateForm()
    void nextTick(() => {
      document.querySelector('#course-reviews')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }
})

// 页面卸载（SPA 导航）时若举报弹窗仍打开，恢复背景滚动，避免滚动锁残留。
onBeforeUnmount(() => {
  document.body.style.overflow = ''
})
</script>

<template>
  <div class="px-4 pb-12 sm:px-0">
    <a
      :href="catalogReturnHref"
      class="mb-3 mt-3 inline-flex items-center gap-1 text-[13px] text-base-content/55 hover:text-primary sm:mt-0"
    >
      <ArrowLeft class="h-3.5 w-3.5" />
      {{ t('courseDetailPage.backToList') }}
    </a>

    <PageHeader :title="props.course.name">
      <template #badge>
        <span v-if="props.course.legacyNames?.length" class="text-[12px] text-base-content/45">
          {{ t('courseDetailPage.legacyNamesLabel') }}{{ (props.course.legacyNames ?? []).join('、') }}
        </span>
        <span class="gf-badge gf-badge-muted">{{ props.course.primaryCode }}</span>
        <span v-if="props.course.reviewScope === 'team'" class="gf-badge gf-badge-info">
          {{ t('courseDetailPage.reviewScopeTeam') }}
        </span>
        <span v-else-if="props.course.reviewScope === 'course'" class="gf-badge gf-badge-info">
          {{ t('courseDetailPage.reviewScopeCourse') }}
        </span>
      </template>
      <template #meta>
        <!-- 信息栏：紧凑语义徽标条（院系 / 教师 / 学分），带形状+微色彩的"面板感"、
             主题色 icon 统一 accent；仅 ~28px 高，不挤压主内容（better-layout: 共享对齐边 + 8px 间距）。 -->
        <div class="mt-3 flex flex-wrap items-center gap-1.5">
          <span
            class="inline-flex items-center gap-1.5 rounded-full border border-line/70 bg-base-100 px-2.5 py-1 text-[12px] leading-none text-base-content/70"
          >
            <Building2 class="h-3.5 w-3.5 text-primary/65" />
            {{ props.course.department }}
          </span>
          <span
            class="inline-flex items-center gap-1.5 rounded-full border border-line/70 bg-base-100 px-2.5 py-1 text-[12px] leading-none text-base-content/70"
          >
            <UsersRound class="h-3.5 w-3.5 text-primary/65" />
            <span class="font-medium text-base-content/80">
              <template v-if="props.course.reviewScope === 'team' && teamInstructorsLabel">
                {{ teamInstructorsLabel }}
              </template>
              <template v-else>
                {{ props.course.teacherName || t('courseDetailPage.noTeacher') }}
              </template>
            </span>
          </span>
          <span
            v-if="formatCredit(props.course.creditX10)"
            class="inline-flex items-center gap-1.5 rounded-full border border-line/70 bg-base-100 px-2.5 py-1 text-[12px] leading-none text-base-content/70"
          >
            <GraduationCap class="h-3.5 w-3.5 text-primary/65" />
            {{ t('courseDetailPage.credit') }}
            <span class="font-semibold tabular-nums text-base-content/80">{{ formatCredit(props.course.creditX10) }}</span>
          </span>
        </div>
      </template>
    </PageHeader>

    <div v-if="props.course.aliases?.length" class="mb-6 flex flex-wrap items-center gap-1.5">
      <span class="text-[12px] text-base-content/45">{{ t('courseDetailPage.aliases') }}：</span>
      <span v-for="alias in props.course.aliases" :key="alias" class="gf-badge gf-badge-ghost text-[11px]">
        {{ alias }}
      </span>
    </div>

    <!-- 评分仪表卡：移动端实例挂在此处（页头下方）；桌面端由右栏实例（hidden xl:block）接管。 -->
    <RatingSummaryCard
      class="xl:hidden"
      :rating-avg="page.props.course.ratingAvg ?? null"
      :review-count="statsReviewCount"
      :distribution="page.props.course.ratingDistribution ?? [0, 0, 0, 0, 0]"
    />

    <!-- 内容区：桌面端（xl+）评价列表为主列，开课记录/相关课程/AI 总结收纳右栏；移动端概括优先，按 DOM 顺序纵向堆叠（页头 → 评分 → AI 总结 → 评价 → 开课/相关）。 -->
    <div class="mt-6 grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,340px)] xl:items-start">
      <!-- 移动端 AI 总结实例：置于评分 hero 与评价列表之间；桌面隐藏，由右栏实例（hidden xl:block）接管 -->
      <div class="xl:hidden">
        <AISummaryCard :course-id="page.props.course.id" />
      </div>
    <section id="course-reviews" class="min-w-0 scroll-mt-4 xl:order-1" aria-labelledby="course-reviews-title">
      <div class="mb-3 flex items-center justify-between gap-2">
        <h2 id="course-reviews-title" class="text-base font-semibold text-base-content">
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

      <div
        v-if="focusOfferingId"
        class="mb-3 flex items-center justify-between gap-2 rounded-lg border border-primary/25 bg-info/10 px-3 py-2 text-[12px] text-base-content/75"
      >
        <span class="min-w-0 truncate">
          {{ t('courseDetailPage.offeringFocusLabel') }}：{{ offeringLabel(focusOfferingId) }}
        </span>
        <button
          type="button"
          class="shrink-0 font-medium text-primary hover:underline"
          @click="setOfferingFocus(0)"
        >
          {{ t('courseDetailPage.offeringFocusClear') }}
        </button>
      </div>

      <p v-if="reviewError" class="mb-3 rounded border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">
        {{ reviewError }}
      </p>

      <!-- 写评 / 编辑表单：外层 grid wrapper 承担打开时的真实高度展开（0fr→1fr），
           内层保持内容高度自适应；关闭由 CSS 做柔和的固定位移淡出。 -->
      <Transition name="gf-local-expand">
      <div v-if="formVisible" class="grid gf-local-expand">
        <div class="min-h-0 overflow-hidden">
        <form
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
            <div class="flex items-center gap-1" role="radiogroup" :aria-label="t('courseDetailPage.rating')">
              <button
                v-for="star in 5"
                :key="star"
                type="button"
                class="rounded p-0.5 transition hover:scale-110"
                role="radio"
                :aria-checked="formRating === star"
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
            <span class="mb-1.5 block text-[13px] text-base-content/70">{{ t('courseDetailPage.content') }}</span>
            <!-- 不用 overflow-hidden 裁圆角：那会把 more 下拉（.vditor-hint）一并裁掉；
                 改为让内层 .vditor 继承圆角，下拉可自然溢出覆盖到下方表单 -->
            <div class="relative rounded-[var(--gf-radius-box)] border border-line/70 bg-base-100 [&_.vditor]:rounded-[inherit]">
              <!-- 与发布/回复同款富文本编辑器；slim-mobile：课评表单嵌套层级多（vw-64），
                   移动端用 7 项精简行，≤320px 视口也单行完整（10 项会在 320px 裁掉 more） -->
              <VditorOfficial
                ref="reviewEditor"
                v-model="formContent"
                :height="380"
                :compact="true"
                :slim-mobile="true"
                :placeholder="t('courseDetailPage.contentPlaceholder')"
                @upload="uploadReviewImages"
                @error="onReviewEditorError"
              />
              <div
                v-if="!reviewEditorReady"
                class="absolute inset-0 z-10 flex items-center justify-center gap-2 rounded-[inherit] bg-base-100/60 text-sm"
                :class="reviewEditorFailed ? 'text-error' : 'text-base-content/55'"
                :role="reviewEditorFailed ? 'alert' : 'status'"
                aria-live="polite"
              >
                <Loader2 v-if="!reviewEditorFailed" class="h-4 w-4 animate-spin" />
                <span>{{ reviewEditorFailed ? t('common.loadFailed') : t('common.loadingShort') }}</span>
              </div>
            </div>
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
            class="gf-button gf-button-md gf-button-primary min-w-[8rem]"
            :disabled="formSubmitting || formSubmitState === 'success'"
            :aria-busy="formSubmitting"
            :class="{ 'gf-submit-error': formSubmitState === 'error' }"
          >
            <template v-if="formSubmitting">
              <Loader2 class="h-4 w-4 animate-spin" aria-hidden="true" />
              {{ t('courseDetailPage.submitting') }}
            </template>
            <template v-else-if="formSubmitState === 'success'">
              <Check class="h-4 w-4" aria-hidden="true" />
              {{ editingReviewId ? t('courseDetailPage.updateSuccess') : t('courseDetailPage.submitSuccess') }}
            </template>
            <template v-else>
              {{ t('courseDetailPage.submitReview') }}
            </template>
          </button>
        </div>
        </form>
        </div>
      </div>
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
          :id="`course-review-${review.id}`"
          class="rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 sm:bg-base-100"
          :class="{ 'gf-review-highlight': highlightedReviewId === review.id }"
        >
          <!-- 头部：左头像 + 作者/开课班信息；右侧星级（对齐 serverless 评论卡） -->
          <div class="flex items-start justify-between gap-3">
            <div class="flex min-w-0 items-center gap-2.5">
              <img
                :src="reviewAvatar(review, 36)"
                :alt="authorLabel(review.author)"
                class="h-9 w-9 shrink-0 rounded-full object-cover"
                loading="lazy"
              />
              <div class="min-w-0">
                <p class="truncate text-[13px] font-medium leading-5 text-base-content">{{ authorLabel(review.author) }}</p>
                <p class="truncate text-[11px] leading-4 text-base-content/45">
                  {{ offeringLabel(review.offeringId) }} · {{ reviewDateLabel(review) }}
                </p>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-0.5 pt-1" role="img" :aria-label="`${review.rating ?? 0} ${t('courseDetailPage.stars')}`">
              <Star
                v-for="star in 5"
                :key="star"
                class="h-4 w-4"
                :class="review.rating && review.rating >= star ? 'fill-warning text-warning' : 'text-base-content/20'"
              />
            </div>
          </div>

          <div
            v-if="review.contentHtml"
            v-code-highlight
            v-math-render
            class="gf-prose gf-prose-post mt-3 text-[14px] leading-6"
            v-html="review.contentHtml"
          />

          <!-- 功能区：点赞 / 点踩 / 分享评论 / 编辑 / 删除 / 举报 + 评价短码（对齐 serverless） -->
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <button
              type="button"
              class="inline-flex min-w-fit shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-1.5 text-xs font-medium leading-none transition hover:-translate-y-px sm:gap-2 sm:px-3"
              :class="review.viewer.isHelpful
                ? 'border-amber-200 bg-amber-50 text-amber-700'
                : 'border-line/70 bg-base-100 text-base-content/70 hover:border-line hover:text-base-content/90'"
              :disabled="helpfulBusyIds.includes(review.id)"
              :aria-pressed="review.viewer.isHelpful"
              :title="review.viewer.isHelpful ? t('courseDetailPage.helpfulUndo') : t('courseDetailPage.helpful')"
              @click="toggleHelpful(review)"
            >
              <ThumbsUp class="h-4 w-4" :class="review.viewer.isHelpful ? 'text-amber-600' : 'text-base-content/45'" />
              <span class="tabular-nums">{{ review.helpfulCount }}</span>
              <span class="text-[10px] font-semibold opacity-80">{{ t('courseDetailPage.like') }}</span>
            </button>
            <button
              type="button"
              class="inline-flex min-w-fit shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-1.5 text-xs font-medium leading-none transition hover:-translate-y-px sm:gap-2 sm:px-3"
              :class="review.viewer.isDisliked
                ? 'border-red-200 bg-red-50 text-red-700'
                : 'border-line/70 bg-base-100 text-base-content/70 hover:border-line hover:text-base-content/90'"
              :disabled="dislikeBusyIds.includes(review.id)"
              :aria-pressed="review.viewer.isDisliked"
              :title="review.viewer.isDisliked ? t('courseDetailPage.dislikeUndo') : t('courseDetailPage.dislike')"
              @click="toggleDislike(review)"
            >
              <ThumbsDown class="h-4 w-4" :class="review.viewer.isDisliked ? 'text-red-600' : 'text-base-content/45'" />
              <span class="tabular-nums">{{ review.dislikeCount }}</span>
              <span class="text-[10px] font-semibold opacity-80">{{ t('courseDetailPage.dislike') }}</span>
            </button>
            <button
              type="button"
              class="inline-flex min-w-fit shrink-0 items-center gap-1.5 rounded-full border border-line/70 bg-base-100 px-2.5 py-1.5 text-xs font-medium leading-none text-base-content/70 transition hover:-translate-y-px hover:border-line hover:text-base-content/90 active:scale-[0.96] sm:gap-2 sm:px-3"
              :disabled="shareBusyId != null"
              :title="t('courseDetailPage.shareTitle')"
              @click="openShareDialog(review, $event)"
            >
              <Share2 class="h-4 w-4 text-base-content/45" />
              <span class="text-[10px] font-semibold opacity-80">{{ shareBusyId === review.id ? t('courseDetailPage.shareGenerating') : t('courseDetailPage.shareComment') }}</span>
            </button>
            <button
              v-if="review.viewer.canEdit"
              type="button"
              class="inline-flex min-w-fit shrink-0 items-center gap-1.5 rounded-full border border-line/70 bg-base-100 px-2.5 py-1.5 text-xs font-medium leading-none text-base-content/70 transition hover:-translate-y-px hover:border-line hover:text-base-content/90 sm:gap-2 sm:px-3"
              @click="startEdit(review)"
            >
              <Pencil class="h-4 w-4 text-base-content/45" />
              <span class="text-[10px] font-semibold opacity-80">{{ t('courseDetailPage.edit') }}</span>
            </button>
            <button
              v-if="review.viewer.canDelete"
              type="button"
              class="inline-flex min-w-fit shrink-0 items-center gap-1.5 rounded-full border border-line/70 bg-base-100 px-2.5 py-1.5 text-xs font-medium leading-none text-base-content/70 transition hover:-translate-y-px hover:border-error/40 hover:bg-error/10 hover:text-error sm:gap-2 sm:px-3"
              @click="askRemoveReview(review)"
            >
              <Trash2 class="h-4 w-4 text-base-content/45" />
              <span class="text-[10px] font-semibold opacity-80">{{ t('courseDetailPage.delete') }}</span>
            </button>
            <button
              v-if="page.layout.viewer.isAuthenticated && !review.viewer.canEdit"
              type="button"
              class="inline-flex min-w-fit shrink-0 items-center gap-1.5 rounded-full border border-line/70 bg-base-100 px-2.5 py-1.5 text-xs font-medium leading-none text-base-content/70 transition hover:-translate-y-px hover:border-error/40 hover:bg-error/10 hover:text-error sm:gap-2 sm:px-3"
              @click="openReport(review)"
            >
              <Flag class="h-4 w-4 text-base-content/45" />
              <span class="text-[10px] font-semibold opacity-80">{{ t('courseDetailPage.report') }}</span>
            </button>
            <span class="ml-auto inline-flex shrink-0 items-center gap-1 whitespace-nowrap text-[11px] tabular-nums text-base-content/45">
              <Hash class="h-3.5 w-3.5" aria-hidden="true" />
              <span class="font-mono">{{ reviewSqid(review.id) }}</span>
            </span>
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

      <!-- 右栏（xl+ 排右侧自滚动，顶部对齐顶栏 80px；移动端在评价列表之后） -->
      <div class="min-w-0 space-y-4 xl:order-2 xl:sticky xl:top-20 xl:max-h-[calc(100vh-5rem)] xl:overflow-y-auto xl:[scrollbar-width:thin]">
    <section class="gf-panel p-5" aria-labelledby="course-offerings-title">
      <h2 id="course-offerings-title" class="mb-3 inline-flex items-center gap-1.5 text-sm font-semibold text-base-content">
        <CalendarDays class="h-4 w-4 text-base-content/45" />
        {{ t('courseDetailPage.offeringsTitle') }}
      </h2>
      <EmptyState
        v-if="!props.course.offerings?.length"
        :icon="CalendarDays"
        :title="t('courseDetailPage.noOfferings')"
      />
      <ul v-else class="divide-y divide-line/60" :aria-label="t('courseDetailPage.offeringsTitle')">
        <li
          v-for="offering in props.course.offerings"
          :key="offering.id"
          class="py-2.5 first:pt-0 last:pb-0"
        >
          <div class="flex flex-wrap items-center gap-1.5">
            <span class="gf-badge gf-badge-muted">{{ shortTerm(offering.termCode) }}</span>
            <span v-if="offering.className || offering.classCode" class="gf-badge gf-badge-info text-[11px]">
              {{ offering.className || offering.classCode }}
            </span>
          </div>
          <p class="mt-1 text-[13px] leading-5 text-base-content/70">
            {{ [offering.campus, offering.faculty, offering.instructors?.join('、')].filter(Boolean).join(' · ') }}
          </p>
        </li>
      </ul>
    </section>

      <!-- 右栏评分卡实例：置于信息栏顶部（评分 → AI 总结 → 相关课程）；移动端由页头下实例（xl:hidden）接管 -->
      <div class="hidden xl:block">
        <RatingSummaryCard
          :rating-avg="page.props.course.ratingAvg ?? null"
          :review-count="statsReviewCount"
          :distribution="page.props.course.ratingDistribution ?? [0, 0, 0, 0, 0]"
        />
      </div>

      <div class="hidden xl:block">
        <AISummaryCard :course-id="page.props.course.id" />
      </div>

      <!-- 相关课程：桌面与移动端均默认展示（无折叠） -->
      <section class="gf-panel p-5" aria-labelledby="course-related-title">
        <h2 id="course-related-title" class="mb-3 inline-flex items-center gap-1.5 text-sm font-semibold text-base-content">
          <BookOpen class="h-4 w-4 text-base-content/45" />
          {{ t('courseDetailPage.relatedTitle') }}
        </h2>

        <p v-if="relatedError" class="mb-3 rounded border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">
          {{ relatedError }}
        </p>

        <EmptyState v-if="relatedLoading" :icon="BookOpen" :title="t('common.loading')" loading />

        <div v-else-if="!relatedError" id="course-related-panel" class="divide-y divide-line/60">
          <div class="pb-3">
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

          <div class="pt-3">
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
                      {{ item.primaryCode }}<template v-if="item.teacherName"> · {{ item.teacherName }}</template><template v-else> · {{ t('courseDetailPage.noTeacher') }}</template>
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
          <div v-if="related?.lineage?.length" class="pt-3">
            <h3 class="mb-3 text-sm font-semibold text-base-content">
              {{ t('courseDetailPage.lineageTitle') }}
            </h3>
            <ul class="space-y-2">
              <li
                v-for="item in related?.lineage ?? []"
                :key="item.relationId"
                class="flex items-center justify-between gap-3 rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 px-3 py-2"
              >
                <span class="min-w-0">
                  <span class="block truncate text-sm">
                    <a
                      v-if="lineageFromHref(item)"
                      :href="lineageFromHref(item)"
                      class="font-medium text-base-content transition hover:text-primary"
                    >
                      {{ item.fromName }}
                    </a>
                    <span v-else class="font-medium text-base-content/60">{{ item.fromName }}</span>
                    <span class="mx-1 text-base-content/35">→</span>
                    <a
                      v-if="lineageToHref(item)"
                      :href="lineageToHref(item)"
                      class="font-medium text-base-content transition hover:text-primary"
                    >
                      {{ item.toName }}
                    </a>
                    <span v-else class="font-medium text-base-content">{{ item.toName }}</span>
                  </span>
                </span>
                <span class="gf-badge gf-badge-ghost shrink-0 text-[11px]">{{ relationLabel(item.relationType) }}</span>
              </li>
            </ul>
          </div>
        </div>
      </section>
      </div>
    </div>

    <!-- 举报弹窗（a11y：焦点陷阱 + Esc + 焦点归还 + 滚动锁） -->
    <Teleport to="body">
      <Transition name="gf-modal">
      <div
        v-if="pendingReport"
        ref="reportDialogEl"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="REPORT_TITLE_ID"
        @click.self="closeReport"
        @keydown="onReportKeydown"
      >
        <div class="w-full max-w-md rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <div class="flex items-start justify-between gap-3">
            <h2 :id="REPORT_TITLE_ID" class="text-base font-bold text-base-content">{{ t('courseDetailPage.reportTitle') }}</h2>
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

    <!-- 删除确认 Dialog（受控，替代 window.confirm；a11y：焦点陷阱 + Esc + 打开即聚焦） -->
    <Teleport to="body">
      <Transition name="gf-modal">
      <div
        v-if="pendingDelete"
        ref="deleteDialogEl"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="alertdialog"
        aria-modal="true"
        :aria-labelledby="DELETE_TITLE_ID"
        @click.self="cancelRemoveReview"
        @keydown="onDeleteKeydown"
      >
        <div class="w-full max-w-sm rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <div class="flex items-start justify-between gap-3">
            <h2 :id="DELETE_TITLE_ID" class="text-base font-bold text-base-content">{{ t('courseDetailPage.confirmDeleteTitle') }}</h2>
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

    <!-- 分享评论弹窗（a11y：焦点陷阱 + Esc + 焦点归还 + 滚动锁；预览与导出双实例） -->
    <Teleport to="body">
      <Transition name="gf-modal">
      <div
        v-if="sharePreview"
        ref="shareDialogEl"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="SHARE_TITLE_ID"
        @click.self="closeShare"
        @keydown="onShareKeydown"
      >
        <div class="flex max-h-[calc(100dvh-2rem)] w-full max-w-[960px] flex-col overflow-hidden rounded-[var(--gf-radius-box)] bg-base-100 shadow-lg ring-1 ring-line">
          <div class="flex flex-col gap-3 border-b border-line/70 p-4 sm:flex-row sm:items-start sm:justify-between sm:p-5">
            <div class="min-w-0">
              <h2 :id="SHARE_TITLE_ID" class="text-[11px] font-black uppercase tracking-[0.28em] text-base-content/55">
                {{ t('courseDetailPage.shareTitle') }}
              </h2>
              <p class="mt-1.5 truncate bg-gradient-to-r from-sky-700 via-slate-700 to-cyan-800 bg-clip-text text-[11px] font-bold text-transparent sm:text-xs">
                {{ t('courseDetailPage.shareTagline') }}
              </p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <button
                type="button"
                class="gf-button gf-button-sm gf-button-primary active:scale-[0.96]"
                :disabled="shareSaving"
                @click="saveShare"
              >
                <Loader2 v-if="shareSaving" class="h-4 w-4 animate-spin" />
                <Download v-else class="h-4 w-4" />
                {{ shareSaving ? t('common.loadingShort') : t('courseDetailPage.shareSave') }}
              </button>
              <button
                type="button"
                class="inline-flex h-8 items-center gap-1.5 rounded-md border border-line/70 bg-base-100 px-3 text-sm font-medium text-base-content/70 transition hover:bg-base-200 hover:text-base-content active:scale-[0.96]"
                @click="closeShare"
              >
                <X class="h-4 w-4" />
                {{ t('common.close') }}
              </button>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain bg-base-200/50 p-4 sm:p-6">
            <!-- 打印墨盒条：模拟打印机出纸，视觉上把卡片「打印」出来（对齐 serverless 分享弹窗） -->
            <div aria-hidden="true" class="share-printer-bar mx-auto mb-4 w-full max-w-[760px] rounded-[28px] bg-gradient-to-b from-base-content/70 to-base-content/25 px-4 py-4 shadow-inner">
              <div class="mx-auto h-4 w-48 rounded-full bg-base-100/60" />
            </div>
            <!-- 预览实例：响应式宽度 -->
            <div class="share-paper share-paper-enter mx-auto w-full max-w-[760px] overflow-hidden rounded-[26px] bg-white shadow-[0_28px_60px_rgba(14,165,233,0.16)]">
              <div class="bg-gradient-to-br from-sky-50 via-white to-cyan-50 px-5 pb-6 pt-6 sm:px-8 sm:pb-8 sm:pt-7">
                <div class="grid grid-cols-1 gap-4 border-b border-slate-100 pb-5 sm:grid-cols-[minmax(0,1fr)_auto]">
                  <div class="min-w-0">
                    <div class="text-[11px] font-black tracking-[0.18em] text-slate-400">{{ t('courseDetailPage.shareBrand') }}</div>
                    <div class="mt-2 text-[20px] font-black leading-tight text-slate-900 sm:text-[28px]">{{ props.course.name }}</div>
                    <div class="mt-2 inline-flex items-center rounded-full bg-slate-900 px-3 py-1 text-xs font-bold text-white">{{ props.course.primaryCode }}</div>
                  </div>
                  <div class="min-w-[138px] rounded-3xl bg-white/90 px-4 py-3 text-right shadow-sm ring-1 ring-slate-100 sm:min-w-[150px]">
                    <div class="text-[11px] font-bold text-slate-400">{{ t('courseDetailPage.shareCourseRating') }}</div>
                    <div class="mt-1 text-lg font-black tabular-nums text-amber-500 sm:text-xl">{{ ratingAvg != null ? ratingAvg.toFixed(1) : '-' }} / 5.0</div>
                    <div class="mt-2 text-[11px] font-bold text-slate-400">{{ t('courseDetailPage.shareReviewCount') }}</div>
                    <div class="mt-1 text-base font-black tabular-nums text-slate-800 sm:text-lg">{{ statsReviewCount }} {{ t('courseDetailPage.shareCountUnit') }}</div>
                  </div>
                </div>

                <div class="mt-5 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                  <div class="flex min-w-0 items-center gap-3">
                    <img :src="sharePreview.avatarUrl" :alt="authorLabel(sharePreview.review.author)" class="h-12 w-12 rounded-2xl object-cover ring-1 ring-slate-100 sm:h-14 sm:w-14" />
                    <div class="min-w-0">
                      <div class="truncate text-base font-black text-slate-900 sm:text-lg">{{ authorLabel(sharePreview.review.author) }}</div>
                      <div class="mt-1 text-xs font-semibold text-slate-400">{{ offeringLabel(sharePreview.review.offeringId) }} · {{ reviewDateLabel(sharePreview.review) }}</div>
                    </div>
                  </div>
                  <div class="shrink-0 rounded-full bg-amber-50 px-3.5 py-1.5 text-[13px] font-black tabular-nums text-amber-600 ring-1 ring-amber-100 sm:px-4 sm:py-2 sm:text-sm">
                    {{ (sharePreview.review.rating ?? 0).toFixed(1) }} / 5
                  </div>
                </div>

                <div class="mt-5 flex flex-wrap gap-2">
                  <span class="rounded-full bg-cyan-50 px-3 py-1 text-xs font-bold text-cyan-700 ring-1 ring-cyan-100">
                    {{ t('courseDetailPage.shareTeacherPrefix') }}{{ props.course.teacherName || t('courseDetailPage.noTeacher') }}
                  </span>
                  <span class="rounded-full bg-indigo-50 px-3 py-1 text-xs font-bold text-indigo-700 ring-1 ring-indigo-100">
                    {{ t('courseDetailPage.shareTermPrefix') }}{{ offeringLabel(sharePreview.review.offeringId) }}
                  </span>
                  <span class="rounded-full bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-700 ring-1 ring-emerald-100">
                    {{ t('courseDetailPage.shareCodePrefix') }}{{ reviewSqid(sharePreview.review.id) }}
                  </span>
                </div>

                <div class="gf-prose gf-prose-post mt-6 rounded-[24px] border border-sky-100 bg-white px-4 py-4 text-[14px] leading-7 text-slate-700 sm:px-6 sm:py-5 sm:text-[15px] sm:leading-8" v-html="sharePreview.markdownHtml" />

                <div class="mt-6 flex items-center justify-between text-xs font-semibold text-slate-400">
                  <span>{{ t('courseDetailPage.shareWatermark') }}</span>
                  <span>{{ t('courseDetailPage.shareSite') }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      </Transition>

      <!-- 导出实例：固定 760px 白卡，隐藏于视口外（html-to-image 捕获）。
           ref 必须绑在内层卡片而非隐藏容器：html-to-image 会复制被捕获节点自身的
           computed style，容器上的 opacity-0 会把整张导出图变成纯白（对齐 serverless：
           ref 挂在 paper 节点，隐藏容器只负责视口外定位）。 -->
      <div
        v-if="sharePreview"
        class="pointer-events-none fixed -left-[10000px] top-0 opacity-0"
        aria-hidden="true"
      >
        <div
          ref="shareExportEl"
          class="share-paper w-[760px] overflow-hidden rounded-[26px] bg-white shadow-[0_28px_60px_rgba(14,165,233,0.16)]"
        >
          <div class="bg-gradient-to-br from-sky-50 via-white to-cyan-50 px-8 pb-8 pt-7">
            <div class="grid grid-cols-[minmax(0,1fr)_auto] gap-4 border-b border-slate-100 pb-5">
              <div class="min-w-0">
                <div class="text-[11px] font-black tracking-[0.18em] text-slate-400">{{ t('courseDetailPage.shareBrand') }}</div>
                <div class="mt-2 text-[28px] font-black leading-tight text-slate-900">{{ props.course.name }}</div>
                <div class="mt-2 inline-flex items-center rounded-full bg-slate-900 px-3 py-1 text-xs font-bold text-white">{{ props.course.primaryCode }}</div>
              </div>
              <div class="min-w-[150px] rounded-3xl bg-white/90 px-4 py-3 text-right shadow-sm ring-1 ring-slate-100">
                <div class="text-[11px] font-bold text-slate-400">{{ t('courseDetailPage.shareCourseRating') }}</div>
                <div class="mt-1 text-xl font-black tabular-nums text-amber-500">{{ ratingAvg != null ? ratingAvg.toFixed(1) : '-' }} / 5.0</div>
                <div class="mt-2 text-[11px] font-bold text-slate-400">{{ t('courseDetailPage.shareReviewCount') }}</div>
                <div class="mt-1 text-lg font-black tabular-nums text-slate-800">{{ statsReviewCount }} {{ t('courseDetailPage.shareCountUnit') }}</div>
              </div>
            </div>

            <div class="mt-5 flex items-center justify-between gap-4">
              <div class="flex min-w-0 items-center gap-3">
                <img :src="sharePreview.avatarUrl" :alt="authorLabel(sharePreview.review.author)" class="h-14 w-14 rounded-2xl object-cover ring-1 ring-slate-100" />
                <div class="min-w-0">
                  <div class="truncate text-lg font-black text-slate-900">{{ authorLabel(sharePreview.review.author) }}</div>
                  <div class="mt-1 text-xs font-semibold text-slate-400">{{ offeringLabel(sharePreview.review.offeringId) }} · {{ reviewDateLabel(sharePreview.review) }}</div>
                </div>
              </div>
              <div class="shrink-0 rounded-full bg-amber-50 px-4 py-2 text-sm font-black tabular-nums text-amber-600 ring-1 ring-amber-100">
                {{ (sharePreview.review.rating ?? 0).toFixed(1) }} / 5
              </div>
            </div>

            <div class="mt-5 flex flex-wrap gap-2">
              <span class="rounded-full bg-cyan-50 px-3 py-1 text-xs font-bold text-cyan-700 ring-1 ring-cyan-100">
                {{ t('courseDetailPage.shareTeacherPrefix') }}{{ props.course.teacherName || t('courseDetailPage.noTeacher') }}
              </span>
              <span class="rounded-full bg-indigo-50 px-3 py-1 text-xs font-bold text-indigo-700 ring-1 ring-indigo-100">
                {{ t('courseDetailPage.shareTermPrefix') }}{{ offeringLabel(sharePreview.review.offeringId) }}
              </span>
              <span class="rounded-full bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-700 ring-1 ring-emerald-100">
                {{ t('courseDetailPage.shareCodePrefix') }}{{ reviewSqid(sharePreview.review.id) }}
              </span>
            </div>

            <div class="gf-prose gf-prose-post mt-6 rounded-[24px] border border-sky-100 bg-white px-6 py-5 text-[15px] leading-8 text-slate-700" v-html="sharePreview.markdownHtml" />

            <div class="mt-6 flex items-center justify-between text-xs font-semibold text-slate-400">
              <span>{{ t('courseDetailPage.shareWatermark') }}</span>
              <span>{{ t('courseDetailPage.shareSite') }}</span>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 写评模板选择器 -->
    <CourseReviewTemplateSelector
      :open="templateSelectorOpen"
      @close="templateSelectorOpen = false"
      @select="applyTemplate($event.id, $event.content)"
    />
  </div>
</template>
