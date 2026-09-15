// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { i18n } from '../src/runtime/i18n'
import type { StickerItem } from '../packages/client/src/contracts/sticker'

const getForumStickers = vi.fn<() => Promise<StickerItem[]>>()

vi.mock('@/runtime/api', () => ({
  getForumStickers: () => getForumStickers(),
}))

const stickers: StickerItem[] = [
  { name: 'smile', url: '/file/img/stickers/smile.png' },
  { name: 'wave', url: '/file/img/stickers/wave.gif' },
]

// useStickerLibrary 的缓存是模块级状态：每个用例 resetModules 后动态 import，
// 保证各用例从空白缓存开始；vi.mock 注册不受 resetModules 影响。
async function importStickerPicker() {
  vi.resetModules()
  const mod = await import('../src/site/components/StickerPicker.vue')
  return mod.default
}

function mountPicker(StickerPicker: typeof import('../src/site/components/StickerPicker.vue').default, props: Record<string, unknown> = {}) {
  return mount(StickerPicker, {
    props: { open: true, ...props },
    global: { plugins: [i18n] },
  })
}

beforeEach(() => {
  i18n.global.locale.value = 'zh'
  vi.clearAllMocks()
})

describe('StickerPicker 表情包选择面板（MADR 0022）', () => {
  test('打开时拉取启用贴纸并渲染懒加载网格预览', async () => {
    getForumStickers.mockResolvedValue(stickers)
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()

    expect(getForumStickers).toHaveBeenCalledTimes(1)
    const cells = wrapper.findAll('.gf-sticker-cell')
    expect(cells).toHaveLength(2)
    const img = cells[0].find('img')
    expect(img.attributes('src')).toBe('/file/img/stickers/smile.png')
    expect(img.attributes('loading')).toBe('lazy')
    expect(cells[1].attributes('aria-label')).toBe('wave')
    wrapper.unmount()
  })

  test('点击贴纸 emit select(name)，token 由宿主拼装', async () => {
    getForumStickers.mockResolvedValue(stickers)
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()

    await wrapper.findAll('.gf-sticker-cell')[1].trigger('click')
    expect(wrapper.emitted('select')).toEqual([['wave']])
    wrapper.unmount()
  })

  test('模块级缓存全站共享：重复打开不再发请求', async () => {
    getForumStickers.mockResolvedValue(stickers)
    const StickerPicker = await importStickerPicker()
    const first = mountPicker(StickerPicker)
    await flushPromises()
    expect(getForumStickers).toHaveBeenCalledTimes(1)
    first.unmount()

    const second = mountPicker(StickerPicker)
    await flushPromises()
    expect(getForumStickers).toHaveBeenCalledTimes(1)
    expect(second.findAll('.gf-sticker-cell')).toHaveLength(2)
    second.unmount()
  })

  test('加载失败显示错误文案与重试，重试成功后渲染列表', async () => {
    getForumStickers.mockRejectedValueOnce(new Error('offline'))
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()

    expect(wrapper.text()).toContain(i18n.global.t('api.stickersLoadFailed'))
    getForumStickers.mockResolvedValueOnce(stickers)
    await wrapper.find(`button[aria-label="${i18n.global.t('common.retry')}"]`).trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.gf-sticker-cell')).toHaveLength(2)
    wrapper.unmount()
  })

  test('空列表显示空态文案', async () => {
    getForumStickers.mockResolvedValue([])
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()

    expect(wrapper.text()).toContain(i18n.global.t('stickers.empty'))
    wrapper.unmount()
  })

  test('点击面板外 emit close；编辑器（.vditor）内点击不关闭', async () => {
    getForumStickers.mockResolvedValue(stickers)
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()
    await nextTick()

    const editorHost = document.createElement('div')
    editorHost.className = 'vditor'
    document.body.appendChild(editorHost)
    editorHost.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')).toBeUndefined()

    document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')).toHaveLength(1)
    editorHost.remove()
    wrapper.unmount()
  })

  test('关闭后卸载外部点击监听：再次分发不再 emit', async () => {
    getForumStickers.mockResolvedValue(stickers)
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()
    await nextTick()
    await wrapper.setProps({ open: false })
    await nextTick()

    document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')).toBeUndefined()
    wrapper.unmount()
  })
})
