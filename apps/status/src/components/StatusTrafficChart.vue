<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { StatusRange, StatusTraffic } from '@/types'

const props = defineProps<{ traffic: StatusTraffic; range: StatusRange }>()
const { t, locale } = useI18n()
const selected = ref<number | null>(null)
const buckets = computed(() => {
  const step = props.range === '24h' ? 3_600_000 : 86_400_000
  const first = Math.floor(Date.parse(props.traffic.startAt) / step) * step
  const end = Date.parse(props.traffic.endAt)
  const values = new Map(props.traffic.series.map(point => [Date.parse(point.time), point]))
  const result = []
  // The successful aggregate query is sparse: omitted buckets have no events.
  for (let time = first; time <= end && result.length < 32; time += step) {
    result.push(values.get(time) ?? { time: new Date(time).toISOString(), pageviews: 0, visitors: 0 })
  }
  return result
})
const maximum = computed(() => Math.max(1, ...buckets.value.flatMap(point => [point.pageviews, point.visitors])))
const activePoint = computed(() => selected.value == null ? undefined : buckets.value[selected.value])
function label(time: string, full = false) {
  return new Date(time).toLocaleString(locale.value, props.range === '24h' && !full ? { hour: '2-digit', minute: '2-digit' } : { month: 'short', day: 'numeric', ...(full && props.range === '24h' ? { hour: '2-digit', minute: '2-digit' } as const : {}) })
}
</script>

<template>
  <div class="traffic-chart">
    <div class="chart-y-axis" aria-hidden="true"><span>{{ maximum.toLocaleString(locale) }}</span><span>{{ Math.round(maximum / 2).toLocaleString(locale) }}</span><span>0</span></div>
    <div class="chart-plot" role="group" :aria-label="t('status.trend')">
      <div class="chart-bars">
        <button v-for="(point, index) in buckets" :key="point.time" type="button" class="chart-bucket" :aria-label="t('status.chartLabel', { time: label(point.time, true), views: point.pageviews, visitors: point.visitors })" @mouseenter="selected = index" @mouseleave="selected = null" @focus="selected = index" @blur="selected = null" @click="selected = index" @keydown.esc="selected = null">
          <span class="bar bar-views" :style="{ height: `${point.pageviews / maximum * 100}%` }"></span>
          <span class="bar bar-visitors" :style="{ height: `${point.visitors / maximum * 100}%` }"></span>
        </button>
      </div>
      <div v-if="activePoint" class="chart-tooltip" role="status"><b>{{ label(activePoint.time, true) }}</b><span>{{ t('status.views') }} <strong>{{ activePoint.pageviews.toLocaleString(locale) }}</strong></span><span>{{ t('status.visitors') }} <strong>{{ activePoint.visitors.toLocaleString(locale) }}</strong></span></div>
      <div class="chart-x-axis" aria-hidden="true"><span>{{ label(traffic.startAt) }}</span><span>{{ label(new Date((Date.parse(traffic.startAt) + Date.parse(traffic.endAt)) / 2).toISOString()) }}</span><span>{{ label(traffic.endAt) }}</span></div>
    </div>
  </div>
</template>

<style scoped>
.traffic-chart { display: flex; gap: 12px; min-width: 0; padding-top: 6px; }
.chart-y-axis { display: flex; flex-direction: column; justify-content: space-between; flex-shrink: 0; width: 32px; height: calc(var(--status-chart-height, 156px) + 6px); transform: translateY(-6px); color: var(--gf-color-icon-muted); font-size: 12px; text-align: right; font-variant-numeric: tabular-nums; }
.chart-plot { position: relative; min-width: 0; flex: 1; }
/* deslop-ignore-next-line 06 -- repeating gradient draws chart gridlines, not atmosphere */
.chart-bars { display: flex; align-items: stretch; gap: clamp(2px, .55vw, 7px); height: var(--status-chart-height, 156px); border-bottom: 1px solid var(--gf-color-line); background: repeating-linear-gradient(to top, transparent 0 calc(50% - .5px), color-mix(in oklch, var(--gf-color-line) 72%, transparent) calc(50% - .5px) calc(50% + .5px), transparent calc(50% + .5px) 100%); }
.chart-bucket { position: relative; display: flex; align-items: flex-end; justify-content: center; flex: 1; gap: 2px; min-width: 0; padding-top: 2px; border-radius: 4px 4px 0 0; cursor: crosshair; transition: background-color 120ms cubic-bezier(0.2, 0, 0, 1); }
.chart-bucket:focus-visible { background: var(--gf-color-base-300); outline: 2px solid var(--gf-color-primary); outline-offset: 2px; }
.bar { display: block; width: 42%; max-width: 16px; min-height: 1px; border-radius: 3px 3px 0 0; transition: opacity 120ms cubic-bezier(0.2, 0, 0, 1); }
.bar-views { background: var(--gf-color-primary); }
.bar-visitors { background: var(--gf-color-accent); }
.chart-x-axis { display: flex; justify-content: space-between; gap: 8px; margin-top: 10px; color: var(--gf-color-icon-muted); font-size: 12px; font-variant-numeric: tabular-nums; }
.chart-tooltip { position: absolute; z-index: 2; top: 0; inset-inline-end: 0; pointer-events: none; display: grid; gap: 6px; min-width: 164px; padding: 10px 12px; border-radius: var(--gf-radius-field); background: var(--gf-color-base-100); color: var(--gf-color-base-content); box-shadow: 0 8px 24px color-mix(in oklch, var(--gf-color-base-content) 9%, transparent), inset 0 0 0 1px var(--gf-color-line); font-size: 12px; }
.chart-tooltip b { font-weight: 600; }
.chart-tooltip span { display: flex; justify-content: space-between; gap: 16px; color: var(--gf-color-icon-muted); }
.chart-tooltip strong { color: var(--gf-color-base-content); font-weight: 600; }
@media (hover: hover) {
  .chart-bucket:hover { background: var(--gf-color-base-300); }
  .chart-bucket:hover .bar { opacity: .82; }
}
@media (max-width: 640px) {
  .traffic-chart { gap: 7px; }
  .chart-y-axis { width: 25px; }
  .chart-bucket { gap: 1px; }
  .bar { border-radius: 2px 2px 0 0; }
}
@media (prefers-reduced-motion: reduce) { .chart-bucket, .bar { transition: none; } }
</style>
