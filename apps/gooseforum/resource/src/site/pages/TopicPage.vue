<script setup lang="ts">
import { authorURL } from '@/runtime/anonymous-identity'
import { useContentUpdates } from '@/runtime/content-updates'
import { useRouter } from 'vue-router'
import { userDisplayName } from '@/runtime/private-notes'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { BookOpen, Clock, Eye, FileText, Heart, HelpCircle, MessageSquare, Sparkles } from '@lucide/vue'
import { formatDateTime, formatNumber } from '@/runtime/format'
import { useShellState } from '@/runtime/shell-state'
import { showUserCard } from '@/runtime/user-card-events'
import { topicDisplayLabel } from '@/runtime/topic-description'
import PostStream from '@/site/components/PostStream.vue'
import UserAvatar from '@/site/components/UserAvatar.vue'
import type { TopicDetailProps, LayoutPayload } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'

const page = defineProps<{
  layout: LayoutPayload
  props: TopicDetailProps
}>()

const view = ref(page.props)
const router = useRouter()
watch(() => page.props, next => { view.value = next })
let refreshingContent = false
useContentUpdates(async () => {
  if (refreshingContent) return
  refreshingContent = true
  const topicID = view.value.topic.id
  try {
    const response = await fetch(`/p/post/${topicID}`, { headers: { 'X-Goose-Page': 'true' } })
    if (topicID !== view.value.topic.id) return
    if (response.status === 404 && view.value.topic.processStatus === 2) {
      await router.push('/settings?tab=content')
    } else if (response.ok) {
      const payload = await response.json()
      if (payload.component === 'topic.detail') view.value = payload.props
    }
  } catch { /* Reconnect/next invalidation reconciles from REST. */ }
  finally { refreshingContent = false }
}, () => !!page.layout.viewer?.isAuthenticated)

const { t } = useI18n()
const shellState = useShellState()
const likeCount = ref(view.value.topic.likeCount)
const isLiked = ref(view.value.topic.isLiked)
const isBookmarked = ref(view.value.topic.isBookmarked)
const postStreamRef = ref<InstanceType<typeof PostStream> | null>(null)

const topicHeaderEl = ref<HTMLElement | null>(null)
const titleEl = ref<HTMLElement | null>(null)
const showHeaderTitle = ref(false)
const isMobileHeaderViewport = ref(false)
const mobileHeaderTitleVisible = ref(false)
const effectiveShowHeaderTitle = computed(() => showHeaderTitle.value && (!isMobileHeaderViewport.value || mobileHeaderTitleVisible.value))
let titleObserver: IntersectionObserver | undefined
let lastHeaderScrollY = 0
let headerScrollFrame = 0

// 仅短文类型（提问 contentType: 1、瞬间 contentType: 2）使用前置图片轮播图窗；长文（讨论、文章）保持经典图文穿插
const isShortForm = computed(() => view.value.topic.contentType === 1 || view.value.topic.contentType === 2)

// 分享/删除确认等需要文案的场景：无标题瞬间回退到正文摘要，再回退到稳定颜文字，避免空文案。
const topicDisplayTitle = computed(() => topicDisplayLabel(view.value.topic.id, view.value.topic.title, view.value.topic.description))

// 提取当前话题图片（仅短文类型提取，长文保持图文穿插）
const topicImages = computed(() => {
  if (!isShortForm.value) {
    return []
  }
  const list: string[] = []
  if (view.value.topic.images && view.value.topic.images.length > 0) {
    for (const url of view.value.topic.images) {
      if (url && !list.includes(url)) list.push(url)
    }
  }
  if (list.length === 0 && view.value.topic.firstImageUrl) {
    list.push(view.value.topic.firstImageUrl)
  }
  // 回退：从首楼 Markdown 与 HTML 中提取图片
  if (list.length === 0) {
    const firstPost = view.value.postStream.posts.find((p) => p.postNo === 1)
    if (firstPost) {
      const mdRegex = /!\[.*?\]\((https?:\/\/[^\s)]+|\/[^\s)]+)\)/g
      let match: RegExpExecArray | null
      while ((match = mdRegex.exec(firstPost.content)) !== null) {
        if (match[1] && !list.includes(match[1])) list.push(match[1])
      }
      const htmlRegex = /<img[^>]+src=["']([^"']+)["']/g
      while ((match = htmlRegex.exec(firstPost.renderedContent)) !== null) {
        if (match[1] && !list.includes(match[1])) list.push(match[1])
      }
    }
  }
  return list
})

