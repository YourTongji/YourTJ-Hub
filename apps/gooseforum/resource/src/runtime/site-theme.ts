import { computed, ref } from 'vue'
import type { ThemePayload } from '@gooseforum/client'

export type ThemePreference = 'auto' | 'light' | 'dark'
export type SiteTheme = 'gf-light' | 'gf-dark'

const STORAGE_KEY = 'goose-site-theme'
const COOKIE_KEY = 'goose-site-theme'
const PREFERENCE_STORAGE_KEY = 'goose-site-theme-preference'
const PREFERENCE_COOKIE_KEY = 'goose-site-theme-preference'
const themes: SiteTheme[] = ['gf-light', 'gf-dark']
const THEME_LINK_ID = 'goose-site-theme-link'
const THEME_PREVIEW_STYLE_ID = 'goose-site-theme-preview'
const THEME_SWITCHING_CLASS = 'theme-switching'
const themeColors: Record<SiteTheme, string> = {
  'gf-light': '#fbfdff',
  'gf-dark': '#101010',
}

// Detect system theme preference immediately on module load
function detectSystemIsDark(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

// System preference tracking - initialize immediately
const systemIsDark = ref(detectSystemIsDark())

// User's preference (auto/light/dark) - must be resolved before theme
const currentPreference = ref<ThemePreference>(resolveInitialPreference())

// Current applied theme - resolve after preference so we can consider auto mode
const currentTheme = ref<SiteTheme>(resolveInitialTheme())

export function useSiteTheme() {
  const isDark = computed(() => currentTheme.value === 'gf-dark')
  const isAuto = computed(() => currentPreference.value === 'auto')

  return {
    theme: currentTheme,
    preference: currentPreference,
    isDark,
    isAuto,
    toggleTheme,
    setPreference: setThemePreference,
  }
}

/**
 * Initialize system theme listener to detect and respond to system theme changes.
 * Should be called once during app initialization.
 */
export function initSystemThemeListener() {
  if (typeof window === 'undefined' || !window.matchMedia) return

  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

  // Update system theme state
  systemIsDark.value = mediaQuery.matches

  // If in auto mode, immediately apply the detected system theme
  if (currentPreference.value === 'auto') {
    const effectiveTheme = getEffectiveTheme()
    if (effectiveTheme !== currentTheme.value) {
      setTheme(effectiveTheme)
    }
  }

  // Listen for future changes
  mediaQuery.addEventListener('change', (e) => {
    systemIsDark.value = e.matches
    // If in auto mode, apply the new system theme
    if (currentPreference.value === 'auto') {
      applyThemeWithTransition(getEffectiveTheme())
    }
  })
}

/**
 * Get the effective theme based on current preference and system state.
 */
function getEffectiveTheme(): SiteTheme {
  if (currentPreference.value === 'auto') {
    return systemIsDark.value ? 'gf-dark' : 'gf-light'
  }
  return currentPreference.value === 'dark' ? 'gf-dark' : 'gf-light'
}

/**
 * Set user's theme preference and apply the effective theme.
 */
export function setThemePreference(preference: ThemePreference) {
  currentPreference.value = preference
  const effectiveTheme = getEffectiveTheme()

  // Only update if theme actually changes
  if (effectiveTheme !== currentTheme.value) {
    applyThemeWithTransition(effectiveTheme)
  }

  writePreferenceCookie(preference)
  try {
    window.localStorage.setItem(PREFERENCE_STORAGE_KEY, preference)
  } catch {
    // Ignore storage failures in private or restricted browsing contexts.
  }
}

export function applyStoredTheme() {
  applyTheme(currentTheme.value)
}

export function applySiteThemePayload(theme?: ThemePayload) {
  for (const name of themes) {
    const color = theme?.enabled ? theme.colors?.[name] : undefined
    themeColors[name] = normalizeThemeColor(color) || (name === 'gf-dark' ? '#101010' : '#fbfdff')
  }
  applySiteThemeLink(theme?.enabled ? theme.href : '')
  applyTheme(currentTheme.value)
}

export function applySiteThemeCss(css: string) {
  let el = document.getElementById(THEME_PREVIEW_STYLE_ID) as HTMLStyleElement | null
  if (!css) {
    el?.remove()
    return
  }
  if (!el) {
    el = document.createElement('style')
    el.id = THEME_PREVIEW_STYLE_ID
  }
  document.head.appendChild(el)
  el.textContent = css
}

function applySiteThemeLink(href?: string) {
  let el = document.getElementById(THEME_LINK_ID) as HTMLLinkElement | null
  if (!href) {
    el?.remove()
    return
  }
  const absoluteHref = new URL(href, window.location.origin).href
  if (!el) {
    el = document.createElement('link')
    el.id = THEME_LINK_ID
    el.rel = 'stylesheet'
  }
  if (el.href === absoluteHref && el.isConnected) {
    return
  }
  if (el.href !== absoluteHref) {
    el.href = href
  }
  document.head.appendChild(el)
}

export function toggleTheme() {
  toggleThemeFromElement()
}

/**
 * 主题切换（带 View Transitions 圆形扩散动画）。
 * 从页面中心向外扩散（circle mask，参考 theme-toggle.rdsx.dev），
 * 旧层垫底被圆形边缘逐步覆盖。无 startViewTransition 时直接切换。
 */
export function toggleThemeFromElement(_trigger?: Element | null) {
  // In auto mode, toggle switches between light and dark but keeps auto preference
  // Otherwise, toggle between light and dark
  const newTheme = currentTheme.value === 'gf-dark' ? 'gf-light' : 'gf-dark'
  applyThemeWithTransition(newTheme)
}

export function setThemeFromElement(theme: SiteTheme, _trigger?: Element | null) {
  if (theme === currentTheme.value) return
  applyThemeWithTransition(theme)
}

export function setTheme(theme: SiteTheme) {
  currentTheme.value = theme
  applyTheme(theme)
  writeThemeCookie(theme)
  try {
    window.localStorage.setItem(STORAGE_KEY, theme)
  } catch {
    // Ignore storage failures in private or restricted browsing contexts.
  }
}
// 引用计数：连续快速切换时，每个进行中的 view transition 占一个引用，
// 只有最后一个完成（或全部失败回退）才移除抑制 class。
let themeSwitchingRefs = 0

function setThemeSwitching(active: boolean) {
  themeSwitchingRefs = Math.max(0, themeSwitchingRefs + (active ? 1 : -1))
  const root = document.documentElement
  if (themeSwitchingRefs > 0) {
    root.classList.add(THEME_SWITCHING_CLASS)
  } else {
    root.classList.remove(THEME_SWITCHING_CLASS)
  }
}

function applyThemeWithTransition(theme: SiteTheme) {
  const apply = () => setTheme(theme)
  if (typeof document.startViewTransition !== 'function') {
    apply()
    return
  }
  // 抑制元素级 CSS transition（导航/标签/按钮上并发 color/background 过渡，
  // 与 view transition 叠加会拥塞主线程样式与绘制），只保留 view transition 自身。
  // 在 startViewTransition 之前同步加上，确保新旧快照捕获时都处于抑制状态。
  setThemeSwitching(true)
  try {
    const transition = document.startViewTransition(apply)
    transition.finished
      .catch(() => {})
      .finally(() => setThemeSwitching(false))
  } catch {
    apply()
    setThemeSwitching(false)
  }
}

function resolveInitialTheme(): SiteTheme {
  // First check if there's an explicit theme cookie/storage
  const documentTheme = document.documentElement.dataset.theme || null
  if (isSiteTheme(documentTheme)) return documentTheme

  try {
    const cookieTheme = readThemeCookie()
    if (isSiteTheme(cookieTheme)) return cookieTheme
  } catch {
    // Fall through to local storage compatibility.
  }

  try {
    const stored = window.localStorage.getItem(STORAGE_KEY)
    if (isSiteTheme(stored)) {
      writeThemeCookie(stored)
      return stored
    }
  } catch {
    // Fall through to check preference.
  }

  // If preference is auto, use system theme
  // currentPreference is already initialized at this point
  if (currentPreference.value === 'auto') {
    return systemIsDark.value ? 'gf-dark' : 'gf-light'
  }

  // If preference is light or dark, use that
  if (currentPreference.value === 'dark') return 'gf-dark'

  return 'gf-light'
}

function resolveInitialPreference(): ThemePreference {
  // Check document attribute (server-side rendered)
  const docPref = document.documentElement.dataset.themePreference ?? null
  if (isThemePreference(docPref)) return docPref

  // Check cookie
  try {
    const cookiePref = readPreferenceCookie() || null
    if (isThemePreference(cookiePref)) return cookiePref
  } catch {
    // Fall through to local storage
  }

  // Check localStorage
  try {
    const stored = window.localStorage.getItem(PREFERENCE_STORAGE_KEY)
    if (isThemePreference(stored)) {
      writePreferenceCookie(stored)
      return stored
    }
  } catch {
    // Fall through to auto
  }

  // Default to auto
  return 'auto'
}

function applyTheme(theme: SiteTheme) {
  document.documentElement.dataset.theme = theme
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', themeColors[theme])
}

function isSiteTheme(value: string | null): value is SiteTheme {
  return themes.includes(value as SiteTheme)
}

function isThemePreference(value: string | null): value is ThemePreference {
  return ['auto', 'light', 'dark'].includes(value as ThemePreference)
}

function readThemeCookie() {
  return document.cookie
    .split('; ')
    .find((item) => item.startsWith(`${COOKIE_KEY}=`))
    ?.split('=')
    .slice(1)
    .join('=') || ''
}

function writeThemeCookie(theme: SiteTheme) {
  const secure = window.location.protocol === 'https:' ? '; Secure' : ''
  document.cookie = `${COOKIE_KEY}=${theme}; path=/; max-age=31536000; samesite=lax${secure}`
}

function readPreferenceCookie() {
  return document.cookie
    .split('; ')
    .find((item) => item.startsWith(`${PREFERENCE_COOKIE_KEY}=`))
    ?.split('=')
    .slice(1)
    .join('=') || ''
}

function writePreferenceCookie(preference: ThemePreference) {
  const secure = window.location.protocol === 'https:' ? '; Secure' : ''
  document.cookie = `${PREFERENCE_COOKIE_KEY}=${preference}; path=/; max-age=31536000; samesite=lax${secure}`
}

function normalizeThemeColor(value?: string) {
  if (!value) return ''
  return /^#[0-9a-fA-F]{6}$/.test(value) ? value : ''
}