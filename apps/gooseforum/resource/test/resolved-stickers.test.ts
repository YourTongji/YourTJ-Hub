import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useResolvedStickers } from '@/site/composables/useResolvedStickers'

const { resolve } = vi.hoisted(() => ({ resolve: vi.fn() }))
vi.mock('@/runtime/api', () => ({ resolveForumStickers: resolve }))

describe('shared personal sticker resolution', () => {
  beforeEach(() => { resolve.mockReset() })
  afterEach(() => { vi.restoreAllMocks() })

  it('resolves explicit personal tokens and deduplicates repeated content', async () => {
    const item = { name: 'u_random', url: '/file/img/shared.png', isOfficial: false }
    resolve.mockResolvedValue([item])
    const library = useResolvedStickers()
    await library.ensureContentStickers(['hello [:sticker:u_random:]', '[:sticker:u_random:]'])
    expect(resolve).toHaveBeenCalledWith(['u_random'])
    expect(library.resolvedUrls.value.get('u_random')).toBe(item.url)
    await library.ensureContentStickers(['[:sticker:u_random:]'])
    expect(resolve).toHaveBeenCalledTimes(1)
  })

  it('batches a long history to the API limit and skips ordinary text', async () => {
    resolve.mockResolvedValue([])
    const library = useResolvedStickers()
    await library.ensureContentStickers(['plain text'])
    expect(resolve).not.toHaveBeenCalled()
    await library.ensureContentStickers(Array.from({ length: 201 }, (_, index) => `[:sticker:s${index}:]`))
    expect(resolve.mock.calls.map((call) => call[0].length)).toEqual([200, 1])
  })

  it('retries failed resolution without replacing existing images', async () => {
    resolve.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce([{ name: 'u_x', url: '/x.png' }])
    const library = useResolvedStickers()
    await library.ensureContentStickers(['[:sticker:u_x:]'])
    expect(library.resolvedUrls.value.size).toBe(0)
    await library.ensureContentStickers(['[:sticker:u_x:]'])
    expect(library.resolvedUrls.value.get('u_x')).toBe('/x.png')
  })

  it('deduplicates pending names while merging independent responses in arrival order', async () => {
    let finishFirst!: (items: { name: string; url: string }[]) => void
    resolve.mockReturnValueOnce(new Promise((done) => { finishFirst = done }))
      .mockResolvedValueOnce([{ name: 'u_y', url: '/y.png' }])
    const library = useResolvedStickers()
    const first = library.ensureContentStickers(['[:sticker:u_x:]'])
    await library.ensureContentStickers(['[:sticker:u_x:][:sticker:u_y:]'])
    expect(resolve.mock.calls.map(([names]) => names)).toEqual([['u_x'], ['u_y']])
    finishFirst([{ name: 'u_x', url: '/x.png' }])
    await first
    expect(Object.fromEntries(library.resolvedUrls.value)).toEqual({ u_x: '/x.png', u_y: '/y.png' })
  })

  it('retains stale images on failure but removes disabled tokens after a successful refresh', async () => {
    const clock = vi.spyOn(Date, 'now').mockReturnValue(100_000)
    resolve.mockResolvedValueOnce([{ name: 'u_x', url: '/x.png' }])
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValueOnce([])
    const library = useResolvedStickers()
    await library.ensureContentStickers(['[:sticker:u_x:]'])
    clock.mockReturnValue(159_999)
    await library.ensureContentStickers(['[:sticker:u_x:]'])
    expect(resolve).toHaveBeenCalledTimes(1)
    clock.mockReturnValue(160_000)
    await library.ensureContentStickers(['[:sticker:u_x:]'])
    expect(library.resolvedUrls.value.get('u_x')).toBe('/x.png')
    await library.ensureContentStickers(['[:sticker:u_x:]'])
    expect(resolve).toHaveBeenCalledTimes(3)
    expect(library.resolvedUrls.value.has('u_x')).toBe(false)
  })

  it('retries only unfinished batches after a partial history resolution fails', async () => {
    resolve.mockResolvedValueOnce([{ name: 's0', url: '/0.png' }])
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValueOnce([{ name: 's200', url: '/200.png' }])
    const library = useResolvedStickers()
    const history = Array.from({ length: 201 }, (_, index) => `[:sticker:s${index}:]`)
    await library.ensureContentStickers(history)
    expect(library.resolvedUrls.value.get('s0')).toBe('/0.png')
    await library.ensureContentStickers(history)
    expect(resolve.mock.calls.map((call) => call[0].length)).toEqual([200, 1, 1])
    expect(resolve.mock.calls[2][0]).toEqual(['s200'])
    expect(library.resolvedUrls.value.get('s200')).toBe('/200.png')
  })
})
