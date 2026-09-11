<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bookmark, Heart, MessageSquare } from '@lucide/vue'
import { bookmarkTopic, likeTopic } from '@/runtime/api'
import type { TopicPayload } from '@gooseforum/client'

// 卡片级快捷互动（issue #380）：初始态来自列表 payload 的观察者字段；
// liked/bookmarked 缺席（访客或状态不可用）时点击引导登录，不假设 false。
// 点赞乐观更新 ±1，失败回滚；语义与详情页 PostStream 一致（action 1/2 幂等切换）。
const props = defineProps<{ topic: TopicPayload }>()

const { t } = useI18n()

const liked = ref(props.topic.liked)
const bookmarked = ref(props.topic.bookmarked)
const likeCount = ref(props.topic.likeCount)
const actingLike = ref(false)
const actingBookmark = ref(false)

// 就地刷新（排序切换/静默重载）按 topic.id 复用实例：非操作中跟随服务端 payload，
// 避免卡片状态过期；操作进行中跳过，完成后由下一次刷新收敛（对齐移动端 override 语义）。
watch(
  () => props.topic,
  (next) => {
    if (actingLike.value || actingBookmark.value) return
    liked.value = next.liked
    bookmarked.value = next.bookmarked
    likeCount.value = next.likeCount
  },
)

const commentUrl = `${props.topic.url}?reply=1`

function jumpToLogin() {
  const current = `${window.location.pathname}${window.location.search}${window.location.hash}`
  window.location.href = `/login?redirect=${encodeURIComponent(current)}`
}

async function toggleLike() {
  if (actingLike.value) return
  if (liked.value === undefined) {
    jumpToLogin()
    return
  }
  const previousLiked = liked.value
  const previousCount = likeCount.value
  const nextLiked = !previousLiked
  actingLike.value = true
  liked.value = nextLiked
  likeCount.value = Math.max(0, previousCount + (nextLiked ? 1 : -1))
  try {
    await likeTopic(props.topic.id, nextLiked ? 1 : 2)
  } catch {
    liked.value = previousLiked
    likeCount.value = previousCount
  } finally {
    actingLike.value = false
  }
}

async function toggleBookmark() {
  if (actingBookmark.value) return
  if (bookmarked.value === undefined) {
    jumpToLogin()
    return
  }
  const previousBookmarked = bookmarked.value
  const nextBookmarked = !previousBookmarked
  actingBookmark.value = true
  bookmarked.value = nextBookmarked
  try {
    await bookmarkTopic(props.topic.id, nextBookmarked ? 1 : 2)
  } catch {
    bookmarked.value = previousBookmarked
  } finally {
    actingBookmark.value = false
  }
}
</script>

<template>
  <div class="flex items-center gap-0.5 text-xs text-base-content/55">
    <button
      type="button"
      class="inline-flex h-7 items-center gap-1.5 rounded-md px-2 transition-colors hover:bg-base-200 hover:text-base-content disabled:cursor-default disabled:opacity-60"
      :class="liked ? 'text-error' : ''"
      :disabled="actingLike"
      :aria-pressed="liked === true"
      :title="t('topic.like')"
      @click="toggleLike"
    >
      <Heart class="h-4 w-4" :class="liked ? 'fill-current' : ''" />
      <span class="tabular-nums">{{ likeCount }}</span>
    </button>
    <button
      type="button"
      class="inline-flex h-7 items-center gap-1.5 rounded-md px-2 transition-colors hover:bg-base-200 hover:text-base-content disabled:cursor-default disabled:opacity-60"
      :class="bookmarked ? 'text-primary' : ''"
      :disabled="actingBookmark"
      :aria-pressed="bookmarked === true"
      :title="bookmarked ? t('topic.bookmarked') : t('topic.bookmark')"
      @click="toggleBookmark"
    >
      <Bookmark class="h-4 w-4" :class="bookmarked ? 'fill-current' : ''" />
    </button>
    <a
      :href="commentUrl"
      class="ml-auto inline-flex h-7 items-center gap-1.5 rounded-md px-2 transition-colors hover:bg-base-200 hover:text-base-content"
      :title="t('topic.reply')"
    >
      <MessageSquare class="h-4 w-4" />
      <span class="tabular-nums">{{ topic.replyCount }}</span>
    </a>
  </div>
</template>
