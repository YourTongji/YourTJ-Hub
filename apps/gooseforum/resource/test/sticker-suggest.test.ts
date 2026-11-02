import { describe, expect, it } from 'vitest'
import { extractStickerQuery, rankStickerSuggestions } from '@/site/utils/sticker-token'

describe('extractStickerQuery', () => {
  it('recognises an open fragment before the caret', () => {
    expect(extractStickerQuery('看 [:sticker:')).toEqual({ length: 10, query: '' })
    expect(extractStickerQuery('看 [:sticker:hu')).toEqual({ length: 12, query: 'hu' })
  })

  it('ignores closed tokens, broken names and escaped fragments', () => {
    expect(extractStickerQuery('[:sticker:huaji:]')).toBeNull()
    expect(extractStickerQuery('[:sticker:huaji:')).toBeNull()
    expect(extractStickerQuery('[:sticker:hu aji')).toBeNull()
    expect(extractStickerQuery('\\[:sticker:hu')).toBeNull()
    expect(extractStickerQuery('\\\\[:sticker:hu')).toEqual({ length: 12, query: 'hu' })
    expect(extractStickerQuery(`[:sticker:${'a'.repeat(65)}`)).toBeNull()
  })
})

describe('rankStickerSuggestions', () => {
  const items = [
    { name: 'u_1', url: '/1.png', label: '我的滑稽' },
    { name: 'huaji', url: '/h.png', label: 'huaji' },
    { name: 'shuai', url: '/s.png', label: 'shuai' },
    { name: 'hug', url: '/g.png', label: 'hug' },
    { name: 'huaji', url: '/dup.png', label: 'huaji' },
  ]

  it('keeps source order for an empty query and drops duplicate names', () => {
    expect(rankStickerSuggestions(items, '').map((item) => item.url)).toEqual(['/1.png', '/h.png', '/s.png', '/g.png'])
    expect(rankStickerSuggestions(items, '', 2)).toHaveLength(2)
  })

  it('ranks exact, then prefix (name or label), then substring matches', () => {
    expect(rankStickerSuggestions(items, 'hu').map((item) => item.name)).toEqual(['huaji', 'hug', 'shuai'])
    expect(rankStickerSuggestions(items, 'hug').map((item) => item.name)).toEqual(['hug'])
    expect(rankStickerSuggestions(items, '我的').map((item) => item.name)).toEqual(['u_1'])
    expect(rankStickerSuggestions(items, 'HU').map((item) => item.name)).toEqual(['huaji', 'hug', 'shuai'])
    expect(rankStickerSuggestions(items, 'zzz')).toEqual([])
  })
})
