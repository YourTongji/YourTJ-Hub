<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, ChevronDown, Download, RefreshCw, ShieldCheck } from '@lucide/vue'
import AdminSection from '@/admin/components/AdminSection.vue'
import { Button } from '@/admin/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/admin/components/ui/select'
import { getFeedSummary, type FeedSummary } from '@/admin/runtime/api'

type Row = FeedSummary['rows'][number]
type Effect = { difference?: number; intervalAvailable?: boolean; lower95?: number | null; upper95?: number | null }

const { t, locale } = useI18n()
const summary = ref<FeedSummary>()
const loadedAt = ref<Date>()
const error = ref('')
const loading = ref(false)
const byVariant = ref(false)
const showPipeline = ref(false)
const showRaw = ref(false)
const filter = ref('all')
const rowPage = ref(1)
const pageSize = 20

// Home tabs in the order people see them; cohort and assignment rows have their own sections.
const funnelFeeds = ['for_you', 'latest', 'following', 'hot', 'popular'] as const
const feedKeys: Record<string, string> = {
  for_you: 'feedAdmin.feedForYou',
  latest: 'feedAdmin.feedLatest',
  following: 'feedAdmin.feedFollowing',
  hot: 'feedAdmin.feedHot',
  popular: 'feedAdmin.feedPopular',
}
const variantKeys: Record<string, string> = {
  control: 'feedAdmin.variantControl',
  treatment: 'feedAdmin.variantTreatment',
  unassigned: 'feedAdmin.variantUnassigned',
}
const metricKeys: Record<string, string> = {
  served: 'feedAdmin.metricServed',
  visible: 'feedAdmin.metricVisible',
  served_open: 'feedAdmin.metricServedOpen',
  visible_open: 'feedAdmin.metricVisibleOpen',
  read_10s: 'feedAdmin.metricRead10s',
  foreground_read_seconds: 'feedAdmin.metricReadSeconds',
  assigned: 'feedAdmin.metricAssigned',
  new_topic_public: 'feedAdmin.metricNewTopicPublic',
  new_topic_visible_24h: 'feedAdmin.metricNewTopicVisible24h',
  new_topic_first_reply: 'feedAdmin.metricNewTopicFirstReply',
  first_reply_seconds: 'feedAdmin.metricFirstReplySeconds',
}
const actionKeys: Record<string, string> = {
  like: 'feedAdmin.actLike',
  bookmark: 'feedAdmin.actBookmark',
  post_like: 'feedAdmin.actPostLike',
  post_bookmark: 'feedAdmin.actPostBookmark',
  public_topic: 'feedAdmin.actPublicTopic',
  public_reply: 'feedAdmin.actPublicReply',
}
const effectKeys: Record<string, string> = {
  public_contributions: 'feedAdmin.effectPublicContributions',
  active_days: 'feedAdmin.effectActiveDays',
  d1: 'feedAdmin.effectD1',
  d7: 'feedAdmin.effectD7',
  corrected_actions: 'feedAdmin.effectCorrectedActions',
}
const interactionActions = ['like', 'bookmark', 'post_like', 'post_bookmark']
const contributionActions = ['public_topic', 'public_reply']

const rows = computed<Row[]>(() => summary.value?.rows || [])
const health = computed(() => summary.value?.health || {})
const num = (value: unknown) => (typeof value === 'number' && Number.isFinite(value) ? value : 0)
const format = (value: number) => value.toLocaleString(locale.value)
const percent = (part: number, whole: number) =>
  whole > 0 ? (part / whole).toLocaleString(locale.value, { style: 'percent', maximumFractionDigits: 1 }) : '—'
const dateTime = (value?: string | number | Date) => {
  if (!value) return t('feedAdmin.notYet')
  const parsed = value instanceof Date ? value : new Date(typeof value === 'number' ? value * 1000 : value)
  return Number.isNaN(parsed.getTime()) || parsed.getTime() <= 0
    ? t('feedAdmin.notYet')
    : parsed.toLocaleString(locale.value, { dateStyle: 'medium', timeStyle: 'short' })
}
const duration = (seconds: number) => {
  if (!seconds) return '—'
  const units: [Intl.NumberFormatOptions['unit'], number][] = [['day', 86400], ['hour', 3600], ['minute', 60]]
  for (const [unit, size] of units) {
    if (seconds >= size)
      return (seconds / size).toLocaleString(locale.value, { style: 'unit', unit, unitDisplay: 'short', maximumFractionDigits: 1 })
  }
  return Math.round(seconds).toLocaleString(locale.value, { style: 'unit', unit: 'second', unitDisplay: 'short' })
}
const bytes = (value: number) =>
  value >= 1024 * 1024
    ? (value / 1024 / 1024).toLocaleString(locale.value, { style: 'unit', unit: 'megabyte', maximumFractionDigits: 1 })
    : (value / 1024).toLocaleString(locale.value, { style: 'unit', unit: 'kilobyte', maximumFractionDigits: 1 })
