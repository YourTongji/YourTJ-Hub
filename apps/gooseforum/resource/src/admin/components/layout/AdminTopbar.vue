<script setup lang="ts">
import { adminText } from '@/admin/runtime/i18n-text'
import { computed, ref } from 'vue'
import { ArrowLeft, Check, Languages, Monitor, Moon, Sun } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/admin/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/admin/components/ui/popover'
import { Separator } from '@/admin/components/ui/separator'
import { SidebarTrigger } from '@/admin/components/ui/sidebar'
import AdminCommandSearch from '@/admin/components/layout/AdminCommandSearch.vue'
import { setLocale, supportedLocales, type Locale } from '@/runtime/i18n'
import { useSiteTheme, type ThemePreference } from '@/runtime/site-theme'
import type { LayoutPayload } from '@gooseforum/client'

defineProps<{
  layout: LayoutPayload
}>()

const { t, locale } = useI18n()
const languageMenuOpen = ref(false)
const themeMenuOpen = ref(false)
// Admin shares the site appearance preference, so both surfaces stay in step.
const { preference, isDark, setPreference } = useSiteTheme()
const themeOptions = computed(() => [
  { value: 'auto' as ThemePreference, label: t('shell.themeAuto'), icon: Monitor },
  { value: 'light' as ThemePreference, label: t('shell.themeLight'), icon: Sun },
  { value: 'dark' as ThemePreference, label: t('shell.themeDark'), icon: Moon },
])

function switchLocale(nextLocale: Locale) {
  setLocale(nextLocale)
  languageMenuOpen.value = false
}
function switchTheme(value: ThemePreference) {
  setPreference(value)
  themeMenuOpen.value = false
}
</script>

<template>
  <header class="sticky top-0 z-50 flex h-16 shrink-0 items-center gap-3 bg-background/90 px-4 backdrop-blur supports-[backdrop-filter]:bg-background/75 sm:gap-4">
    <SidebarTrigger class="-ms-1" />
    <Separator orientation="vertical" class="h-6" />

    <AdminCommandSearch />

    <div class="flex-1" />

    <div class="ms-auto flex items-center gap-1.5 sm:gap-2">
      <Popover v-model:open="themeMenuOpen">
        <PopoverTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            type="button"
            class="text-muted-foreground"
            :aria-label="t('shell.switchTheme')"
            :title="t('shell.switchTheme')"
          >
            <Moon v-if="isDark" class="size-4" />
            <Sun v-else class="size-4" />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" class="w-40 p-1">
          <button
            v-for="option in themeOptions"
            :key="option.value"
            type="button"
            class="flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-start text-sm transition-colors duration-150 hover:bg-accent"
            :aria-pressed="preference === option.value"
            @click="switchTheme(option.value)"
          >
            <component :is="option.icon" class="size-4 text-muted-foreground" />
            <span class="flex-1">{{ option.label }}</span>
            <Check v-if="preference === option.value" class="size-4 text-foreground" />
          </button>
        </PopoverContent>
      </Popover>
      <Popover v-model:open="languageMenuOpen">
        <PopoverTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            type="button"
            class="text-muted-foreground"
            :aria-label="t('shell.switchLanguage')"
            :title="t('shell.switchLanguage')"
          >
            <Languages class="size-4" />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" class="w-40 p-1">
          <button
            v-for="item in supportedLocales"
            :key="item"
            type="button"
            class="flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-start text-sm transition-colors duration-150 hover:bg-accent"
            :aria-pressed="locale === item"
            @click="switchLocale(item)"
          >
            <span class="flex-1">{{ t(`locale.${item}`) }}</span>
            <Check v-if="locale === item" class="size-4 text-foreground" />
          </button>
        </PopoverContent>
      </Popover>
      <Button as-child variant="outline" class="hidden md:inline-flex">
        <a href="/">
          <ArrowLeft class="size-4" />
          {{ adminText('k007y') }}
        </a>
      </Button>
      <img
        v-if="layout.viewer.isAuthenticated"
        :src="layout.viewer.avatarUrl"
        :alt="layout.viewer.username"
        class="size-9 rounded-full object-cover outline outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10"
      />
    </div>
  </header>
</template>
