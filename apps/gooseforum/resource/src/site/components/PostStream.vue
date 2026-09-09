<script lang="ts">
import type { TopicDetailPayload } from '@gooseforum/client'

export interface PostStreamTopicActions {
  likeCount: number
  isLiked: boolean
  isBookmarked: boolean
  isWatched: boolean
  processStatus: number
  authorDeleted: boolean
  moderatorRemoved: boolean
  isOwnTopic: boolean
  canModerateTopic: boolean
  createdAt: string
  updatedAt: string
  replyCount: number
  viewCount: number
  maxPostNo: number
  participants: TopicDetailPayload['participants']
  author: TopicDetailPayload['author']
  description: string
}
</script>

<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, ref, Teleport, useSlots, watch } from 'vue'
import { AlertTriangle, Ban, Bell, BookOpen, Bookmark, ChevronsUp, Clock, CornerDownLeft, Flag, Heart, HelpCircle, History, Loader2, MoreHorizontal, PencilLine, RotateCcw, Share2, Sparkles, Trash2, X } from '@lucide/vue'
import { PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui'
import { bookmarkTopic, deletePost, deleteTopic, getPostRevisions, getPostWindow, likeTopic, createPost, sensitiveWordsFromError, submitReport, updateModerationTopicStatus, updateModerationPostStatus, updatePost, watchTopic, likePost, bookmarkPost, reportContentEvent, privacyEraseContent, type PostRevisionResult } from '@/runtime/api'
import { formatDateTime, formatNumber } from '@/runtime/format'
import { useFlashMessages } from '@/runtime/flash-message'
import { fetchPage } from '@/runtime/router'
import { showUserCard } from '@/runtime/user-card-events'
import { measurePostViewportProgressFromRects } from '@/runtime/post-viewport-progress'
import { usePostViewMode } from '@/runtime/post-view-mode'
import { buildReplyForest, flattenReplyForest, postTreeIndentLevel, type ForestRow } from '@/runtime/reply-forest'
import MarkdownImageViewer from '@/site/components/MarkdownImageViewer.vue'
import PostPositionRail from '@/site/components/PostPositionRail.vue'
import PostReplyReference from '@/site/components/PostReplyReference.vue'
import PostReplyRow from '@/site/components/PostReplyRow.vue'
import TopicFloatingControls from '@/site/components/TopicFloatingControls.vue'
import TopicImageGallery from '@/site/components/TopicImageGallery.vue'
import TopicList from '@/site/components/TopicList.vue'
import UserAvatar from '@/site/components/UserAvatar.vue'
import { buildBeamAvatarDataUri } from '@/site/utils/course-review-share'
import type { PostPayload, PostWindowPayload, ReplyTargetPayload, TopicPayload, ViewerPayload } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'
import { useCaptchaChallenge } from '@/site/composables/useCaptchaChallenge'
import { useQuickPublish } from '@/site/composables/useQuickPublish'

const props = withDefaults(defineProps<{
  topicId: number
  topicTitle: string
  contentType?: 0 | 1 | 2 | 3
  topicImages?: string[]
  categories?: Array<{ id: number; name?: string }>
  initialPostStream: PostWindowPayload
  viewer: ViewerPayload
  canPost: boolean
  /** 内容级操作状态（首楼操作区 + 概览栏）。WikiPage 等场景不传则不渲染。 */
  topicActions?: PostStreamTopicActions
  /** 互动状态独立供给（Wiki 页面正文区操作与评论流共用同一内容）。 */
  interactions?: { likeCount: number; isLiked: boolean; isBookmarked: boolean; isWatched: boolean }
  hotTopics?: TopicPayload[]
  /** 隐藏首楼（postNo === 1）：Wiki 正文已在页面上方渲染，评论流不再重复。 */
  hideFirstPost?: boolean
  /** 隐藏首楼时是否连其回复也隐藏：Wiki 页只保留回复栏，不展示内容楼层列表。 */
  initialPostStreamHidden?: boolean
  /** 滚动时是否改写 /p/post/:id 路径（Wiki 页面关闭）。 */
  syncUrl?: boolean
  /** 初始流为空时自动加载第一页（Wiki 页面无 SSR 评论流）。 */
  autoLoadFirstWindow?: boolean
  /** 是否允许匿名发布（wiki 评论区，issue #524）；true 时发布栏显示匿名勾选项。 */
  allowAnonymous?: boolean
  /** 宽版卡片（默认）：右栏概览 + 卡片右伸 292px（对齐内容页 rail）。嵌入 Wiki 正文区时关闭。 */
  wide?: boolean
}>(), {
  wide: true,
})

const slots = useSlots()
const hasAside = computed(() => Boolean(props.topicActions) || Boolean(slots.aside))
// Wiki 文章页（initialPostStreamHidden）只保留回复栏：首楼楼层已被 sortedPosts 过滤，
// 这里同时隐藏底部「热门内容」内容列表，避免正文下方残留内容列表区块。
const hasHotTopics = computed(() => Boolean(props.hotTopics?.length) && !props.initialPostStreamHidden)

const emit = defineEmits<{
  'topic-state': [likeCount: number]
  'interaction-state': [state: { likeCount: number; isLiked: boolean; isBookmarked: boolean; isWatched: boolean }]
}>()

const { t } = useI18n()
const { push: pushFlash } = useFlashMessages()
const PostComposer = defineAsyncComponent(() => import('@/site/components/PostComposer.vue'))
const initialPostStream = props.initialPostStream
const initialPosts = initialPostStream.posts
const {
  captchaRequired,
  captchaId,
  captchaImg,
  captchaCode,
  captchaLoading,
  loadCaptcha,
  clearCaptcha,
  challengeFromError,
} = useCaptchaChallenge()
const postContent = ref('')
const targetPostId = ref(0)
const anonymous = ref(false)
const likeCount = ref(props.interactions?.likeCount ?? props.topicActions?.likeCount ?? 0)
const isLiked = ref(props.interactions?.isLiked ?? props.topicActions?.isLiked ?? false)
const isBookmarked = ref(props.interactions?.isBookmarked ?? props.topicActions?.isBookmarked ?? false)
const isWatched = ref(props.interactions?.isWatched ?? props.topicActions?.isWatched ?? false)
const actionMessage = ref('')
const actingLike = ref(false)
const actingBookmark = ref(false)
const actingWatch = ref(false)
const actingModeration = ref(false)
const moreActionsOpen = ref(false)
const submitting = ref(false)
const deletingPostId = ref(0)
const deletingTopic = ref(false)
const editingPostId = ref(0)
const savingEditPostId = ref(0)
const postDraftBeforeEdit = ref('')
const targetPostBeforeEdit = ref(0)
const pendingDeletePost = ref<PostPayload | null>(null)
const pendingDeleteTopic = ref(false)
const pendingModerationAction = ref<'ban' | 'unban' | null>(null)
const historyPost = ref<PostPayload | null>(null)
const historyVersions = ref<PostRevisionResult['versions'] | null>(null)
const historyLoading = ref(false)
const historyLoadingMore = ref(false)
const historyHasMore = ref(false)
const historyBeforeVersion = ref(0)
const historyError = ref('')
// 弹窗请求代次：关闭/重新打开弹窗时自增，在途响应回来时丢弃过期结果，
// 避免旧请求写入下一个弹窗的状态（加载更多进行中关闭的竞态）。
const historyRequestSeq = ref(0)
const pendingReport = ref<{ targetType: 'topic' | 'post'; targetId: number; title: string; excerpt: string } | null>(null)
const reportReason = ref('spam')
const reportNote = ref('')
const reportSubmitting = ref(false)
const reportError = ref('')
const moderatingPostIds = ref<number[]>([])
const posts = ref<PostPayload[]>([...initialPosts])
const postPageStarts = ref<number[]>(initialPosts[0]?.postNo ? [initialPosts[0].postNo] : [])
const replyTargets = ref<ReplyTargetPayload[]>([...(initialPostStream.replyTargets || [])])
const replyTargetMap = computed(() => new Map(replyTargets.value.map((target) => [target.id, target])))
const topicProcessStatus = ref(props.topicActions?.processStatus ?? 0)
const targetPost = computed(() => posts.value.find((post) => post.id === targetPostId.value))
// @mention 本地上下文（issue #564）：回复目标 > 主题作者 > 参与者，按此优先级传入 composer
const mentionUsers = computed(() => {
  const list: Array<import('@/runtime/mention').MentionUser> = []
  const targetAuthor = targetPost.value?.author
  if (targetAuthor && !targetPost.value?.isAnonymous && targetAuthor.id > 0) {
    list.push({ ...targetAuthor, tag: 'reply-target' })
  }
  const topicAuthor = props.topicActions?.author
  if (topicAuthor && topicAuthor.id > 0) {
    list.push({ ...topicAuthor, tag: 'topic-author' })
  }
  for (const participant of props.topicActions?.participants ?? []) {
    if (participant.id > 0) list.push({ ...participant, tag: 'participant' })
  }
  return list
})
const postHasBefore = ref(initialPostStream.hasBefore)
const postHasAfter = ref(initialPostStream.hasAfter)
const postBeforePostNo = ref(initialPostStream.beforePostNo || firstPostNo(initialPosts))
const postAfterPostNo = ref(initialPostStream.afterPostNo || lastPostNo(initialPosts))
const postMaxNo = ref(initialPostStream.maxPostNo || initialMaxPostNo())
const postAutoLoadAfter = ref(true)
const loadingPostWindow = ref(false)
const loadingPostDirection = ref<'before' | 'after' | 'anchor' | null>(null)
const postWindowError = ref('')
const deleteErrorMessage = ref('')
const errorMessage = ref('')
const successMessage = ref('')
const sensitiveWords = ref<string[]>([])
const postLoadMoreEl = ref<HTMLElement | null>(null)
const markdownImageViewer = ref<InstanceType<typeof MarkdownImageViewer> | null>(null)
const composerOpen = ref(false)
const composerMode = computed(() => editingPostId.value ? 'edit' : 'create')
// PostComposer 首次打开后保持挂载：若随开合销毁重建，内层 <Transition> 首次挂载不播 enter、
// leave 也会被整体销毁绕过，弹簧上弹/收回动画将失效。此值只在首次打开时置 true，永不重置。
const composerMounted = ref(false)
watch(composerOpen, (open) => {
  if (open) composerMounted.value = true
}, { immediate: true })
const mobilePostRailOpen = ref(false)
const activePostNo = ref(firstPostNo(initialPosts) || 1)
const postRailProgressCurrent = ref(0)
const postRailProgressStart = ref(0)
const postRailProgressEnd = ref(0)
const postMaxRange = computed(() => Math.max(postMaxNo.value, ...posts.value.map((post) => post.postNo || 0)))
const firstPost = computed(() => posts.value.find((post) => post.postNo === 1))
const hasPostRail = computed(() => postMaxRange.value > 0)
const postRailCurrentNo = computed(() => {
  const fallback = firstPostNo(posts.value) || 1
  return clampPostNo(activePostNo.value || fallback)
})
const postRailCurrentLabel = computed(() => {
  const activePost = posts.value.find((post) => post.postNo === postRailCurrentNo.value)
  return activePost ? formatRailDate(activePost.createdAt) : props.topicActions ? formatRailDate(props.topicActions.createdAt) : ''
})
const postRailStartLabel = computed(() => props.topicActions ? formatRailDate(props.topicActions.createdAt) : '')
const postRailEndLabel = computed(() => props.topicActions
  ? formatRailDate(postHasAfter.value ? props.topicActions.updatedAt : posts.value[posts.value.length - 1]?.createdAt || props.topicActions.updatedAt)
  : '')
const postRailBusy = computed(() => navigationPhase.value !== 'idle' || (loadingPostWindow.value && loadingPostDirection.value === 'anchor'))
const actionMessageSuccess = computed(() =>
  [
    t('topic.bookmarkAdded'),
    t('topic.bookmarkRemoved'),
    t('topic.watchAdded'),
    t('topic.watchRemoved'),
    t('topic.moderationBanSuccess'),
    t('topic.moderationUnbanSuccess'),
  ].includes(actionMessage.value),
)
const reportReasons = ['spam', 'abuse', 'illegal', 'irrelevant', 'other']
const floatingTopicActions = computed(() => {
  const actions = [
    {
      key: 'like',
      icon: Heart,
      active: isLiked.value,
      acting: actingLike.value,
      fill: true,
      title: t('topic.like'),
      activeClass: 'bg-error/10 text-error hover:bg-error/10',
      onClick: toggleLike,
    },
    {
      key: 'bookmark',
      icon: Bookmark,
      active: isBookmarked.value,
      acting: actingBookmark.value,
      fill: true,
      title: isBookmarked.value ? t('topic.bookmarked') : t('topic.bookmark'),
      activeClass: 'bg-info/10 text-primary hover:bg-info/10',
      onClick: toggleBookmark,
    },
    {
      key: 'watch',
      icon: Bell,
      active: isWatched.value,
      acting: actingWatch.value,
      fill: true,
      title: isWatched.value ? t('topic.watched') : t('topic.watch'),
      activeClass: 'bg-success/10 text-success hover:bg-success/15',
      onClick: toggleWatch,
    },
  ]

  if (props.topicActions?.canModerateTopic) {
    const isBanned = topicProcessStatus.value === 1
    actions.push({
      key: isBanned ? 'unban' : 'ban',
      icon: isBanned ? RotateCcw : Ban,
      active: false,
      acting: actingModeration.value,
      fill: false,
      title: isBanned ? t('topic.moderationUnban') : t('topic.moderationBan'),
      activeClass: 'text-base-content/75 hover:bg-base-200 hover:text-base-content',
      onClick: async () => requestTopicModeration(isBanned ? 'unban' : 'ban'),
    })
  }

  if (props.topicActions?.isOwnTopic && !isTopicRemoved()) {
    actions.push({
      key: 'delete-topic',
      icon: Trash2,
      active: false,
      acting: deletingTopic.value,
      fill: false,
      title: t('topic.deleteTopic'),
      activeClass: 'text-error hover:bg-error/10 hover:text-error',
      onClick: async () => requestDeleteTopic(),
    })
  }

  return actions
})
let postLoadObserver: IntersectionObserver | undefined
let postBottomLoadFrame = 0
let activePostScrollFrame = 0
const highlightedPostId = ref<number | null>(null)
let highlightTimer: number | undefined
const navigationPhase = ref<'idle' | 'loading' | 'scrolling'>('idle')
const navigationTargetPostNo = ref(0)
const navigationTargetPostId = ref(0)
let postRailResumeFrame = 0
let postRailResumeLastScrollY = 0
let postRailResumeStableFrames = 0
let postElements: HTMLElement[] = []
const postNavigationTargetTop = 160
// 树状视图折叠态：仅会话内有效，切换话题时重置（不持久化，默认全展开）。
const collapsedIds = ref(new Set<number>())

watch(
  () => props.interactions,
  (next) => {
    if (!next) return
    likeCount.value = next.likeCount
    isLiked.value = next.isLiked
    isBookmarked.value = next.isBookmarked
    isWatched.value = next.isWatched
  },
  { deep: true },
)

watch(
  () => [likeCount.value, isLiked.value, isBookmarked.value, isWatched.value] as const,
  ([nextLikeCount, nextLiked, nextBookmarked, nextWatched]) => {
    emit('topic-state', nextLikeCount)
    emit('interaction-state', {
      likeCount: nextLikeCount,
      isLiked: nextLiked,
      isBookmarked: nextBookmarked,
      isWatched: nextWatched,
    })
  },
)

onMounted(() => {
  void nextTick(observePostLoader)
  void nextTick(collectPostElements)
  void nextTick(scheduleActivePostFromScroll)
  setupPostBottomLoadFallback()
  window.addEventListener('scroll', scheduleActivePostFromScroll, { passive: true })
  window.addEventListener('resize', scheduleActivePostFromScroll)
  if (props.autoLoadFirstWindow && !initialPosts.length) {
    void loadFirstWindow()
  }
})

watch(
  () => props.topicId,
  () => {
    likeCount.value = props.interactions?.likeCount ?? props.topicActions?.likeCount ?? 0
    isLiked.value = props.interactions?.isLiked ?? props.topicActions?.isLiked ?? false
    isBookmarked.value = props.interactions?.isBookmarked ?? props.topicActions?.isBookmarked ?? false
    isWatched.value = props.interactions?.isWatched ?? props.topicActions?.isWatched ?? false
    topicProcessStatus.value = props.topicActions?.processStatus ?? 0
    pendingModerationAction.value = null
    actingModeration.value = false
    resetPostsFromProps()
    collapsedIds.value = new Set()
    mobilePostRailOpen.value = false
    void nextTick(observePostLoader)
    void nextTick(collectPostElements)
    void nextTick(scheduleActivePostFromScroll)
    void nextTick(syncInitialPostTarget)
  },
  { immediate: true },
)

watch(
  () => posts.value.map((post) => `${post.id}:${post.postNo}`).join(','),
  () => {
    void nextTick(() => {
      collectPostElements()
      scheduleActivePostFromScroll()
    })
  },
)

onBeforeUnmount(() => {
  postLoadObserver?.disconnect()
  window.removeEventListener('scroll', scheduleActivePostFromScroll)
  window.removeEventListener('scroll', schedulePostBottomLoadCheck)
  window.removeEventListener('resize', scheduleActivePostFromScroll)
  window.removeEventListener('resize', schedulePostBottomLoadCheck)
  window.cancelAnimationFrame(postBottomLoadFrame)
  window.cancelAnimationFrame(activePostScrollFrame)
  window.cancelAnimationFrame(postRailResumeFrame)
  navigationTargetPostId.value = 0
  window.clearTimeout(highlightTimer)
})

function setupPostBottomLoadFallback() {
  window.addEventListener('scroll', schedulePostBottomLoadCheck, { passive: true })
  window.addEventListener('resize', schedulePostBottomLoadCheck)
}

function schedulePostBottomLoadCheck() {
  if (postBottomLoadFrame) return
  postBottomLoadFrame = window.requestAnimationFrame(() => {
    postBottomLoadFrame = 0
    void maybeLoadRepliesNearViewportEdge()
  })
}

function isNearDocumentBottom() {
  const documentElement = document.documentElement
  const fullHeight = Math.max(documentElement.scrollHeight, document.body?.scrollHeight || 0)
  return fullHeight - (window.scrollY + window.innerHeight) <= 480
}

async function maybeLoadRepliesNearViewportEdge() {
  if (loadingPostWindow.value || postWindowError.value) return

  if (!postHasAfter.value || !isNearDocumentBottom()) return

  postAutoLoadAfter.value = true
  await loadPostWindow('after')
  await nextTick()
  if (postHasAfter.value && isNearDocumentBottom()) {
    schedulePostBottomLoadCheck()
  }
}

async function loadMoreRepliesManually() {
  postAutoLoadAfter.value = true
  await loadPostWindow('after')
}

async function loadFirstWindow() {
  if (loadingPostWindow.value) return

  loadingPostWindow.value = true
  loadingPostDirection.value = 'after'
  postWindowError.value = ''
  try {
    const payload = await getPostWindow({
      topicId: props.topicId,
      limit: 20,
    })
    applyPostWindowPayload(payload, 'replace')
    await nextTick()
    collectPostElements()
    if (!payload.hasAfter) {
      disablePostAutoLoadAfter()
    }
    if (postAutoLoadAfter.value) {
      observePostLoader()
    }
    scheduleActivePostFromScroll()
  } catch (error) {
    postWindowError.value = error instanceof Error ? error.message : t('api.repliesLoadFailed')
  } finally {
    loadingPostWindow.value = false
    loadingPostDirection.value = null
  }
}

function observePostLoader() {
  observePostAfterLoader()
}

function observePostAfterLoader() {
  postLoadObserver?.disconnect()
  if (!postLoadMoreEl.value || !postHasAfter.value || !postAutoLoadAfter.value || !('IntersectionObserver' in window)) return

  postLoadObserver = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting && postHasAfter.value && postAutoLoadAfter.value && !loadingPostWindow.value && !postWindowError.value) {
        void loadPostWindow('after')
      }
    },
    { rootMargin: '360px 0px' },
  )
  postLoadObserver.observe(postLoadMoreEl.value)
}

