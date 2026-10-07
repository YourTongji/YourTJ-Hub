<script setup lang="ts">
import { computed } from 'vue'
import type { UserBadgePayload } from '@gooseforum/client'
import UserAvatar from './UserAvatar.vue'

const props = defineProps<{ avatarUrl: string; avatarAlt: string; coverUrl?: string; badge?: UserBadgePayload | null }>()
const coverStyle = computed(() => {
  const fallback = 'linear-gradient(135deg, var(--gf-color-base-200) 0%, var(--gf-color-info-content) 52%, var(--gf-color-base-200) 100%)'
  const cover = props.coverUrl?.trim()
  return { backgroundImage: cover ? `url(${JSON.stringify(cover)}), ${fallback}` : fallback }
})
</script>

<template>
  <header>
    <div class="relative">
      <div class="h-36 border-b border-line bg-base-300 bg-cover bg-center sm:h-60" :style="coverStyle" />
      <div class="absolute right-4 top-full z-10 mt-2 flex flex-wrap items-center justify-end gap-2 sm:hidden">
        <slot name="actions" />
      </div>
    </div>
    <div class="relative z-0 px-4 pb-4 sm:px-5">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div class="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-start sm:gap-4 sm:flex-1">
          <UserAvatar :src="avatarUrl" :alt="avatarAlt" :badge="badge" size="large" img-class="rounded-full"
            class="-mt-9 h-24 w-24 shrink-0 rounded-full border-2 border-base-100 bg-base-100 shadow-sm sm:-mt-10 sm:h-28 sm:w-28" />
          <div class="min-w-0 sm:flex-1 sm:pt-3"><slot name="identity" /></div>
        </div>
        <div class="hidden shrink-0 flex-wrap items-center gap-2 sm:flex"><slot name="actions" /></div>
      </div>
      <slot name="meta" />
    </div>
  </header>
</template>
