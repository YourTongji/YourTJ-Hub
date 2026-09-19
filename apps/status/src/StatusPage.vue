<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { Activity, ArrowDown, ArrowUp, CircleAlert, CircleCheck, CircleHelp, CircleX, Clock3, Cpu, Database, Globe2, HardDrive, Radio, RefreshCw, Users, Wifi } from '@lucide/vue'
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
const currentYear = new Date().getFullYear()
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
const signalPresentation = computed(() => {
  switch (signal.value) {
    case 'servicesUp':
    case 'live':
      return { icon: CircleCheck, tone: 'ok' }
    case 'servicesDown':
      return { icon: CircleX, tone: 'error' }
    case 'servicesDegraded':
    case 'servicesMaintenance':
    case 'servicesPending':
      return { icon: CircleAlert, tone: 'warning' }
    default:
      return { icon: CircleHelp, tone: 'muted' }
  }
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
const sourceItems = [
  { key: 'server' as const, name: 'Komari', logo: '/source-logos/komari.png' },
  { key: 'traffic' as const, name: 'Umami', logo: '/source-logos/umami.svg' },
  { key: 'uptime' as const, name: 'Uptime Kuma', logo: '/source-logos/uptime-kuma.svg' },
]
function sourceStatusClass(source: 'server' | 'traffic' | 'uptime') {
  if (source === 'uptime' && sourceState('uptime') === 'sourceOk' && snapshot.value?.uptime.data?.monitors.some(monitor => uptimeMonitorState(monitor, true, now.value) === 'down')) return 'status-problem'
  return sourceState(source) === 'sourceOk' ? 'status-ok' : 'status-muted'
}
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

    <section class="status-hero" aria-labelledby="status-signal">
      <div class="status-hero-main">
        <div class="status-signal-label">
          <div class="status-signal-copy">
            <div class="status-signal-heading">
              <span class="status-signal-icon" :class="`is-${signalPresentation.tone}`" aria-hidden="true"><component :is="signalPresentation.icon" :size="30" :stroke-width="1.8" /></span>
              <h2 id="status-signal" aria-live="polite">{{ t(`status.${signal}`) }}</h2>
            </div>
            <p>{{ t('status.probeNote') }}</p>
          </div>
        </div>
        <p class="status-live-label"><Clock3 :size="14" aria-hidden="true" />{{ t('status.autoRefresh') }}</p>
      </div>
      <div class="status-source-rail" role="list" :aria-label="t('status.dataSource')">
        <div v-for="source in sourceItems" :key="source.key" class="status-source-chip" :class="[{ connected: sourceStatusClass(source.key) === 'status-ok' }, sourceStatusClass(source.key)]" role="listitem">
          <span class="source-logo-ring" :class="[sourceStatusClass(source.key), `brand-${source.key}`]" aria-hidden="true"><img :src="source.logo" :alt="`${source.name} logo`" /></span>
          <b>{{ source.name }}</b><span>{{ t(`status.${sourceState(source.key)}`) }}</span>
        </div>
      </div>
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
    <footer class="status-footer">{{ t('status.footnote', { year: currentYear, separator: '|' }) }}</footer>
  </div>
</template>

<style scoped>
.status-page {
  --status-primary: var(--gf-color-primary);
  --status-accent: var(--gf-color-accent);
  --status-value-size: 1.625rem;
  --status-chart-height: 156px;
  position: relative;
  max-width: 1120px;
  margin: 0 auto;
  padding-bottom: 56px;
  color: var(--gf-color-base-content);
  font-size: 14px;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}
.status-page :deep(.gf-page-header) { position: relative; align-items: flex-end; margin-bottom: 22px; padding: 10px 4px 2px; }
.status-page :deep(.gf-page-header h1) { font-size: clamp(1.8rem, 1.45rem + 1.4vw, 2.5rem); line-height: 1.2; letter-spacing: -.055em; }
.status-page :deep(.gf-page-header-title) {
  color: transparent;
  -webkit-text-fill-color: transparent;
  -webkit-text-stroke: 1.15px color-mix(in oklch, var(--gf-color-base-content) 44%, transparent);
  -webkit-mask-image: linear-gradient(to bottom, #000 0%, #000 54%, transparent 100%);
  mask-image: linear-gradient(to bottom, #000 0%, #000 54%, transparent 100%);
}
.status-page :deep(.gf-page-header > div:first-child > p) { max-width: 42rem; margin-top: 8px; text-wrap: pretty; }
.status-refresh {
  min-height: 38px;
  padding-inline: 15px;
  border-color: color-mix(in oklch, var(--status-primary) 24%, var(--gf-color-line));
  background: color-mix(in oklch, var(--status-primary) 8%, var(--gf-color-base-100));
  color: var(--status-primary);
  transition-property: transform, background-color, border-color, color, opacity;
  transition-duration: 120ms;
  transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
}
.status-refresh:hover { border-color: color-mix(in oklch, var(--status-primary) 44%, var(--gf-color-line)); background: color-mix(in oklch, var(--status-primary) 13%, var(--gf-color-base-100)); }
.status-refresh:active:not(:disabled) { transform: scale(.96); }
.status-refresh:disabled { cursor: wait; opacity: .62; }
.status-page button:focus-visible { outline: 2px solid var(--status-primary); outline-offset: 3px; }
.status-hero {
  position: relative;
  isolation: isolate;
  overflow: hidden;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(330px, .92fr);
  align-items: center;
  gap: 24px;
  min-height: 176px;
  padding: 28px 30px;
  border: 1px solid color-mix(in oklch, var(--status-primary) 19%, var(--gf-color-line));
  border-radius: var(--gf-radius-box);
  background: var(--gf-color-base-100);
  box-shadow: inset 0 1px 0 color-mix(in oklch, var(--gf-color-base-content) 8%, transparent), 0 5px 16px -10px color-mix(in oklch, var(--gf-color-base-content) 20%, transparent);
}
.status-hero-main { display: flex; flex-direction: column; align-items: flex-start; gap: 22px; min-width: 0; }
.status-signal-label { min-width: 0; }
.status-signal-copy { min-width: 0; }
.status-signal-heading { display: inline-flex; align-items: center; gap: 10px; max-width: 100%; }
.status-signal-icon { display: inline-flex; align-items: center; justify-content: center; flex: 0 0 auto; width: 30px; height: 30px; }
.status-signal-icon.is-ok { color: var(--gf-color-success); }
.status-signal-icon.is-error { color: var(--gf-color-error); }
.status-signal-icon.is-warning { color: var(--gf-color-warning); }
.status-signal-icon.is-muted { color: var(--gf-color-icon-muted); }
.status-signal-copy h2 { margin: 0; font-size: clamp(1.35rem, 1.1rem + .9vw, 1.85rem); line-height: 1.18; font-weight: 700; letter-spacing: -.045em; text-wrap: balance; }
.status-signal-copy p { max-width: 58ch; margin-top: 4px; font-size: 12px; line-height: 1.55; color: var(--gf-color-icon-muted); text-wrap: pretty; }
.status-live-label { display: inline-flex; align-items: center; gap: 6px; flex-shrink: 0; padding: 0; font-size: 12px; color: var(--gf-color-icon-muted); white-space: nowrap; }
.status-live-label svg { flex-shrink: 0; }
.status-source-rail { position: relative; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; min-width: 0; }
.status-source-chip { display: grid; grid-template-columns: auto minmax(0, 1fr); grid-template-rows: auto auto; align-items: center; column-gap: 8px; min-width: 0; padding: 10px 11px; border: 1px solid var(--gf-color-line); border-radius: var(--gf-radius-field); background: var(--gf-color-base-200); color: var(--gf-color-icon-muted); }
.source-logo-ring { grid-row: span 2; display: grid; place-items: center; width: 32px; height: 32px; border: 1.5px solid var(--gf-color-icon-muted); border-radius: 50%; background: #fff; overflow: hidden; }
.source-logo-ring img { width: 20px; height: 20px; object-fit: contain; }
.source-logo-ring.brand-server img { border-radius: 50%; }
.source-logo-ring.status-ok { border-color: var(--gf-color-success); }
.source-logo-ring.status-problem { border-color: var(--gf-color-error); }
.source-logo-ring.status-muted { border-color: var(--gf-color-icon-muted); }
.status-source-chip b { min-width: 0; overflow: hidden; color: var(--gf-color-base-content); font-size: 12px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.status-source-chip span { min-width: 0; overflow: hidden; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.status-panel {
  min-width: 0;
  position: relative;
  isolation: isolate;
  overflow: hidden;
  margin-top: 20px;
  padding: 26px;
  border: 1px solid color-mix(in oklch, var(--gf-color-line) 92%, transparent);
  border-radius: var(--gf-radius-box);
  background: var(--gf-color-base-100);
  box-shadow: inset 0 1px 0 color-mix(in oklch, var(--gf-color-base-content) 6%, transparent), 0 5px 16px -10px color-mix(in oklch, var(--gf-color-base-content) 18%, transparent);
}
.status-panel > * { position: relative; z-index: 1; }
.status-page :deep(.status-panel-heading) { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px 20px; margin-bottom: 22px; }
.status-page :deep(.status-panel-heading h2) { margin: 0; font-size: 19px; line-height: 1.3; font-weight: 700; letter-spacing: -.035em; text-wrap: balance; }
.status-page :deep(.status-panel-heading p) { max-width: 60ch; margin-top: 5px; font-size: 13px; color: var(--gf-color-icon-muted); line-height: 1.55; text-wrap: pretty; }
.status-range { display: flex; padding: 3px; gap: 2px; flex-shrink: 0; border: 1px solid var(--gf-color-line); border-radius: 999px; background: color-mix(in oklch, var(--gf-color-base-200) 86%, transparent); box-shadow: inset 0 1px 0 color-mix(in oklch, var(--gf-color-base-content) 5%, transparent); }
.status-range button {
  min-height: 30px;
  padding: 5px 11px;
  border-radius: 999px;
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
.status-range button[aria-pressed=true] { background: var(--gf-color-base-100); color: var(--gf-color-base-content); box-shadow: 0 2px 6px color-mix(in oklch, var(--gf-color-base-content) 10%, transparent), inset 0 0 0 1px color-mix(in oklch, var(--status-primary) 18%, transparent); }
.status-traffic-grid, .status-resource-grid { display: grid; gap: 12px; margin-bottom: 26px; }
.status-traffic-grid { grid-template-columns: repeat(12, minmax(0, 1fr)); }
.status-metric, .status-resource { min-width: 0; padding: 16px 18px; border: 1px solid var(--gf-color-line); border-radius: .75rem; background: color-mix(in oklch, var(--gf-color-base-200) 52%, transparent); box-shadow: inset 0 1px 0 color-mix(in oklch, var(--gf-color-base-content) 5%, transparent); }
.status-traffic-grid .status-metric:nth-child(1) { grid-column: span 5; }
.status-traffic-grid .status-metric:nth-child(2) { grid-column: span 4; }
.status-traffic-grid .status-metric:nth-child(3) { grid-column: span 3; }
.status-metric { min-height: 112px; transition-property: transform, border-color, background-color, box-shadow; transition-duration: 180ms; transition-timing-function: cubic-bezier(.2, 0, 0, 1); }
@media (hover: hover) { .status-metric:hover, .status-resource:hover { transform: translateY(-2px); border-color: color-mix(in oklch, var(--status-primary) 24%, var(--gf-color-line)); background: color-mix(in oklch, var(--status-primary) 5%, var(--gf-color-base-200)); box-shadow: inset 0 1px 0 color-mix(in oklch, var(--gf-color-base-content) 6%, transparent), 0 10px 24px color-mix(in oklch, var(--gf-color-base-content) 6%, transparent); } }
.metric-label { display: flex; align-items: center; gap: 7px; font-size: 12px; line-height: 1.4; color: var(--gf-color-icon-muted); }
.metric-label svg { flex-shrink: 0; width: 14px; height: 14px; }
.status-metric strong, .status-resource strong { display: flex; align-items: center; gap: 10px; margin: 9px 0 5px; font-size: clamp(1.55rem, 1.2rem + 1vw, 2rem); font-weight: 700; letter-spacing: -.055em; line-height: 1.1; }
.metric-caption { display: block; min-height: 18px; font-size: 12px; color: var(--gf-color-icon-muted); }
.metric-caption b { color: var(--gf-color-base-content); font-weight: 500; margin-inline-start: 4px; }
.status-chart-panel { padding: 21px 0 0; border-top: 1px solid var(--gf-color-line); }
.chart-heading, .chart-title { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px 16px; }
.chart-heading { margin-bottom: 17px; }
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
.status-page :deep(.status-panel-source) { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 8px 16px; margin-top: 20px; padding-top: 15px; border-top: 1px solid var(--gf-color-line); font-size: 12px; color: var(--gf-color-icon-muted); }
.status-page :deep(.status-source > span) { display: inline-flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.status-page :deep(.status-source b) { font-weight: 600; color: var(--gf-color-base-content); }
.status-page :deep(.source-badge) { display: inline-flex; align-items: center; color: var(--gf-color-icon-muted); font-size: 12px; font-weight: 500; }
.status-page :deep(.source-badge.connected) { color: var(--gf-color-success); }
.status-node-tag { display: inline-flex; align-items: center; gap: 6px; min-height: 30px; padding: 5px 10px; border: 1px solid var(--gf-color-line); font-size: 12px; background: color-mix(in oklch, var(--gf-color-base-200) 72%, transparent); border-radius: 999px; }
.status-resource-grid { grid-template-columns: repeat(12, minmax(0, 1fr)); }
.status-resource-grid .status-resource:nth-child(1) { grid-column: span 3; }
.status-resource-grid .status-resource:nth-child(2) { grid-column: span 3; }
.status-resource-grid .status-resource:nth-child(3) { grid-column: span 3; }
.status-resource-grid .status-resource:nth-child(4) { grid-column: span 3; }
.resource-track { height: 4px; margin-top: 10px; border-radius: 999px; overflow: hidden; background: var(--gf-color-base-300); }
.resource-track > span { display: block; height: 100%; background: var(--status-primary); border-radius: inherit; transition: width 480ms cubic-bezier(.16, 1, .3, 1); }
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
.status-footer { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 8px; margin-top: 30px; padding-top: 24px; border-top: 1px solid var(--gf-color-line); color: var(--gf-color-icon-muted); font-size: 12px; }
.spinning { animation: status-spin 1.4s linear infinite; }
.is-loading { opacity: .42; }
@keyframes status-spin { to { transform: rotate(360deg); } }
@media (max-width: 1100px) {
  .status-page { max-width: 1024px; }
  .status-hero { grid-template-columns: 1fr; }
  .status-source-rail { max-width: 660px; }
  .status-resource-grid { grid-template-columns: repeat(6, minmax(0, 1fr)); }
  .status-resource-grid .status-resource:nth-child(1), .status-resource-grid .status-resource:nth-child(2) { grid-column: span 2; }
  .status-resource-grid .status-resource:nth-child(3), .status-resource-grid .status-resource:nth-child(4) { grid-column: span 2; }
}
@media (max-width: 639.98px) {
  .status-page { --status-chart-height: 136px; padding: 0 0 28px; }
  .status-page :deep(.gf-page-header) { padding: 12px 0 16px; margin-bottom: 12px; }
  .status-hero { align-items: stretch; gap: 20px; padding: 22px 16px; }
  .status-hero-main { gap: 16px; }
  .status-live-label { padding-inline-start: 9px; }
  .status-source-rail { grid-template-columns: 1fr; }
  .status-panel { margin-top: 16px; padding: 20px 16px; }
  .status-page :deep(.status-panel-heading) { align-items: flex-start; margin-bottom: 16px; }
  .status-range { width: 100%; }
  .status-range button { flex: 1; padding-inline: 5px; }
  .status-traffic-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .status-traffic-grid .status-metric:nth-child(1), .status-traffic-grid .status-metric:nth-child(2) { grid-column: span 1; }
  .status-traffic-grid .status-metric:nth-child(3) { grid-column: 1 / -1; }
  .status-traffic-grid, .status-resource-grid { gap: 20px 16px; }
  .status-metric, .status-resource { padding: 14px; }
  .status-resource-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .status-resource-grid .status-resource:nth-child(1), .status-resource-grid .status-resource:nth-child(2), .status-resource-grid .status-resource:nth-child(3), .status-resource-grid .status-resource:nth-child(4) { grid-column: span 1; }
  .status-chart-panel { padding: 14px 0 0; }
  .status-traffic-footer { align-items: flex-start; flex-direction: column-reverse; gap: 12px; }
  .status-traffic-footer > p { max-width: 100%; }
  .status-traffic-footer dl { width: 100%; justify-content: space-between; gap: 16px; }
  .chart-heading { align-items: flex-start; gap: 10px; }
  .status-page :deep(.status-panel-source) { align-items: flex-start; flex-direction: column; }
}
@media (max-width: 359.98px) {
  .status-traffic-grid, .status-resource-grid { grid-template-columns: 1fr; }
  .status-traffic-grid .status-metric:nth-child(1), .status-traffic-grid .status-metric:nth-child(2), .status-traffic-grid .status-metric:nth-child(3), .status-resource-grid .status-resource:nth-child(1), .status-resource-grid .status-resource:nth-child(2), .status-resource-grid .status-resource:nth-child(3), .status-resource-grid .status-resource:nth-child(4) { grid-column: auto; }
}
@media (prefers-reduced-motion: reduce) {
  .status-refresh, .status-range button, .status-metric, .status-resource, .resource-track > span { transition: none; }
  .spinning { animation: none; }
}
</style>
