// @vitest-environment happy-dom
import { afterEach, describe, expect, it } from 'vitest'
import { decorateStickerTokens, stickerInlineAt } from '../src/runtime/sticker-inline-editor'

const assets = new Map([
  ['huaji', { url: '/s/huaji.png', label: 'huaji' }],
  ['doge', { url: '/s/doge.png', label: 'doge' }],
])
const assetFor = (name: string) => assets.get(name)

function mount(html: string): HTMLElement {
  document.body.innerHTML = `<pre id="editor" contenteditable="true">${html}</pre>`
  return document.querySelector<HTMLElement>('#editor')!
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('decorateStickerTokens', () => {
  it('wraps resolvable tokens and keeps the token text for Markdown', () => {
    const root = mount('<p>你好[:sticker:huaji:]再见[:sticker:doge:][:sticker:nobody:]</p>')
    expect(decorateStickerTokens(root, assetFor)).toBe(2)
    const spans = [...root.querySelectorAll<HTMLElement>('.gf-sticker-inline')]
    expect(spans.map((span) => span.dataset.sticker)).toEqual(['huaji', 'doge'])
    expect(spans[0].getAttribute('contenteditable')).toBe('false')
    expect(spans[0].getAttribute('role')).toBe('img')
    expect(spans[0].style.backgroundImage).toContain('/s/huaji.png')
    expect(root.textContent).toBe('你好[:sticker:huaji:]再见[:sticker:doge:][:sticker:nobody:]')
  })

  it('is idempotent and leaves code, links and escaped tokens as text', () => {
    const root = mount('<p><code>[:sticker:huaji:]</code> <a href="#">[:sticker:huaji:]</a> \\[:sticker:huaji:] <strong>[:sticker:huaji:]</strong></p>')
    expect(decorateStickerTokens(root, assetFor)).toBe(1)
    expect(decorateStickerTokens(root, assetFor)).toBe(0)
    expect(root.querySelector('strong .gf-sticker-inline')).not.toBeNull()
    expect(root.querySelector('code .gf-sticker-inline, a .gf-sticker-inline')).toBeNull()
  })

  it('unwraps a sticker that no longer resolves', () => {
    const root = mount('<p>[:sticker:huaji:]</p>')
    decorateStickerTokens(root, assetFor)
    expect(root.querySelector('.gf-sticker-inline')).not.toBeNull()
    decorateStickerTokens(root, () => undefined)
    expect(root.querySelector('.gf-sticker-inline')).toBeNull()
    expect(root.textContent).toBe('[:sticker:huaji:]')
  })

  it('moves a caret left inside the token text to after the image', () => {
    const root = mount('<p></p>')
    // Vditor 重渲染后光标落在 token 文本节点末尾（刚输入闭合的 `]`）
    const text = document.createTextNode('[:sticker:huaji:]')
    root.querySelector('p')!.append('前', text)
    const range = document.createRange()
    range.setStart(text, text.textContent!.length)
    range.collapse(true)
    getSelection()!.removeAllRanges()
    getSelection()!.addRange(range)
    decorateStickerTokens(root, assetFor)
    const caret = getSelection()!.getRangeAt(0)
    const span = root.querySelector('.gf-sticker-inline')!
    expect(stickerInlineAt(caret.startContainer)).toBeNull()
    expect(caret.startContainer).toBe(span.parentNode)
    expect(caret.startOffset).toBe([...span.parentNode!.childNodes].indexOf(span) + 1)
  })
})
