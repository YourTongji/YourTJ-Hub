<script setup lang="ts">
import {
  computed,
  defineAsyncComponent,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
} from 'vue'
import { useI18n } from 'vue-i18n'
import type { LayoutPayload } from '@gooseforum/client'
import CampusMapMinePanel from '@/site/components/CampusMapMinePanel.vue'
import CampusMapSchedulePanel from '@/site/components/CampusMapSchedulePanel.vue'
import {
  officialCampusId,
  officialLocationTarget,
  type CampusMapTarget,
} from '@/site/campus-map/official-location'
import {
  ArrowLeft,
  ArrowUpRight,
  BookOpen,
  Building2,
  CalendarDays,
  Check,
  ChevronRight,
  PanelLeftClose,
  PanelLeftOpen,
  LocateFixed,
  Compass,
  Dumbbell,
  Info,
  Layers,
  List,
  MapPin,
  Minus,
  Plus,
  RotateCcw,
  Search,
  Share2,
  Trees,
  Utensils,
  Waves,
  X,
} from '@lucide/vue'
import {
  campuses,
  getCampus,
  containingCampus,
} from '@/site/campus-map/campuses'
import {
  requestLocation,
  type MapLocation,
  type LocationFailure,
} from '@/site/campus-map/location'
import {
  buildCatalog,
  makePlace,
  navigationHref as getNavigationHref,
  searchPlaces,
  type CampusData,
  type CampusPlace,
  type Category,
} from '@/site/campus-map/catalog'
import { categoryColors } from '@/site/campus-map/style'

const CampusCanvas = defineAsyncComponent(
  () => import('@/site/campus-map/CampusCanvas.vue'),
)
const demoMinePanel = import.meta.env.DEV && new URLSearchParams(window.location.search).get('demo') === '1'
  ? defineAsyncComponent(() => import('@/site/components/CampusMapMineDemoPanel.vue'))
  : null
const { t, te } = useI18n()
const page = defineProps<{ layout?: LayoutPayload }>()
const showTimetable = new URLSearchParams(window.location.search).get('mine') === '1'
function sportLabel(activity: string): string {
  const key = `campusMap.sports.${activity}`
  return te(key) ? t(key) : activity
}
const campus = ref(
  getCampus(new URLSearchParams(window.location.search).get('campus')),
)
const panelOpen = ref(true)
const location = ref<MapLocation | null>(null)
const locationStatus = ref<
  LocationFailure | 'idle' | 'loading' | 'success' | 'outside'
