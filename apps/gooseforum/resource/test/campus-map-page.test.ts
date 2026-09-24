// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import CampusMapPage from '../src/site/pages/CampusMapPage.vue'
import { requestLocation } from '../src/site/campus-map/location'
import { readFileSync } from 'node:fs'
import type { CampusDatasetKey, CampusStatus, LayoutPayload } from '@gooseforum/client'

const { focusLocation } = vi.hoisted(() => ({ focusLocation: vi.fn() }))
const campusApi = vi.hoisted(() => ({ status: vi.fn(), dataset: vi.fn() }))
vi.mock('../src/runtime/campus-api', async (original) => ({
  ...await original<typeof import('../src/runtime/campus-api')>(),
  campusAPI: campusApi,
}))
vi.mock('../src/site/campus-map/location', async (original) => ({
  ...await original<typeof import('../src/site/campus-map/location')>(),
  requestLocation: vi.fn(),
}))
vi.mock('../src/site/campus-map/CampusCanvas.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return { __esModule: true, default: defineComponent({
    name: 'CampusCanvas',
    props: ['selected'],
    emits: ['ready', 'failure'],
    setup(_, { expose }) {
      expose({ focusLocation })
      return () => h('div', { class: 'test-canvas' })
    },
  }) }
})
const dataset = JSON.parse(readFileSync('src/site/campus-map/data/siping.geojson', 'utf8'))
const jiading = JSON.parse(readFileSync('src/site/campus-map/data/jiading.geojson', 'utf8'))
const zhangjiangDataset = JSON.parse(readFileSync('src/site/campus-map/data/zhangjiang.geojson', 'utf8'))
const outside = { longitude: 103.85, latitude: 1.29, accuracy: 18, timestamp: 42 }
const shanghaiDateParts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date())
const shanghaiDate = `${shanghaiDateParts.find(part => part.type === 'year')?.value}-${shanghaiDateParts.find(part => part.type === 'month')?.value}-${shanghaiDateParts.find(part => part.type === 'day')?.value}`
let wrapper: VueWrapper | undefined
let fetchData: ReturnType<typeof vi.fn>
const originalClipboard = Object.getOwnPropertyDescriptor(navigator, 'clipboard')
beforeEach(async () => {
  vi.clearAllMocks()
  window.history.replaceState({}, '', '/map')
  await setLocale('en')
  fetchData = vi.fn().mockResolvedValue({ ok: true, json: async () => dataset })
  vi.stubGlobal('fetch', fetchData)
  vi.mocked(requestLocation).mockResolvedValue(outside)
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  i18n.global.locale.value = 'zh'
  vi.unstubAllGlobals()
  if (originalClipboard) Object.defineProperty(navigator, 'clipboard', originalClipboard)
  else Reflect.deleteProperty(navigator, 'clipboard')
})
async function openPage() {
  wrapper = mount(CampusMapPage, { props: { layout: { viewer: { id: 42, isAuthenticated: true } } as LayoutPayload }, global: { plugins: [i18n] } })
  await flushPromises()
  return wrapper
}
it.each([['en', 'Basketball'], ['de', 'Basketball'], ['ja', 'バスケットボール']])(
  'renders sport controls and searchable labels in %s', async (locale, basketball) => {
    await setLocale(locale as 'en' | 'de' | 'ja')
    const page = await openPage()
    await page.get('.atlas-categories button:nth-child(2)').trigger('click')
    expect(page.get('.atlas-sports').text()).toContain(basketball)
    await page.get('.atlas-search input').setValue(basketball)
    expect(page.findAll('.atlas-place').length).toBeGreaterThan(0)
    expect(page.get('.atlas-place small').text()).toContain(basketball)
  },
)
it('shows an external map route for the selected building and keeps the map open', async () => {
  const building = dataset.features.find((feature) => feature.properties.building)
  window.history.replaceState(
    {},
    '',
    `/map#place=${encodeURIComponent(String(building.id))}`,
  )
  const page = await openPage()
  const link = page.get('.atlas-detail a.atlas-share')
  const destination = new URL(link.attributes('href')!)
  expect(destination.hostname).toBe('api.map.baidu.com')
  expect(destination.searchParams.get('destination')).toBe(
    `latlng:${building.properties.center[1]},${building.properties.center[0]}|name:${building.properties.name}`,
  )
  expect(destination.searchParams.get('coord_type')).toBe('wgs84')
  expect(destination.searchParams.get('mode')).toBe('walking')
  expect(link.attributes('target')).toBe('_blank')
  expect(link.text()).toBe('Navigate')
  expect(page.get('.atlas-detail h2').exists()).toBe(true)
})
it('does not offer navigation for Zhangjiang schematic buildings', async () => {
  const building = zhangjiangDataset.features.find((feature) => feature.properties.building)
  window.history.replaceState(
    {},
    '',
    `/map?campus=zhangjiang#place=${encodeURIComponent(String(building.id))}`,
  )
  fetchData.mockResolvedValueOnce({ ok: true, json: async () => zhangjiangDataset })
  const page = await openPage()
  expect(page.get('.atlas-detail h2').exists()).toBe(true)
  expect(page.find('.atlas-detail a.atlas-share').exists()).toBe(false)
})
it('reports an outside-campus fix from the uncalibrated plan without promising an overlay', async () => {
  window.history.replaceState({}, '', '/map?campus=zhangjiang')
  const page = await openPage()
  await page.get('[aria-label="Show my location"]').trigger('click')
  await flushPromises()
  expect(page.get('.atlas-location-notice').text()).toContain('outside')
  expect(page.get('.atlas-location-notice').text()).toContain('schematic')
  expect(focusLocation).not.toHaveBeenCalled()
})
it.each([false, true])('focuses a cached fix after readiness (data already loaded: %s)', async (loaded) => {
  let resolveData!: (response: unknown) => void
  if (!loaded) fetchData.mockReturnValueOnce(new Promise((resolve) => { resolveData = resolve }))
  const page = await openPage()
  await page.get('[aria-label="Show my location"]').trigger('click')
  await flushPromises()
  expect(focusLocation).not.toHaveBeenCalled()
  if (!loaded) {
    resolveData({ ok: true, json: async () => dataset })
    await flushPromises()
  }
  page.findComponent({ name: 'CampusCanvas' }).vm.$emit('ready')
  await flushPromises()
  expect(focusLocation).toHaveBeenCalledOnce()
})
it('distinguishes unavailable data from a renderer failure and retries loading', async () => {
  fetchData.mockRejectedValueOnce(new Error('offline'))
  const page = await openPage()
  expect(page.text()).toContain('Place data could not be loaded')
  expect(page.text()).not.toContain('You can still browse the place list')
  await page.get('.atlas-map-status button').trigger('click')
  await flushPromises()
  page.findComponent({ name: 'CampusCanvas' }).vm.$emit('failure')
  await flushPromises()
  expect(page.text()).toContain('You can still browse the place list')
  expect(page.text()).not.toContain('Place data could not be loaded')
})

