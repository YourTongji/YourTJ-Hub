<script setup lang="ts">
import { computed } from 'vue'
import { adminText } from '@/admin/runtime/i18n-text'
import type { AdminStickerMarketPack } from '@/admin/types'

const props = defineProps<{
  pack: AdminStickerMarketPack
}>()

// The worker updates itemCount/failedCount after every item of the current run.
const processed = computed(() => Math.min(props.pack.itemCount + props.pack.failedCount, props.pack.importTotal))
const percent = computed(() => (props.pack.importTotal > 0 ? Math.round((processed.value / props.pack.importTotal) * 100) : 0))
</script>

<template>
  <p v-if="pack.status === 'queued' || pack.importTotal === 0" class="text-xs text-muted-foreground">
    {{ adminText('stickerMarketWaiting') }}
  </p>
  <div v-else class="w-36">
    <div
      role="progressbar"
      :aria-label="adminText('stickerMarketProgressLabel', { name: pack.name })"
      aria-valuemin="0"
      :aria-valuemax="pack.importTotal"
      :aria-valuenow="processed"
      class="h-1.5 overflow-hidden rounded-full bg-muted"
    >
      <div
        class="h-full rounded-full bg-primary transition-[width] duration-300 ease-out motion-reduce:transition-none"
        :style="{ width: `${percent}%` }"
      />
    </div>
    <p class="mt-1 text-xs tabular-nums text-muted-foreground">
      {{ adminText('stickerMarketProgress', { done: processed, total: pack.importTotal }) }}
    </p>
  </div>
</template>
