// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { version as vueVersion } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import TwemojiHtmlFixture from './fixtures/TwemojiHtmlFixture.vue'
import TwemojiEditableFixture from './fixtures/TwemojiEditableFixture.vue'
import TwemojiFlexFixture from './fixtures/TwemojiFlexFixture.vue'
import TwemojiTemplateFixture from './fixtures/TwemojiTemplateFixture.vue'
import { renderMarkdownPreview } from '../src/runtime/markdown'
import { parseTwemoji, renderTwemojiHtml, TWEMOJI_VERSION, TwemojiText } from '../src/runtime/twemoji'

const compilerDomPackage = JSON.parse(
  readFileSync(resolve(process.cwd(), 'node_modules/@vue/compiler-dom/package.json'), 'utf8'),
) as { version: string }

describe('shared Twemoji mapping', () => {
  it('keeps compiler helper symbols on the same Vue compiler version', () => {
    expect(compilerDomPackage.version).toBe(vueVersion)
  })

  it('keeps Unicode text and matches complete joined, regional, keycap, and tone sequences', () => {
    const text = '中文 A😀 🐦‍🔥 🇺🇸 1️⃣ 👩🏽‍💻!'
    const segments = parseTwemoji(text)

    expect(segments.filter((segment) => segment.type === 'emoji').map((segment) => segment.value)).toEqual([
      '😀',
      '🐦‍🔥',
      '🇺🇸',
      '1️⃣',
      '👩🏽‍💻',
    ])
    expect(segments.map((segment) => segment.value).join('')).toBe(text)
  })

  it('normalizes text variation selectors without changing the displayed Unicode', () => {
    expect(parseTwemoji('❤\uFE0F')).toEqual([
      { type: 'emoji', value: '❤\uFE0F', asset: '2764.png' },
    ])
  })

  it('respects default text presentation and explicit variation selectors', () => {
    expect(parseTwemoji('© ® ™ ❤')).toEqual([
      { type: 'text', value: '© ® ™ ❤' },
    ])
    expect(parseTwemoji('©\uFE0E ®\uFE0E ™\uFE0E ❤\uFE0E')).toEqual([
      { type: 'text', value: '©\uFE0E ®\uFE0E ™\uFE0E ❤\uFE0E' },
    ])
    expect(parseTwemoji('©\uFE0F ®\uFE0F ™\uFE0F ❤\uFE0F')).toEqual([
      { type: 'emoji', value: '©\uFE0F', asset: 'a9.png' },
      { type: 'text', value: ' ' },
      { type: 'emoji', value: '®\uFE0F', asset: 'ae.png' },
      { type: 'text', value: ' ' },
      { type: 'emoji', value: '™\uFE0F', asset: '2122.png' },
      { type: 'text', value: ' ' },
      { type: 'emoji', value: '❤\uFE0F', asset: '2764.png' },
    ])
    expect(parseTwemoji('😀\uFE0E')).toEqual([
      { type: 'text', value: '😀\uFE0E' },
    ])
    expect(parseTwemoji('😀\uFE0F')).toEqual([
      { type: 'emoji', value: '😀\uFE0F', asset: '1f600.png' },
    ])
  })

  it('keeps the Taiwan flag sequence as its original Unicode text', () => {
    const flag = '🇹🇼'
    expect(parseTwemoji(flag)).toEqual([{ type: 'text', value: flag }])
  })

  it('leaves unsupported sequences as original text and pins the released artwork', () => {
    const unsupported = '🐦‍🛹'
    expect(parseTwemoji(unsupported)).toEqual([{ type: 'text', value: unsupported }])
    expect(TWEMOJI_VERSION).toBe('17.0.3')
  })
})

