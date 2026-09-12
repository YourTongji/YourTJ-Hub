<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bell, LayoutGrid, List, Mail, RefreshCw, UsersRound } from '@lucide/vue'
import { fetchPage } from '@/runtime/router'
import { useHomeFeedMode } from '@/runtime/home-feed-mode'
import { countNewTopics, firstPageUrl, prependTopics } from '@/site/utils/home-feed-refresh'
import EmptyState from '@/site/components/EmptyState.vue'
import TopicListFooter from '@/site/components/TopicListFooter.vue'
import TopicList from '@/site/components/TopicList.vue'
import type { HomeProps, LayoutPayload, PagePayload, TopicPayload } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: HomeProps
  pageUrl: string
}>()
const { t, locale } = useI18n()
const announcementReadStorageKey = 'goose:announcement:last-read-published-at'
const announcementReminderWindow = 7 * 24 * 60 * 60 * 1000

const topics = ref<TopicPayload[]>([])
const pagination = ref<HomeProps['pagination']>(page.props.pagination)
const announcement = ref<HomeProps['announcement']>(page.props.announcement)
const requiresEmailVerification = ref(page.layout.viewer.requiresEmailVerification)
const loadingMore = ref(false)
const refreshing = ref(false)
const checkingForNew = ref(false)
const newTopicCount = ref(0)
const mockNewTopicCount = import.meta.env.DEV
  ? Math.max(0, Number.parseInt(import.meta.env.VITE_MOCK_NEW_TOPIC_COUNT || '0', 10) || 0)
  : 0
const loadError = ref('')
const refreshStatusMessage = ref('')
const loadMoreSentinel = ref<HTMLElement | null>(null)
const pullDistance = ref(0)
const pullActive = ref(false)
const pullStartY = ref(0)
const pullRefreshEnabled = ref(false)
const announcementUnread = ref(shouldRemindAnnouncement())
const pullThreshold = 72
const pullMaxDistance = 108
const refreshPollMs = 45_000
let observer: IntersectionObserver | undefined
let refreshPollTimer: number | undefined
let refreshPollingActive = false
let refreshViewportQuery: MediaQueryList | undefined

const hasTopics = computed(() => topics.value.length > 0)
const showPinnedLabels = computed(() => page.props.sort === '' || page.props.sort === 'latest')
const pullLabel = computed(() => {
  if (refreshing.value) return t('topicList.refreshing')
  return pullDistance.value >= pullThreshold ? t('topicList.releaseToRefresh') : t('topicList.pullToRefresh')
})

// 信息流样式由所有首页分页共享；列表模式下桌面端保留 hover 弹层预览。
const { feedMode, setFeedMode: setSharedFeedMode } = useHomeFeedMode()

// 列表/卡片切换：绝对定位的「药丸」滑到激活按钮下方。
// 位置按按钮实际渲染尺寸测量（绝对值定位相对于容器的 padding box）；
// 字体加载、窗口缩放、i18n 文案宽度变化均由 ResizeObserver 兜底
// （同时观察按钮本身，容器尺寸不变但按钮宽度变化时也能及时重测）。
const feedModeGroup = ref<HTMLElement | null>(null)
const feedModeTableBtn = ref<HTMLElement | null>(null)
const feedModeCardBtn = ref<HTMLElement | null>(null)
const feedModePillLeft = ref(0)
const feedModePillWidth = ref(0)
const feedModePillReady = ref(false)
const feedModePillShouldAnimate = ref(false)
const reducedMotion = ref(false)
let feedModeResizeObserver: ResizeObserver | undefined
let feedModeMotionQuery: MediaQueryList | undefined

function updateFeedModePill() {
  const group = feedModeGroup.value
  const activeBtn = feedMode.value === 'table' ? feedModeTableBtn.value : feedModeCardBtn.value
  if (!group || !activeBtn) return
  const groupRect = group.getBoundingClientRect()
  if (groupRect.width === 0 || groupRect.height === 0) return // keep-alive 隐藏期间跳过测量
  const activeRect = activeBtn.getBoundingClientRect()
  const borderLeft = parseFloat(getComputedStyle(group).borderLeftWidth) || 0
  feedModePillLeft.value = activeRect.left - groupRect.left - borderLeft
  feedModePillWidth.value = activeRect.width
  feedModePillReady.value = true
}

