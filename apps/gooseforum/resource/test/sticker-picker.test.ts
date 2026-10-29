// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { i18n } from '../src/runtime/i18n'
import type { StickerItem, StickerMarketPack, StickerMarketPackDetail, StickerMarketPage } from '../packages/client/src/contracts/sticker'

const getForumStickers = vi.fn<() => Promise<StickerItem[]>>()
const getMyStickerPacks = vi.fn<() => Promise<StickerMarketPack[]>>()
const getStickerMarket = vi.fn<(query: unknown) => Promise<StickerMarketPage>>()
const getStickerMarketPack = vi.fn<(id: number) => Promise<StickerMarketPackDetail>>()
const saveMyStickerPack = vi.fn<(id: number) => Promise<StickerMarketPack>>()
const deleteMyStickerPack = vi.fn<(id: number) => Promise<void>>()

vi.mock('@/runtime/api', () => ({
  getForumStickers: () => getForumStickers(),
  getMyStickerPacks: () => getMyStickerPacks(),
  getStickerMarket: (query: unknown) => getStickerMarket(query),
  getStickerMarketPack: (id: number) => getStickerMarketPack(id),
  saveMyStickerPack: (id: number) => saveMyStickerPack(id),
  deleteMyStickerPack: (id: number) => deleteMyStickerPack(id),
}))

const marketPack: StickerMarketPack = { id: 9201, name: 'Pack One', topic: 'anime', stickerCount: 2, coverUrl: '/file/img/stickers/market/a.png' }
const marketStickers: StickerItem[] = [
  { name: 'm_aaaa', url: '/file/img/stickers/market/a.png', displayName: 'Pack One 1', pack: 'market' },
  { name: 'm_bbbb', url: '/file/img/stickers/market/b.png', displayName: 'Pack One 2', pack: 'market' },
]

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
  getMyStickerPacks.mockResolvedValue([])
})

describe('StickerPicker 表情包选择面板（MADR 0030）', () => {
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

  test('已添加的市场表情包显示为页签，点开后插入其中的贴纸', async () => {
    getForumStickers.mockResolvedValue(stickers)
    getMyStickerPacks.mockResolvedValue([marketPack])
    getStickerMarketPack.mockResolvedValue({ pack: marketPack, stickers: marketStickers })
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()

    await wrapper.get('button[aria-label="Pack One"]').trigger('click')
    await flushPromises()
    expect(getStickerMarketPack).toHaveBeenCalledWith(9201)
    const cells = wrapper.findAll('.gf-sticker-cell')
    expect(cells.map(cell => cell.attributes('aria-label'))).toEqual(['Pack One 1', 'Pack One 2'])
    await cells[1].trigger('click')
    expect(wrapper.emitted('select')).toEqual([['m_bbbb']])
    wrapper.unmount()
  })

  test('市场浏览：添加表情包乐观更新，失败时回滚并提示', async () => {
    getForumStickers.mockResolvedValue(stickers)
    getStickerMarket.mockResolvedValue({ topics: [{ id: 'anime', label: '动画' }], packs: [marketPack], total: 1, page: 1, pageSize: 24 })
    saveMyStickerPack.mockRejectedValueOnce(new Error('最多添加 50 个表情包，请先移除一些。'))
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()

    await wrapper.get(`button[aria-label="${i18n.global.t('stickers.openMarket')}"]`).trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('动画')
    const add = () => wrapper.find(`button[aria-label="${i18n.global.t('stickers.addPackLabel', { name: 'Pack One' })}"]`)
    await add().trigger('click')
    await flushPromises()
    expect(saveMyStickerPack).toHaveBeenCalledWith(9201)
    expect(add().exists()).toBe(true)
    expect(wrapper.get('[role="alert"]').text()).toContain('50')

    saveMyStickerPack.mockResolvedValueOnce(marketPack)
    await add().trigger('click')
    await flushPromises()
    expect(wrapper.find(`button[aria-label="${i18n.global.t('stickers.removePackLabel', { name: 'Pack One' })}"]`).exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Pack One"]').exists()).toBe(true)
    wrapper.unmount()
  })

  test('触屏长按显示放大预览，松手不插入；之后单击照常插入', async () => {
    vi.useFakeTimers()
    getForumStickers.mockResolvedValue(stickers)
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()

    const cell = wrapper.findAll('.gf-sticker-cell')[0]
    await cell.trigger('pointerdown', { pointerType: 'touch', clientX: 10, clientY: 10 })
    await vi.advanceTimersByTimeAsync(400)
    expect(wrapper.find('.gf-sticker-preview img').attributes('src')).toBe('/file/img/stickers/smile.png')
    await cell.trigger('pointerup', { pointerType: 'touch' })
    await cell.trigger('click')
    expect(wrapper.find('.gf-sticker-preview').exists()).toBe(false)
    expect(wrapper.emitted('select')).toBeUndefined()

    await cell.trigger('pointerdown', { pointerType: 'touch', clientX: 10, clientY: 10 })
    await cell.trigger('pointerup', { pointerType: 'touch' })
    await cell.trigger('click')
    expect(wrapper.emitted('select')).toEqual([['smile']])
    vi.useRealTimers()
    wrapper.unmount()
  })

  test('滑动滚动会取消长按，pointercancel 关闭预览', async () => {
    vi.useFakeTimers()
    getForumStickers.mockResolvedValue(stickers)
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()

    const grid = wrapper.get('.gf-sticker-picker-grid')
    const cell = wrapper.findAll('.gf-sticker-cell')[0]
    await cell.trigger('pointerdown', { pointerType: 'touch', clientX: 10, clientY: 10 })
    await grid.trigger('pointermove', { pointerType: 'touch', clientX: 10, clientY: 40 })
    await vi.advanceTimersByTimeAsync(400)
    expect(wrapper.find('.gf-sticker-preview').exists()).toBe(false)

    await cell.trigger('pointerdown', { pointerType: 'touch', clientX: 10, clientY: 10 })
    await vi.advanceTimersByTimeAsync(400)
    expect(wrapper.find('.gf-sticker-preview').exists()).toBe(true)
    await grid.trigger('pointercancel', { pointerType: 'touch' })
    expect(wrapper.find('.gf-sticker-preview').exists()).toBe(false)
    vi.useRealTimers()
    wrapper.unmount()
  })

  test('鼠标悬停片刻显示预览，移出网格关闭；Esc 先关预览再关面板', async () => {
    vi.useFakeTimers()
    getForumStickers.mockResolvedValue(stickers)
    const StickerPicker = await importStickerPicker()
    const wrapper = mountPicker(StickerPicker)
    await flushPromises()

    const cells = wrapper.findAll('.gf-sticker-cell')
    await cells[1].trigger('pointerenter', { pointerType: 'mouse' })
    await vi.advanceTimersByTimeAsync(350)
    expect(wrapper.find('.gf-sticker-preview-label').text()).toBe('wave')
    await wrapper.get('.gf-sticker-picker-grid').trigger('pointerleave', { pointerType: 'mouse' })
    expect(wrapper.find('.gf-sticker-preview').exists()).toBe(false)

    await cells[0].trigger('pointerenter', { pointerType: 'mouse' })
    await vi.advanceTimersByTimeAsync(350)
    await wrapper.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('.gf-sticker-preview').exists()).toBe(false)
    expect(wrapper.emitted('close')).toBeUndefined()
    await wrapper.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    expect(wrapper.emitted('close')).toHaveLength(1)
    vi.useRealTimers()
    wrapper.unmount()
  })
})
