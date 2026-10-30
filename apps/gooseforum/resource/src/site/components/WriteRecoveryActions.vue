<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ code?: string }>()
const { t } = useI18n()
const kind = computed(() => {
  if (props.code === 'permission.emailRequired') return 'email'
  if (props.code === 'auth.required') return 'login'
  return undefined
})
const target = computed(() => {
  if (kind.value === 'email') return '/settings?tab=profile'
  const current = typeof window === 'undefined' ? '/' : window.location.pathname + window.location.search + window.location.hash
  return '/login?redirect=' + encodeURIComponent(current)
})
</script>

<template>
  <div v-if="kind" data-write-recovery class="mt-2 min-w-0 space-y-2">
    <a
      :href="target"
      target="_blank"
      rel="noopener noreferrer"
      class="gf-button gf-button-md gf-button-secondary h-auto min-h-11 whitespace-normal py-2 text-center"
    >{{ t(kind === 'email' ? 'auth.writeRecovery.email' : 'auth.writeRecovery.login') }}</a>
    <p class="text-sm text-base-content/65">{{ t('auth.writeRecovery.hint') }}</p>
  </div>
</template>
