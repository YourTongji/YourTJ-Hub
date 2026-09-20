<script setup lang="ts">
import { computed } from 'vue'
import { School } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
const props = defineProps<{ href: string; terms?: boolean; privacy?: boolean }>()
const { t, locale } = useI18n()
const loginUrl = computed(() => `${props.href}${props.href.includes('?') ? '&' : '?'}locale=${encodeURIComponent(locale.value)}`)
</script>

<template>
  <div class="space-y-2">
    <a :href="loginUrl" class="gf-button gf-button-lg gf-button-secondary w-full whitespace-normal text-center" data-tongji-login>
      <School class="h-5 w-5 shrink-0" aria-hidden="true" />
      {{ t('auth.tongjiLogin') }}
    </a>
    <p class="text-xs leading-relaxed text-base-content/55">{{ t('auth.tongjiSignupHint') }}</p>
    <p v-if="terms || privacy" class="text-xs leading-relaxed text-base-content/55">
      {{ t('auth.tongjiPolicies') }}
      <a v-if="terms" href="/terms" target="_blank" rel="noopener noreferrer" class="text-primary">{{ t('auth.termsLink') }}</a>
      <span v-if="terms && privacy"> · </span>
      <a v-if="privacy" href="/privacy" target="_blank" rel="noopener noreferrer" class="text-primary">{{ t('auth.privacyLink') }}</a>
    </p>
  </div>
</template>
