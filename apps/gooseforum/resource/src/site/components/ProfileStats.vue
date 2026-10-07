<script setup lang="ts">
import { formatNumber } from '@/runtime/format'
defineProps<{ items: Array<{ label: string; value: number; featured?: boolean }> }>()
</script>
<template>
  <section class="grid gap-y-4 border-b border-line pb-4" :class="items.length === 2 ? 'grid-cols-2' : 'grid-cols-3 sm:grid-cols-6'">
    <div v-for="item in items" :key="item.label" class="min-w-0 text-center">
      <div class="text-base font-bold tabular-nums lg:text-lg" :class="item.featured ? 'text-primary' : 'text-base-content'">{{ formatNumber(item.value) }}</div>
      <div class="mt-0.5 truncate text-[11px] font-medium lg:text-xs" :class="item.featured ? 'text-primary/80' : 'text-base-content/55'">{{ item.label }}</div>
    </div>
  </section>
</template>