it('shows clearly labeled local timetable samples and maps their reviewed buildings', async () => {
  window.history.replaceState({}, '', '/map?mine=1&demo=1')
  const page = await openPage()
  await vi.waitFor(() => expect(page.get('.atlas-mine').text()).toContain('示例数据：仅用于本地界面预览，不是本人课表。'))
  expect(page.findAll('.atlas-mine__course')).toHaveLength(4)
  expect(campusApi.status).not.toHaveBeenCalled()

  await page.findAll('.atlas-mine__course')[0]!.trigger('click')
  await flushPromises()
  expect(window.location.hash).toBe('#place=way%2F183383474')
})

it('locates a confirmed timetable building using only the generic map feature id', async () => {
  window.history.replaceState({}, '', '/map?mine=1')
  campusApi.status.mockResolvedValue({
    enabled: true,
    binding: { maskedId: '***01', revision: 'revision-1', needsAuthorization: false },
    candidate: null,
  } satisfies CampusStatus)
  const event = { name: '数学分析', room: '北115', campus: '四平路校区', day: 2, start: 3, end: 4, weeks: [3], teacher: '', credits: '' }
  campusApi.dataset.mockImplementation(async (key: CampusDatasetKey) => ({
    key,
    status: 'ready',
    updatedAt: '2026-09-23T00:00:00Z',
    metrics: key === 'calendar' ? [{ label: '教学周', value: '3', unit: '周' }, { label: '学期周数', value: '18', unit: '周' }] : [],
    columns: [], rows: [], series: [],
    events: key === 'today' || key === 'timetable' ? [event] : [],
    ...(key === 'today' ? { teachingDay: { date: shanghaiDate, sourceDate: shanghaiDate, kind: 'none', label: '', sectionCount: 11 } } : {}),
  }))

  const page = await openPage()
  expect(page.text()).toContain('Your official Tongji timetable')
  expect(campusApi.dataset.mock.calls.map(([key]) => key).sort()).toEqual(['calendar', 'timetable', 'today'])
  await page.get('.atlas-mine__course').trigger('click')
  await flushPromises()
  expect(page.get('.atlas-mine__selected').text()).toContain('四平路校区 · 北 · 115')
  expect(page.get('.atlas-mine__selected').text()).not.toContain('Building location could not be verified')
  expect(page.findComponent({ name: 'CampusCanvas' }).props('selected').id).toBe('way/183383474')
  expect(window.location.hash).toBe('#place=way%2F183383474')
  expect(page.find('.atlas-place').exists()).toBe(false)
  expect(page.find('[aria-label="Show my location"]').exists()).toBe(true)
  expect(window.location.search).toBe('?mine=1')
  expect(window.location.href).not.toContain('数学分析')
  expect(window.location.href).not.toContain('北115')
})

