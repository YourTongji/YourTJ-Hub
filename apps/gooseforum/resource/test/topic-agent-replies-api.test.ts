import { afterEach, expect, test, vi } from 'vitest'
vi.mock('../src/runtime/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }))
import { updateTopicAgentReplies } from '../src/runtime/api'
afterEach(() => vi.unstubAllGlobals())
test('unparseable reply policy response uses a save fallback', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('invalid json')))
  await expect(updateTopicAgentReplies(42, true)).rejects.toThrow('publish.saveFailed')
})
