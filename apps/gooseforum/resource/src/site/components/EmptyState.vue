<script setup lang="ts">
import type { Component } from 'vue'
import LogoLoader from '@/site/components/LogoLoader.vue'

defineProps<{
  icon?: Component
  title: string
  description?: string
  loading?: boolean
}>()
</script>

<template>
  <div class="flex min-h-56 flex-col items-center justify-center px-6 text-center">
    <LogoLoader v-if="loading" :size="40" />
    <div
      v-else-if="icon"
      class="flex h-12 w-12 items-center justify-center rounded-2xl border border-line/60 bg-base-200/80 text-base-content/45 shadow-sm"
    >
      <component
        :is="icon"
        class="h-6 w-6 text-base-content/40 transition-colors"
      />
    </div>
    <h2 class="mt-3 text-base font-semibold text-base-content">{{ title }}</h2>
    <p v-if="description" class="mt-1 text-sm text-base-content/55">{{ description }}</p>
    <div v-if="$slots.default" class="mt-4">
      <slot />
    </div>
  </div>
</template>
