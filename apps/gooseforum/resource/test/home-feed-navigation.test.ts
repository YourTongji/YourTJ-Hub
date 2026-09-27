import { afterEach, describe, expect, test } from 'vitest'
import { homeFeedNavigation } from '../src/runtime/home-feed-navigation'

afterEach(() => homeFeedNavigation.cancel())

describe('home feed navigation', () => {
  test('keeps the newest destination when requests finish out of order', () => {
    const first = homeFeedNavigation.begin('/?sort=hot')
    const second = homeFeedNavigation.begin('/?sort=popular')

    expect(homeFeedNavigation.complete(first)).toBe(false)
    expect(homeFeedNavigation.pendingUrl.value).toBe('/?sort=popular')
    expect(homeFeedNavigation.complete(second)).toBe(true)
    expect(homeFeedNavigation.pendingUrl.value).toBeNull()
  })

  test('keeps the failed destination available for retry until another navigation starts', () => {
    const request = homeFeedNavigation.begin('/?sort=following')

    expect(homeFeedNavigation.fail(request, '/?sort=following')).toBe(true)
    expect(homeFeedNavigation.failedUrl.value).toBe('/?sort=following')

    homeFeedNavigation.begin('/?sort=hot')
    expect(homeFeedNavigation.failedUrl.value).toBeNull()
  })

  test('cancels a pending destination when leaving the home feed', () => {
    const request = homeFeedNavigation.begin('/?sort=hot')

    homeFeedNavigation.cancel()

    expect(homeFeedNavigation.complete(request)).toBe(false)
    expect(homeFeedNavigation.pendingUrl.value).toBeNull()
  })
})