it('waits for the current campus map data when a course is clicked first', async () => {
  window.history.replaceState({}, '', '/map?mine=1')
  campusApi.status.mockResolvedValue({
    enabled: true,
    binding: { maskedId: '***01', revision: 'revision-1', needsAuthorization: false },
    candidate: null,
  } satisfies CampusStatus)
  const event = { name: '数学分析', room: '北115', campus: '四平路校区', day: 2, start: 3, end: 4, weeks: [3], teacher: '', credits: '' }
  campusApi.dataset.mockImplementation(async (key: CampusDatasetKey) => ({
    key, status: 'ready', updatedAt: '',
    metrics: key === 'calendar' ? [{ label: '教学周', value: '3', unit: '周' }, { label: '学期周数', value: '18', unit: '周' }] : [],
    columns: [], rows: [], series: [], events: key === 'today' ? [event] : [],
    ...(key === 'today' ? { teachingDay: { date: shanghaiDate, sourceDate: shanghaiDate, kind: 'none', label: '', sectionCount: 11 } } : {}),
  }))
  let resolveMap!: (response: unknown) => void
  fetchData.mockReturnValueOnce(new Promise(resolve => { resolveMap = resolve }))

  const page = await openPage()
  await page.get('.atlas-mine__course').trigger('click')
  expect(page.get('.atlas-mine__course').attributes('aria-pressed')).toBe('true')
  expect(window.location.hash).toBe('')

  resolveMap({ ok: true, json: async () => dataset })
  await flushPromises()
  expect(page.findComponent({ name: 'CampusCanvas' }).props('selected').id).toBe('way/183383474')
  expect(window.location.hash).toBe('#place=way%2F183383474')
  expect(window.location.href).not.toContain('数学分析')
  expect(window.location.href).not.toContain('北115')
})