function setFeedMode(mode: 'table' | 'card') {
  // 只有当前可见分页中的主动点击才应播放药丸动画；其他 KeepAlive
  // 实例在激活时只同步到最终位置，不重复播放同一段过渡。
  feedModePillShouldAnimate.value = mode !== feedMode.value
  setSharedFeedMode(mode)
}

// 最新、热门、流行页面各自是 KeepAlive 实例；任一实例切换模式时，
// 当前可见实例负责播放动画，其他实例只更新自己的最终位置。
watch(feedMode, () => void nextTick(updateFeedModePill))

// 激活按钮文字在白药丸到达时（delay 与药丸滑动时长一致，均由 --gf-feed-slide-ms 决定）翻白；
// 非激活按钮文字立即变灰（随药丸离去淡出）。reduced-motion 下全部瞬时切换。
function feedModeButtonClass(active: boolean): string[] {
  const base = active ? 'text-primary-content' : 'text-base-content/55 hover:text-base-content'
  if (reducedMotion.value || !feedModePillShouldAnimate.value) return [base]
  return active
    ? [base, 'transition-colors', 'delay-[var(--gf-feed-slide-ms)]', 'duration-100']
    : [base, 'transition-colors', 'duration-200']
}

function handleFeedModeMotionChange(event: MediaQueryListEvent) {
  reducedMotion.value = event.matches
}

// 切换语言后重新测量：ResizeObserver 只能捕获宽度变化的场景，
// 对「两个按钮总宽不变但各自宽度互换」的极端情况仍需要显式兜底。
watch(
  () => locale.value,
  () => void nextTick(updateFeedModePill),
)

const announcementItems = computed(() => announcement.value.items || [])
const hasMultipleAnnouncements = computed(() => announcementItems.value.length > 1)
const activeAnnouncementIndex = ref(0)
const activeAnnouncement = computed(() => announcementItems.value[activeAnnouncementIndex.value] || null)
const announcementPaused = ref(false)
let announcementTimer: number | undefined