>('idle')
const locationNotice = ref(true)
let locationSequence = 0
let mineSelectionVersion = 0
let focusAfterLoad = false
let mapLoad: { campusId: string; promise: Promise<void> } | null = null
const campusName = computed(() => t(`campusMap.campuses.${campus.value.id}`))
const data = ref<CampusData | null>(null)
const query = ref('')
const category = ref<Category>('all')
const sport = ref('')
const selected = ref<CampusPlace | null>(null)
const selectedFromTimetable = ref(false)
const showAll = ref(false)
const infoDialog = ref<HTMLDialogElement>()
const buildingScheduleDialog = ref<HTMLDialogElement>()
const buildingScheduleOpen = ref(false)
const shareFallback = ref(false)
const shareInput = ref<HTMLInputElement>()
const shareUrl = ref('')
const mobileResults = ref(false)
const dimensional = ref(true)
const mapReady = ref(false)
const failure = ref<'data' | 'renderer' | null>(null)
const copied = ref(false)
const bearing = ref(-22)
const input = ref<HTMLInputElement>()
const canvas = ref<{
  reset: () => void
  zoom: (direction: number) => void
  north: () => void
  focusLocation: () => void
}>()
const reducedMotion = ref(false)
let controller: AbortController | undefined
let copyTimer: ReturnType<typeof setTimeout> | undefined
let motion: MediaQueryList | undefined
const syncMotion = () => {
  reducedMotion.value = motion?.matches ?? false
}
const places = computed(() => (data.value ? buildCatalog(data.value) : []))
const results = computed(() =>
  searchPlaces(places.value, query.value, category.value, sport.value, sportLabel),
)
const activeSearch = computed(
  () => query.value.trim() !== '' || category.value !== 'all' || showAll.value,
)
const suggestions = computed(() => campus.value.suggestions)
const selectedNavigationHref = computed(() =>
  selected.value
    ? getNavigationHref(selected.value, campus.value.coordinateMode, navigator.userAgent)
    : undefined,
)
const selectedWebNavigationHref = computed(() =>
  selected.value && /Android/i.test(navigator.userAgent)
    ? getNavigationHref(selected.value, campus.value.coordinateMode)
    : undefined,
)
const shownPlaces = computed(() =>
  activeSearch.value
    ? results.value
    : places.value
        .filter((p) => suggestions.value.includes(p.name))
        .sort(
          (a, b) =>
            suggestions.value.indexOf(a.name) -
            suggestions.value.indexOf(b.name),
        ),
)
const categories = [
  { key: 'all' as const, icon: Compass },
  { key: 'sport' as const, icon: Dumbbell },
  { key: 'academic' as const, icon: Building2 },
  { key: 'library' as const, icon: BookOpen },
  { key: 'food' as const, icon: Utensils },
  { key: 'living' as const, icon: MapPin },
]
const icons = {
  academic: Building2,
  library: BookOpen,
  sport: Dumbbell,
  food: Utensils,
  living: MapPin,
  place: Trees,
}
function iconFor(place: CampusPlace) {
  return place.sports.includes('swimming') ? Waves : icons[place.category]
}
function nameFor(place: CampusPlace) {
  return place.name.replace(/^同济大学/, '')
}
function categoryLabel(key: string) {
  return t(`campusMap.categories.${key}`)
}
function chooseCategory(value: Category) {
  category.value = value
  sport.value = ''
  closePlace()
  mobileResults.value = value !== 'all'
}
function browsePlaces() {
  showAll.value = true
  mobileResults.value = true
}
function select(id: string, fromTimetable = false) {
  mineSelectionVersion++
  selectedFromTimetable.value = fromTimetable
  const feature = data.value?.features.find((f) => String(f.id) === id)
  if (!feature?.properties.campus) {
    closePlace()
    return
  }
  panelOpen.value = true
  selected.value = makePlace(feature)
  mobileResults.value = false
  copied.value = false
  shareFallback.value = false
  const url = new URL(window.location.href)
  url.hash = `place=${encodeURIComponent(id)}`
  window.history.replaceState(window.history.state, '', url)
}
async function resolveMineLocation(campusName: string, room: string): Promise<CampusMapTarget | undefined> {
  const campusId = officialCampusId(campusName)
  if (!campusId) return undefined
  let pending: Promise<void> | undefined
  if (campus.value.id !== campusId) pending = switchCampus(campusId, false, false)
  else {
    closePlace()
    if (!data.value) pending = mapLoad?.campusId === campusId ? mapLoad.promise : load()
  }
  const version = mineSelectionVersion
  await pending
  if (version !== mineSelectionVersion || campus.value.id !== campusId) return undefined
  return officialLocationTarget(campusName, room, data.value)
}
function selectMinePlace(target: CampusMapTarget | null | undefined) {
  const version = ++mineSelectionVersion
  if (!target) {
    closePlace()
    return
  }
  if (version !== mineSelectionVersion || campus.value.id !== target.campusId) return
  select(target.featureId, true)
}
function selectBuildingScheduleTarget(target: CampusMapTarget | null) {
  if (!target) return
  buildingScheduleDialog.value?.close()
  select(target.featureId)
}
function openBuildingSchedule() {
  buildingScheduleOpen.value = true
  nextTick(() => buildingScheduleDialog.value?.showModal())
}
function matchMapLocation(campusName: string, room: string): CampusMapTarget | undefined {
  return officialLocationTarget(campusName, room, data.value)
}
function closePlace() {
  mineSelectionVersion++
  selected.value = null
  selectedFromTimetable.value = false
  const url = new URL(window.location.href)
  url.hash = ''
  window.history.replaceState(window.history.state, '', url)
}
function restoreSelection() {
  const params = new URLSearchParams(window.location.hash.slice(1))
  const id = params.get('place')
  if (id) select(id)
  else selected.value = null
}
function togglePanel() {
  panelOpen.value = !panelOpen.value
}
function menuHref(showMine: boolean) {
  const params = new URLSearchParams({ campus: campus.value.id })
  if (showMine) params.set('mine', '1')
  return `/map?${params}`
}
function switchCampus(id: string, fromLocation = false, invalidateMineSelection = true): Promise<void> {
  if (invalidateMineSelection) mineSelectionVersion++
  if (!fromLocation) {
    locationNotice.value = false
    locationSequence++
    focusAfterLoad = false
    if (locationStatus.value === 'loading') locationStatus.value = 'idle'
  }
  closePlace()
  campus.value = getCampus(id)
  query.value = ''
  category.value = 'all'
  sport.value = ''
  showAll.value = false
  mobileResults.value = false
  data.value = null
  mapReady.value = false
  const url = new URL(window.location.href)
  url.searchParams.set('campus', campus.value.id)
  window.history.replaceState(window.history.state, '', url)
  return load()
}
function changeCampus(event: Event) {
  switchCampus((event.target as HTMLSelectElement).value)
}
function ready() {
  mapReady.value = true
  if (focusAfterLoad) {
    focusAfterLoad = false
    canvas.value?.focusLocation()
  }
}
async function locate() {
  const sequence = ++locationSequence
  locationStatus.value = 'loading'
  locationNotice.value = true
  try {
    const point = await requestLocation(
      navigator.geolocation,
      window.isSecureContext,
    )
    if (sequence !== locationSequence) return
    location.value = point
    const destination = containingCampus(point.longitude, point.latitude)
    locationStatus.value = destination ? 'success' : 'outside'
    closePlace()
    if (destination && destination.id !== campus.value.id) {
      focusAfterLoad = true
      switchCampus(destination.id, true)
    } else if (campus.value.coordinateMode !== 'schematic') {
      await nextTick()
      if (sequence !== locationSequence) return
      // A cached fix can arrive before the data, async canvas or map style.
      if (mapReady.value && canvas.value) canvas.value.focusLocation()
      else focusAfterLoad = true
    }
  } catch (error) {
    if (sequence === locationSequence)
      locationStatus.value = [
        'unsupported',
        'insecure',
        'denied',
        'unavailable',
        'timeout',
      ].includes(String(error))
        ? (error as LocationFailure)
        : 'unavailable'
  }
}
function load(): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  failure.value = null
  const current = controller
  const pending = (async () => {
    try {
      const response = await fetch(campus.value.url, { signal: current.signal })
      if (!response.ok) throw new Error('Map data unavailable')
      const nextData = (await response.json()) as CampusData
      if (current.signal.aborted) return
      data.value = nextData
      restoreSelection()
    } catch {
      if (!current.signal.aborted) failure.value = 'data'
    }
  })()
  mapLoad = { campusId: campus.value.id, promise: pending }
  void pending.then(() => {
    if (mapLoad?.promise === pending) mapLoad = null
  })
  return pending
}
async function share() {
  const url = new URL(window.location.href)
  if (showTimetable) url.searchParams.delete('mine')
  const publicUrl = url.toString()
  try {
    await navigator.clipboard.writeText(publicUrl)
    copied.value = true
    clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copied.value = false
    }, 2200)
  } catch {
    shareUrl.value = publicUrl
    shareFallback.value = true
    await nextTick()
    shareInput.value?.select()
  }
}
function keyboard(event: KeyboardEvent) {
  if (infoDialog.value?.open) return
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    closePlace()
    panelOpen.value = true
    void nextTick(() => input.value?.focus())
  }
  if (event.key === 'Escape') {
    mobileResults.value = false
    closePlace()
    input.value?.blur()
  }
}
function reset() {
  closePlace()
  canvas.value?.reset()
}
function retry() {
  data.value = null
  mapReady.value = false
  void nextTick(load)
}
function clearLocation() {
  locationSequence++
  location.value = null
  locationStatus.value = 'idle'
  focusAfterLoad = false
}
onMounted(() => {
  window.addEventListener('pagehide', clearLocation)
  motion = window.matchMedia('(prefers-reduced-motion: reduce)')
  syncMotion()
  motion.addEventListener('change', syncMotion)
  window.addEventListener('keydown', keyboard)
  window.addEventListener('hashchange', restoreSelection)
  void load()
})
onBeforeUnmount(() => {
  clearLocation()
  window.removeEventListener('pagehide', clearLocation)
  controller?.abort()
  clearTimeout(copyTimer)
  motion?.removeEventListener('change', syncMotion)
  window.removeEventListener('keydown', keyboard)
  window.removeEventListener('hashchange', restoreSelection)
})
</script>