it('keeps an unconfirmed campus/building combination unpinned', async () => {
  window.history.replaceState({}, '', '/map?mine=1')
  campusApi.status.mockResolvedValue({
    enabled: true,
    binding: { maskedId: '***01', revision: 'revision-1', needsAuthorization: false },
    candidate: null,
  } satisfies CampusStatus)
  const event = { name: '数学分析', room: '北115', campus: '嘉定校区', day: 2, start: 3, end: 4, weeks: [3], teacher: '', credits: '' }
  campusApi.dataset.mockImplementation(async (key: CampusDatasetKey) => ({
    key, status: 'ready', updatedAt: '', metrics: [], columns: [], rows: [], series: [],
    events: key === 'today' ? [event] : [],
    ...(key === 'today' ? { teachingDay: { date: shanghaiDate, sourceDate: shanghaiDate, kind: 'none', label: '', sectionCount: 11 } } : {}),
  }))

  const page = await openPage()
  await page.get('.atlas-mine__course').trigger('click')
  await flushPromises()
  expect(page.get('.atlas-mine__selected').text()).toContain('Building location could not be verified')
  expect(page.findComponent({ name: 'CampusCanvas' }).props('selected')).toBeNull()
  expect(window.location.hash).toBe('')
})

it('switches to Jiading and selects its single Jishi Building feature', async () => {
  window.history.replaceState({}, '', '/map?mine=1')
  campusApi.status.mockResolvedValue({
    enabled: true,
    binding: { maskedId: '***01', revision: 'revision-1', needsAuthorization: false },
    candidate: null,
  } satisfies CampusStatus)
  const event = { name: '线性代数', room: '济事北楼 A101', campus: '嘉定校区', day: 2, start: 1, end: 2, weeks: [3], teacher: '', credits: '' }
  campusApi.dataset.mockImplementation(async (key: CampusDatasetKey) => ({
    key, status: 'ready', updatedAt: '', metrics: [], columns: [], rows: [], series: [],
    events: key === 'today' ? [event] : [],
    ...(key === 'today' ? { teachingDay: { date: shanghaiDate, sourceDate: shanghaiDate, kind: 'none', label: '', sectionCount: 11 } } : {}),
  }))
  const building = jiading.features.find((feature: { id: string }) => feature.id === 'way/135405205')
  fetchData.mockResolvedValue({
    ok: true,
    json: async () => ({ ...dataset, features: [...dataset.features, building] }),
  })

  const page = await openPage()
  await page.get('.atlas-mine__course').trigger('click')
  await flushPromises()

  expect(page.get('.atlas-campus select').element).toHaveProperty('value', 'jiading')
  expect(page.findComponent({ name: 'CampusCanvas' }).props('selected').id).toBe('way/135405205')
  expect(window.location.search).toBe('?mine=1&campus=jiading')
  expect(window.location.hash).toBe('#place=way%2F135405205')
  expect(window.location.href).not.toContain('线性代数')
  expect(window.location.href).not.toContain('A101')
})

