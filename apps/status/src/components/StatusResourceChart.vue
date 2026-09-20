<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { StatusServer, StatusServerRange } from '@/types'

const props = defineProps<{ points: StatusServer['history']; asOf?: string; range: StatusServerRange }>()
const { t, locale } = useI18n()
const selected = ref<number | null>(null)
type Metric = 'cpu' | 'memoryPercent'
type Scale = { min: number; max: number }
const end = computed(() => Date.parse(props.asOf ?? props.points.at(-1)?.time ?? new Date().toISOString()))
const span = computed(() => ({ '1h': 1, '6h': 6, '24h': 24, '7d': 168 })[props.range] * 3_600_000)
const start = computed(() => end.value - span.value)
const x = (time: string) => Math.max(0, Math.min(720, (Date.parse(time) - start.value) / span.value * 720))
const scales = computed<Record<Metric, Scale>>(() => {
  const makeScale = (key: Metric): Scale => {
    const values = props.points.map(point => point[key]).filter(Number.isFinite)
    const minValue = values.length ? Math.min(...values) : 0
    const maxValue = values.length ? Math.max(...values) : 1
    const spanValue = Math.max(maxValue - minValue, 1)
    const padding = Math.max(spanValue * .16, .5)
    return {
      min: Math.max(0, minValue - padding),
      max: Math.min(100, maxValue + padding),
    }
  }
  return { cpu: makeScale('cpu'), memoryPercent: makeScale('memoryPercent') }
})
const y = (value: number, key: Metric) => {
  const scale = scales.value[key]
  return 160 - (value - scale.min) / Math.max(scale.max - scale.min, .001) * 150
}
function axisLabels(key: Metric) {
  const scale = scales.value[key]
  return axisValues(key).map(value => `${value.toFixed(scale.max - scale.min < 10 ? 1 : 0)}%`)
}
function axisValues(key: Metric) {
  const scale = scales.value[key]
  return [scale.max, (scale.min + scale.max) / 2, scale.min]
}
function startsSegment(index: number) {
  const previous = props.points[index - 1]
  return !previous || Date.parse(props.points[index].time) - Date.parse(previous.time) > Math.max(120_000, span.value / 119 * 2)
}
function line(key: 'cpu' | 'memoryPercent') {
  return props.points.map((point, index) => {
    return `${startsSegment(index) ? 'M' : 'L'}${x(point.time).toFixed(2)},${y(point[key], key).toFixed(2)}`
  }).join(' ')
}
function area(key: 'cpu' | 'memoryPercent') {
  let path = ''
  let segment: StatusServer['history'][number][] = []
  const closeSegment = () => {
    if (!segment.length) return
    const first = segment[0]
    const last = segment.at(-1)!
    path += `M${x(first.time).toFixed(2)},160 L${segment.map(point => `${x(point.time).toFixed(2)},${y(point[key], key).toFixed(2)}`).join(' L')} L${x(last.time).toFixed(2)},160 Z `
    segment = []
  }
  props.points.forEach((point, index) => {
    if (startsSegment(index)) closeSegment()
    segment.push(point)
  })
  closeSegment()
  return path.trim()
}
function timeLabel(time: number) {
  const date = new Date(time)
  const clock = date.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
  return span.value >= 86_400_000 ? `${date.toLocaleDateString(locale.value, { month: 'numeric', day: 'numeric' })}\n${clock}` : clock
}
const activePoint = computed(() => selected.value == null ? undefined : props.points[selected.value])
function pointLabel(point: StatusServer['history'][number]) {
  return `${timeLabel(Date.parse(point.time))} · ${t('status.cpu')} ${point.cpu.toFixed(1)}% · ${t('status.memory')} ${point.memoryPercent.toFixed(1)}%`
}
</script>

