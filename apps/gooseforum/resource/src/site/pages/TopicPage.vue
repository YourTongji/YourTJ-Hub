<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { BookOpen, Clock, Eye, FileText, Heart, HelpCircle, MessageSquare, Sparkles } from '@lucide/vue'
import { formatDateTime, formatNumber } from '@/runtime/format'
import { useShellState } from '@/runtime/shell-state'
import { showUserCard } from '@/runtime/user-card-events'
import PostStream from '@/site/components/PostStream.vue'
import UserAvatar from '@/site/components/UserAvatar.vue'
import type { TopicDetailProps, LayoutPayload } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'

const page = defineProps<{
  layout: LayoutPayload
  props: TopicDetailProps
}>()

const { t } = useI18n()
const shellState = useShellState()
const likeCount = ref(page.props.topic.likeCount)
const isLiked = ref(page.props.topic.isLiked)
const isBookmarked = ref(page.props.topic.isBookmarked)
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
const isShortForm = computed(() => page.props.topic.contentType === 1 || page.props.topic.contentType === 2)

// 提取当前话题图片（仅短文类型提取，长文保持图文穿插）
const topicImages = computed(() => {
  if (!isShortForm.value) {
    return []
  }
  const list: string[] = []
  if (page.props.topic.images && page.props.topic.images.length > 0) {
    for (const url of page.props.topic.images) {
      if (url && !list.includes(url)) list.push(url)
    }
  }
  if (list.length === 0 && page.props.topic.firstImageUrl) {
    list.push(page.props.topic.firstImageUrl)
  }
  // 回退：从首楼 Markdown 与 HTML 中提取图片
  if (list.length === 0) {
    const firstPost = page.props.postStream.posts.find((p) => p.postNo === 1)
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
function authorDisplayName(author: { username: string; nickname?: string }) {
  return author.nickname || author.username
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
  () => page.props.topic.id,
  () => {
    likeCount.value = page.props.topic.likeCount
    isLiked.value = page.props.topic.isLiked
    isBookmarked.value = page.props.topic.isBookmarked
    mobileHeaderTitleVisible.value = false
    if (typeof window !== 'undefined') {
      lastHeaderScrollY = window.scrollY
    }
    void nextTick(observeTitle)
  },
  { immediate: true },
)

watch(
  () => [page.props.topic.title, page.props.topic.categories, effectiveShowHeaderTitle.value] as const,
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
      <h1 ref="titleEl" class="break-words text-2xl font-bold leading-tight text-base-content [overflow-wrap:anywhere] sm:text-3xl">
        {{ page.props.topic.title }}
      </h1>
      <!-- 桌面端元数据栏：完整横排平铺（sm 及以上屏幕） -->
      <div class="mt-3 hidden sm:flex sm:flex-wrap sm:items-center sm:gap-x-4 sm:gap-y-2 text-[13px] text-base-content/55">
        <a
          :href="`/u/${page.props.topic.author.id}`"
          class="inline-flex items-center gap-2 font-medium text-base-content/75 hover:text-primary"
          @click="showUserCard(page.props.topic.author, $event)"
        >
          <UserAvatar :src="page.props.topic.author.avatarUrl" :alt="page.props.topic.author.username" class="h-5 w-5 rounded-full object-cover" />
          {{ authorDisplayName(page.props.topic.author) }}
        </a>
        <!-- Content type badge -->
        <span v-if="page.props.topic.contentType === 1" class="inline-flex items-center gap-1.5 rounded-full bg-success/20 px-2 py-0.5 text-[12px] font-semibold text-success">
          <HelpCircle class="h-3.5 w-3.5" />
          {{ t('publish.contentTypes.question') }}
        </span>
        <span v-else-if="page.props.topic.contentType === 2" class="inline-flex items-center gap-1.5 rounded-full bg-purple-500/20 px-2 py-0.5 text-[12px] font-semibold text-purple-500">
          <Sparkles class="h-3.5 w-3.5" />
          {{ t('publish.contentTypes.thought') }}
        </span>
        <span v-else-if="page.props.topic.contentType === 3" class="inline-flex items-center gap-1.5 rounded-full bg-amber-500/15 px-2.5 py-0.5 text-[12px] font-semibold text-amber-600 dark:text-amber-400">
          <BookOpen class="h-3.5 w-3.5" />
          {{ t('publish.contentTypes.article') }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <Clock class="h-3.5 w-3.5" />
          {{ formatDateTime(page.props.topic.createdAt) }}
        </span>
        <a
          v-for="category in page.props.topic.categories"
          :key="category.id"
          :href="category.url"
          class="inline-flex items-center gap-1.5 rounded-sm text-base-content/75 hover:text-primary"
        >
          <span class="h-2 w-2 rounded-[3px]" :style="{ backgroundColor: category.color }" />
          {{ category.name }}
        </a>
        <span class="inline-flex items-center gap-1.5">
          <MessageSquare class="h-3.5 w-3.5" />
          {{ formatNumber(page.props.topic.replyCount) }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <Eye class="h-3.5 w-3.5" />
          {{ formatNumber(page.props.topic.viewCount) }}
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
              :href="`/u/${page.props.topic.author.id}`"
              class="inline-flex items-center gap-1.5 font-medium text-base-content/80 hover:text-primary truncate"
              @click="showUserCard(page.props.topic.author, $event)"
            >
              <UserAvatar :src="page.props.topic.author.avatarUrl" :alt="page.props.topic.author.username" class="h-5 w-5 rounded-full object-cover" />
              <span class="truncate">{{ authorDisplayName(page.props.topic.author) }}</span>
            </a>
            <span v-if="page.props.topic.contentType === 1" class="inline-flex items-center gap-1 rounded-full bg-success/15 px-2 py-0.5 text-[11px] font-semibold text-success">
              <HelpCircle class="h-3 w-3" />
              {{ t('publish.contentTypes.question') }}
            </span>
            <span v-else-if="page.props.topic.contentType === 2" class="inline-flex items-center gap-1 rounded-full bg-purple-500/15 px-2 py-0.5 text-[11px] font-semibold text-purple-600 dark:text-purple-400">
              <Sparkles class="h-3 w-3" />
              {{ t('publish.contentTypes.thought') }}
            </span>
            <span v-else-if="page.props.topic.contentType === 3" class="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-[11px] font-semibold text-amber-600 dark:text-amber-400">
              <BookOpen class="h-3 w-3" />
              {{ t('publish.contentTypes.article') }}
            </span>
          </div>
          <div v-if="page.props.topic.categories && page.props.topic.categories.length > 0" class="flex items-center gap-1.5 shrink-0">
            <a
              v-for="category in page.props.topic.categories"
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
            <span>{{ formatDateTime(page.props.topic.createdAt) }}</span>
          </span>
          <div class="flex items-center gap-3 shrink-0">
            <span class="inline-flex items-center gap-1">
              <MessageSquare class="h-3.5 w-3.5" />
              <span class="tabular-nums font-medium">{{ formatNumber(page.props.topic.replyCount) }}</span>
            </span>
            <span class="inline-flex items-center gap-1">
              <Eye class="h-3.5 w-3.5" />
              <span class="tabular-nums font-medium">{{ formatNumber(page.props.topic.viewCount) }}</span>
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
      :topic-id="page.props.topic.id"
      :topic-title="page.props.topic.title"
      :content-type="page.props.topic.contentType"
      :topic-images="topicImages"
      :categories="page.props.topic.categories"
      :initial-post-stream="page.props.postStream"
      :viewer="page.layout.viewer"
      :can-post="page.props.permissions.canPost"
      :hot-topics="page.props.hotTopics"
      :topic-actions="{
        likeCount: page.props.topic.likeCount,
        isLiked: page.props.topic.isLiked,
        isBookmarked: page.props.topic.isBookmarked,
        isWatched: page.props.topic.isWatched,
        processStatus: page.props.topic.processStatus,
        authorDeleted: page.props.topic.authorDeleted,
        moderatorRemoved: page.props.topic.moderatorRemoved,
        isOwnTopic: page.props.permissions.isOwnTopic,
        canModerateTopic: page.props.permissions.canModerateTopic,
        createdAt: page.props.topic.createdAt,
        updatedAt: page.props.topic.updatedAt,
        replyCount: page.props.topic.replyCount,
        viewCount: page.props.topic.viewCount,
        maxPostNo: page.props.topic.maxPostNo,
        participants: page.props.topic.participants,
        author: page.props.topic.author,
        description: page.props.topic.description,
      }"
      @topic-state="handleTopicState"
    />
  </div>
</template>
