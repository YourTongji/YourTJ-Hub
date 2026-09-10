<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Bird,
  CalendarDays,
  ExternalLink,
  Feather,
  Loader2,
  Radio,
  UserX,
} from '@lucide/vue'
import { getUserCard, followUser } from '@/runtime/api'
// 关注状态单一事实源：卡片缓存的可变状态必须经它登记/广播，跨 surface 才能一致（issue #593）
import { broadcastFollowChange, getFollowChangeSeq, getKnownFollowState, onFollowChange, recordFollowState } from '@/runtime/follow-state'
import { formatDate, formatNumber, timeAgo } from '@/runtime/format'
import type { UserCardShowDetail } from '@/runtime/user-card-events'
import type { UserCardPayload } from '@gooseforum/client'
import { socialIcons, socialLabels, type SimpleIcon } from '@/site/utils/social-icons'
import { badgeClass, badgeIconURL, badgeTooltip } from '@/site/utils/badge-style'
import UserAvatar from './UserAvatar.vue'

const { t } = useI18n()
const visible = ref(false)
const loading = ref(false)
const error = ref('')
const fallbackUser = ref<UserCardShowDetail['user'] | null>(null)
const card = ref<UserCardPayload | null>(null)
const position = ref({ left: 0, top: 0 })
const cardEl = ref<HTMLElement | null>(null)
const activeBadgeCode = ref('')
const cache = new Map<number, UserCardPayload>()
const cacheFetchedAt = new Map<number, number>()
// 缓存命中后超过该时长触发后台重验（stale-while-revalidate）：
// 卡片组件与应用同生命周期，缓存一旦永不失效，跨 surface 的关注变更永远不可见
const CACHE_REVALIDATE_TTL_MS = 60_000
let requestToken = 0
let preferredSide: 'top' | 'bottom' | null = null

const displayName = computed(() => card.value?.nickname || fallbackUser.value?.username || card.value?.username || '')
const username = computed(() => card.value?.username || fallbackUser.value?.username || '')
const avatarUrl = computed(() => card.value?.avatarUrl || fallbackUser.value?.avatarUrl || '')
const wornBadge = computed(() => card.value?.wornBadge || fallbackUser.value?.wornBadge || null)
const profileUrl = computed(() => `/u/${card.value?.userId || fallbackUser.value?.id || 0}`)
// 悬停卡片简介只展示 bio，签名不再回退到简介行
const bioText = computed(() => card.value?.bio || '')
// 全卡背景：用户封面铺满整卡（无封面/加载中保持纯色底，避免彩色跳动）
const coverStyle = computed(() => {
  const coverUrl = card.value?.profileCoverUrl?.trim()
  return coverUrl
    ? { backgroundImage: `url(${JSON.stringify(coverUrl)})`, backgroundSize: 'cover', backgroundPosition: 'center' }
    : {}
})
const isFollowing = ref(false)
const followLoading = ref(false)
const followError = ref('')
const externalLinks = computed(() => {
  const links: Array<{ key: string; label: string; url: string; icon?: SimpleIcon }> = []
  const primaryUrl = normalizeWebsiteURL(card.value?.website || '')
  if (primaryUrl) {
    links.push({
      key: 'website',
      label: card.value?.websiteName || formatLinkLabel(primaryUrl),
      url: primaryUrl,
    })
  }
  const externalInformation = card.value?.externalInformation || {}
  for (const [key, item] of Object.entries(externalInformation)) {
    const url = normalizeWebsiteURL(item?.link || '')
    if (!url) continue
    links.push({ key, label: socialLabels[key] || formatLinkLabel(url), url, icon: socialIcons[key] })
  }
  return links
})
const visibleBadges = computed(() => (card.value?.badges || []).slice(0, 5))
const isAccountClosed = computed(() => Boolean(card.value?.isAccountClosed))