const feedLabel = (feed: string) => (feedKeys[feed] ? t(feedKeys[feed]) : feed)
const variantLabel = (variant: string) => (variantKeys[variant] ? t(variantKeys[variant]) : variant || '—')
const metricLabel = (metric: string) => {
  if (metricKeys[metric]) return t(metricKeys[metric])
  const attributed = /^(exact|inferred)_(.+)$/.exec(metric)
  if (attributed && actionKeys[attributed[2]])
    return t(attributed[1] === 'exact' ? 'feedAdmin.metricExact' : 'feedAdmin.metricInferred', { action: t(actionKeys[attributed[2]]) })
  return metric
}

function sum(match: (row: Row) => boolean) {
  return rows.value.reduce((total, row) => (match(row) ? total + num(row.count) : total), 0)
}
const attributed = (row: Row, actions: string[]) =>
  actions.some((action) => row.metric === `exact_${action}` || row.metric === `inferred_${action}`)

const service = computed(() => {
  if (!summary.value?.enabled) return { label: t('feedAdmin.serviceOff'), hint: t('feedAdmin.serviceOffHint'), tone: 'muted' }
  if (!summary.value.rankingReady) return { label: t('feedAdmin.servicePreparing'), hint: t('feedAdmin.servicePreparingHint'), tone: 'warn' }
  return { label: t('feedAdmin.serviceRunning'), hint: t('feedAdmin.serviceRunningHint'), tone: 'ok' }
})
const healthWarn = computed(
  () => num(health.value.dropped) > 0 || num(health.value.backgroundFailures) > 0 || health.value.previousEpochIncomplete === true,
)

const funnel = computed(() => {
  const variants = byVariant.value
    ? [...new Set(rows.value.filter((row) => feedKeys[row.feed]).map((row) => row.variant || 'unassigned'))].sort()
    : ['']
  return funnelFeeds.flatMap((feed) =>
    variants.map((variant) => {
      const match = (row: Row) => row.feed === feed && (!variant || (row.variant || 'unassigned') === variant)
      const metric = (name: string) => sum((row) => match(row) && row.metric === name)
      const served = metric('served')
      const visible = metric('visible')
      const opened = metric('visible_open')
      return {
        key: `${feed}:${variant}`,
        feed,
        variant,
        served,
        visible,
        opened,
        read: metric('read_10s'),
        interactions: sum((row) => match(row) && attributed(row, interactionActions)),
        contributions: sum((row) => match(row) && attributed(row, contributionActions)),
        visibleRate: percent(visible, served),
        openRate: percent(opened, visible),
      }
    }),
  ).filter((row) => !byVariant.value || row.served + row.visible + row.opened > 0)
})
const funnelEmpty = computed(() => funnel.value.every((row) => row.served + row.visible + row.opened === 0))
const hasInferred = computed(() => rows.value.some((row) => row.metric.startsWith('inferred_') && num(row.count) > 0))
const columns = [
  ['colServed', 'colServedHint'],
  ['colVisible', 'colVisibleHint'],
  ['colVisibleRate', 'colVisibleRateHint'],
  ['colOpened', 'colOpenedHint'],
  ['colOpenRate', 'colOpenRateHint'],
  ['colRead', 'colReadHint'],
  ['colInteractions', 'colInteractionsHint'],
  ['colContributions', 'colContributionsHint'],
] as const

const cohort = computed(() => {
  const metric = (name: string) => sum((row) => row.feed === 'new_topics' && row.metric === name)
  const published = metric('new_topic_public')
  const seen = metric('new_topic_visible_24h')
  const replied = metric('new_topic_first_reply')
  const waitSeconds = metric('first_reply_seconds')
  return {
    published,
    seen,
    replied,
    seenShare: percent(seen, published),
    repliedShare: percent(replied, published),
    wait: replied > 0 ? duration(waitSeconds / replied) : '—',
  }
})

