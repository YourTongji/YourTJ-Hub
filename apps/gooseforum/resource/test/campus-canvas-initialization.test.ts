// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CampusCanvas from '../src/site/campus-map/CampusCanvas.vue'
import { getCampus } from '../src/site/campus-map/campuses'
import type { CampusData } from '../src/site/campus-map/catalog'

const state = vi.hoisted(() => ({ fail: false, remove: vi.fn(), resize: vi.fn(), on: vi.fn(), fitBounds: vi.fn(), created: vi.fn() }))
vi.mock('maplibre-gl', () => ({
 setWorkerUrl: vi.fn(),
 Map: class {
  constructor() { state.created(); if (state.fail) throw new Error('GPU initialization failed') }
  getCanvas() { return document.createElement('canvas') }
  on = state.on
  remove = state.remove
  resize = state.resize
  fitBounds = state.fitBounds
 },
}))
let observe: ReturnType<typeof vi.fn>
beforeEach(() => {
 vi.clearAllMocks(); state.fail = false
 observe = vi.fn()
 vi.stubGlobal('ResizeObserver', class { observe = observe; disconnect = vi.fn() })
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
test('constructor failure reports failure and never starts resize', () => {
 vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ getExtension: () => null } as unknown as WebGL2RenderingContext)
 state.fail = true
 const wrapper = open()
 expect(wrapper.emitted('failure')).toHaveLength(1)
 expect(observe).not.toHaveBeenCalled()
 wrapper.unmount()
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

test('an initialization error disposes the map and suppresses a late load', () => {
 vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ getExtension: () => null } as unknown as WebGL2RenderingContext)
 const wrapper = open()
 const error = state.on.mock.calls.find(([name]) => name === 'error')![1]
 const load = state.on.mock.calls.find(([name]) => name === 'load')![1]
 error(); load()
 expect(wrapper.emitted('failure')).toHaveLength(1)
 expect(wrapper.emitted('ready')).toBeUndefined()
 expect(state.remove).toHaveBeenCalledOnce()
 wrapper.unmount()
 expect(state.remove).toHaveBeenCalledOnce()
})
