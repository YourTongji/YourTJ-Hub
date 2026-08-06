<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bell, LayoutGrid, List, Mail, Plus, UsersRound } from '@lucide/vue'
import { fetchPage } from '@/runtime/router'
import EmptyState from '@/site/components/EmptyState.vue'
import TopicListFooter from '@/site/components/TopicListFooter.vue'
import TopicList from '@/site/components/TopicList.vue'
import type { HomeProps, LayoutPayload, PagePayload, TopicPayload } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: HomeProps
  pageUrl: string
}>()
const { t } = useI18n()
const announcementReadStorageKey = 'goose:announcement:last-read-published-at'
const announcementReminderWindow = 7 * 24 * 60 * 60 * 1000

const topics = ref<TopicPayload[]>([])
const pagination = ref<HomeProps['pagination']>(page.props.pagination)
const loadingMore = ref(false)
const loadError = ref('')
const loadMoreSentinel = ref<HTMLElement | null>(null)
const announcementUnread = ref(shouldRemindAnnouncement())
let observer: IntersectionObserver | undefined

const hasTopics = computed(() => topics.value.length > 0)
const showPinnedLabels = computed(() => page.props.sort === '' || page.props.sort === 'latest')

// 信息流样式：列表（表格）↔ 卡片，桌面与移动端均可切换，选择记忆在本地；
// 列表模式下桌面端保留 hover 弹层预览。
const feedStorageKey = 'goose:home-feed-mode'
const feedMode = ref<'table' | 'card'>(readFeedMode())

function readFeedMode(): 'table' | 'card' {
  try {
    return window.localStorage.getItem(feedStorageKey) === 'card' ? 'card' : 'table'
  } catch {
    return 'table'
  }
}

function setFeedMode(mode: 'table' | 'card') {
  feedMode.value = mode
  try {
    window.localStorage.setItem(feedStorageKey, mode)
  } catch {
    // Storage may be unavailable in private or restricted browsing contexts.
  }
}

const announcementItems = computed(() => page.props.announcement.items || [])
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
    loadError.value = ''
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
  () => [page.props.announcement.enabled, page.props.announcement.publishedAt] as const,
  () => refreshAnnouncementReminder(),
)

watch(hasMultipleAnnouncements, (multiple) => {
  if (multiple) startAnnouncementRotation()
  else stopAnnouncementRotation()
})

async function loadMore() {
  if (loadingMore.value || !pagination.value.hasNext || !pagination.value.nextUrl) return

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
  const publishedAt = parseAnnouncementTime(page.props.announcement.publishedAt)
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
  if (!page.props.announcement.enabled) return false
  const publishedAt = parseAnnouncementTime(page.props.announcement.publishedAt)
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
})
onActivated(() => {
  void nextTick(observeSentinel)
  startAnnouncementRotation()
})
onDeactivated(() => {
  observer?.disconnect()
  stopAnnouncementRotation()
})

onBeforeUnmount(() => {
  observer?.disconnect()
  window.removeEventListener('storage', syncAnnouncementRead)
  stopAnnouncementRotation()
})

</script>

<template>
    <div class="pb-12">
      <aside
        v-if="page.layout.viewer.requiresEmailVerification"
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
        v-if="page.props.announcement.enabled && (announcementItems.length > 0 || page.props.announcement.html)"
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
            <div v-else class="gf-prose gf-prose-announcement" v-html="page.props.announcement.html" />
          </div>
        </div>
      </aside>

      <section class="gf-card overflow-hidden">
        <div class="gf-home-topic-toolbar">
          <div class="gf-home-topic-tools">
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
            <div
              class="flex items-center gap-0.5 rounded-full border border-line bg-base-100 p-0.5"
              role="group"
              :aria-label="t('topicList.feedMode')"
            >
              <button
                type="button"
                class="inline-flex h-7 items-center gap-1 rounded-full px-2.5 text-xs font-semibold transition-colors"
                :class="feedMode === 'table' ? 'bg-primary text-primary-content' : 'text-base-content/55 hover:text-base-content'"
                :aria-pressed="feedMode === 'table'"
                @click="setFeedMode('table')"
              >
                <List class="h-3.5 w-3.5" />
                {{ t('topicList.feedModeTable') }}
              </button>
              <button
                type="button"
                class="inline-flex h-7 items-center gap-1 rounded-full px-2.5 text-xs font-semibold transition-colors"
                :class="feedMode === 'card' ? 'bg-primary text-primary-content' : 'text-base-content/55 hover:text-base-content'"
                :aria-pressed="feedMode === 'card'"
                @click="setFeedMode('card')"
              >
                <LayoutGrid class="h-3.5 w-3.5" />
                {{ t('topicList.feedModeCard') }}
              </button>
            </div>
          </div>
          <a href="/publish" class="gf-button gf-button-md gf-button-primary shrink-0 whitespace-nowrap px-3 sm:h-8">
            <Plus class="h-4 w-4" />
            {{ t('topicList.newTopic') }}
          </a>
        </div>

        <TopicList :topics="topics" home :show-pinned="showPinnedLabels" :feed-mode="feedMode">
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

@media (prefers-reduced-motion: reduce) {
  .announcement-unread-bell {
    animation: none;
  }
}
</style>
