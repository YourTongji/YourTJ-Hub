// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import VditorOfficial from '../src/site/components/VditorOfficial.vue'
import { i18n } from '../src/runtime/i18n'

type Entry = string | { name: string; icon?: string; toolbar?: Entry[] }

const originalMatchMedia = window.matchMedia

function mockViewport(mobile: boolean) {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: mobile && query.includes('520px'),
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

function toolbarOf(props: Record<string, unknown>): Entry[] {
  const wrapper = mount(VditorOfficial, { props: { placeholder: '', modelValue: '', ...props }, global: { plugins: [i18n] } })
  const list = (wrapper.vm as unknown as { resolveToolbar(): Entry[] }).resolveToolbar()
  wrapper.unmount()
  return list
}

const nameOf = (entry: Entry) => (typeof entry === 'string' ? entry : entry.name)
const allNames = (list: Entry[]): string[] => list.flatMap((entry) => [nameOf(entry), ...(typeof entry === 'object' && entry.toolbar ? allNames(entry.toolbar) : [])])

afterEach(() => {
  window.matchMedia = originalMatchMedia
})

describe('桌面端编辑器工具栏（issue #1630）', () => {
  test('长文：按意图分组，去掉开发者/重复主题项，导入按钮位于全屏前', () => {
    mockViewport(false)
    const list = toolbarOf({ outline: true, importMarkdown: true, stickerPicker: true })
    const names = list.map(nameOf)
    expect(names.slice(0, 2)).toEqual(['headings', '|'])
    expect(names.indexOf('import-markdown')).toBe(names.indexOf('fullscreen') - 1)
    // 表情包紧跟表情
    expect(names.indexOf('sticker')).toBe(names.indexOf('emoji') + 1)
    for (const removed of ['devtools', 'info', 'export', 'code-theme', 'content-theme', 'record', 'outline']) {
      expect(allNames(list)).not.toContain(removed)
    }
    const more = list.find((entry) => nameOf(entry) === 'more') as { toolbar: Entry[] }
    // 「更多」菜单分组：插入 | 段落 | 编辑模式（单选组） | 查看与帮助
    expect(more.toolbar.map(nameOf)).toEqual([
      'line', 'math-inline', '|',
      'insert-before', 'insert-after', 'outdent', 'indent', '|',
      'edit-mode', '|',
      'preview', 'both', 'help',
    ])
  })

  test('回复「更多」菜单：标题单选组单独成组；课评 slim 去掉标题时不留空组', () => {
    mockViewport(false)
    const moreOf = (props: Record<string, unknown>) =>
      (toolbarOf(props).find((entry) => nameOf(entry) === 'more') as { toolbar: Entry[] }).toolbar.map(nameOf)
    expect(moreOf({ compact: true }).slice(0, 2)).toEqual(['headings', '|'])
    const slim = moreOf({ compact: true, slimMobile: true })
    expect(slim[0]).not.toBe('|')
    expect(slim).not.toContain('headings')
  })

  test('主行每个按钮都使用同一套 Lucide 描边图标', () => {
    mockViewport(false)
    for (const props of [{ importMarkdown: true, stickerPicker: true }, { simple: true }, { compact: true }]) {
      const buttons = toolbarOf(props).filter((entry) => nameOf(entry) !== '|')
      for (const entry of buttons) {
        expect(typeof entry).toBe('object')
        expect((entry as { icon?: string }).icon).toMatch(/^<svg viewBox="0 0 24 24"[^>]*stroke:currentColor;stroke-width:1\.75/)
      }
    }
  })

  test('短文与回复：插入组（图片/表情）在最前；未开启时不出现导入按钮', () => {
    mockViewport(false)
    expect(toolbarOf({ simple: true }).map(nameOf).slice(0, 3)).toEqual(['upload', 'emoji', '|'])
    expect(toolbarOf({ compact: true }).map(nameOf).slice(0, 3)).toEqual(['emoji', 'upload', '|'])
    expect(allNames(toolbarOf({ compact: true }))).not.toContain('import-markdown')
  })

  test('移动端长文：导入收进 more 首位，不挤占主行', () => {
    mockViewport(true)
    const list = toolbarOf({ importMarkdown: true })
    expect(list.map(nameOf)).not.toContain('import-markdown')
    const more = list.find((entry) => nameOf(entry) === 'more') as { toolbar: Entry[] }
    expect(nameOf(more.toolbar[0])).toBe('import-markdown')
  })

  test('编辑区根节点声明排版档位，默认 post', () => {
    mockViewport(false)
    const article = mount(VditorOfficial, { props: { placeholder: '', modelValue: '', profile: 'article' }, global: { plugins: [i18n] } })
    expect(article.find('.vditor-official').attributes('data-gf-rich-profile')).toBe('article')
    article.unmount()
    const plain = mount(VditorOfficial, { props: { placeholder: '', modelValue: '' }, global: { plugins: [i18n] } })
    expect(plain.find('.vditor-official').attributes('data-gf-rich-profile')).toBe('post')
    plain.unmount()
  })
})