// 优先展示用户昵称，未设置昵称时回退到账号名
function authorDisplayName(author: { id?: number; username: string; nickname?: string }) {
  return userDisplayName(author.id, author.username, author.nickname)
}

function observeTitle() {
  titleObserver?.disconnect()
  showHeaderTitle.value = false

  if (!titleEl.value || !('IntersectionObserver' in window)) return

  titleObserver = new IntersectionObserver(
    (entries) => {
      showHeaderTitle.value = !entries[0]?.isIntersecting
    },
    { threshold: 0, rootMargin: '-80px 0px 0px 0px' },
  )
  titleObserver.observe(titleEl.value)
}

function updateHeaderViewport() {
  const wasMobile = isMobileHeaderViewport.value
  const isMobile = window.innerWidth < 768
  isMobileHeaderViewport.value = isMobile
  if (isMobile && !wasMobile) {
    mobileHeaderTitleVisible.value = false
    return
  }
  if (!isMobile) {
    mobileHeaderTitleVisible.value = true
  }
}

function updateMobileHeaderTitle() {
  if (headerScrollFrame) return
  headerScrollFrame = window.requestAnimationFrame(applyMobileHeaderTitle)
}

function applyMobileHeaderTitle() {
  headerScrollFrame = 0
  const scrollY = window.scrollY
  const delta = scrollY - lastHeaderScrollY
  if (Math.abs(delta) < 4) {
    return
  }

  if (isMobileHeaderViewport.value) {
    mobileHeaderTitleVisible.value = delta > 0
  }
  lastHeaderScrollY = scrollY
}

function setupHeaderTitleBehavior() {
  lastHeaderScrollY = window.scrollY
  updateHeaderViewport()
  window.addEventListener('scroll', updateMobileHeaderTitle, { passive: true })
  window.addEventListener('resize', updateHeaderViewport)
}

onMounted(() => {
  setupHeaderTitleBehavior()
  void nextTick(observeTitle)
})

watch(
  () => view.value.topic.id,
  () => {
    likeCount.value = view.value.topic.likeCount
    isLiked.value = view.value.topic.isLiked
    isBookmarked.value = view.value.topic.isBookmarked
    mobileHeaderTitleVisible.value = false
    if (typeof window !== 'undefined') {
      lastHeaderScrollY = window.scrollY
    }
    void nextTick(observeTitle)
  },
  { immediate: true },
)

watch(
  () => [view.value.topic.title, view.value.topic.categories, effectiveShowHeaderTitle.value] as const,
  ([title, categories, show]) => {
    shellState.headerTitle = title
    shellState.headerTags = categories.map((category) => ({
      id: category.id,
      name: category.name,
      color: category.color,
    }))
    shellState.showHeaderTitle = show
    shellState.isTopicPage = true
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  titleObserver?.disconnect()
  window.removeEventListener('scroll', updateMobileHeaderTitle)
  window.removeEventListener('resize', updateHeaderViewport)
  window.cancelAnimationFrame(headerScrollFrame)
  shellState.headerTitle = ''
  shellState.headerTags = []
  shellState.showHeaderTitle = false
  shellState.isTopicPage = false
})

function handleTopicState(nextLikeCount: number) {
  likeCount.value = nextLikeCount
}
</script>

