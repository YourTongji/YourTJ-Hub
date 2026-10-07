import { afterEach, describe, expect, test, vi } from 'vitest'
import { getMentionTargets } from '../src/runtime/api'

afterEach(() => vi.unstubAllGlobals())

describe('dedicated mention-target API', () => {
  test('queries the endpoint with the public limit and preserves actor type', async () => {
    const response = new Response(JSON.stringify({
      code: 0,
      result: [{
        userId: 34,
        username: 'helper-bot',
        nickname: 'Helper',
        avatarUrl: '/avatars/34.png',
        actorType: 'bot',
      }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    const fetchMock = vi.fn().mockResolvedValue(response)
    vi.stubGlobal('fetch', fetchMock)
    const controller = new AbortController()

    const targets = await getMentionTargets('helper', controller.signal)

    expect(fetchMock).toHaveBeenCalledWith('/api/forum/mention-targets?q=helper&limit=20', {
      headers: { Accept: 'application/json' },
      signal: controller.signal,
    })
    expect(targets).toEqual([{
      userId: 34,
      username: 'helper-bot',
      nickname: 'Helper',
      avatarUrl: '/avatars/34.png',
      actorType: 'bot',
    }])
  })
})