function collectPostElements() {
  postElements = Array.from(document.querySelectorAll<HTMLElement>('[data-post-no]'))
}

function keepNavigationTargetPinned() {
  if (!navigationTargetPostId.value) return false

  const element = document.getElementById(`post-${navigationTargetPostId.value}`)
  if (!element) return false

  const delta = element.getBoundingClientRect().top - postNavigationTargetTop
  if (Math.abs(delta) < 1) return false

  window.scrollBy({ top: delta, behavior: 'auto' })
  return true
}

function resumePostRailSyncWhenSettled() {
  navigationPhase.value = 'scrolling'
  window.cancelAnimationFrame(postRailResumeFrame)
  postRailResumeFrame = 0
  postRailResumeLastScrollY = window.scrollY
  postRailResumeStableFrames = 0
  const startedAt = performance.now()
  const settle = () => {
    const pinned = keepNavigationTargetPinned()
    const currentY = window.scrollY
    if (!pinned && Math.abs(currentY - postRailResumeLastScrollY) < 1) {
      postRailResumeStableFrames += 1
    } else {
      postRailResumeStableFrames = 0
      postRailResumeLastScrollY = currentY
    }
    if (postRailResumeStableFrames >= 8 || performance.now() - startedAt > 2600) {
      navigationPhase.value = 'idle'
      navigationTargetPostNo.value = 0
      navigationTargetPostId.value = 0
      postRailResumeFrame = 0
      syncPostRailProgress()
      return
    }
    postRailResumeFrame = window.requestAnimationFrame(settle)
  }
  postRailResumeFrame = window.requestAnimationFrame(settle)
}

function scheduleActivePostFromScroll() {
  if (navigationPhase.value !== 'idle' || activePostScrollFrame) return
  activePostScrollFrame = window.requestAnimationFrame(() => {
    activePostScrollFrame = 0
    syncPostRailProgress()
  })
}

function syncPostRailProgress() {
  const progress = measurePostViewportProgress()
  if (progress.postNo >= 0) {
    activePostNo.value = progress.postNo
    postRailProgressCurrent.value = progress.current
    postRailProgressStart.value = progress.start
    postRailProgressEnd.value = progress.end
    syncPostURL(progress.postNo)
  }
}

function syncPostURL(postNo: number) {
  if (props.syncUrl === false) return
  if (navigationPhase.value !== 'idle' || postNo < 1) return
  const pageStartPostNo = pageStartForPost(postNo)
  if (!pageStartPostNo) return
  const path = pageStartPostNo > 1
    ? `/p/post/${props.topicId}/${pageStartPostNo}`
    : `/p/post/${props.topicId}`
  if (window.location.pathname === path && !window.location.hash) return
  const state = window.history.state
  window.history.replaceState(state ? { ...state, current: path } : state, '', path)
}

function pageStartForPost(postNo: number) {
  let result = 0
  for (const start of postPageStarts.value) {
    if (start > postNo) break
    result = start
  }
  return result
}

function measurePostViewportProgress() {
  const markerY = Math.min(window.innerHeight * 0.38, 340)
  const viewportTop = 88
  const viewportBottom = window.innerHeight - 96
  return measurePostViewportProgressFromRects({
    posts: postElements.map((element) => {
      const rect = element.getBoundingClientRect()
      return {
        postNo: Number(element.dataset.postNo || 0),
        top: rect.top,
        bottom: rect.bottom,
        height: rect.height,
      }
    }),
    markerY,
    viewportTop,
    viewportBottom,
    maxPostNo: postMaxRange.value,
    visibleSlotSize: visibleSlotSize(),
  })
}

function resetPostsFromProps() {
  posts.value = [...initialPosts]
  postPageStarts.value = initialPosts[0]?.postNo ? [initialPosts[0].postNo] : []
  replyTargets.value = [...(initialPostStream.replyTargets || [])]
  postHasBefore.value = initialPostStream.hasBefore
  postHasAfter.value = initialPostStream.hasAfter
  postBeforePostNo.value = initialPostStream.beforePostNo || firstPostNo(initialPosts)
  postAfterPostNo.value = initialPostStream.afterPostNo || lastPostNo(initialPosts)
  postMaxNo.value = initialPostStream.maxPostNo || initialMaxPostNo()
  postAutoLoadAfter.value = true
  navigationPhase.value = 'idle'
  navigationTargetPostNo.value = 0
  navigationTargetPostId.value = 0
  activePostNo.value = firstPostNo(initialPosts) || 1
  syncProgressForPostNo(activePostNo.value)
  postWindowError.value = ''
  editingPostId.value = 0
}

function firstPostNo(items: PostPayload[]) {
  return items.length ? items[0].postNo || 0 : 0
}

function lastPostNo(items: PostPayload[]) {
  return items.length ? items[items.length - 1].postNo || 0 : 0
}

function initialMaxPostNo() {
  return Math.max(props.topicActions?.maxPostNo || 0, props.initialPostStream.maxPostNo || 0, lastPostNo(initialPosts))
}

function clampPostNo(postNo: number) {
  const maxPostNo = Math.max(1, postMaxRange.value || 1)
  return Math.min(maxPostNo, Math.max(1, Math.round(postNo)))
}

function progressForPostNo(postNo: number) {
  return progressForPostNoFraction(postNo, 0.5)
}

function progressForPostNoFraction(postNo: number, fraction: number) {
  const maxPostNo = Math.max(1, postMaxRange.value || 1)
  if (maxPostNo <= 1) return Math.min(1, Math.max(0, fraction))
  return Math.min(1, Math.max(0, (Math.max(1, postNo) - 1 + Math.min(1, Math.max(0, fraction))) / maxPostNo))
}

function visibleSlotSize() {
  return 1 / Math.max(1, postMaxRange.value || 1)
}

function syncProgressForPostNo(postNo: number) {
  const progress = progressForPostNo(postNo)
  postRailProgressCurrent.value = progress
  postRailProgressStart.value = Math.max(0, progress - visibleSlotSize() / 2)
  postRailProgressEnd.value = Math.min(1, progress + visibleSlotSize() / 2)
}

function findClosestLoadedPost(postNo: number) {
  let closest: PostPayload | undefined
  let closestDistance = Number.POSITIVE_INFINITY
  for (const post of posts.value) {
    if (!post.postNo) continue
    const distance = Math.abs(post.postNo - postNo)
    if (distance < closestDistance) {
      closest = post
      closestDistance = distance
    }
  }
  return closest
}

function formatRailDate(value: string) {
  const normalized = value.replace(' ', 'T')
  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) return value.slice(0, 10)
  const now = new Date()
  const options: Intl.DateTimeFormatOptions = date.getFullYear() === now.getFullYear()
    ? { month: 'short', day: 'numeric' }
    : { year: 'numeric', month: 'short', day: 'numeric' }
  return new Intl.DateTimeFormat(undefined, options).format(date)
}

