<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRight, MousePointer2 } from '@lucide/vue'
import type { StatusDevices } from '@/types'
import { deviceGraph, linkPath, type DeviceNode, type DeviceLink, type Dimension } from '@/runtime/device-sankey'

const props = defineProps<{ data: StatusDevices }>()
const { t, locale } = useI18n()
const root = ref<HTMLElement>()
const width = ref(900)
const stage = ref(0)
const selection = ref<{ kind: 'node' | 'link'; id: string } | null>(null)
const gradientId = useId().replaceAll(':', '-')
let observer: ResizeObserver | undefined
let dismiss: ReturnType<typeof setTimeout> | undefined
const compact = computed(() => width.value < 600)
const dimensions = computed<Dimension[]>(() => compact.value ? stage.value === 0 ? ['device', 'os'] : ['os', 'browser'] : ['device', 'os', 'browser'])
const graph = computed(() => deviceGraph(props.data, width.value, dimensions.value))
const format = (n: number) => new Intl.NumberFormat(locale.value).format(n)
const percent = (n: number) => new Intl.NumberFormat(locale.value, { style: 'percent', ...(n / props.data.visitors < .001 ? { maximumSignificantDigits: 1 } : { maximumFractionDigits: 1 }) }).format(n / props.data.visitors)
const label = (node: DeviceNode) => t(`devices.labels.${node.category}`)
const source = (link: DeviceLink) => link.source as DeviceNode
const target = (link: DeviceLink) => link.target as DeviceNode
const palette: Record<string, string> = { 'yourtj-app': '#df795e', mobile: '#27a99a', laptop: '#5987db', desktop: '#c79953', tablet: '#a18bce', windows: '#5987db', android: '#27a99a', ios: '#8d8dcb', macos: '#b38a67', linux: '#bf9c43', chromeos: '#73a483', chrome: '#5987db', edge: '#27a99a', safari: '#8d8dcb', webview: '#b38a67', firefox: '#ca8659', other: '#929cae', unknown: '#a6abb5' }
const color = (node: DeviceNode) => palette[node.category] ?? '#929cae'
const selected = computed(() => {
  if (selection.value?.kind === 'node') {
    const node = graph.value.nodes.find(n => n.id === selection.value?.id)
    return node ? { label: label(node), value: node.value ?? 0 } : null
  }
  const link = graph.value.links.find(l => l.id === selection.value?.id)
  return link ? { label: `${label(source(link))} → ${label(target(link))}`, value: link.value } : null
})
function choose(kind: 'node' | 'link', id: string) { clearTimeout(dismiss); selection.value = { kind, id } }
function hold() { clearTimeout(dismiss) }
function leave() { dismiss = setTimeout(() => { selection.value = null }, 140) }
function clear() { clearTimeout(dismiss); selection.value = null }
function related(link: DeviceLink) {
  return !selection.value || (selection.value.kind === 'link' ? link.id === selection.value.id : source(link).id === selection.value.id || target(link).id === selection.value.id)
}
function nodeRelated(node: DeviceNode) { return !selection.value || node.id === selection.value.id || graph.value.links.some(l => related(l) && (source(l).id === node.id || target(l).id === node.id)) }
function labelX(node: DeviceNode) {
  if (compact.value) return node.depth === 0 ? node.x1! + 10 : node.x0! - 10
  return node.depth === 0 ? node.x0! - 12 : node.x1! + 12
}
function anchor(node: DeviceNode) { return (compact.value ? node.depth !== 0 : node.depth === 0) ? 'end' : 'start' }
function aria(name: string, value: number) { return t('devices.detail', { name, count: format(value), percent: percent(value) }) }
watch([() => props.data, dimensions], clear)
onMounted(() => {
  const measure = () => { if (root.value?.clientWidth) width.value = root.value.clientWidth }
  measure()
  observer = new ResizeObserver(measure)
  if (root.value) observer.observe(root.value)
})
onBeforeUnmount(() => { observer?.disconnect(); clearTimeout(dismiss) })
</script>

