// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest'
import { createApp, defineComponent, h } from 'vue'
import type { PagePayload } from '@gooseforum/client'
import { homeFeedNavigation } from '../src/runtime/home-feed-navigation'
import { installNavigation } from '../src/runtime/router'

const mocks = vi.hoisted(() => ({
  fetch: vi.fn(),
  setNavigating: vi.fn(),
}))

vi.mock('../src/runtime/navigation-state', () => ({
  useNavigationState: () => ({ setNavigating: mocks.setNavigating }),
}))
vi.mock('../src/runtime/page-registry', () => ({ resolvePageComponent: vi.fn(async () => ({})) }))
vi.mock('@gooseforum/client', () => ({ createGooseClient: () => ({ pages: { fetch: mocks.fetch } }) }))

const component = defineComponent({ setup: () => () => h('div') })
let app: ReturnType<typeof createApp> | undefined

function payload(url: string, marker = url): PagePayload {
  return { component: 'home.index', url, props: { marker } } as PagePayload
}

async function setup(onPage: (page: { payload: PagePayload }) => void) {
  window.history.replaceState({}, '', '/')
  const router = installNavigation({ payload: payload('/'), component }, component, onPage as never)
  app = createApp(component)
  app.use(router)
  await router.isReady()
  app.mount(document.createElement('div'))
  mocks.setNavigating.mockClear()
  return router
}

afterEach(() => {
  app?.unmount()
  app = undefined
  homeFeedNavigation.cancel()
  mocks.fetch.mockReset()
  mocks.setNavigating.mockReset()
  vi.restoreAllMocks()
})

it('commits only the last home feed response and skips the shell navigation indicator', async () => {
  const committed: string[] = []
  const router = await setup((page) => committed.push(page.payload.url))
  committed.length = 0

  const responses = new Map<string, (page: PagePayload) => void>()
  mocks.fetch.mockImplementation((url: URL) => new Promise((resolve) => {
    responses.set(`${url.pathname}${url.search}`, resolve)
  }))

  const hotNavigation = router.push('/?sort=hot')
  await vi.waitFor(() => expect(responses.has('/?sort=hot')).toBe(true))
  const popularNavigation = router.push('/?sort=popular')
  await vi.waitFor(() => expect(responses.size).toBe(2))

  responses.get('/?sort=popular')!(payload('/?sort=popular'))
  await popularNavigation
  responses.get('/?sort=hot')!(payload('/?sort=hot'))
  await hotNavigation

  expect(committed).toEqual(['/?sort=popular'])
  expect(router.currentRoute.value.fullPath).toBe('/?sort=popular')
  expect(homeFeedNavigation.pendingUrl.value).toBeNull()
  expect(mocks.setNavigating).not.toHaveBeenCalled()
})

it('keeps the current feed and saves the failed destination for retry', async () => {
  const committed: string[] = []
  const router = await setup((page) => committed.push(page.payload.url))
  committed.length = 0
  mocks.fetch.mockRejectedValueOnce(new Error('offline'))

  await router.push('/?sort=following')

  expect(router.currentRoute.value.fullPath).toBe('/')
  expect(committed).toEqual([])
  expect(homeFeedNavigation.failedUrl.value).toBe('/?sort=following')
  expect(mocks.setNavigating).not.toHaveBeenCalled()
})

it('loads home feed payloads for browser back and forward navigation', async () => {
  const committed: string[] = []
  const router = await setup((page) => committed.push(page.payload.url))
  committed.length = 0
  mocks.fetch.mockImplementation(async (url: URL) => payload(`${url.pathname}${url.search}`))

  await router.push('/?sort=hot')
  router.back()
  await vi.waitFor(() => expect(router.currentRoute.value.fullPath).toBe('/'))
  await vi.waitFor(() => expect(committed.at(-1)).toBe('/'))
  router.forward()
  await vi.waitFor(() => expect(router.currentRoute.value.fullPath).toBe('/?sort=hot'))
  await vi.waitFor(() => expect(committed.at(-1)).toBe('/?sort=hot'))

  expect(committed).toEqual(['/?sort=hot', '/', '/?sort=hot'])
  expect(mocks.setNavigating).not.toHaveBeenCalled()
})

it('does not let an earlier duplicate destination overwrite the latest response', async () => {
  const committed: string[] = []
  const router = await setup((page) => committed.push(String(page.payload.props?.marker)))
  committed.length = 0
  const responses: ((page: PagePayload) => void)[] = []
  mocks.fetch.mockImplementation(() => new Promise((resolve) => responses.push(resolve)))

  const firstNavigation = router.push('/?sort=hot')
  await vi.waitFor(() => expect(responses).toHaveLength(1))
  const secondNavigation = router.push('/?sort=hot')
  await vi.waitFor(() => expect(responses).toHaveLength(2))
  responses[0]!(payload('/?sort=hot', 'stale'))
  await firstNavigation
  responses[1]!(payload('/?sort=hot', 'current'))
  await secondNavigation

  expect(committed).toEqual(['current'])
  expect(router.currentRoute.value.fullPath).toBe('/?sort=hot')
})