const assigned = computed(() => ({
  control: sum((row) => row.metric === 'assigned' && row.variant === 'control'),
  treatment: sum((row) => row.metric === 'assigned' && row.variant === 'treatment'),
}))
const signed = (value: number) => value.toLocaleString(locale.value, { maximumFractionDigits: 2, signDisplay: 'exceptZero' })
function effects(result: string): [string, Effect][] {
  try {
    const parsed = JSON.parse(result)
    if (!parsed?.effects || typeof parsed.effects !== 'object') return []
    return Object.keys(effectKeys)
      .filter((name) => parsed.effects[name] && typeof parsed.effects[name] === 'object')
      .map((name) => [name, parsed.effects[name] as Effect])
  } catch {
    return []
  }
}
const periods = computed(() =>
  (summary.value?.periods || []).map((period) => ({
    ...period,
    status: period.aborted ? 'aborted' : period.result ? 'completed' : 'collecting',
    reason: period.aborted === 'parameters changed' ? t('feedAdmin.abortedParams') : period.aborted,
    effects: period.result ? effects(period.result) : [],
  })),
)
const statusKeys = { aborted: 'feedAdmin.statusAborted', completed: 'feedAdmin.statusCompleted', collecting: 'feedAdmin.statusCollecting' } as const

const feedOptions = computed(() => [...new Set(rows.value.map((row) => row.feed))])
const filtered = computed(() => rows.value.filter((row) => filter.value === 'all' || row.feed === filter.value))
const rowPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize)))
const visibleRows = computed(() => filtered.value.slice((rowPage.value - 1) * pageSize, rowPage.value * pageSize))
function setFilter(value: unknown) {
  filter.value = String(value || 'all')
  rowPage.value = 1
}

// Parameters stay inspectable as labels and values, never as a JSON block.
function fields(value: unknown, prefix = ''): [string, string][] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return []
  return Object.entries(value).flatMap(([key, item]): [string, string][] => {
    const label = prefix ? `${prefix} / ${key}` : key
    if (item && typeof item === 'object' && !Array.isArray(item)) return fields(item, label)
    if (Array.isArray(item)) return [[label, item.map(String).join(' · ')]]
    return [[label, typeof item === 'boolean' ? t(item ? 'feedAdmin.enabled' : 'feedAdmin.disabled') : String(item ?? '—')]]
  })
}
const parameters = computed(() => fields(health.value.parameters))

async function load() {
  loading.value = true
  error.value = ''
  try {
    summary.value = await getFeedSummary()
    loadedAt.value = new Date()
    rowPage.value = 1
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('common.loadFailed')
  } finally {
    loading.value = false
  }
}
function download() {
  if (!summary.value) return
  const url = URL.createObjectURL(new Blob([JSON.stringify(summary.value, null, 2)], { type: 'application/json' }))
  const link = document.createElement('a')
  link.href = url
  link.download = 'feed-aggregates.json'
  link.click()
  URL.revokeObjectURL(url)
}
onMounted(load)

