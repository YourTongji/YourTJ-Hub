import { describe, expect, it } from 'vitest'
import { maskPersonalStickerTokens, parseStickerSegments, stickerDeleteRangeAt, stickerNamesInContent, stickerPreviewLabel } from '@/site/utils/sticker-token'

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
  it('预览用统一文案隐藏官方及个人 token 名', () => {
    expect(stickerPreviewLabel('发个 [:sticker:滑稽:]', 'Animated sticker')).toBe('发个 [Animated sticker]')
    expect(stickerPreviewLabel('[:sticker:u_92f1a:]', 'Animated sticker')).toBe('[Animated sticker]')
  })

  it('未知 token 同样缩写', () => {
    expect(stickerPreviewLabel('[:sticker:停用的:]', 'Animated sticker')).toBe('[Animated sticker]')
  })

  it('无 token 原文返回', () => {
    expect(stickerPreviewLabel('普通文本')).toBe('普通文本')
  })
})

describe('maskPersonalStickerTokens', () => {
  const personal = `[:sticker:u_${'a1'.repeat(24)}:]`

  it('隐藏个人 token 随机名', () => {
    expect(maskPersonalStickerTokens(`早 ${personal}`, 'Animated sticker')).toBe('早 [Animated sticker]')
  })

  it('不影响官方与可解析 token', () => {
    expect(maskPersonalStickerTokens('[:sticker:滑稽:] 文本', 'Animated sticker')).toBe('[:sticker:滑稽:] 文本')
  })

  it('隐藏已停用市场 token 的随机名', () => {
    expect(maskPersonalStickerTokens(`[:sticker:m_${'b2'.repeat(20)}:]`, 'Animated sticker')).toBe('[Animated sticker]')
    expect(maskPersonalStickerTokens('[:sticker:m_short:]', 'Animated sticker')).toBe('[:sticker:m_short:]')
  })

  it('非 48 位 hex 的 u_ token 不遮蔽', () => {
    expect(maskPersonalStickerTokens('[:sticker:u_92f1a:]', 'Animated sticker')).toBe('[:sticker:u_92f1a:]')
  })

  it('无 token 原文返回', () => {
    expect(maskPersonalStickerTokens('普通文本', 'Animated sticker')).toBe('普通文本')
  })
})