<template>
  <div
    class="campus-atlas"
    :class="{
      'campus-atlas--selected': selected && panelOpen && !selectedFromTimetable,
      'campus-atlas--collapsed': !panelOpen,
    }"
  >
    <header class="atlas-header">
      <a href="/" class="atlas-brand" :aria-label="t('campusMap.community')"
        ><span class="atlas-brand__mark"
          ><Compass :size="22" :stroke-width="1.5" /></span
        ><span
          >yourtj<span class="atlas-brand__separator">/</span
          ><span class="atlas-brand__name">{{
            t('campusMap.atlas')
          }}</span></span
        ></a
      >
      <div class="atlas-campus">
        <MapPin :size="14" /><select
          :value="campus.id"
          :aria-label="t('campusMap.switchCampus')"
          @change="changeCampus"
        >
          <option v-for="item in campuses" :key="item.id" :value="item.id">
            {{ t(`campusMap.campuses.${item.id}`) }}
          </option>
        </select>
      </div>
      <nav class="atlas-header__actions">
        <button
          type="button"
          :aria-label="t('campusMap.about')"
          @click="infoDialog?.showModal()"
        >
          <Info :size="18" /></button
        ><a href="/"
          >{{ t('campusMap.community') }}<ArrowUpRight :size="15"
        /></a>
      </nav>
    </header>

    <main class="atlas-world">
      <CampusCanvas
        :key="campus.id"
        :campus="campus"
        :panel-open="panelOpen"
        :location="location"
        v-if="data"
        ref="canvas"
        :data="data"
        :places="places"
        :selected="selected"
        :category="category"
        :sport="sport"
        :dimensional="dimensional"
        :reduced-motion="reducedMotion"
        @select="select"
        @ready="ready"
        @failure="failure = 'renderer'"
        @bearing="bearing = $event"
      />
      <div class="atlas-vignette" />

      <button
        class="atlas-panel-toggle"
        type="button"
        :aria-label="
          panelOpen ? t('campusMap.collapse') : t('campusMap.expand')
        "
        :aria-expanded="panelOpen"
        aria-controls="atlas-panel"
        @click="togglePanel"
      >
        <PanelLeftClose v-if="panelOpen" :size="18" /><PanelLeftOpen
          v-else
          :size="18"
        /><span v-if="!panelOpen">{{ t('campusMap.expand') }}</span>
      </button>
      <aside
        id="atlas-panel"
        v-show="panelOpen"
        class="atlas-explorer"
        :class="{ 'atlas-explorer--results': mobileResults }"
        :aria-label="t('campusMap.explore')"
      >
        <nav class="atlas-menu-switch" :aria-label="t('campusMap.menu.label')">
          <a :href="menuHref(false)" :aria-current="!showTimetable ? 'page' : undefined">
            <MapPin :size="15" />{{ t('campusMap.menu.places') }}
          </a>
          <a :href="menuHref(true)" :aria-current="showTimetable ? 'page' : undefined">
            <BookOpen :size="15" />{{ t('campusMap.menu.courses') }}
          </a>
        </nav>
        <component
          :is="demoMinePanel"
          v-if="showTimetable && demoMinePanel"
          @select="selectMinePlace"
        />
        <CampusMapMinePanel
          v-else-if="showTimetable"
          :authenticated="page.layout?.viewer.isAuthenticated ?? false"
          :resolve-location="resolveMineLocation"
          @select="selectMinePlace"
        />
        <template v-else>
        <div class="atlas-intro">
          <div class="atlas-eyebrow"><span /> TONGJI CAMPUS ATLAS</div>
          <h1>
            {{ t('campusMap.title') }}
            <span>{{ t('campusMap.titleEnd') }}</span>
          </h1>
          <p>{{ t('campusMap.subtitle') }}</p>
        </div>
        <div class="atlas-search">
          <Search :size="18" /><input
            ref="input"
            v-model="query"
            :aria-label="t('campusMap.search')"
            :placeholder="t('campusMap.searchPlaceholder')"
            @focus="mobileResults = true"
          /><button
            v-if="query"
            type="button"
            :aria-label="t('campusMap.clear')"
            @click="query = ''"
          >
            <X :size="15" /></button
          ><kbd v-else>⌘ K</kbd>
        </div>
        <div class="atlas-categories" :aria-label="t('campusMap.filter')">
          <button
            v-for="item in categories"
            :key="item.key"
            type="button"
            :aria-pressed="category === item.key"
            :class="{ active: category === item.key }"
            @click="chooseCategory(item.key)"
          >
            <component :is="item.icon" :size="15" />{{
              categoryLabel(item.key)
            }}
          </button>
        </div>
        <div v-if="category === 'sport'" class="atlas-sports">
          <button
            type="button"
            :aria-pressed="sport === ''"
            @click="sport = ''"
          >
            {{ t('campusMap.allSports') }}</button
          ><button
            v-for="key in [...new Set(places.flatMap((place) => place.sports))]"
            :key="key"
            type="button"
            :aria-pressed="sport === key"
            @click="sport = sport === key ? '' : key"
          >
            {{ sportLabel(key) }}
          </button>
        </div>
        <section class="atlas-results">
          <div class="atlas-results__heading">
            <span>{{
              activeSearch
                ? t('campusMap.results', { count: results.length })
                : t('campusMap.startHere')
            }}</span
            ><button
              class="atlas-mobile-close"
              type="button"
              :aria-label="t('campusMap.close')"
              @click="mobileResults = false"
            >
              <X :size="16" /></button
            ><span v-if="!activeSearch" class="atlas-results__hint">{{
              t('campusMap.onMap')
            }}</span>
          </div>
          <div class="atlas-results__list" aria-live="polite">
            <button
              v-for="place in shownPlaces"
              :key="place.id"
              type="button"
              class="atlas-place"
              :class="{ 'atlas-place--active': selected?.id === place.id }"
              @click="select(place.id)"
            >
              <span
                class="atlas-place__icon"
                :style="{ '--place-color': categoryColors[place.category] }"
                ><component
                  :is="iconFor(place)"
                  :size="19"
                  :stroke-width="1.5" /></span
              ><span class="atlas-place__text"
                ><strong>{{ nameFor(place) }}</strong
                ><small
                  >{{ categoryLabel(place.category)
                  }}<template v-if="place.sports.length">
                    ·
                    {{
                      place.sports.map(sportLabel).join(' / ')
                    }}</template
                  ></small
                ></span
              ><ChevronRight :size="15" class="atlas-place__arrow" />
            </button>
            <p v-if="!shownPlaces.length && data" class="atlas-empty">
              {{ t('campusMap.noResults') }}
            </p>
          </div>
          <button
            v-if="!activeSearch"
            type="button"
            class="atlas-browse"
            @click="browsePlaces"
          >
            <List :size="15" />{{
              t('campusMap.browse', { count: places.length })
            }}<ArrowUpRight :size="14" />
          </button>
          <button
            v-else-if="category === 'all' && showAll && !query"
            type="button"
            class="atlas-browse"
            @click="showAll = false"
          >
            <ArrowLeft :size="14" />{{ t('campusMap.backToStart') }}
          </button>
        </section>
        <div class="atlas-explorer__foot">
          <Trees :size="14" /><span>{{ t('campusMap.exploreNote') }}</span>
        </div>
        </template>
      </aside>

      <div v-if="!mapReady || failure" class="atlas-map-status" role="status">
        <template v-if="failure"
          ><MapPin :size="25" /><strong>{{ t(failure === 'data' ? 'campusMap.dataFailed' : 'campusMap.failed') }}</strong
          ><span>{{ t(failure === 'data' ? 'campusMap.dataFallback' : 'campusMap.fallback') }}</span
          ><button type="button" @click="retry">
            {{ t('campusMap.retry') }}
          </button></template
        ><template v-else
          ><Compass class="atlas-loading" :size="28" /><span>{{
            t('campusMap.loading')
          }}</span></template
        >
      </div>

      <div class="atlas-orientation">
        <button
          type="button"
          :aria-label="t('campusMap.north')"
          @click="canvas?.north()"
        >
          <span>N</span
          ><Compass
            :size="32"
            :stroke-width="1.2"
            :style="{ transform: `rotate(${-bearing}deg)` }"
          />
        </button>
      </div>
      <div
        v-if="locationNotice && locationStatus !== 'idle'"
        class="atlas-location-notice"
        role="status"
        aria-live="polite"
      >
        <LocateFixed :size="16" /><span>{{
          t(locationStatus === 'outside' && campus.coordinateMode === 'schematic'
            ? 'campusMap.outsideSchematic'
            : `campusMap.location.${locationStatus}`, {
            accuracy: Math.round(location?.accuracy ?? 0),
          })
        }}</span
        ><button
          v-if="locationStatus !== 'loading'"
          type="button"
          :aria-label="t('campusMap.close')"
          @click="locationNotice = false"
        >
          <X :size="14" />
        </button>
      </div>
      <div class="atlas-controls" :aria-label="t('campusMap.controls')">
        <button
          type="button"
          :aria-label="t('campusMap.locate')"
          :title="t('campusMap.locate')"
          :disabled="locationStatus === 'loading'"
          :class="{ 'atlas-locating': locationStatus === 'loading' }"
          @click="locate"
        >
          <LocateFixed :size="19" />
        </button>
        <button
          type="button"
          class="atlas-view"
          :aria-pressed="dimensional"
          @click="dimensional = !dimensional"
        >
          <Layers :size="17" /><span>{{ dimensional ? '2.5D' : '2D' }}</span>
        </button>
        <div class="atlas-zoom">
          <button
            type="button"
            :aria-label="t('campusMap.zoomIn')"
            @click="canvas?.zoom(1)"
          >
            <Plus :size="19" /></button
          ><button
            type="button"
            :aria-label="t('campusMap.zoomOut')"
            @click="canvas?.zoom(-1)"
          >
            <Minus :size="19" />
          </button>
        </div>
        <button type="button" :aria-label="t('campusMap.reset')" @click="reset">
          <RotateCcw :size="17" />
        </button>
      </div>

      <Transition name="atlas-detail"
        ><section
          v-if="selected && panelOpen && !selectedFromTimetable"
          class="atlas-detail"
          :aria-label="t('campusMap.details')"
        >
          <div
            class="atlas-detail__cover"
            :class="`atlas-detail__cover--${selected.category}`"
          >
            <div class="atlas-detail__contour" />
            <component
              :is="iconFor(selected)"
              :size="46"
              :stroke-width="1"
            /><span>{{ categoryLabel(selected.category) }}</span
            ><button
              type="button"
              :aria-label="t('campusMap.close')"
              @click="closePlace"
            >
              <X :size="17" />
            </button>
          </div>
          <div class="atlas-detail__body">
            <div class="atlas-eyebrow">
              {{ campusName }}<span class="atlas-detail__dot">·</span
              >{{
                selected.indoor
                  ? t('campusMap.building')
                  : t('campusMap.outdoor')
              }}
            </div>
            <h2>{{ nameFor(selected) }}</h2>
            <p v-if="selected.aliases.length" class="atlas-detail__alias">
              {{ selected.aliases[0] }}
            </p>
            <div v-if="selected.sports.length" class="atlas-detail__sports">
              <span v-for="activity in selected.sports" :key="activity">{{
                sportLabel(activity)
              }}</span>
            </div>
            <p class="atlas-detail__note">
              {{
                selected.category === 'sport'
                  ? t('campusMap.sportNote')
                  : t('campusMap.placeNote')
              }}
            </p>
            <button v-if="selected.indoor && selected.category === 'academic'" type="button" class="atlas-schedule-open" @click="openBuildingSchedule">
              <CalendarDays :size="16" />{{ t('campusMap.schedule.openBuilding') }}
            </button>
            <a
              v-if="selectedNavigationHref"
              class="atlas-share"
              :href="selectedNavigationHref"
              target="_blank"
              rel="noopener noreferrer"
            >{{ t('campusMap.navigate') }}</a>
            <a
              v-if="selectedWebNavigationHref"
              class="atlas-share"
              :href="selectedWebNavigationHref"
              target="_blank"
              rel="noopener noreferrer"
            >{{ t('campusMap.navigateWeb') }}</a>
            <button type="button" class="atlas-share" @click="share">
              <Check v-if="copied" :size="16" /><Share2 v-else :size="16" />{{
                copied ? t('campusMap.copied') : t('campusMap.share')
              }}</button
            ><input
              v-if="shareFallback"
              ref="shareInput"
              class="atlas-share-url"
              readonly
              :value="shareUrl"
              :aria-label="t('campusMap.share')"
            />
          </div></section
      ></Transition>

      <dialog ref="buildingScheduleDialog" class="atlas-building-schedule" :aria-label="t('campusMap.schedule.title')" @close="buildingScheduleOpen = false">
        <div class="atlas-building-schedule__header">
          <h2>{{ t('campusMap.schedule.title') }}</h2>
          <button type="button" :aria-label="t('campusMap.close')" @click="buildingScheduleDialog?.close()"><X :size="18" /></button>
        </div>
        <CampusMapSchedulePanel
          v-if="buildingScheduleOpen && selected"
          :building="{ campusId: campus.id, featureId: selected.id, name: nameFor(selected) }"
          :match-location="matchMapLocation"
          :resolve-location="resolveMineLocation"
          @select="selectBuildingScheduleTarget"
        />
      </dialog>

      <div v-if="campus.coordinateMode === 'schematic'" class="atlas-plan-note">
        {{ t('campusMap.schematicNote') }}
      </div>
      <footer class="atlas-map-footer">
        <div class="atlas-legend">
          <span
            v-for="key in ['academic', 'sport', 'living'] as const"
            :key="key"
            ><i :style="{ background: categoryColors[key] }" />{{
              categoryLabel(key)
            }}</span
          ><span
            ><i style="background: #a3d2d8" />{{ t('campusMap.water') }}</span
          >
        </div>
        <div class="atlas-attribution">
          <a
            href="https://www.openstreetmap.org/copyright"
            target="_blank"
            rel="noopener noreferrer"
            >© OpenStreetMap</a
          ><span>·</span
          ><button type="button" @click="infoDialog?.showModal()">
            {{ t('campusMap.dataNote') }}
          </button>
        </div>
      </footer>

      <dialog
        ref="infoDialog"
        class="atlas-info"
        :aria-label="t('campusMap.about')"
      >
        <button
          type="button"
          class="atlas-info__close"
          :aria-label="t('campusMap.close')"
          @click="infoDialog?.close()"
        >
          <X :size="20" /></button
        ><Compass :size="30" />
        <h2>{{ t('campusMap.about') }}</h2>
        <p>{{ t('campusMap.aboutText') }}</p>
        <p>{{ t('campusMap.coverageText') }}</p>
        <a
          href="https://github.com/WALKERKILLER/YourTJ-Pulse"
          target="_blank"
          rel="noopener noreferrer"
          >YourTJ Pulse <ArrowUpRight :size="14" /></a
        ><a
          href="https://www.openstreetmap.org/copyright"
          target="_blank"
          rel="noopener noreferrer"
          >OpenStreetMap · ODbL <ArrowUpRight :size="14"
        /></a>
      </dialog>
    </main>
  </div>