function startAnnouncementRotation() {
  stopAnnouncementRotation()
  if (!hasMultipleAnnouncements.value) return
  if (typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  announcementTimer = window.setInterval(() => {
    activeAnnouncementIndex.value = (activeAnnouncementIndex.value + 1) % announcementItems.value.length
  }, 6000)
}

function stopAnnouncementRotation() {
  if (announcementTimer !== undefined) {
    window.clearInterval(announcementTimer)
    announcementTimer = undefined
  }
}

function pauseAnnouncementRotation() {
  announcementPaused.value = true
  stopAnnouncementRotation()
}

function resumeAnnouncementRotation() {
  announcementPaused.value = false
  startAnnouncementRotation()
}

function selectAnnouncement(index: number) {
  activeAnnouncementIndex.value = index
}

watch(
  () => page.pageUrl,
  () => {
    topics.value = [...page.props.topics]
    pagination.value = page.props.pagination
    announcement.value = page.props.announcement
    requiresEmailVerification.value = page.layout.viewer.requiresEmailVerification
    newTopicCount.value = mockNewTopicCount
    loadError.value = ''
    refreshStatusMessage.value = ''
    void nextTick(observeSentinel)
  },
  { immediate: true },
)

watch(
  () => page.props.topics,
  (incoming) => {
    const unseenByID = new Map(incoming.map((topic) => [topic.id, topic.unseen]))
    topics.value = topics.value.map((topic) =>
      unseenByID.has(topic.id) ? { ...topic, unseen: unseenByID.get(topic.id) } : topic,
    )
  },
)

watch(
  () => [announcement.value.enabled, announcement.value.publishedAt] as const,
  () => refreshAnnouncementReminder(),
)

watch(hasMultipleAnnouncements, (multiple) => {
  if (multiple) startAnnouncementRotation()
  else stopAnnouncementRotation()
})

function currentFirstPageUrl() {
  return firstPageUrl(page.pageUrl, window.location.origin)
}

async function checkForNewTopics() {
  // 仅供本地设计预览：固定待刷新数量，避免 45 秒探测覆盖 mock 状态。
  if (mockNewTopicCount > 0) return
  if (checkingForNew.value || refreshing.value || loadingMore.value || document.hidden) return
  checkingForNew.value = true
  try {
    const payload = (await fetchPage(currentFirstPageUrl())) as PagePayload<HomeProps>
    newTopicCount.value = countNewTopics(topics.value, payload.props.topics)
  } catch {
    // 后台探测失败不打断用户；显式刷新时才显示可恢复错误。
  } finally {
    checkingForNew.value = false
  }
}

async function refreshFirstPage(mode: 'prepend' | 'replace') {
  if (refreshing.value || loadingMore.value) return
  refreshing.value = true
  loadError.value = ''
  refreshStatusMessage.value = t('topicList.refreshing')
  try {
    const payload = (await fetchPage(currentFirstPageUrl())) as PagePayload<HomeProps>
    topics.value = mode === 'prepend'
      ? prependTopics(topics.value, payload.props.topics)
      : [...payload.props.topics]
    pagination.value = payload.props.pagination
    announcement.value = payload.props.announcement
    requiresEmailVerification.value = payload.layout.viewer.requiresEmailVerification
    newTopicCount.value = 0
    activeAnnouncementIndex.value = 0
    refreshAnnouncementReminder()
    refreshStatusMessage.value = t('topicList.refreshComplete')
    void nextTick(observeSentinel)
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : t('common.loadFailed')
    refreshStatusMessage.value = t('topicList.refreshFailed')
  } finally {
    refreshing.value = false
    pullDistance.value = 0
    pullActive.value = false
  }
}

function handlePullStart(event: TouchEvent) {
  if (!pullRefreshEnabled.value || refreshing.value || loadingMore.value || window.scrollY > 0) return
  const touch = event.touches[0]
  if (!touch) return
  pullStartY.value = touch.clientY
  pullActive.value = true
}

function handlePullMove(event: TouchEvent) {
  if (!pullActive.value || window.scrollY > 0) return
  const touch = event.touches[0]
  if (!touch) return
  const delta = touch.clientY - pullStartY.value
  if (delta <= 0) {
    pullDistance.value = 0
    return
  }
  event.preventDefault()
  // 阻尼保持手势可控，达到阈值后仍保留少量余量。
  pullDistance.value = Math.min(pullMaxDistance, delta * 0.56)
}

function handlePullEnd() {
  if (!pullActive.value) return
  const shouldRefresh = pullDistance.value >= pullThreshold
  pullActive.value = false
  if (shouldRefresh) void refreshFirstPage('replace')
  else pullDistance.value = 0
}

// 轮询与视口宽度联动：窄窗口（<640px，移动端布局）自动停止 45s 探测，
// 回到桌面宽度自动恢复；change 监听在组件卸载前保持挂载，
// keep-alive 后台实例由 refreshPollingActive 守卫，不会误启轮询。
function handleRefreshViewportChange() {
  syncRefreshPolling()
}

function startRefreshPolling() {
  refreshPollingActive = true
  syncRefreshPolling()
}

function stopRefreshPolling() {
  refreshPollingActive = false
  stopRefreshPollTimer()
}

function syncRefreshPolling() {
  stopRefreshPollTimer()
  if (refreshPollingActive && refreshViewportQuery?.matches) {
    refreshPollTimer = window.setInterval(() => void checkForNewTopics(), refreshPollMs)
  }
}

function stopRefreshPollTimer() {
  if (refreshPollTimer !== undefined) {
    window.clearInterval(refreshPollTimer)
    refreshPollTimer = undefined
  }
}

async function loadMore() {
  if (loadingMore.value || refreshing.value || !pagination.value.hasNext || !pagination.value.nextUrl) return

  loadingMore.value = true
  loadError.value = ''
  try {
    const payload = (await fetchPage(new URL(pagination.value.nextUrl, window.location.origin))) as PagePayload<HomeProps>
    topics.value = mergeTopics(topics.value, payload.props.topics)
    pagination.value = payload.props.pagination
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : t('common.loadFailed')
  } finally {
    loadingMore.value = false
  }
}

function mergeTopics(current: TopicPayload[], incoming: TopicPayload[]) {
  const seen = new Set(current.map((topic) => topic.id))
  return [...current, ...incoming.filter((topic) => !seen.has(topic.id))]
}

function sortTabLabel(key: string, fallback?: string) {
  if (key === 'latest') return t('topicList.tabs.latest')
  if (key === 'hot') return t('topicList.tabs.hot')
  if (key === 'popular') return t('topicList.tabs.popular')
  return fallback || key
}

// 铃铛为提醒开关：摇铃（未读提醒）↔ 静止（已读静音），
// 按下停止/恢复，用运动与静止两种状态指示。
function toggleAnnouncementRead() {
  announcementUnread.value = !announcementUnread.value
  const publishedAt = parseAnnouncementTime(announcement.value.publishedAt)
  if (!Number.isFinite(publishedAt)) return
  try {
    if (announcementUnread.value) {
      window.localStorage.removeItem(announcementReadStorageKey)
    } else {
      window.localStorage.setItem(announcementReadStorageKey, String(publishedAt))
    }
  } catch {
    // Storage may be unavailable in private or restricted browsing contexts.
  }
}

function shouldRemindAnnouncement() {
  if (!announcement.value.enabled) return false
  const publishedAt = parseAnnouncementTime(announcement.value.publishedAt)
  if (!Number.isFinite(publishedAt)) return false
  const age = Date.now() - publishedAt
  // 负数 age 说明浏览器时区与服务器不一致导致解析偏移，视为刚发布仍按未读提醒；
  // 只限制提醒窗口上限，避免旧公告一直打扰。
  if (age > announcementReminderWindow) return false

  try {
    const lastReadAt = Number(window.localStorage.getItem(announcementReadStorageKey) || 0)
    return !Number.isFinite(lastReadAt) || lastReadAt < publishedAt
  } catch {
    return true
  }
}

function parseAnnouncementTime(value?: string) {
  return Date.parse((value || '').replace(' ', 'T'))
}

function refreshAnnouncementReminder() {
  announcementUnread.value = shouldRemindAnnouncement()
}

function syncAnnouncementRead(event: StorageEvent) {
  if (event.key === announcementReadStorageKey) refreshAnnouncementReminder()
}

function observeSentinel() {
  observer?.disconnect()
  if (!loadMoreSentinel.value || !('IntersectionObserver' in window)) return
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) void loadMore()
    },
    { rootMargin: '480px 0px' },
  )
  observer.observe(loadMoreSentinel.value)
}

