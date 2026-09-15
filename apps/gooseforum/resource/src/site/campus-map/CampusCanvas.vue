<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import maplibregl, {
  type GeoJSONSource,
  type MapLayerMouseEvent,
} from 'maplibre-gl'
import workerUrl from 'maplibre-gl/dist/maplibre-gl-csp-worker.js?url'
import 'maplibre-gl/dist/maplibre-gl.css'
import { categoryColors, makeMapStyle, selectableLayers } from './style'
import {
  featureBounds,
  type CampusData,
  type CampusPlace,
  type Category,
} from './catalog'

import type { CampusConfig } from './campuses'
import { locationData, type MapLocation } from './location'
import { layoutCampusLabels, type CampusLabel } from './labels'

const props = defineProps<{
  campus: CampusConfig
  panelOpen: boolean
  location: MapLocation | null
  data: CampusData
  places: CampusPlace[]
  selected: CampusPlace | null
  category: Category
  sport: string
  dimensional: boolean
  reducedMotion: boolean
}>()
const emit = defineEmits<{
  select: [id: string]
  ready: []
  failure: []
  bearing: [value: number]
}>()
const host = ref<HTMLDivElement>()
const mapLabels = ref<CampusLabel[]>([])
let map: maplibregl.Map | undefined
let resize: ResizeObserver | undefined
let frame = 0
let alive = true
const featured = [
  '同济大学图书馆',
  '三好坞餐厅',
  '同济大学游泳馆',
  '129体育馆',
  '经纬楼',
  '教学南楼',
  '教学北楼',
  '西南一楼',
  '大学生活动中心',
]
const candidates = computed(() =>
  props.places.filter(
    (p) =>
      p.id === props.selected?.id ||
      ((props.category === 'all'
        ? p.named || p.category === 'sport'
        : p.category === props.category) &&
        (!props.sport || p.sports.includes(props.sport))),
  ),
)

function updateLabels() {
  frame = 0
  if (!map || !host.value) return
  mapLabels.value = layoutCampusLabels(candidates.value, {
    width: host.value.clientWidth,
    height: host.value.clientHeight,
    zoom: map.getZoom(),
    selectedId: props.selected?.id,
    featured: [...featured, ...props.campus.suggestions],
    project: (center) => map!.project(center),
  })
  emit('bearing', map.getBearing())
}
function queueLabels() {
  if (!frame) frame = requestAnimationFrame(updateLabels)
}
function selectedChanged() {
  if (!map?.isStyleLoaded()) return
  const selected = props.selected
  ;(map.getSource('selection') as GeoJSONSource).setData({
    type: 'FeatureCollection',
    features: selected ? [selected.feature] : [],
  })
  if (selected)
    map.fitBounds(featureBounds(selected.feature), {
      maxZoom: 17.4,
      padding: {
        top: window.innerWidth < 700 && props.panelOpen ? 210 : 90,
        right: 100,
        bottom: window.innerWidth < 700 && props.panelOpen ? 300 : 70,
        left: window.innerWidth < 700 || !props.panelOpen ? 60 : 360,
      },
      duration: props.reducedMotion ? 0 : 650,
    })
  queueLabels()
}
function selectFeature(event: MapLayerMouseEvent) {
  const feature = event.features?.[0]
  if (feature?.id !== undefined && feature.properties?.campus === true)
    emit('select', String(feature.id))
}
onMounted(() => {
  if (!host.value) return
  try {
    // A same-origin worker preserves the forum's script-src 'self' CSP.
    maplibregl.setWorkerUrl(workerUrl)
    map = new maplibregl.Map({
      container: host.value,
      style: makeMapStyle(props.data, props.dimensional),
      center: [
        (props.campus.bounds[0]! + props.campus.bounds[2]!) / 2,
        (props.campus.bounds[1]! + props.campus.bounds[3]!) / 2,
      ],
      zoom: window.innerWidth < 700 ? 15.5 : 15.9,
      pitch: props.dimensional ? 32 : 0,
      bearing: props.campus.bearing,
      minZoom: 2,
      maxZoom: 20,
      maxPitch: 55,
      attributionControl: false,
      canvasContextAttributes: { antialias: true },
    })
    reset(0)
    map.on('load', () => {
      if (!alive) return
      for (const layer of selectableLayers) {
        map!.on('click', layer, selectFeature)
        map!.on('mouseenter', layer, () => {
          map!.getCanvas().style.cursor = 'pointer'
        })
        map!.on('mouseleave', layer, () => {
          map!.getCanvas().style.cursor = ''
        })
      }
      queueLabels()
      selectedChanged()
      updateLocation()
      emit('ready')
    })
    map.on('error', () => {
      if (alive) emit('failure')
    })
    map.on('move', queueLabels)
    let narrow = window.innerWidth < 700
    resize = new ResizeObserver(() => {
      map?.resize()
      const nextNarrow = window.innerWidth < 700
      if (nextNarrow !== narrow) {
        narrow = nextNarrow
        if (props.selected) selectedChanged()
        else reset()
      }
      queueLabels()
    })
    resize.observe(host.value)
  } catch {
    emit('failure')
  }
})
watch(() => props.selected, selectedChanged)
watch(() => props.category, queueLabels)
watch(() => props.sport, queueLabels)
watch(
  () => props.dimensional,
  (value) => {
    if (!map?.isStyleLoaded()) return
    map.setLayoutProperty('buildings', 'visibility', value ? 'visible' : 'none')
    map.setLayoutProperty(
      'building-roofs',
      'visibility',
      value ? 'none' : 'visible',
    )
    map.easeTo({
      pitch: value ? 32 : 0,
      duration: props.reducedMotion ? 0 : 500,
    })
  },
)
function reset(duration = props.reducedMotion ? 0 : 600) {
  if (!map) return
  const b = props.campus.bounds,
    narrow = window.innerWidth < 700
  map.fitBounds(
    [
      [b[0]!, b[1]!],
      [b[2]!, b[3]!],
    ],
    {
      bearing: props.campus.bearing,
      maxZoom: 18.2,
      padding: {
        top: narrow && props.panelOpen ? 200 : 55,
        bottom: 65,
        left: !narrow && props.panelOpen ? 350 : 45,
        right: narrow ? 65 : 85,
      },
      duration,
    },
  )
}
function updateLocation() {
  if (!map?.getSource('location')) return
  ;(map.getSource('location') as GeoJSONSource).setData(
    locationData(
      props.campus.coordinateMode === 'schematic' ? null : props.location,
    ),
  )
}
function focusLocation() {
  if (!map || !props.location) return
  updateLocation()
  map.easeTo({
    center: [props.location.longitude, props.location.latitude],
    zoom: Math.max(
      12,
      Math.min(18, 18 - Math.log2(Math.max(1, props.location.accuracy / 25))),
    ),
    offset: [window.innerWidth >= 700 && props.panelOpen ? 145 : 0, 0],
    duration: props.reducedMotion ? 0 : 700,
  })
}
watch(() => props.location, updateLocation)
watch(
  () => props.panelOpen,
  () => {
    if (props.selected) selectedChanged()
    else reset()
  },
)
function zoom(direction: number) {
  map?.zoomTo(map.getZoom() + direction, {
    duration: props.reducedMotion ? 0 : 250,
  })
}
function north() {
  map?.easeTo({ bearing: 0, duration: props.reducedMotion ? 0 : 400 })
}
defineExpose({ reset, zoom, north, focusLocation })
onBeforeUnmount(() => {
  alive = false
  resize?.disconnect()
  cancelAnimationFrame(frame)
  map?.remove()
  map = undefined
})
</script>

