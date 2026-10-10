<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Loader2 } from '@lucide/vue'

const { t } = useI18n()
</script>

<template>
  <div class="fixed inset-0 z-[2100] flex items-center justify-center bg-black/40 backdrop-blur-xs">
    <div class="flex items-center gap-3 rounded-2xl border border-line/70 bg-base-100 px-5 py-3.5 shadow-2xl">
      <Loader2 class="h-5 w-5 animate-spin text-primary" />
      <span class="text-sm font-medium text-base-content/80">{{ t('common.loadingShort') }}</span>
    </div>
  </div>
</template>
