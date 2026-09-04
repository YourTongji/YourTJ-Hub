<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import {
  Bell,
  BookOpen,
  CalendarRange,
  FileText,
  Flame,
  Heart,
  Inbox,
  Library,
  Link,
  MessageCircle,
  Languages,
  LogOut,
  Menu,
  Monitor,
  Moon,
  Palette,
  PenSquare,
  Scale,
  Sun,
  TrendingUp,
  Search,
  Settings,
  Shield,
  GraduationCap,
  UserRound,
} from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import GlobalFlash from './GlobalFlash.vue'
import { setLocale, supportedLocales, type Locale } from '@/runtime/i18n'
import { queueFlashMessage } from '@/runtime/flash-message'
import { useSiteTheme, setThemePreference, type ThemePreference } from '@/runtime/site-theme'
import { safeUrl } from '@/runtime/safe-url'
import { useNavigationState } from '@/runtime/navigation-state'
import { useUnreadStatus } from '@/runtime/unread-status'
import { useWikiSearchPanel } from '@/runtime/use-wiki-search'
import type { LayoutPayload } from '@gooseforum/client'
import type { UserCardShowDetail } from '@/runtime/user-card-events'
import UserAvatar from './UserAvatar.vue'
import type UserCardComponent from './UserCard.vue'
import WikiSidebar from './WikiSidebar.vue'
import WikiSearchPanel from './WikiSearchPanel.vue'
import PublishMenu from './PublishMenu.vue'
import QuickPublishModal from './QuickPublishModal.vue'

import { useShellState } from '@/runtime/shell-state'

const route = useRoute()
const shellState = useShellState()
const isPublishPage = computed(() => route?.path === '/publish' || route?.name === 'publish')
const isTopicPage = computed(() => {
  if (shellState.isTopicPage) return true
  const path = route?.path || ''
  return path.startsWith('/p/post') || path.startsWith('/topics') || path.startsWith('/p/topic')
})

const props = defineProps<{
  layout: LayoutPayload
  rail?: boolean
  headerTitle?: string
  headerTags?: Array<{ id: number | string; name: string; color?: string }>
  showHeaderTitle?: boolean
}>()

interface SidebarNavItem {
  key: string
  label: string
  i18nLabel?: string
  url: string
  active: boolean
}

interface SidebarGroupItem {
  key: string
  title: string
  i18nLabel?: string
  items: SidebarNavItem[]
}
interface SidebarCategoryItem extends SidebarNavItem {
  id: number
  color: string
}


interface SidebarSection {
  key: string
  /** 有标题时渲染分组标题；无标题（浏览组）时直接平铺，保持视觉主导。 */
  title?: string
  items: SidebarNavItem[]
}

const MobileDrawer = defineAsyncComponent(() => import('./MobileDrawer.vue'))
const UserCard = shallowRef<typeof UserCardComponent | null>(null)
const drawerOpen = ref(false)
const headerElevated = ref(false)
const themeMenuOpen = ref(false)
const langMenuOpen = ref(false)
const userMenuOpen = ref(false)
const closeTimers: Record<'theme' | 'lang' | 'user', number | undefined> = {
  theme: undefined,
  lang: undefined,
  user: undefined,
}
const { navigating } = useNavigationState()
const { t, te, locale } = useI18n()
const { isDark, preference } = useSiteTheme()
const unreadStatus = useUnreadStatus()

const themeOptions = computed(() => [
  { value: 'auto' as const, label: t('shell.themeAuto'), icon: Monitor },
  { value: 'light' as const, label: t('shell.themeLight'), icon: Sun },
  { value: 'dark' as const, label: t('shell.themeDark'), icon: Moon },
])

function selectTheme(preference: ThemePreference) {
  setThemePreference(preference)
  themeMenuOpen.value = false
}

const hasUnreadNotification = computed(() => unreadStatus.notifications.value)
const hasUnreadMessage = computed(() => unreadStatus.messages.value)
const hasModerationReports = computed(() => unreadStatus.moderationReports.value)
const notificationTitle = computed(() => unreadStatus.notificationMessage.value)
const asArray = <T>(value: T[] | null | undefined): T[] => (Array.isArray(value) ? value : [])
const activeSidebarKey = computed(() => props.layout.sidebar.activeKey || 'topics')
const isWikiMode = computed(() => props.layout.sidebar.mode === 'wiki')
const wikiTree = computed(() => props.layout.sidebar.wikiTree || [])
// wiki 局内搜索面板是全局单例（桌面侧栏胶囊 / 移动 header 按钮 / ⌘K、/ 快捷键共用）。
// 面板打开时收起移动抽屉，避免面板关闭后用户仍停留在抽屉层。
const { panelOpen: wikiSearchOpen, openPanel: openWikiSearch } = useWikiSearchPanel()
watch(wikiSearchOpen, (open) => {
  if (open) drawerOpen.value = false
})