describe('stickerDeleteRangeAt', () => {
  const content = '看 [:sticker:滑稽:] 好'
  const tokenStart = content.indexOf('[:sticker:')
  const tokenEnd = tokenStart + '[:sticker:滑稽:]'.length

  it('完整 token：Backspace 紧跟其后一次删完，Delete 不越界', () => {
    expect(stickerDeleteRangeAt(content, tokenEnd, 'backward')).toEqual({ start: tokenStart, end: tokenEnd })
    expect(stickerDeleteRangeAt(content, tokenEnd, 'forward')).toBeNull()
  })

  it('完整 token：Delete 正好在起点一次删完，Backspace 不越界', () => {
    expect(stickerDeleteRangeAt(content, tokenStart, 'forward')).toEqual({ start: tokenStart, end: tokenEnd })
    expect(stickerDeleteRangeAt(content, tokenStart, 'backward')).toBeNull()
  })

  it('光标落在 token 内部两方向都整段删除', () => {
    const inside = tokenStart + '[:sticker:'.length
    expect(stickerDeleteRangeAt(content, inside, 'backward')).toEqual({ start: tokenStart, end: tokenEnd })
    expect(stickerDeleteRangeAt(content, inside, 'forward')).toEqual({ start: tokenStart, end: tokenEnd })
  })

  it('相邻 token 各删各的', () => {
    const pair = '[:sticker:a:][:sticker:b:]'
    expect(stickerDeleteRangeAt(pair, 13, 'backward')).toEqual({ start: 0, end: 13 })
    expect(stickerDeleteRangeAt(pair, 13, 'forward')).toEqual({ start: 13, end: 26 })
  })

  it('未闭合片段：两端都可整段删除', () => {
    const fragment = '看 [:sticker:na'
    const start = fragment.indexOf('[:sticker:')
    expect(stickerDeleteRangeAt(fragment, fragment.length, 'backward')).toEqual({ start, end: fragment.length })
    expect(stickerDeleteRangeAt(fragment, start, 'forward')).toEqual({ start, end: fragment.length })
  })

  it('片段后已有空白时只删除片段本身', () => {
    expect(stickerDeleteRangeAt('[:sticker:na 后文', 12, 'backward')).toEqual({ start: 0, end: 12 })
  })

  it('单独前缀（后面再无字符）视为片段', () => {
    expect(stickerDeleteRangeAt('[:sticker:', 10, 'backward')).toEqual({ start: 0, end: 10 })
    expect(stickerDeleteRangeAt('[:sticker:', 0, 'forward')).toEqual({ start: 0, end: 10 })
  })

  it('前缀后紧跟终止字符不算片段', () => {
    expect(stickerDeleteRangeAt('[:sticker: 好', 10, 'backward')).toBeNull()
    expect(stickerDeleteRangeAt('[:sticker:]', 11, 'backward')).toBeNull()
    expect(stickerDeleteRangeAt('[:sticker:[:sticker:a:]', 10, 'backward')).toBeNull()
  })

  it('普通文本与相邻标点不干预', () => {
    expect(stickerDeleteRangeAt('hello world', 5, 'backward')).toBeNull()
    expect(stickerDeleteRangeAt('hello world', 5, 'forward')).toBeNull()
    expect(stickerDeleteRangeAt('(:sticker', 9, 'backward')).toBeNull()
  })

  it('名字超过 64 字符不算合法 token 也不成片段', () => {
    const overlong = `[:sticker:${'a'.repeat(65)}]`
    expect(stickerDeleteRangeAt(overlong, overlong.length - 1, 'backward')).toBeNull()
  })

  it('恰好 64 个名字字符的 token 整个命中', () => {
    const token = `[:sticker:${'a'.repeat(64)}:]`
    expect(stickerDeleteRangeAt(token, token.length, 'backward')).toEqual({ start: 0, end: token.length })
    expect(stickerDeleteRangeAt(token, 0, 'forward')).toEqual({ start: 0, end: token.length })
  })

  it('名字后紧跟冒号但未闭合时，末尾冒号也在片段内', () => {
    const fragment = '[:sticker:huaji:'
    expect(stickerDeleteRangeAt(fragment, fragment.length, 'backward')).toEqual({ start: 0, end: fragment.length })
  })

  it('前缀后紧跟冒号时回落默认删除', () => {
    expect(stickerDeleteRangeAt('[:sticker::x', 10, 'backward')).toBeNull()
    expect(stickerDeleteRangeAt('[:sticker::x', 10, 'forward')).toBeNull()
  })

  it('光标落在片段内部（非两端）两方向都整段删除', () => {
    const fragment = '看 [:sticker:na 后文'
    expect(stickerDeleteRangeAt(fragment, 8, 'backward')).toEqual({ start: 2, end: 14 })
    expect(stickerDeleteRangeAt(fragment, 8, 'forward')).toEqual({ start: 2, end: 14 })
  })

  it('渲染端可识别的 token 都被删除区间完整覆盖（防名字字符集漂移）', () => {
    const personal = `u_${'a1'.repeat(24)}`
    for (const name of ['滑稽', 'flower-1', personal, 'a'.repeat(64)]) {
      const token = `[:sticker:${name}:]`
      const content = `前缀 ${token} 后缀`
      expect(stickerNamesInContent([content])).toEqual([name])
      const start = content.indexOf(token)
      expect(stickerDeleteRangeAt(content, start + token.length, 'backward')).toEqual({ start, end: start + token.length })
      expect(stickerDeleteRangeAt(content, start, 'forward')).toEqual({ start, end: start + token.length })
    }
  })
})