const toneDot = { ok: 'bg-emerald-500', warn: 'bg-amber-500', muted: 'bg-muted-foreground/50' } as const
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="flex items-center gap-2 text-sm text-muted-foreground">
        <ShieldCheck class="size-4 shrink-0" />
        <span>{{ t('feedAdmin.privacyNote') }}</span>
        <span v-if="loadedAt" class="hidden sm:inline">· {{ t('feedAdmin.loadedAt', { time: dateTime(loadedAt) }) }}</span>
      </p>
      <div class="flex gap-2">
        <Button variant="outline" size="sm" :disabled="loading" @click="load">
          <RefreshCw class="size-4" :class="{ 'animate-spin': loading }" />{{ t('feedAdmin.refresh') }}
        </Button>
        <Button variant="outline" size="sm" :disabled="!summary || loading" @click="download">
          <Download class="size-4" />{{ t('feedAdmin.export') }}
        </Button>
      </div>
    </div>

    <p v-if="error" role="alert" class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">{{ error }}</p>
    <div v-if="loading && !summary" role="status" class="rounded-lg border bg-card p-12 text-center text-muted-foreground">{{ t('common.loading') }}</div>

    <template v-if="summary">
      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <AdminSection body-class="space-y-1.5 p-4">
          <p class="text-sm text-muted-foreground">{{ t('feedAdmin.serviceLabel') }}</p>
          <p class="flex items-center gap-2 text-lg font-semibold">
            <span class="size-2 rounded-full" :class="toneDot[service.tone as keyof typeof toneDot]" />{{ service.label }}
          </p>
          <p class="text-xs leading-5 text-muted-foreground">{{ service.hint }}</p>
        </AdminSection>
        <AdminSection body-class="space-y-1.5 p-4">
          <p class="text-sm text-muted-foreground">{{ t('feedAdmin.rolloutLabel') }}</p>
          <p class="text-lg font-semibold tabular-nums">{{ percent(summary.rolloutPercent, 100) }}</p>
          <p class="text-xs leading-5 text-muted-foreground">{{ t('feedAdmin.rolloutHint') }}</p>
        </AdminSection>
        <AdminSection body-class="space-y-1.5 p-4" data-testid="capture-status">
          <p class="text-sm text-muted-foreground">{{ t('feedAdmin.captureLabel') }}</p>
          <p class="flex items-center gap-2 text-lg font-semibold">
            <span class="size-2 rounded-full" :class="summary.metricsEnabled ? toneDot.ok : toneDot.muted" />{{ t(summary.metricsEnabled ? 'feedAdmin.captureOn' : 'feedAdmin.captureOff') }}
          </p>
          <p class="text-xs leading-5 text-muted-foreground">{{ t('feedAdmin.captureHint', { days: summary.rawRetentionDays }) }}</p>
        </AdminSection>
        <AdminSection body-class="space-y-1.5 p-4">
          <p class="text-sm text-muted-foreground">{{ t('feedAdmin.healthLabel') }}</p>
          <p class="flex items-center gap-2 text-lg font-semibold">
            <span class="size-2 rounded-full" :class="healthWarn ? toneDot.warn : toneDot.ok" />{{ t(healthWarn ? 'feedAdmin.healthWarn' : 'feedAdmin.healthOk') }}
          </p>
          <p class="text-xs leading-5 tabular-nums text-muted-foreground">
            {{ t('feedAdmin.healthHint', { accepted: format(num(health.accepted)), dropped: format(num(health.dropped)), failures: format(num(health.backgroundFailures)) }) }}
          </p>
        </AdminSection>
      </div>
      <p v-if="health.previousEpochIncomplete === true" class="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-foreground">
        <AlertTriangle class="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />{{ t('feedAdmin.incomplete') }}
      </p>

      <AdminSection>
        <div class="flex flex-wrap items-end justify-between gap-3 px-4 pt-4">
          <div>
            <h2 class="text-base font-semibold">{{ t('feedAdmin.funnelTitle') }}</h2>
            <p class="text-sm text-muted-foreground">{{ t('feedAdmin.funnelWindow') }}</p>
          </div>
          <div role="group" :aria-label="t('feedAdmin.funnelTitle')" class="inline-flex rounded-lg bg-muted p-0.5 text-sm">
            <button
              v-for="option in [false, true]"
              :key="String(option)"
              type="button"
              :aria-pressed="byVariant === option"
              class="rounded-md px-3 py-1 font-medium transition-colors"
              :class="byVariant === option ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
              @click="byVariant = option"
            >{{ t(option ? 'feedAdmin.viewByVariant' : 'feedAdmin.viewTotal') }}</button>
          </div>
        </div>
        <div v-if="funnelEmpty" class="px-4 py-10 text-center">
          <p class="font-medium">{{ t('feedAdmin.noData') }}</p>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('feedAdmin.noDataHint') }}</p>
        </div>
        <div v-else class="mt-3 overflow-x-auto">
          <table class="w-full min-w-[760px] text-sm">
            <thead class="text-xs text-muted-foreground">
              <tr class="border-y bg-muted/40">
                <th scope="col" class="px-4 py-2 text-start font-medium">{{ t('feedAdmin.colFeed') }}</th>
                <th v-for="[label, hint] in columns" :key="label" scope="col" class="px-3 py-2 text-end font-medium">
                  <span class="cursor-help underline decoration-dotted decoration-muted-foreground/50 underline-offset-4" :title="t(`feedAdmin.${hint}`)">{{ t(`feedAdmin.${label}`) }}</span>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y tabular-nums">
              <tr v-for="row in funnel" :key="row.key" class="hover:bg-muted/30">
                <th scope="row" class="px-4 py-2.5 text-start font-medium">
                  {{ feedLabel(row.feed) }}
                  <span v-if="row.variant" class="ms-1 text-xs font-normal text-muted-foreground">{{ variantLabel(row.variant) }}</span>
                </th>
                <td class="px-3 py-2.5 text-end">{{ format(row.served) }}</td>
                <td class="px-3 py-2.5 text-end">{{ format(row.visible) }}</td>
                <td class="px-3 py-2.5 text-end text-muted-foreground">{{ row.visibleRate }}</td>
                <td class="px-3 py-2.5 text-end">{{ format(row.opened) }}</td>
                <td class="px-3 py-2.5 text-end text-muted-foreground">{{ row.openRate }}</td>
                <td class="px-3 py-2.5 text-end">{{ format(row.read) }}</td>
                <td class="px-3 py-2.5 text-end">{{ format(row.interactions) }}</td>
                <td class="px-3 py-2.5 text-end">{{ format(row.contributions) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="px-4 py-3 text-xs text-muted-foreground">
          <template v-if="hasInferred">{{ t('feedAdmin.inferredNote') }} </template>{{ t('feedAdmin.funnelNote') }}
        </p>
      </AdminSection>

      <div class="grid gap-6 xl:grid-cols-2">
        <AdminSection body-class="p-4">
          <h2 class="text-base font-semibold">{{ t('feedAdmin.newTitle') }}</h2>
          <p class="text-sm text-muted-foreground">{{ t('feedAdmin.newWindow') }}</p>
          <dl class="mt-4 grid grid-cols-2 gap-x-4 gap-y-5">
            <div>
              <dt class="text-xs text-muted-foreground">{{ t('feedAdmin.newPublished') }}</dt>
              <dd class="mt-1 text-xl font-semibold tabular-nums">{{ format(cohort.published) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">{{ t('feedAdmin.newSeen') }}</dt>
              <dd class="mt-1 text-xl font-semibold tabular-nums">{{ format(cohort.seen) }}</dd>
              <dd class="text-xs text-muted-foreground">{{ t('feedAdmin.newShare', { percent: cohort.seenShare }) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">{{ t('feedAdmin.newReplied') }}</dt>
              <dd class="mt-1 text-xl font-semibold tabular-nums">{{ format(cohort.replied) }}</dd>
              <dd class="text-xs text-muted-foreground">{{ t('feedAdmin.newShare', { percent: cohort.repliedShare }) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">{{ t('feedAdmin.newWait') }}</dt>
              <dd class="mt-1 text-xl font-semibold tabular-nums">{{ cohort.wait }}</dd>
            </div>
          </dl>
          <p class="mt-4 text-xs leading-5 text-muted-foreground">{{ t('feedAdmin.newNote') }}</p>
        </AdminSection>

        <AdminSection body-class="p-4">
          <h2 class="text-base font-semibold">{{ t('feedAdmin.expTitle') }}</h2>
          <p class="text-sm tabular-nums text-muted-foreground">{{ t('feedAdmin.expAssigned', { control: format(assigned.control), treatment: format(assigned.treatment) }) }}</p>
          <p v-if="!periods.length" class="mt-4 rounded-md bg-muted/40 px-3 py-6 text-center text-sm text-muted-foreground">{{ t('feedAdmin.periodsEmpty') }}</p>
          <ul v-else class="mt-4 space-y-3">
            <li v-for="period in periods" :key="period.id" class="rounded-md border p-3">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <span
                  class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="{
                    'bg-primary/10 text-primary': period.status === 'collecting',
                    'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300': period.status === 'completed',
                    'bg-muted text-muted-foreground': period.status === 'aborted',
                  }"
                >{{ t(statusKeys[period.status as keyof typeof statusKeys]) }}</span>
                <span class="text-xs tabular-nums text-muted-foreground">
                  {{ t('feedAdmin.control') }} {{ t('feedAdmin.people', { count: format(period.assignedControl) }) }} · {{ t('feedAdmin.treatment') }} {{ t('feedAdmin.people', { count: format(period.assignedTreatment) }) }}
                </span>
              </div>
              <dl class="mt-2 grid grid-cols-2 gap-2 text-xs">
                <div><dt class="text-muted-foreground">{{ t('feedAdmin.enrollUntil') }}</dt><dd class="tabular-nums">{{ dateTime(period.enrollUntil) }}</dd></div>
                <div><dt class="text-muted-foreground">{{ t('feedAdmin.analyzeAt') }}</dt><dd class="tabular-nums">{{ dateTime(period.analyzeAt) }}</dd></div>
              </dl>
              <p v-if="period.reason" class="mt-2 text-xs text-muted-foreground">{{ t('feedAdmin.abortedReason', { reason: period.reason }) }}</p>
              <table v-if="period.effects.length" class="mt-3 w-full text-xs">
                <thead class="text-muted-foreground">
                  <tr><th scope="col" class="py-1 text-start font-medium" /><th scope="col" class="py-1 text-end font-medium">{{ t('feedAdmin.effectDiff') }}</th></tr>
                </thead>
                <tbody class="divide-y tabular-nums">
                  <tr v-for="[name, effect] in period.effects" :key="name">
                    <th scope="row" class="py-1.5 text-start font-normal">{{ t(effectKeys[name]) }}</th>
                    <td class="py-1.5 text-end">
                      <span class="font-medium">{{ signed(num(effect.difference)) }}</span>
                      <span class="block text-muted-foreground">{{
                        effect.intervalAvailable && effect.lower95 != null && effect.upper95 != null
                          ? t('feedAdmin.interval', { lower: signed(effect.lower95), upper: signed(effect.upper95) })
                          : t('feedAdmin.insufficient')
                      }}</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </li>
          </ul>
          <p class="mt-4 text-xs leading-5 text-muted-foreground">{{ t('feedAdmin.resultLimitation') }}</p>
        </AdminSection>
      </div>

      <AdminSection>
        <button type="button" class="flex w-full items-center justify-between gap-3 px-4 py-3 text-start hover:bg-muted/30" :aria-expanded="showPipeline" aria-controls="feed-pipeline" @click="showPipeline = !showPipeline">
          <span class="font-medium">{{ t('feedAdmin.pipelineTitle') }}</span>
          <ChevronDown class="size-4 text-muted-foreground transition-transform" :class="{ 'rotate-180': showPipeline }" />
        </button>
        <dl v-show="showPipeline" id="feed-pipeline" class="grid gap-x-6 gap-y-3 border-t px-4 py-4 text-sm sm:grid-cols-2 lg:grid-cols-3">
          <div><dt class="text-xs text-muted-foreground">{{ t('feedAdmin.lastRank') }}</dt><dd class="tabular-nums">{{ dateTime(num(health.lastRankAt)) }}</dd></div>
          <div><dt class="text-xs text-muted-foreground">{{ t('feedAdmin.queueLength') }}</dt><dd class="tabular-nums">{{ format(num(health.queueLength)) }}</dd></div>
          <div><dt class="text-xs text-muted-foreground">{{ t('feedAdmin.queueBytes') }}</dt><dd class="tabular-nums">{{ bytes(num(health.queueBytes)) }}</dd></div>
          <div><dt class="text-xs text-muted-foreground">{{ t('feedAdmin.lag') }}</dt><dd class="tabular-nums">{{ duration(num(health.oldestQueuedMs) / 1000) }}</dd></div>
          <div><dt class="text-xs text-muted-foreground">{{ t('feedAdmin.sampleQueue') }}</dt><dd class="tabular-nums">{{ format(num(health.sampleQueueLength)) }}</dd></div>
          <div><dt class="text-xs text-muted-foreground">{{ t('feedAdmin.sampleBytes') }}</dt><dd class="tabular-nums">{{ bytes(num(health.sampleQueueBytes)) }}</dd></div>
        </dl>
      </AdminSection>

      <AdminSection>
        <button type="button" class="flex w-full items-center justify-between gap-3 px-4 py-3 text-start hover:bg-muted/30" :aria-expanded="showRaw" aria-controls="feed-raw" @click="showRaw = !showRaw">
          <span>
            <span class="block font-medium">{{ t('feedAdmin.rawTitle') }}</span>
            <span class="block text-xs text-muted-foreground">{{ t('feedAdmin.rawHint') }}</span>
          </span>
          <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="{ 'rotate-180': showRaw }" />
        </button>
        <div v-show="showRaw" id="feed-raw" class="space-y-3 border-t px-4 py-4">
          <p v-if="summary.truncated" role="status" class="rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-foreground">{{ t('feedAdmin.truncated') }}</p>
          <div class="flex flex-wrap items-center justify-between gap-3">
            <Select :model-value="filter" @update:model-value="setFilter">
              <SelectTrigger class="h-9 w-48" :aria-label="t('feedAdmin.feed')"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{{ t('feedAdmin.allFeeds') }}</SelectItem>
                <SelectItem v-for="feed in feedOptions" :key="feed" :value="feed">{{ feedLabel(feed) }}</SelectItem>
              </SelectContent>
            </Select>
            <span class="text-xs tabular-nums text-muted-foreground">{{ t('feedAdmin.rows', { count: format(filtered.length) }) }}</span>
          </div>
          <div class="overflow-x-auto rounded-md border">
            <table class="w-full min-w-[640px] text-sm">
              <thead class="bg-muted/40 text-xs text-muted-foreground">
                <tr>
                  <th scope="col" class="px-3 py-2 text-start font-medium">{{ t('feedAdmin.day') }}</th>
                  <th scope="col" class="px-3 py-2 text-start font-medium">{{ t('feedAdmin.feed') }}</th>
                  <th scope="col" class="px-3 py-2 text-start font-medium">{{ t('feedAdmin.variant') }}</th>
                  <th scope="col" class="px-3 py-2 text-start font-medium">{{ t('feedAdmin.metric') }}</th>
                  <th scope="col" class="px-3 py-2 text-start font-medium">{{ t('feedAdmin.versions') }}</th>
                  <th scope="col" class="px-3 py-2 text-end font-medium">{{ t('feedAdmin.count') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y">
                <tr v-if="!visibleRows.length"><td colspan="6" class="px-3 py-6 text-center text-muted-foreground">{{ t('feedAdmin.noData') }}</td></tr>
                <tr v-for="(row, index) in visibleRows" :key="`${row.day}-${row.feed}-${row.metric}-${row.variant}-${row.hash}-${row.capability}-${index}`">
                  <td class="px-3 py-2 tabular-nums">{{ row.day }}</td>
                  <td class="px-3 py-2">{{ feedLabel(row.feed) }}</td>
                  <td class="px-3 py-2 text-muted-foreground">{{ variantLabel(row.variant) }}</td>
                  <td class="px-3 py-2">{{ metricLabel(row.metric) }}</td>
                  <td class="px-3 py-2 font-mono text-xs text-muted-foreground" :title="row.hash">{{ row.hash.slice(0, 8) || '—' }}</td>
                  <td class="px-3 py-2 text-end tabular-nums">{{ format(num(row.count)) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-if="rowPages > 1" class="flex items-center justify-end gap-2">
            <Button variant="outline" size="sm" :disabled="rowPage <= 1" @click="rowPage--">{{ t('feedAdmin.previous') }}</Button>
            <span class="text-xs tabular-nums text-muted-foreground">{{ rowPage }} / {{ rowPages }}</span>
            <Button variant="outline" size="sm" :disabled="rowPage >= rowPages" @click="rowPage++">{{ t('feedAdmin.next') }}</Button>
          </div>
          <div v-if="parameters.length" class="pt-2">
            <h3 class="text-sm font-medium">{{ t('feedAdmin.parameters') }}</h3>
            <p class="font-mono text-xs text-muted-foreground" :title="summary.paramsHash">{{ t('feedAdmin.paramsHint', { hash: summary.paramsHash.slice(0, 12) }) }}</p>
            <dl class="mt-2 grid gap-x-6 gap-y-1.5 text-xs sm:grid-cols-2 lg:grid-cols-3">
              <div v-for="[label, value] in parameters" :key="label" class="flex justify-between gap-3 border-b border-dashed py-1">
                <dt class="truncate font-mono text-muted-foreground" :title="label">{{ label }}</dt>
                <dd class="shrink-0 tabular-nums">{{ value }}</dd>
              </div>
            </dl>
          </div>
        </div>
      </AdminSection>
    </template>
  </div>
</template>