// 浏览组（排序入口）：默认平铺在侧栏顶部，不加重重复述首页 tab 的分组感。
const browseItems = computed<SidebarNavItem[]>(() => [
  sidebarItem('topics', t('shell.nav.topics'), '/'),
  sidebarItem('hot', t('shell.nav.hot'), '/?sort=hot'),
  sidebarItem('popular', t('shell.nav.popular'), '/?sort=popular'),
])

// 功能组：站点能力页。
const functionItems = computed<SidebarNavItem[]>(() => [
  sidebarItem('courses', t('shell.nav.courses'), '/courses'),
  sidebarItem('schedule', t('shell.nav.schedule'), '/schedule'),
  sidebarItem('wiki', t('shell.nav.wiki'), '/wiki'),
])

// 个人组：登录后的个人内容入口（私信/通知已上移 navbar，不再重复）。
const personalItems = computed<SidebarNavItem[]>(() => {
  if (!props.layout.viewer.isAuthenticated) return []
  const items: SidebarNavItem[] = [sidebarItem('drafts', t('shell.nav.drafts'), '/drafts')]
  items.push(...serverSidebarItems(props.layout.sidebar.main))
  return items
})

// 管理组：按权限可见。
const adminItems = computed<SidebarNavItem[]>(() => {
  const items: SidebarNavItem[] = []
  if (props.layout.viewer.isModerator) {
    items.push(sidebarItem('moderation', t('shell.nav.moderation'), '/moderation'))
  }
  // 课评审核入口：CourseManager 权限（Admin 通过 adminPermissions 全量包含，id=6）。
  if (props.layout.viewer.isAuthenticated && props.layout.viewer.adminPermissions.includes(6)) {
    items.push(sidebarItem('courseReviews', t('shell.nav.courseReviews'), '/moderation/course-reviews'))
    items.push(sidebarItem('courseManage', t('shell.nav.courseManage'), '/moderation/courses'))
  }
  return items
})

// 侧栏分区：浏览组无标题（就是首页的三个视角），其余组带标题归组。
const sidebarSections = computed<SidebarSection[]>(() => {
  const sections: SidebarSection[] = [
    { key: 'browse', items: browseItems.value },
    { key: 'function', title: t('shell.groupFunction'), items: functionItems.value },
  ]
  if (personalItems.value.length) {
    sections.push({ key: 'personal', title: t('shell.groupPersonal'), items: personalItems.value })
  }
  if (adminItems.value.length) {
    sections.push({ key: 'admin', title: t('shell.groupAdmin'), items: adminItems.value })
  }
  return sections
})

// 移动抽屉沿用同一分区，浏览组在抽屉中补标题以维持结构可读。
const drawerSections = computed<SidebarSection[]>(() =>
  sidebarSections.value.map((section) =>
    section.key === 'browse' ? { ...section, title: t('shell.groupBrowse') } : section,
  ),
)


// 分类列表：回归原版侧栏「分类」分区（色点 + 名称），首页不再叠加横滚分类 rail。
const categoryItems = computed<SidebarCategoryItem[]>(() =>
  asArray(props.layout.sidebar.categories).map((category) => {
    const key = `category_${category.id}`
    return {
      key,
      id: category.id,
      label: category.label,
      url: safeUrl(category.url, 'site-link'),
      color: category.color,
      active: activeSidebarKey.value === key,
    }
  }),
)
const resourceItems = computed<SidebarNavItem[]>(() => [
  sidebarItem('links', t('shell.nav.links'), '/links'),
  sidebarItem('sponsors', t('shell.nav.sponsors'), '/sponsors'),
  ...serverSidebarItems(props.layout.sidebar.resources),
])
const sidebarGroups = computed<SidebarGroupItem[]>(() =>
  asArray(props.layout.sidebar.groups)
    .map((group) => ({
      key: group.key,
      title: displayNavLabel(group),
      i18nLabel: group.i18nLabel,
      items: serverSidebarItems(group.items),
    }))
    .filter((group) => group.title && group.items.length > 0),
)
const headerResourceItems = computed(() => {
  const configured = serverSidebarItems(props.layout.header)
  return configured.length > 0
    ? configured
    : ['sponsors', 'links']
      .map((key) => resourceItems.value.find((item) => item.key === key))
      .filter((item): item is NonNullable<typeof item> => Boolean(item))
})
const footerLinks = computed(() => asArray(props.layout.footer.links).map((link) => ({
  ...link,
  url: safeUrl(link.url, 'site-link'),
})))
const footerPrimary = computed(() => asArray(props.layout.footer.primary))
const hasFooter = computed(() => footerLinks.value.length > 0 || footerPrimary.value.length > 0)
const safeFooter = computed(() => ({ links: footerLinks.value, primary: footerPrimary.value }))
const brandType = computed(() => props.layout.site.brandType || 'default')
const brandText = computed(() => props.layout.site.brandText || props.layout.site.name)
const brandImage = computed(() => safeUrl(props.layout.site.brandImage, 'image'))
const hasHeaderTitle = computed(() => Boolean(props.showHeaderTitle && props.headerTitle))
const searchQuery = ref('')
const searchInput = ref<HTMLInputElement | null>(null)
const sidebarIconMap = {
  topics: MessageCircle,
  hot: Flame,
  popular: TrendingUp,
  courses: BookOpen,
  schedule: CalendarRange,
  wiki: Library,
  messages: Inbox,
  notifications: Bell,
  drafts: FileText,
  moderation: Scale,
  courseReviews: GraduationCap,
  courseManage: BookOpen,
  links: Link,
  sponsors: Heart,
} as const
let userCardLoading: Promise<void> | undefined