function normalizeWebsiteURL(value: string) {
  const url = value.trim()
  if (!url) return ''
  if (/^https?:\/\//i.test(url)) return url
  return `https://${url}`
}

function formatLinkLabel(url: string) {
  return url.replace(/^https?:\/\//i, '').replace(/^www\./i, '').replace(/\/$/, '')
}

function hideNow() {
  visible.value = false
  activeBadgeCode.value = ''
  preferredSide = null
}

function placeCard(target: HTMLElement) {
  const rect = target.getBoundingClientRect()
  const cardWidth = Math.min(320, window.innerWidth - 24)
  const measuredHeight = cardEl.value?.offsetHeight || 0
  const cardHeight = Math.max(measuredHeight, 220)
  const gap = 10
  const viewportPadding = 12
  const viewportWidth = window.innerWidth

  let left = rect.left
  left = Math.max(viewportPadding, Math.min(left, viewportWidth - cardWidth - viewportPadding))
  const belowTop = rect.bottom + gap
  const aboveTop = rect.top - cardHeight - gap
  if (!preferredSide) {
    const belowSpace = window.innerHeight - rect.bottom - gap - viewportPadding
    const aboveSpace = rect.top - gap - viewportPadding
    preferredSide = belowSpace >= cardHeight || belowSpace >= aboveSpace ? 'bottom' : 'top'
  }
  const top = preferredSide === 'top' ? aboveTop : belowTop

  position.value = {
    left,
    top: Math.max(viewportPadding, Math.min(top, window.innerHeight - cardHeight - viewportPadding)),
  }
}

function storeInCache(userId: number, payload: UserCardPayload) {
  cache.set(userId, payload)
  cacheFetchedAt.set(userId, Date.now())
  recordFollowState(userId, payload.isFollowing)
}

// 回源落地（PR #600 review 2）：若请求飞行期间发生了同 tab 关注变更广播，
// 响应里的 isFollowing 是变更前的旧快照，关注状态以广播为准，其余字段照常采纳。
function adoptFreshCard(userId: number, fresh: UserCardPayload, seqAtRequest: number) {
  if (getFollowChangeSeq(userId) !== seqAtRequest) {
    const known = getKnownFollowState(userId)
    if (known !== undefined) fresh.isFollowing = known
  }
  storeInCache(userId, fresh)
}

// stale-while-revalidate：立即展示缓存，后台向 /api/user-card 回源。
// 命中缓存不再意味着「本会话内永远正确」——用户主页取关、其他标签页变更都要靠这里收敛。
async function revalidateCard(userId: number, token: number) {
  const seqAtRequest = getFollowChangeSeq(userId)
  try {
    const result = await getUserCard(userId)
    adoptFreshCard(userId, result, seqAtRequest)
    // 仅当仍在展示同一用户时才刷新 UI；token 过期说明已切换到别的卡片，只更新缓存
    if (token !== requestToken) return
    if (card.value?.userId === userId) {
      card.value = result
      isFollowing.value = result.isFollowing
    }
  } catch {
    // 重验失败保留缓存展示，下次打开再试
  }
}

async function show(event: Event) {
  const detail = (event as CustomEvent<UserCardShowDetail>).detail
  if (!detail?.user?.id || !detail.target) return

  fallbackUser.value = detail.user
  visible.value = true
  error.value = ''
  followError.value = ''
  isFollowing.value = getKnownFollowState(detail.user.id) ?? Boolean(detail.user.isFollowing)
  requestAnimationFrame(() => placeCard(detail.target))

  const cached = cache.get(detail.user.id)
  if (cached) {
    card.value = cached
    isFollowing.value = getKnownFollowState(detail.user.id) ?? cached.isFollowing
    loading.value = false
    const fetchedAt = cacheFetchedAt.get(detail.user.id) || 0
    if (Date.now() - fetchedAt >= CACHE_REVALIDATE_TTL_MS) {
      void revalidateCard(detail.user.id, ++requestToken)
    }
    return
  }

  const token = ++requestToken
  loading.value = true
  card.value = null
  const seqAtRequest = getFollowChangeSeq(detail.user.id)
  try {
    const result = await getUserCard(detail.user.id)
    adoptFreshCard(detail.user.id, result, seqAtRequest)
    if (token !== requestToken) return
    card.value = result
    isFollowing.value = result.isFollowing
    requestAnimationFrame(() => placeCard(detail.target))
  } catch {
    if (token !== requestToken) return
    error.value = t('userCard.unavailable')
  } finally {
    if (token === requestToken) loading.value = false
  }
}

async function toggleFollow() {
  const userCard = card.value
  if (!userCard || userCard.isSelf || followLoading.value) return
  followLoading.value = true
  followError.value = ''
  try {
    await followUser(userCard.userId, isFollowing.value)
  } catch (e) {
    // 只有关注操作本身的失败才算「关注失败」，保持原状态并内联提示
    followError.value = e instanceof Error ? e.message : t('api.followFailed')
    followLoading.value = false
    return
  }
  // 以下不再有「关注失败」：POST 已成功，翻转与广播是本地的即时反馈
  const target = !isFollowing.value
  isFollowing.value = target
  userCard.isFollowing = target
  broadcastFollowChange(userCard.userId, target)
  // 回源拿权威结果与粉丝数；失败只代表资料未刷新，不算关注失败（PR #600 review 1），
  // 计数等字段由 SWR/TTL 下次重验兜底。loading 保持到回源结束，避免飞行期间再次点击发出反向 action
  const seqAtRequest = getFollowChangeSeq(userCard.userId)
  try {
    const fresh = await getUserCard(userCard.userId)
    adoptFreshCard(userCard.userId, fresh, seqAtRequest)
    if (card.value?.userId === userCard.userId) {
      card.value = fresh
      isFollowing.value = fresh.isFollowing
    }
  } catch {
    // 回源失败保留乐观状态，静默等待下次重验
  }
  followLoading.value = false
}

function onDocumentPointerDown(event: PointerEvent) {
  if (!visible.value) return
  const target = event.target
  if (target instanceof Node && cardEl.value?.contains(target)) return
  hideNow()
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') hideNow()
}

let offFollowChange: (() => void) | undefined

onMounted(() => {
  window.addEventListener('goose:user-card-show', show)
  document.addEventListener('pointerdown', onDocumentPointerDown)
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('scroll', hideNow, { passive: true })
  window.addEventListener('resize', hideNow)
  window.addEventListener('goose:page', hideNow)
  // 用户主页等处关注/取关后，同步纠正打开中的卡片与缓存条目
  offFollowChange = onFollowChange(({ userId, isFollowing: following }) => {
    const cached = cache.get(userId)
    if (cached) cached.isFollowing = following
    if (card.value?.userId === userId) isFollowing.value = following
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('goose:user-card-show', show)
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('scroll', hideNow)
  window.removeEventListener('resize', hideNow)
  window.removeEventListener('goose:page', hideNow)
  offFollowChange?.()
})

</script>

<template>
  <Teleport to="body">
    <Transition name="user-card-pop">
      <div
        v-if="visible"
        ref="cardEl"
        class="gf-menu-surface fixed z-[90] w-[min(20rem,calc(100vw-1.5rem))] p-3 text-base-content"
        :style="{ left: `${position.left}px`, top: `${position.top}px`, ...coverStyle }"
        @click.stop
      >
      <!-- 全卡背景蒙版：主题色半透明罩层，保证封面图上的文字对比度（better-colors） -->
      <div class="pointer-events-none absolute inset-0 rounded-[inherit] bg-base-100/80" aria-hidden="true" />
      <div class="relative">
      <div class="flex items-start gap-3">
        <!-- 头像单环：a 固定 56×56 圆环（flex 消除 inline-block 基线空隙，避免 ring 变椭圆） -->
        <a v-if="!isAccountClosed" :href="profileUrl" class="flex h-14 w-14 shrink-0 items-center justify-center rounded-full ring-2 ring-base-100">
          <UserAvatar :src="avatarUrl" :alt="username" :badge="wornBadge" size="medium" class="h-14 w-14 rounded-full" img-class="rounded-full" />
        </a>
        <span v-else class="flex h-14 w-14 shrink-0 items-center justify-center rounded-full ring-2 ring-base-100">
          <UserAvatar :src="avatarUrl" :alt="username" :badge="null" size="medium" class="h-14 w-14 rounded-full opacity-80 grayscale" img-class="rounded-full" />
        </span>
        <div class="min-w-0 flex-1">
          <div class="flex min-w-0 items-center gap-2">
            <a v-if="!isAccountClosed" :href="profileUrl" class="truncate text-base font-bold text-base-content hover:text-primary">{{ displayName }}</a>
            <span v-else class="truncate text-base font-bold text-base-content/70">{{ displayName }}</span>
            <span v-if="card?.isAdmin" class="gf-badge gf-badge-warning shrink-0 rounded text-[11px]">Admin</span>
            <span v-if="isAccountClosed" class="gf-badge gf-badge-muted shrink-0 rounded text-[11px]">{{ t('userCard.accountClosedBadge') }}</span>
          </div>
          <div class="mt-0.5 flex items-center gap-2 text-xs text-base-content/55">
            <template v-if="!isAccountClosed">
              <span class="truncate">@{{ username }}</span>
              <span v-if="card?.isOnline" class="inline-flex items-center gap-1 text-success">
                <Radio class="h-3 w-3" />
                {{ t('userCard.online') }}
              </span>
              <span v-else-if="card?.lastActiveTime">{{ t('userCard.activeAt', { time: timeAgo(card.lastActiveTime) }) }}</span>
            </template>
          </div>
        </div>
      </div>

        <Transition name="user-card-content" mode="out-in">
          <div v-if="loading" key="loading" class="mt-3 min-h-[164px]">
            <div class="space-y-2">
              <div class="h-4 w-full rounded bg-base-300" />
              <div class="h-4 w-3/4 rounded bg-base-300" />
            </div>
            <div class="mt-3 grid grid-cols-4 divide-x divide-line border-y border-line py-2">
              <div v-for="item in 4" :key="item" class="px-2 text-center">
                <div class="mx-auto h-4 w-7 rounded bg-base-300" />
                <div class="mx-auto mt-1 h-3 w-8 rounded bg-base-300" />
              </div>
            </div>
            <div class="mt-3 flex items-center justify-between gap-3">
              <div class="flex items-center gap-1.5 text-xs text-base-content/55">
                <Loader2 class="h-3.5 w-3.5 animate-spin" />
                {{ t('userCard.loading') }}
              </div>
              <div class="h-8 w-24 rounded-md bg-base-300" />
            </div>
          </div>
          <div v-else-if="error" key="error" class="gf-status-message gf-status-message-error mt-3 flex min-h-[164px] items-center">{{ error }}</div>
          <div v-else-if="isAccountClosed" key="closed" class="mt-3">
            <!-- 已注销用户：单独小资料卡，说明账号已注销而非「资料不可用」（better-ui / better-writing） -->
            <div class="flex min-h-[164px] flex-col items-center justify-center gap-3 rounded-[var(--gf-radius-field)] border border-dashed border-line bg-base-200/50 px-6 py-6 text-center">
              <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-base-300/80 text-icon-muted">
                <UserX class="h-6 w-6" aria-hidden="true" />
              </div>
              <div>
                <p class="text-sm font-semibold text-base-content/80">{{ t('userCard.accountClosedTitle') }}</p>
                <p class="mx-auto mt-1 max-w-[15rem] text-xs leading-5 text-base-content/55">{{ t('userCard.accountClosedDescription') }}</p>
              </div>
            </div>
          </div>
          <div v-else key="content">
        <p v-if="bioText" class="mt-3 line-clamp-2 text-sm leading-relaxed text-base-content/75">{{ bioText }}</p>

        <aside
          v-if="card?.signature"
          class="gf-profile-signature gf-user-card__signature"
          :aria-label="t('user.signatureLabel')"
        >
          <div class="gf-profile-signature__row">
            <Feather class="gf-profile-signature__icon gf-user-card__signature-icon" aria-hidden="true" />
            <p class="gf-profile-signature__text gf-user-card__signature-text">{{ card.signature }}</p>
          </div>
          <svg class="gf-profile-signature__squiggle" viewBox="0 0 100 8" preserveAspectRatio="none" aria-hidden="true">
            <path
              d="M2 5 C 10 0, 18 8, 26 5 S 42 8, 50 5 S 66 8, 74 5 S 90 8, 98 5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
            />
          </svg>
        </aside>

        <div v-if="visibleBadges.length" class="mt-3 flex gap-2">
          <span
            v-for="badge in visibleBadges"
            :key="badge.code"
            class="group relative flex h-8 w-8 shrink-0 items-center justify-center"
            tabindex="0"
            @mouseenter="activeBadgeCode = badge.code"
            @mouseleave="activeBadgeCode = ''"
            @focus="activeBadgeCode = badge.code"
            @blur="activeBadgeCode = ''"
          >
            <span
              class="flex h-8 w-8 items-center justify-center ring-1 ring-inset transition duration-150"
              :class="[badgeClass(badge.color, badge.level), activeBadgeCode === badge.code ? '-translate-y-0.5 scale-110 shadow-md' : 'shadow-none']"
              style="clip-path: polygon(25% 5%, 75% 5%, 100% 50%, 75% 95%, 25% 95%, 0 50%)"
            >
              <img :src="badgeIconURL(badge)" :alt="badge.name" class="h-4 w-4 object-contain" />
            </span>
            <span
              v-if="activeBadgeCode === badge.code"
              class="gf-tooltip pointer-events-none absolute left-1/2 top-full z-10 mt-2 w-max max-w-48 -translate-x-1/2 leading-5"
            >
              {{ badgeTooltip(badge) }}
            </span>
          </span>
        </div>

        <div class="mt-3 grid grid-cols-4 divide-x divide-line border-y border-line py-2">
          <div class="px-2 text-center">
            <div class="text-sm font-bold tabular-nums text-base-content">{{ formatNumber(card?.topicCount || 0) }}</div>
            <div class="mt-0.5 text-[11px] text-base-content/55">{{ t('userCard.stats.topics') }}</div>
          </div>
          <div class="px-2 text-center">
            <div class="text-sm font-bold tabular-nums text-base-content">{{ formatNumber(card?.replyCount || 0) }}</div>
            <div class="mt-0.5 text-[11px] text-base-content/55">{{ t('userCard.stats.replies') }}</div>
          </div>
          <div class="px-2 text-center">
            <div class="text-sm font-bold tabular-nums text-base-content">{{ formatNumber(card?.likeReceivedCount || 0) }}</div>
            <div class="mt-0.5 text-[11px] text-base-content/55">{{ t('userCard.stats.likes') }}</div>
          </div>
          <div class="px-2 text-center">
            <div class="text-sm font-bold tabular-nums text-base-content">{{ formatNumber(card?.followerCount || 0) }}</div>
            <div class="mt-0.5 text-[11px] text-base-content/55">{{ t('userCard.stats.followers') }}</div>
          </div>
        </div>

        <div v-if="externalLinks.length" class="mt-3 flex items-center gap-2 border-b border-line pb-3">
          <a
            v-for="link in externalLinks.slice(0, 8)"
            :key="`${link.key}-${link.url}`"
            :href="link.url"
            target="_blank"
            rel="noopener noreferrer"
            class="group relative inline-flex h-7 w-7 items-center justify-center rounded-md text-icon-muted transition hover:bg-base-200 hover:text-primary"
            :title="link.label"
            :aria-label="link.label"
          >
            <Bird v-if="link.key === 'website'" class="h-4 w-4" />
            <svg
              v-else-if="link.icon"
              class="h-4 w-4"
              viewBox="0 0 24 24"
              fill="currentColor"
              aria-hidden="true"
            >
              <path :d="link.icon.path" />
            </svg>
            <ExternalLink v-else class="h-4 w-4" />
            <span class="gf-tooltip pointer-events-none absolute bottom-full left-1/2 z-10 mb-2 max-w-40 -translate-x-1/2 truncate opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100">
              {{ link.label }}
            </span>
          </a>
        </div>

        <div class="mt-3 flex items-center justify-between gap-3">
          <!-- 加入日期：flex-1 独占剩余空间，不被按钮组挤压 -->
          <div class="min-w-0 flex-1">
            <span class="inline-flex items-center gap-1.5 text-xs text-base-content/55">
              <CalendarDays class="h-3.5 w-3.5 shrink-0" />
              <span class="truncate">{{ t('userCard.joinedAt', { date: card?.createdAt ? formatDate(card.createdAt) : '-' }) }}</span>
            </span>
          </div>
          <!-- 底部操作组：关注（主色文字按钮）+ 查看主页（次级图标按钮），宽度最小化 -->
          <div class="flex shrink-0 items-center gap-1.5">
            <button
              v-if="card && !card.isSelf"
              type="button"
              class="gf-button gf-button-sm"
              :class="isFollowing ? 'bg-base-300 text-base-content hover:bg-base-300' : 'bg-primary text-primary-content hover:bg-primary'"
              :disabled="followLoading"
              @click="toggleFollow"
            >
              <Loader2 v-if="followLoading" class="h-3.5 w-3.5 animate-spin" />
              {{ isFollowing ? t('userCard.following') : t('userCard.follow') }}
            </button>
            <a
              :href="profileUrl"
              class="gf-icon-button h-8 w-8"
              :title="t('userCard.viewProfile')"
              :aria-label="t('userCard.viewProfile')"
            >
              <ExternalLink class="h-4 w-4" />
            </a>
          </div>
        </div>
        <!-- 关注失败内联提示（issue #593）：失败保持原状态并给出可操作文案，不再静默吞错 -->
        <p v-if="followError" role="alert" class="mt-2 text-xs text-error">{{ followError }}</p>
          </div>
        </Transition>
      </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* 卡片内签名：紧凑手帐式（复用主页签名视觉语言，缩小到卡片密度） */
.gf-user-card__signature {
  margin-top: 0.5rem;
}

.gf-user-card__signature-icon {
  height: 0.8rem;
  width: 0.8rem;
}

.gf-user-card__signature-text {
  font-size: 0.75rem;
}

.gf-user-card__signature-text:lang(en),
.gf-user-card__signature-text:lang(de) {
  font-size: 0.75rem;
}
</style>