describe('Vue Twemoji rendering', () => {
  it('does not compile-transform editable subtrees and keeps ordinary and false regions rendered', async () => {
    const wrapper = mount(TwemojiEditableFixture, {
      props: {
        editable: false,
        editableAttribute: 'contenteditable',
        editableValue: false,
        editableAttrs: { contenteditable: false },
      },
    })

    const ordinary = wrapper.find('.ordinary')
    expect(ordinary.findAll('img.twemoji').map((image) => image.attributes('alt'))).toEqual([
      '🐦‍🔥',
      '😀',
    ])
    expect(ordinary.text()).toBe('ordinary 中文  mixed  text')
    expect(wrapper.find('.editable-false').findAll('img.twemoji')).toHaveLength(1)

    for (const selector of [
      '.editable-boolean',
      '.editable-empty',
      '.editable-true',
      '.editable-dynamic',
      '.editable-dynamic-argument',
      '.editable-object-binding',
    ]) {
      const editableRegion = wrapper.find(selector)
      expect(editableRegion.exists()).toBe(true)
      expect(editableRegion.findAll('img.twemoji')).toHaveLength(0)
    }
    expect(wrapper.find('.editable-boolean').text()).toContain('nested 🧑🏽‍💻')
    expect(wrapper.find('.editable-empty').text()).toContain('nested 🇨🇳')
    expect(wrapper.find('.editable-true').text()).toContain('nested ❤️')

    await wrapper.setProps({ editable: true, editableValue: true })
    expect(wrapper.find('.editable-dynamic').findAll('img.twemoji')).toHaveLength(0)
    expect(wrapper.find('.editable-dynamic-argument').findAll('img.twemoji')).toHaveLength(0)
    wrapper.unmount()
  })

  it('keeps runtime HTML replacement in sync with editable state changes', () => {
    const richHtml = document.createElement('div')
    richHtml.setAttribute('contenteditable', 'false')
    richHtml.innerHTML = '<p>mixed 中文 🐦‍🔥 text</p><div contenteditable="">nested ✨</div>'
    renderTwemojiHtml(richHtml)
    expect(richHtml.querySelector('p img.twemoji')?.alt).toBe('🐦‍🔥')
    expect(richHtml.querySelector('div[contenteditable=""] img.twemoji')).toBeNull()

    richHtml.setAttribute('contenteditable', '')
    renderTwemojiHtml(richHtml)
    expect(richHtml.querySelector('img.twemoji')).toBeNull()
    expect(richHtml.textContent).toContain('🐦‍🔥')

    richHtml.setAttribute('contenteditable', 'false')
    renderTwemojiHtml(richHtml)
    expect(richHtml.querySelector('p img.twemoji')?.alt).toBe('🐦‍🔥')
    expect(richHtml.querySelector('div[contenteditable=""] img.twemoji')).toBeNull()
  })

  it('re-renders direct DOM edits once and refreshes the HTML cache', () => {
    const richHtml = document.createElement('div')
    richHtml.innerHTML = '<p>initial 🐦‍🔥</p>'
    renderTwemojiHtml(richHtml)
    expect(richHtml.querySelectorAll('img.twemoji')).toHaveLength(1)

    richHtml.innerHTML = '<p>direct edit ❤️ and 🇨🇳</p>'
    renderTwemojiHtml(richHtml)
    expect(richHtml.querySelectorAll('img.twemoji')).toHaveLength(2)
    const rendered = richHtml.innerHTML
    renderTwemojiHtml(richHtml)
    expect(richHtml.innerHTML).toBe(rendered)

    richHtml.setAttribute('contenteditable', 'true')
    renderTwemojiHtml(richHtml)
    expect(richHtml.querySelectorAll('img.twemoji')).toHaveLength(0)
    richHtml.innerHTML = '<p>editable direct edit 🐦‍🔥</p>'
    renderTwemojiHtml(richHtml)
    expect(richHtml.querySelectorAll('img.twemoji')).toHaveLength(0)

    richHtml.setAttribute('contenteditable', 'false')
    renderTwemojiHtml(richHtml)
    expect(richHtml.querySelectorAll('img.twemoji')).toHaveLength(1)
  })

  it('renders the Vditor preview after syntax highlighting without changing its structure', () => {
    const preview = document.createElement('div')
    preview.innerHTML = `<pre><code><span class="hljs-keyword">const</span> emoji = '🐦‍🔥'</code></pre>`

    renderTwemojiHtml(preview)

    expect(preview.querySelector('.hljs-keyword')?.textContent).toBe('const')
    // MADR 0089：代码区域跳过 Twemoji，等宽对齐与语法高亮不被 img 破坏。
    expect(preview.querySelector('pre code img.twemoji')).toBeNull()
    expect(preview.querySelector('pre code')?.textContent).toContain('🐦‍🔥')
  })

  it('renders static and reactive text nodes with image alt text', () => {
    const wrapper = mount(TwemojiTemplateFixture, {
      props: { text: '中文 comment 🐦‍🔥!' },
    })

    const images = wrapper.findAll('img.twemoji')
    expect(images.map((image) => image.attributes('alt'))).toEqual([
      '😀',
      '🐦‍🔥',
    ])
    expect(images[1].attributes('src')).toContain(
      '/twemoji/72x72/1f426-200d-1f525.png',
    )
    // 模板中的 code 元素同样跳过替换（编译期与运行时共用同一排除清单）。
    expect(wrapper.find('code img.twemoji').exists()).toBe(false)
    expect(wrapper.find('code').text()).toBe('🐦‍🔥')
  })

  it('keeps default-text copyright readable under flex parents', () => {
    const wrapper = mount(TwemojiFlexFixture, {
      props: { primary: '© 2026 YourTJ Community. All rights reserved.' },
    })

    const host = wrapper.find('.footer-primary')
    expect(host.element.childNodes).toHaveLength(1)
    expect(host.element.textContent).toBe('© 2026 YourTJ Community. All rights reserved.')
    expect(host.find('img.twemoji').exists()).toBe(false)
    wrapper.unmount()
  })

  it('renders plain text without a wrapper when no emoji is present', () => {
    const wrapper = mount(TwemojiText, { props: { text: '#2026' } })

    expect(wrapper.text()).toBe('#2026')
    expect(wrapper.find('span').exists()).toBe(false)
    wrapper.unmount()
  })

  it('preserves Markdown structure and keeps code emoji as source text in v-html output', () => {
    const html = renderMarkdownPreview('**Markdown 🐦‍🔥** and `🔥`\n\n[Link ❤️](https://example.com)')
    const wrapper = mount(TwemojiHtmlFixture, { props: { html } })

    expect(wrapper.findAll('img.twemoji').map((image) => image.attributes('alt'))).toEqual([
      '🐦‍🔥',
      '❤️',
    ])
    expect(wrapper.find('strong').exists()).toBe(true)
    expect(wrapper.find('a').attributes('href')).toBe('https://example.com')
    expect(wrapper.find('code img.twemoji').exists()).toBe(false)
    expect(wrapper.find('code').text()).toBe('🔥')
  })

  it('does not replace emoji inside code, pre, kbd or samp in runtime HTML', () => {
    const richHtml = document.createElement('div')
    richHtml.innerHTML =
      '<p>para 😀 text</p><pre>block 😀 code</pre>' +
      '<p><code>inline 😀</code> <kbd>key 😀</kbd> <samp>out 😀</samp></p>'

    renderTwemojiHtml(richHtml)

    expect(richHtml.querySelector('p img.twemoji')).not.toBeNull()
    for (const tag of ['pre', 'code', 'kbd', 'samp']) {
      expect(richHtml.querySelector(`${tag} img.twemoji`)).toBeNull()
      expect(richHtml.querySelector(tag)?.textContent).toContain('😀')
    }
  })

  it('falls back to source Unicode when an image cannot load', async () => {
    const wrapper = mount(TwemojiTemplateFixture, {
      props: { text: '中文 comment 🐦‍🔥!' },
    })

    for (const image of wrapper.findAll('img.twemoji')) {
      await image.trigger('error')
    }
    expect(wrapper.find('img.twemoji').exists()).toBe(false)
    expect(wrapper.text()).toContain('😀')
    expect(wrapper.text()).toContain('🐦‍🔥')
    wrapper.unmount()
  })

  it('falls back to source Unicode for a missing rich-HTML image', async () => {
    const wrapper = mount(TwemojiHtmlFixture, {
      props: { html: '<p>离线显示 🐦‍🔥</p>' },
    })

    await wrapper.find('img.twemoji').trigger('error')
    expect(wrapper.find('img.twemoji').exists()).toBe(false)
    expect(wrapper.find('p').text()).toBe('离线显示 🐦‍🔥')
    wrapper.unmount()
  })

  it('preserves rich HTML and plain Unicode when copying selected Twemoji content', () => {
    const wrapper = mount(TwemojiHtmlFixture, {
      props: {
        html: '<p><strong>复制 🐦‍🔥</strong> and <a href="https://example.com"><em>❤️</em></a></p><ul><li>list 🔥</li></ul>',
      },
    })
    const range = document.createRange()
    range.selectNodeContents(wrapper.element)
    const selection = window.getSelection()!
    selection.removeAllRanges()
    selection.addRange(range)

    const values: Record<string, string> = {}
    const event = new Event('copy', { bubbles: true, cancelable: true })
    Object.defineProperty(event, 'clipboardData', {
      value: { setData: (type: string, value: string) => (values[type] = value) },
    })
    document.dispatchEvent(event)

    expect(event.defaultPrevented).toBe(true)
    expect(values['text/plain']).toBe('复制 🐦‍🔥 and ❤️\nlist 🔥')
    expect(values['text/html']).toBe('<p><strong>复制 🐦‍🔥</strong> and <a href="https://example.com"><em>❤️</em></a></p><ul><li>list 🔥</li></ul>')
    selection.removeAllRanges()
    wrapper.unmount()
  })

  it('does not replace a clipboard already handled by another editor', () => {
    const wrapper = mount(TwemojiHtmlFixture, {
      props: { html: '<p>copy 🐦‍🔥</p>' },
      attachTo: document.body,
    })
    const range = document.createRange()
    range.selectNodeContents(wrapper.element)
    const selection = window.getSelection()!
    selection.removeAllRanges()
    selection.addRange(range)

    for (const mode of ['prevented', 'prepopulated'] as const) {
      const values: Record<string, string> = {}
      const types: string[] = []
      const event = new Event('copy', { bubbles: true, cancelable: true })
      const clipboardData = {
        types,
        setData: (type: string, value: string) => {
          values[type] = value
          if (!types.includes(type)) types.push(type)
        },
      }
      Object.defineProperty(event, 'clipboardData', { value: clipboardData })
      const existingHandler = (copyEvent: Event) => {
        const copy = copyEvent as ClipboardEvent
        if (mode === 'prevented') copy.preventDefault()
        else copy.clipboardData?.setData('text/plain', 'editor-owned copy')
      }
      document.addEventListener('copy', existingHandler, {
        once: true,
      })
      wrapper.element.dispatchEvent(event)

      if (mode === 'prevented') {
        expect(event.defaultPrevented).toBe(true)
        expect(values).toEqual({})
      } else {
        expect(values).toEqual({ 'text/plain': 'editor-owned copy' })
      }
    }

    selection.removeAllRanges()
    wrapper.unmount()
  })

  it('does not take ownership of copy events inside contenteditable', () => {
    const editor = document.createElement('div')
    editor.contentEditable = 'true'
    editor.innerHTML = '<p>editable 🐦‍🔥</p>'
    const image = document.createElement('img')
    image.className = 'twemoji'
    image.alt = '🐦‍🔥'
    image.dataset.emoji = '🐦‍🔥'
    editor.querySelector('p')?.append(image)
    document.body.append(editor)
    const range = document.createRange()
    range.selectNodeContents(editor)
    const selection = window.getSelection()!
    selection.removeAllRanges()
    selection.addRange(range)
    const values: Record<string, string> = {}
    const event = new Event('copy', { bubbles: true, cancelable: true })
    Object.defineProperty(event, 'clipboardData', {
      value: { types: [], setData: (type: string, value: string) => (values[type] = value) },
    })

    document.dispatchEvent(event)

    expect(event.defaultPrevented).toBe(false)
    expect(values).toEqual({})
    selection.removeAllRanges()
    editor.remove()
  })

  it('joins disjoint selected ranges without altering their Unicode', () => {
    const wrapper = mount(TwemojiHtmlFixture, {
      props: { html: '<p>first 🐦‍🔥</p><p>second 🇨🇳</p>' },
      attachTo: document.body,
    })
    const paragraphs = wrapper.element.querySelectorAll('p')
    const ranges = Array.from(paragraphs, (paragraph) => {
      const range = document.createRange()
      range.selectNodeContents(paragraph)
      return range
    })
    const fakeSelection = {
      isCollapsed: false,
      rangeCount: ranges.length,
      getRangeAt: (index: number) => ranges[index],
    } as Selection
    const getSelection = vi
      .spyOn(window, 'getSelection')
      .mockReturnValue(fakeSelection)
    const values: Record<string, string> = {}
    const event = new Event('copy', { bubbles: true, cancelable: true })
    Object.defineProperty(event, 'clipboardData', {
      value: { types: [], setData: (type: string, value: string) => (values[type] = value) },
    })

    document.dispatchEvent(event)

    expect(event.defaultPrevented).toBe(true)
    expect(values['text/plain']).toContain('first 🐦‍🔥')
    expect(values['text/plain']).toContain('second 🇨🇳')
    expect(values['text/html']).toContain('first 🐦‍🔥')
    expect(values['text/html']).toContain('second 🇨🇳')
    getSelection.mockRestore()
    wrapper.unmount()
  })
})