watch(
  () => props.layout.sidebar.activeKey,
  () => {
    drawerOpen.value = false
    langMenuOpen.value = false
    userMenuOpen.value = false
  },
)

onMounted(() => {
  if (props.layout.viewer.isAuthenticated) {
    unreadStatus.startPolling(props.layout.unread)
  }
  updateHeaderElevated()
  window.addEventListener('scroll', updateHeaderElevated, { passive: true })
  window.addEventListener('goose:user-card-show', ensureUserCardForEvent)
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', updateHeaderElevated)
  window.removeEventListener('goose:user-card-show', ensureUserCardForEvent)
})

watch(
  () => props.layout.unread,
  (unread) => {
    if (props.layout.viewer.isAuthenticated) {
      unreadStatus.applyUnread(unread)
    }
  },
  { deep: true },
)

function setLang(lang: Locale) {
  setLocale(lang)
  langMenuOpen.value = false
}

function openDrawer() {
  drawerOpen.value = true
}

function closeDrawer() {
  drawerOpen.value = false
}

function submitSearch() {
  const query = searchQuery.value.trim()
  if (!query) {
    searchInput.value?.focus()
    return
  }
  // 复用全局 SPA 导航（router 拦截 a[href] 点击）；绕过则回退整页跳转。
  const link = document.createElement('a')
  link.href = `/search?q=${encodeURIComponent(query)}`
  link.click()
}

async function logout() {
  const res = await fetch('/api/logout', { method: 'POST' })
  if (res.ok) {
    const data = await res.json().catch(() => null)
    if (data && typeof data.code === 'number' && data.code !== 0) {
      // 服务端撤销失败（session.revoke.failed）：cookie 已清除但会话行可能仍在，
      // 跨刷新提示用户登录态可能依然有效。
      queueFlashMessage(t('shell.logoutFailed'), 'error')
    }
  }
  window.location.reload()
}

function navIcon(item: SidebarNavItem) {
  return sidebarIconMap[item.key as keyof typeof sidebarIconMap] || Link
}

function displayNavLabel(item: { label?: string; title?: string; i18nLabel?: string }) {
  return item.i18nLabel && te(item.i18nLabel) ? t(item.i18nLabel) : item.label || item.title || ''
}

function sidebarItem(key: string, label: string, url: string): SidebarNavItem {
  return {
    key,
    label,
    url,
    active: activeSidebarKey.value === key,
  }
}

function serverSidebarItems(items: typeof props.layout.sidebar.main): SidebarNavItem[] {
  return asArray(items).map((item) => ({
    key: item.key,
    label: displayNavLabel(item),
    i18nLabel: item.i18nLabel,
    url: safeUrl(item.url, 'site-link'),
    active: activeSidebarKey.value === item.key,
  }))
}

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function updateHeaderElevated() {
  headerElevated.value = window.scrollY > 8
}

function setHoverMenu(menu: 'theme' | 'lang' | 'user', open: boolean) {
  window.clearTimeout(closeTimers[menu])
  closeTimers[menu] = undefined
  if (menu === 'theme') themeMenuOpen.value = open
  else if (menu === 'lang') langMenuOpen.value = open
  else userMenuOpen.value = open
}

function closeHoverMenuSoon(menu: 'theme' | 'lang' | 'user') {
  window.clearTimeout(closeTimers[menu])
  closeTimers[menu] = window.setTimeout(() => {
    if (menu === 'theme') themeMenuOpen.value = false
    else if (menu === 'lang') langMenuOpen.value = false
    else userMenuOpen.value = false
  }, 120)
}

function ensureUserCardForEvent(event: Event) {
  if (UserCard.value) return
  const detail = (event as CustomEvent<UserCardShowDetail>).detail
  if (!detail?.user?.id || !detail.target) return
  void loadUserCard().then(async () => {
    await nextTick()
    window.dispatchEvent(new CustomEvent<UserCardShowDetail>('goose:user-card-show', { detail }))
  })
}

