<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown, ChevronRight, Clock, Eye, MessageSquare } from '@lucide/vue'
import { formatDateTime, formatNumber } from '@/runtime/format'
import { showUserCard } from '@/runtime/user-card-events'
import { consumeWikiJumpState } from '@/runtime/use-wiki-search'
import MarkdownImageViewer from '@/site/components/MarkdownImageViewer.vue'
import PostStream from '@/site/components/PostStream.vue'
import UserAvatar from '@/site/components/UserAvatar.vue'
import WikiPageActions from '@/site/components/WikiPageActions.vue'
import WikiToc from '@/site/components/WikiToc.vue'
import type { LayoutPayload, WikiDetailProps } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: WikiDetailProps
}>()


const { t } = useI18n()
const markdownImageViewer = ref<InstanceType<typeof MarkdownImageViewer> | null>(null)
const mobileTocOpen = ref(false)

// ---- wiki 局内搜索命中定位（段落级锚点，方案 B）----
// hitAnchors 为搜索面板跳转带入的命中段落集合；⌘G/Ctrl+G 连续定位下一处。
// 直接 URL hash（#s-<n> 或标题锚点）在挂载后定位并短暂高亮。
const hitAnchors = ref<string[]>([])
const hitCursor = ref(0)

function flashElement(element: HTMLElement) {
  element.classList.remove('wiki-hit-flash')
  void element.offsetWidth // 强制 reflow，重触发 CSS 动画
  element.classList.add('wiki-hit-flash')
}

function scrollToAnchor(anchor: string) {
  const element = document.getElementById(anchor)
  if (!(element instanceof HTMLElement)) return
  const top = element.getBoundingClientRect().top + window.scrollY - 88
  window.scrollTo({ top: Math.max(0, top), behavior: 'smooth' })
  // 平滑滚动是异步的：等滚动结束（scrollend，fallback 定时器）再触发高亮动画，
  // 避免动画在滚动途中就淡出。
  let flashed = false
  const finishFlash = () => {
    if (flashed) return
    flashed = true
    if (document.contains(element)) flashElement(element)
  }
  window.addEventListener('scrollend', finishFlash, { once: true })
  window.setTimeout(finishFlash, 700)
}

function jumpToNextHit() {
  if (hitAnchors.value.length === 0) return
  hitCursor.value = (hitCursor.value + 1) % hitAnchors.value.length
  scrollToAnchor(hitAnchors.value[hitCursor.value])
}

function handleHitKeydown(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'g') {
    event.preventDefault()
    jumpToNextHit()
  }
}

onMounted(() => {
  const hash = window.location.hash.slice(1)
  if (hash) {
    requestAnimationFrame(() => scrollToAnchor(decodeURIComponent(hash)))
  }
  const jump = consumeWikiJumpState()
  if (jump && jump.anchors.length) {
    hitAnchors.value = jump.anchors
    hitCursor.value = 0
    requestAnimationFrame(() => scrollToAnchor(jump.anchors[0]))
    document.addEventListener('keydown', handleHitKeydown)
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleHitKeydown)
})

const interactions = ref({
  likeCount: page.props.page.likeCount,
  isLiked: page.props.page.liked,
  isBookmarked: page.props.page.bookmarked,
  isWatched: page.props.page.watched,
})

const emptyPostStream = {
  posts: [],
  replyTargets: [],
  hasBefore: false,
  hasAfter: false,
  total: 0,
  maxPostNo: 0,
}

