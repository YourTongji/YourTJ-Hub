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
.traffic-chart { display: flex; gap: 12px; padding-top: 8px; min-width: 0; }
.chart-y-axis { display: flex; flex-direction: column; justify-content: space-between; flex-shrink: 0; width: 30px; text-align: right; height: calc(var(--status-chart-height, 144px) + 6px); transform: translateY(-6px); font-size: 12px; color: var(--gf-color-icon-muted); }
.chart-plot { min-width: 0; flex: 1; position: relative; }
.chart-bars { height: var(--status-chart-height, 144px); display: flex; align-items: stretch; gap: clamp(2px, .6vw, 8px); border-bottom: 1px solid var(--gf-color-line); background: repeating-linear-gradient(to top, transparent 0 71px, color-mix(in oklch, var(--gf-color-line) 65%, transparent) 71px 72px); }
.chart-bucket { position: relative; flex: 1; display: flex; align-items: flex-end; justify-content: center; gap: 2px; min-width: 0; border-radius: 4px 4px 0 0; cursor: crosshair; padding-top: 2px; }
.chart-bucket:hover, .chart-bucket:focus { background: var(--gf-color-base-200); outline: 1px solid var(--gf-color-line); outline-offset: 1px; }
.bar { display: block; width: 42%; max-width: 16px; border-radius: 3px 3px 0 0; }
.bar-views { background: color-mix(in oklch, var(--gf-color-primary) 78%, var(--gf-color-base-100)); }.bar-visitors { background: var(--gf-color-accent); }
.chart-bucket:hover .bar-views, .chart-bucket:focus .bar-views { background: var(--gf-color-primary); }
.chart-x-axis { display: flex; justify-content: space-between; font-size: 12px; color: var(--gf-color-icon-muted); margin-top: 12px; gap: 8px; }
.chart-tooltip { position: absolute; top: 0; right: 0; pointer-events: none; display: grid; gap: 6px; min-width: 155px; padding: 10px 12px; border: 1px solid var(--gf-color-line); background: var(--gf-color-base-100); color: var(--gf-color-base-content); box-shadow: 0 4px 20px color-mix(in oklch, var(--gf-color-base-content) 8%, transparent); border-radius: 8px; font-size: 12px; z-index: 1; }.chart-tooltip b { font-weight: 600; }.chart-tooltip span { display: flex; justify-content: space-between; gap: 16px; color: var(--gf-color-icon-muted); }.chart-tooltip strong { color: var(--gf-color-base-content); }
@media (max-width: 640px) { .traffic-chart { gap: 7px; }.chart-y-axis { font-size: 12px; width: 24px; }.chart-bucket { gap: 1px; }.bar { border-radius: 2px 2px 0 0; }.chart-x-axis { font-size: 12px; } }
</style>
