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
import { AlertTriangle, Ban, Bell, Bookmark, ChevronDown, ChevronUp, ChevronsUp, Clock, CornerDownLeft, Flag, Heart, History, Loader2, PencilLine, RotateCcw, Share2, Trash2, X } from '@lucide/vue'
import { bookmarkTopic, deletePost, deleteTopic, getPostRevisions, getPostWindow, likeTopic, createPost, submitReport, updateModerationTopicStatus, updateModerationPostStatus, updatePost, watchTopic, likePost, bookmarkPost, reportContentEvent, privacyEraseContent, type PostRevisionResult } from '@/runtime/api'
import { formatDateTime, formatNumber } from '@/runtime/format'
import { useFlashMessages } from '@/runtime/flash-message'
import { fetchPage } from '@/runtime/router'
import { showUserCard } from '@/runtime/user-card-events'
import { measurePostViewportProgressFromRects } from '@/runtime/post-viewport-progress'
import MarkdownImageViewer from '@/site/components/MarkdownImageViewer.vue'
import PostPositionRail from '@/site/components/PostPositionRail.vue'
import PostReplyReference from '@/site/components/PostReplyReference.vue'
import TopicFloatingControls from '@/site/components/TopicFloatingControls.vue'
import TopicList from '@/site/components/TopicList.vue'
import UserAvatar from '@/site/components/UserAvatar.vue'
import type { PostPayload, PostWindowPayload, ReplyTargetPayload, TopicPayload, ViewerPayload } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'
import { useCaptchaChallenge } from '@/site/composables/useCaptchaChallenge'

const props = withDefaults(defineProps<{
  topicId: number
  topicTitle: string
  contentType?: 0 | 1 | 2 | 3
  initialPostStream: PostWindowPayload
  viewer: ViewerPayload
  canPost: boolean
  /** 话题级操作状态（首楼操作区 + 概览栏）。WikiPage 等场景不传则不渲染。 */
  topicActions?: PostStreamTopicActions
  /** 互动状态独立供给（Wiki 页面正文区操作与评论流共用同一话题）。 */
  interactions?: { likeCount: number; isLiked: boolean; isBookmarked: boolean; isWatched: boolean }
  hotTopics?: TopicPayload[]
  /** 隐藏首楼（postNo === 1）：Wiki 正文已在页面上方渲染，评论流不再重复。 */
  hideFirstPost?: boolean
  /** 隐藏首楼时是否连其回复也隐藏：Wiki 页只保留回复栏，不展示话题楼层列表。 */
  initialPostStreamHidden?: boolean
  /** 滚动时是否改写 /p/post/:id 路径（Wiki 页面关闭）。 */
  syncUrl?: boolean
  /** 初始流为空时自动加载第一页（Wiki 页面无 SSR 评论流）。 */
  autoLoadFirstWindow?: boolean
  /** 宽版卡片（默认）：右栏概览 + 卡片右伸 292px（对齐话题页 rail）。嵌入 Wiki 正文区时关闭。 */
  wide?: boolean
}>(), {
  wide: true,
})

const slots = useSlots()
const hasAside = computed(() => Boolean(props.topicActions) || Boolean(slots.aside))
// Wiki 文章页（initialPostStreamHidden）只保留回复栏：首楼楼层已被 postGroups 过滤，
// 这里同时隐藏底部「热门话题」话题列表，避免正文下方残留话题列表区块。
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
const likeCount = ref(props.interactions?.likeCount ?? props.topicActions?.likeCount ?? 0)
const isLiked = ref(props.interactions?.isLiked ?? props.topicActions?.isLiked ?? false)
const isBookmarked = ref(props.interactions?.isBookmarked ?? props.topicActions?.isBookmarked ?? false)
const isWatched = ref(props.interactions?.isWatched ?? props.topicActions?.isWatched ?? false)
const actionMessage = ref('')
const actingLike = ref(false)
const actingBookmark = ref(false)
const actingWatch = ref(false)
const actingModeration = ref(false)
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
  // 深层链接：若目标回复被折叠在"前 3 条预览"之外，自动展开所在分组
  const group = postGroups.value.find((g) => g.root.id === postId || g.replies.some((reply) => reply.id === postId))
  if (group && group.root.id !== postId) {
    const replyIndex = group.replies.findIndex((reply) => reply.id === postId)
    if (replyIndex >= nestedRepliesPreviewCount) {
      const next = new Set(expandedReplyGroups.value)
      next.add(group.root.id)
      expandedReplyGroups.value = next
    }
  }
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

interface NestedPostGroup {
  root: PostPayload
  replies: PostPayload[]
}

const nestedRepliesPreviewCount = 3
const expandedReplyGroups = ref<Set<number>>(new Set())

