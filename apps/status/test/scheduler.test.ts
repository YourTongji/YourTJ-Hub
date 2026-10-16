import { afterEach, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import scheduler, { stuckRunListTest } from '../worker/scheduler'

afterEach(() => {
  vi.unstubAllGlobals()
  stuckRunListTest.perPage = 100
})

const dispatchUrl = 'https://api.github.com/repos/YourTongji/YourTJ-Hub/actions/workflows/collect-status.yml/dispatches'
const listPrefix = 'https://api.github.com/repos/YourTongji/YourTJ-Hub/actions/workflows/collect-status.yml/runs?'
const pendingUrl = (id: number) => `https://api.github.com/repos/YourTongji/YourTJ-Hub/actions/runs/${id}/pending_deployments`
const cancelUrl = (id: number) => `https://api.github.com/repos/YourTongji/YourTJ-Hub/actions/runs/${id}/force-cancel`

function isListUrl(url: unknown): url is string {
  return typeof url === 'string' && url.startsWith(listPrefix)
}

function stubFetchSequence(...responses: Response[]) {
  const fetcher = vi.fn()
  for (const response of responses) fetcher.mockResolvedValueOnce(response)
  vi.stubGlobal('fetch', fetcher)
  return fetcher
}

function emptyList() {
  return new Response(JSON.stringify({ workflow_runs: [] }), { status: 200 })
}

function agedIso(minutesAgo: number) {
  return new Date(Date.now() - minutesAgo * 60 * 1000).toISOString()
}

it.each([
  ['2,17,32,47 * * * *', 'public'],
  ['7 * * * *', 'devices'],
])('dispatches %s to the fixed main collector without GitHub schedule events', async (cron, kind) => {
  const fetcher = stubFetchSequence(emptyList(), new Response(null, { status: 204 }))
  await scheduler.scheduled({ cron }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' })
  expect(fetcher).toHaveBeenCalledTimes(2)
  expect(isListUrl(fetcher.mock.calls[0]![0])).toBe(true)
  expect(fetcher.mock.calls[0]![0]).toContain('status=waiting')
  expect(fetcher.mock.calls[0]![0]).toContain('branch=main')
  expect(fetcher.mock.calls[0]![0]).toContain('per_page=100')
  expect(fetcher.mock.calls[0]![0]).toContain('page=1')
  expect(fetcher.mock.calls[0]![0]).toContain('created=<')
  expect(fetcher.mock.calls[0]![1].method).toBe('GET')
  const [url, init] = fetcher.mock.calls[1]!
  expect(url).toBe(dispatchUrl)
  expect(init.method).toBe('POST')
  expect(init.redirect).toBe('manual')
  expect(init.signal).toBeInstanceOf(AbortSignal)
  expect(init.headers.Authorization).toBe('Bearer test-only-token')
  expect(JSON.parse(init.body)).toEqual({ ref: 'main', inputs: { kind } })
})

it('force-cancels aged waiting runs when pending_deployments are empty or unprotected', async () => {
  const old = agedIso(21)
  const fresh = agedIso(5)
  const fetcher = stubFetchSequence(
    new Response(JSON.stringify({
      workflow_runs: [
        { id: 11, head_branch: 'main', status: 'waiting', created_at: old },
        { id: 12, head_branch: 'main', status: 'waiting', created_at: fresh },
        { id: 13, head_branch: 'dev', status: 'waiting', created_at: old },
      ],
    }), { status: 200 }),
    new Response(JSON.stringify([]), { status: 200 }),
    new Response(null, { status: 202 }),
    new Response(null, { status: 204 }),
  )
  await scheduler.scheduled({ cron: '7 * * * *' }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' })
  expect(fetcher.mock.calls.map(call => [call[0], call[1].method])).toEqual([
    [expect.stringMatching(/^https:\/\/api\.github\.com\/.*\/runs\?/), 'GET'],
    [pendingUrl(11), 'GET'],
    [cancelUrl(11), 'POST'],
    [dispatchUrl, 'POST'],
  ])
})

it('force-cancels when every pending deployment has empty reviewers and wait_timer 0', async () => {
  const old = agedIso(25)
  const fetcher = stubFetchSequence(
    new Response(JSON.stringify({
      workflow_runs: [{ id: 21, head_branch: 'main', status: 'waiting', created_at: old }],
    }), { status: 200 }),
    new Response(JSON.stringify([
      { environment: { name: 'status-collector' }, wait_timer: 0, reviewers: [] },
    ]), { status: 200 }),
    new Response(null, { status: 202 }),
    new Response(null, { status: 204 }),
  )
  await scheduler.scheduled({ cron: '7 * * * *' }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' })
  expect(fetcher.mock.calls.map(call => call[0])).toEqual([
    expect.stringMatching(/\/runs\?/),
    pendingUrl(21),
    cancelUrl(21),
    dispatchUrl,
  ])
})

it('skips force-cancel when pending deployment has reviewers', async () => {
  const old = agedIso(25)
  const fetcher = stubFetchSequence(
    new Response(JSON.stringify({
      workflow_runs: [{ id: 31, head_branch: 'main', status: 'waiting', created_at: old }],
    }), { status: 200 }),
    new Response(JSON.stringify([
      { environment: { name: 'status-collector' }, wait_timer: 0, reviewers: [{ type: 'User', reviewer: { login: 'alice' } }] },
    ]), { status: 200 }),
    new Response(null, { status: 204 }),
  )
  await scheduler.scheduled({ cron: '7 * * * *' }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' })
  expect(fetcher.mock.calls.map(call => [call[0], call[1].method])).toEqual([
    [expect.stringMatching(/\/runs\?/), 'GET'],
    [pendingUrl(31), 'GET'],
    [dispatchUrl, 'POST'],
  ])
})

it('skips force-cancel when pending deployment wait_timer is greater than zero', async () => {
  const old = agedIso(25)
  const fetcher = stubFetchSequence(
    new Response(JSON.stringify({
      workflow_runs: [{ id: 41, head_branch: 'main', status: 'waiting', created_at: old }],
    }), { status: 200 }),
    new Response(JSON.stringify([
      { environment: { name: 'status-collector' }, wait_timer: 30, reviewers: [] },
    ]), { status: 200 }),
    new Response(null, { status: 204 }),
  )
  await scheduler.scheduled({ cron: '7 * * * *' }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' })
  expect(fetcher.mock.calls.map(call => call[0])).toEqual([
    expect.stringMatching(/\/runs\?/),
    pendingUrl(41),
    dispatchUrl,
  ])
})

it('paginates waiting-run listing until a short page', async () => {
  stuckRunListTest.perPage = 2
  const old = agedIso(30)
  const page1 = [
    { id: 1000, head_branch: 'main', status: 'waiting', created_at: old },
    { id: 1001, head_branch: 'main', status: 'waiting', created_at: old },
  ]
  const page2 = [{ id: 2001, head_branch: 'main', status: 'waiting', created_at: old }]
  const fetcher = vi.fn(async (url: string) => {
    if (isListUrl(url) && url.includes('page=1')) {
      return new Response(JSON.stringify({ workflow_runs: page1 }), { status: 200 })
    }
    if (isListUrl(url) && url.includes('page=2')) {
      return new Response(JSON.stringify({ workflow_runs: page2 }), { status: 200 })
    }
    if (typeof url === 'string' && url.includes('/pending_deployments')) {
      return new Response(JSON.stringify([]), { status: 200 })
    }
    if (typeof url === 'string' && url.includes('/force-cancel')) {
      return new Response(null, { status: 202 })
    }
    if (url === dispatchUrl) return new Response(null, { status: 204 })
    throw new Error(`unexpected url ${url}`)
  })
  vi.stubGlobal('fetch', fetcher)
  await scheduler.scheduled({ cron: '7 * * * *' }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' })
  const listCalls = fetcher.mock.calls.filter(call => isListUrl(call[0]))
  expect(listCalls).toHaveLength(2)
  expect(listCalls[0]![0]).toContain('page=1')
  expect(listCalls[0]![0]).toContain('per_page=2')
  expect(listCalls[1]![0]).toContain('page=2')
  expect(fetcher.mock.calls.some(call => call[0] === cancelUrl(1000))).toBe(true)
  expect(fetcher.mock.calls.some(call => call[0] === cancelUrl(1001))).toBe(true)
  expect(fetcher.mock.calls.some(call => call[0] === cancelUrl(2001))).toBe(true)
  expect(fetcher.mock.calls.at(-1)![0]).toBe(dispatchUrl)
})

it('continues dispatch when stuck-run cleanup fails', async () => {
  const fetcher = vi.fn()
    .mockRejectedValueOnce(new Error('list failed'))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
  vi.stubGlobal('fetch', fetcher)
  await scheduler.scheduled({ cron: '7 * * * *' }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' })
  expect(fetcher).toHaveBeenCalledTimes(2)
  expect(fetcher.mock.calls[1]![0]).toBe(dispatchUrl)
})

it('fails closed for unknown schedules or missing credentials', async () => {
  const fetcher = vi.fn()
  vi.stubGlobal('fetch', fetcher)
  for (const cron of ['* * * * *', 'toString', '__proto__']) {
    await expect(scheduler.scheduled({ cron }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' })).rejects.toThrow('Unknown')
  }
  await expect(scheduler.scheduled({ cron: '7 * * * *' }, {})).rejects.toThrow('missing')
  expect(fetcher).not.toHaveBeenCalled()
})

it.each([200, 301, 401, 403, 429, 500])('reports rejected dispatch %s without reading its body or treating it as success', async status => {
  const response = new Response('private upstream error', { status })
  const fetcher = stubFetchSequence(emptyList(), response)
  await expect(scheduler.scheduled({ cron: '7 * * * *' }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' }))
    .rejects.toThrow(`Status dispatch rejected (${status})`)
  expect(fetcher).toHaveBeenCalledTimes(2)
  expect(response.bodyUsed).toBe(true)
})

it('sanitizes network errors and does not retry an ambiguously accepted dispatch', async () => {
  const fetcher = vi.fn()
    .mockResolvedValueOnce(emptyList())
    .mockRejectedValueOnce(new Error('private upstream error'))
  vi.stubGlobal('fetch', fetcher)
  await expect(scheduler.scheduled({ cron: '7 * * * *' }, { GITHUB_DISPATCH_TOKEN: 'test-only-token' }))
    .rejects.toThrow('Status dispatch request failed')
  expect(fetcher).toHaveBeenCalledTimes(2)
})

it('never triggers collection for an HTTP request', () => {
  const fetcher = vi.fn()
  vi.stubGlobal('fetch', fetcher)
  expect(scheduler.fetch().status).toBe(404)
  expect(fetcher).not.toHaveBeenCalled()
})

it('isolates production scheduling from preview, public routes and stored provider data', () => {
  const config = JSON.parse(readFileSync('wrangler.scheduler.jsonc', 'utf8'))
  expect(config.triggers.crons).toEqual([])
  expect(config.workers_dev).toBe(false)
  expect(config.preview_urls).toBe(false)
  const production = config.env.production
  expect(production.triggers.crons).toEqual(['2,17,32,47 * * * *', '7 * * * *'])
  expect(production.workers_dev).toBe(false)
  expect(production.preview_urls).toBe(false)
  for (const target of [config, production]) {
    for (const binding of ['routes', 'r2_buckets', 'kv_namespaces', 'services', 'vars']) expect(target[binding]).toBeUndefined()
  }
})