it('ignores an older campus switch when a later course is selected', async () => {
  window.history.replaceState({}, '', '/map?mine=1')
  campusApi.status.mockResolvedValue({
    enabled: true,
    binding: { maskedId: '***01', revision: 'revision-1', needsAuthorization: false },
    candidate: null,
  } satisfies CampusStatus)
  const events = [
    { name: '济事课程', room: '济事北楼 A101', campus: '嘉定校区', day: 2, start: 1, end: 2, weeks: [3], teacher: '', credits: '' },
    { name: '南楼课程', room: '南楼203', campus: '四平路校区', day: 2, start: 3, end: 4, weeks: [3], teacher: '', credits: '' },
  ]
  campusApi.dataset.mockImplementation(async (key: CampusDatasetKey) => ({
    key, status: 'ready', updatedAt: '',
    metrics: key === 'calendar' ? [{ label: '教学周', value: '3', unit: '周' }, { label: '学期周数', value: '18', unit: '周' }] : [],
    columns: [], rows: [], series: [], events: key === 'today' || key === 'timetable' ? events : [],
    ...(key === 'today' ? { teachingDay: { date: shanghaiDate, sourceDate: shanghaiDate, kind: 'none', label: '', sectionCount: 11 } } : {}),
  }))
  const page = await openPage()
  let resolveJiading!: (response: unknown) => void
  fetchData.mockImplementationOnce(() => new Promise(resolve => { resolveJiading = resolve }))
  fetchData.mockResolvedValueOnce({ ok: true, json: async () => dataset })

  await page.findAll('.atlas-mine__course')[0]!.trigger('click')
  await page.vm.$nextTick()
  await page.findAll('.atlas-mine__course')[1]!.trigger('click')
  await flushPromises()
  resolveJiading({ ok: true, json: async () => jiading })
  await flushPromises()

  expect(page.get('.atlas-campus select').element).toHaveProperty('value', 'siping')
  expect(page.findComponent({ name: 'CampusCanvas' }).props('selected').id).toBe('way/183383472')
  expect(window.location.hash).toBe('#place=way%2F183383472')
  expect(window.location.href).not.toContain('济事课程')
  expect(window.location.href).not.toContain('A101')
})

it('clears the mapped feature when the course scope changes or private data clears', async () => {
  window.history.replaceState({}, '', '/map?mine=1')
  campusApi.status.mockResolvedValue({
    enabled: true,
    binding: { maskedId: '***01', revision: 'revision-1', needsAuthorization: false },
    candidate: null,
  } satisfies CampusStatus)
  const event = { name: '数学分析', room: '北115', campus: '四平路校区', day: 2, start: 3, end: 4, weeks: [3], teacher: '', credits: '' }
  campusApi.dataset.mockImplementation(async (key: CampusDatasetKey) => ({
    key, status: 'ready', updatedAt: '',
    metrics: key === 'calendar' ? [{ label: '教学周', value: '3', unit: '周' }, { label: '学期周数', value: '18', unit: '周' }] : [],
    columns: [], rows: [], series: [], events: key === 'today' || key === 'timetable' ? [event] : [],
    ...(key === 'today' ? { teachingDay: { date: shanghaiDate, sourceDate: shanghaiDate, kind: 'none', label: '', sectionCount: 11 } } : {}),
  }))
  const page = await openPage()
  await page.get('.atlas-mine__course').trigger('click')
  await flushPromises()
  expect(window.location.hash).toBe('#place=way%2F183383474')

  await page.get('.atlas-mine__tabs button:nth-child(2)').trigger('click')
  expect(window.location.hash).toBe('')
  expect(page.findComponent({ name: 'CampusCanvas' }).props('selected')).toBeNull()
  await page.get('.atlas-mine__course').trigger('click')
  await flushPromises()
  expect(window.location.hash).toBe('#place=way%2F183383474')

  window.dispatchEvent(new Event('pagehide'))
  await flushPromises()
  expect(window.location.hash).toBe('')
  expect(page.findComponent({ name: 'CampusCanvas' }).props('selected')).toBeNull()
})

it('shares the public atlas URL from private map mode', async () => {
  window.history.replaceState({}, '', '/map?campus=siping&mine=1#place=relation%2F18788114')
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText: vi.fn().mockRejectedValue(new Error('clipboard unavailable')) },
  })
  const page = await openPage()
  await page.get('button.atlas-share').trigger('click')
  await flushPromises()

  expect(page.get('.atlas-share-url').element).toHaveProperty('value', 'http://localhost:3000/map?campus=siping#place=relation%2F18788114')
})