</template>

<style scoped>
.campus-atlas {
  --atlas-ink: #273c30;
  --atlas-muted: #7b8473;
  --atlas-paper: #fffef8;
  --atlas-green: #2d6047;
  --atlas-line: #e6e8db;
  position: relative;
  height: 100dvh;
  min-height: 540px;
  overflow: hidden;
  background: #f0eee5;
  color: var(--atlas-ink);
  font-family: 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif;
  font-size: 14px;
}
.campus-atlas button,
.campus-atlas a {
  -webkit-tap-highlight-color: transparent;
}
.campus-atlas button {
  cursor: pointer;
}
.campus-atlas button:focus-visible,
.campus-atlas a:focus-visible,
.campus-atlas input:focus-visible {
  outline: 2px solid #437b58;
  outline-offset: 3px;
}
.campus-atlas button:disabled {
  cursor: default;
}
.atlas-header {
  height: 68px;
  position: relative;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 0 28px;
  background: #fffef9;
  border-bottom: 1px solid var(--atlas-line);
}
.atlas-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 24px;
  font-weight: 750;
  letter-spacing: -1px;
  text-decoration: none;
  color: var(--atlas-ink);
}
.atlas-brand__mark {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  background: var(--atlas-green);
  color: #f9f9e8;
  border-radius: 10px;
}
.atlas-brand__separator {
  font-weight: 300;
  color: #b3baa8;
  margin: 0 13px;
}
.atlas-brand__name {
  font-size: 16px;
  font-weight: 550;
  letter-spacing: 1px;
}
.atlas-campus {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  color: #66735d;
}
.atlas-campus__tag {
  font-size: 9px;
  letter-spacing: 1.3px;
  border-left: 1px solid #d3d9c9;
  padding-left: 10px;
  margin-left: 7px;
  color: #8a9682;
}
.atlas-header__actions {
  display: flex;
  gap: 20px;
  align-items: center;
}
.atlas-header__actions a {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: #64735e;
  text-decoration: none;
}
.atlas-header__actions button {
  background: none;
  border: 0;
  padding: 8px;
  color: #687961;
}
.atlas-world {
  height: calc(100% - 68px);
  position: relative;
  isolation: isolate;
}
.atlas-vignette {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(90deg, #f6f3e644, transparent 42%),
    linear-gradient(0deg, #f2efdfb8, transparent 18%);
}
.atlas-explorer {
  position: absolute;
  z-index: 3;
  left: 26px;
  top: 26px;
  width: 304px;
  max-height: calc(100% - 105px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid #fffdf9;
  border-radius: 18px;
  background: #fffef8f5;
  box-shadow:
    0 10px 40px #5260440c,
    0 1px 3px #52604406;
  backdrop-filter: blur(14px);
}
.atlas-menu-switch {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 5px;
  flex: none;
  margin: 14px 16px 0;
  padding: 4px;
  border-radius: 10px;
  background: #f1f2e9;
}
.atlas-menu-switch a {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  border-radius: 7px;
  color: #68735f;
  font-size: 12px;
  text-decoration: none;
  transition:
    background-color 160ms ease,
    color 160ms ease,
    box-shadow 160ms ease,
    transform 160ms ease;
}
.atlas-menu-switch a:hover {
  color: #344c30;
}
.atlas-menu-switch a:active {
  transform: scale(0.97);
}
.atlas-menu-switch a[aria-current='page'] {
  background: #fffef8;
  color: #344c30;
  box-shadow: 0 1px 4px #52604418;
  animation: atlas-menu-select 180ms ease-out;
}
@keyframes atlas-menu-select {
  from {
    opacity: 0.65;
    transform: scale(0.97);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}
.atlas-menu-switch a:focus-visible {
  outline: 2px solid #829b6e;
  outline-offset: 1px;
}
.atlas-intro {
  padding: 25px 23px 20px;
}
.atlas-eyebrow {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 9px;
  letter-spacing: 1.8px;
  color: #778970;
  font-weight: 600;
}
.atlas-eyebrow > span:first-child:not(.atlas-detail__dot) {
  width: 5px;
  height: 5px;
  background: #6f945b;
  border-radius: 50%;
}
.atlas-intro h1 {
  font-family: 'Noto Serif SC', 'Songti SC', serif;
  font-size: 28px;
  line-height: 1.6;
  letter-spacing: 1px;
  font-weight: 600;
  margin: 14px 0 9px;
}
.atlas-intro h1 span {
  display: block;
}
.atlas-intro p {
  font-size: 11px;
  color: #8a907d;
  letter-spacing: 0.4px;
  margin: 0;
}
.atlas-search {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 46px;
  flex: none;
  margin: 0 17px;
  background: #f1f2e9;
  border: 1px solid #e2e5d6;
  border-radius: 9px;
  padding: 0 12px;
  color: #839176;
}
.atlas-search:focus-within {
  border-color: #829b6e;
  background: #fcfcf5;
  box-shadow: 0 0 0 3px #b4c6a328;
}
.atlas-search input {
  width: 100%;
  min-width: 0;
  background: none;
  border: 0;
  color: var(--atlas-ink);
  font: inherit;
  font-size: 12px;
  outline: none !important;
}
.atlas-search input::placeholder {
  color: #929985;
}
.atlas-search kbd {
  white-space: nowrap;
  font: 10px system-ui;
  color: #9ba68d;
}
.atlas-search button {
  padding: 2px;
  background: none;
  border: 0;
  color: #728365;
  display: grid;
  place-items: center;
}
.atlas-categories {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 7px;
  padding: 15px 17px;
}
.atlas-categories button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid #eceee1;
  border-radius: 7px;
  background: #fbfcf4;
  color: #747f6a;
  font: inherit;
  font-size: 11px;
  padding: 9px 2px;
  transition:
    background 0.15s,
    color 0.15s;
}
.atlas-categories button:hover {
  background: #edf1e2;
}
.atlas-categories button.active {
  background: var(--atlas-green);
  border-color: var(--atlas-green);
  color: #fffef5;
}
.atlas-sports {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 0 18px 14px;
}
.atlas-sports button {
  font-size: 10px;
  padding: 4px 7px;
  border-radius: 20px;
  border: 0;
  background: #f0f1e8;
  color: #728365;
}
.atlas-sports button[aria-pressed='true'] {
  background: #f1e0d2;
  color: #9b6347;
}
.atlas-results {
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0 17px;
}
.atlas-results__heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 13px 3px 6px;
  border-top: 1px solid var(--atlas-line);
  font-size: 10px;
  letter-spacing: 1px;
  color: #6b795f;
  flex: none;
}
.atlas-results__hint {
  font-size: 9px;
  color: #a3ac96;
  letter-spacing: 0;
}
.atlas-results__list {
  overflow: auto;
  min-height: 0;
  max-height: 320px;
  scrollbar-width: thin;
  scrollbar-color: #d7deca transparent;
}
.atlas-place {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 11px;
  text-align: left;
  padding: 11px 3px;
  background: none;
  border: 0;
  border-bottom: 1px solid #edf0e5;
  color: var(--atlas-ink);
}
.atlas-place:hover,
.atlas-place--active {
  background: #f2f5e9;
  border-radius: 7px;
}
.atlas-place__icon {
  display: grid;
  place-items: center;
  flex: none;
  width: 37px;
  height: 37px;
  border-radius: 9px;
  background: color-mix(in srgb, var(--place-color) 16%, white);
  color: var(--place-color);
}
.atlas-place__text {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.atlas-place strong {
  font-size: 12px;
  font-weight: 550;
  line-height: 1.45;
}
.atlas-place small {
  font-size: 9px;
  color: #97a087;
}
.atlas-place__arrow {
  color: #aab49b;
}
.atlas-empty {
  padding: 25px 8px;
  color: #7f8c72;
  font-size: 12px;
  line-height: 1.8;
}
.atlas-browse {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 14px 0;
  border: 0;
  background: none;
  color: #77866b;
  font-size: 10px;
}
.atlas-browse svg:last-child {
  margin-left: auto;
}
.atlas-browse svg:first-child {
  margin-right: auto;
}
.atlas-explorer__foot {
  display: flex;
  gap: 6px;
  justify-content: center;
  align-items: center;
  font-size: 9px;
  color: #9ba68b;
  background: #f3f5ec;
  padding: 10px 12px;
  flex: none;
}
.atlas-orientation {
  position: absolute;
  right: 32px;
  top: 25px;
  z-index: 2;
}
.atlas-orientation button {
  border: 0;
  background: none;
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 5px;
  color: #526849;
}
.atlas-orientation span {
  font-size: 10px;
  font-weight: 600;
}
.atlas-orientation svg {
  transition: transform 0.2s;
}
.atlas-controls {
  position: absolute;
  right: 26px;
  bottom: 64px;
  display: flex;
  flex-direction: column;
  gap: 9px;
  z-index: 4;
}
.atlas-controls > button,
.atlas-zoom {
  border: 1px solid #fffdf8;
  background: #fffef8ed;
  border-radius: 10px;
  box-shadow: 0 3px 15px #57623d0c;
}
.atlas-controls button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  color: #53694a;
  min-width: 42px;
  min-height: 42px;
  padding: 10px;
  font: inherit;
  font-size: 10px;
}
.atlas-zoom {
  overflow: hidden;
}
.atlas-zoom button {
  width: 100%;
  background: none;
  border: 0;
}
.atlas-zoom button + button {
  border-top: 1px solid #e8ebdc;
}
.atlas-controls button:hover {
  background: #f1f4e5;
}
.atlas-view {
  flex-direction: column;
  font-weight: 600 !important;
}
.atlas-map-footer {
  position: absolute;
  left: 27px;
  right: 27px;
  bottom: 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 15px;
  font-size: 9px;
  color: #7d8d6b;
  z-index: 3;
}
.atlas-legend {
  display: flex;
  gap: 16px;
  align-items: center;
  background: #fffef7bb;
  padding: 10px 14px;
  border-radius: 7px;
}
.atlas-legend span {
  display: flex;
  align-items: center;
  gap: 5px;
}
.atlas-legend i {
  width: 6px;
  height: 6px;
  border-radius: 2px;
}
.atlas-attribution {
  display: flex;
  align-items: center;
  gap: 8px;
  font: 9px system-ui;
}
.atlas-attribution a {
  color: inherit;
  text-decoration: none;
}
.atlas-attribution button {
  padding: 0;
  background: none;
  border: 0;
  color: inherit;
  font: inherit;
}
.atlas-detail {
  position: absolute;
  right: 85px;
  bottom: 66px;
  width: 285px;
  z-index: 5;
  background: var(--atlas-paper);
  border-radius: 15px;
  overflow: hidden;
  border: 1px solid #fffdf7;
  box-shadow: 0 12px 45px #42543518;
}
.atlas-detail__cover {
  height: 107px;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #8e9471;
  background: #e9eddc;
  overflow: hidden;
}
.atlas-detail__cover--sport {
  background: #eaddcc;
  color: #ad8666;
}
.atlas-detail__cover--library {
  background: #ece0ce;
  color: #b19470;
}
.atlas-detail__cover--living {
  background: #e0e5ed;
  color: #99a9bf;
}
.atlas-detail__cover > span {
  position: absolute;
  left: 17px;
  bottom: 13px;
  font-size: 9px;
  letter-spacing: 2px;
}
.atlas-detail__cover > button {
  position: absolute;
  top: 10px;
  right: 10px;
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border: 0;
  border-radius: 50%;
  background: #fff9;
  color: #849076;
}
.atlas-detail__contour {
  position: absolute;
  inset: -100px 35px;
  border: 1px solid #ffffff60;
  border-radius: 50%;
  transform: rotate(-40deg);
  box-shadow:
    0 0 0 20px #ffffff20,
    0 0 0 21px #ffffff60,
    0 0 0 45px #ffffff20,
    0 0 0 46px #ffffff40;
}
.atlas-detail__cover > svg {
  position: relative;
}
.atlas-detail__body {
  padding: 18px 20px 20px;
}
.atlas-detail__body .atlas-eyebrow {
  font-size: 8px;
  letter-spacing: 1px;
}
.atlas-detail__dot {
  color: #b1baa1;
}
.atlas-detail h2 {
  margin: 11px 0 8px;
  font:
    600 22px/1.4 'Noto Serif SC',
    'Songti SC',
    serif;
}
.atlas-detail__alias {
  font-size: 10px;
  color: #9ca58e;
  margin: 0 0 13px;
}
.atlas-detail__sports {
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
  padding-top: 6px;
}
.atlas-detail__sports > span {
  font-size: 10px;
  padding: 5px 8px;
  background: #f4ece1;
  color: #a5815e;
  border-radius: 5px;
}
.atlas-detail__note {
  font-size: 10px;
  line-height: 1.9;
  color: #8f9a7e;
  margin: 14px 0;
}
.atlas-schedule-open { display: flex; align-items: center; justify-content: center; gap: 8px; min-height: 38px; margin-bottom: 8px; border: 1px solid #dce7dd; border-radius: 10px; background: #edf4ed; color: #315b3e; font: inherit; font-weight: 600; cursor: pointer; }
.atlas-building-schedule { width: min(540px, calc(100vw - 24px)); max-height: min(84dvh, 780px); padding: 18px; border: 1px solid #e3e8de; border-radius: 16px; background: #fffef8; color: #24342f; }
.atlas-building-schedule::backdrop { background: #17231d88; backdrop-filter: blur(3px); }
.atlas-building-schedule__header { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.atlas-building-schedule__header h2 { margin: 0; font-size: 18px; }
.atlas-building-schedule__header button { display: grid; place-items: center; width: 34px; height: 34px; border: 0; border-radius: 8px; background: transparent; color: inherit; }
.atlas-share {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  border: 0;
  border-radius: 8px;
  background: var(--atlas-green);
  color: #fffdf2;
  width: 100%;
  padding: 11px;
  font-size: 11px;
  cursor: pointer;
  text-decoration: none;
}
.atlas-detail-enter-active,
.atlas-detail-leave-active {
  transition:
    opacity 0.2s,
    transform 0.2s;
}
.atlas-detail-enter-from,
.atlas-detail-leave-to {
  opacity: 0;
  transform: translateY(12px);
}
.atlas-map-status {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 13px;
  background: #f2f1e7ee;
  color: #71805f;
  font-size: 12px;
  z-index: 1;
  padding-left: 340px;
}
.atlas-map-status > span {
  max-width: 260px;
  text-align: center;
  line-height: 1.8;
}
.atlas-map-status button {
  padding: 8px 20px;
  border: 1px solid #adbc97;
  background: #fffdf3;
  border-radius: 7px;
}
.atlas-loading {
  animation: atlas-spin 4s linear infinite;
}
@keyframes atlas-spin {
  to {
    transform: rotate(360deg);
  }
}
.atlas-info {
  margin: auto;
  border: 0;
  color: var(--atlas-ink);
  position: fixed;
  inset: 0;
  width: calc(100% - 32px);
  height: fit-content;
  max-height: calc(100dvh - 40px);
  overflow: auto;
  max-width: 410px;
  padding: 32px;
  border-radius: 18px;
  background: var(--atlas-paper);
  box-shadow: 0 20px 70px #1b321433;
}
.atlas-info > svg {
  color: #82936c;
}
.atlas-info h2 {
  font-family: 'Noto Serif SC', 'Songti SC', serif;
  font-size: 22px;
  margin: 20px 0 16px;
}
.atlas-info p {
  font-size: 12px;
  line-height: 1.9;
  color: #859075;
}
.atlas-info a {
  display: flex;
  gap: 6px;
  align-items: center;
  color: #4d7551;
  font-size: 11px;
  margin-top: 15px;
  text-decoration: none;
}
.atlas-info__close {
  position: absolute;
  right: 15px;
  top: 15px;
  border: 0;
  background: none;
  color: #7b8b6a;
}
.atlas-mobile-close {
  display: none;
}
.atlas-info::backdrop {
  background: #233e3066;
  backdrop-filter: blur(4px);
}
.atlas-share-url {
  width: 100%;
  margin-top: 8px;
  font-size: 11px;
  padding: 8px;
  border: 1px solid var(--atlas-line);
  border-radius: 6px;
}
@media (min-width: 1500px) {
  .atlas-explorer {
    left: 36px;
    top: 34px;
    width: 322px;
  }
  .atlas-intro {
    padding: 29px 25px 24px;
  }
  .atlas-intro h1 {
    font-size: 31px;
  }
  .atlas-place {
    padding: 13px 3px;
  }
}
@media (max-height: 760px) and (min-width: 701px) {
  .atlas-intro {
    padding: 18px 23px 14px;
  }
  .atlas-intro h1 {
    font-size: 23px;
    line-height: 1.4;
    margin: 10px 0 8px;
  }
  .atlas-intro h1 span {
    display: block;
    font-size: 22px;
    white-space: nowrap;
  }
  .atlas-intro p {
    display: none;
  }
  .atlas-explorer {
    top: 20px;
  }
  .atlas-results__list {
    max-height: 230px;
  }
  .atlas-explorer__foot {
    display: none;
  }
}
@media (max-width: 1000px) and (min-width: 701px) {
  .atlas-explorer {
    width: 275px;
    left: 20px;
  }
  .atlas-detail {
    width: 255px;
    right: 78px;
  }
  .atlas-brand__name {
    font-size: 13px;
  }
  .atlas-campus__tag {
    display: none;
  }
}
@media (max-width: 700px) {
  .campus-atlas {
    min-height: 450px;
  }
  .atlas-header {
    height: 56px;
    padding: 0 16px;
    gap: 12px;
  }
  .atlas-brand {
    font-size: 21px;
    gap: 7px;
  }
  .atlas-brand__mark {
    width: 29px;
    height: 29px;
    border-radius: 8px;
  }
  .atlas-brand__mark svg {
    width: 18px;
  }
  .atlas-brand__separator {
    margin: 0 8px;
  }
  .atlas-brand__name {
    font-size: 12px;
    letter-spacing: 0;
  }
  .atlas-campus {
    font-size: 10px;
    margin-left: auto;
    gap: 4px;
  }
  .atlas-campus svg {
    width: 12px;
  }
  .atlas-campus__tag,
  .atlas-header__actions a {
    display: none;
  }
  .atlas-header__actions {
    gap: 0;
  }
  .atlas-header__actions button {
    padding: 5px;
  }
  .atlas-world {
    height: calc(100% - 56px);
  }
  .atlas-explorer {
    left: 12px;
    right: 12px;
    top: 12px;
    width: auto;
    max-height: calc(100% - 135px);
    border-radius: 13px;
    box-shadow: 0 5px 25px #394f3010;
  }
  .atlas-intro,
  .atlas-explorer__foot {
    display: none;
  }
  .atlas-search {
    margin: 11px 11px 0;
    height: 43px;
    min-height: 43px;
  }
  .atlas-categories {
    display: flex;
    overflow: auto;
    gap: 6px;
    padding: 10px 11px;
    scrollbar-width: none;
    flex: none;
  }
  .atlas-categories button {
    flex: none;
    white-space: nowrap;
    font-size: 10px;
    padding: 8px 9px;
  }
  .atlas-categories button svg {
    width: 13px;
  }
  .atlas-results {
    display: none;
  }
  .atlas-explorer--results .atlas-results {
    display: flex;
    max-height: 290px;
  }
  .atlas-results__list {
    max-height: 240px;
  }
  .atlas-mobile-close {
    display: grid;
    place-items: center;
    border: 0;
    background: none;
    color: #7d8d6c;
    padding: 5px;
  }
  .atlas-explorer--results .atlas-results__hint {
    display: none;
  }
  .atlas-sports {
    padding: 0 12px 10px;
  }
  .atlas-orientation {
    top: 143px;
    right: 18px;
  }
  .atlas-orientation button {
    padding: 0;
  }
  .atlas-orientation svg {
    width: 27px;
  }
  .atlas-controls {
    right: 13px;
    bottom: 80px;
    gap: 7px;
  }
  .atlas-controls button {
    min-width: 38px;
    min-height: 38px;
    padding: 9px;
  }
  .atlas-map-footer {
    left: 13px;
    right: 13px;
    bottom: 16px;
    flex-wrap: wrap;
    gap: 11px;
  }
  .atlas-legend {
    gap: 12px;
    padding: 8px 10px;
    font-size: 8px;
  }
  .atlas-attribution {
    font-size: 8px;
    width: 100%;
    justify-content: flex-end;
  }
  .atlas-detail {
    left: 12px;
    right: 12px;
    bottom: 60px;
    width: auto;
    display: flex;
    border-radius: 12px;
  }
  .atlas-detail__cover {
    width: 85px;
    flex: none;
    height: auto;
    min-height: 180px;
  }
  .atlas-detail__cover > svg {
    width: 35px;
  }
  .atlas-detail__cover > span {
    left: 12px;
    bottom: 17px;
    font-size: 8px;
    writing-mode: vertical-rl;
    letter-spacing: 3px;
  }
  .atlas-detail__cover > button {
    left: 9px;
    top: 9px;
  }
  .atlas-detail__body {
    padding: 15px 16px;
    flex: 1;
    min-width: 0;
  }
  .atlas-detail h2 {
    font-size: 20px;
    margin-top: 8px;
  }
  .atlas-detail__alias {
    font-size: 9px;
  }
  .atlas-detail__note {
    font-size: 11px;
    margin: 9px 0;
  }
  .atlas-detail__sports {
    padding-top: 1px;
  }
  .atlas-detail__sports > span {
    font-size: 9px;
    padding: 4px 6px;
  }
  .atlas-share {
    font-size: 12px;
    padding: 9px;
  }
  .campus-atlas--selected .atlas-controls {
    bottom: 335px;
  }
  .atlas-map-status {
    padding: 160px 20px 0;
    justify-content: flex-start;
  }
  .atlas-info {
    margin: auto;
    border: 0;
    color: var(--atlas-ink);
    padding: 27px;
  }
}
@media (min-width: 701px) {
  .atlas-detail {
    left: auto;
    top: 26px;
    right: 85px;
    bottom: auto;
    width: min(304px, calc(100% - 420px));
    max-height: calc(100% - 110px);
    overflow: auto;
  }
}
@media (prefers-reduced-motion: reduce) {
  .campus-atlas * {
    animation: none !important;
    transition: none !important;
  }
}
.atlas-campus select {
  border: 0;
  background: transparent;
  color: var(--atlas-ink);
  font: inherit;
  font-weight: 600;
  padding: 8px 20px 8px 4px;
  cursor: pointer;
  max-width: 220px;
}
.atlas-panel-toggle {
  position: absolute;
  z-index: 6;
  top: 32px;
  left: 329px;
  height: 40px;
  min-width: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid var(--atlas-line);
  border-radius: 0 10px 10px 0;
  background: var(--atlas-paper);
  color: var(--atlas-green);
  box-shadow: 3px 3px 12px #253e2910;
}
.campus-atlas--collapsed .atlas-panel-toggle {
  left: 24px;
  border-radius: 12px;
  padding: 0 14px;
}
.atlas-location-notice {
  position: absolute;
  bottom: 58px;
  left: 50%;
  transform: translateX(-25%);
  z-index: 8;
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: calc(100% - 440px);
  padding: 12px 14px;
  background: #fffef9f5;
  border: 1px solid #dbe3db;
  border-radius: 12px;
  box-shadow: 0 5px 20px #273c3012;
  color: #426047;
  font-size: 12px;
}
.atlas-location-notice > svg {
  flex-shrink: 0;
  color: #2779e3;
}
.atlas-location-notice button {
  border: 0;
  background: none;
  display: flex;
  padding: 4px;
}
.campus-atlas--collapsed .atlas-location-notice {
  max-width: calc(100% - 180px);
  transform: translateX(-50%);
}
.atlas-locating svg {
  animation: atlas-spin 2s linear infinite;
}
@media (max-width: 1000px) and (min-width: 701px) {
  .atlas-panel-toggle {
    left: 294px;
  }
}
@media (max-width: 699px) {
  .atlas-panel-toggle {
    top: 8px;
    left: auto;
    right: 12px;
    height: 28px;
    min-width: 34px;
    border-radius: 8px;
  }
  .campus-atlas--collapsed .atlas-panel-toggle {
    top: 16px;
    left: 14px;
    right: auto;
    height: 40px;
  }
  .atlas-campus select {
    max-width: 145px;
    font-size: 12px;
    padding-right: 4px;
  }
  .atlas-location-notice,
  .campus-atlas--collapsed .atlas-location-notice {
    left: 14px;
    right: 70px;
    bottom: 46px;
    max-width: none;
    transform: none;
    font-size: 11px;
  }
  .campus-atlas--selected .atlas-location-notice {
    bottom: 235px;
  }
}
.atlas-plan-note {
  position: absolute;
  bottom: 58px;
  left: 50%;
  transform: translateX(-25%);
  font-size: 11px;
  color: #6f796a;
  max-width: 360px;
  padding: 8px 12px;
  background: #fffef9d9;
  border-radius: 8px;
  pointer-events: none;
}
@media (max-width: 699px) {
  .atlas-plan-note {
    left: 14px;
    right: 75px;
    transform: none;
    bottom: 48px;
    font-size: 10px;
  }
}
@media (min-width: 1400px) {
  .atlas-panel-toggle {
    left: 357px;
    top: 40px;
  }
}
@media (min-width: 701px) {
  .campus-atlas--selected .atlas-panel-toggle {
    left: 329px;
    top: 32px;
  }
}
@media (max-width: 700px) {
  .atlas-search {
    margin-right: 48px;
  }
  .atlas-panel-toggle {
    top: 31px;
    right: 21px;
    left: auto;
  }
  .campus-atlas--collapsed .atlas-panel-toggle {
    top: 16px;
    left: 14px;
    right: auto;
  }
}
</style>
