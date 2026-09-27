// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest'
import { createApp, defineComponent, h } from 'vue'
import type { PagePayload } from '@gooseforum/client'
import { installNavigation } from '../src/runtime/router'

vi.mock('../src/runtime/navigation-state', () => ({ useNavigationState: () => ({ setNavigating: vi.fn() }) }))
vi.mock('../src/runtime/page-registry', () => ({ resolvePageComponent: vi.fn() }))
vi.mock('@gooseforum/client', () => ({ createGooseClient: () => ({ pages: { fetch: vi.fn() } }) }))

const component = defineComponent({ setup: () => () => h('div') })
let app: ReturnType<typeof createApp> | undefined
afterEach(() => {
  app?.unmount()
  app = undefined
  vi.restoreAllMocks()
})

it.each([
  ['/map', '/map?mine=1'],
  ['/map?mine=1', '/map'],
])('reloads the document when changing map privacy mode: %s → %s', async (from, to) => {
  window.history.replaceState({}, '', from)
  const assign = vi.spyOn(window.location, 'assign').mockImplementation(() => {})
  const router = installNavigation(
    { payload: {} as PagePayload, component },
    component,
    () => {},
  )
  app = createApp(component)
  app.use(router)
  await router.isReady()

  await router.push(to)

  expect(assign).toHaveBeenCalledWith(to)
})
