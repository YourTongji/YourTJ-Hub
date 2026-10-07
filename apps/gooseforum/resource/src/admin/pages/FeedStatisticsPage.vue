<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { BasicPage } from '@/admin/components/global-layout'
import FeedStatistics from '@/admin/pages/stats/FeedStatistics.vue'
const { t } = useI18n()
</script>
<template>
  <BasicPage
    :title="t('feedAdmin.title')"
    :description="t('feedAdmin.description')"
    ><FeedStatistics
  /></BasicPage>
</template>
