<script setup lang="ts">
import { Award, Bookmark, FileText, List, MessageCircle, UserRound } from '@lucide/vue'
defineProps<{ tabs: Array<{ key: string; label: string; url: string; active: boolean }> }>()
const icons: Record<string, typeof UserRound> = { summary: UserRound, activity: List, bookmarks: Bookmark, badges: Award, topics: FileText, replies: MessageCircle }
</script>
<template>
  <nav class="grid border-y border-line" :style="{ gridTemplateColumns: `repeat(${tabs.length}, minmax(0, 1fr))` }">
    <a v-for="tab in tabs" :key="tab.key" :href="tab.url" :aria-current="tab.active ? 'page' : undefined"
      class="inline-flex min-h-11 min-w-0 items-center justify-center gap-2 px-2 py-2 text-sm font-semibold"
      :class="tab.active ? 'text-primary shadow-[inset_0_-2px_0_var(--gf-color-primary)]' : 'text-base-content/55 hover:text-base-content'">
      <component :is="icons[tab.key] || List" class="h-4 w-4 shrink-0" /><span class="break-words">{{ tab.label }}</span>
    </a>
  </nav>
</template>