// 楼层分组：回复其目标楼层在当前窗口内可见的帖子，收进目标楼层的嵌套回复区展示
// （限制一层缩进；任意深度的嵌套回复持续上溯挂到真实 root 下，用引用条保留对话脉络）
const postGroups = computed<NestedPostGroup[]>(() => {
  const byId = new Map(posts.value.map((post) => [post.id, post]))
  const childrenByParent = new Map<number, PostPayload[]>()
  const roots: PostPayload[] = []

  // 沿 replyToPostId 链持续上溯到真实 root（防环：已访问节点直接截断）
  const resolveRoot = (post: PostPayload): PostPayload => {
    const visited = new Set<number>()
    let cursor: PostPayload = post
    while (cursor.replyToPostId) {
      if (visited.has(cursor.id)) break
      visited.add(cursor.id)
      const next = byId.get(cursor.replyToPostId)
      if (!next) break
      cursor = next
    }
    return cursor
  }

  for (const post of posts.value) {
    const parent = post.replyToPostId ? byId.get(post.replyToPostId) : undefined
    if (!parent) {
      roots.push(post)
      continue
    }
    // 持续上溯至真实 root：A→B→C→D 时 D 也必须挂到 A 下，保证有渲染出口
    const effectiveParent = resolveRoot(post)
    const siblings = childrenByParent.get(effectiveParent.id)
    if (siblings) siblings.push(post)
    else childrenByParent.set(effectiveParent.id, [post])
  }

  let groups = roots.map((root) => ({
    root,
    replies: (childrenByParent.get(root.id) ?? []).sort((a, b) => a.postNo - b.postNo),
  }))

  // 隐藏首楼（Wiki 正文已在页面上方渲染）：首楼本身不渲染，挂在首楼下的回复提升为独立楼层。
  // 提升后与其余 root 楼合并，按 postNo 升序重排，避免首楼回复（postNo > 1）插到更小的楼号之前。
  // initialPostStreamHidden 时连首楼回复也一并隐藏（Wiki 页只留回复栏，不展示话题列表楼层）。
  if (props.hideFirstPost) {
    const firstGroup = groups.find((group) => group.root.postNo === 1)
    if (firstGroup) {
      if (props.initialPostStreamHidden) {
        groups = groups.filter((group) => group.root.postNo !== 1)
      } else {
        const promotedReplies = firstGroup.replies.map((reply) => ({ root: reply, replies: [] as PostPayload[] }))
        groups = [
          ...promotedReplies,
          ...groups.filter((group) => group.root.postNo !== 1),
        ].sort((a, b) => (a.root.postNo || 0) - (b.root.postNo || 0))
      }
    }
  }

  return groups
})

// For Q&A topics, separate answers from comments
const isQuestionTopic = computed(() => props.contentType === 1)
// Blog-like content types (Articles and Thoughts) - no replies, different layout
const isBlogLikeTopic = computed(() => props.contentType === 2 || props.contentType === 3)

const answerGroups = computed<NestedPostGroup[]>(() => {
  if (!isQuestionTopic.value) return []
  return postGroups.value.filter((group) => group.root.isAnswer && group.root.postNo > 1)
})

const commentGroups = computed<NestedPostGroup[]>(() => {
  if (!isQuestionTopic.value) return postGroups.value
  // For Q&A: show question (postNo=1) and comments (non-answer posts)
  return postGroups.value.filter((group) => !group.root.isAnswer || group.root.postNo === 1)
})

// Choose which groups to render based on content type
const renderGroups = computed<NestedPostGroup[]>(() => {
  return isQuestionTopic.value ? commentGroups.value : postGroups.value
})

function visibleReplies(group: NestedPostGroup) {
  return expandedReplyGroups.value.has(group.root.id) ? group.replies : group.replies.slice(0, nestedRepliesPreviewCount)
}

function groupRepliesExpanded(group: NestedPostGroup) {
  return expandedReplyGroups.value.has(group.root.id)
}

function toggleGroupReplies(rootId: number) {
  const next = new Set(expandedReplyGroups.value)
  if (next.has(rootId)) next.delete(rootId)
  else next.add(rootId)
  expandedReplyGroups.value = next
}

function applyPostWindowPayload(payload: Awaited<ReturnType<typeof getPostWindow>>, mergeMode: 'replace' | 'prepend' | 'append') {
  const pageStartPostNo = firstPostNo(payload.posts)
  mergePosts(payload.posts, mergeMode)
  if (mergeMode === 'replace') {
    postPageStarts.value = pageStartPostNo ? [pageStartPostNo] : []
  } else if (pageStartPostNo && !postPageStarts.value.includes(pageStartPostNo)) {
    postPageStarts.value = [...postPageStarts.value, pageStartPostNo].sort((a, b) => a - b)
  }
  mergeReplyTargets(payload.replyTargets || [], mergeMode)
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
  composerOpen.value = true
}

function updateComposerOpen(open: boolean) {
  composerOpen.value = open
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
  focusPostComposer()
}

function cancelPostTarget() {
  targetPostId.value = 0
  errorMessage.value = ''
}

