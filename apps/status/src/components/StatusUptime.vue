<script setup lang="ts">
import { computed, onBeforeUnmount, onDeactivated, ref } from 'vue'
import { ArrowUpRight } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { StatusSnapshot, StatusUptimeMonitor } from '@/types'
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
const fresh = computed(() => !props.failed && props.source?.state === 'ok' && isRecentStatusTime(props.source.fetchedAt, props.now, 150_000))
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
        <div class="monitor-heading"><div><span class="monitor-source-logo" :class="`state-${state(monitor)}`" aria-hidden="true"><img src="/source-logos/uptime-kuma.svg" alt="Uptime Kuma logo" /></span><h3>{{ monitor.name }}</h3><span class="monitor-type">{{ monitor.type }}</span></div><span class="monitor-state" :class="`check-${state(monitor)}`">{{ t(`status.check${state(monitor)}`) }}</span></div>
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
    <div class="status-panel-source status-source uptime-source"><span>{{ t('status.dataSource') }} <b>Uptime Kuma</b><span class="source-badge" :class="{ connected: sourceState === 'ok' }">{{ t(`status.uptimeSource${sourceState}`) }}</span></span><time>{{ t('status.updated', { time: formatDate(source?.fetchedAt) }) }}</time></div>
  </section>
</template>

