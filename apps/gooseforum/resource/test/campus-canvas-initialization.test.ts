// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { AJAXError } from 'maplibre-gl'
import CampusCanvas from '../src/site/campus-map/CampusCanvas.vue'
import { getCampus } from '../src/site/campus-map/campuses'
import type { CampusData } from '../src/site/campus-map/catalog'

const state = vi.hoisted(() => ({ fail: false, remove: vi.fn(), resize: vi.fn(), on: vi.fn(), fitBounds: vi.fn(), created: vi.fn() }))
vi.mock('maplibre-gl', async (importOriginal) => ({
 AJAXError: (await importOriginal<typeof import('maplibre-gl')>()).AJAXError,
 setWorkerUrl: vi.fn(),
 Map: class {
  constructor(options: { container: HTMLElement }) {
   state.created()
   options.container.classList.add('maplibregl-map')
   options.container.append(document.createElement('canvas'), document.createElement('div'))
   if (state.fail) throw new Error('GPU initialization failed')
  }
  getCanvas() { return document.createElement('canvas') }
  on = state.on
  remove = state.remove
  resize = state.resize
  fitBounds = state.fitBounds
  isStyleLoaded() { return false }
  getSource() { return undefined }
 },
}))
let observe: ReturnType<typeof vi.fn>
let disconnect: ReturnType<typeof vi.fn>
beforeEach(() => {
 vi.clearAllMocks(); state.fail = false
 observe = vi.fn()
 disconnect = vi.fn()
 vi.stubGlobal('ResizeObserver', class { observe = observe; disconnect = disconnect })
})
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })
function open() {
 return mount(CampusCanvas, { props: { campus: getCampus('siping'), panelOpen: false, location: null,
 data: { type: 'FeatureCollection', features: [] } as CampusData, places: [], selected: null, category: 'all', sport: '', dimensional: false, reducedMotion: true } })
}
test('missing WebGL2 fails before creating a map or resize observer', () => {
 vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
 const wrapper = open()
 expect(wrapper.emitted('failure')).toHaveLength(1)
 expect(state.created).not.toHaveBeenCalled()
 expect(observe).not.toHaveBeenCalled()
 wrapper.unmount()
})
test('constructor failure clears partially inserted DOM and never starts resize', () => {
 vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ getExtension: () => null } as unknown as WebGL2RenderingContext)
 state.fail = true
 const wrapper = open()
 expect(wrapper.emitted('failure')).toHaveLength(1)
 expect(observe).not.toHaveBeenCalled()
 const host = wrapper.get('.campus-canvas__map').element
 expect(host.childElementCount).toBe(0)
 expect(host.classList.contains('maplibregl-map')).toBe(false)
 expect(host.classList.contains('campus-canvas__map')).toBe(true)
 expect(state.remove).not.toHaveBeenCalled()
 wrapper.unmount()
})

test('remove failure still clears the map host and does not repeat disposal', () => {
 vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ getExtension: () => null } as unknown as WebGL2RenderingContext)
 state.remove.mockImplementationOnce(() => { throw new Error('Painter was not initialized') })
 const wrapper = open()
 const error = state.on.mock.calls.find(([name]) => name === 'error')![1]
 error({ error: new Error('initialization failed') })
 const host = wrapper.get('.campus-canvas__map').element
 expect(host.childElementCount).toBe(0)
 expect(host.classList.contains('maplibregl-map')).toBe(false)
 expect(host.classList.contains('campus-canvas__map')).toBe(true)
 expect(wrapper.emitted('failure')).toHaveLength(1)
 expect(disconnect).toHaveBeenCalledOnce()
 wrapper.unmount()
 expect(state.remove).toHaveBeenCalledOnce()
})
test('supported map starts normally and removes itself on unmount', () => {
 vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ getExtension: () => null } as unknown as WebGL2RenderingContext)
 const wrapper = open()
 expect(wrapper.emitted('failure')).toBeUndefined()
 expect(observe).toHaveBeenCalledOnce()
 expect(state.on).toHaveBeenCalledWith('load', expect.any(Function))
 wrapper.unmount()
 expect(state.remove).toHaveBeenCalledOnce()
})

test.each([
 { error: new Error('initialization failed') },
 { error: new AJAXError(503, 'Service Unavailable', '/map-data', new Blob()), sourceId: 'buildings' },
])('an initialization error disposes the map and suppresses a late load: %j', (event) => {
 vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ getExtension: () => null } as unknown as WebGL2RenderingContext)
 const wrapper = open()
 const error = state.on.mock.calls.find(([name]) => name === 'error')![1]
 const load = state.on.mock.calls.find(([name]) => name === 'load')![1]
 error(event); load()
 expect(wrapper.emitted('failure')).toHaveLength(1)
 expect(wrapper.emitted('ready')).toBeUndefined()
 expect(state.remove).toHaveBeenCalledOnce()
 wrapper.unmount()
 expect(state.remove).toHaveBeenCalledOnce()
})

test.each([
 { sourceId: 'buildings', tile: { state: 'errored' } },
 { sourceId: 'buildings' },
])('a source HTTP error after load preserves the working map: %j', (metadata) => {
 vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ getExtension: () => null } as unknown as WebGL2RenderingContext)
 const wrapper = open()
 try {
  const load = state.on.mock.calls.find(([name]) => name === 'load')![1]
  const error = state.on.mock.calls.find(([name]) => name === 'error')![1]
  load()
  error({ error: new AJAXError(503, 'Service Unavailable', '/map-data', new Blob()), ...metadata })
  expect(wrapper.emitted('ready')).toHaveLength(1)
  expect(wrapper.emitted('failure')).toBeUndefined()
  expect(state.remove).not.toHaveBeenCalled()
 } finally { wrapper.unmount() }
 expect(state.remove).toHaveBeenCalledOnce()
})

test.each([
 { error: new Error('Style update failed') },
 { error: new Error('Renderer failed') },
 { error: new Error('Worker failed'), sourceId: 'buildings' },
 { error: new Error('Worker failed'), sourceId: 'buildings', tile: { state: 'errored' } },
 { error: new AJAXError(503, 'Service Unavailable', '/style.json', new Blob()) },
])('an unclassified error after load reports failure and disposes once: %j', (event) => {
 vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ getExtension: () => null } as unknown as WebGL2RenderingContext)
 const wrapper = open()
 try {
  const load = state.on.mock.calls.find(([name]) => name === 'load')![1]
  const error = state.on.mock.calls.find(([name]) => name === 'error')![1]
  load()
  error(event)
  expect(wrapper.emitted('failure')).toHaveLength(1)
  error(event)
  load()
  expect(wrapper.emitted('ready')).toHaveLength(1)
  expect(wrapper.emitted('failure')).toHaveLength(1)
  expect(state.remove).toHaveBeenCalledOnce()
  expect(disconnect).toHaveBeenCalledOnce()
 } finally { wrapper.unmount() }
 expect(state.remove).toHaveBeenCalledOnce()
})