function handleInteractionChange(state: { likeCount: number; isLiked: boolean; isBookmarked: boolean; isWatched: boolean }) {
  interactions.value = state
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
</script>

<template>
  <div class="min-w-0">
    <div class="min-w-0" @click="handleMarkdownImageClick">
      <section class="gf-card xl:w-[calc(100%+292px)]">
        <div class="min-w-0 xl:grid xl:grid-cols-[minmax(0,1fr)_256px]">
          <!-- 左列：标题 + 正文 -->
          <div class="min-w-0">
            <div class="px-4 py-4 sm:px-5 sm:pt-5">
              <div class="flex flex-wrap items-center gap-2">
                <a href="/wiki" class="text-[13px] font-medium text-base-content/55 hover:text-primary">{{ t('wiki.home') }}</a>
                <span class="text-base-content/35">/</span>
                <span class="text-[13px] font-medium text-base-content/75">{{ page.props.page.namespace }}</span>
              </div>
              <h1 class="mt-2 break-words text-2xl font-bold leading-tight text-base-content [overflow-wrap:anywhere] sm:text-3xl">
                {{ page.props.page.title }}
              </h1>
              <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-[13px] text-base-content/55">
                <span class="inline-flex items-center gap-1.5">
                  <Clock class="h-3.5 w-3.5" />
                  {{ formatDateTime(page.props.page.updatedAt) }}
                </span>
                <span class="inline-flex items-center gap-1.5">
                  <Eye class="h-3.5 w-3.5" />
                  {{ formatNumber(page.props.page.viewCount) }}
                </span>
                <span class="inline-flex items-center gap-1.5">
                  <MessageSquare class="h-3.5 w-3.5" />
                  {{ formatNumber(page.props.page.postCount) }}
                </span>
              </div>
            </div>

            <!-- 移动端（xl 以下）：右栏 aside 隐藏，操作按钮 + 目录折叠展示在此保持可达 -->
            <div class="xl:hidden">
              <div class="border-t border-line/70 px-4 py-4 sm:px-5">
                <WikiPageActions
                  :page="page.props.page"
                  :can-edit="page.props.page.canEdit"
                  @interaction-change="handleInteractionChange"
                />
              </div>
              <div v-if="(page.props.page.toc || []).length" class="border-t border-line/70">
                <button
                  type="button"
                  class="gf-button gf-button-sm gf-button-muted m-4 sm:m-5"
                  :aria-expanded="mobileTocOpen"
                  @click="mobileTocOpen = !mobileTocOpen"
                >
                  <ChevronRight v-if="!mobileTocOpen" class="h-4 w-4" />
                  <ChevronDown v-else class="h-4 w-4" />
                  {{ t('wiki.tocTitle') }}
                </button>
                <div v-if="mobileTocOpen" class="pb-2">
                  <WikiToc :items="page.props.page.toc || []" />
                </div>
              </div>
            </div>

            <div class="border-t border-line/70 px-4 py-4 sm:px-5 sm:py-5">
              <div
                v-if="page.props.page.content"
                v-code-copy
                v-code-highlight
                v-math-render
                class="gf-prose gf-prose-post"
                v-html="page.props.page.content"
              />
              <div v-else class="rounded border border-dashed border-line bg-base-200/60 px-4 py-8 text-center text-sm text-base-content/55">
                {{ t('wiki.contentEmpty') }}
              </div>
            </div>
          </div>

          <!-- 右栏：目录 + 贡献者 + 操作 -->
          <aside class="hidden min-w-0 border-l border-line/70 xl:block">
            <div class="sticky top-19">
              <WikiToc :items="page.props.page.toc || []" />

              <div class="border-t border-line px-4 py-4">
                <WikiPageActions
                  :page="page.props.page"
                  :can-edit="page.props.page.canEdit"
                  @interaction-change="handleInteractionChange"
                />
              </div>

              <div v-if="page.props.contributors.length" class="border-t border-line px-4 py-4">
                <h3 class="text-sm font-semibold text-base-content/55">{{ t('wiki.contributors') }}</h3>
                <ul class="mt-3 space-y-2.5">
                  <li v-for="contributor in page.props.contributors" :key="contributor.userId || contributor.username">
                    <a
                      v-if="contributor.userId"
                      :href="`/u/${contributor.userId}`"
                      class="flex min-w-0 items-center gap-2.5 rounded-md p-1 transition-colors hover:bg-base-200"
                      @click="showUserCard({ id: contributor.userId, username: contributor.username, avatarUrl: contributor.avatarUrl }, $event)"
                    >
                      <UserAvatar
                        :src="contributor.avatarUrl"
                        :alt="contributor.username"
                        class="h-7 w-7 shrink-0 rounded-full object-cover ring-1 ring-line"
                      />
                      <span class="min-w-0 flex-1 truncate text-[13px] font-medium text-base-content/80">{{ contributor.username }}</span>
                      <span class="shrink-0 text-xs tabular-nums text-base-content/45">{{ contributor.count }}</span>
                    </a>
                    <a
                      v-else-if="contributor.githubUrl"
                      :href="contributor.githubUrl"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="flex min-w-0 items-center gap-2.5 rounded-md p-1 transition-colors hover:bg-base-200"
                      :title="contributor.githubUrl"
                    >
                      <img
                        v-if="contributor.avatarUrl"
                        :src="contributor.avatarUrl"
                        :alt="contributor.username"
                        loading="lazy"
                        class="h-7 w-7 shrink-0 rounded-full object-cover ring-1 ring-line"
                      />
                      <span
                        v-else
                        class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-base-200 text-[11px] font-bold text-base-content/55"
                      >
                        {{ contributor.username.slice(0, 1).toUpperCase() }}
                      </span>
                      <span class="min-w-0 flex-1 truncate text-[13px] font-medium text-base-content/80">{{ contributor.username }}</span>
                      <span class="shrink-0 text-xs tabular-nums text-base-content/45">{{ contributor.count }}</span>
                    </a>
                    <div
                      v-else
                      class="flex min-w-0 items-center gap-2.5 rounded-md p-1"
                    >
                      <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-base-200 text-[11px] font-bold text-base-content/55">
                        {{ contributor.username.slice(0, 1).toUpperCase() }}
                      </span>
                      <span class="min-w-0 flex-1 truncate text-[13px] font-medium text-base-content/80">{{ contributor.username }}</span>
                      <span class="shrink-0 text-xs tabular-nums text-base-content/45">{{ contributor.count }}</span>
                    </div>
                  </li>
                </ul>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <!-- 评论流：复用话题评论，按 topicId（保留回复栏；话题列表首帖已由正文承接） -->
      <PostStream
        class="mt-4"
        :topic-id="page.props.page.topicId"
        :topic-title="page.props.page.title"
        :initial-post-stream="emptyPostStream"
        :viewer="page.layout.viewer"
        :can-post="page.layout.viewer.isAuthenticated"
        :interactions="interactions"
        :hot-topics="page.props.hotTopics"
        hide-first-post
        :initial-post-stream-hidden="true"
        :sync-url="false"
        auto-load-first-window
        wide
        @interaction-state="handleInteractionChange"
      />
    </div>

    <MarkdownImageViewer ref="markdownImageViewer" />
  </div>
</template>
