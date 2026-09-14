<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { Activity, ArrowDown, ArrowUp, Clock3, Cpu, Database, Globe2, HardDrive, Radio, RefreshCw, Users, Wifi } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { StatusRange, StatusServerRange, StatusSnapshot } from '@/types'
import { getStatus } from '@/runtime/status-api'
import { isRecentStatusTime } from '@/runtime/status-time'
import PageHeader from '@/components/PageHeader.vue'
import StatusTrafficChart from '@/components/StatusTrafficChart.vue'
import StatusResourceChart from '@/components/StatusResourceChart.vue'
import StatusUptime from '@/components/StatusUptime.vue'
import { uptimeMonitorState } from '@/runtime/uptime-status'

const { t, locale } = useI18n()
const range = ref<StatusRange>('24h')
const serverRange = ref<StatusServerRange>('1h')
const snapshot = ref<StatusSnapshot | null>(null)
const loading = ref(false)
const failed = ref(false)
const now = ref(Date.now())
let active = false
let sequence = 0
let controller: AbortController | undefined
let timer: ReturnType<typeof setTimeout> | undefined
let clock: ReturnType<typeof setInterval> | undefined

const server = computed(() => snapshot.value?.server.data)
// Keep current readings visible during a scope change, but never relabel an old curve.
const historyServer = computed(() => snapshot.value?.serverRange === serverRange.value ? server.value : null)
const current = computed(() => server.value?.current)
const traffic = computed(() => snapshot.value?.range === range.value ? snapshot.value.traffic.data : null)
const serverFresh = computed(() => sourceState('server') === 'sourceOk')
const signal = computed(() => {
  if (!snapshot.value && loading.value) return 'checking'
  const uptime = snapshot.value?.uptime
  if (uptime && uptime.state !== 'unconfigured') {
    const states = uptime.data?.monitors.map(monitor => uptimeMonitorState(monitor, sourceState('uptime') === 'sourceOk', now.value)) ?? []
    if (!states.length) return 'unknown'
    if (states.every(state => state === 'up')) return 'servicesUp'
    if (states.some(state => state === 'down')) return states.every(state => state === 'down') ? 'servicesDown' : 'servicesDegraded'
    if (states.some(state => state === 'unknown')) return 'unknown'
    if (states.some(state => state === 'maintenance')) return 'servicesMaintenance'
    return 'servicesPending'
  }
  if (!serverFresh.value) return 'unknown'
  return isRecentStatusTime(current.value?.observedAt, now.value, 150_000) ? 'live' : 'noSignal'
})
const rangeOptions = computed(() => [
  { value: '24h' as const, label: t('status.range24h') },
  { value: '7d' as const, label: t('status.range7d') },
  { value: '30d' as const, label: t('status.range30d') },
])
const serverRangeOptions = computed(() => (['1h', '6h', '24h', '7d'] as const).map(value => ({ value, label: t(`status.scope${value}`) })))

