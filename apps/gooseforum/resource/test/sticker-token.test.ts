import { describe, expect, it } from 'vitest'
import { parseStickerSegments, stickerPreviewLabel } from '@/site/utils/sticker-token'

const urlByName = new Map([
  ['滑稽', '/file/img/stickers/huaji.png'],
  ['flower-1', '/file/img/stickers/flower-1.png'],
])

describe('parseStickerSegments', () => {
  it('无 token 时返回单文本段且内容一致', () => {
    const content = '普通消息，没有表情包'
    expect(parseStickerSegments(content, urlByName)).toEqual([{ type: 'text', text: content }])
  })

  it('识别启用表情包为贴纸段', () => {
    expect(parseStickerSegments('[:sticker:滑稽:]', urlByName)).toEqual([
      { type: 'sticker', name: '滑稽', url: '/file/img/stickers/huaji.png' },
    ])
  })

  it('未知或停用 token 保持原文', () => {
    const content = '未知 [:sticker:不存在:] 与停用 [:sticker:flower-1:] 之后的文本'
    const segments = parseStickerSegments(content, new Map())
    expect(segments).toEqual([{ type: 'text', text: content }])
  })

  it('混合文本与多个 token 正确切分', () => {
    const segments = parseStickerSegments('看这个 [:sticker:滑稽:] 再看 [:sticker:flower-1:]!', urlByName)
    expect(segments).toEqual([
      { type: 'text', text: '看这个 ' },
      { type: 'sticker', name: '滑稽', url: '/file/img/stickers/huaji.png' },
      { type: 'text', text: ' 再看 ' },
      { type: 'sticker', name: 'flower-1', url: '/file/img/stickers/flower-1.png' },
      { type: 'text', text: '!' },
    ])
  })

  it('相邻 token 不丢中间文本', () => {
    const segments = parseStickerSegments('[:sticker:滑稽:][:sticker:flower-1:]', urlByName)
    expect(segments).toHaveLength(2)
    expect(segments[0]).toMatchObject({ type: 'sticker', name: '滑稽' })
    expect(segments[1]).toMatchObject({ type: 'sticker', name: 'flower-1' })
  })
})

describe('stickerPreviewLabel', () => {
  it('把 token 缩写为 [name]', () => {
    expect(stickerPreviewLabel('发个 [:sticker:滑稽:] 给你')).toBe('发个 [滑稽] 给你')
  })

  it('未知 token 同样缩写', () => {
    expect(stickerPreviewLabel('[:sticker:停用的:]')).toBe('[停用的]')
  })

  it('无 token 原文返回', () => {
    expect(stickerPreviewLabel('普通文本')).toBe('普通文本')
  })
})
