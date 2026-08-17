<script setup lang="ts">
import { Search } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { useWikiSearchPanel } from '@/runtime/use-wiki-search'

const { t } = useI18n()
const { openPanel } = useWikiSearchPanel()
</script>

<template>
  <button
    type="button"
    class="flex h-9 w-full items-center gap-2 rounded-md border border-line bg-base-100 px-2 text-[13px] font-medium text-base-content/60 transition-colors duration-150 hover:border-primary/40 hover:bg-base-200 hover:text-base-content"
    :aria-label="t('wikiSearch.openSearch')"
    :title="t('wikiSearch.openSearch')"
    @click="openPanel"
  >
    <Search class="h-4 w-4 shrink-0 text-icon-muted" aria-hidden="true" />
    <span class="min-w-0 flex-1 truncate text-left">{{ t('wikiSearch.placeholder') }}</span>
    <kbd class="shrink-0 rounded border border-line bg-base-200 px-1.5 py-0.5 text-[10px] font-semibold text-base-content/55">
      ⌘K
    </kbd>
  </button>
</template>