onMounted(() => {
  observeSentinel()
  window.addEventListener('storage', syncAnnouncementRead)
  startAnnouncementRotation()
  const standalone = window.matchMedia('(display-mode: standalone)').matches
    || Boolean((navigator as Navigator & { standalone?: boolean }).standalone)
  pullRefreshEnabled.value = standalone && window.matchMedia('(pointer: coarse)').matches
  if (pullRefreshEnabled.value) document.documentElement.classList.add('gf-pwa-pull-refresh')
  refreshViewportQuery = window.matchMedia('(min-width: 640px)')
  refreshViewportQuery.addEventListener('change', handleRefreshViewportChange)
  startRefreshPolling()
  feedModeMotionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  reducedMotion.value = feedModeMotionQuery.matches
  feedModeMotionQuery.addEventListener('change', handleFeedModeMotionChange)
  void nextTick(updateFeedModePill)
  feedModeResizeObserver = new ResizeObserver(() => updateFeedModePill())
  if (feedModeGroup.value) feedModeResizeObserver.observe(feedModeGroup.value)
  // 额外观察两个按钮：某些布局变化（如换行、padding 调整）可能只改变按钮宽度而容器不变
  if (feedModeTableBtn.value) feedModeResizeObserver.observe(feedModeTableBtn.value)
  if (feedModeCardBtn.value) feedModeResizeObserver.observe(feedModeCardBtn.value)
})
onActivated(() => {
  // KeepAlive 页面重新显示时，药丸已经应该处于共享状态的最终位置，
  // 不把这次同步误认为用户主动切换。
  feedModePillShouldAnimate.value = false
  void nextTick(observeSentinel)
  startAnnouncementRotation()
  if (pullRefreshEnabled.value) document.documentElement.classList.add('gf-pwa-pull-refresh')
  startRefreshPolling()
  void nextTick(updateFeedModePill)
})
onDeactivated(() => {
  observer?.disconnect()
  stopAnnouncementRotation()
  stopRefreshPolling()
  document.documentElement.classList.remove('gf-pwa-pull-refresh')
})