function clearPostValidation() {
  errorMessage.value = ''
  successMessage.value = ''
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

// 优先展示用户昵称，未设置昵称时回退到账号名
function authorDisplayName(author: { username: string; nickname?: string }) {
  return author.nickname || author.username
}

function canEditPost(post: PostPayload) {
  return post.isOwnPost && !post.isHidden && !isPostRemoved(post)
}

function canDeleteRenderedPost(post: PostPayload) {
  return post.isOwnPost && !post.isHidden && !isPostRemoved(post) && !isFirstPost(post)
}

function startEditPost(post: PostPayload) {
  if (savingEditPostId.value || deletingPostId.value === post.id) return
  // 首楼本质上是话题本体，其编辑应进入“发布话题”编辑态（可改标题/分类/正文），
  // 而不是回复楼层那样就地编辑正文。此分支在 PR #217 中被误删，属行为回归（issue #379）。
  if (isFirstPost(post)) {
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
  focusPostComposer()
}

function cancelEditPost() {
  if (savingEditPostId.value) return
  editingPostId.value = 0
  errorMessage.value = ''
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
  try {
    const createdPost = await createPost(props.topicId, content, postId, {
      captchaId: captchaId.value,
      captchaCode: captchaCode.value,
    })
    clearCaptcha()
    postContent.value = ''
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
      errorMessage.value = t('server.auth.captcha.invalid')
    } else {
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

        <article
          v-for="(group, index) in renderGroups"
          :id="`post-${group.root.id}`"
          :key="group.root.id"
          :data-post-no="group.root.postNo"
          class="group relative grid scroll-mt-20 grid-cols-[40px_minmax(0,1fr)] gap-2.5 px-3 py-4 transition-[background-color] sm:grid-cols-[52px_minmax(0,1fr)] sm:gap-4 sm:p-5"
          :class="{
            'border-t border-line xl:border-t-transparent': index > 0,
            'bg-info/10': highlightedPostId === group.root.id,
            '[border-top-left-radius:calc(var(--gf-radius-box)-var(--gf-border))] [border-top-right-radius:calc(var(--gf-radius-box)-var(--gf-border))]': index === 0 && !postHasBefore,
          }"
        >
          <div v-if="index > 0" class="pointer-events-none absolute left-5 right-5 top-0 hidden border-t border-line xl:block" aria-hidden="true" />
          <a
            :href="`/u/${group.root.author.id}`"
            class="sticky top-19 self-start pt-1"
            @click="showUserCard(group.root.author, $event)"
          >
            <UserAvatar :src="group.root.author.avatarUrl" :alt="group.root.author.username" :badge="group.root.author.wornBadge" class="h-9 w-9 rounded-full ring-1 ring-line sm:h-10 sm:w-10" img-class="rounded-full" />
          </a>
          <div class="min-w-0">
            <div class="mb-1.5 flex min-w-0 items-start justify-between gap-2">
              <div class="min-w-0">
                <div class="flex min-w-0 items-center gap-2">
                  <a :href="`/u/${group.root.author.id}`" class="min-w-0 truncate font-semibold text-base-content hover:text-primary">{{ authorDisplayName(group.root.author) }}</a>
                  <span v-if="group.root.postNo" class="hidden shrink-0 text-xs font-semibold tabular-nums text-base-content/55 sm:inline">#{{ formatNumber(group.root.postNo) }}</span>
                </div>
                <div class="mt-0.5 flex items-center gap-2 text-xs text-base-content/55 sm:hidden">
                  <span v-if="group.root.postNo" class="font-semibold tabular-nums text-base-content/55">#{{ formatNumber(group.root.postNo) }}</span>
                  <time class="truncate">{{ formatDateTime(group.root.createdAt) }}</time>
                </div>
              </div>
              <div class="flex shrink-0 items-center gap-0.5 sm:gap-1.5">
                <button
                  v-if="canEditPost(group.root)"
                  type="button"
                  class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-icon-muted transition hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="savingEditPostId === group.root.id || deletingPostId === group.root.id"
                  :title="t('common.edit')"
                  @click="startEditPost(group.root)"
                >
                  <PencilLine class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('common.edit') }}</span>
                </button>
                <button
                  v-if="canDeleteRenderedPost(group.root)"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="deletingPostId === group.root.id"
                  :title="deletingPostId === group.root.id ? t('topic.deleting') : t('topic.delete')"
                  @click="requestDeletePost(group.root)"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ deletingPostId === group.root.id ? t('topic.deleting') : t('topic.delete') }}</span>
                </button>
                <button
                  v-if="(!viewer.isAuthenticated || canPost) && !group.root.isHidden && !isPostRemoved(group.root)"
                  type="button"
                  class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-icon-muted transition hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 sm:h-8 sm:w-8"
                  :title="t('topic.reply')"
                  @click="replyTo(group.root)"
                >
                  <CornerDownLeft class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.reply') }}</span>
                </button>
                <button
                  v-if="viewer.isAuthenticated && !group.root.isHidden && !isPostRemoved(group.root)"
                  type="button"
                  class="inline-flex h-7 shrink-0 items-center gap-1 rounded-md px-1 text-icon-muted transition hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 sm:h-8 sm:px-1.5"
                  :class="{ 'text-error hover:text-error': postActionState(group.root).isLiked }"
                  :title="t('topic.like')"
                  :disabled="postActionState(group.root).actingLike"
                  @click="togglePostLike(group.root)"
                >
                  <Heart class="h-3.5 w-3.5" :fill="postActionState(group.root).isLiked ? 'currentColor' : 'none'" />
                  <span v-if="postActionState(group.root).likeCount" class="hidden text-xs font-semibold tabular-nums sm:inline">{{ formatNumber(postActionState(group.root).likeCount) }}</span>
                  <span class="sr-only">{{ t('topic.like') }}</span>
                </button>
                <button
                  v-if="viewer.isAuthenticated && !group.root.isHidden && !isPostRemoved(group.root)"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  :class="{ 'text-primary hover:text-primary': postActionState(group.root).isBookmarked }"
                  :title="postActionState(group.root).isBookmarked ? t('topic.bookmarked') : t('topic.bookmark')"
                  :disabled="postActionState(group.root).actingBookmark"
                  @click="togglePostBookmark(group.root)"
                >
                  <Bookmark class="h-3.5 w-3.5" :fill="postActionState(group.root).isBookmarked ? 'currentColor' : 'none'" />
                  <span class="sr-only">{{ t('topic.bookmark') }}</span>
                </button>
                <button
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-base-200 hover:text-base-content focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
                  :title="t('topic.share')"
                  @click="sharePost(group.root)"
                >
                  <Share2 class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.share') }}</span>
                </button>
                <button
                  v-if="!isFirstPost(group.root) && !group.root.isOwnPost && !group.root.isHidden && !isPostRemoved(group.root)"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-warning/10 hover:text-warning focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warning focus-visible:ring-offset-2"
                  :title="t('topic.report')"
                  @click="requestPostReport(group.root)"
                >
                  <Flag class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.report') }}</span>
                </button>
                <button
                  v-if="!isFirstPost(group.root) && group.root.canModerate && group.root.processStatus === 0"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error focus-visible:ring-offset-2 disabled:opacity-50"
                  :disabled="postModerationBusy(group.root.id)"
                  :title="t('topic.moderationBan')"
                  @click="moderatePost(group.root, 'ban')"
                >
                  <Ban class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.moderationBan') }}</span>
                </button>
                <button
                  v-else-if="!isFirstPost(group.root) && group.root.canModerate && group.root.processStatus === 1"
                  type="button"
                  class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:opacity-50"
                  :disabled="postModerationBusy(group.root.id)"
                  :title="t('topic.moderationUnban')"
                  @click="moderatePost(group.root, 'unban')"
                >
                  <RotateCcw class="h-3.5 w-3.5" />
                  <span class="sr-only">{{ t('topic.moderationUnban') }}</span>
                </button>
                <time class="hidden w-36 shrink-0 text-right text-xs text-base-content/55 sm:-ml-1 sm:block">{{ formatDateTime(group.root.createdAt) }}</time>
                <span
                  v-if="isFirstPost(group.root)"
                  class="shrink-0 self-center rounded bg-base-200 px-1 py-0.5 text-[11px] font-semibold text-base-content/55 sm:px-1.5 sm:text-xs"
                >
                  {{ t('topic.originalPost') }}
                </span>
              </div>
            </div>
            <PostReplyReference v-if="group.root.replyToPostId" :target="replyTargetFor(group.root)" />
            <div v-if="group.root.isAuthorDeleted" class="rounded border border-dashed border-line bg-base-200/60 px-3 py-3 text-sm text-base-content/55">
              <div class="font-semibold text-base-content/70">{{ t('topic.authorDeletedTitle') }}</div>
              <div class="mt-1 leading-6">{{ t('topic.authorDeletedPlaceholder') }}</div>
            </div>
            <div v-else-if="group.root.isModeratorRemoved" class="rounded border border-dashed border-line bg-base-200/60 px-3 py-3 text-sm text-base-content/55">
              <div class="font-semibold text-base-content/70">{{ t('topic.moderatorRemovedTitle') }}</div>
              <div class="mt-1 leading-6">{{ t('topic.moderatorRemovedPlaceholder') }}</div>
            </div>
            <div v-else-if="group.root.isHidden && !group.root.canModerate" class="rounded border border-line bg-base-200/60 px-3 py-2 text-sm text-base-content/45">
              {{ t('topic.hiddenReplyPlaceholder') }}
            </div>
            <div v-else v-code-copy v-code-highlight v-math-render class="gf-prose gf-prose-post" :class="{ 'gf-prose-article': isBlogLikeTopic && isFirstPost(group.root), 'gf-prose-thought': props.contentType === 2 && isFirstPost(group.root) }" v-html="group.root.renderedContent" />
            <div v-if="group.root.isHidden && !isPostRemoved(group.root) && group.root.canModerate" class="mt-2 inline-flex rounded bg-base-200 px-2 py-1 text-xs font-semibold text-base-content/45">
              {{ t('topic.hiddenReplyBadge') }}
            </div>
            <div v-if="!group.root.lastEditedAt && group.root.updatedAt && group.root.updatedAt !== group.root.createdAt" class="mt-2 text-xs font-medium text-base-content/55">
              {{ t('topic.editedAt', { time: formatDateTime(group.root.updatedAt) }) }}
            </div>
            <div v-if="group.root.lastEditedAt && group.root.lastEditor" class="mt-2 text-xs font-medium text-base-content/55">
              {{ lastEditedLabel(group.root) }}
            </div>
            <div v-if="isFirstPost(group.root) && topicActions" class="mt-4 flex flex-wrap items-center gap-2 border-t border-line pt-3">
              <button
                type="button"
                class="gf-button gf-button-sm px-2.5"
                :class="isLiked ? 'bg-error/10 text-error hover:bg-error/10' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'"
                :disabled="actingLike || isTopicRemoved()"
                @click="toggleLike"
              >
                <Heart class="h-4 w-4" :fill="isLiked ? 'currentColor' : 'none'" />
                {{ likeCount ? formatNumber(likeCount) : t('topic.like') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-sm px-2.5"
                :class="isBookmarked ? 'bg-info/10 text-primary hover:bg-info/10' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'"
                :disabled="actingBookmark || isTopicRemoved()"
                @click="toggleBookmark"
              >
                <Bookmark class="h-4 w-4" :fill="isBookmarked ? 'currentColor' : 'none'" />
                {{ isBookmarked ? t('topic.bookmarked') : t('topic.bookmark') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-sm px-2.5"
                :class="isWatched ? 'bg-success/10 text-success hover:bg-success/15' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'"
                :disabled="actingWatch || isTopicRemoved()"
                @click="toggleWatch"
              >
                <Bell class="h-4 w-4" :fill="isWatched ? 'currentColor' : 'none'" />
                {{ isWatched ? t('topic.watched') : t('topic.watch') }}
              </button>
              <button
                v-if="group.root.revisionCount > 1"
                type="button"
                class="gf-button gf-button-sm px-2.5 text-base-content/55 hover:bg-base-200 hover:text-base-content"
                @click="openPostHistory(group.root)"
              >
                <History class="h-4 w-4" />
                {{ t('topic.editHistory') }}
              </button>
              <button
                v-if="!topicActions.isOwnTopic && !isTopicRemoved()"
                type="button"
                class="gf-button gf-button-sm px-2.5 text-base-content/55 hover:bg-warning/10 hover:text-warning"
                @click="requestTopicReport"
              >
                <Flag class="h-4 w-4" />
                {{ t('topic.report') }}
              </button>
              <button
                v-if="topicActions.isOwnTopic && !isTopicRemoved()"
                type="button"
                class="gf-button gf-button-sm px-2.5 text-base-content/55 hover:bg-error/10 hover:text-error"
                @click="requestDeleteTopic"
              >
                <Trash2 class="h-4 w-4" />
                {{ t('topic.deleteTopic') }}
              </button>
              <button
                v-if="topicActions.canModerateTopic && topicProcessStatus === 0"
                type="button"
                class="gf-button gf-button-sm px-2.5 text-base-content/55 hover:bg-base-200 hover:text-base-content"
                :disabled="actingModeration"
                @click="requestTopicModeration('ban')"
              >
                <Ban class="h-4 w-4" />
                {{ t('topic.moderationBan') }}
              </button>
              <button
                v-else-if="topicActions.canModerateTopic && topicProcessStatus === 1"
                type="button"
                class="gf-button gf-button-sm px-2.5 text-base-content/55 hover:bg-base-200 hover:text-base-content"
                :disabled="actingModeration"
                @click="requestTopicModeration('unban')"
              >
                <RotateCcw class="h-4 w-4" />
                {{ t('topic.moderationUnban') }}
              </button>
              <span v-if="actionMessage" class="text-xs" :class="actionMessageSuccess ? 'text-base-content/75' : 'text-error'">{{ actionMessage }}</span>
            </div>

            <div v-if="group.replies.length" class="mt-4 space-y-2 border-t border-line pt-3">
              <div
                v-for="(reply, replyIndex) in visibleReplies(group)"
                :id="`post-${reply.id}`"
                :key="reply.id"
                :data-post-no="reply.postNo"
                class="relative rounded-lg border border-line/80 bg-base-200/40 px-3 py-2.5 transition-colors sm:px-4"
                :class="{
                  'bg-info/10': highlightedPostId === reply.id,
                  'border-t border-line/80': replyIndex > 0,
                }"
              >
                <PostReplyReference v-if="reply.replyToPostId && reply.replyToPostId !== group.root.id" :target="replyTargetFor(reply)" />
                <div class="flex min-w-0 items-center gap-2">
                  <a :href="`/u/${reply.author.id}`" class="shrink-0" @click="showUserCard(reply.author, $event)">
                    <UserAvatar :src="reply.author.avatarUrl" :alt="reply.author.username" :badge="reply.author.wornBadge" class="h-6 w-6 rounded-full ring-1 ring-line" img-class="rounded-full" />
                  </a>
                  <a :href="`/u/${reply.author.id}`" class="min-w-0 truncate text-sm font-semibold text-base-content hover:text-primary" @click="showUserCard(reply.author, $event)">{{ authorDisplayName(reply.author) }}</a>
                  <span class="shrink-0 text-xs font-semibold tabular-nums text-base-content/55">#{{ formatNumber(reply.postNo) }}</span>
                  <time class="ml-auto shrink-0 truncate text-xs text-base-content/55">{{ formatDateTime(reply.createdAt) }}</time>
                </div>
                <div v-if="reply.isAuthorDeleted" class="mt-2 rounded border border-dashed border-line bg-base-100 px-3 py-2.5 text-sm text-base-content/55">
                  <div class="font-semibold text-base-content/70">{{ t('topic.authorDeletedTitle') }}</div>
                  <div class="mt-1 leading-6">{{ t('topic.authorDeletedPlaceholder') }}</div>
                </div>
                <div v-else-if="reply.isModeratorRemoved" class="mt-2 rounded border border-dashed border-line bg-base-100 px-3 py-2.5 text-sm text-base-content/55">
                  <div class="font-semibold text-base-content/70">{{ t('topic.moderatorRemovedTitle') }}</div>
                  <div class="mt-1 leading-6">{{ t('topic.moderatorRemovedPlaceholder') }}</div>
                </div>
                <div v-else-if="reply.isHidden && !reply.canModerate" class="mt-2 rounded border border-line bg-base-100 px-3 py-2 text-sm text-base-content/45">
                  {{ t('topic.hiddenReplyPlaceholder') }}
                </div>
                <div v-else v-code-copy v-code-highlight v-math-render class="gf-prose gf-prose-post mt-2" v-html="reply.renderedContent" />
                <div v-if="reply.isHidden && !isPostRemoved(reply) && reply.canModerate" class="mt-2 inline-flex rounded bg-base-200 px-2 py-1 text-xs font-semibold text-base-content/45">
                  {{ t('topic.hiddenReplyBadge') }}
                </div>
                <div v-if="!reply.lastEditedAt && reply.updatedAt && reply.updatedAt !== reply.createdAt" class="mt-2 text-xs font-medium text-base-content/55">
                  {{ t('topic.editedAt', { time: formatDateTime(reply.updatedAt) }) }}
                </div>
                <div v-if="reply.lastEditedAt && reply.lastEditor" class="mt-2 text-xs font-medium text-base-content/55">
                  {{ lastEditedLabel(reply) }}
                </div>
                <div class="mt-2 flex items-center gap-1">
                  <button
                    v-if="(!viewer.isAuthenticated || canPost) && !reply.isHidden && !isPostRemoved(reply)"
                    type="button"
                    class="inline-flex h-7 items-center gap-1 rounded px-1.5 text-xs font-semibold text-base-content/55 transition hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                    :title="t('topic.reply')"
                    @click="replyTo(reply)"
                  >
                    <CornerDownLeft class="h-3.5 w-3.5" />
                    {{ t('topic.reply') }}
                  </button>
                  <button
                    v-if="canEditPost(reply)"
                    type="button"
                    class="gf-icon-button h-7 w-7 shrink-0 hover:bg-info/10 hover:text-primary"
                    :title="t('common.edit')"
                    @click="startEditPost(reply)"
                  >
                    <PencilLine class="h-3.5 w-3.5" />
                    <span class="sr-only">{{ t('common.edit') }}</span>
                  </button>
                  <button
                    v-if="viewer.isAuthenticated && !reply.isHidden && !isPostRemoved(reply)"
                    type="button"
                    class="inline-flex h-7 shrink-0 items-center gap-1 rounded px-1.5 text-base-content/55 transition hover:bg-error/10 hover:text-error disabled:cursor-not-allowed disabled:opacity-50"
                    :class="{ 'text-error hover:text-error': postActionState(reply).isLiked }"
                    :title="t('topic.like')"
                    :disabled="postActionState(reply).actingLike"
                    @click="togglePostLike(reply)"
                  >
                    <Heart class="h-3.5 w-3.5" :fill="postActionState(reply).isLiked ? 'currentColor' : 'none'" />
                    <span v-if="postActionState(reply).likeCount" class="text-xs font-semibold tabular-nums">{{ formatNumber(postActionState(reply).likeCount) }}</span>
                    <span class="sr-only">{{ t('topic.like') }}</span>
                  </button>
                  <button
                    v-if="viewer.isAuthenticated && !reply.isHidden && !isPostRemoved(reply)"
                    type="button"
                    class="gf-icon-button h-7 w-7 shrink-0 hover:bg-info/10 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50"
                    :class="{ 'text-primary hover:text-primary': postActionState(reply).isBookmarked }"
                    :title="postActionState(reply).isBookmarked ? t('topic.bookmarked') : t('topic.bookmark')"
                    :disabled="postActionState(reply).actingBookmark"
                    @click="togglePostBookmark(reply)"
                  >
                    <Bookmark class="h-3.5 w-3.5" :fill="postActionState(reply).isBookmarked ? 'currentColor' : 'none'" />
                    <span class="sr-only">{{ t('topic.bookmark') }}</span>
                  </button>
                  <button
                    type="button"
                    class="gf-icon-button h-7 w-7 shrink-0 hover:bg-base-200 hover:text-base-content"
                    :title="t('topic.share')"
                    @click="sharePost(reply)"
                  >
                    <Share2 class="h-3.5 w-3.5" />
                    <span class="sr-only">{{ t('topic.share') }}</span>
                  </button>
                  <button
                    v-if="canDeleteRenderedPost(reply)"
                    type="button"
                    class="gf-icon-button h-7 w-7 shrink-0 hover:bg-error/10 hover:text-error"
                    :title="t('topic.delete')"
                    @click="requestDeletePost(reply)"
                  >
                    <Trash2 class="h-3.5 w-3.5" />
                    <span class="sr-only">{{ t('topic.delete') }}</span>
                  </button>
                  <button
                    v-if="!isFirstPost(reply) && !reply.isOwnPost && !reply.isHidden && !isPostRemoved(reply)"
                    type="button"
                    class="gf-icon-button h-7 w-7 shrink-0 hover:bg-warning/10 hover:text-warning"
                    :title="t('topic.report')"
                    @click="requestPostReport(reply)"
                  >
                    <Flag class="h-3.5 w-3.5" />
                    <span class="sr-only">{{ t('topic.report') }}</span>
                  </button>
                </div>
              </div>
              <button
                v-if="group.replies.length > nestedRepliesPreviewCount"
                type="button"
                class="inline-flex h-8 w-full items-center justify-center gap-1 rounded-md text-xs font-semibold text-primary transition hover:bg-info/10"
                :aria-expanded="groupRepliesExpanded(group)"
                @click="toggleGroupReplies(group.root.id)"
              >
                <ChevronUp v-if="groupRepliesExpanded(group)" class="h-3.5 w-3.5" />
                <ChevronDown v-else class="h-3.5 w-3.5" />
                {{ groupRepliesExpanded(group) ? t('topic.collapseReplies') : t('topic.expandReplies', { count: group.replies.length - nestedRepliesPreviewCount }) }}
              </button>
            </div>
          </div>
        </article>

        <!-- Q&A Answers Section -->
        <div v-if="isQuestionTopic && answerGroups.length > 0" class="border-t border-line px-4 py-5 xl:border-t-transparent">
          <div class="mb-4 flex items-center gap-2">
            <h3 class="text-base font-semibold text-base-content">{{ t('topic.answers') }}</h3>
            <span class="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-semibold text-primary">{{ answerGroups.length }}</span>
          </div>
          <div class="space-y-4">
            <article
              v-for="(group, answerIndex) in answerGroups"
              :id="`post-${group.root.id}`"
              :key="group.root.id"
              :data-post-no="group.root.postNo"
              class="group relative rounded-lg border border-primary/30 bg-primary/5 p-4 transition-[background-color] hover:border-primary/50 hover:bg-primary/10"
              :class="{ 'bg-info/10': highlightedPostId === group.root.id }"
            >
              <div class="mb-3 flex min-w-0 items-start justify-between gap-2">
                <div class="min-w-0">
                  <div class="flex min-w-0 flex-wrap items-center gap-2">
                    <a :href="`/u/${group.root.author.id}`" class="min-w-0 flex items-center gap-2" @click="showUserCard(group.root.author, $event)">
                      <UserAvatar :src="group.root.author.avatarUrl" :alt="group.root.author.username" :badge="group.root.author.wornBadge" class="h-6 w-6 rounded-full ring-1 ring-line" img-class="rounded-full" />
                      <span class="min-w-0 truncate text-sm font-semibold text-base-content hover:text-primary">{{ authorDisplayName(group.root.author) }}</span>
                    </a>
                    <span class="shrink-0 text-xs font-semibold tabular-nums text-base-content/55">#{{ formatNumber(group.root.postNo) }}</span>
                    <time class="shrink-0 text-xs text-base-content/55">{{ formatDateTime(group.root.createdAt) }}</time>
                    <span class="shrink-0 rounded bg-success/20 px-1.5 py-0.5 text-[11px] font-semibold text-success">{{ t('topic.answer') }}</span>
                  </div>
                </div>
                <div class="flex shrink-0 items-center gap-1">
                  <button
                    v-if="canEditPost(group.root)"
                    type="button"
                    class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-icon-muted transition hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                    :disabled="savingEditPostId === group.root.id || deletingPostId === group.root.id"
                    :title="t('common.edit')"
                    @click="startEditPost(group.root)"
                  >
                    <PencilLine class="h-3.5 w-3.5" />
                    <span class="sr-only">{{ t('common.edit') }}</span>
                  </button>
                  <button
                    v-if="canDeleteRenderedPost(group.root)"
                    type="button"
                    class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                    :disabled="deletingPostId === group.root.id"
                    :title="deletingPostId === group.root.id ? t('topic.deleting') : t('topic.delete')"
                    @click="requestDeletePost(group.root)"
                  >
                    <Trash2 class="h-3.5 w-3.5" />
                    <span class="sr-only">{{ deletingPostId === group.root.id ? t('topic.deleting') : t('topic.delete') }}</span>
                  </button>
                  <button
                    v-if="(!viewer.isAuthenticated || canPost) && !group.root.isHidden && !isPostRemoved(group.root)"
                    type="button"
                    class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-icon-muted transition hover:bg-info/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 sm:h-8 sm:w-8"
                    :title="t('topic.reply')"
                    @click="replyTo(group.root)"
                  >
                    <CornerDownLeft class="h-3.5 w-3.5" />
                    <span class="sr-only">{{ t('topic.reply') }}</span>
                  </button>
                  <button
                    v-if="viewer.isAuthenticated && !group.root.isHidden && !isPostRemoved(group.root)"
                    type="button"
                    class="inline-flex h-7 shrink-0 items-center gap-1 rounded-md px-1 text-icon-muted transition hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 sm:h-8 sm:px-1.5"
                    :class="{ 'text-error hover:text-error': postActionState(group.root).isLiked }"
                    :title="t('topic.like')"
                    :disabled="postActionState(group.root).actingLike"
                    @click="togglePostLike(group.root)"
                  >
                    <Heart class="h-3.5 w-3.5" :fill="postActionState(group.root).isLiked ? 'currentColor' : 'none'" />
                    <span v-if="postActionState(group.root).likeCount" class="hidden text-xs font-semibold tabular-nums sm:inline">{{ formatNumber(postActionState(group.root).likeCount) }}</span>
                    <span class="sr-only">{{ t('topic.like') }}</span>
                  </button>
                  <button
                    type="button"
                    class="gf-icon-button h-7 w-7 shrink-0 sm:h-8 sm:w-8 hover:bg-base-200 hover:text-base-content focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
                    :title="t('topic.share')"
                    @click="sharePost(group.root)"
                  >
                    <Share2 class="h-3.5 w-3.5" />
                    <span class="sr-only">{{ t('topic.share') }}</span>
                  </button>
                </div>
              </div>
              <PostReplyReference v-if="group.root.replyToPostId" :target="replyTargetFor(group.root)" />
              <div v-if="group.root.isAuthorDeleted" class="rounded border border-dashed border-line bg-base-200/60 px-3 py-3 text-sm text-base-content/55">
                <div class="font-semibold text-base-content/70">{{ t('topic.authorDeletedTitle') }}</div>
                <div class="mt-1 leading-6">{{ t('topic.authorDeletedPlaceholder') }}</div>
              </div>
              <div v-else-if="group.root.isModeratorRemoved" class="rounded border border-dashed border-line bg-base-200/60 px-3 py-3 text-sm text-base-content/55">
                <div class="font-semibold text-base-content/70">{{ t('topic.moderatorRemovedTitle') }}</div>
                <div class="mt-1 leading-6">{{ t('topic.moderatorRemovedPlaceholder') }}</div>
              </div>
              <div v-else-if="group.root.isHidden && !group.root.canModerate" class="rounded border border-line bg-base-200/60 px-3 py-2 text-sm text-base-content/45">
                {{ t('topic.hiddenReplyPlaceholder') }}
              </div>
              <div v-else v-code-copy v-code-highlight v-math-render class="gf-prose gf-prose-post" :class="{ 'gf-prose-article': isBlogLikeTopic && isFirstPost(group.root), 'gf-prose-thought': props.contentType === 2 && isFirstPost(group.root) }" v-html="group.root.renderedContent" />
              <div v-if="!group.root.lastEditedAt && group.root.updatedAt && group.root.updatedAt !== group.root.createdAt" class="mt-2 text-xs font-medium text-base-content/55">
                {{ t('topic.editedAt', { time: formatDateTime(group.root.updatedAt) }) }}
              </div>
              <div v-if="group.root.lastEditedAt && group.root.lastEditor" class="mt-2 text-xs font-medium text-base-content/55">
                {{ lastEditedLabel(group.root) }}
              </div>
            </article>
          </div>
        </div>

        <div v-if="(postHasAfter || loadingPostDirection === 'after' || postWindowError || (!postHasAfter && posts.length)) && !isBlogLikeTopic" ref="postLoadMoreEl" class="relative border-t border-line px-4 py-3 text-center xl:border-t-transparent">
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

      <aside v-if="hasAside && !isBlogLikeTopic" class="hidden min-w-0 xl:block">
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
    v-if="topicActions && !isBlogLikeTopic"
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

  <!-- Reply composer - hidden for Articles and Thoughts -->
  <PostComposer
    v-if="composerMounted && !isBlogLikeTopic"
    v-model="postContent"
    v-model:captcha-code="captchaCode"
    :open="composerOpen"
    :authenticated="viewer.isAuthenticated"
    :captcha-img="captchaImg"
    :captcha-loading="captchaLoading"
    :captcha-required="captchaRequired"
    :error-message="errorMessage"
    :mode="composerMode"
    :submitting="editingPostId ? savingEditPostId > 0 : submitting"
    :success-message="successMessage"
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
                  :src="pendingDeletePost.author.avatarUrl"
                  :alt="pendingDeletePost.author.username"
                  class="h-6 w-6 shrink-0 rounded-full object-cover ring-1 ring-line"
                />
                <div class="min-w-0 truncate text-xs font-semibold text-base-content/55">
                  @{{ pendingDeletePost.author.username }}
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
                  <div v-if="version.content" v-code-copy v-code-highlight v-math-render class="gf-prose gf-prose-post mt-2 border-t border-line/70 pt-2" v-html="version.renderedHTML" />
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
  max-width: 100%;
  height: auto;
  border-radius: 0.375rem;
}
</style>