<template>
  <div class="min-w-0">
    <header ref="topicHeaderEl" class="relative z-10 border-b border-line/70 px-4 py-4 sm:mb-4 sm:px-0 sm:pb-4 sm:pt-0 xl:w-[calc(100%+292px)]">
      <!-- 待审话题（issue #975）：只有作者与审核员能打开，说明谁能看到、何时公开 -->
      <p
        v-if="view.topic.processStatus === 2"
        role="status"
        data-test="topic-pending-review"
        class="mb-3 flex items-start gap-2 rounded-lg border border-warning/20 bg-warning/10 px-3 py-2 text-sm leading-5 text-warning"
      >
        <Clock class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
        <span>{{ view.permissions.isOwnTopic ? t('topic.pendingReviewBanner') : t('topic.pendingReviewBannerModerator') }}</span>
      </p>
      <!-- 无标题瞬间不渲染大标题（未填写标题的瞬间标题恒为空） -->
      <h1 v-if="view.topic.title" ref="titleEl" class="break-words text-2xl font-bold leading-tight text-base-content [overflow-wrap:anywhere] sm:text-3xl">
        {{ view.topic.title }}
      </h1>
      <!-- 桌面端元数据栏：完整横排平铺（sm 及以上屏幕） -->
      <div class="mt-3 hidden sm:flex sm:flex-wrap sm:items-center sm:gap-x-4 sm:gap-y-2 text-[13px] text-base-content/55">
        <a
          :href="authorURL(view.topic.author)"
          class="inline-flex items-center gap-2 font-medium text-base-content/75 hover:text-primary"
          @click="showUserCard(view.topic.author, $event)"
        >
          <UserAvatar :src="view.topic.author.avatarUrl" :alt="view.topic.author.username" class="h-5 w-5 rounded-full object-cover" />
          {{ authorDisplayName(view.topic.author) }}
        </a>
        <!-- Content type badge -->
        <span v-if="view.topic.contentType === 1" class="inline-flex items-center gap-1.5 rounded-full bg-success/20 px-2 py-0.5 text-[12px] font-semibold text-success">
          <HelpCircle class="h-3.5 w-3.5" />
          {{ t('publish.contentTypes.question') }}
        </span>
        <span v-else-if="view.topic.contentType === 2" class="inline-flex items-center gap-1.5 rounded-full bg-purple-500/20 px-2 py-0.5 text-[12px] font-semibold text-purple-500">
          <Sparkles class="h-3.5 w-3.5" />
          {{ t('publish.contentTypes.thought') }}
        </span>
        <span v-else-if="view.topic.contentType === 3" class="inline-flex items-center gap-1.5 rounded-full bg-amber-500/15 px-2.5 py-0.5 text-[12px] font-semibold text-amber-600 dark:text-amber-400">
          <BookOpen class="h-3.5 w-3.5" />
          {{ t('publish.contentTypes.article') }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <Clock class="h-3.5 w-3.5" />
          {{ formatDateTime(view.topic.createdAt) }}
        </span>
        <a
          v-for="category in view.topic.categories"
          :key="category.id"
          :href="category.url"
          class="inline-flex items-center gap-1.5 rounded-sm text-base-content/75 hover:text-primary"
        >
          <span class="h-2 w-2 rounded-[3px]" :style="{ backgroundColor: category.color }" />
          {{ category.name }}
        </a>
        <span class="inline-flex items-center gap-1.5">
          <MessageSquare class="h-3.5 w-3.5" />
          {{ formatNumber(view.topic.replyCount) }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <Eye class="h-3.5 w-3.5" />
          {{ formatNumber(view.topic.viewCount) }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <Heart class="h-3.5 w-3.5" />
          {{ formatNumber(likeCount) }}
        </span>
      </div>

      <!-- 移动端元数据栏：两行优雅规划，时间与分区合理归位，计数项同排完整展示（<sm 屏幕） -->
      <div class="mt-2.5 flex flex-col gap-2 text-[13px] text-base-content/55 sm:hidden">
        <!-- 第 1 行：作者、内容类型、右侧分区标签 -->
        <div class="flex items-center justify-between gap-2 min-w-0">
          <div class="flex items-center gap-2 min-w-0 flex-wrap">
            <a
              :href="authorURL(view.topic.author)"
              class="inline-flex items-center gap-1.5 font-medium text-base-content/80 hover:text-primary truncate"
              @click="showUserCard(view.topic.author, $event)"
            >
              <UserAvatar :src="view.topic.author.avatarUrl" :alt="view.topic.author.username" class="h-5 w-5 rounded-full object-cover" />
              <span class="truncate">{{ authorDisplayName(view.topic.author) }}</span>
            </a>
            <span v-if="view.topic.contentType === 1" class="inline-flex items-center gap-1 rounded-full bg-success/15 px-2 py-0.5 text-[11px] font-semibold text-success">
              <HelpCircle class="h-3 w-3" />
              {{ t('publish.contentTypes.question') }}
            </span>
            <span v-else-if="view.topic.contentType === 2" class="inline-flex items-center gap-1 rounded-full bg-purple-500/15 px-2 py-0.5 text-[11px] font-semibold text-purple-600 dark:text-purple-400">
              <Sparkles class="h-3 w-3" />
              {{ t('publish.contentTypes.thought') }}
            </span>
            <span v-else-if="view.topic.contentType === 3" class="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-[11px] font-semibold text-amber-600 dark:text-amber-400">
              <BookOpen class="h-3 w-3" />
              {{ t('publish.contentTypes.article') }}
            </span>
          </div>
          <div v-if="view.topic.categories && view.topic.categories.length > 0" class="flex items-center gap-1.5 shrink-0">
            <a
              v-for="category in view.topic.categories"
              :key="category.id"
              :href="category.url"
              class="inline-flex items-center gap-1 text-xs font-medium text-base-content/75 hover:text-primary"
            >
              <span class="h-2 w-2 rounded-[3px]" :style="{ backgroundColor: category.color }" />
              <span>{{ category.name }}</span>
            </a>
          </div>
        </div>

        <!-- 第 2 行：左侧时间，右侧三项计数整齐排在同一行，绝不折行 -->
        <div class="flex items-center justify-between gap-2 text-xs text-base-content/55">
          <span class="inline-flex items-center gap-1 text-xs shrink-0">
            <Clock class="h-3.5 w-3.5" />
            <span>{{ formatDateTime(view.topic.createdAt) }}</span>
          </span>
          <div class="flex items-center gap-3 shrink-0">
            <span class="inline-flex items-center gap-1">
              <MessageSquare class="h-3.5 w-3.5" />
              <span class="tabular-nums font-medium">{{ formatNumber(view.topic.replyCount) }}</span>
            </span>
            <span class="inline-flex items-center gap-1">
              <Eye class="h-3.5 w-3.5" />
              <span class="tabular-nums font-medium">{{ formatNumber(view.topic.viewCount) }}</span>
            </span>
            <span class="inline-flex items-center gap-1">
              <Heart class="h-3.5 w-3.5" />
              <span class="tabular-nums font-medium">{{ formatNumber(likeCount) }}</span>
            </span>
          </div>
        </div>
      </div>
    </header>

    <PostStream
      ref="postStreamRef"
      :topic-id="view.topic.id"
      :topic-title="topicDisplayTitle"
      :topic-edit-title="view.topic.title"
      :content-type="view.topic.contentType"
      :topic-images="topicImages"
      :categories="view.topic.categories"
      :initial-post-stream="view.postStream"
      :viewer="page.layout.viewer"
      :can-post="view.permissions.canPost"
      :hot-topics="view.hotTopics"
      :topic-actions="{
        likeCount: view.topic.likeCount,
        isLiked: view.topic.isLiked,
        isBookmarked: view.topic.isBookmarked,
        isWatched: view.topic.isWatched,
        processStatus: view.topic.processStatus,
        authorDeleted: view.topic.authorDeleted,
        moderatorRemoved: view.topic.moderatorRemoved,
        isOwnTopic: view.permissions.isOwnTopic,
        canModerateTopic: view.permissions.canModerateTopic,
        createdAt: view.topic.createdAt,
        updatedAt: view.topic.updatedAt,
        replyCount: view.topic.replyCount,
        viewCount: view.topic.viewCount,
        maxPostNo: view.topic.maxPostNo,
        participants: view.topic.participants,
        author: view.topic.author,
        description: view.topic.description,
      }"
      @topic-state="handleTopicState"
    />
  </div>
</template>