function number(value: number | null | undefined) {
  return value == null ? '—' : new Intl.NumberFormat(locale.value, value >= 100_000 ? { notation: 'compact', maximumFractionDigits: 1 } : {}).format(value)
}
function percent(value: number | null | undefined) { return value == null ? '—' : `${value.toFixed(1)}%` }
function bytes(value: number | undefined) {
  if (value == null) return '—'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  const index = value > 0 ? Math.min(Math.floor(Math.log(value) / Math.log(1024)), 4) : 0
  return `${(value / 1024 ** index).toFixed(index ? 1 : 0)} ${units[index]}`
}
function dateTime(value: string | undefined) {
  return value ? new Date(value).toLocaleString(locale.value, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '—'
}
function duration(value: number | null | undefined, long = false) {
  if (value == null) return '—'
  if (long && value >= 86400) return `${t('status.days', { count: Math.floor(value / 86400) })} ${t('status.hours', { count: Math.floor(value % 86400 / 3600) })}`
  if (long && value >= 3600) return t('status.hours', { count: Math.floor(value / 3600) })
  return `${t('status.minutes', { count: Math.floor(value / 60) })} ${t('status.seconds', { count: Math.floor(value % 60) })}`
}
function usage(used: number | undefined, total: number | undefined) { return used != null && total ? Math.min(100, used / total * 100) : undefined }
function sourceState(source: 'server' | 'traffic' | 'uptime') {
  const state = snapshot.value?.[source].state
  if (!state) return loading.value ? 'checking' : 'sourceUnavailable'
  if (state === 'ok' && (failed.value || !isRecentStatusTime(snapshot.value?.[source].fetchedAt, now.value, source === 'traffic' ? 600_000 : 150_000))) return 'sourceStale'
  return { ok: 'sourceOk', stale: 'sourceStale', unavailable: 'sourceUnavailable', unconfigured: 'sourceUnconfigured' }[state]
}
function sourceNote(source: 'server' | 'traffic') {
  const notes: Record<string, string> = { sourceStale: 'staleNote', sourceUnavailable: 'unavailableNote', sourceUnconfigured: 'unconfiguredNote' }
  return notes[sourceState(source)]
}

async function refresh() {
  if (!active || document.hidden) return
  clearTimeout(timer)
  controller?.abort()
  const request = ++sequence
  const requestController = new AbortController()
  controller = requestController
  // Bound the browser request too, including a stalled forum connection.
  const timeout = setTimeout(() => requestController.abort(), 12_000)
  loading.value = true
  try {
    const result = await getStatus(range.value, requestController.signal, serverRange.value)
    if (request !== sequence) return
    snapshot.value = result
    failed.value = false
  } catch {
    if (request === sequence) failed.value = true
  } finally {
    clearTimeout(timeout)
    if (request === sequence) {
      loading.value = false
      now.value = Date.now()
      if (active && !document.hidden) timer = setTimeout(refresh, 30_000)
    }
  }
}
function suspendRequest() {
  ++sequence
  controller?.abort()
  clearTimeout(timer)
  loading.value = false
}
function visibilityChanged() {
  now.value = Date.now()
  if (document.hidden) suspendRequest()
  else void refresh()
}
function start() {
  if (active) return
  active = true
  document.addEventListener('visibilitychange', visibilityChanged)
  clock = setInterval(() => { now.value = Date.now() }, 10_000)
  void refresh()
}
function stop() {
  active = false
  suspendRequest()
  clearInterval(clock)
  document.removeEventListener('visibilitychange', visibilityChanged)
}
watch([range, serverRange], () => { void refresh() })
onMounted(start)
onActivated(start)
onDeactivated(stop)
onBeforeUnmount(stop)

const resources = computed(() => [
  { key: 'cpu', icon: Cpu, value: percent(current.value?.cpu), amount: current.value?.cpu, detail: server.value ? t('status.cores', { count: server.value.cpuCores }) : '—' },
  { key: 'memory', icon: Database, value: percent(usage(current.value?.memoryUsed, current.value?.memoryTotal)), amount: usage(current.value?.memoryUsed, current.value?.memoryTotal), detail: t('status.usedOf', { used: bytes(current.value?.memoryUsed), total: bytes(current.value?.memoryTotal) }) },
  { key: 'disk', icon: HardDrive, value: percent(usage(current.value?.diskUsed, current.value?.diskTotal)), amount: usage(current.value?.diskUsed, current.value?.diskTotal), detail: t('status.usedOf', { used: bytes(current.value?.diskUsed), total: bytes(current.value?.diskTotal) }) },
])
</script>

<template>
  <div class="status-page">
    <PageHeader :title="t('status.title')" :description="t('status.description')" compact>
      <template #actions>
        <button class="status-refresh gf-button gf-button-sm gf-button-secondary" type="button" :disabled="loading" @click="refresh">
          <RefreshCw :class="{ spinning: loading }" :size="14" />{{ t(loading ? 'status.refreshing' : 'status.refresh') }}
        </button>
      </template>
    </PageHeader>

    <section class="status-hero" :class="{ 'is-live': signal === 'live' || signal === 'servicesUp', 'is-down': signal === 'servicesDown' || signal === 'servicesDegraded' }" aria-labelledby="status-signal">
      <div class="status-signal-label">
        <span class="signal-dot" aria-hidden="true"></span>
        <div class="status-signal-copy">
          <h2 id="status-signal" aria-live="polite">{{ t(`status.${signal}`) }}</h2>
          <p>{{ t('status.probeNote') }}</p>
        </div>
      </div>
      <p class="status-live-label"><Clock3 :size="14" aria-hidden="true" />{{ t('status.autoRefresh') }}</p>
    </section>
    <p v-if="failed" class="status-notice" role="alert">{{ t(snapshot ? 'status.loadFailed' : 'status.firstLoadFailed') }}</p>
    <StatusUptime :source="snapshot?.uptime" :now="now" :failed="failed" :loading="loading" />

    <section class="status-panel status-traffic" aria-labelledby="traffic-title" :aria-busy="loading">
      <div class="status-panel-heading"><div><h2 id="traffic-title">{{ t('status.trafficTitle') }}</h2><p>{{ t('status.trafficDescription') }}</p></div><div class="status-range" role="group" :aria-label="t('status.rangeLabel')"><button v-for="option in rangeOptions" :key="option.value" type="button" :aria-pressed="range === option.value" @click="range = option.value">{{ option.label }}</button></div></div>
      <p v-if="sourceNote('traffic')" class="status-notice">{{ t(`status.${sourceNote('traffic')}`) }}</p>
      <div class="status-traffic-grid">
        <div class="status-metric"><div class="metric-label"><Users :size="16" />{{ t('status.visitors') }}</div><strong :class="{ 'is-loading': loading && !traffic }">{{ number(traffic?.visitors) }}</strong><span class="metric-caption">{{ rangeOptions.find(option => option.value === range)?.label }}</span></div>
        <div class="status-metric"><div class="metric-label"><Globe2 :size="16" />{{ t('status.views') }}</div><strong :class="{ 'is-loading': loading && !traffic }">{{ number(traffic?.pageviews) }}</strong><span class="metric-caption">{{ t('status.visits') }} <b>{{ number(traffic?.visits) }}</b></span></div>
        <div class="status-metric metric-active"><div class="metric-label"><Radio :size="16" />{{ t('status.active') }}</div><strong :class="{ 'is-loading': loading && !traffic }">{{ number(traffic?.activeVisitors) }}</strong><span class="metric-caption">{{ t('status.activeNote') }}</span></div>
      </div>
      <div class="status-chart-panel">
        <div class="chart-heading"><h3>{{ t('status.trend') }}</h3><div class="status-legend"><span><i class="legend-views"></i>{{ t('status.views') }}</span><span><i class="legend-visitors"></i>{{ t('status.visitors') }}</span></div></div>
        <StatusTrafficChart v-if="traffic?.seriesAvailable && traffic.series.length" :traffic="traffic" :range="range" />
        <div v-else class="status-chart-empty"><Activity :size="26" :stroke-width="1.3" /><span>{{ t(loading && !traffic ? 'status.checking' : traffic?.seriesAvailable ? 'status.emptyChart' : 'status.chartUnavailable') }}</span></div>
        <div class="status-traffic-footer"><p>{{ t('status.trendNote') }}</p><dl><div><dt>{{ t('status.bounce') }}</dt><dd>{{ percent(traffic?.bounceRate) }}</dd></div><div><dt>{{ t('status.duration') }}</dt><dd>{{ duration(traffic?.averageDuration) }}</dd></div></dl></div>
      </div>
      <p v-if="traffic && (traffic.activeVisitors == null || !traffic.seriesAvailable)" class="status-probe-note">{{ t('status.partialNote') }}</p>
      <div class="status-panel-source status-source"><span>{{ t('status.dataSource') }} <b>Umami</b><span class="source-badge" :class="{ connected: sourceState('traffic') === 'sourceOk' }">{{ t(`status.${sourceState('traffic')}`) }}</span></span><time>{{ t('status.updated', { time: dateTime(snapshot?.traffic.fetchedAt) }) }}</time></div>
    </section>

    <section class="status-panel status-infra" aria-labelledby="infra-title">
      <div class="status-panel-heading"><div><h2 id="infra-title">{{ t('status.serverTitle') }}</h2><p>{{ t('status.serverDescription') }}</p></div><span class="status-node-tag"><span>{{ server?.region }}</span>{{ server?.name || t('status.node') }}</span></div>
      <p v-if="sourceNote('server')" class="status-notice">{{ t(`status.${sourceNote('server')}`) }}</p>
      <div class="status-resource-grid"><div v-for="resource in resources" :key="resource.key" class="status-resource"><div class="metric-label"><component :is="resource.icon" :size="16" />{{ t(`status.${resource.key}`) }}</div><strong>{{ resource.value }}</strong><div class="resource-track" role="meter" :aria-label="t(`status.${resource.key}`)" :aria-valuenow="resource.amount" :aria-valuetext="resource.value" :aria-valuemin="0" :aria-valuemax="100"><span :style="{ width: `${resource.amount ?? 0}%` }" :class="{ 'is-high': (resource.amount ?? 0) >= 85 }"></span></div><p>{{ resource.detail }}</p></div><div class="status-resource resource-network"><div class="metric-label"><Wifi :size="16" />{{ t('status.network') }}</div><div class="network-reading"><ArrowUp :size="15" /><span>{{ t('status.upload') }}</span><b>{{ bytes(current?.networkUp) }}<small>/s</small></b></div><div class="network-reading"><ArrowDown :size="15" /><span>{{ t('status.download') }}</span><b>{{ bytes(current?.networkDown) }}<small>/s</small></b></div></div></div>
      <div class="status-chart-panel resource-chart-panel" :aria-busy="loading">
        <div class="chart-heading">
          <div class="chart-title"><h3>{{ t('status.history') }}</h3><div class="status-legend"><span><i class="legend-views"></i>CPU</span><span><i class="legend-visitors"></i>{{ t('status.memory') }}</span></div></div>
          <div class="status-range status-resource-range" role="group" :aria-label="t('status.serverRangeLabel')">
            <button v-for="option in serverRangeOptions" :key="option.value" type="button" :aria-pressed="serverRange === option.value" @click="serverRange = option.value">{{ option.label }}</button>
          </div>
        </div>
        <StatusResourceChart v-if="historyServer?.historyAvailable && historyServer.history.length" :points="historyServer.history" :as-of="server?.historyFetchedAt ?? snapshot?.server.fetchedAt" :range="serverRange" />
        <div v-else class="status-chart-empty"><Activity :size="26" :stroke-width="1.3" /><span>{{ t(loading && !historyServer ? 'status.checking' : historyServer?.historyAvailable ? 'status.emptyChart' : 'status.chartUnavailable') }}</span></div>
      </div>
      <div class="status-host-meta"><Clock3 :size="14" /><span>{{ t('status.uptime') }} {{ duration(current?.uptime, true) }}</span></div>
      <p v-if="server?.historyStale" class="status-notice">{{ t('status.historyStale') }}</p>
      <p class="status-probe-note">{{ t('status.historyNote') }}</p>
      <div class="status-panel-source status-source"><span>{{ t('status.dataSource') }} <b>Komari</b><span class="source-badge" :class="{ connected: sourceState('server') === 'sourceOk' }">{{ t(`status.${sourceState('server')}`) }}</span></span><time>{{ t('status.sampleTime', { time: dateTime(current?.observedAt) }) }}</time></div>
    </section>
    <footer class="status-footer"><Activity :size="14" /><span>YourTJ</span><span class="footer-divider">/</span>{{ t('status.footnote') }}</footer>
  </div>
</template>

<style scoped>
.status-page {
  --status-primary: var(--gf-color-primary);
  --status-accent: var(--gf-color-accent);
  --status-value-size: 1.625rem;
  --status-chart-height: 156px;
  max-width: 1024px;
  margin: 0 auto;
  padding-bottom: 40px;
  color: var(--gf-color-base-content);
  font-size: 14px;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}
.status-page :deep(.gf-page-header) { margin-bottom: 20px; }
.status-page :deep(.gf-page-header h1) { letter-spacing: -.025em; }
.status-page :deep(.gf-page-header > div:first-child > p) { max-width: 42rem; text-wrap: pretty; }
.status-refresh {
  min-height: 36px;
  transition-property: transform, background-color, border-color, color, opacity;
  transition-duration: 120ms;
  transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
}
.status-refresh:active:not(:disabled) { transform: scale(.96); }
.status-refresh:disabled { cursor: wait; opacity: .62; }
.status-page button:focus-visible { outline: 2px solid var(--status-primary); outline-offset: 3px; }
.status-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px 24px;
  padding: 20px 22px;
  border: 1px solid var(--gf-color-line);
  border-radius: var(--gf-radius-box);
  background: var(--gf-color-base-100);
}
.status-signal-label { display: flex; align-items: flex-start; gap: 12px; min-width: 0; }
.status-signal-copy { min-width: 0; }
.status-signal-copy h2 { margin: 0; font-size: 15px; line-height: 1.4; font-weight: 600; text-wrap: balance; }
.status-signal-copy p { max-width: 58ch; margin-top: 4px; font-size: 12px; line-height: 1.55; color: var(--gf-color-icon-muted); text-wrap: pretty; }
.signal-dot {
  flex-shrink: 0;
  width: 9px;
  height: 9px;
  margin-top: 6px;
  /* deslop-ignore-next-line 19 -- a semantic status dot is intentionally circular */
  border-radius: 50%;
  background: var(--gf-color-icon-muted);
}
.is-live .signal-dot { background: var(--gf-color-success); }
.is-down .signal-dot { background: var(--gf-color-error); }
.status-live-label { display: inline-flex; align-items: center; gap: 6px; flex-shrink: 0; font-size: 12px; color: var(--gf-color-icon-muted); white-space: nowrap; }
.status-live-label svg { flex-shrink: 0; }
.status-panel {
  min-width: 0;
  margin-top: 24px;
  padding: 24px;
  border: 1px solid var(--gf-color-line);
  border-radius: var(--gf-radius-box);
  background: var(--gf-color-base-100);
}
.status-page :deep(.status-panel-heading) { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px 20px; margin-bottom: 20px; }
.status-page :deep(.status-panel-heading h2) { margin: 0; font-size: 18px; line-height: 1.35; font-weight: 600; letter-spacing: -.015em; text-wrap: balance; }
.status-page :deep(.status-panel-heading p) { max-width: 60ch; margin-top: 4px; font-size: 13px; color: var(--gf-color-icon-muted); line-height: 1.55; text-wrap: pretty; }
.status-range { display: flex; padding: 3px; gap: 2px; flex-shrink: 0; border: 0; border-radius: var(--gf-radius-field); background: var(--gf-color-base-200); box-shadow: inset 0 0 0 1px var(--gf-color-line); }
.status-range button {
  min-height: 30px;
  padding: 5px 11px;
  border-radius: calc(var(--gf-radius-field) - 2px);
  color: var(--gf-color-icon-muted);
  font-size: 12px;
  line-height: 18px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition-property: background-color, color, box-shadow, transform;
  transition-duration: 120ms;
  transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
}
.status-range button:active { transform: scale(.96); }
@media (hover: hover) { .status-range button:hover { color: var(--gf-color-base-content); } }
.status-range button[aria-pressed=true] { background: var(--gf-color-base-100); color: var(--gf-color-base-content); box-shadow: 0 1px 2px color-mix(in oklch, var(--gf-color-base-content) 8%, transparent); }
.status-traffic-grid, .status-resource-grid { display: grid; gap: 24px; margin-bottom: 24px; }
.status-traffic-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.status-metric, .status-resource { min-width: 0; padding: 0; }
.metric-label { display: flex; align-items: center; gap: 7px; font-size: 12px; line-height: 1.4; color: var(--gf-color-icon-muted); }
.metric-label svg { flex-shrink: 0; width: 14px; height: 14px; }
.status-metric strong, .status-resource strong { display: flex; align-items: center; gap: 10px; margin: 7px 0 5px; font-size: var(--status-value-size); font-weight: 600; letter-spacing: -.035em; line-height: 1.25; }
.metric-caption { display: block; min-height: 18px; font-size: 12px; color: var(--gf-color-icon-muted); }
.metric-caption b { color: var(--gf-color-base-content); font-weight: 500; margin-inline-start: 4px; }
.status-chart-panel { padding: 18px 0 0; border-top: 1px solid var(--gf-color-line); }
.chart-heading, .chart-title { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px 16px; }
.chart-heading { margin-bottom: 16px; }
.chart-heading h3 { font-size: 13px; font-weight: 600; }
.status-legend, .status-legend > span { display: flex; align-items: center; gap: 12px; font-size: 12px; color: var(--gf-color-icon-muted); }
.status-legend > span { gap: 6px; }
.status-legend i { height: 5px; width: 14px; border-radius: 999px; }
.legend-views { background: var(--status-primary); }
.legend-visitors { background: var(--status-accent); }
.status-chart-empty { min-height: calc(var(--status-chart-height) + 28px); display: flex; flex-direction: column; justify-content: center; align-items: center; gap: 10px; color: var(--gf-color-icon-muted); font-size: 12px; }
.status-traffic-footer { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; margin-top: 16px; }
.status-traffic-footer > p { max-width: 58ch; font-size: 12px; line-height: 1.55; color: var(--gf-color-icon-muted); text-wrap: pretty; }
.status-traffic-footer dl { display: flex; gap: 24px; flex-shrink: 0; }
.status-traffic-footer dt { color: var(--gf-color-icon-muted); font-size: 12px; }
.status-traffic-footer dd { margin-top: 3px; font-size: 14px; font-weight: 600; }
.status-page :deep(.status-panel-source) { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 8px 16px; margin-top: 16px; padding-top: 14px; border-top: 1px solid var(--gf-color-line); font-size: 12px; color: var(--gf-color-icon-muted); }
.status-source > span { display: inline-flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.status-source b { font-weight: 600; color: var(--gf-color-base-content); }
.source-badge { display: inline-flex; align-items: center; color: var(--gf-color-icon-muted); font-size: 12px; font-weight: 500; }
.source-badge.connected { color: var(--gf-color-success); }
.status-node-tag { display: inline-flex; align-items: center; gap: 6px; min-height: 30px; padding: 5px 9px; font-size: 12px; background: var(--gf-color-base-200); border-radius: var(--gf-radius-field); box-shadow: inset 0 0 0 1px var(--gf-color-line); }
.status-resource-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); }
.resource-track { height: 4px; margin-top: 10px; border-radius: 999px; overflow: hidden; background: var(--gf-color-base-300); }
.resource-track > span { display: block; height: 100%; background: var(--status-primary); border-radius: inherit; }
.resource-track > span.is-high { background: var(--gf-color-warning); }
.status-resource p { font-size: 12px; color: var(--gf-color-icon-muted); margin-top: 8px; }
.resource-network { display: flex; flex-direction: column; }
.network-reading { display: grid; grid-template-columns: auto auto 1fr; align-items: center; gap: 5px; margin-top: 10px; font-size: 12px; color: var(--gf-color-icon-muted); }
.network-reading > svg { color: var(--gf-color-icon-muted); flex-shrink: 0; }
.network-reading b { justify-self: end; color: var(--gf-color-base-content); font-weight: 600; white-space: nowrap; }
.network-reading small { font-size: 12px; color: var(--gf-color-icon-muted); font-weight: 400; }
.status-host-meta { display: flex; align-items: center; gap: 6px; margin-top: 14px; font-size: 12px; color: var(--gf-color-icon-muted); }
.status-probe-note { margin-top: 10px; font-size: 12px; line-height: 1.55; color: var(--gf-color-icon-muted); text-wrap: pretty; }
.status-notice { padding: 10px 12px; margin: 14px 0; border: 1px solid var(--gf-color-line); border-radius: var(--gf-radius-field); background: var(--gf-color-base-200); font-size: 12px; line-height: 1.55; }
.status-footer { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 8px; padding-top: 24px; color: var(--gf-color-icon-muted); font-size: 12px; }
.status-footer > span:first-of-type { font-weight: 600; color: var(--gf-color-base-content); }
.footer-divider { opacity: .35; }
.spinning { animation: status-spin 1.4s linear infinite; }
.is-loading { opacity: .42; }
@keyframes status-spin { to { transform: rotate(360deg); } }
@media (max-width: 1100px) {
  .status-page { max-width: 896px; }
  .status-resource-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 639.98px) {
  .status-page { --status-chart-height: 136px; padding: 0 0 28px; }
  .status-page :deep(.gf-page-header) { padding: 12px 0 16px; margin-bottom: 12px; }
  .status-hero { align-items: flex-start; flex-direction: column; gap: 12px; padding: 17px 16px; }
  .status-live-label { padding-inline-start: 21px; }
  .status-panel { margin-top: 16px; padding: 18px 16px; }
  .status-page :deep(.status-panel-heading) { align-items: flex-start; margin-bottom: 16px; }
  .status-range { width: 100%; }
  .status-range button { flex: 1; padding-inline: 5px; }
  .status-traffic-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .status-metric:nth-child(3) { grid-column: 1 / -1; }
  .status-traffic-grid, .status-resource-grid { gap: 20px 16px; }
  .status-metric, .status-resource { padding: 0; }
  .status-resource-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .status-chart-panel { padding: 14px 0 0; }
  .status-traffic-footer { align-items: flex-start; flex-direction: column-reverse; gap: 12px; }
  .status-traffic-footer > p { max-width: 100%; }
  .status-traffic-footer dl { width: 100%; justify-content: space-between; gap: 16px; }
  .chart-heading { align-items: flex-start; gap: 10px; }
  .status-page :deep(.status-panel-source) { align-items: flex-start; flex-direction: column; }
}
@media (max-width: 359.98px) {
  .status-traffic-grid, .status-resource-grid { grid-template-columns: 1fr; }
  .status-metric:nth-child(3) { grid-column: auto; }
}
@media (prefers-reduced-motion: reduce) {
  .status-refresh, .status-range button { transition: none; }
  .spinning { animation: none; }
}
</style>