onBeforeUnmount(() => {
  observer?.disconnect()
  window.removeEventListener('storage', syncAnnouncementRead)
  stopAnnouncementRotation()
  stopRefreshPolling()
  document.documentElement.classList.remove('gf-pwa-pull-refresh')
  feedModeResizeObserver?.disconnect()
  feedModeMotionQuery?.removeEventListener('change', handleFeedModeMotionChange)
  refreshViewportQuery?.removeEventListener('change', handleRefreshViewportChange)
})

</script>

<template>
    <div
      class="relative pb-12"
      @touchstart.passive="handlePullStart"
      @touchmove="handlePullMove"
      @touchend.passive="handlePullEnd"
      @touchcancel.passive="handlePullEnd"
    >
      <div
        v-if="pullRefreshEnabled"
        class="pointer-events-none absolute inset-x-0 top-0 z-20 flex justify-center sm:hidden"
        aria-hidden="true"
        :style="{
          opacity: refreshing ? 1 : Math.min(1, pullDistance / pullThreshold),
          transform: `translateY(${Math.max(-32, pullDistance - 44)}px)`,
        }"
      >
        <div class="gf-pull-refresh-indicator inline-flex items-center gap-2 rounded-full bg-base-100/95 px-3 py-2 text-xs font-semibold text-base-content shadow-sm ring-1 ring-line/70 backdrop-blur-sm">
          <RefreshCw
            class="h-3.5 w-3.5 text-primary"
            :class="refreshing && !reducedMotion ? 'animate-spin' : ''"
          />
          <span>{{ pullLabel }}</span>
        </div>
      </div>
      <p class="sr-only" role="status" aria-live="polite">{{ refreshStatusMessage }}</p>

      <aside
        v-if="requiresEmailVerification"
        class="gf-email-verification mb-0 border-y border-warning/30 bg-warning/10 sm:-mt-3 sm:mb-3 sm:rounded-b-lg sm:border-x sm:border-t-0"
        :aria-label="t('topicList.emailVerification.title')"
      >
        <div class="flex items-center gap-2 px-3 py-2 text-[13px] leading-5 text-warning sm:px-4 sm:text-sm">
          <Mail class="h-4 w-4 shrink-0 text-warning" />
          <div class="min-w-0 flex-1">
            <span class="font-semibold text-warning">{{ t('topicList.emailVerification.title') }}</span>
            <span class="mx-1 text-warning">·</span>
            <span>{{ t('topicList.emailVerification.description') }}</span>
          </div>
          <a href="/settings" class="shrink-0 font-semibold text-warning hover:text-warning">
            {{ t('topicList.emailVerification.action') }}
          </a>
        </div>
      </aside>

      <aside
        v-if="announcement.enabled && (announcementItems.length > 0 || announcement.html)"
        class="gf-panel gf-announcement-panel mb-0 overflow-hidden border border-primary/15 bg-gradient-to-r from-primary/5 via-base-100 to-base-100 px-3 py-2.5 sm:mb-3 sm:px-4 sm:py-3"
        :aria-label="t('topicList.announcement')"
        @mouseenter="pauseAnnouncementRotation"
        @mouseleave="resumeAnnouncementRotation"
      >
        <div class="flex items-start gap-2 sm:gap-2.5">
          <button
            type="button"
            class="-mx-1 inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full transition"
            :class="announcementUnread
              ? 'text-primary hover:bg-primary/15 active:bg-primary/25'
              : 'text-base-content/45 hover:bg-base-300/70 hover:text-base-content/75'"
            :title="announcementUnread ? t('topicList.markAnnouncementRead') : t('topicList.markAnnouncementUnread')"
            :aria-label="announcementUnread ? t('topicList.markAnnouncementRead') : t('topicList.markAnnouncementUnread')"
            :aria-pressed="announcementUnread"
            @click="toggleAnnouncementRead"
          >
            <Bell class="h-4 w-4" :class="announcementUnread ? 'announcement-unread-bell' : ''" />
          </button>
          <div class="min-w-0 flex-1">
            <template v-if="activeAnnouncement">
              <div class="gf-prose gf-prose-announcement">
                <span
                  v-if="activeAnnouncement.title"
                  class="mb-1 block text-[11px] font-bold uppercase tracking-wide text-primary"
                >
                  {{ activeAnnouncement.title }}
                </span>
                <div v-html="activeAnnouncement.html" />
              </div>
              <div
                v-if="hasMultipleAnnouncements"
                class="mt-1.5 flex items-center gap-0.5"
                role="tablist"
                :aria-label="t('topicList.announcement')"
              >
                <button
                  v-for="(item, index) in announcementItems"
                  :key="item.id"
                  type="button"
                  role="tab"
                  class="group flex h-5 w-5 items-center justify-center rounded-full transition-colors hover:bg-base-300/70"
                  :class="index === activeAnnouncementIndex ? 'bg-primary/10' : ''"
                  :aria-label="t('topicList.announcement') + ' ' + (index + 1)"
                  :aria-selected="index === activeAnnouncementIndex"
                  @click="selectAnnouncement(index)"
                >
                  <span
                    class="block rounded-full transition-all duration-200"
                    :class="index === activeAnnouncementIndex ? 'h-2 w-4 bg-primary' : 'h-2 w-2 bg-base-content/30 group-hover:bg-base-content/55'"
                  />
                </button>
              </div>
            </template>
            <div v-else class="gf-prose gf-prose-announcement" v-html="announcement.html" />
          </div>
        </div>
      </aside>

      <section :class="feedMode === 'card' ? '' : 'gf-card overflow-hidden'">
        <div
          class="gf-home-topic-toolbar"
          :class="feedMode === 'card' ? 'gf-home-topic-toolbar-card' : ''"
        >
          <div class="gf-home-topic-tabs">
            <a
              v-for="tab in page.props.tabs"
              :key="tab.key"
              :href="tab.url"
              class="gf-tab"
              :class="tab.active ? 'gf-tab-active' : 'gf-tab-idle'"
            >
              {{ sortTabLabel(tab.key, tab.label) }}
            </a>
          </div>
          <div class="gf-home-topic-actions flex shrink-0 items-center gap-2">
            <button
              type="button"
              class="gf-home-refresh-button hidden h-8 shrink-0 items-center gap-1.5 rounded-full border px-2.5 text-xs font-semibold sm:inline-flex"
              :class="newTopicCount > 0
                ? 'border-primary/25 bg-primary/10 text-primary'
                : 'border-line bg-base-100 text-base-content/55 hover:bg-base-200 hover:text-base-content'"
              :disabled="refreshing || loadingMore"
              :title="newTopicCount > 0 ? t('topicList.newTopics', { count: newTopicCount }) : t('topicList.refreshTopics')"
              :aria-label="newTopicCount > 0 ? t('topicList.newTopics', { count: newTopicCount }) : t('topicList.refreshTopics')"
              @click="refreshFirstPage('prepend')"
            >
              <RefreshCw
                class="h-3.5 w-3.5"
                :class="refreshing && !reducedMotion ? 'animate-spin' : ''"
              />
              <span v-if="newTopicCount > 0" class="tabular-nums whitespace-nowrap">
                {{ t('topicList.newTopics', { count: newTopicCount }) }}
              </span>
            </button>

            <div
              ref="feedModeGroup"
              class="relative flex shrink-0 items-center gap-0.5 rounded-full border border-line bg-base-100 p-0.5"
              role="group"
              :aria-label="t('topicList.feedMode')"
            >
              <span
                aria-hidden="true"
                class="pointer-events-none absolute bottom-0.5 top-0.5 rounded-full bg-primary"
                :class="feedModePillReady && feedModePillShouldAnimate && !reducedMotion
                  ? 'transition-[left,width] duration-[var(--gf-feed-slide-ms)] ease-out'
                  : ''"
                :style="{ left: `${feedModePillLeft}px`, width: `${feedModePillWidth}px` }"
              />
              <button
                ref="feedModeTableBtn"
                type="button"
                class="relative z-10 inline-flex h-7 items-center gap-1 rounded-full px-2.5 text-xs font-semibold"
                :class="feedModeButtonClass(feedMode === 'table')"
                :aria-pressed="feedMode === 'table'"
                @click="setFeedMode('table')"
              >
                <List class="h-3.5 w-3.5" />
                {{ t('topicList.feedModeTable') }}
              </button>
              <button
                ref="feedModeCardBtn"
                type="button"
                class="relative z-10 inline-flex h-7 items-center gap-1 rounded-full px-2.5 text-xs font-semibold"
                :class="feedModeButtonClass(feedMode === 'card')"
                :aria-pressed="feedMode === 'card'"
                @click="setFeedMode('card')"
              >
                <LayoutGrid class="h-3.5 w-3.5" />
                {{ t('topicList.feedModeCard') }}
              </button>
            </div>
          </div>
        </div>

        <TopicList :topics="topics" :viewer-id="page.layout.viewer.id" home :show-pinned="showPinnedLabels" :feed-mode="feedMode">
          <template #empty>
            <EmptyState v-if="!hasTopics" :icon="UsersRound" :title="t('topicList.emptyTitle')" :description="t('topicList.emptyDescription')" />
          </template>
        </TopicList>

        <div ref="loadMoreSentinel">
          <TopicListFooter
            :pagination="pagination"
            :loading-more="loadingMore"
            :has-topics="hasTopics"
            :load-error="loadError"
            :bare="feedMode === 'card'"
            @load-more="loadMore"
          />
        </div>
      </section>
    </div>