<style scoped>
.uptime-external {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 5px 9px;
  border-radius: var(--gf-radius-field);
  border: 1px solid var(--gf-color-line);
  background: color-mix(in oklch, var(--gf-color-base-200) 76%, transparent);
  box-shadow: inset 0 1px 0 color-mix(in oklch, var(--gf-color-base-content) 6%, transparent);
  color: var(--gf-color-base-content);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  transition-property: background-color, color, transform;
  transition-duration: 120ms;
  transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
}
.uptime-external:active { transform: scale(.96); }
@media (hover: hover) { .uptime-external:hover { background: var(--gf-color-base-300); } }
.uptime-external:focus-visible { outline: 2px solid var(--gf-color-primary); outline-offset: 3px; }
.uptime-monitors { display: grid; gap: 12px; }
.uptime-monitor { min-width: 0; padding: 18px; border: 1px solid var(--gf-color-line); border-radius: .75rem; background: color-mix(in oklch, var(--gf-color-base-200) 48%, transparent); box-shadow: inset 0 1px 0 color-mix(in oklch, var(--gf-color-base-content) 5%, transparent); transition-property: transform, border-color, background-color, box-shadow; transition-duration: 180ms; transition-timing-function: cubic-bezier(.2, 0, 0, 1); }
@media (hover: hover) { .uptime-monitor:hover { transform: translateY(-2px); border-color: color-mix(in oklch, var(--gf-color-primary) 25%, var(--gf-color-line)); background: color-mix(in oklch, var(--gf-color-primary) 4%, var(--gf-color-base-200)); box-shadow: inset 0 1px 0 color-mix(in oklch, var(--gf-color-base-content) 6%, transparent), 0 10px 24px color-mix(in oklch, var(--gf-color-base-content) 5%, transparent); } }
.monitor-heading, .monitor-heading > div { display: flex; align-items: center; gap: 8px; }
.monitor-heading { justify-content: space-between; flex-wrap: wrap; gap: 10px 16px; }
.monitor-heading > div { min-width: 0; flex-wrap: wrap; }
.monitor-heading h3 { font-size: 15px; line-height: 1.35; font-weight: 700; letter-spacing: -.02em; overflow-wrap: anywhere; }
.monitor-source-logo { display: grid; place-items: center; width: 32px; height: 32px; flex: 0 0 32px; border: 1.5px solid var(--gf-color-icon-muted); border-radius: 50%; background: #fff; overflow: hidden; }
.monitor-source-logo img { width: 19px; height: 19px; object-fit: contain; }
.monitor-source-logo.state-up { border-color: var(--gf-color-success); }
.monitor-source-logo.state-down { border-color: var(--gf-color-error); }
.monitor-type { color: var(--gf-color-icon-muted); font-size: 11px; line-height: 18px; font-weight: 500; }
.monitor-state { display: inline-flex; align-items: center; min-height: 28px; padding: 4px 0; color: var(--check-color); font-size: 12px; font-weight: 600; white-space: nowrap; }
.check-up { --check-color: var(--gf-color-success); }
.check-down { --check-color: var(--gf-color-error); }
.check-pending { --check-color: var(--gf-color-warning); }
.check-maintenance { --check-color: var(--gf-color-info); }
.check-unknown { --check-color: var(--gf-color-icon-muted); }
.uptime-metrics { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; margin: 19px 0; }
.uptime-metrics > div { min-width: 0; }
.uptime-metrics p { font-size: 12px; color: var(--gf-color-icon-muted); }
.uptime-metrics strong { display: block; margin: 6px 0 4px; font-size: clamp(1.5rem, 1.2rem + .7vw, 1.9rem); font-weight: 700; letter-spacing: -.055em; line-height: 1.15; font-variant-numeric: tabular-nums; }
.uptime-metrics small { display: block; font-size: 12px; line-height: 1.5; color: var(--gf-color-icon-muted); text-wrap: pretty; }
.heartbeat-history { min-width: 0; }
.heartbeat-strip { display: flex; gap: 3px; min-height: 28px; }
.heartbeat-strip button { flex: 1; min-width: 0; height: 28px; padding: 0; border: 0; border-radius: 5px; background: var(--check-color); opacity: .72; cursor: pointer; transition-property: opacity, transform, filter; transition-duration: 120ms; transition-timing-function: cubic-bezier(0.2, 0, 0, 1); }
.heartbeat-strip button:focus-visible { opacity: 1; outline: 2px solid var(--gf-color-base-content); outline-offset: 2px; }
.heartbeat-strip button:active { transform: scale(.96); }
.heartbeat-footer { position: relative; margin-top: 8px; }
.heartbeat-caption { display: flex; align-items: center; justify-content: space-between; gap: 4px 8px; flex-wrap: wrap; min-height: 36px; font-size: 12px; line-height: 18px; color: var(--gf-color-icon-muted); transition: opacity 120ms cubic-bezier(0.2, 0, 0, 1), visibility 120ms cubic-bezier(0.2, 0, 0, 1); }
.heartbeat-detail { position: absolute; inset: 0; display: flex; align-items: center; min-height: 36px; padding: 0 10px; margin: 0; border: 1px solid var(--gf-color-line); border-radius: var(--gf-radius-field); background: color-mix(in oklch, var(--gf-color-base-100) 84%, transparent); color: var(--gf-color-base-content); font-size: 12px; line-height: 18px; opacity: 0; visibility: hidden; transform: translateY(-2px); transition: opacity 120ms cubic-bezier(0.2, 0, 0, 1), transform 120ms cubic-bezier(0.2, 0, 0, 1), visibility 120ms cubic-bezier(0.2, 0, 0, 1); }
.is-active .heartbeat-caption { opacity: 0; visibility: hidden; }
.is-active .heartbeat-detail { opacity: 1; visibility: visible; transform: translateY(0); }
.uptime-notice, .uptime-no-checks { padding: 10px 0; color: var(--gf-color-icon-muted); font-size: 12px; line-height: 1.55; }
@media (hover: hover) { .heartbeat-strip button:hover { opacity: 1; } }
@media (max-width: 639.98px) {
  .uptime-heading { align-items: flex-start; }
  .uptime-external { width: auto; }
  .uptime-monitor { padding: 15px 11px; }
  .uptime-metrics { gap: 16px; }
  .heartbeat-strip { gap: 2px; }
  .heartbeat-strip button { border-radius: 4px; }
}
@media (max-width: 359.98px) { .uptime-metrics { grid-template-columns: 1fr; } }
@media (prefers-reduced-motion: reduce) {
  .uptime-external, .uptime-monitor, .heartbeat-strip button, .heartbeat-caption, .heartbeat-detail { transition: none; transform: none; }
}
</style>
