<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import PageHeader from '@/site/components/PageHeader.vue'
import type { LayoutPayload, PrivacyPageProps } from '@gooseforum/client'

defineProps<{
  layout: LayoutPayload
  props: PrivacyPageProps
}>()
const { t } = useI18n()
</script>

<template>
  <div class="pb-12">
    <PageHeader :title="t('privacyPage.title')" :description="t('privacyPage.subtitle')" compact />
    <div class="rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 sm:bg-base-100 sm:p-6">
      <div v-if="!props.enabled" class="text-sm text-base-content/55">{{ t('privacyPage.unavailable') }}</div>
      <div v-else class="prose prose-sm max-w-none text-base-content/80" v-html="props.contentHtml" />
    </div>
  </div>
</template>