function findPostHashId() {
  const match = window.location.hash.match(/^#post-(\d+)$/)
  return match ? Number(match[1]) : 0
}

function findPostPathNo() {
  const match = window.location.pathname.match(/^\/p\/post\/\d+\/(\d+)\/?$/)
  return match ? Number(match[1]) : 0
}

async function syncInitialPostTarget() {
  if (findPostHashId()) {
    await syncPostHash()
    return
  }
  const postNo = findPostPathNo()
  if (postNo > 1) await jumpToPostNo(postNo)
}

async function syncPostHash() {
  const postId = findPostHashId()
  if (!postId) return

  if (!posts.value.some((post) => post.id === postId)) {
    navigationPhase.value = 'loading'
    loadingPostWindow.value = true
    loadingPostDirection.value = 'anchor'
    postWindowError.value = ''
    try {
      const payload = await getPostWindow({
        topicId: props.topicId,
        anchorPostId: postId,
        limit: 20,
      })
      applyPostWindowPayload(payload, 'replace')
      await nextTick()
      collectPostElements()
    } catch (error) {
      postWindowError.value = error instanceof Error ? error.message : t('api.repliesLoadFailed')
    } finally {
      loadingPostWindow.value = false
      loadingPostDirection.value = null
      navigationPhase.value = 'idle'
    }
  }

  highlightPost(postId)
  await nextTick()
  const element = document.getElementById(`post-${postId}`)
  if (element) {
    navigationTargetPostId.value = postId
    scrollPostIntoComfortView(element, 'auto')
    resumePostRailSyncWhenSettled()
  }
}

function highlightPost(postId: number) {
  highlightedPostId.value = postId
  window.clearTimeout(highlightTimer)
  highlightTimer = window.setTimeout(() => {
    highlightedPostId.value = null
  }, 2400)
}

function mergePosts(nextReplies: PostPayload[], mode: 'replace' | 'prepend' | 'append') {
  if (mode === 'replace') {
    posts.value = nextReplies
    return
  }

  const seen = new Set(posts.value.map((post) => post.id))
  const filtered = nextReplies.filter((post) => !seen.has(post.id))
  posts.value = mode === 'prepend' ? [...filtered, ...posts.value] : [...posts.value, ...filtered]
}

function mergeReplyTargets(nextTargets: ReplyTargetPayload[], mode: 'replace' | 'prepend' | 'append') {
  if (mode === 'replace') {
    replyTargets.value = nextTargets
    return
  }
  const merged = new Map(replyTargets.value.map((target) => [target.id, target]))
  for (const target of nextTargets) merged.set(target.id, target)
  replyTargets.value = [...merged.values()]
}

function replyTargetFor(post: PostPayload) {
  return post.replyToPostId ? replyTargetMap.value.get(post.replyToPostId) : undefined
}

// 楼层流：按 postNo 升序（#519 起扁平与树状共用同一基础序列——扁平全量平铺，
// 树状在其上构建回复森林）。跨窗口加载的楼层天然按楼号归位。
const sortedPosts = computed<PostPayload[]>(() => {
  const list = [...posts.value]
  list.sort((a, b) => (a.postNo || 0) - (b.postNo || 0))
  if (!props.hideFirstPost) return list
  // Wiki 页（hideFirstPost）：首楼正文已在页面上方渲染，楼层流不再重复首楼本身。
  // initialPostStreamHidden 时连首楼的整棵回复链也一并隐藏（Wiki 页只留评论流），
  // 保留话题级回复（reply_to=0）。
  if (props.initialPostStreamHidden) {
    const hiddenIds = new Set<number>()
    const first = list.find((post) => post.postNo === 1)
    if (first?.id) hiddenIds.add(first.id)
    for (let grew = true; grew; ) {
      grew = false
      for (const post of list) {
        if (!hiddenIds.has(post.id) && post.replyToPostId && hiddenIds.has(post.replyToPostId)) {
          hiddenIds.add(post.id)
          grew = true
        }
      }
    }
    return list.filter((post) => !hiddenIds.has(post.id))
  }
  return list.filter((post) => post.postNo !== 1)
})

// For Q&A topics, separate answers from comments
const isQuestionTopic = computed(() => props.contentType === 1)
// Blog-like content types (Articles only) - no replies, different layout
const isBlogLikeTopic = computed(() => props.contentType === 3)
// 一楼回复/回答主操作醒目化判定：文章(3)、瞬间(2)、问题(1)统一使用 Solid Primary 实体胶囊高亮显示
const isProminentReply = computed(() => isQuestionTopic.value || isBlogLikeTopic.value || props.contentType === 2)
// 短文类型判定：1=提问, 2=瞬间（仅短文采用前置图窗；长文 0/3 保持经典图文穿插）
const isShortFormTopic = computed(() => props.contentType === 1 || props.contentType === 2)
// 首楼是否具备短文置顶图窗（用于移动端图窗置顶与信息层级调优）
const hasShortFormImages = computed(() => isShortFormTopic.value && Boolean(props.topicImages && props.topicImages.length > 0))

// 正文渲染净化：仅当短文类型首楼图片在置顶轮播视窗呈现时，才剥离正文中重复的 <img> 标记；长文 100% 保持图文穿插
function renderedPostContent(post: PostPayload) {
  let html = post.renderedContent
  if (isShortFormTopic.value && isFirstPost(post) && props.topicImages && props.topicImages.length > 0) {
    for (const imgUrl of props.topicImages) {
      const escaped = imgUrl.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      const pImgRegex = new RegExp(`<p>\\s*<img[^>]*src=["']${escaped}["'][^>]*>\\s*<\\/p>`, 'gi')
      html = html.replace(pImgRegex, '')
      const inlineImgRegex = new RegExp(`<img[^>]*src=["']${escaped}["'][^>]*>`, 'gi')
      html = html.replace(inlineImgRegex, '')
    }
  }
  return html
}

// Q&A 链上溯工具（qaPostsById / qaChainRootId）：原「两层布局」的堆叠分组已随 #519 移除，
// 现仅用于 append 增量加载时判定哪些链内子回复需要 forceFlat 兜底为森林根节点。
const qaPostsById = computed(() => {
  const map = new Map<number, PostPayload>()
  for (const post of posts.value) map.set(post.id, post)
  return map
})

// append 增量加载时（树状视图），链根位于上一窗口边界之前的子回复兜底为根节点平铺，
// 避免把新内容嵌进读者早已滚过的旧楼层卡片里、出现在视口上方导致不可见；扁平视图全量平铺，不受影响。
const qaForceFlatPostIds = ref(new Set<number>())

// 返回 post 所属回复链的根楼层 id（供 append 边界判定 forceFlat 用）；
// 回复 #1 或无目标 → null；链根未加载或链异常深（>64 跳，含脏数据成环）→ null。
function qaChainRootId(post: PostPayload): number | null {
  if (qaForceFlatPostIds.value.has(post.id)) return null
  if (!post.replyToPostId) return null
  const firstId = firstPost.value?.id
  if (firstId && post.replyToPostId === firstId) return null
  const byId = qaPostsById.value
  let current: PostPayload | undefined = byId.get(post.replyToPostId)
  let hops = 0
  while (current?.replyToPostId && current.replyToPostId !== firstId) {
    if (hops++ >= 64) return null
    current = byId.get(current.replyToPostId)
    if (!current) return null
  }
  return current?.id ?? null
}
// 视图模式：扁平/树状胶囊切换。默认按内容类型（提问=树状，其余=扁平），
// 用户手动选择按内容类型独立记忆（localStorage，与 home-feed-mode 同一客户端模式）。
const { viewMode: postViewMode, setViewMode: setPostViewMode } = usePostViewMode(() => props.contentType)

// 树状视图：真实父子嵌套的回复森林。兜底为根节点平铺（无目标/回复首楼/目标未加载/
// 祖先链成环或超限/forceFlat），保证内容零丢失；Wiki 页沿用 sortedPosts 过滤结果。
const replyForest = computed(() => buildReplyForest(sortedPosts.value, {
  firstPostId: firstPost.value?.id,
  forceFlatIds: qaForceFlatPostIds.value,
}))

// 树状主流层 = 森林根节点（沿用主流楼层卡渲染）；每个根卡的树状块渲染其全部后代行。
const treeRootPosts = computed(() => replyForest.value.map((node) => node.post))

const treeRowsByRoot = computed<Map<number, ForestRow[]>>(() => {
  const map = new Map<number, ForestRow[]>()
  for (const root of replyForest.value) {
    const rows = flattenReplyForest(root.children, collapsedIds.value)
    if (rows.length) map.set(root.post.id, rows)
  }
  return map
})

function treeRowsFor(post: PostPayload): ForestRow[] {
  return treeRowsByRoot.value.get(post.id) ?? []
}

function treeIndentLevel(depth: number) {
  return postTreeIndentLevel(depth)
}

function toggleTreeCollapse(postId: number) {
  const next = new Set(collapsedIds.value)
  if (next.has(postId)) next.delete(postId)
  else next.add(postId)
  collapsedIds.value = next
}


const renderPosts = computed<PostPayload[]>(() => {
  if (postViewMode.value === 'tree') {
    return treeRootPosts.value
  }
  // #519：扁平 = 全量按楼号平铺，与普通话题一致；QA 链内子回复不再堆叠进链根卡片，
  // 回复上下文统一由引用条（PostReplyReference）承接。
  return sortedPosts.value
})

// #520：树状视图孤根弱提示。森林根楼层回复的是首楼以外、且目标不在当前窗口的楼层时
// （深链跨窗 / append 边界 forceFlat），父子嵌套无法表达「回复谁」，用短提示承接上下文，
// 替代全文引用条；其余场景返回 null 不渲染。目标不可见时（隐藏/审核移除/隐私清除，
// 后端 buildReplyTargetPayload 早退只下发 { id, unavailable }，无作者与楼号）降级为
// unavailable 态，保证「回复了某楼」这一事实仍可见且不泄漏被隐藏目标的作者与楼号。
function treeRootReplyHint(post: PostPayload): { username?: string; postNo?: number; unavailable?: boolean; isAnonymous?: boolean } | null {
  if (postViewMode.value !== 'tree') return null
  if (!post.replyToPostId) return null
  const firstId = firstPost.value?.id
  if (firstId && post.replyToPostId === firstId) return null
  const target = replyTargetMap.value.get(post.replyToPostId)
  // 首楼不在当前窗口时用 replyTargets.postNo 判定目标是否首楼（深链打开后段楼层）。
  if (!firstId && (!target || target.postNo === 1)) return null
  if (!target || target.unavailable || !target.author.username) return { unavailable: true }
  return { username: target.author.username, postNo: target.postNo, isAnonymous: Boolean(target.isAnonymous) }
}

// 引用条规则：回复话题首楼（或无目标）视为话题级回复，不重复引用首楼正文；
// 回复其他楼层才显示可折叠引用消息。目标不在当前窗口时由 replyTargets 兜底渲染
// （含 unavailable 降级态）。深链打开后段楼层时首楼可能不在当前窗口，
// 此时用 replyTargets.postNo 判定目标是否首楼，不能依赖当前窗口的 firstPost。
function showReplyReference(post: PostPayload) {
  if (!post.replyToPostId) return false
  if (firstPost.value?.id) return post.replyToPostId !== firstPost.value.id
  const target = replyTargetMap.value.get(post.replyToPostId)
  if (!target) return true
  return target.postNo !== 1
}

function applyPostWindowPayload(payload: Awaited<ReturnType<typeof getPostWindow>>, mergeMode: 'replace' | 'prepend' | 'append') {
  const pageStartPostNo = firstPostNo(payload.posts)
  // append：合并前记录窗口边界，用于把链根在边界之前的子回复改为平铺（见 qaForceFlatPostIds）。
  const appendBoundaryPostNo = mergeMode === 'append' ? postAfterPostNo.value : 0
  mergePosts(payload.posts, mergeMode)
  if (mergeMode === 'replace') {
    postPageStarts.value = pageStartPostNo ? [pageStartPostNo] : []
    qaForceFlatPostIds.value = new Set()
  } else if (pageStartPostNo && !postPageStarts.value.includes(pageStartPostNo)) {
    postPageStarts.value = [...postPageStarts.value, pageStartPostNo].sort((a, b) => a - b)
  }
  mergeReplyTargets(payload.replyTargets || [], mergeMode)
  if (mergeMode === 'append' && isQuestionTopic.value && appendBoundaryPostNo > 0) {
    const byId = qaPostsById.value
    const nextForceFlat = new Set(qaForceFlatPostIds.value)
    for (const post of payload.posts) {
      const rootId = qaChainRootId(post)
      if (rootId == null) continue
      const root = byId.get(rootId)
      if (root && (root.postNo || 0) < appendBoundaryPostNo) nextForceFlat.add(post.id)
    }
    qaForceFlatPostIds.value = nextForceFlat
  }
  const nextMaxPostNo = Math.max(postMaxNo.value, payload.maxPostNo || 0)
  postMaxNo.value = nextMaxPostNo
  syncLoadedPostWindowBounds(payload.hasBefore, payload.hasAfter, nextMaxPostNo)
}

function syncLoadedPostWindowBounds(hasBefore = postHasBefore.value, hasAfter = postHasAfter.value, maxPostNo = postMaxNo.value) {
  const loadedFirstPostNo = firstPostNo(posts.value)
  const loadedLastPostNo = lastPostNo(posts.value)
  postHasBefore.value = hasBefore && loadedFirstPostNo > 1
  postHasAfter.value = hasAfter && loadedLastPostNo < maxPostNo
  postBeforePostNo.value = loadedFirstPostNo
  postAfterPostNo.value = loadedLastPostNo
}

function disablePostAutoLoadAfter() {
  postAutoLoadAfter.value = false
  postLoadObserver?.disconnect()
}

function firstVisiblePostElement() {
  for (const element of postElements) {
    const rect = element.getBoundingClientRect()
    if (rect.bottom > 96) return element
  }
  return postElements[0] || null
}

async function keepScrollPositionWhilePrepending<T>(operation: () => Promise<T>) {
  const anchor = firstVisiblePostElement()
  const beforeTop = anchor?.getBoundingClientRect().top ?? 0
  const result = await operation()
  await nextTick()
  collectPostElements()
  if (anchor) {
    const afterTop = anchor.getBoundingClientRect().top
    window.scrollBy({ top: afterTop - beforeTop, behavior: 'auto' })
  }
  return result
}

async function loadPostWindow(direction: 'before' | 'after') {
  if (loadingPostWindow.value) return
  if (direction === 'after' && (!postHasAfter.value || !postAutoLoadAfter.value)) return
  if (direction === 'before' && !postHasBefore.value) return

  loadingPostWindow.value = true
  loadingPostDirection.value = direction
  postWindowError.value = ''
  try {
    if (direction === 'before') {
      await keepScrollPositionWhilePrepending(async () => {
        const payload = await getPostWindow({
          topicId: props.topicId,
          beforePostNo: postBeforePostNo.value,
          limit: 20,
        })
        applyPostWindowPayload(payload, 'prepend')
        return payload
      })
    } else {
      const payload = await getPostWindow({
        topicId: props.topicId,
        afterPostNo: postAfterPostNo.value,
        limit: 20,
      })
      applyPostWindowPayload(payload, 'append')
      await nextTick()
      collectPostElements()
      if (!payload.hasAfter) {
        disablePostAutoLoadAfter()
      }
    }
    if (postAutoLoadAfter.value) {
      observePostLoader()
    }
    scheduleActivePostFromScroll()
  } catch (error) {
    postWindowError.value = error instanceof Error ? error.message : t('api.repliesLoadFailed')
  } finally {
    loadingPostWindow.value = false
    loadingPostDirection.value = null
  }
}

async function jumpToPostNo(postNo: number) {
  const target = clampPostNo(postNo)
  if (loadingPostWindow.value) {
    activePostNo.value = target
    syncProgressForPostNo(target)
    return
  }

  disablePostAutoLoadAfter()
  navigationPhase.value = 'loading'
  navigationTargetPostNo.value = target
  navigationTargetPostId.value = 0
  activePostNo.value = target
  syncProgressForPostNo(target)
  const loaded = posts.value.find((post) => post.postNo === target)
  if (loaded) {
    activePostNo.value = loaded.postNo
    syncProgressForPostNo(loaded.postNo)
    const element = await findPostElementAfterLayout(loaded.id)
    if (element) {
      navigationTargetPostId.value = loaded.id
      scrollPostIntoComfortView(element)
    }
    resumePostRailSyncWhenSettled()
    return
  }

  loadingPostWindow.value = true
  loadingPostDirection.value = 'anchor'
  postWindowError.value = ''
  try {
    const payload = await getPostWindow({
      topicId: props.topicId,
      anchorPostNo: target,
      limit: 20,
    })
    applyPostWindowPayload(payload, 'replace')
    await nextTick()
    collectPostElements()
    const closest = findClosestLoadedPost(target)
    if (closest) {
      activePostNo.value = closest.postNo
      syncProgressForPostNo(closest.postNo)
      const element = await findPostElementAfterLayout(closest.id)
      if (element) {
        navigationTargetPostId.value = closest.id
        scrollPostIntoComfortView(element, 'auto')
      }
      resumePostRailSyncWhenSettled()
    } else {
      navigationPhase.value = 'idle'
      navigationTargetPostNo.value = 0
      navigationTargetPostId.value = 0
    }
  } catch (error) {
    postWindowError.value = error instanceof Error ? error.message : t('api.repliesLoadFailed')
    navigationPhase.value = 'idle'
    navigationTargetPostNo.value = 0
    navigationTargetPostId.value = 0
  } finally {
    loadingPostWindow.value = false
    loadingPostDirection.value = null
  }
}

async function jumpToLatestPost() {
  await jumpToPostNo(postMaxRange.value)
}

function jumpToTopicBody() {
  void jumpToPostNo(1)
}

function focusPostComposer() {
  mobilePostRailOpen.value = false
  // #515：文档出现横向溢出时，移动端 fixed 浮层会随横向平移偏出视口；
  // 呼出回复编辑器前把横向滚动归位，保证面板始终水平居中可见。
  if (typeof document !== 'undefined' && document.scrollingElement) {
    document.scrollingElement.scrollLeft = 0
  }
  composerOpen.value = true
}

function updateComposerOpen(open: boolean) {
  composerOpen.value = open
  if (!open) sensitiveWords.value = []
  if (!open && editingPostId.value) {
    cancelEditPost()
  }
}

function openFloatingPostComposer() {
  if (editingPostId.value) {
    cancelEditPost()
  }
  targetPostId.value = 0
  focusPostComposer()
}

function closeMobilePostRail() {
  mobilePostRailOpen.value = false
}

async function selectPostFromRail(postNo: number) {
  closeMobilePostRail()
  if (postNo <= 1) {
    jumpToTopicBody()
    return
  }
  await jumpToPostNo(postNo)
}

async function jumpToLatestPostFromRail() {
  closeMobilePostRail()
  await jumpToLatestPost()
}

function jumpToTopicBodyFromRail() {
  closeMobilePostRail()
  jumpToTopicBody()
}

function isElementMostlyVisible(element: HTMLElement) {
  const rect = element.getBoundingClientRect()
  return rect.top >= 96 && rect.bottom <= window.innerHeight - 120
}

function scrollPostIntoComfortView(element: HTMLElement, behavior: ScrollBehavior = 'smooth') {
  const targetTop = element.getBoundingClientRect().top + window.scrollY - postNavigationTargetTop
  window.scrollTo({
    top: Math.max(0, targetTop),
    behavior,
  })
}

function waitForAnimationFrame() {
  return new Promise<void>((resolve) => {
    window.requestAnimationFrame(() => resolve())
  })
}

async function findPostElementAfterLayout(postId: number) {
  let lastTop: number | null = null
  let stableFrames = 0
  for (let attempts = 0; attempts < 10; attempts += 1) {
    await nextTick()
    await waitForAnimationFrame()
    const element = document.getElementById(`post-${postId}`)
    if (!element) continue

    const top = element.getBoundingClientRect().top
    if (lastTop !== null && Math.abs(top - lastTop) < 1) {
      stableFrames += 1
      if (stableFrames >= 2) return element
    } else {
      stableFrames = 0
      lastTop = top
    }
  }
  return document.getElementById(`post-${postId}`)
}

async function revealCreatedPost(postId: number) {
  if (!postId) return

  navigationPhase.value = 'loading'
  const payload = await getPostWindow({
    topicId: props.topicId,
    anchorPostId: postId,
    limit: 20,
  })
  applyPostWindowPayload(payload, 'replace')
  const createdPost = payload.posts.find((post) => post.id === postId)
  if (createdPost?.postNo) {
    navigationTargetPostNo.value = createdPost.postNo
    activePostNo.value = createdPost.postNo
    syncProgressForPostNo(createdPost.postNo)
  }
  highlightPost(postId)
  const element = await findPostElementAfterLayout(postId)
  if (element && !isElementMostlyVisible(element)) {
    navigationTargetPostId.value = postId
    scrollPostIntoComfortView(element)
    resumePostRailSyncWhenSettled()
    return
  }
  navigationPhase.value = 'idle'
  navigationTargetPostNo.value = 0
  navigationTargetPostId.value = 0
  collectPostElements()
  scheduleActivePostFromScroll()
}

async function toggleLike() {
  if (actingLike.value) return

  const nextLiked = !isLiked.value
  const previousLiked = isLiked.value
  const previousCount = likeCount.value
  actingLike.value = true
  actionMessage.value = ''
  isLiked.value = nextLiked
  likeCount.value = Math.max(0, likeCount.value + (nextLiked ? 1 : -1))
  try {
    await likeTopic(props.topicId, nextLiked ? 1 : 2)
  } catch (error) {
    isLiked.value = previousLiked
    likeCount.value = previousCount
    actionMessage.value = error instanceof Error ? error.message : t('api.likeFailed')
  } finally {
    actingLike.value = false
  }
}

async function toggleBookmark() {
  if (actingBookmark.value) return

  const nextBookmarked = !isBookmarked.value
  const previousBookmarked = isBookmarked.value
  actingBookmark.value = true
  actionMessage.value = ''
  isBookmarked.value = nextBookmarked
  try {
    await bookmarkTopic(props.topicId, nextBookmarked ? 1 : 2)
    actionMessage.value = nextBookmarked ? t('topic.bookmarkAdded') : t('topic.bookmarkRemoved')
  } catch (error) {
    isBookmarked.value = previousBookmarked
    actionMessage.value = error instanceof Error ? error.message : t('api.bookmarkFailed')
  } finally {
    actingBookmark.value = false
  }
}

async function toggleWatch() {
  if (actingWatch.value) return

  const nextWatched = !isWatched.value
  const previousWatched = isWatched.value
  actingWatch.value = true
  actionMessage.value = ''
  isWatched.value = nextWatched
  try {
    await watchTopic(props.topicId, nextWatched ? 1 : 2)
    actionMessage.value = nextWatched ? t('topic.watchAdded') : t('topic.watchRemoved')
  } catch (error) {
    isWatched.value = previousWatched
    actionMessage.value = error instanceof Error ? error.message : t('api.watchFailed')
  } finally {
    actingWatch.value = false
  }
}

interface PostActionState {
  likeCount: number
  isLiked: boolean
  isBookmarked: boolean
  actingLike: boolean
  actingBookmark: boolean
}

const postActions = ref<Record<number, PostActionState>>({})

function postActionState(post: PostPayload): PostActionState {
  let state = postActions.value[post.id]
  if (!state) {
    state = {
      likeCount: post.likeCount || 0,
      isLiked: post.isLiked || false,
      isBookmarked: post.isBookmarked || false,
      actingLike: false,
      actingBookmark: false,
    }
    postActions.value = { ...postActions.value, [post.id]: state }
  }
  return state
}

async function togglePostLike(post: PostPayload) {
  const state = postActionState(post)
  if (state.actingLike) return

  const nextLiked = !state.isLiked
  const previousLiked = state.isLiked
  const previousCount = state.likeCount
  state.actingLike = true
  state.isLiked = nextLiked
  state.likeCount = Math.max(0, state.likeCount + (nextLiked ? 1 : -1))
  try {
    await likePost(post.id, nextLiked ? 1 : 2)
  } catch (error) {
    state.isLiked = previousLiked
    state.likeCount = previousCount
    pushFlash(error instanceof Error ? error.message : t('api.likeFailed'))
  } finally {
    state.actingLike = false
  }
}

async function togglePostBookmark(post: PostPayload) {
  const state = postActionState(post)
  if (state.actingBookmark) return

  const nextBookmarked = !state.isBookmarked
  const previousBookmarked = state.isBookmarked
  state.actingBookmark = true
  state.isBookmarked = nextBookmarked
  try {
    await bookmarkPost(post.id, nextBookmarked ? 1 : 2)
    pushFlash(nextBookmarked ? t('topic.bookmarkAdded') : t('topic.bookmarkRemoved'))
  } catch (error) {
    state.isBookmarked = previousBookmarked
    pushFlash(error instanceof Error ? error.message : t('api.bookmarkFailed'))
  } finally {
    state.actingBookmark = false
  }
}

async function sharePost(post: PostPayload) {
  const url = `${window.location.origin}/p/post/${props.topicId}/${post.postNo}#post-${post.id}`
  if (navigator.share) {
    try {
      await navigator.share({ title: props.topicTitle, url })
      return
    } catch {
      return // 用户取消分享
    }
  }
  try {
    await navigator.clipboard.writeText(url)
    pushFlash(t('topic.linkCopied'))
  } catch {
    pushFlash(t('topic.shareFailed'))
  }
}

function replyTo(post: PostPayload) {
  if (editingPostId.value) {
    cancelEditPost()
  }
  targetPostId.value = post.id
  errorMessage.value = ''
  successMessage.value = ''
  sensitiveWords.value = []
  focusPostComposer()
}

function cancelPostTarget() {
  targetPostId.value = 0
  errorMessage.value = ''
  sensitiveWords.value = []
}

function clearPostValidation() {
  errorMessage.value = ''
  successMessage.value = ''
  sensitiveWords.value = []
}

function handlePostImageInserted(count: number) {
  errorMessage.value = ''
  successMessage.value = count > 1 ? t('publish.imagesInserted', { count }) : t('publish.imageInserted')
}

function handlePostImageError(message: string) {
  errorMessage.value = message
}

function isFirstPost(post: PostPayload) {
  return post.postNo === 1
}

function isPostRemoved(post: PostPayload) {
  return post.isAuthorDeleted || post.isModeratorRemoved
}

function isTopicRemoved() {
  return Boolean(props.topicActions?.authorDeleted || props.topicActions?.moderatorRemoved)
}

// 优先展示用户昵称，未设置昵称时回退到账号名；匿名楼层展示匿名占位
function authorDisplayName(author: { username: string; nickname?: string }) {
  return author.nickname || author.username
}

function isAnonymousPost(post: PostPayload) {
  return Boolean(post.isAnonymous)
}

// 匿名楼层占位头像：复用课程评价的 boring-avatars beam 占位（seed 用楼层 id，跨语言稳定）
function anonymousAvatarSrc(post: PostPayload, size = 36): string {
  return buildBeamAvatarDataUri(`anonymous-${post.id}`, size)
}

function canEditPost(post: PostPayload) {
  return post.isOwnPost && !post.isHidden && !isPostRemoved(post)
}

function canDeleteRenderedPost(post: PostPayload) {
  return post.isOwnPost && !post.isHidden && !isPostRemoved(post) && !isFirstPost(post)
}

function startEditPost(post: PostPayload) {
  if (savingEditPostId.value || deletingPostId.value === post.id) return
  // 首楼本质上是话题本体，其编辑应进入“发布话题”编辑态（可改标题/分类/正文）。
  // 提问（contentType: 1）与瞬间（contentType: 2）在进入帖子主页后，
  // 编辑功能保持和发布时一致的弹层编辑器 QuickPublishModal；
  // 文章（contentType: 3）与常规长文跳转到文章发布/编辑页面。
  if (isFirstPost(post)) {
    if (props.contentType === 1 || props.contentType === 2) {
      const { openQuickPublishEdit } = useQuickPublish()
      openQuickPublishEdit({
        topicId: props.topicId,
        contentType: props.contentType,
        title: props.topicTitle,
        content: post.content,
        categoryIds: props.categories?.map((c) => c.id) || [],
        images: props.topicImages,
      })
      return
    }
    window.location.href = `/publish?id=${props.topicId}`
    return
  }
  if (!editingPostId.value) {
    postDraftBeforeEdit.value = postContent.value
    targetPostBeforeEdit.value = targetPostId.value
  }
  targetPostId.value = 0
  editingPostId.value = post.id
  postContent.value = post.content
  errorMessage.value = ''
  successMessage.value = ''
  sensitiveWords.value = []
  focusPostComposer()
}

function cancelEditPost() {
  if (savingEditPostId.value) return
  editingPostId.value = 0
  errorMessage.value = ''
  sensitiveWords.value = []
  postContent.value = postDraftBeforeEdit.value
  targetPostId.value = targetPostBeforeEdit.value
  postDraftBeforeEdit.value = ''
  targetPostBeforeEdit.value = 0
}

async function savePostEdit() {
  if (savingEditPostId.value) return

  const post = posts.value.find((item) => item.id === editingPostId.value)
  if (!post) {
    cancelEditPost()
    return
  }

  const content = postContent.value.trim()
  if (!content) {
    errorMessage.value = t('topic.replyRequired')
    return
  }
  if (content === post.content.trim()) {
    cancelEditPost()
    composerOpen.value = false
    return
  }

  savingEditPostId.value = post.id
  errorMessage.value = ''
  successMessage.value = ''
  sensitiveWords.value = []
  try {
    const updated = await updatePost(post.id, content)
    const index = posts.value.findIndex((item) => item.id === post.id)
    if (index >= 0) {
      posts.value[index] = {
        ...posts.value[index],
        content: updated.content,
        renderedContent: updated.renderedContent,
        updatedAt: updated.updatedAt,
        lastEditor: { ...posts.value[index].author, id: updated.lastEditorId },
        lastEditedAt: updated.lastEditedAt,
        revisionCount: updated.revisionCount,
      }
    }
    editingPostId.value = 0
    postContent.value = postDraftBeforeEdit.value
    targetPostId.value = targetPostBeforeEdit.value
    postDraftBeforeEdit.value = ''
    targetPostBeforeEdit.value = 0
    composerOpen.value = false
    pushFlash(t('topic.replyUpdated'), 'success')
  } catch (error) {
    sensitiveWords.value = sensitiveWordsFromError(error)
    errorMessage.value = error instanceof Error ? error.message : t('api.replyUpdateFailed')
  } finally {
    savingEditPostId.value = 0
  }
}

async function submitPost() {
  if (editingPostId.value) {
    await savePostEdit()
    return
  }

  const postId = targetPost.value?.id || 0
  const content = postContent.value.trim()
  if (submitting.value) return

  if (!content) {
    errorMessage.value = t('topic.replyRequired')
    successMessage.value = ''
    return
  }

  submitting.value = true
  errorMessage.value = ''
  successMessage.value = ''
  sensitiveWords.value = []
  try {
    const createdPost = await createPost(props.topicId, content, postId, {
      captchaId: captchaId.value,
      captchaCode: captchaCode.value,
      isAnonymous: props.allowAnonymous ? anonymous.value : undefined,
    })
    clearCaptcha()
    postContent.value = ''
    anonymous.value = false
    targetPostId.value = 0
    composerOpen.value = false
    pushFlash(t('topic.replyPosted'), 'success')
    const createdPostId = typeof createdPost === 'object' && createdPost !== null ? createdPost.id : createdPost
    try {
      if (typeof createdPostId === 'number') {
        await revealCreatedPost(createdPostId)
      } else {
        await refreshCurrentPage()
      }
    } catch (error) {
      postWindowError.value = error instanceof Error ? error.message : t('api.repliesLoadFailed')
    }
  } catch (error) {
    if (challengeFromError(error)) {
      sensitiveWords.value = []
      errorMessage.value = t('server.auth.captcha.invalid')
    } else {
      sensitiveWords.value = sensitiveWordsFromError(error)
      errorMessage.value = error instanceof Error ? error.message : t('api.replyFailed')
    }
  } finally {
    submitting.value = false
  }
}

async function refreshCurrentPage() {
  const payload = await fetchPage(new URL(window.location.href))
  window.dispatchEvent(new CustomEvent('goose:page', { detail: payload }))
}

function requestDeletePost(post: PostPayload) {
  if (savingEditPostId.value === post.id) return
  pendingDeletePost.value = post
  deleteErrorMessage.value = ''
  void reportContentEvent('content_delete_clicked', 'post', post.id)
}

function closeDeleteDialog() {
  if (deletingPostId.value) return
  pendingDeletePost.value = null
  deleteErrorMessage.value = ''
}

function requestDeleteTopic() {
  if (deletingTopic.value || isTopicRemoved()) return
  pendingDeleteTopic.value = true
  deleteErrorMessage.value = ''
  void reportContentEvent('content_delete_clicked', 'topic', props.topicId)
}

function closeDeleteTopicDialog() {
  if (deletingTopic.value) return
  pendingDeleteTopic.value = false
  deleteErrorMessage.value = ''
}

async function removeTopic() {
  if (deletingTopic.value || !pendingDeleteTopic.value) return

  deletingTopic.value = true
  deleteErrorMessage.value = ''
  try {
    void reportContentEvent('content_delete_confirmed', 'topic', props.topicId)
    await deleteTopic(props.topicId)
    pendingDeleteTopic.value = false
    pushFlash(t('topic.topicDeleted'), 'success')
    await refreshCurrentPage()
  } catch (error) {
    deleteErrorMessage.value = error instanceof Error ? error.message : t('api.topicDeleteFailed')
  } finally {
    deletingTopic.value = false
  }
}

/** 隐私紧急删除（PRD R8）：跳过 30 天恢复窗口，全渠道立即彻底删除。 */
async function privacyEraseTopic() {
  if (deletingTopic.value || !pendingDeleteTopic.value) return
  if (!window.confirm(t('topic.privacyEraseConfirm'))) return
  deletingTopic.value = true
  deleteErrorMessage.value = ''
  try {
    await privacyEraseContent('topic', props.topicId)
    pendingDeleteTopic.value = false
    pushFlash(t('topic.privacyEraseSuccess'), 'success')
    await refreshCurrentPage()
  } catch (error) {
    deleteErrorMessage.value = error instanceof Error ? error.message : t('api.topicDeleteFailed')
  } finally {
    deletingTopic.value = false
  }
}

async function privacyErasePost() {
  if (!pendingDeletePost.value || deletingPostId.value) return
  if (!window.confirm(t('topic.privacyEraseConfirm'))) return
  deletingPostId.value = pendingDeletePost.value.id
  deleteErrorMessage.value = ''
  try {
    await privacyEraseContent('post', pendingDeletePost.value.id)
    const deletedId = pendingDeletePost.value.id
    pendingDeletePost.value = null
    pushFlash(t('topic.privacyEraseSuccess'), 'success')
    posts.value = posts.value.filter((post) => post.id !== deletedId)
  } catch (error) {
    deleteErrorMessage.value = error instanceof Error ? error.message : t('api.replyDeleteFailed')
  } finally {
    deletingPostId.value = 0
  }
}

function requestTopicModeration(action: 'ban' | 'unban') {
  actionMessage.value = ''
  pendingModerationAction.value = action
}

function closeTopicModerationDialog() {
  if (actingModeration.value) return
  pendingModerationAction.value = null
}

async function updateTopicModerationFromDetail() {
  if (actingModeration.value || !pendingModerationAction.value) return
  actingModeration.value = true
  actionMessage.value = ''
  const action = pendingModerationAction.value
  try {
    await updateModerationTopicStatus(props.topicId, action)
    topicProcessStatus.value = action === 'ban' ? 1 : 0
    pendingModerationAction.value = null
    actionMessage.value = action === 'ban' ? t('topic.moderationBanSuccess') : t('topic.moderationUnbanSuccess')
    pushFlash(actionMessage.value, 'success')
  } catch (error) {
    actionMessage.value = error instanceof Error ? error.message : t('api.moderationActionFailed')
    pushFlash(actionMessage.value, 'error')
  } finally {
    actingModeration.value = false
  }
}

function openLogin() {
  window.location.href = `/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search + window.location.hash)}`
}

function requestReport(target: { targetType: 'topic' | 'post'; targetId: number; title: string; excerpt: string }) {
  if (!props.viewer.isAuthenticated) {
    openLogin()
    return
  }
  pendingReport.value = target
  reportReason.value = 'spam'
  reportNote.value = ''
  reportError.value = ''
}

function handleMarkdownImageClick(event: MouseEvent) {
  const target = event.target
  if (!(target instanceof HTMLElement)) return

  const image = target.closest('.gf-prose-post img')
  if (!(image instanceof HTMLImageElement)) return

  const imageSrc = image.currentSrc || image.src
  if (!imageSrc) return

  const anchor = image.closest('a')
  if (anchor && !sameUrl(anchor.href, imageSrc)) return

  event.preventDefault()
  event.stopPropagation()

  const markdownImages = Array.from(document.querySelectorAll<HTMLImageElement>('.gf-prose-post img'))
    .map((item) => ({
      src: item.currentSrc || item.src,
      alt: item.alt || '',
    }))
    .filter((item) => item.src)
  const index = markdownImages.findIndex((item) => sameUrl(item.src, imageSrc))

  markdownImageViewer.value?.open(markdownImages, index >= 0 ? index : 0)
}

function sameUrl(left: string, right: string) {
  try {
    return new URL(left, window.location.href).href === new URL(right, window.location.href).href
  } catch {
    return left === right
  }
}

function requestTopicReport() {
  requestReport({
    targetType: 'topic',
    targetId: props.topicId,
    title: props.topicTitle,
    excerpt: props.topicActions?.description || '',
  })
}

function requestPostReport(post: PostPayload) {
  requestReport({
    targetType: 'post',
    targetId: post.id,
    title: t('topic.replyReportTitle', { no: post.postNo || post.id }),
    excerpt: post.content,
  })
}

function closeReportDialog() {
  if (reportSubmitting.value) return
  pendingReport.value = null
  reportError.value = ''
}

async function submitCurrentReport() {
  if (!pendingReport.value || reportSubmitting.value) return
  reportSubmitting.value = true
  reportError.value = ''
  try {
    await submitReport(pendingReport.value.targetType, pendingReport.value.targetId, reportReason.value, reportNote.value)
    pendingReport.value = null
    pushFlash(t('topic.reportSubmitted'), 'success')
  } catch (error) {
    reportError.value = error instanceof Error ? error.message : t('api.reportFailed')
  } finally {
    reportSubmitting.value = false
  }
}

function postModerationBusy(postId: number) {
  return moderatingPostIds.value.includes(postId)
}

async function moderatePost(post: PostPayload, action: 'ban' | 'unban') {
  if (postModerationBusy(post.id)) return
  moderatingPostIds.value = [...moderatingPostIds.value, post.id]
  try {
    await updateModerationPostStatus(post.id, action)
    post.processStatus = action === 'ban' ? 1 : 0
    post.isHidden = action === 'ban'
    pushFlash(action === 'ban' ? t('topic.replyModerationBanSuccess') : t('topic.replyModerationUnbanSuccess'), 'success')
  } catch (error) {
    pushFlash(error instanceof Error ? error.message : t('api.moderationActionFailed'), 'error')
  } finally {
    moderatingPostIds.value = moderatingPostIds.value.filter(id => id !== post.id)
  }
}

async function removePost(postId: number) {
  if (deletingPostId.value || savingEditPostId.value === postId) return

  deletingPostId.value = postId
  errorMessage.value = ''
  successMessage.value = ''
  deleteErrorMessage.value = ''
  try {
    const removedPost = posts.value.find((post) => post.id === postId)
    void reportContentEvent('content_delete_confirmed', 'post', postId)
    const deleteResult = await deletePost(postId)
    if (deleteResult.hasChildren) {
      // 子回复仍依赖这个节点维持讨论树，因此保留楼层并切换为墓碑态。
      posts.value = posts.value.map((post) => post.id === postId
        ? { ...post, content: '', renderedContent: '', isAuthorDeleted: true }
        : post)
    } else {
      posts.value = posts.value.filter((post) => post.id !== postId)
    }
    if (targetPostId.value === postId) {
      targetPostId.value = 0
    }
    if (editingPostId.value === postId) {
      editingPostId.value = 0
      postContent.value = postDraftBeforeEdit.value
      targetPostId.value = targetPostBeforeEdit.value
      postDraftBeforeEdit.value = ''
      targetPostBeforeEdit.value = 0
    }
    if (!deleteResult.hasChildren && removedPost?.postNo && activePostNo.value === removedPost.postNo) {
      const closest = findClosestLoadedPost(removedPost.postNo)
      activePostNo.value = closest?.postNo || lastPostNo(posts.value) || firstPostNo(posts.value) || 1
    }
    syncLoadedPostWindowBounds()
    syncProgressForPostNo(activePostNo.value || 1)
    await nextTick()
    collectPostElements()
    scheduleActivePostFromScroll()
    successMessage.value = t('topic.replyDeleted')
    pendingDeletePost.value = null
  } catch (error) {
    deleteErrorMessage.value = error instanceof Error ? error.message : t('api.replyDeleteFailed')
  } finally {
    deletingPostId.value = 0
  }
}

async function openPostHistory(post: PostPayload) {
  historyRequestSeq.value++
  const seq = historyRequestSeq.value
  historyPost.value = post
  historyVersions.value = null
  historyHasMore.value = false
  historyBeforeVersion.value = 0
  historyError.value = ''
  historyLoading.value = true
  try {
    const result = await getPostRevisions(post.id)
    if (seq !== historyRequestSeq.value) return // 弹窗已关闭/重新打开，丢弃过期响应
    historyVersions.value = result.versions
    historyHasMore.value = result.hasMore
    historyBeforeVersion.value = result.beforeVersion
  } catch (error) {
    if (seq !== historyRequestSeq.value) return
    historyError.value = error instanceof Error ? error.message : t('api.revisionsLoadFailed')
  } finally {
    if (seq === historyRequestSeq.value) historyLoading.value = false
  }
}

// 加载更早版本：游标分页（后端按 beforeVersion 返回更早一页，升序排列），
// 前插到列表头部，避免单次响应随编辑次数无界增长。
async function loadEarlierHistoryVersions() {
  if (historyLoadingMore.value || !historyHasMore.value || !historyPost.value) return
  const seq = historyRequestSeq.value
  historyLoadingMore.value = true
  try {
    const result = await getPostRevisions(historyPost.value.id, historyBeforeVersion.value)
    if (seq !== historyRequestSeq.value) return // 弹窗已关闭/重新打开，丢弃过期响应
    historyVersions.value = [...result.versions, ...(historyVersions.value ?? [])]
    historyHasMore.value = result.hasMore
    historyBeforeVersion.value = result.beforeVersion
  } catch (error) {
    if (seq !== historyRequestSeq.value) return
    historyError.value = error instanceof Error ? error.message : t('api.revisionsLoadFailed')
  } finally {
    if (seq === historyRequestSeq.value) historyLoadingMore.value = false
  }
}

function closePostHistory() {
  if (historyLoading.value) return
  historyRequestSeq.value++
  historyPost.value = null
  historyVersions.value = null
  historyHasMore.value = false
  historyBeforeVersion.value = 0
  historyError.value = ''
  historyLoadingMore.value = false
}

function lastEditedLabel(post: PostPayload) {
  if (!post.lastEditedAt || !post.lastEditor) return ''
  return t('topic.lastEditedBy', {
    time: formatDateTime(post.lastEditedAt),
    user: authorDisplayName(post.lastEditor),
  })
}

defineExpose({ openFloatingPostComposer, focusPostComposer })
</script>

<template>
  <section class="gf-card" :class="wide ? 'xl:w-[calc(100%+292px)]' : ''" @click="handleMarkdownImageClick">
    <div class="min-w-0" :class="hasAside ? 'xl:grid xl:grid-cols-[minmax(0,1fr)_256px]' : ''">
      <div class="min-w-0">
        <slot name="before" />

        <span v-if="posts.length" id="posts" class="block scroll-mt-20" aria-hidden="true" />

        <div v-if="postHasBefore" class="relative px-4 py-3 text-center">
          <button
            v-if="postHasBefore"
            type="button"
            class="inline-flex h-8 items-center gap-1.5 rounded-md px-2 text-xs font-semibold text-primary transition-colors hover:bg-info/10 hover:text-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="loadingPostWindow"
            @click="loadPostWindow('before')"
          >
            <Loader2 v-if="loadingPostDirection === 'before'" class="h-3.5 w-3.5 animate-spin" />
            <ChevronsUp v-else class="h-3.5 w-3.5" />
            {{ t('topic.loadEarlierReplies') }}
          </button>
        </div>
        <!-- 视图切换胶囊：按内容类型默认（提问=树状，其余=扁平），用户选择按类型记忆 -->
        <div v-if="posts.length" class="flex items-center justify-end px-4 pt-2.5 sm:px-5">
          <div class="flex items-center rounded-full bg-base-200/70 p-0.5" role="group">
            <button
              type="button"
              class="rounded-full px-3 py-1 text-xs font-medium transition-colors"
              :class="postViewMode === 'flat' ? 'bg-base-100 text-base-content shadow-sm' : 'text-base-content/55 hover:text-base-content'"
              :aria-pressed="postViewMode === 'flat'"
              @click="setPostViewMode('flat')"
            >
              {{ t('topic.viewFlat') }}
            </button>
            <button
              type="button"
              class="rounded-full px-3 py-1 text-xs font-medium transition-colors"
              :class="postViewMode === 'tree' ? 'bg-base-100 text-base-content shadow-sm' : 'text-base-content/55 hover:text-base-content'"
              :aria-pressed="postViewMode === 'tree'"
              @click="setPostViewMode('tree')"
            >
              {{ t('topic.viewTree') }}
            </button>
          </div>
        </div>

        <article
          v-for="(post, index) in renderPosts"
          :id="`post-${post.id}`"
          :key="post.id"
          :data-post-no="post.postNo"
          class="group relative scroll-mt-20 transition-[background-color]"
          :class="[
            isFirstPost(post) && hasShortFormImages
              ? 'flex flex-col px-3 pt-0 pb-4 sm:grid sm:grid-cols-[52px_minmax(0,1fr)] sm:gap-4 sm:p-5'
              : 'grid grid-cols-[40px_minmax(0,1fr)] gap-2.5 px-3 py-4 sm:grid-cols-[52px_minmax(0,1fr)] sm:gap-4 sm:p-5',
            {
              'border-t border-line xl:border-t-transparent': index > 0,
              'bg-info/10': highlightedPostId === post.id,
              '[border-top-left-radius:calc(var(--gf-radius-box)-var(--gf-border))] [border-top-right-radius:calc(var(--gf-radius-box)-var(--gf-border))]': index === 0 && !postHasBefore,
            },
          ]"
        >
          <div v-if="index > 0" class="pointer-events-none absolute left-5 right-5 top-0 hidden border-t border-line xl:block" aria-hidden="true" />

          <!-- 移动端短文置顶多图轮播视窗（小红书/现代社媒图文风格：图窗在最上，满宽展开） -->
          <div
            v-if="isFirstPost(post) && hasShortFormImages"
            class="block sm:hidden -mx-3 mb-3.5 overflow-hidden [border-top-left-radius:calc(var(--gf-radius-box)-var(--gf-border))] [border-top-right-radius:calc(var(--gf-radius-box)-var(--gf-border))]"
          >
            <TopicImageGallery
              :images="topicImages!"
              :title="props.topicTitle"
            />
          </div>

          <!-- 移动端短文作者与头像栏（图窗下方，正文上方） -->
          <div
            v-if="isFirstPost(post) && hasShortFormImages"
            class="flex sm:hidden items-center justify-between gap-2.5 mb-2.5"
          >
            <div class="flex items-center gap-2.5 min-w-0">
              <a
                v-if="!isAnonymousPost(post)"
                :href="`/u/${post.author.id}`"
                class="shrink-0 pt-0.5"
                @click="showUserCard(post.author, $event)"
              >
                <UserAvatar
                  :src="post.author.avatarUrl"
                  :alt="post.author.username"
                  :badge="post.author.wornBadge"
                  class="h-9 w-9 rounded-full ring-1 ring-line"
                  img-class="rounded-full"
                />
              </a>
              <span v-else class="shrink-0 pt-0.5">
                <UserAvatar :src="anonymousAvatarSrc(post)" :alt="t('topic.authorAnonymous')" class="h-9 w-9 rounded-full ring-1 ring-line" img-class="rounded-full" />
              </span>
              <div class="min-w-0 flex flex-col">
                <div class="flex items-center gap-1.5 min-w-0">
                  <a
                    v-if="!isAnonymousPost(post)"
                    :href="`/u/${post.author.id}`"
                    class="min-w-0 truncate font-semibold text-sm text-base-content hover:text-primary"
                  >
                    {{ authorDisplayName(post.author) }}
                  </a>
                  <span v-else class="min-w-0 truncate font-semibold text-sm text-base-content/55">{{ t('topic.authorAnonymous') }}</span>
                  <span
                    v-if="props.contentType === 1"
                    class="shrink-0 inline-flex items-center gap-0.5 rounded-full bg-success/15 px-1.5 py-0.2 text-[10px] font-semibold text-success"
                  >
                    <HelpCircle class="h-2.5 w-2.5" />
                    {{ t('publish.contentTypes.question') }}
                  </span>
                  <span
                    v-else-if="props.contentType === 2"
                    class="shrink-0 inline-flex items-center gap-0.5 rounded-full bg-purple-500/15 px-1.5 py-0.2 text-[10px] font-semibold text-purple-600 dark:text-purple-400"
                  >
                    <Sparkles class="h-2.5 w-2.5" />
                    {{ t('publish.contentTypes.thought') }}
                  </span>
                </div>
                <time class="text-xs text-base-content/55">{{ formatDateTime(post.createdAt) }}</time>
              </div>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <button
                v-if="canEditPost(post)"
                type="button"
                class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-icon-muted transition hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="savingEditPostId === post.id || deletingPostId === post.id"
                :title="t('common.edit')"
                @click="startEditPost(post)"
              >
                <PencilLine class="h-3.5 w-3.5" />
                <span class="sr-only">{{ t('common.edit') }}</span>
              </button>
            </div>
          </div>

          <a
            v-if="!isAnonymousPost(post)"
            :href="`/u/${post.author.id}`"
            class="sticky top-19 self-start pt-1"
            :class="isFirstPost(post) && hasShortFormImages ? 'hidden sm:block' : 'block'"
            @click="showUserCard(post.author, $event)"
          >
            <UserAvatar :src="post.author.avatarUrl" :alt="post.author.username" :badge="post.author.wornBadge" class="h-9 w-9 rounded-full ring-1 ring-line sm:h-10 sm:w-10" img-class="rounded-full" />
          </a>
          <span
            v-else
            class="sticky top-19 self-start pt-1"
            :class="isFirstPost(post) && hasShortFormImages ? 'hidden sm:block' : 'block'"
          >
            <UserAvatar :src="anonymousAvatarSrc(post)" :alt="t('topic.authorAnonymous')" class="h-9 w-9 rounded-full ring-1 ring-line sm:h-10 sm:w-10" img-class="rounded-full" />
          </span>
          <div class="min-w-0">
            <div
              class="mb-1.5 min-w-0 items-start justify-between gap-2"
              :class="isFirstPost(post) && hasShortFormImages ? 'hidden sm:flex' : 'flex'"
            >
              <div class="min-w-0">
                <div class="flex min-w-0 items-center gap-2">
                  <a v-if="!isAnonymousPost(post)" :href="`/u/${post.author.id}`" class="min-w-0 truncate font-semibold text-base-content hover:text-primary">{{ authorDisplayName(post.author) }}</a>
                  <span v-else class="min-w-0 truncate font-semibold text-base-content/55">{{ t('topic.authorAnonymous') }}</span>
                  <span v-if="post.postNo" class="hidden shrink-0 text-xs font-semibold tabular-nums text-base-content/55 sm:inline">#{{ formatNumber(post.postNo) }}</span>
                  <span v-if="isQuestionTopic && post.isAnswer" class="hidden shrink-0 rounded bg-success/20 px-1.5 py-0.5 text-[11px] font-semibold text-success sm:inline">{{ t('topic.answer') }}</span>
                </div>
                <div class="mt-0.5 flex items-center gap-2 text-xs text-base-content/55 sm:hidden">
                  <span v-if="post.postNo" class="font-semibold tabular-nums text-base-content/55">#{{ formatNumber(post.postNo) }}</span>
                  <span v-if="isQuestionTopic && post.isAnswer" class="shrink-0 rounded bg-success/20 px-1.5 py-0.5 text-[11px] font-semibold text-success">{{ t('topic.answer') }}</span>
                  <time class="truncate">{{ formatDateTime(post.createdAt) }}</time>
                </div>
              </div>
              <div class="flex shrink-0 items-center gap-1 sm:gap-1.5">
                <!-- 1. 编辑按钮：作者/可编辑者可见（首楼与回复楼层均可用） -->
                <button
                  v-if="canEditPost(post)"
                  type="button"
                  class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-icon-muted transition hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="savingEditPostId === post.id || deletingPostId === post.id"
                  :title="t('common.edit')"
                  @click="startEditPost(post)"
                >
                  <PencilLine class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('common.edit') }}</span>
                </button>

                <!-- 2. 操作按钮组：非首楼移动+桌面均展示；首楼仅在桌面端展示快捷按钮，移动端收纳到首楼底部避免昵称被截断 -->
                <button
                  v-if="canDeleteRenderedPost(post)"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  :class="isFirstPost(post) ? 'hidden sm:inline-flex' : 'inline-flex'"
                  :disabled="deletingPostId === post.id"
                  :title="deletingPostId === post.id ? t('topic.deleting') : t('topic.delete')"
                  @click="requestDeletePost(post)"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ deletingPostId === post.id ? t('topic.deleting') : t('topic.delete') }}</span>
                </button>
                <button
                  v-if="(!viewer.isAuthenticated || canPost) && !post.isHidden && !isPostRemoved(post)"
                  type="button"
                  class="h-7 w-7 shrink-0 items-center justify-center rounded-md text-icon-muted transition hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 sm:h-8 sm:w-8"
                  :class="isFirstPost(post) ? 'hidden sm:inline-flex' : 'inline-flex'"
                  :title="t('topic.reply')"
                  @click="replyTo(post)"
                >
                  <CornerDownLeft class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.reply') }}</span>
                </button>
                <button
                  v-if="viewer.isAuthenticated && !post.isHidden && !isPostRemoved(post)"
                  type="button"
                  class="h-7 shrink-0 items-center gap-1 rounded-md px-1 text-icon-muted transition hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 sm:h-8 sm:px-1.5"
                  :class="[
                    isFirstPost(post) ? 'hidden sm:inline-flex' : 'inline-flex',
                    { 'text-error hover:text-error': postActionState(post).isLiked },
                  ]"
                  :title="t('topic.like')"
                  :disabled="postActionState(post).actingLike"
                  @click="togglePostLike(post)"
                >
                  <Heart class="h-3.5 w-3.5" :fill="postActionState(post).isLiked ? 'currentColor' : 'none'" />
                  <span v-if="postActionState(post).likeCount" class="hidden text-xs font-semibold tabular-nums sm:inline">{{ formatNumber(postActionState(post).likeCount) }}</span>
                  <span class="sr-only">{{ t('topic.like') }}</span>
                </button>
                <button
                  v-if="viewer.isAuthenticated && !post.isHidden && !isPostRemoved(post)"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  :class="[
                    isFirstPost(post) ? 'hidden sm:inline-flex' : 'inline-flex',
                    { 'text-primary hover:text-primary': postActionState(post).isBookmarked },
                  ]"
                  :title="postActionState(post).isBookmarked ? t('topic.bookmarked') : t('topic.bookmark')"
                  :disabled="postActionState(post).actingBookmark"
                  @click="togglePostBookmark(post)"
                >
                  <Bookmark class="h-3.5 w-3.5" :fill="postActionState(post).isBookmarked ? 'currentColor' : 'none'" />
                  <span class="sr-only">{{ t('topic.bookmark') }}</span>
                </button>
                <button
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-base-200 hover:text-base-content focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
                  :class="isFirstPost(post) ? 'hidden sm:inline-flex' : 'inline-flex'"
                  :title="t('topic.share')"
                  @click="sharePost(post)"
                >
                  <Share2 class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.share') }}</span>
                </button>
                <button
                  v-if="!post.isOwnPost && !post.isHidden && !isPostRemoved(post)"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-warning/10 hover:text-warning focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warning focus-visible:ring-offset-2"
                  :class="isFirstPost(post) ? 'hidden sm:inline-flex' : 'inline-flex'"
                  :title="t('topic.report')"
                  @click="requestPostReport(post)"
                >
                  <Flag class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.report') }}</span>
                </button>
                <button
                  v-if="post.canModerate && post.processStatus === 0"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error focus-visible:ring-offset-2 disabled:opacity-50"
                  :class="isFirstPost(post) ? 'hidden sm:inline-flex' : 'inline-flex'"
                  :disabled="postModerationBusy(post.id)"
                  :title="t('topic.moderationBan')"
                  @click="moderatePost(post, 'ban')"
                >
                  <Ban class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.moderationBan') }}</span>
                </button>
                <button
                  v-else-if="post.canModerate && post.processStatus === 1"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:opacity-50"
                  :class="isFirstPost(post) ? 'hidden sm:inline-flex' : 'inline-flex'"
                  :disabled="postModerationBusy(post.id)"
                  :title="t('topic.moderationUnban')"
                  @click="moderatePost(post, 'unban')"
                >
                  <RotateCcw class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.moderationUnban') }}</span>
                </button>

                <!-- 大屏时间展示 -->
                <time class="hidden w-36 shrink-0 text-right text-xs text-base-content/55 sm:-ml-1 sm:block">{{ formatDateTime(post.createdAt) }}</time>

                <!-- 首楼内容类型徽章：只显示唯一且明确的类型徽章，未配置时回退为“正文” -->
                <template v-if="isFirstPost(post)">
                  <span
                    v-if="props.contentType === 1"
                    class="shrink-0 self-center inline-flex items-center gap-1 rounded-full bg-success/15 px-2 py-0.5 text-[11px] font-semibold text-success"
                  >
                    <HelpCircle class="h-3 w-3" />
                    {{ t('publish.contentTypes.question') }}
                  </span>
                  <span
                    v-else-if="props.contentType === 2"
                    class="shrink-0 self-center inline-flex items-center gap-1 rounded-full bg-purple-500/15 px-2 py-0.5 text-[11px] font-semibold text-purple-600 dark:text-purple-400"
                  >
                    <Sparkles class="h-3 w-3" />
                    {{ t('publish.contentTypes.thought') }}
                  </span>
                  <span
                    v-else-if="props.contentType === 3"
                    class="shrink-0 self-center inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-[11px] font-semibold text-amber-600 dark:text-amber-400"
                  >
                    <BookOpen class="h-3 w-3" />
                    {{ t('publish.contentTypes.article') }}
                  </span>
                  <span
                    v-else
                    class="shrink-0 self-center rounded bg-base-200 px-1 py-0.5 text-[11px] font-semibold text-base-content/55 sm:px-1.5 sm:text-xs"
                  >
                    {{ t('topic.originalPost') }}
                  </span>
                </template>
              </div>
            </div>
            <PostReplyReference v-if="postViewMode !== 'tree' && showReplyReference(post)" :target="replyTargetFor(post)" />
            <!-- #520：树状视图不渲染全文引用条；孤根（目标在窗口外）以短提示兜底上下文，
                 目标不可见（隐藏/移除/清理）时降级为「原回复不可见」 -->
            <p
              v-else-if="postViewMode === 'tree' && treeRootReplyHint(post)"
              data-test="reply-context-hint"
              class="mb-2 inline-flex max-w-full items-center gap-1.5 rounded-full bg-base-200/60 px-2.5 py-1 text-xs font-medium text-base-content/55"
            >
              <CornerDownLeft class="h-3 w-3 shrink-0" aria-hidden="true" />
              <span v-if="treeRootReplyHint(post)!.unavailable" class="min-w-0 truncate">{{ t('topic.replyTargetUnavailable') }}</span>
              <template v-else>
                <span v-if="treeRootReplyHint(post)!.isAnonymous" class="min-w-0 truncate">{{ t('topic.authorAnonymous') }}</span>
                <span v-else class="min-w-0 truncate">{{ t('topic.replyTo', { user: `@${treeRootReplyHint(post)!.username}` }) }}</span>
                <span v-if="treeRootReplyHint(post)!.postNo" class="shrink-0 tabular-nums">#{{ treeRootReplyHint(post)!.postNo }}</span>
              </template>
            </p>
            <div v-if="post.isAuthorDeleted" class="rounded border border-dashed border-line bg-base-200/60 px-3 py-3 text-sm text-base-content/55">
              <div class="font-semibold text-base-content/70">{{ t('topic.authorDeletedTitle') }}</div>
              <div class="mt-1 leading-6">{{ t('topic.authorDeletedPlaceholder') }}</div>
            </div>
            <div v-else-if="post.isModeratorRemoved" class="rounded border border-dashed border-line bg-base-200/60 px-3 py-3 text-sm text-base-content/55">
              <div class="font-semibold text-base-content/70">{{ t('topic.moderatorRemovedTitle') }}</div>
              <div class="mt-1 leading-6">{{ t('topic.moderatorRemovedPlaceholder') }}</div>
            </div>
            <div v-else-if="post.isHidden && !post.canModerate" class="rounded border border-line bg-base-200/60 px-3 py-2 text-sm text-base-content/45">
              {{ t('topic.hiddenReplyPlaceholder') }}
            </div>
            <div v-else>
              <!-- 置顶多图轮播视窗（移动端已在最顶部置顶展示，桌面端在此保留） -->
              <TopicImageGallery
                v-if="isShortFormTopic && isFirstPost(post) && topicImages && topicImages.length > 0"
                :images="topicImages"
                :title="props.topicTitle"
                class="mb-4 hidden sm:block"
              />
              <div
                v-code-copy
                v-code-highlight
                v-math-render
                v-content-enhancements
                class="gf-prose gf-prose-post"
                :class="{ 'gf-prose-article': isBlogLikeTopic && isFirstPost(post), 'gf-prose-thought': props.contentType === 2 && isFirstPost(post) }"
                v-html="renderedPostContent(post)"
              />
            </div>
            <div v-if="post.isHidden && !isPostRemoved(post) && post.canModerate" class="mt-2 inline-flex rounded bg-base-200 px-2 py-1 text-xs font-semibold text-base-content/45">
              {{ t('topic.hiddenReplyBadge') }}
            </div>
            <div v-if="!post.lastEditedAt && post.updatedAt && post.updatedAt !== post.createdAt" class="mt-2 text-xs font-medium text-base-content/55">
              {{ t('topic.editedAt', { time: formatDateTime(post.updatedAt) }) }}
            </div>
            <div v-if="post.lastEditedAt && post.lastEditor" class="mt-2 text-xs font-medium text-base-content/55">
              {{ lastEditedLabel(post) }}
            </div>
            <div v-if="isFirstPost(post) && topicActions" class="mt-4 border-t border-line/60 pt-3">
              <!-- 桌面端操作栏：完整平铺展开，不必收纳入更多菜单（sm 及以上屏幕显示） -->
              <div class="hidden sm:flex sm:items-center sm:justify-between sm:gap-2">
                <div class="flex flex-wrap items-center gap-2">
                  <!-- 回复 / 回答按钮（“问题”、“文章”、“瞬间”主操作醒目化，去除计数） -->
                  <button
                    v-if="(!viewer.isAuthenticated || canPost) && !isTopicRemoved()"
                    type="button"
                    class="gf-button gf-button-sm rounded-full active:scale-95 transition-all duration-150 flex items-center gap-1.5"
                    :class="isProminentReply
                      ? 'bg-primary text-primary-content font-medium px-3.5 shadow-sm hover:shadow hover:bg-primary/90'
                      : 'px-3 text-base-content/70 hover:bg-base-200 hover:text-base-content'"
                    @click="replyTo(firstPost || post)"
                  >
                    <CornerDownLeft class="h-4 w-4 shrink-0" />
                    <span>{{ isQuestionTopic ? t('topic.writeAnswer') : t('topic.reply') }}</span>
                  </button>

                  <!-- 点赞按钮 -->
                  <button
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-3 active:scale-95 transition-all"
                    :class="isLiked ? 'bg-error/10 text-error font-medium hover:bg-error/15' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
                    :disabled="actingLike || isTopicRemoved()"
                    @click="toggleLike"
                  >
                    <Heart class="h-4 w-4 shrink-0 transition-transform" :class="{ 'scale-110': isLiked }" :fill="isLiked ? 'currentColor' : 'none'" />
                    <span>{{ likeCount ? formatNumber(likeCount) : t('topic.like') }}</span>
                  </button>

                  <!-- 收藏按钮 -->
                  <button
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-3 active:scale-95 transition-all"
                    :class="isBookmarked ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 font-medium hover:bg-amber-500/15' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
                    :disabled="actingBookmark || isTopicRemoved()"
                    @click="toggleBookmark"
                  >
                    <Bookmark class="h-4 w-4 shrink-0 transition-transform" :class="{ 'scale-110': isBookmarked }" :fill="isBookmarked ? 'currentColor' : 'none'" />
                    <span>{{ isBookmarked ? t('topic.bookmarked') : t('topic.bookmark') }}</span>
                  </button>

                  <!-- 关注话题 -->
                  <button
                    v-if="viewer.isAuthenticated && !isTopicRemoved()"
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-3 text-base-content/70 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
                    :class="{ 'text-success font-medium hover:text-success': isWatched }"
                    :disabled="actingWatch"
                    @click="toggleWatch"
                  >
                    <Bell class="h-4 w-4 shrink-0" :fill="isWatched ? 'currentColor' : 'none'" />
                    <span>{{ isWatched ? t('topic.watched') : t('topic.watch') }}</span>
                  </button>

                  <!-- 查看编辑历史 -->
                  <button
                    v-if="post.revisionCount > 1"
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-3 text-base-content/70 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
                    @click="openPostHistory(post)"
                  >
                    <History class="h-4 w-4 shrink-0" />
                    <span>{{ t('topic.editHistory') }}</span>
                  </button>
                </div>

                <div class="flex items-center gap-1.5">
                  <!-- 分享按钮 -->
                  <button
                    type="button"
                    class="gf-icon-button h-8 w-8 rounded-full text-base-content/60 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
                    :title="t('topic.share')"
                    @click="sharePost(firstPost || post)"
                  >
                    <Share2 class="h-4 w-4" />
                    <span class="sr-only">{{ t('topic.share') }}</span>
                  </button>

                  <!-- 举报话题 -->
                  <button
                    v-if="viewer.isAuthenticated && !topicActions.isOwnTopic && !isTopicRemoved()"
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-2.5 text-base-content/70 hover:bg-warning/10 hover:text-warning active:scale-95 transition-all"
                    @click="requestTopicReport"
                  >
                    <Flag class="h-3.5 w-3.5 shrink-0" />
                    <span>{{ t('topic.report') }}</span>
                  </button>

                  <!-- 删除话题（作者） -->
                  <button
                    v-if="topicActions.isOwnTopic && !isTopicRemoved()"
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-2.5 text-error hover:bg-error/10 active:scale-95 transition-all"
                    @click="requestDeleteTopic"
                  >
                    <Trash2 class="h-3.5 w-3.5 shrink-0" />
                    <span>{{ t('topic.deleteTopic') }}</span>
                  </button>

                  <!-- 封禁/解封（版主） -->
                  <button
                    v-if="topicActions.canModerateTopic && topicProcessStatus === 0"
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-2.5 text-warning hover:bg-warning/10 active:scale-95 transition-all"
                    :disabled="actingModeration"
                    @click="requestTopicModeration('ban')"
                  >
                    <Ban class="h-3.5 w-3.5 shrink-0" />
                    <span>{{ t('topic.moderationBan') }}</span>
                  </button>
                  <button
                    v-else-if="topicActions.canModerateTopic && topicProcessStatus === 1"
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-2.5 text-primary hover:bg-info/10 active:scale-95 transition-all"
                    :disabled="actingModeration"
                    @click="requestTopicModeration('unban')"
                  >
                    <RotateCcw class="h-3.5 w-3.5 shrink-0" />
                    <span>{{ t('topic.moderationUnban') }}</span>
                  </button>
                </div>
              </div>

              <!-- 移动端操作栏：单行高频互动 + 优雅的 Popover 收纳菜单（<sm 屏幕显示） -->
              <div class="flex sm:hidden items-center justify-between gap-1">
                <!-- 左侧核心高频互动：回复、点赞、收藏 -->
                <div class="flex items-center gap-1.5">
                  <!-- 回复 / 回答按钮（“问题”、“文章”、“瞬间”移动端主操作醒目化，去除计数） -->
                  <button
                    v-if="(!viewer.isAuthenticated || canPost) && !isTopicRemoved()"
                    type="button"
                    class="gf-button gf-button-sm rounded-full active:scale-95 transition-all duration-150 flex items-center gap-1.5"
                    :class="isProminentReply
                      ? 'bg-primary text-primary-content font-medium px-3 shadow-xs hover:shadow hover:bg-primary/90'
                      : 'px-2.5 text-base-content/70 hover:bg-base-200 hover:text-base-content'"
                    @click="replyTo(firstPost || post)"
                  >
                    <CornerDownLeft class="h-3.5 w-3.5 shrink-0" />
                    <span>{{ isQuestionTopic ? t('topic.writeAnswer') : t('topic.reply') }}</span>
                  </button>

                  <!-- 点赞按钮 -->
                  <button
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-2.5 active:scale-95 transition-all"
                    :class="isLiked ? 'bg-error/10 text-error font-medium hover:bg-error/15' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
                    :disabled="actingLike || isTopicRemoved()"
                    @click="toggleLike"
                  >
                    <Heart class="h-3.5 w-3.5 shrink-0 transition-transform" :class="{ 'scale-110': isLiked }" :fill="isLiked ? 'currentColor' : 'none'" />
                    <span>{{ likeCount ? formatNumber(likeCount) : t('topic.like') }}</span>
                  </button>

                  <!-- 收藏按钮 -->
                  <button
                    type="button"
                    class="gf-button gf-button-sm rounded-full px-2.5 active:scale-95 transition-all"
                    :class="isBookmarked ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 font-medium hover:bg-amber-500/15' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
                    :disabled="actingBookmark || isTopicRemoved()"
                    @click="toggleBookmark"
                  >
                    <Bookmark class="h-3.5 w-3.5 shrink-0 transition-transform" :class="{ 'scale-110': isBookmarked }" :fill="isBookmarked ? 'currentColor' : 'none'" />
                  </button>
                </div>

                <!-- 右侧分享与更多收纳 -->
                <div class="flex items-center gap-1">
                  <!-- 分享按钮 -->
                  <button
                    type="button"
                    class="gf-icon-button h-8 w-8 rounded-full text-base-content/60 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
                    :title="t('topic.share')"
                    @click="sharePost(firstPost || post)"
                  >
                    <Share2 class="h-4 w-4" />
                    <span class="sr-only">{{ t('topic.share') }}</span>
                  </button>

                  <!-- 更多操作 Popover（关注、编辑历史、举报、删除、封禁） -->
                  <PopoverRoot v-model:open="moreActionsOpen">
                    <PopoverTrigger as-child>
                      <button
                        type="button"
                        class="gf-icon-button h-8 w-8 rounded-full text-base-content/60 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
                        :title="t('topic.more')"
                      >
                        <MoreHorizontal class="h-4 w-4" />
                        <span class="sr-only">{{ t('topic.more') }}</span>
                      </button>
                    </PopoverTrigger>
                    <PopoverPortal>
                      <PopoverContent
                        side="top"
                        align="end"
                        :side-offset="8"
                        :collision-padding="16"
                        class="z-[70] min-w-[160px] max-w-[200px] rounded-2xl border border-line/60 bg-base-100/98 p-1.5 shadow-[0_12px_36px_-6px_rgba(0,0,0,0.14),0_4px_12px_-2px_rgba(0,0,0,0.06)] backdrop-blur-md outline-none animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 duration-150"
                      >
                        <div class="flex flex-col gap-0.5" role="menu">
                          <!-- 关注话题 -->
                          <button
                            v-if="viewer.isAuthenticated && !isTopicRemoved()"
                            type="button"
                            role="menuitem"
                            class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-base-content/85 transition-colors hover:bg-base-200/80 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 cursor-pointer"
                            :class="{ 'text-success hover:text-success': isWatched }"
                            :disabled="actingWatch"
                            @click="toggleWatch(); moreActionsOpen = false"
                          >
                            <Bell class="h-4 w-4 shrink-0" :fill="isWatched ? 'currentColor' : 'none'" />
                            <span>{{ isWatched ? t('topic.watched') : t('topic.watch') }}</span>
                          </button>

                          <!-- 查看编辑历史 -->
                          <button
                            v-if="post.revisionCount > 1"
                            type="button"
                            role="menuitem"
                            class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-base-content/85 transition-colors hover:bg-base-200/80 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 cursor-pointer"
                            @click="openPostHistory(post); moreActionsOpen = false"
                          >
                            <History class="h-4 w-4 shrink-0" />
                            <span>{{ t('topic.editHistory') }}</span>
                          </button>

                          <!-- 举报话题 -->
                          <button
                            v-if="viewer.isAuthenticated && !topicActions.isOwnTopic && !isTopicRemoved()"
                            type="button"
                            role="menuitem"
                            class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-base-content/85 transition-colors hover:bg-warning/10 hover:text-warning active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warning/60 cursor-pointer"
                            @click="requestTopicReport(); moreActionsOpen = false"
                          >
                            <Flag class="h-4 w-4 shrink-0" />
                            <span>{{ t('topic.report') }}</span>
                          </button>

                          <div v-if="(topicActions.isOwnTopic || topicActions.canModerateTopic) && !isTopicRemoved()" class="my-1 border-t border-line/60" />

                          <!-- 删除话题（作者） -->
                          <button
                            v-if="topicActions.isOwnTopic && !isTopicRemoved()"
                            type="button"
                            role="menuitem"
                            class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-error transition-colors hover:bg-error/10 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error/60 cursor-pointer"
                            @click="requestDeleteTopic(); moreActionsOpen = false"
                          >
                            <Trash2 class="h-4 w-4 shrink-0" />
                            <span>{{ t('topic.deleteTopic') }}</span>
                          </button>

                          <!-- 封禁/解封（版主） -->
                          <button
                            v-if="topicActions.canModerateTopic && topicProcessStatus === 0"
                            type="button"
                            role="menuitem"
                            class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-warning transition-colors hover:bg-warning/10 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warning/60 cursor-pointer"
                            :disabled="actingModeration"
                            @click="requestTopicModeration('ban'); moreActionsOpen = false"
                          >
                            <Ban class="h-4 w-4 shrink-0" />
                            <span>{{ t('topic.moderationBan') }}</span>
                          </button>
                          <button
                            v-else-if="topicActions.canModerateTopic && topicProcessStatus === 1"
                            type="button"
                            role="menuitem"
                            class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-primary transition-colors hover:bg-info/10 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 cursor-pointer"
                            :disabled="actingModeration"
                            @click="requestTopicModeration('unban'); moreActionsOpen = false"
                          >
                            <RotateCcw class="h-4 w-4 shrink-0" />
                            <span>{{ t('topic.moderationUnban') }}</span>
                          </button>
                        </div>
                      </PopoverContent>
                    </PopoverPortal>
                  </PopoverRoot>
                </div>
              </div>
              <span v-if="actionMessage" class="mt-2 block text-xs" :class="actionMessageSuccess ? 'text-base-content/75' : 'text-error'">{{ actionMessage }}</span>
            </div>

          </div>
          <!-- 树状视图：真实父子嵌套；父子上下文由嵌套缩进表达（#520 不再渲染引用条），
               孤根短提示在主流层兜底；min-w-0 + overflow-x-clip 防深层行撑破视口（#515） -->
          <div
            v-if="postViewMode === 'tree' && treeRowsFor(post).length"
            class="col-span-2 mt-1 min-w-0 space-y-2 overflow-x-clip border-l-2 border-line/70"
          >
            <PostReplyRow
              v-for="row in treeRowsFor(post)"
              :id="`post-${row.post.id}`"
              :key="row.post.id"
              :data-post-no="row.post.postNo"
              :post="row.post"
              :highlighted="highlightedPostId === row.post.id"
              :authenticated="viewer.isAuthenticated"
              :can-post="canPost"
              :saving-edit="savingEditPostId === row.post.id"
              :deleting="deletingPostId === row.post.id"
              :moderation-busy="postModerationBusy(row.post.id)"
              :action-state="postActionState(row.post)"
              :indent-level="treeIndentLevel(row.depth)"
              :collapsible="row.hasChildren"
              :collapsed="collapsedIds.has(row.post.id)"
              @reply="replyTo(row.post)"
              @edit="startEditPost(row.post)"
              @delete="requestDeletePost(row.post)"
              @like="togglePostLike(row.post)"
              @bookmark="togglePostBookmark(row.post)"
              @share="sharePost(row.post)"
              @report="requestPostReport(row.post)"
              @moderate="(action) => moderatePost(row.post, action)"
              @toggle-collapse="toggleTreeCollapse(row.post.id)"
            />
          </div>
        </article>

        <div v-if="postHasAfter || loadingPostDirection === 'after' || postWindowError || (!postHasAfter && posts.length)" ref="postLoadMoreEl" class="relative border-t border-line px-4 py-3 text-center xl:border-t-transparent">
          <div class="pointer-events-none absolute left-5 right-5 top-0 hidden border-t border-line xl:block" aria-hidden="true" />
          <button
            v-if="postHasAfter && postWindowError"
            type="button"
            class="gf-button gf-button-sm gf-button-secondary text-xs"
            :disabled="loadingPostWindow"
            @click="loadPostWindow('after')"
          >
            <Loader2 v-if="loadingPostDirection === 'after'" class="h-3.5 w-3.5 animate-spin" />
            {{ t('topic.retryLoadReplies') }}
          </button>
          <p v-else-if="postWindowError" class="text-xs text-error">{{ postWindowError }}</p>
          <p v-else-if="postHasAfter && loadingPostDirection === 'after'" class="inline-flex items-center justify-center gap-1.5 text-xs font-medium text-base-content/55">
            <Loader2 class="h-3.5 w-3.5 animate-spin" />
            {{ t('topic.loadingMoreReplies') }}
          </p>
          <button
            v-else-if="postHasAfter"
            type="button"
            class="gf-button gf-button-sm gf-button-secondary text-xs"
            :disabled="loadingPostWindow"
            @click="loadMoreRepliesManually"
          >
            {{ t('topic.loadMoreReplies') }}
          </button>
          <p v-else-if="!postHasAfter && posts.length" class="text-xs font-medium text-base-content/55">{{ t('topic.allRepliesShown') }}</p>
        </div>
        <span class="block h-px scroll-mb-28" aria-hidden="true" />
      </div>

      <aside v-if="hasAside" class="hidden min-w-0 xl:block">
        <slot name="aside">
          <div v-if="topicActions" class="sticky top-19">
            <div class="px-4 py-4">
              <h2 class="text-sm font-semibold text-base-content/55">{{ t('topic.overview') }}</h2>
            </div>

            <dl class="space-y-4 border-t border-line px-4 py-5 text-sm">
              <div class="flex items-center justify-between gap-4">
                <dt class="font-semibold text-base-content/55">{{ t('topic.replyCount') }}</dt>
                <dd class="text-right font-semibold tabular-nums text-base-content">{{ formatNumber(topicActions.replyCount) }}</dd>
              </div>
              <div class="flex items-center justify-between gap-4">
                <dt class="font-semibold text-base-content/55">{{ t('topic.viewCount') }}</dt>
                <dd class="text-right font-semibold tabular-nums text-base-content">{{ formatNumber(topicActions.viewCount) }}</dd>
              </div>
              <div class="flex items-center justify-between gap-4">
                <dt class="font-semibold text-base-content/55">{{ t('topic.participants') }}</dt>
                <dd class="text-right font-semibold tabular-nums text-base-content">{{ topicActions.participants.length }}</dd>
              </div>
            </dl>

            <div v-if="topicActions.participants.length" class="border-t border-line px-4 py-4">
              <h3 class="mb-3 text-sm font-semibold text-base-content/55">{{ t('topic.activeParticipants') }}</h3>
              <div class="flex flex-wrap gap-1.5">
                <a
                  v-for="participant in topicActions.participants"
                  :key="participant.id"
                  :href="`/u/${participant.id}`"
                  class="rounded-full"
                  @click="showUserCard(participant, $event)"
                >
                  <UserAvatar :src="participant.avatarUrl" :alt="participant.username" class="h-8 w-8 rounded-full object-cover ring-1 ring-line transition hover:ring-primary/40" />
                </a>
              </div>
            </div>

            <PostPositionRail
              v-if="topicActions.replyCount > 0 && postMaxRange > 0"
              class="border-t border-line"
              :current="postRailCurrentNo"
              :max="postMaxRange"
              :start-label="postRailStartLabel"
              :end-label="postRailEndLabel"
              :current-label="postRailCurrentLabel"
              :busy="postRailBusy"
              :progress-current="postRailProgressCurrent"
              :progress-end="postRailProgressEnd"
              :progress-start="postRailProgressStart"
              @earliest="jumpToTopicBodyFromRail"
              @latest="jumpToLatestPostFromRail"
              @select="selectPostFromRail"
            />
          </div>
        </slot>
      </aside>

      <section v-if="hasHotTopics" class="border-t border-line xl:col-span-2">
        <div class="overflow-hidden bg-base-100 [border-bottom-left-radius:calc(var(--gf-radius-box)-1px)] [border-bottom-right-radius:calc(var(--gf-radius-box)-1px)]">
          <TopicList :topics="hotTopics!" home />
        </div>
      </section>
    </div>
  </section>

  <TopicFloatingControls
    v-if="topicActions"
    v-model:mobile-rail-open="mobilePostRailOpen"
    :open="composerOpen"
    :actions="floatingTopicActions"
    :authenticated="viewer.isAuthenticated"
    :can-post="canPost"
    :current-label="postRailCurrentLabel"
    :current-no="postRailCurrentNo"
    :end-label="postRailEndLabel"
    :has-rail="hasPostRail"
    :max-no="postMaxRange"
    :progress-current="postRailProgressCurrent"
    :progress-end="postRailProgressEnd"
    :progress-start="postRailProgressStart"
    :rail-busy="postRailBusy"
    :start-label="postRailStartLabel"
    @earliest="jumpToTopicBodyFromRail"
    @latest="jumpToLatestPostFromRail"
    @open-reply="openFloatingPostComposer"
    @select-rail="selectPostFromRail"
  />

  <PostComposer
    v-if="composerMounted"
    v-model="postContent"
    v-model:captcha-code="captchaCode"
    v-model:anonymous="anonymous"
    :allow-anonymous="allowAnonymous"
    :open="composerOpen"
    :authenticated="viewer.isAuthenticated"
    :captcha-img="captchaImg"
    :captcha-loading="captchaLoading"
    :captcha-required="captchaRequired"
    :current-user-id="viewer.id"
    :error-message="errorMessage"
    :mention-users="mentionUsers"
    :mode="composerMode"
    :submitting="editingPostId ? savingEditPostId > 0 : submitting"
    :success-message="successMessage"
    :sensitive-words="sensitiveWords"
    :target="targetPost"
    @clear-target="cancelPostTarget"
    @clear-validation="clearPostValidation"
    @image-error="handlePostImageError"
    @image-inserted="handlePostImageInserted"
    @refresh-captcha="loadCaptcha()"
    @submit="submitPost"
    @update:open="updateComposerOpen"
  />

  <MarkdownImageViewer ref="markdownImageViewer" />

  <Teleport to="body">
    <Transition name="gf-modal">
      <div
        v-if="pendingDeletePost"
        class="fixed inset-0 z-[110] overflow-y-auto bg-neutral/50 px-3 py-4 backdrop-blur-sm sm:px-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-post-title"
        aria-describedby="delete-post-description"
        @click.self="closeDeleteDialog"
      >
        <div class="mx-auto flex min-h-full max-w-md items-center justify-center">
          <div class="gf-menu-surface w-full p-4 sm:p-5">
            <div class="flex items-start gap-3">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-[var(--gf-radius-field)] bg-error/10 text-error ring-1 ring-error/15">
                <AlertTriangle class="h-5 w-5" aria-hidden="true" />
              </div>
              <div class="min-w-0 flex-1">
                <h2 id="delete-post-title" class="text-base font-semibold leading-6 text-base-content">{{ t('topic.deleteReplyTitle') }}</h2>
                <p id="delete-post-description" class="mt-1 text-sm leading-6 text-base-content/60">{{ t('topic.deleteReplyDescription') }}</p>
              </div>
              <button
                type="button"
                class="gf-icon-button -mr-1 -mt-1 h-8 w-8 shrink-0 text-base-content/45 transition-colors hover:bg-base-300 hover:text-base-content disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="Boolean(deletingPostId)"
                :aria-label="t('common.close')"
                @click="closeDeleteDialog"
              >
                <X class="h-4 w-4" aria-hidden="true" />
              </button>
            </div>

            <div class="mt-4 rounded-[var(--gf-radius-field)] border border-line bg-base-200/55 p-3">
              <div class="flex min-w-0 items-center gap-2">
                <UserAvatar
                  v-if="!isAnonymousPost(pendingDeletePost)"
                  :src="pendingDeletePost.author.avatarUrl"
                  :alt="pendingDeletePost.author.username"
                  class="h-6 w-6 shrink-0 rounded-full object-cover ring-1 ring-line"
                />
                <UserAvatar
                  v-else
                  :src="anonymousAvatarSrc(pendingDeletePost, 24)"
                  :alt="t('topic.authorAnonymous')"
                  class="h-6 w-6 shrink-0 rounded-full object-cover ring-1 ring-line"
                />
                <div class="min-w-0 truncate text-xs font-semibold text-base-content/55">
                  <template v-if="isAnonymousPost(pendingDeletePost)">{{ t('topic.authorAnonymous') }}</template>
                  <template v-else>@{{ pendingDeletePost.author.username }}</template>
                  <span class="ml-1.5 font-medium tabular-nums text-base-content/40">#{{ formatNumber(pendingDeletePost.postNo) }}</span>
                </div>
              </div>
              <p class="mt-2 line-clamp-3 whitespace-pre-wrap break-words text-sm leading-6 text-base-content/75 [overflow-wrap:anywhere]">{{ pendingDeletePost.content }}</p>
            </div>

            <p
              v-if="deleteErrorMessage"
              class="mt-3 rounded-[var(--gf-radius-field)] border border-error/20 bg-error/10 px-3 py-2 text-sm leading-5 text-error"
              role="alert"
            >
              {{ deleteErrorMessage }}
            </p>

            <div class="mt-3 flex items-start gap-2.5 rounded-[var(--gf-radius-field)] border border-line/80 bg-base-200/40 px-3 py-2.5">
              <Clock class="mt-0.5 h-3.5 w-3.5 shrink-0 text-base-content/45" aria-hidden="true" />
              <p class="text-xs leading-5 text-base-content/55">{{ t('topic.deleteNotice') }}</p>
            </div>

            <div class="mt-3">
              <button
                type="button"
                class="inline-flex min-h-8 items-center rounded-[var(--gf-radius-field)] px-1 text-left text-xs font-medium text-base-content/55 transition-colors hover:bg-base-200 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="Boolean(deletingPostId)"
                @click="privacyErasePost"
              >
                {{ t('topic.privacyErase') }}
              </button>
            </div>

            <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <button
                type="button"
                class="gf-button gf-button-lg gf-button-muted active:scale-[0.96]"
                :disabled="Boolean(deletingPostId)"
                @click="closeDeleteDialog"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-lg gf-button-danger active:scale-[0.96]"
                :disabled="Boolean(deletingPostId)"
                :aria-busy="deletingPostId === pendingDeletePost.id"
                @click="removePost(pendingDeletePost.id)"
              >
                <Loader2 v-if="deletingPostId === pendingDeletePost.id" class="h-4 w-4 animate-spin" aria-hidden="true" />
                <Trash2 v-else class="h-4 w-4" aria-hidden="true" />
                {{ deletingPostId === pendingDeletePost.id ? t('topic.deleting') : t('topic.confirmDelete') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <Teleport to="body">
    <Transition name="gf-modal">
      <div
        v-if="pendingReport"
        class="fixed inset-0 z-[110] flex items-center justify-center bg-neutral/45 px-4 py-6 backdrop-blur-sm"
        role="dialog"
        aria-modal="true"
        aria-labelledby="report-title"
        @click.self="closeReportDialog"
      >
        <div class="gf-menu-surface w-full max-w-sm p-4">
          <div class="flex items-start gap-3">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning">
              <Flag class="h-5 w-5" />
            </div>
            <div class="min-w-0 flex-1">
              <h2 id="report-title" class="text-base font-bold text-base-content">{{ t('topic.reportTitle') }}</h2>
              <p class="mt-1 line-clamp-2 text-sm leading-6 text-base-content/55">{{ pendingReport.title }}</p>
            </div>
            <button
              type="button"
              class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75 disabled:cursor-not-allowed disabled:opacity-50"
              :disabled="reportSubmitting"
              @click="closeReportDialog"
            >
              <X class="h-4 w-4" />
            </button>
          </div>

          <div class="mt-4 space-y-3">
            <label v-for="reason in reportReasons" :key="reason" class="flex cursor-pointer items-center gap-2 text-sm text-base-content/75">
              <input v-model="reportReason" class="radio radio-sm" type="radio" name="report-reason" :value="reason" />
              <span>{{ t(`topic.reportReasons.${reason}`) }}</span>
            </label>
            <textarea
              v-model="reportNote"
              class="gf-textarea min-h-24"
              maxlength="300"
              :placeholder="t('topic.reportNotePlaceholder')"
            />
          </div>

          <p v-if="reportError" class="mt-3 text-sm text-error">{{ reportError }}</p>

          <div class="mt-4 flex justify-end gap-2">
            <button
              type="button"
              class="gf-button gf-button-md gf-button-muted"
              :disabled="reportSubmitting"
              @click="closeReportDialog"
            >
              {{ t('common.cancel') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-md gf-button-primary"
              :disabled="reportSubmitting"
              @click="submitCurrentReport"
            >
              <Loader2 v-if="reportSubmitting" class="h-4 w-4 animate-spin" />
              <Flag v-else class="h-4 w-4" />
              {{ reportSubmitting ? t('common.loadingShort') : t('topic.submitReport') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <Teleport to="body">
    <Transition name="gf-modal">
      <div
        v-if="pendingModerationAction"
        class="fixed inset-0 z-[110] flex items-center justify-center bg-neutral/45 px-4 py-6 backdrop-blur-sm"
        role="dialog"
        aria-modal="true"
        aria-labelledby="ban-topic-title"
        @click.self="closeTopicModerationDialog"
      >
        <div class="gf-menu-surface w-full max-w-sm p-4">
          <div class="flex items-start gap-3">
            <AlertTriangle class="mt-0.5 h-5 w-5 shrink-0 text-error" />
            <div class="min-w-0 flex-1">
              <h2 id="ban-topic-title" class="text-base font-bold text-base-content">
                {{ pendingModerationAction === 'ban' ? t('topic.moderationBanTitle') : t('topic.moderationUnbanTitle') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-base-content/55">
                {{ pendingModerationAction === 'ban' ? t('topic.moderationBanDescription') : t('topic.moderationUnbanDescription') }}
              </p>
            </div>
            <button
              type="button"
              class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75 disabled:cursor-not-allowed disabled:opacity-50"
              :disabled="actingModeration"
              @click="closeTopicModerationDialog"
            >
              <X class="h-4 w-4" />
            </button>
          </div>

          <div class="mt-4 flex justify-end gap-2">
            <button
              type="button"
              class="gf-button gf-button-md gf-button-muted"
              :disabled="actingModeration"
              @click="closeTopicModerationDialog"
            >
              {{ t('common.cancel') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-md gf-button-danger"
              :disabled="actingModeration"
              @click="updateTopicModerationFromDetail"
            >
              <Loader2 v-if="actingModeration" class="h-4 w-4 animate-spin" />
              <component :is="pendingModerationAction === 'ban' ? Ban : RotateCcw" v-else class="h-4 w-4" />
              {{ actingModeration ? t('common.loadingShort') : (pendingModerationAction === 'ban' ? t('topic.confirmModerationBan') : t('topic.confirmModerationUnban')) }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <Teleport to="body">
    <Transition name="gf-modal">
      <div
        v-if="pendingDeleteTopic"
        class="fixed inset-0 z-[110] overflow-y-auto bg-neutral/50 px-3 py-4 backdrop-blur-sm sm:px-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-topic-title"
        aria-describedby="delete-topic-description"
        @click.self="closeDeleteTopicDialog"
      >
        <div class="mx-auto flex min-h-full max-w-md items-center justify-center">
          <div class="gf-menu-surface w-full p-4 sm:p-5">
            <div class="flex items-start gap-3">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-[var(--gf-radius-field)] bg-error/10 text-error ring-1 ring-error/15">
                <AlertTriangle class="h-5 w-5" aria-hidden="true" />
              </div>
              <div class="min-w-0 flex-1">
                <h2 id="delete-topic-title" class="text-base font-semibold leading-6 text-base-content">{{ t('topic.deleteTopicTitle') }}</h2>
                <p id="delete-topic-description" class="mt-1 text-sm leading-6 text-base-content/60">{{ t('topic.deleteTopicDescription') }}</p>
              </div>
              <button
                type="button"
                class="gf-icon-button -mr-1 -mt-1 h-8 w-8 shrink-0 text-base-content/45 transition-colors hover:bg-base-300 hover:text-base-content disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="deletingTopic"
                :aria-label="t('common.close')"
                @click="closeDeleteTopicDialog"
              >
                <X class="h-4 w-4" aria-hidden="true" />
              </button>
            </div>

            <div class="mt-4 rounded-[var(--gf-radius-field)] border border-line bg-base-200/55 p-3">
              <div class="flex min-w-0 items-center gap-2">
                <UserAvatar
                  :src="topicActions?.author.avatarUrl || ''"
                  :alt="topicActions?.author.username || ''"
                  class="h-6 w-6 shrink-0 rounded-full object-cover ring-1 ring-line"
                />
                <div class="min-w-0 truncate text-xs font-semibold text-base-content/55">
                  {{ authorDisplayName(topicActions?.author || { username: '', nickname: '' }) }}
                  <span v-if="topicActions?.author.username" class="ml-1.5 font-medium text-base-content/40">@{{ topicActions.author.username }}</span>
                </div>
              </div>
              <div class="mt-2 line-clamp-2 break-words text-sm font-semibold leading-5 text-base-content [overflow-wrap:anywhere]">
                {{ topicTitle }}
              </div>
              <p
                v-if="topicActions?.description"
                class="mt-1.5 line-clamp-2 whitespace-pre-wrap break-words text-xs leading-5 text-base-content/55 [overflow-wrap:anywhere]"
              >
                {{ topicActions.description }}
              </p>
            </div>

            <p
              v-if="deleteErrorMessage"
              class="mt-3 rounded-[var(--gf-radius-field)] border border-error/20 bg-error/10 px-3 py-2 text-sm leading-5 text-error"
              role="alert"
            >
              {{ deleteErrorMessage }}
            </p>

            <div class="mt-3 flex items-start gap-2.5 rounded-[var(--gf-radius-field)] border border-line/80 bg-base-200/40 px-3 py-2.5">
              <Clock class="mt-0.5 h-3.5 w-3.5 shrink-0 text-base-content/45" aria-hidden="true" />
              <p class="text-xs leading-5 text-base-content/55">{{ t('topic.deleteNotice') }}</p>
            </div>

            <div class="mt-3">
              <button
                type="button"
                class="inline-flex min-h-8 items-center rounded-[var(--gf-radius-field)] px-1 text-left text-xs font-medium text-base-content/55 transition-colors hover:bg-base-200 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="deletingTopic"
                @click="privacyEraseTopic"
              >
                {{ t('topic.privacyErase') }}
              </button>
            </div>

            <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <button
                type="button"
                class="gf-button gf-button-lg gf-button-muted active:scale-[0.96]"
                :disabled="deletingTopic"
                @click="closeDeleteTopicDialog"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-lg gf-button-danger active:scale-[0.96]"
                :disabled="deletingTopic"
                :aria-busy="deletingTopic"
                @click="removeTopic"
              >
                <Loader2 v-if="deletingTopic" class="h-4 w-4 animate-spin" aria-hidden="true" />
                <Trash2 v-else class="h-4 w-4" aria-hidden="true" />
                {{ deletingTopic ? t('topic.deleting') : t('topic.confirmDelete') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <Teleport to="body">
    <Transition name="gf-modal">
      <div
        v-if="historyPost"
        class="fixed inset-0 z-[110] overflow-y-auto bg-neutral/50 px-3 py-4 backdrop-blur-sm sm:px-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="post-history-title"
        @click.self="closePostHistory"
      >
        <div class="mx-auto flex min-h-full max-w-2xl items-center justify-center">
          <div class="gf-menu-surface w-full p-4 sm:p-5">
            <div class="flex items-start gap-3">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-[var(--gf-radius-field)] bg-info/10 text-primary ring-1 ring-info/15">
                <History class="h-5 w-5" aria-hidden="true" />
              </div>
              <div class="min-w-0 flex-1">
                <h2 id="post-history-title" class="text-base font-semibold leading-6 text-base-content">{{ t('topic.editHistory') }}</h2>
                <p class="mt-1 text-sm leading-6 text-base-content/60">{{ t('topic.historyForPost', { no: formatNumber(historyPost.postNo) }) }}</p>
              </div>
              <button
                type="button"
                class="gf-icon-button -mr-1 -mt-1 h-8 w-8 shrink-0 text-base-content/45 transition-colors hover:bg-base-300 hover:text-base-content disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="historyLoading"
                :aria-label="t('common.close')"
                @click="closePostHistory"
              >
                <X class="h-4 w-4" aria-hidden="true" />
              </button>
            </div>

            <div v-if="historyLoading" class="mt-4 flex items-center justify-center gap-2 py-8 text-sm text-base-content/55">
              <Loader2 class="h-4 w-4 animate-spin" aria-hidden="true" />
              {{ t('common.loadingShort') }}
            </div>
            <p v-else-if="historyError" class="mt-4 text-sm text-error" role="alert">{{ historyError }}</p>
            <p v-else-if="!historyVersions || !historyVersions.length" class="mt-4 py-6 text-center text-sm text-base-content/55">
              {{ t('topic.historyEmpty') }}
            </p>
            <template v-else>
              <div class="mt-4 max-h-[60vh] space-y-3 overflow-y-auto pr-1">
                <div
                  v-for="version in historyVersions"
                  :key="version.version"
                  class="rounded-[var(--gf-radius-field)] border border-line bg-base-200/40 p-3"
                >
                  <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-base-content/55">
                    <span class="font-semibold tabular-nums text-base-content/75">{{ t('topic.version', { version: formatNumber(version.version) }) }}</span>
                    <span class="inline-flex min-w-0 items-center gap-1.5">
                      <UserAvatar :src="version.editor.avatarUrl" :alt="version.editor.username" class="h-4 w-4 shrink-0 rounded-full ring-1 ring-line" img-class="rounded-full" />
                      <span class="truncate">{{ authorDisplayName(version.editor) }}</span>
                    </span>
                    <time class="truncate">{{ formatDateTime(version.createdAt) }}</time>
                    <span v-if="version.processStatus !== 0 && !version.content" class="rounded bg-warning/10 px-1.5 py-0.5 font-semibold text-warning">
                      {{ t('topic.historyPending') }}
                    </span>
                  </div>
                  <div v-if="version.content" v-code-copy v-code-highlight v-math-render v-content-enhancements class="gf-prose gf-prose-post mt-2 border-t border-line/70 pt-2" v-html="version.renderedHTML" />
                  <p v-else class="mt-2 border-t border-line/70 pt-2 text-xs text-base-content/45">
                    {{ version.processStatus !== 0 ? t('topic.historyPendingPlaceholder') : t('topic.historyContentEmpty') }}
                  </p>
                </div>
              </div>
              <div v-if="historyHasMore" class="mt-4 flex justify-center">
                <button
                  type="button"
                  class="inline-flex h-8 items-center gap-1.5 rounded-md px-2 text-xs font-semibold text-primary transition-colors hover:bg-info/10 hover:text-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
                  :disabled="historyLoadingMore"
                  @click="loadEarlierHistoryVersions"
                >
                  <Loader2 v-if="historyLoadingMore" class="h-3.5 w-3.5 animate-spin" />
                  <ChevronsUp v-else class="h-3.5 w-3.5" />
                  {{ t('common.loadMore') }}
                </button>
              </div>
            </template>

            <div class="mt-5 flex justify-end">
              <button
                type="button"
                class="gf-button gf-button-lg gf-button-muted active:scale-[0.96]"
                @click="closePostHistory"
              >
                {{ t('common.close') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* Article styling: enhanced typography for long-form content */
.gf-prose-article {
  max-width: 100%;
  font-size: 1.0625rem;
  line-height: 1.8;
}

.gf-prose-article h1 {
  font-size: 2rem;
  margin-top: 2rem;
  margin-bottom: 1rem;
}

.gf-prose-article h2 {
  font-size: 1.75rem;
  margin-top: 1.75rem;
  margin-bottom: 0.875rem;
}

.gf-prose-article h3 {
  font-size: 1.5rem;
  margin-top: 1.5rem;
  margin-bottom: 0.75rem;
}

.gf-prose-article p {
  margin-bottom: 1.5rem;
}

.gf-prose-article img {
  display: block;
  max-width: 100%;
  height: auto;
  border-radius: 0.5rem;
  margin: 2rem auto;
}

.gf-prose-article pre {
  font-size: 0.9375rem;
  padding: 1.5rem;
  border-radius: 0.5rem;
}

.gf-prose-article blockquote {
  font-size: 1.125rem;
  padding-left: 1.5rem;
  border-left-width: 4px;
  font-style: italic;
}

/* Thought styling: compact and lightweight */
.gf-prose-thought {
  font-size: 1rem;
  line-height: 1.7;
}

.gf-prose-thought p {
  margin-bottom: 1rem;
}

.gf-prose-thought img {
  display: block;
  max-width: 100%;
  height: auto;
  border-radius: 0.375rem;
  margin: 1rem auto;
}
</style>