</template>

<style scoped>
.announcement-unread-bell {
  /* 未读提醒：持续摇铃标识运动状态；点击后移除该类进入静止态 */
  animation: announcement-bell-ring 2s ease-in-out infinite;
  transform-origin: 50% 15%;
}

@keyframes announcement-bell-ring {
  0%, 30%, 100% {
    transform: rotate(0) scale(1);
  }
  5% {
    transform: rotate(20deg) scale(1.12);
  }
  10% { transform: rotate(-18deg) scale(1.12); }
  15% {
    transform: rotate(14deg) scale(1.08);
  }
  20% { transform: rotate(-10deg) scale(1.05); }
  25% { transform: rotate(5deg) scale(1.02); }
}

/* 信息流切换：药丸滑动时长；激活按钮文字翻白的 delay 与之共享（见 feedModeButtonClass），
   调整滑动速度时只改这一处。 */
.gf-home-topic-toolbar {
  --gf-feed-slide-ms: 300ms;
}

.gf-home-refresh-button {
  transition-property: background-color, border-color, color, opacity, transform;
  transition-duration: 150ms;
  transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
}

.gf-home-refresh-button:active:not(:disabled) {
  transform: scale(0.96);
}

.gf-home-refresh-button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.gf-pull-refresh-indicator {
  transition: opacity 150ms cubic-bezier(0.2, 0, 0, 1);
}

:global(html.gf-pwa-pull-refresh) {
  overscroll-behavior-y: contain;
}

/* 卡片模式：工具栏自身成为独立的圆角卡片（四角圆角 + 边框 + 阴影），
   下方帖子卡片悬浮在页面背景上，与工具栏相互独立。 */
.gf-home-topic-toolbar-card {
  border: var(--gf-border) solid var(--gf-color-line);
  border-radius: var(--gf-radius-box);
  box-shadow: 0 2px 12px rgb(0 0 0 / calc(var(--gf-depth) * 0.04));
}

/* 移动端保持通栏风格：还原为仅底部一条分隔线的扁平工具栏 */
@media (max-width: 639.98px) {
  .gf-home-topic-toolbar-card {
    border-top: 0;
    border-left: 0;
    border-right: 0;
    border-radius: 0;
    box-shadow: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .announcement-unread-bell {
    animation: none;
  }

  .gf-home-refresh-button,
  .gf-pull-refresh-indicator {
    transition-duration: 0.01ms;
  }

  .gf-home-refresh-button:active:not(:disabled) {
    transform: none;
  }
}
</style>