<template>
  <div ref="root" class="device-chart" @keydown.esc="clear">
    <div v-if="compact" class="device-stages" role="group" :aria-label="t('devices.stageLabel')">
      <button type="button" :aria-pressed="stage === 0" @click="stage = 0">{{ t('devices.device') }} <ArrowRight :size="12" /> {{ t('devices.os') }}</button>
      <button type="button" :aria-pressed="stage === 1" @click="stage = 1">{{ t('devices.os') }} <ArrowRight :size="12" /> {{ t('devices.browser') }}</button>
    </div>
    <div class="device-columns" :class="{ compact }"><span v-for="dimension in dimensions" :key="dimension">{{ t(`devices.${dimension}`) }}</span></div>
    <svg class="device-sankey" :viewBox="`0 0 ${width} 320`" role="group" :aria-label="t('devices.chartLabel')" @mouseleave="leave" @mouseenter="hold">
      <defs><linearGradient v-for="(link, index) in graph.links" :id="`${gradientId}-${index}`" :key="link.id" gradientUnits="userSpaceOnUse" :x1="source(link).x1" :x2="target(link).x0"><stop offset="0%" :stop-color="color(source(link))" /><stop offset="100%" :stop-color="color(target(link))" /></linearGradient></defs>
      <g v-for="(link, index) in graph.links" :key="link.id" class="device-flow" :class="{ dimmed: !related(link), selected: !!selection && related(link) }" tabindex="0" role="img" :aria-label="aria(`${label(source(link))} → ${label(target(link))}`, link.value)" @mouseenter="choose('link', link.id)" @focus="choose('link', link.id)" @blur="leave" @click="choose('link', link.id)">
        <path :d="linkPath(link) ?? ''" :stroke="`url(#${gradientId}-${index})`" :stroke-width="link.width" class="flow-ribbon" />
        <path :d="linkPath(link) ?? ''" stroke="transparent" :stroke-width="Math.max(link.width ?? 0, 10)" class="flow-hit" />
      </g>
      <g v-for="node in graph.nodes" :key="node.id" class="device-node" :class="{ dimmed: !nodeRelated(node) }" tabindex="0" role="img" :aria-label="aria(label(node), node.value ?? 0)" @mouseenter="choose('node', node.id)" @focus="choose('node', node.id)" @blur="leave" @click="choose('node', node.id)">
        <rect :x="node.x0" :y="node.y0" :width="node.x1! - node.x0!" :height="node.y1! - node.y0!" rx="2" :fill="color(node)" />
        <text :x="labelX(node)" :y="(node.y0! + node.y1!) / 2 - 2" :text-anchor="anchor(node)">{{ label(node) }}<tspan :x="labelX(node)" dy="16" class="node-count">{{ format(node.value ?? 0) }} · {{ percent(node.value ?? 0) }}</tspan></text>
      </g>
    </svg>
    <div class="device-detail" :class="{ active: selected }" aria-live="polite">
      <span v-if="selected"><b>{{ selected.label }}</b><span>{{ format(selected.value) }} {{ t('devices.people') }}</span><span>{{ percent(selected.value) }}</span></span>
      <span v-else class="device-hint"><MousePointer2 :size="13" /><span>{{ t(compact ? 'devices.tapHint' : 'devices.hoverHint') }}</span></span>
    </div>
  </div>
</template>

<style scoped>
.device-chart { min-width: 0; }
.device-columns { display: grid; grid-template-columns: repeat(3, 1fr); margin: 4px 0 14px; color: var(--gf-color-icon-muted); font-size: 12px; }
.device-columns span:nth-child(2) { text-align: center; }
.device-columns span:last-child { text-align: right; }
.device-columns.compact { grid-template-columns: 1fr 1fr; }
.device-sankey { width: 100%; height: 320px; overflow: visible; }
.device-flow, .device-node { cursor: pointer; transition: opacity 180ms ease; }
.device-flow path { fill: none; }
.flow-ribbon { opacity: .24; transition: opacity 180ms ease; pointer-events: none; }
.device-flow.selected .flow-ribbon, .device-flow:focus-visible .flow-ribbon { opacity: .65; }
.device-flow.dimmed { opacity: .18; }
.device-node.dimmed { opacity: .32; }
.device-node text { font-size: 12px; font-weight: 600; fill: var(--gf-color-base-content); paint-order: stroke; stroke: var(--gf-color-base-100); stroke-width: 4px; stroke-linejoin: round; }
.device-node .node-count { fill: var(--gf-color-icon-muted); font-size: 11px; font-weight: 400; }
.device-node:focus-visible { outline: none; }
.device-node:focus-visible rect { stroke: var(--gf-color-base-content); stroke-width: 2; }
.device-detail { display: flex; align-items: center; justify-content: center; height: 56px; margin-top: 10px; padding: 8px 12px; border-radius: 8px; background: var(--gf-color-base-200); color: var(--gf-color-icon-muted); font-size: 12px; line-height: 18px; transition: background-color 160ms ease, color 160ms ease; }
.device-detail.active { background: color-mix(in oklch, var(--gf-color-primary) 7%, var(--gf-color-base-100)); color: var(--gf-color-base-content); }
.device-detail > span { display: flex; flex-wrap: wrap; align-items: center; justify-content: center; gap: 4px 14px; }
.device-detail b { font-weight: 600; }
.device-detail .device-hint { flex-wrap: nowrap; gap: 6px; text-align: center; }
.device-hint svg { flex-shrink: 0; }
.device-stages { display: flex; gap: 4px; margin-bottom: 20px; padding: 3px; border: 1px solid var(--gf-color-line); border-radius: 9px; background: var(--gf-color-base-200); }
.device-stages button { flex: 1; display: flex; align-items: center; justify-content: center; gap: 5px; min-height: 34px; padding: 4px; font-size: 11px; border-radius: 6px; color: var(--gf-color-icon-muted); }
.device-stages button[aria-pressed=true] { color: var(--gf-color-base-content); background: var(--gf-color-base-100); box-shadow: 0 1px 3px #0000000a; }
@media (prefers-reduced-motion: reduce) { .device-flow, .device-node, .flow-ribbon, .device-detail { transition: none; } }
</style>
