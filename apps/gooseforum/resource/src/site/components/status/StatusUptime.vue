<script setup lang="ts">
import { computed, onBeforeUnmount, onDeactivated, ref } from 'vue'
import { Activity, ArrowUpRight } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { StatusSnapshot, StatusUptimeMonitor } from '@gooseforum/client'
import { uptimeMonitorState } from '@/runtime/uptime-status'
import { isRecentStatusTime } from '@/runtime/status-time'

const props = defineProps<{ source?: StatusSnapshot['uptime']; now: number; failed: boolean; loading: boolean }>()
const { t, locale } = useI18n()
const selected = ref<{ id: number; point: NonNullable<StatusUptimeMonitor['current']> } | null>(null)
const detailVisible = ref(false)
let hideTimer: ReturnType<typeof setTimeout> | undefined
function cancelHide() {
  clearTimeout(hideTimer)
  hideTimer = undefined
}
function showDetail(id: number, point: NonNullable<StatusUptimeMonitor['current']>) {
  cancelHide()
  selected.value = { id, point }
  detailVisible.value = true
}
function hideDetail(event: MouseEvent | FocusEvent) {
  const history = event.currentTarget as HTMLElement
  // Moving between checks (or retaining keyboard focus) must not dismiss the detail.
  if (event.type === 'focusout' && history.contains((event as FocusEvent).relatedTarget as Node | null)) return
  if (event.type === 'mouseleave' && history.contains(document.activeElement)) return
  cancelHide()
  hideTimer = setTimeout(() => { detailVisible.value = false; hideTimer = undefined }, 120)
}
function dismissDetail() {
  cancelHide()
  detailVisible.value = false
}
onDeactivated(dismissDetail)
onBeforeUnmount(cancelHide)
const fresh = computed(() => !props.failed && props.source?.state === 'ok' && isRecentStatusTime(props.source.fetchedAt, props.now, 90_000))
const sourceState = computed(() => props.source?.state === 'ok' && !fresh.value ? 'stale' : props.source?.state ?? 'unavailable')
const state = (monitor: StatusUptimeMonitor) => uptimeMonitorState(monitor, fresh.value, props.now)
const formatDate = (value?: string) => value ? new Date(value).toLocaleString(locale.value, { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : '—'
const ping = (value: number | null | undefined) => value == null ? '—' : `${new Intl.NumberFormat(locale.value, { maximumFractionDigits: 1 }).format(value)} ms`
const beatLabel = (point: NonNullable<StatusUptimeMonitor['current']>) => `${formatDate(point.time)} · ${t(`status.check${point.status}`)} · ${ping(point.ping)}`
</script>

<template>
  <section class="status-panel uptime-section" aria-labelledby="uptime-title" :aria-busy="loading">
    <div class="status-panel-heading uptime-heading"><div><h2 id="uptime-title">{{ t('status.availabilityTitle') }}</h2><p>{{ t('status.availabilityDescription') }}</p></div><a v-if="source?.data" class="uptime-external" :href="source.data.statusPageUrl" target="_blank" rel="noopener noreferrer">{{ t('status.independentPage') }}<ArrowUpRight :size="14" /></a></div>
    <p v-if="sourceState !== 'ok'" class="uptime-notice">{{ t(loading && !source ? 'status.checking' : sourceState === 'stale' ? 'status.staleNote' : sourceState === 'unconfigured' ? 'status.unconfiguredNote' : 'status.unavailableNote') }}</p>
    <div class="uptime-monitors">
      <article v-for="monitor in source?.data?.monitors" :key="monitor.id" class="uptime-monitor">
        <div class="monitor-heading"><div><Activity :size="17" /><h3>{{ monitor.name }}</h3><span class="monitor-type">{{ monitor.type }}</span></div><span class="monitor-state" :class="`check-${state(monitor)}`"><i></i>{{ t(`status.check${state(monitor)}`) }}</span></div>
        <div class="uptime-metrics"><div><p>{{ t('status.availability24h') }}</p><strong class="uptime-percentage">{{ monitor.uptime24h == null ? '—' : `${monitor.uptime24h.toFixed(2)}%` }}</strong><small>{{ t('status.collectedCoverage') }}</small></div><div><p>{{ t('status.latestResponse') }}</p><strong class="uptime-ping">{{ ping(monitor.current?.ping) }}</strong><small>{{ t('status.checkTime', { time: formatDate(monitor.current?.time) }) }}</small></div></div>
        <div class="heartbeat-history" @mouseenter="cancelHide" @mouseleave="hideDetail" @focusout="hideDetail" @keydown.esc="dismissDetail">
          <div v-if="monitor.history.length" class="heartbeat-strip" :aria-label="t('status.recentChecks', { count: monitor.history.length })">
            <button v-for="(point, index) in monitor.history" :key="`${point.time}-${index}`" type="button" :class="`check-${point.status}`" :aria-label="beatLabel(point)" @focus="showDetail(monitor.id, point)" @mouseenter="showDetail(monitor.id, point)" @click="showDetail(monitor.id, point)"></button>
          </div>
          <p v-else class="uptime-no-checks">{{ t('status.noChecks') }}</p>
          <div class="heartbeat-footer" :class="{ 'is-active': detailVisible && selected?.id === monitor.id }">
            <div class="heartbeat-caption" :aria-hidden="detailVisible && selected?.id === monitor.id"><span>{{ t('status.recentChecks', { count: monitor.history.length }) }}</span><span>{{ formatDate(monitor.history[0]?.time) }} — {{ formatDate(monitor.history.at(-1)?.time) }}</span></div>
            <p class="heartbeat-detail" role="status" :aria-hidden="!detailVisible || selected?.id !== monitor.id">{{ selected?.id === monitor.id ? beatLabel(selected.point) : '' }}</p>
          </div>
        </div>
      </article>
    </div>
    <p v-if="source?.data && !source.data.monitors.length" class="uptime-no-checks">{{ t('status.noMonitors') }}</p>
    <div class="status-panel-source uptime-source"><span>Uptime Kuma · {{ t(`status.uptimeSource${sourceState}`) }}</span><span>{{ t('status.updated', { time: formatDate(source?.fetchedAt) }) }}</span></div>
  </section>
</template>

<style scoped>
.uptime-external { display: inline-flex; align-items: center; gap: 5px; flex-shrink: 0; font-size: 12px; color: var(--gf-color-primary); padding: 6px 0; }
.uptime-monitors { display: grid; gap: 20px; }
.uptime-monitor { min-width: 0; }
.uptime-monitor + .uptime-monitor { padding-top: 20px; border-top: 1px solid var(--gf-color-line); }
.monitor-heading, .monitor-heading > div { display: flex; align-items: center; gap: 8px; }
.monitor-heading { justify-content: space-between; flex-wrap: wrap; }
.monitor-heading > div { min-width: 0; flex-wrap: wrap; }
.monitor-heading h3 { font-size: 14px; font-weight: 500; overflow-wrap: anywhere; }
.monitor-heading svg { color: var(--gf-color-primary); flex-shrink: 0; }
.monitor-type { font-size: 12px; text-transform: uppercase; color: var(--gf-color-icon-muted); }
.monitor-state { display: flex; align-items: center; gap: 7px; font-size: 12px; color: var(--check-color); }
.monitor-state i { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.check-up { --check-color: var(--gf-color-success); }
.check-down { --check-color: var(--gf-color-error); }
.check-pending { --check-color: var(--gf-color-warning); }
.check-maintenance { --check-color: var(--gf-color-info); }
.check-unknown { --check-color: var(--gf-color-icon-muted); }
.uptime-metrics { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); margin: 20px 0; }
.uptime-metrics > div + div { padding-left: 20px; border-left: 1px solid var(--gf-color-line); }
.uptime-metrics > div:first-child { padding-right: 20px; }
.uptime-metrics p { font-size: 12px; color: var(--gf-color-icon-muted); }
.uptime-metrics strong { display: block; font-size: var(--status-value-size, 1.5rem); font-weight: 600; letter-spacing: -.025em; line-height: 1.3; margin: 6px 0; }
.uptime-metrics small { display: block; font-size: 12px; line-height: 1.6; color: var(--gf-color-icon-muted); }
.heartbeat-strip { display: flex; gap: 3px; }
.heartbeat-strip button { flex: 1; min-width: 0; height: 28px; padding: 0; border: 0; border-radius: 3px; background: var(--check-color); opacity: .8; cursor: pointer; }
.heartbeat-strip button:hover, .heartbeat-strip button:focus-visible { opacity: 1; outline: 2px solid var(--gf-color-base-content); outline-offset: 2px; }
.heartbeat-footer { position: relative; margin-top: 8px; }
.heartbeat-caption { display: flex; align-items: center; justify-content: space-between; gap: 4px 8px; flex-wrap: wrap; min-height: 40px; font-size: 12px; line-height: 18px; color: var(--gf-color-icon-muted); transition: opacity 140ms ease, visibility 140ms; }
.heartbeat-detail { position: absolute; inset: 0; display: flex; align-items: center; font-size: 12px; line-height: 18px; padding: 0 10px; margin: 0; background: var(--gf-color-base-200); border-radius: var(--gf-radius-field); opacity: 0; visibility: hidden; transform: translateY(-2px); transition: opacity 140ms ease, transform 140ms ease, visibility 140ms; }
.is-active .heartbeat-caption { opacity: 0; visibility: hidden; }
.is-active .heartbeat-detail { opacity: 1; visibility: visible; transform: translateY(0); }
.uptime-notice, .uptime-no-checks { font-size: 12px; line-height: 1.6; color: var(--gf-color-icon-muted); padding: 12px 0; }
@media (max-width: 639.98px) {
  .uptime-metrics > div + div { padding-left: 12px; }
  .uptime-metrics > div:first-child { padding-right: 12px; }
  .heartbeat-strip { gap: 1px; }
  .heartbeat-strip button { border-radius: 2px; }
}
@media (prefers-reduced-motion: reduce) {
  .heartbeat-caption, .heartbeat-detail { transition: none; transform: none; }
}
</style>
