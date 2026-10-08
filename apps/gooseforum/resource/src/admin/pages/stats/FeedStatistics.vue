<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Activity, Download, RefreshCw } from '@lucide/vue'
import AdminSection from '@/admin/components/AdminSection.vue'
import { Button } from '@/admin/components/ui/button'
import { Badge } from '@/admin/components/ui/badge'
import { getFeedSummary, type FeedSummary } from '@/admin/runtime/api'
const { t, locale } = useI18n()
const summary = ref<FeedSummary>()
const error = ref('')
const loading = ref(false)
const filter = ref('all')
const rowPage = ref(1)
const number = (value: unknown) => (typeof value === 'number' ? value : 0)
const healthNumber = (key: string) => number(summary.value?.health[key])
const format = (value: number) => value.toLocaleString(locale.value)
const date = (value: string | number) =>
  value
    ? new Date(typeof value === 'number' ? value * 1000 : value).toLocaleString(
        locale.value,
      )
    : t('feedAdmin.notYet')
const filtered = computed(() =>
  (summary.value?.rows || []).filter(
    (row) => filter.value === 'all' || row.feed === filter.value,
  ),
)
const feeds = computed(() => [
  ...new Set(summary.value?.rows.map((row) => row.feed) || []),
])
const visibleRows = computed(() =>
  filtered.value.slice((rowPage.value - 1) * 20, rowPage.value * 20),
)
const rowPages = computed(() =>
  Math.max(1, Math.ceil(filtered.value.length / 20)),
)
// Detailed parameters remain inspectable as labels and values, never JSON blocks.
function fields(value: unknown, prefix = ''): [string, string][] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return []
  return Object.entries(value).flatMap(([key, item]): [string, string][] => {
    const label = prefix ? `${prefix} / ${key}` : key
    if (item && typeof item === 'object' && !Array.isArray(item))
      return fields(item, label)
    if (Array.isArray(item))
      return [[label, item.map((value) => String(value)).join(' · ')]]
    return [
      [
        label,
        typeof item === 'boolean'
          ? t(item ? 'feedAdmin.enabled' : 'feedAdmin.disabled')
          : String(item ?? '—'),
      ],
    ]
  })
}
const parameters = computed(() => fields(summary.value?.health.parameters))
function effects(result: string): [string, Record<string, unknown>][] {
  try {
    const parsed = JSON.parse(result)
    return parsed?.effects && typeof parsed.effects === 'object'
      ? Object.entries(parsed.effects).filter(
          (entry): entry is [string, Record<string, unknown>] =>
            !!entry[1] && typeof entry[1] === 'object',
        )
      : []
  } catch {
    return []
  }
}
async function load() {
  loading.value = true
  error.value = ''
  try {
    summary.value = await getFeedSummary()
    rowPage.value = 1
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('common.loadFailed')
  } finally {
    loading.value = false
  }
}
function download() {
  if (!summary.value) return
  const url = URL.createObjectURL(
    new Blob([JSON.stringify(summary.value, null, 2)], {
      type: 'application/json',
    }),
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
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="flex items-center gap-2 text-sm text-muted-foreground">
        <Activity class="size-4" />{{ t('feedAdmin.aggregateOnly') }}
      </p>
      <div class="flex gap-2">
        <Button variant="outline" :disabled="loading" @click="load"
          ><RefreshCw class="size-4" :class="{ 'animate-spin': loading }" />{{
            t('feed.refresh')
          }}</Button
        ><Button
          variant="outline"
          :disabled="!summary || loading"
          @click="download"
          ><Download class="size-4" />{{ t('feed.export') }}</Button
        >
      </div>
    </div>
    <p
      v-if="error"
      role="alert"
      class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"
    >
      {{ error }}
    </p>
    <div
      v-if="loading && !summary"
      role="status"
      class="rounded-lg border bg-card p-12 text-center text-muted-foreground"
    >
      {{ t('common.loading') }}
    </div>
    <template v-if="summary">
      <div class="grid grid-cols-2 gap-3 xl:grid-cols-4 xl:gap-4">
        <AdminSection body-class="p-5" data-testid="capture-status"
          ><p class="text-sm text-muted-foreground">{{ t('feed.capture') }}</p>
          <p class="mt-2 text-2xl font-semibold">
            {{
              t(
                summary.metricsEnabled
                  ? 'feedAdmin.enabled'
                  : 'feedAdmin.disabled',
              )
            }}
          </p>
          <div class="mt-3 flex flex-wrap gap-2">
            <Badge variant="secondary">{{
              t(
                summary.enabled
                  ? 'feedAdmin.rankingEnabled'
                  : 'feedAdmin.rankingDisabled',
              )
            }}</Badge
            ><Badge
              :variant="summary.rankingReady ? 'outline' : 'destructive'"
              >{{
                t(
                  summary.rankingReady
                    ? 'feedAdmin.ready'
                    : 'feedAdmin.notReady',
                )
              }}</Badge
            >
          </div></AdminSection
        >
        <AdminSection body-class="p-5"
          ><p class="text-sm text-muted-foreground">
            {{ t('feedAdmin.rollout') }}
          </p>
          <p class="mt-2 text-2xl font-semibold">
            {{ summary.rolloutPercent
            }}<span class="ml-1 text-lg text-muted-foreground">%</span>
          </p>
          <p class="mt-3 text-xs text-muted-foreground">
            {{ t('feedAdmin.retention', { days: summary.rawRetentionDays }) }}
          </p></AdminSection
        >
        <AdminSection body-class="p-5"
          ><p class="text-sm text-muted-foreground">
            {{ t('feedAdmin.accepted') }}
          </p>
          <p class="mt-2 text-2xl font-semibold">
            {{ format(healthNumber('accepted')) }}
          </p>
          <p class="mt-3 text-xs text-muted-foreground">
            {{
              t('feedAdmin.queued', {
                count: format(healthNumber('queueLength')),
              })
            }}
          </p></AdminSection
        >
        <AdminSection body-class="p-5"
          ><p class="text-sm text-muted-foreground">
            {{ t('feedAdmin.failures') }}
          </p>
          <p
            class="mt-2 text-2xl font-semibold"
            :class="{
              'text-destructive': healthNumber('backgroundFailures') > 0,
            }"
          >
            {{ format(healthNumber('backgroundFailures')) }}
          </p>
          <p class="mt-3 text-xs text-muted-foreground">
            {{
              t('feedAdmin.dropped', { count: format(healthNumber('dropped')) })
            }}
          </p></AdminSection
        >
      </div>
      <AdminSection
        ><template #header
          ><div class="flex flex-wrap items-center justify-between gap-2 p-4">
            <h2 class="font-semibold">{{ t('feedAdmin.health') }}</h2>
            <span class="text-xs text-muted-foreground"
              >{{ t('feedAdmin.lastRank') }} ·
              {{ date(healthNumber('lastRankAt')) }}</span
            >
          </div></template
        >
        <dl class="grid grid-cols-2 gap-5 p-5 lg:grid-cols-4">
          <div
            v-for="[key, value] in [
              ['queueBytes', `${format(healthNumber('queueBytes'))} B`],
              ['lag', `${format(healthNumber('oldestQueuedMs'))} ms`],
              ['sampleQueue', format(healthNumber('sampleQueueLength'))],
              ['sampleBytes', `${format(healthNumber('sampleQueueBytes'))} B`],
            ]"
            :key="key"
          >
            <dt class="text-sm text-muted-foreground">
              {{ t(`feedAdmin.${key}`) }}
            </dt>
            <dd class="mt-1 font-medium">{{ value }}</dd>
          </div>
        </dl>
        <p
          v-if="summary.health.previousEpochIncomplete"
          class="border-t p-4 text-sm text-destructive"
        >
          {{ t('feedAdmin.incomplete') }}
        </p>
      </AdminSection>
      <AdminSection
        ><template #header
          ><div class="flex flex-wrap items-center justify-between gap-3 p-4">
            <div>
              <h2 class="font-semibold">{{ t('feedAdmin.aggregates') }}</h2>
              <p class="mt-1 text-xs text-muted-foreground">
                {{ t('feedAdmin.window') }}
              </p>
            </div>
            <select
              v-model="filter"
              :aria-label="t('feedAdmin.feed')"
              class="h-9 rounded-md border bg-background px-3 text-sm"
              @change="rowPage = 1"
            >
              <option value="all">{{ t('feedAdmin.allFeeds') }}</option>
              <option v-for="feed in feeds" :key="feed" :value="feed">
                {{ feed }}
              </option>
            </select>
          </div></template
        >
        <p
          v-if="summary.truncated"
          class="border-b bg-muted/30 px-4 py-3 text-sm text-muted-foreground"
        >
          {{ t('feed.truncated') }}
        </p>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead
              class="border-b bg-muted/45 text-left text-xs text-muted-foreground"
            >
              <tr>
                <th
                  v-for="key in [
                    'day',
                    'feed',
                    'variant',
                    'versions',
                    'metric',
                    'count',
                  ]"
                  :key="key"
                  class="whitespace-nowrap px-4 py-3"
                >
                  {{ t(`feedAdmin.${key}`) }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr v-for="(row, index) in visibleRows" :key="index">
                <td class="whitespace-nowrap px-4 py-3">{{ row.day }}</td>
                <td class="px-4 py-3">
                  {{ row.feed
                  }}<span class="mt-1 block text-xs text-muted-foreground">{{
                    row.capability
                  }}</span>
                </td>
                <td class="px-4 py-3">
                  <Badge variant="secondary">{{ row.variant || '—' }}</Badge
                  ><span class="mt-1 block text-xs text-muted-foreground">{{
                    row.weightVariant
                  }}</span>
                </td>
                <td
                  class="px-4 py-3 text-xs text-muted-foreground"
                  :title="`${row.hash} / ${row.rankHash}`"
                >
                  <span class="block font-mono"
                    >{{ row.hash.slice(0, 8) }} /
                    {{ row.rankHash.slice(0, 8) }}</span
                  ><span class="mt-1 block">{{ row.experiment || '—' }}</span>
                </td>
                <td class="px-4 py-3">{{ row.metric }}</td>
                <td class="px-4 py-3 font-medium tabular-nums">
                  {{ format(row.count) }}
                </td>
              </tr>
              <tr v-if="!filtered.length">
                <td colspan="6" class="p-10 text-center text-muted-foreground">
                  {{ t('feedAdmin.noData') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div
          v-if="filtered.length"
          class="flex flex-wrap items-center justify-between gap-2 border-t px-4 py-3 text-sm text-muted-foreground"
        >
          <span>{{ t('feedAdmin.rows', { count: filtered.length }) }}</span>
          <div class="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              :disabled="rowPage <= 1"
              @click="rowPage--"
              >{{ t('feedAdmin.previous') }}</Button
            ><span>{{ rowPage }} / {{ rowPages }}</span
            ><Button
              size="sm"
              variant="outline"
              :disabled="rowPage >= rowPages"
              @click="rowPage++"
              >{{ t('feedAdmin.next') }}</Button
            >
          </div>
        </div>
      </AdminSection>
      <AdminSection
        ><template #header
          ><h2 class="p-4 font-semibold">
            {{ t('feedAdmin.experiments') }}
          </h2></template
        >
        <p
          v-if="!summary.periods.length"
          class="p-8 text-center text-sm text-muted-foreground"
        >
          {{ t('feedAdmin.noExperiments') }}
        </p>
        <div v-else class="divide-y">
          <article
            v-for="period in summary.periods"
            :key="period.id"
            class="space-y-4 p-5"
          >
            <div class="flex flex-wrap items-center justify-between gap-2">
              <h3 class="break-all font-medium">{{ period.id }}</h3>
              <Badge :variant="period.aborted ? 'destructive' : 'secondary'">{{
                t(
                  period.aborted
                    ? 'feedAdmin.aborted'
                    : period.result
                      ? 'feedAdmin.completed'
                      : 'feedAdmin.collecting',
                )
              }}</Badge>
            </div>
            <p v-if="period.aborted" class="text-sm text-destructive">
              {{ period.aborted }}
            </p>
            <dl class="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
              <div>
                <dt class="text-muted-foreground">
                  {{ t('feedAdmin.control') }}
                </dt>
                <dd class="mt-1 font-medium">
                  {{ format(period.assignedControl) }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('feedAdmin.treatment') }}
                </dt>
                <dd class="mt-1 font-medium">
                  {{ format(period.assignedTreatment) }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('feedAdmin.enrollUntil') }}
                </dt>
                <dd class="mt-1">{{ date(period.enrollUntil) }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('feedAdmin.analyzeAt') }}
                </dt>
                <dd class="mt-1">{{ date(period.analyzeAt) }}</dd>
              </div>
            </dl>
            <div
              v-for="[metric, effect] in effects(period.result)"
              :key="metric"
              class="flex flex-wrap items-center justify-between gap-2 rounded-md bg-muted/40 px-3 py-2 text-sm"
            >
              <span>{{ metric }}</span
              ><span class="font-mono"
                >Δ {{ number(effect.difference).toFixed(3) }} ·
                {{
                  effect.intervalAvailable
                    ? `[${number(effect.lower95).toFixed(3)}, ${number(effect.upper95).toFixed(3)}]`
                    : t('feedAdmin.insufficient')
                }}</span
              >
            </div>
            <p v-if="period.result" class="text-xs text-muted-foreground">
              {{ t('feedAdmin.resultLimitation') }}
            </p>
          </article>
        </div></AdminSection
      >
      <AdminSection
        ><details class="group">
          <summary class="cursor-pointer p-4 text-sm font-medium">
            {{ t('feedAdmin.parameters')
            }}<span
              class="ml-2 font-mono text-xs font-normal text-muted-foreground"
              >{{ summary.paramsHash.slice(0, 12) }}</span
            >
          </summary>
          <dl class="grid gap-x-8 gap-y-3 border-t p-5 text-xs sm:grid-cols-2">
            <div
              v-for="[key, value] in parameters"
              :key="key"
              class="flex min-w-0 flex-wrap justify-between gap-1"
            >
              <dt class="break-all text-muted-foreground">{{ key }}</dt>
              <dd class="break-all font-mono">{{ value }}</dd>
            </div>
          </dl>
        </details></AdminSection
      >
    </template>
  </div>
</template>