<template>
  <div class="campus-canvas" aria-label="校园交互地图">
    <div ref="host" class="campus-canvas__map" />
    <div class="campus-labels">
      <button
        v-for="label in mapLabels"
        :key="label.place.id"
        type="button"
        class="campus-label"
        :class="{
          'campus-label--sport': label.place.category === 'sport',
          'campus-label--selected': selected?.id === label.place.id,
        }"
        :style="{
          left: `${label.x}px`,
          top: `${label.y}px`,
          '--pin-color': categoryColors[label.place.category],
        }"
        :aria-label="label.place.name"
        :title="label.place.name"
        @click="emit('select', label.place.id)"
      >
        <span class="campus-label__dot" /><span>{{ label.text }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.campus-canvas,
.campus-canvas__map,
.campus-labels {
  position: absolute;
  inset: 0;
}
.campus-labels {
  pointer-events: none;
  overflow: hidden;
}
.campus-label {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  transform: translate(-50%, -50%);
  pointer-events: auto;
  cursor: pointer;
  border: 0;
  background: none;
  white-space: nowrap;
  color: #405444;
  font:
    500 12px/1.35 'PingFang SC',
    'Microsoft YaHei',
    sans-serif;
  text-shadow:
    0 1px 2px #fffdf3,
    1px 0 2px #fffdf3,
    -1px 0 2px #fffdf3,
    0 -1px 2px #fffdf3;
  transition: color 0.2s;
}
.campus-label__dot {
  width: 7px;
  height: 7px;
  border: 2px solid #fffdf3;
  background: var(--pin-color);
  border-radius: 50%;
  box-shadow: 0 1px 3px #35554220;
}
.campus-label--sport {
  color: #875337;
  font-weight: 650;
}
.campus-label--sport .campus-label__dot {
  width: 11px;
  height: 11px;
  border-width: 2px;
}
.campus-label:hover,
.campus-label--selected {
  color: #174e39;
  z-index: 2;
}
.campus-label--selected span:last-child {
  background: #fffdf6;
  padding: 5px 10px;
  border-radius: 6px;
  box-shadow: 0 3px 15px #254c2920;
}
.campus-label:focus-visible {
  outline: 2px solid #28654c;
  outline-offset: 4px;
  border-radius: 4px;
}
</style>
