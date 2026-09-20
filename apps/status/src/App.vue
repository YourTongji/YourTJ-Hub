<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Check, ChevronDown, Languages, Moon, Sun } from '@lucide/vue'
import type { Locale } from './runtime/i18n'
import StatusPage from './StatusPage.vue'
const { t, locale } = useI18n()

const languages: Array<{ value: Locale; label: string }> = [
  { value: 'zh', label: '中文' },
  { value: 'en', label: 'EN' },
  { value: 'ja', label: '日本語' },
  { value: 'de', label: 'DE' },
]
const nav = ref<HTMLElement | null>(null)
const languageMenu = ref<HTMLElement | null>(null)
const languageOpen = ref(false)
const compactNav = ref(false)
const darkTheme = ref(document.documentElement.dataset.theme === 'gf-dark')
const currentLanguage = computed(() => languages.find(item => item.value === locale.value) ?? languages[0])
let lastScrollY = 0
let motionPreference: MediaQueryList | undefined

function theme() {
  darkTheme.value = !darkTheme.value
  const nextTheme = darkTheme.value ? 'gf-dark' : 'gf-light'
  document.documentElement.dataset.theme = nextTheme
  try { localStorage.setItem('theme', darkTheme.value ? 'dark' : 'light') } catch { /* storage is optional */ }
}
function focusSelectedLanguage() {
  nextTick(() => languageMenu.value?.querySelector<HTMLButtonElement>('[aria-checked="true"]')?.focus({ preventScroll: true }))
}
function toggleLanguage() {
  languageOpen.value = !languageOpen.value
  if (languageOpen.value) focusSelectedLanguage()
}
function selectLanguage(next: Locale) {
  locale.value = next
  document.documentElement.lang = next
  languageOpen.value = false
}
function closeLanguage(event?: KeyboardEvent) {
  if (event?.key !== 'Escape' || !languageOpen.value) return
  languageOpen.value = false
}
function onDocumentPointerDown(event: PointerEvent) {
  if (!nav.value?.contains(event.target as Node)) languageOpen.value = false
}
function onMenuKeydown(event: KeyboardEvent) {
  const items = Array.from(languageMenu.value?.querySelectorAll<HTMLButtonElement>('button') ?? [])
  const current = items.findIndex(item => item === document.activeElement)
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    items[(current + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length]?.focus()
  } else if (event.key === 'Home' || event.key === 'End') {
    event.preventDefault()
    items[event.key === 'Home' ? 0 : items.length - 1]?.focus()
  }
}
function onScroll() {
  const latest = window.scrollY
  const delta = latest - lastScrollY
  lastScrollY = latest
  if (motionPreference?.matches) return
  if (delta > 0 && latest > 96) compactNav.value = true
  else if (delta < 0) compactNav.value = false
}
function onMotionPreferenceChange() {
  if (motionPreference?.matches) compactNav.value = false
}
onMounted(() => {
  document.addEventListener('pointerdown', onDocumentPointerDown)
  document.addEventListener('keydown', closeLanguage)
  motionPreference = window.matchMedia('(prefers-reduced-motion: reduce)')
  lastScrollY = window.scrollY
  compactNav.value = !motionPreference.matches && lastScrollY > 96
  window.addEventListener('scroll', onScroll, { passive: true })
  motionPreference.addEventListener('change', onMotionPreferenceChange)
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  document.removeEventListener('keydown', closeLanguage)
  window.removeEventListener('scroll', onScroll)
  motionPreference?.removeEventListener('change', onMotionPreferenceChange)
})
</script>
<template>
  <div class="site-shell">
    <nav ref="nav" class="site-nav" :class="{ 'is-compact': compactNav }" aria-label="YourTJ">
      <a href="/" class="brand" aria-label="YourTJ Status">
        <img class="brand-mark" src="/logo-motion.svg" alt="" aria-hidden="true" width="38" height="38" />
        <span class="brand-copy">YourTJ Status</span>
      </a>
      <div class="site-tools">
        <div class="language-control">
          <button class="language-toggle" type="button" aria-haspopup="menu" :aria-expanded="languageOpen" :aria-label="`${t('status.language')}: ${currentLanguage.label}`" @click="toggleLanguage">
            <Languages :size="18" :stroke-width="1.7" aria-hidden="true" />
            <span class="language-label">{{ currentLanguage.label }}</span>
            <ChevronDown class="language-chevron" :class="{ open: languageOpen }" :size="15" :stroke-width="1.7" aria-hidden="true" />
          </button>
          <div v-if="languageOpen" ref="languageMenu" class="language-menu" role="menu" :aria-label="t('status.language')" @keydown="onMenuKeydown">
            <button v-for="item in languages" :key="item.value" type="button" role="menuitemradio" :aria-checked="item.value === locale" :tabindex="item.value === locale ? 0 : -1" @click="selectLanguage(item.value)">
              <span>{{ item.label }}</span><Check v-if="item.value === locale" :size="16" :stroke-width="1.8" aria-hidden="true" />
            </button>
          </div>
        </div>
        <button class="theme-toggle" type="button" :aria-label="t('status.theme')" :aria-pressed="darkTheme" @click="theme">
          <span aria-hidden="true" class="theme-icon theme-icon-sun"><Sun :size="18" :stroke-width="1.7" /></span>
          <span aria-hidden="true" class="theme-icon theme-icon-moon"><Moon :size="18" :stroke-width="1.7" /></span>
        </button>
        <a class="community-link" href="https://f.yourtj.de"><span>{{ t('status.community') }}</span><ArrowUpRight :size="14" :stroke-width="1.8" /></a>
      </div>
    </nav>
    <main><StatusPage /></main>
  </div>
</template>
