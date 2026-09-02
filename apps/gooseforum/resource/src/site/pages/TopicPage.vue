<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Clock, Eye, Heart, MessageSquare } from '@lucide/vue'
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
const topicHeaderEl = ref<HTMLElement | null>(null)
const titleEl = ref<HTMLElement | null>(null)
const showHeaderTitle = ref(false)
const isMobileHeaderViewport = ref(false)
const mobileHeaderTitleVisible = ref(false)
const effectiveShowHeaderTitle = computed(() => showHeaderTitle.value && (!isMobileHeaderViewport.value || mobileHeaderTitleVisible.value))
let titleObserver: IntersectionObserver | undefined
let lastHeaderScrollY = 0
let headerScrollFrame = 0

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
})

function handleTopicState(nextLikeCount: number) {
  likeCount.value = nextLikeCount
}

</script>

<template>
  <div class="min-w-0">
    <div class="min-w-0">
      <header ref="topicHeaderEl" class="relative z-10 border-b border-line/70 px-4 py-4 sm:mb-4 sm:px-0 sm:pb-4 sm:pt-0 xl:w-[calc(100%+292px)]">
        <h1 ref="titleEl" class="break-words text-2xl font-bold leading-tight text-base-content [overflow-wrap:anywhere] sm:text-3xl">{{ page.props.topic.title }}</h1>
        <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-[13px] text-base-content/55">
          <a
            :href="`/u/${page.props.topic.author.id}`"
            class="inline-flex items-center gap-2 font-medium text-base-content/75 hover:text-primary"
            @click="showUserCard(page.props.topic.author, $event)"
          >
            <UserAvatar :src="page.props.topic.author.avatarUrl" :alt="page.props.topic.author.username" class="h-5 w-5 rounded-full object-cover" />
            {{ authorDisplayName(page.props.topic.author) }}
          </a>
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
      </header>

      <PostStream
        :topic-id="page.props.topic.id"
        :topic-title="page.props.topic.title"
        :content-type="page.props.topic.contentType"
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
  </div>
</template>