<template>
  <div class="resource-history">
    <div class="resource-axis resource-axis-cpu" :aria-label="t('status.cpu')"><span v-for="label in axisLabels('cpu')" :key="label">{{ label }}</span></div>
    <!-- deslop-ignore-next-line 24 -- data visualization SVG, not an icon -->
    <div class="resource-plot"><svg viewBox="0 0 720 170" preserveAspectRatio="none" role="group" :aria-label="`${t('status.history')} · ${t(`status.scope${range}`)} · CPU / ${t('status.memory')}`"><defs><linearGradient id="resource-memory-area" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="var(--gf-color-accent)" stop-opacity=".17" /><stop offset="100%" stop-color="var(--gf-color-accent)" stop-opacity="0" /></linearGradient><linearGradient id="resource-cpu-area" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="var(--gf-color-primary)" stop-opacity=".18" /><stop offset="100%" stop-color="var(--gf-color-primary)" stop-opacity="0" /></linearGradient></defs><path :d="area('memoryPercent')" class="resource-area memory-area" aria-hidden="true" /><path :d="area('cpu')" class="resource-area cpu-area" aria-hidden="true" /><line v-for="value in axisValues('memoryPercent')" :key="`horizontal-${value}`" x1="0" x2="720" :y1="y(value, 'memoryPercent')" :y2="y(value, 'memoryPercent')" class="grid-line grid-line-horizontal" /><line v-for="fraction in [0, .25, .5, .75, 1]" :key="`vertical-${fraction}`" :x1="fraction * 720" :x2="fraction * 720" y1="10" y2="160" class="grid-line grid-line-vertical" /><path :d="line('memoryPercent')" class="memory-line" aria-hidden="true" /><path :d="line('cpu')" class="cpu-line" aria-hidden="true" /><template v-for="(point, index) in points" :key="point.time"><circle class="resource-point-hit" :cx="x(point.time)" :cy="y(point.cpu, 'cpu')" r="9" tabindex="0" role="button" :aria-label="pointLabel(point)" @mouseenter="selected = index" @mouseleave="selected = null" @focus="selected = index" @blur="selected = null" @click="selected = index" @keydown.esc="selected = null" /></template></svg><div v-if="activePoint" class="resource-selection" aria-hidden="true"><span class="resource-guide" :style="{ left: `${x(activePoint.time) / 720 * 100}%` }"></span><span class="resource-selection-point resource-selection-cpu" :style="{ left: `${x(activePoint.time) / 720 * 100}%`, top: `${y(activePoint.cpu, 'cpu') / 170 * 100}%` }"></span><span class="resource-selection-point resource-selection-memory" :style="{ left: `${x(activePoint.time) / 720 * 100}%`, top: `${y(activePoint.memoryPercent, 'memoryPercent') / 170 * 100}%` }"></span></div><div v-if="activePoint" class="resource-tooltip" role="status"><b>{{ timeLabel(Date.parse(activePoint.time)) }}</b><span>{{ t('status.cpu') }} <strong>{{ activePoint.cpu.toFixed(1) }}%</strong></span><span>{{ t('status.memory') }} <strong>{{ activePoint.memoryPercent.toFixed(1) }}%</strong></span></div><div class="resource-ticks" aria-hidden="true"><span>{{ timeLabel(start) }}</span><span>{{ timeLabel(start + span / 2) }}</span><span>{{ timeLabel(end) }}</span></div></div>
    <div class="resource-axis resource-axis-memory" :aria-label="t('status.memory')"><span v-for="label in axisLabels('memoryPercent')" :key="label">{{ label }}</span></div>
  </div>
</template>

<style scoped>
.resource-ticks > span { white-space: pre-line; text-align: center; }
.resource-ticks > span:first-child { text-align: start; }
.resource-ticks > span:last-child { text-align: end; }
.resource-history { display: grid; grid-template-columns: 32px minmax(0, 1fr) 38px; gap: 12px; min-width: 0; }
.resource-axis { display: flex; flex-direction: column; justify-content: space-between; flex-shrink: 0; height: calc(var(--status-chart-height, 156px) - 7px); padding-top: 4px; color: var(--gf-color-primary); font-size: 12px; text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
.resource-axis-memory { color: var(--gf-color-accent); text-align: left; }
.resource-plot { position: relative; flex: 1; min-width: 0; }
.resource-plot svg { width: 100%; height: var(--status-chart-height, 156px); overflow: visible; }
.resource-area { pointer-events: none; }
.memory-area { fill: url(#resource-memory-area); }
.cpu-area { fill: url(#resource-cpu-area); }
.grid-line { fill: none; stroke: color-mix(in oklch, var(--gf-color-line) 82%, transparent); stroke-dasharray: 2 6; stroke-width: .7; vector-effect: non-scaling-stroke; }
.grid-line-vertical { opacity: .76; }
.grid-line-horizontal { opacity: .9; }
.cpu-line, .memory-line { fill: none; stroke-width: 2; vector-effect: non-scaling-stroke; stroke-linejoin: round; stroke-linecap: round; }
.cpu-line { stroke: var(--gf-color-primary); }
.memory-line { stroke: var(--gf-color-accent); stroke-dasharray: 5 4; }
.resource-point-hit { fill: transparent; cursor: crosshair; pointer-events: all; }
.resource-selection { position: absolute; inset: 0 0 auto; height: var(--status-chart-height, 156px); pointer-events: none; }
.resource-guide { position: absolute; top: 5.9%; bottom: 5.9%; width: 1px; background: color-mix(in oklch, var(--gf-color-primary) 34%, transparent); transform: translateX(-.5px); }
.resource-selection-point { position: absolute; width: 8px; height: 8px; border: 2px solid var(--gf-color-base-100); border-radius: 50%; transform: translate(-50%, -50%); }
.resource-selection-cpu { background: var(--gf-color-primary); box-shadow: 0 0 0 1px var(--gf-color-primary); }
.resource-selection-memory { background: var(--gf-color-accent); box-shadow: 0 0 0 1px var(--gf-color-accent); }
.resource-tooltip { position: absolute; z-index: 2; top: 8px; inset-inline-end: 8px; pointer-events: none; display: grid; gap: 6px; width: min(176px, calc(100% - 16px)); padding: 10px 12px; border-radius: var(--gf-radius-field); background: var(--gf-color-base-100); color: var(--gf-color-base-content); box-shadow: 0 8px 24px color-mix(in oklch, var(--gf-color-base-content) 9%, transparent), inset 0 0 0 1px var(--gf-color-line); font-size: 12px; }
.resource-tooltip b { font-weight: 600; white-space: pre-line; }
.resource-tooltip span { display: flex; justify-content: space-between; gap: 16px; color: var(--gf-color-icon-muted); }
.resource-tooltip strong { color: var(--gf-color-base-content); font-weight: 600; }
.resource-ticks { display: flex; justify-content: space-between; gap: 8px; margin-top: 8px; color: var(--gf-color-icon-muted); font-size: 12px; font-variant-numeric: tabular-nums; }
@media (max-width: 640px) { .resource-history { grid-template-columns: 28px minmax(0, 1fr) 30px; gap: 7px; }.resource-axis { font-size: 11px; } }
</style>