it('clears the private course list when revalidation fails and aborts requests on exit', async () => {
  window.history.replaceState({}, '', '/map?mine=1')
  campusApi.status.mockResolvedValue({
    enabled: true,
    binding: { maskedId: '***01', revision: 'revision-1', needsAuthorization: false },
    candidate: null,
  } satisfies CampusStatus)
  const event = { name: '线性代数', room: '济事楼201', campus: '嘉定校区', day: 2, start: 1, end: 2, weeks: [3], teacher: '', credits: '' }
  campusApi.dataset.mockImplementation(async (key: CampusDatasetKey) => ({
    key, status: 'ready', updatedAt: '', metrics: [], columns: [], rows: [], series: [],
    events: key === 'today' ? [event] : [],
    ...(key === 'today' ? { teachingDay: { date: shanghaiDate, sourceDate: shanghaiDate, kind: 'none', label: '', sectionCount: 11 } } : {}),
  }))
  const page = await openPage()
  expect(page.text()).toContain('线性代数')
  campusApi.status.mockRejectedValue(new Error('session expired'))
  window.dispatchEvent(new Event('focus'))
  await flushPromises()
  expect(page.text()).not.toContain('线性代数')
  expect(page.text()).toContain('Your timetable is temporarily unavailable')
  const signals = campusApi.dataset.mock.calls.map(([, signal]) => signal as AbortSignal)
  page.unmount()
  expect(signals.every(signal => signal.aborted)).toBe(true)
})

it('clears the timetable immediately on session clear and ignores late responses', async () => {
  window.history.replaceState({}, '', '/map?mine=1')
  campusApi.status.mockResolvedValue({
    enabled: true,
    binding: { maskedId: '***01', revision: 'revision-1', needsAuthorization: false },
    candidate: null,
  } satisfies CampusStatus)
  let resolveToday!: (value: unknown) => void
  campusApi.dataset.mockImplementation((key: CampusDatasetKey) => key === 'today'
    ? new Promise(resolve => { resolveToday = resolve })
    : Promise.resolve({ key, status: 'ready', updatedAt: '', metrics: [], columns: [], rows: [], series: [], events: [] }))
  const page = await openPage()
  await flushPromises()
  window.dispatchEvent(new Event('goose:session-cleared'))
  await page.vm.$nextTick()
  expect(page.text()).toContain('Sign in to view your timetable.')
  resolveToday({ key: 'today', status: 'ready', updatedAt: '', metrics: [], columns: [], rows: [], series: [], events: [{ name: 'private late result' }] })
  await flushPromises()
  expect(page.text()).not.toContain('private late result')
  expect(page.text()).toContain('Sign in to view your timetable.')
})

it('discards timetable responses when the official binding revision changes mid-read', async () => {
  window.history.replaceState({}, '', '/map?mine=1')
  campusApi.status
    .mockResolvedValueOnce({ enabled: true, binding: { maskedId: '***01', revision: 'revision-1', needsAuthorization: false }, candidate: null })
    .mockResolvedValueOnce({ enabled: true, binding: { maskedId: '***02', revision: 'revision-2', needsAuthorization: false }, candidate: null })
  campusApi.dataset.mockImplementation(async (key: CampusDatasetKey) => ({
    key, status: 'ready', updatedAt: '',
    metrics: key === 'calendar' ? [{ label: '教学周', value: '3', unit: '周' }, { label: '学期周数', value: '18', unit: '周' }] : [],
    columns: [], rows: [], series: [],
    events: key === 'today' ? [{ name: 'must not cross bindings', room: '北115', campus: '四平路校区', day: 2, start: 1, end: 2, weeks: [3], teacher: '', credits: '' }] : [],
    ...(key === 'today' ? { teachingDay: { date: shanghaiDate, sourceDate: shanghaiDate, kind: 'none', label: '', sectionCount: 11 } } : {}),
  }))

  const page = await openPage()
  expect(page.text()).not.toContain('must not cross bindings')
  expect(page.text()).toContain('Your timetable is temporarily unavailable')
  expect(campusApi.status).toHaveBeenCalledTimes(2)
})
