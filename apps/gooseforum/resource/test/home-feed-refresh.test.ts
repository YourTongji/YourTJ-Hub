import { describe, expect, test } from 'vitest'
import type { TopicPayload } from '@gooseforum/client'
import { countNewTopics, firstPageUrl, prependTopics } from '../src/site/utils/home-feed-refresh'

function topic(id: number): TopicPayload {
  return {
    id,
    title: `Topic ${id}`,
    description: '',
    url: `/topics/${id}`,
    author: { id: 1, username: 'mock-user', avatarUrl: '' },
    participants: [],
    categories: [],
    replyCount: 0,
    viewCount: 0,
    pinWeight: 0,
    processStatus: 0,
    activityText: '',
    lastUpdateTime: '2026-09-06T12:00:00Z',
    contentType: 0,
  }
}

describe('home feed refresh helpers', () => {
  test('counts only ids newer than the newest topic already loaded', () => {
    expect(countNewTopics([topic(10), topic(9), topic(8)], [topic(12), topic(11), topic(10)])).toBe(2)
  })

  test('does not report a hot-feed reorder as new content', () => {
    expect(countNewTopics([topic(12), topic(11), topic(10)], [topic(11), topic(12), topic(10)])).toBe(0)
  })

  test('prepends the refreshed first page while preserving older loaded topics without duplicates', () => {
    expect(prependTopics([topic(10), topic(9), topic(8)], [topic(12), topic(11), topic(10)]).map((item) => item.id))
      .toEqual([12, 11, 10, 9, 8])
  })

  test('normalizes the current sort url back to page one', () => {
    expect(firstPageUrl('/?sort=hot&page=4', 'https://forum.example').toString())
      .toBe('https://forum.example/?sort=hot')
  })
})
