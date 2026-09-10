import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { useHomeFeedMode } from '../src/runtime/home-feed-mode'

describe('useHomeFeedMode', () => {
  const localStorage = {
    getItem: vi.fn(),
    setItem: vi.fn(),
  }

  beforeEach(() => {
    localStorage.getItem.mockReset()
    localStorage.setItem.mockReset()
    vi.stubGlobal('window', { localStorage })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('shares one reactive mode between cached homepage instances', () => {
    const latestPage = useHomeFeedMode()
    const popularPage = useHomeFeedMode()

    latestPage.setFeedMode('table')
    latestPage.setFeedMode('card')

    expect(popularPage.feedMode).toBe(latestPage.feedMode)
    expect(popularPage.feedMode.value).toBe('card')
    expect(localStorage.setItem).toHaveBeenLastCalledWith('goose:home-feed-mode', 'card')
  })
})
