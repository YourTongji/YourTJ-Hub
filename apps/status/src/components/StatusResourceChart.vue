<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { StatusServer, StatusServerRange } from '@/types'

const props = defineProps<{ points: StatusServer['history']; asOf?: string; range: StatusServerRange }>()
const { t, locale } = useI18n()
const end = computed(() => Date.parse(props.asOf ?? props.points.at(-1)?.time ?? new Date().toISOString()))
const span = computed(() => ({ '1h': 1, '6h': 6, '24h': 24, '7d': 168 })[props.range] * 3_600_000)
const start = computed(() => end.value - span.value)
const x = (time: string) => Math.max(0, Math.min(720, (Date.parse(time) - start.value) / span.value * 720))
const y = (value: number) => 160 - value / 100 * 150
function line(key: 'cpu' | 'memoryPercent') {
  return props.points.map((point, index) => {
    const previous = props.points[index - 1]
    // The provider downsamples long ranges to 120 points. Break gaps larger
    // than two expected intervals, with a two-minute floor for the hourly view.
    const move = !previous || Date.parse(point.time) - Date.parse(previous.time) > Math.max(120_000, span.value / 119 * 2)
    return `${move ? 'M' : 'L'}${x(point.time).toFixed(2)},${y(point[key]).toFixed(2)}`
  }).join(' ')
}
function timeLabel(time: number) {
  const date = new Date(time)
  const clock = date.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
  return span.value >= 86_400_000 ? `${date.toLocaleDateString(locale.value, { month: 'numeric', day: 'numeric' })}\n${clock}` : clock
}
</script>

<template>
  <div class="resource-history">
    <div class="resource-axis" aria-hidden="true"><span>100%</span><span>50%</span><span>0%</span></div>
    <!-- deslop-ignore-next-line 24 -- data visualization SVG, not an icon -->
    <div class="resource-plot"><svg viewBox="0 0 720 170" preserveAspectRatio="none" role="img" :aria-label="`${t('status.history')} · ${t(`status.scope${range}`)} · CPU / ${t('status.memory')}`"><line v-for="value in [0, 50, 100]" :key="value" x1="0" x2="720" :y1="y(value)" :y2="y(value)" class="grid-line" /><path :d="line('memoryPercent')" class="memory-line" /><path :d="line('cpu')" class="cpu-line" /><circle v-for="point in points" :key="point.time" :cx="x(point.time)" :cy="y(point.cpu)" r="1.8" class="cpu-point"><title>{{ timeLabel(Date.parse(point.time)) }} · CPU {{ point.cpu.toFixed(1) }}% · {{ t('status.memory') }} {{ point.memoryPercent.toFixed(1) }}%</title></circle></svg><div class="resource-ticks" aria-hidden="true"><span>{{ timeLabel(start) }}</span><span>{{ timeLabel(start + span / 2) }}</span><span>{{ timeLabel(end) }}</span></div></div>
  </div>
</template>

<style scoped>
.resource-ticks > span { white-space: pre-line; text-align: center; }
.resource-ticks > span:first-child { text-align: start; }
.resource-ticks > span:last-child { text-align: end; }
.resource-history { display: flex; gap: 12px; min-width: 0; }
.resource-axis { display: flex; flex-direction: column; justify-content: space-between; flex-shrink: 0; width: 32px; height: calc(var(--status-chart-height, 156px) - 7px); padding-top: 4px; color: var(--gf-color-icon-muted); font-size: 12px; text-align: right; font-variant-numeric: tabular-nums; }
.resource-plot { flex: 1; min-width: 0; }
.resource-plot svg { width: 100%; height: var(--status-chart-height, 156px); overflow: visible; }
.grid-line { stroke: color-mix(in oklch, var(--gf-color-line) 82%, transparent); stroke-dasharray: 3 5; stroke-width: .7; }
.cpu-line, .memory-line { fill: none; stroke-width: 2; vector-effect: non-scaling-stroke; stroke-linejoin: round; stroke-linecap: round; }
.cpu-line { stroke: var(--gf-color-primary); }
.memory-line { stroke: var(--gf-color-accent); stroke-dasharray: 5 4; }
.cpu-point { fill: var(--gf-color-primary); }
.resource-ticks { display: flex; justify-content: space-between; gap: 8px; margin-top: 8px; color: var(--gf-color-icon-muted); font-size: 12px; font-variant-numeric: tabular-nums; }
@media (max-width: 640px) { .resource-history { gap: 7px; }.resource-axis { width: 25px; } }
</style>