async function loadUserCard() {
  if (UserCard.value) return
  if (!userCardLoading) {
    userCardLoading = import('./UserCard.vue')
      .then((module) => {
        UserCard.value = module.default
      })
      .finally(() => {
        userCardLoading = undefined
      })
  }
  await userCardLoading
}
</script>

<template>
  <div class="min-h-screen bg-base-200 text-base-content">
    <div
      v-show="navigating"
      class="fixed left-0 top-0 z-[100] h-0.5 w-full overflow-hidden bg-info/10"
    >
      <div class="h-full w-24 animate-[gf-loading-bar_1s_ease-in-out_infinite] rounded-r-full bg-primary sm:w-36" />
    </div>

    <header
      class="sticky top-0 z-50 border-b border-line bg-base-100/95 backdrop-blur-sm transition-[background-color,border-color,box-shadow,backdrop-filter] duration-200"
      :class="headerElevated
        ? 'sm:shadow-[0_1px_10px_rgb(15_23_42/0.04)]'
        : 'sm:shadow-none'"
    >
      <div class="mx-auto grid h-14 w-full max-w-[1600px] grid-cols-[minmax(0,1fr)_auto] items-center gap-2 px-3 sm:h-16 sm:gap-4 sm:px-5 md:grid-cols-[auto_minmax(0,1fr)_auto] lg:gap-8 lg:px-8">
        <div class="flex min-w-0 items-center gap-2 sm:gap-4 lg:gap-8">
          <button
            type="button"
            class="gf-icon-button h-10 w-10 rounded-full active:scale-[0.96] motion-reduce:active:scale-100 lg:hidden"
            :aria-label="t('shell.openMenu')"
            @click="openDrawer"
          >
            <Menu class="h-5 w-5" />
          </button>
          <button
            v-if="hasHeaderTitle"
            type="button"
            class="flex min-w-0 flex-1 flex-col items-start justify-center gap-0.5 self-stretch text-left transition md:hidden"
            @click="scrollToTop"
          >
            <span class="block max-w-full truncate text-lg font-semibold leading-6 text-base-content hover:text-primary">
              {{ headerTitle }}
            </span>
            <span
              v-if="headerTags?.length"
              class="flex max-w-full items-center gap-1 overflow-hidden text-[11px] font-medium leading-4 text-base-content/55"
            >
              <span
                v-for="tag in headerTags"
                :key="tag.id"
                class="inline-flex min-w-0 shrink-0 items-center gap-1"
              >
                <span
                  class="h-1.5 w-1.5 rounded-[2px]"
                  :style="{ backgroundColor: tag.color || 'var(--gf-color-base-content)' }"
                />
                <span class="max-w-20 truncate">{{ tag.name }}</span>
              </span>
            </span>
          </button>
          <a
            href="/"
            class="min-w-0 items-center gap-2"
            :class="hasHeaderTitle ? 'hidden md:flex' : 'flex'"
          >
            <img
              v-if="brandType === 'image' && brandImage"
              :src="brandImage"
              :alt="layout.site.name"
              class="h-8 w-auto max-w-32 shrink-0 object-contain sm:max-w-40 sm:h-9"
            />
            <span
              v-else-if="brandType === 'text'"
              class="max-w-36 truncate text-xl font-semibold tracking-tighter text-primary sm:max-w-44 sm:text-2xl md:max-w-none"
            >
              {{ brandText }}
            </span>
            <img
              v-else
              src="/static/pic/brand-default.webp"
              :alt="layout.site.name"
              class="h-8 w-auto max-w-32 shrink-0 object-contain sm:max-w-40 sm:h-9"
            />
          </a>
          <button
            v-if="hasHeaderTitle"
            type="button"
            class="hidden min-w-0 items-center gap-3 text-left transition md:flex"
            @click="scrollToTop"
          >
            <span class="block max-w-[280px] truncate text-lg font-semibold leading-6 text-base-content hover:text-primary">
              {{ headerTitle }}
            </span>
            <span
              v-if="headerTags?.length"
              class="flex min-w-0 items-center gap-1.5 overflow-hidden text-[11px] font-medium leading-4 text-base-content/55"
            >
              <span
                v-for="tag in headerTags"
                :key="tag.id"
                class="inline-flex min-w-0 shrink-0 items-center gap-1"
              >
                <span
                  class="h-1.5 w-1.5 rounded-[2px]"
                  :style="{ backgroundColor: tag.color || 'var(--gf-color-base-content)' }"
                />
                <span class="max-w-24 truncate">{{ tag.name }}</span>
              </span>
            </span>
          </button>
          <nav
            v-if="!hasHeaderTitle"
            class="hidden items-center gap-1 md:flex"
            aria-label="Header navigation"
          >
            <a
              v-for="item in headerResourceItems"
              :key="item.key"
              :href="item.url"
              class="inline-flex h-7 items-center rounded-md px-2 text-sm font-medium text-base-content/75 transition-colors duration-150 hover:bg-base-300 hover:text-base-content"
            >
              {{ item.label }}
            </a>
          </nav>
        </div>

        <div class="hidden min-w-0 md:block">
          <!-- 桌面居中搜索（Figma v3 Header）：胶囊搜索栏，回车走 SPA 跳转 /search?q= -->
          <form
            role="search"
            class="relative mx-auto w-full max-w-[480px]"
            @submit.prevent="submitSearch"
          >
            <Search
              class="pointer-events-none absolute left-3.5 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-base-content/45"
              aria-hidden="true"
            />
            <input
              ref="searchInput"
              v-model="searchQuery"
              type="search"
              name="q"
              autocomplete="off"
              :placeholder="t('searchPage.inputPlaceholder')"
              :aria-label="t('shell.search')"
              class="h-10 w-full rounded-full border border-line bg-base-200 pl-10 pr-4 text-sm text-base-content transition-[border-color,background-color,box-shadow] duration-150 placeholder:text-base-content/50 hover:border-base-content/25 focus:border-primary/60 focus:bg-base-100 focus:outline-none focus:ring-2 focus:ring-primary/15"
            />
          </form>
        </div>

        <div class="flex shrink-0 items-center justify-end gap-0.5 sm:gap-1">
          <template v-if="layout.viewer.isAuthenticated">
            <!-- 发布 CTA：sm+ 常驻 navbar 呼出菜单；<sm 由右下角 FAB 承担，不占 navbar 空间 -->
            <PublishMenu variant="navbar" />

            <!-- 通知 / 私信：从头像菜单提升为直达图标按钮，36px 热区（display 覆盖需用 sm:inline-grid，保住 place-items-center 居中） -->
            <a
              href="/notifications"
              class="gf-icon-button relative h-9 w-9 rounded-full active:scale-[0.96] motion-reduce:active:scale-100"
              :aria-label="t('shell.nav.notifications')"
              :title="notificationTitle || t('shell.nav.notifications')"
            >
              <Bell class="h-5 w-5" />
              <span
                v-show="hasUnreadNotification"
                class="absolute right-1 top-1 h-2.5 w-2.5 rounded-full bg-error ring-2 ring-base-100"
                aria-hidden="true"
              />
            </a>
            <a
              href="/messages"
              class="gf-icon-button relative hidden h-9 w-9 rounded-full active:scale-[0.96] motion-reduce:active:scale-100 sm:inline-grid"
              :aria-label="t('shell.nav.messages')"
              :title="t('shell.nav.messages')"
            >
              <Inbox class="h-5 w-5" />
              <span
                v-show="hasUnreadMessage"
                class="absolute right-1 top-1 h-2.5 w-2.5 rounded-full bg-error ring-2 ring-base-100"
                aria-hidden="true"
              />
            </a>
          </template>

          <!-- 搜索入口（<md）：wiki 模式打开局内搜索面板（与侧栏胶囊/⌘K 同一面板）；
               论坛模式跳转全局搜索页。同一时点只呈现一个搜索入口，避免双搜索图标。 -->
          <a
            v-if="!isWikiMode"
            href="/search"
            class="gf-icon-button h-9 w-9 rounded-full active:scale-[0.96] motion-reduce:active:scale-100 md:hidden"
            :aria-label="t('shell.search')"
            :title="t('shell.search')"
          >
            <Search class="h-5 w-5" />
          </a>
          <button
            v-else
            type="button"
            class="gf-icon-button h-9 w-9 rounded-full active:scale-[0.96] motion-reduce:active:scale-100 md:hidden"
            :aria-label="t('wikiSearch.openSearch')"
            :title="t('wikiSearch.openSearch')"
            @click="openWikiSearch"
          >
            <Search class="h-5 w-5" />
          </button>

          <div
            class="relative hidden sm:block"
            @mouseenter="setHoverMenu('theme', true)"
            @mouseleave="closeHoverMenuSoon('theme')"
            @focusin="setHoverMenu('theme', true)"
            @focusout="closeHoverMenuSoon('theme')"
          >
            <button
              type="button"
              class="gf-icon-button h-9 w-9 rounded-full active:scale-[0.96] motion-reduce:active:scale-100"
              :aria-label="t('shell.switchTheme')"
              :title="t('shell.switchTheme')"
              :aria-expanded="themeMenuOpen"
              @click="themeMenuOpen = !themeMenuOpen"
            >
              <component :is="themeOptions.find(opt => opt.value === preference)?.icon ?? Monitor" class="h-5 w-5" />
            </button>
            <Transition name="gf-menu">
              <div
                v-if="themeMenuOpen"
                class="absolute right-0 top-full z-[70] w-36 pt-2"
              >
                <div class="gf-menu-surface overflow-hidden py-1">
                  <button
                    v-for="option in themeOptions"
                    :key="option.value"
                    class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors duration-150 hover:bg-base-200"
                    :class="preference === option.value ? 'font-semibold text-primary' : 'text-base-content/75'"
                    type="button"
                    @click="selectTheme(option.value)"
                  >
                    <component :is="option.icon" class="h-4 w-4" />
                    {{ option.label }}
                  </button>
                </div>
              </div>
            </Transition>
          </div>

          <div
            class="relative hidden sm:block"
            @mouseenter="setHoverMenu('lang', true)"
            @mouseleave="closeHoverMenuSoon('lang')"
            @focusin="setHoverMenu('lang', true)"
            @focusout="closeHoverMenuSoon('lang')"
          >
            <button
              type="button"
              class="gf-icon-button h-9 w-9 rounded-full active:scale-[0.96] motion-reduce:active:scale-100"
              :aria-label="t('shell.switchLanguage')"
              :title="t('shell.switchLanguage')"
              :aria-expanded="langMenuOpen"
              aria-haspopup="menu"
              @click="langMenuOpen = !langMenuOpen"
            >
              <Languages class="h-5 w-5" />
            </button>
            <Transition name="gf-menu">
              <div
                v-if="langMenuOpen"
                class="absolute right-0 top-full z-[70] w-36 pt-2"
              >
                <div class="gf-menu-surface overflow-hidden py-1" role="menu">
                  <button
                    v-for="item in supportedLocales"
                    :key="item"
                    class="block w-full px-3 py-1.5 text-left text-sm transition-colors duration-150 hover:bg-base-200"
                    :class="locale === item ? 'font-semibold text-primary' : 'text-base-content/75'"
                    type="button"
                    role="menuitem"
                    @click="setLang(item)"
                  >
                    {{ t(`locale.${item}`) }}
                  </button>
                </div>
              </div>
            </Transition>
          </div>

          <template v-if="layout.viewer.isAuthenticated">
            <div
              class="relative"
              @mouseenter="setHoverMenu('user', true)"
              @mouseleave="closeHoverMenuSoon('user')"
              @focusin="setHoverMenu('user', true)"
              @focusout="closeHoverMenuSoon('user')"
            >
              <button
                type="button"
                class="relative ml-1 flex h-10 w-10 items-center justify-center rounded-full transition-colors duration-150 hover:bg-base-300 active:scale-[0.96] motion-reduce:active:scale-100"
                :aria-label="t('shell.userMenu')"
                :aria-expanded="userMenuOpen"
                aria-haspopup="menu"
              >
                <UserAvatar :src="layout.viewer.avatarUrl" :alt="layout.viewer.username" class="h-9 w-9 rounded-full object-cover ring-1 ring-line/80" />
              </button>
              <Transition name="gf-menu">
                <div
                  v-if="userMenuOpen"
                  class="absolute right-0 top-full z-[70] w-56 pt-2"
                >
                  <div class="gf-menu-surface overflow-hidden">
                    <div class="border-b border-line/70 px-3 py-2.5">
                      <div class="truncate text-sm font-semibold text-base-content">{{ layout.viewer.username }}</div>
                    </div>
                    <div class="py-1">
                      <a :href="`/u/${layout.viewer.id}`" class="gf-menu-item">
                        <UserRound class="h-4 w-4 text-icon-muted" /> {{ t('shell.profile') }}
                      </a>
                      <!-- 移动端从 navbar 收进的入口：sm 起隐藏（navbar 已有直达按钮） -->
                      <a href="/notifications" class="gf-menu-item sm:hidden">
                        <Bell class="h-4 w-4 text-icon-muted" />
                        <span class="min-w-0 flex-1">{{ t('shell.nav.notifications') }}</span>
                        <span v-show="hasUnreadNotification" class="h-2 w-2 rounded-full bg-error" />
                      </a>
                      <a href="/messages" class="gf-menu-item sm:hidden">
                        <Inbox class="h-4 w-4 text-icon-muted" />
                        <span class="min-w-0 flex-1">{{ t('shell.nav.messages') }}</span>
                        <span v-show="hasUnreadMessage" class="h-2 w-2 rounded-full bg-error" />
                      </a>
                      <div class="sm:hidden">
                        <div class="px-3 pb-1 pt-1.5 text-[10px] font-bold uppercase tracking-wide text-base-content/55">
                          {{ t('shell.switchTheme') }}
                        </div>
                        <button
                          v-for="option in themeOptions"
                          :key="option.value"
                          class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors duration-150 hover:bg-base-200"
                          :class="preference === option.value ? 'font-semibold text-primary' : 'text-base-content/75'"
                          type="button"
                          @click="selectTheme(option.value)"
                        >
                          <component :is="option.icon" class="h-4 w-4" />
                          <span>{{ option.label }}</span>
                        </button>
                      </div>
                      <div class="sm:hidden">
                        <div class="px-3 pb-1 pt-1.5 text-[10px] font-bold uppercase tracking-wide text-base-content/55">
                          {{ t('shell.switchLanguage') }}
                        </div>
                        <button
                          v-for="item in supportedLocales"
                          :key="item"
                          class="block w-full px-3 py-1.5 text-left text-sm transition-colors duration-150 hover:bg-base-200"
                          :class="locale === item ? 'font-semibold text-primary' : 'text-base-content/75'"
                          type="button"
                          @click="setLang(item)"
                        >
                          {{ t(`locale.${item}`) }}
                        </button>
                      </div>
                      <a href="/drafts" class="gf-menu-item">
                        <FileText class="h-4 w-4 text-icon-muted" /> {{ t('shell.nav.drafts') }}
                      </a>
                    </div>
                    <div class="border-t border-line/70 py-1">
                      <a href="/settings" class="gf-menu-item">
                        <Settings class="h-4 w-4 text-icon-muted" /> {{ t('shell.settings') }}
                      </a>
                      <a v-if="layout.viewer.canAccessAdmin" href="/theme-preview" class="gf-menu-item">
                        <Palette class="h-4 w-4 text-icon-muted" /> {{ t('shell.themePreview') }}
                      </a>
                      <a v-if="layout.viewer.canAccessAdmin" href="/admin" class="gf-menu-item-warning">
                        <Shield class="h-4 w-4" /> {{ t('shell.admin') }}
                      </a>
                    </div>
                    <div class="border-t border-line/70 py-1">
                      <button class="gf-menu-item-danger" type="button" @click="logout">
                        <LogOut class="h-4 w-4" /> {{ t('shell.logout') }}
                      </button>
                    </div>
                  </div>
                </div>
              </Transition>
            </div>
          </template>
          <template v-else>
            <a href="/login" class="rounded-md px-3 py-2 text-sm font-medium text-base-content/75 hover:bg-base-300">{{ t('shell.login') }}</a>
            <a href="/login?register=true" class="gf-button gf-button-md gf-button-neutral hidden sm:inline-flex">{{ t('shell.register') }}</a>
          </template>
        </div>
      </div>
    </header>

    <GlobalFlash />

    <main
      class="gf-shell-main mx-auto grid w-full max-w-[1600px] grid-cols-1 gap-0 px-0 py-0 sm:gap-3 sm:px-5 sm:py-3 lg:grid-cols-[210px_minmax(0,1fr)] lg:px-8 xl:grid-cols-[224px_minmax(0,1fr)]"
      :class="{ 'xl:grid-cols-[224px_minmax(0,1fr)_280px]': rail }"
    >
      <aside class="gf-scrollbar-none sticky top-16 -my-3 hidden h-[calc(100vh-4rem)] overflow-y-auto self-start lg:block" aria-label="Sidebar">
        <WikiSidebar v-if="isWikiMode" :tree="wikiTree" />
        <nav v-else class="py-3">
          <div class="pb-2">
            <template v-for="(section, sectionIndex) in sidebarSections" :key="section.key">
              <div
                v-if="section.title"
                class="mb-1 px-2 text-[10px] font-bold uppercase tracking-wide text-base-content/55"
                :class="sectionIndex > 0 ? 'mt-2' : ''"
              >
                {{ section.title }}
              </div>
              <div :class="section.title ? 'space-y-px' : 'space-y-0.5'">
                <a
                  v-for="item in section.items"
                  :key="item.key"
                  :href="item.url"
                  class="flex items-center gap-2 rounded-md px-2 text-[13px] font-medium transition-colors duration-150"
                  :class="[
                    section.title ? 'h-7' : 'h-8',
                    item.active ? 'bg-info/10 text-primary' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content',
                  ]"
                >
                  <component
                    :is="navIcon(item)"
                    v-if="navIcon(item)"
                    class="h-4 w-4 shrink-0"
                    aria-hidden="true"
                  />
                  <span class="min-w-0 flex-1 truncate">{{ item.label }}</span>
                  <span
                    v-if="item.key === 'moderation' && hasModerationReports"
                    class="h-2 w-2 shrink-0 rounded-full bg-error/100"
                    aria-hidden="true"
                  />
                </a>
              </div>
            </template>

            <div v-if="resourceItems.length" class="mt-2">
              <div class="mb-1 px-2 text-[10px] font-bold uppercase tracking-wide text-base-content/55">{{ t('shell.resources') }}</div>
              <div class="space-y-px">
                <a
                  v-for="item in resourceItems"
                  :key="item.key"
                  :href="item.url"
                  class="flex h-7 items-center gap-2 rounded-md px-2 text-[13px] font-medium transition-colors duration-150"
                  :class="item.active ? 'bg-info/10 text-primary' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
                >
                  <component
                    :is="navIcon(item)"
                    v-if="navIcon(item)"
                    class="h-4 w-4 shrink-0"
                    aria-hidden="true"
                  />
                  <span class="truncate">{{ item.label }}</span>
                </a>
              </div>
            </div>

            <div
              v-for="group in sidebarGroups"
              :key="group.key"
              class="mt-2"
            >
              <div class="mb-1 px-2 text-[10px] font-bold uppercase tracking-wide text-base-content/55">{{ group.title }}</div>
              <div class="space-y-px">
                <a
                  v-for="item in group.items"
                  :key="item.key"
                  :href="item.url"
                  class="flex h-7 items-center gap-2 rounded-md px-2 text-[13px] font-medium transition-colors duration-150"
                  :class="item.active ? 'bg-info/10 text-primary' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
                >
                  <component
                    :is="navIcon(item)"
                    v-if="navIcon(item)"
                    class="h-4 w-4 shrink-0"
                    aria-hidden="true"
                  />
                  <span class="truncate">{{ item.label }}</span>
                </a>
              </div>
            </div>

            <div v-if="categoryItems.length" class="mt-2">
              <div class="mb-1 px-2 text-[10px] font-bold uppercase tracking-wide text-base-content/55">{{ t('shell.categories') }}</div>
              <div class="space-y-px">
                <a
                  v-for="category in categoryItems"
                  :key="category.key"
                  :href="category.url"
                  class="flex h-7 items-center gap-2 rounded-md px-2 text-[13px] font-medium transition-colors duration-150"
                  :class="category.active ? 'bg-base-300 text-base-content' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
                  :aria-current="category.active ? 'page' : undefined"
                >
                  <span class="h-2 w-2 rounded-[3px]" :style="{ backgroundColor: category.color }" />
                  <span class="truncate">{{ category.label }}</span>
                </a>
              </div>
            </div>
          </div>

          <footer v-if="hasFooter" class="mt-0 px-2 pt-0.5 text-xs leading-5 text-base-content/75">
            <div v-if="footerLinks.length" class="flex flex-wrap items-center gap-x-3 gap-y-0.5">
              <template v-for="link in footerLinks" :key="`${link.name}-${link.url}`">
                <a
                  v-if="link.url"
                  :href="link.url"
                  class="inline-flex min-h-5 items-center rounded hover:text-primary"
                >
                  {{ link.name }}
                </a>
                <span v-else class="inline-flex min-h-5 items-center rounded">{{ link.name }}</span>
              </template>
            </div>
            <div v-if="footerPrimary.length" class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-base-content/75">
              <span
                v-for="item in footerPrimary"
                :key="item"
                class="inline-flex min-h-5 items-center rounded"
              >
                {{ item }}
              </span>
            </div>
          </footer>
        </nav>
      </aside>

      <section class="gf-shell-content min-w-0">
        <slot />
      </section>

      <aside v-if="rail" id="goose-shell-rail" class="hidden min-w-0 xl:block">
        <slot name="rail" />
      </aside>

      <section
        id="goose-shell-wide-content"
        class="min-w-0 empty:hidden lg:col-start-2 lg:row-start-2 xl:col-start-2 xl:col-span-2"
      />
    </main>

    <MobileDrawer
      v-if="drawerOpen"
      :open="drawerOpen"
      :sections="drawerSections"
      :resource-items="resourceItems"
      :sidebar-groups="sidebarGroups"
      :category-items="categoryItems"
      :wiki-mode="isWikiMode"
      :wiki-tree="wikiTree"
      :footer="safeFooter"
      :has-moderation-reports="hasModerationReports"
      :close-label="t('shell.closeMenu')"
      :menu-label="t('shell.menu')"
      :resources-label="t('shell.resources')"
      :sidebar-icon="navIcon"
      :categories-label="t('shell.categories')"
      @close="closeDrawer"
    />

    <component :is="UserCard" v-if="UserCard" />
    <WikiSearchPanel v-if="isWikiMode" />
    <QuickPublishModal v-if="layout.viewer.isAuthenticated" :layout="layout" />

    <!-- 移动端发布 FAB：<sm 显示（navbar 上的发布按钮 sm+ 才渲染）。
         56px 直径（拇指可达），点击向上呼出发布类型菜单，层级低于抽屉 z-[60]。
         若已在 /publish 发布页面或 /p/post 等帖子详情页，则主动隐去，避免遮挡内容和互动控件。 -->
    <PublishMenu v-if="layout.viewer.isAuthenticated && !isPublishPage && !isTopicPage" variant="fab" />
  </div>
</template>
