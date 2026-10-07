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

it('restores the loaded For You batch on browser back without fetching a new first page', async () => {
  const { activeForYouSession, updateForYouSession } = await import('../src/runtime/for-you-sessions')
  const committed: PagePayload[] = []
  const router = await setup((page) => committed.push(page.payload))
  const home = { ...payload('/?sort=for_you'), layout: { viewer: { id: 12 } },
    props: { sort: 'for_you', snapshotId: 'original', topics: [{ id: 1 }, { id: 2 }],
      pagination: { page: 1, nextPage: 2, hasNext: true, nextUrl: '/?cursor=original' } } } as unknown as PagePayload
  mocks.fetch.mockResolvedValueOnce(home)
  await router.push('/?sort=for_you')
  expect(activeForYouSession.value).not.toBe('')
  updateForYouSession({ ...home.props, topics: [{ id: 1 }, { id: 2 }, { id: 3 }],
    pagination: { page: 2, nextPage: 3, hasNext: true, nextUrl: '/?cursor=page-three' } } as never)
  mocks.fetch.mockResolvedValueOnce({ ...payload('/p/post/2'), component: 'topic.detail', layout: home.layout })
  await router.push('/p/post/2')
  mocks.fetch.mockClear()
  router.back()
  await vi.waitFor(() => expect(router.currentRoute.value.fullPath).toBe('/?sort=for_you'))
  await vi.waitFor(() => expect(committed.at(-1)?.url).toBe('/?sort=for_you'))
  expect(mocks.fetch).not.toHaveBeenCalled()
  const restored = committed.at(-1)!.props as { topics: { id: number }[]; pagination: { nextUrl: string } }
  expect(restored.topics.map((t) => t.id)).toEqual([1, 2, 3])
  expect(restored.pagination.nextUrl).toContain('page-three')
})

it('a late page response cannot switch the account back to its previous owner', async () => {
  const { fetchPage } = await import('../src/runtime/router')
  const { feedAccount, resetFeedAccount } = await import('../src/runtime/feed-telemetry')
  resetFeedAccount(12)
  let resolve!: (page: PagePayload) => void
  mocks.fetch.mockImplementation(() => new Promise((r) => { resolve = r }))
  const result = fetchPage(new URL('http://localhost/?sort=for_you&cursor=old'))
  resetFeedAccount(13)
  resolve({ ...payload('/?sort=for_you'), layout: { viewer: { id: 12 } } } as PagePayload)
  await expect(result).rejects.toThrow()
  expect(feedAccount()).toBe(13)
  resetFeedAccount(0)
})
