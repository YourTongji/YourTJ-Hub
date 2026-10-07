<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AdminSection from '@/admin/components/AdminSection.vue'
import { getFeedSummary, type FeedSummary } from '@/admin/runtime/api'
const { t } = useI18n()
const summary = ref<FeedSummary>()
const error = ref('')
const loading = ref(false)
async function load() {
  loading.value = true
  error.value = ''
  try {
    summary.value = await getFeedSummary()
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('common.loadFailed')
  } finally {
    loading.value = false
  }
}
function download() {
  if (!summary.value) return
  const url = URL.createObjectURL(
    new Blob([JSON.stringify(summary.value, null, 2)], { type: 'application/json' }),
  )
  const link = document.createElement('a')
  link.href = url
  link.download = 'feed-aggregates.json'
  link.click()
  URL.revokeObjectURL(url)
}
onMounted(load)
</script>
<template>
  <AdminSection :title="t('feed.statistics')">
    <div class="mb-3 flex gap-3">
      <button class="btn btn-sm" :disabled="loading" @click="load">{{ t('feed.refresh') }}</button
      ><button class="btn btn-sm" :disabled="!summary" @click="download">{{ t('feed.export') }}</button>
    </div>
    <p v-if="error" class="text-error">{{ error }}</p>
    <template v-if="summary">
      <p v-if="summary.truncated" class="text-warning text-sm">{{ t('feed.truncated') }}</p>
      <p class="text-sm text-base-content/70">
        {{ t('feed.capture') }}: {{ summary.metricsEnabled }} · {{ summary.rolloutPercent }}% ·
        {{ summary.rawRetentionDays }}d
      </p>
      <pre class="my-3 overflow-auto text-xs">{{ JSON.stringify(summary.health, null, 2) }}</pre>
      <div class="max-h-96 overflow-auto">
        <table class="table table-xs">
          <thead>
            <tr>
              <th>UTC+8</th>
              <th>Feed</th>
              <th>Variant</th>
              <th>Weights</th>
              <th>Params</th>
              <th>Capability</th>
              <th>Metric</th>
              <th>Count</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in summary.rows" :key="index">
              <td>{{ row.day }}</td>
              <td>{{ row.feed }}</td>
              <td>{{ row.variant }}</td>
              <td>{{ row.weightVariant }}</td>
              <td :title="`${row.hash} / ${row.rankHash}`">{{ row.hash.slice(0, 8) }}</td>
              <td>{{ row.capability }}</td>
              <td>{{ row.metric }}</td>
              <td>{{ row.count }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <pre v-for="period in summary.periods" :key="period.id" class="mt-3 overflow-auto text-xs">{{
        JSON.stringify(period, null, 2)
      }}</pre>
    </template>
  </AdminSection>
</template>
