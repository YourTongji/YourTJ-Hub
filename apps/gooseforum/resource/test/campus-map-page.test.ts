// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import CampusMapPage from '../src/site/pages/CampusMapPage.vue'
import { requestLocation } from '../src/site/campus-map/location'
import { readFileSync } from 'node:fs'

const { focusLocation } = vi.hoisted(() => ({ focusLocation: vi.fn() }))
vi.mock('../src/site/campus-map/location', async (original) => ({
  ...await original<typeof import('../src/site/campus-map/location')>(),
  requestLocation: vi.fn(),
}))
vi.mock('../src/site/campus-map/CampusCanvas.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return { __esModule: true, default: defineComponent({
    name: 'CampusCanvas',
    emits: ['ready', 'failure'],
    setup(_, { expose }) {
      expose({ focusLocation })
      return () => h('div', { class: 'test-canvas' })
    },
  }) }
})
const dataset = JSON.parse(readFileSync('src/site/campus-map/data/siping.geojson', 'utf8'))
const zhangjiangDataset = JSON.parse(readFileSync('src/site/campus-map/data/zhangjiang.geojson', 'utf8'))
const outside = { longitude: 103.85, latitude: 1.29, accuracy: 18, timestamp: 42 }
let wrapper: VueWrapper | undefined
let fetchData: ReturnType<typeof vi.fn>
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
})
async function openPage() {
  wrapper = mount(CampusMapPage, { global: { plugins: [i18n] } })
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
  expect(destination.hostname).toBe('maps.apple.com')
  expect(destination.searchParams.get('daddr')).toBe(
    `${building.properties.center[1]},${building.properties.center[0]}`,
  )
  expect(destination.searchParams.get('q')).toBe(building.properties.name)
  expect(destination.searchParams.get('dirflg')).toBe('w')
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
