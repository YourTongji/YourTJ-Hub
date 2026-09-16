// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import StickersManagementPage from '../src/admin/pages/management/StickersManagementPage.vue'
import { deleteSticker, getStickers, importStickerPack, saveSticker } from '../src/admin/runtime/api'
import { uploadImageFile } from '../src/runtime/api'
import type { AdminPayload, AdminSticker } from '../src/admin/types'

vi.mock('../src/runtime/api', () => ({ uploadImageFile: vi.fn() }))
vi.mock('../src/admin/runtime/api', () => ({
  deleteSticker: vi.fn(),
  getStickers: vi.fn(),
  importStickerPack: vi.fn(),
  saveSticker: vi.fn(),
}))
vi.mock('../src/admin/runtime/i18n-text', () => ({
  adminText: (key: string, params?: Record<string, unknown>) => key + (params ? JSON.stringify(params) : ''),
}))

const sticker = (overrides: Partial<AdminSticker> = {}): AdminSticker => ({
  id: 1,
  name: 'chiikawa',
  fileName: 'stickers/abc.png',
  url: '/f/stickers/abc.png',
  sortOrder: 1000,
  isEnabled: true,
  createdBy: 7,
  ...overrides,
})

const payload = {
  component: 'manage-home',
  props: {},
  meta: { title: 'admin' },
  layout: {},
  url: '/admin/stickers',
  version: 'test',
} as unknown as AdminPayload

let wrapper: VueWrapper | undefined
afterEach(() => {
  wrapper?.unmount()
  document.body.innerHTML = ''
  vi.resetAllMocks()
})

function mountPage() {
  // reka-ui Dialog 通过 DialogPortal teleport 到 document.body；
  // 模态内容需用 DOMWrapper(document.body) 断言。
  wrapper = mount(StickersManagementPage, { props: { payload }, attachTo: document.body })
  return { page: wrapper, body: new DOMWrapper(document.body) }
}

async function clickHeaderButton(text: string) {
  const button = wrapper!.findAll('button').find(item => item.text().includes(text))
  await button!.trigger('click')
}

describe('stickers management page', () => {
  test('renders sticker rows after loading', async () => {
    vi.mocked(getStickers).mockResolvedValue([
      sticker(),
      sticker({ id: 2, name: '悲伤蛙', sortOrder: 20, isEnabled: false }),
    ])
    const { page } = mountPage()
    await flushPromises()

    const rows = page.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('chiikawa')
    expect(rows[1].text()).toContain('悲伤蛙')
    expect(page.get('img[src="/f/stickers/abc.png"]').attributes('alt')).toBe('chiikawa')
    // 启用状态开关按行渲染（编辑/导入两个模态均未打开）
    expect(page.findAll('[role="switch"]')).toHaveLength(2)
  })

  test('create dialog submits with id=0 and reloads', async () => {
    vi.mocked(getStickers).mockResolvedValue([])
    vi.mocked(saveSticker).mockResolvedValue(undefined)
    const { page, body } = mountPage()
    await flushPromises()
    expect(page.text()).toContain('k002v')

    await clickHeaderButton('k00vgb')
    const form = body.get('form')

    // 空名称被拦截，不触发保存
    await form.trigger('submit')
    expect(saveSticker).not.toHaveBeenCalled()

    const nameInput = body.findAll('input').find(input => !input.attributes('type') || input.attributes('type') === 'text')
    await nameInput!.setValue('chiikawa')
    await form.trigger('submit')

    expect(saveSticker).not.toHaveBeenCalled()
    const image = new File(['png'], 'smile.png', { type: 'image/png' })
    const input = body.get('input[type="file"][accept="image/*"]')
    Object.defineProperty(input.element, 'files', { value: [image], configurable: true })
    await input.trigger('change')
    vi.mocked(uploadImageFile).mockResolvedValue('/file/img/owned.png')
    await form.trigger('submit')
    await flushPromises()
    expect(uploadImageFile).toHaveBeenCalledWith(image, 'smile.png')
    expect(saveSticker).toHaveBeenCalledTimes(1)
    expect(saveSticker).toHaveBeenCalledWith({ id: 0, name: 'chiikawa', fileName: '/file/img/owned.png', sortOrder: 1000, isEnabled: true })
    expect(getStickers).toHaveBeenCalledTimes(2)
  })

  test('row switch toggles enabled state via save', async () => {
    vi.mocked(getStickers).mockResolvedValue([sticker({ id: 9, name: 'usagi', isEnabled: false })])
    vi.mocked(saveSticker).mockResolvedValue(undefined)
    const { page } = mountPage()
    await flushPromises()

    await page.get('[role="switch"]').trigger('click')
    expect(saveSticker).toHaveBeenCalledWith({ id: 9, name: 'usagi', sortOrder: 1000, isEnabled: true })
  })

  test('delete confirm flow calls deleteSticker', async () => {
    vi.mocked(getStickers).mockResolvedValue([sticker()])
    vi.mocked(deleteSticker).mockResolvedValue(undefined)
    const { page, body } = mountPage()
    await flushPromises()

    const dangerButton = page.findAll('button').find(button => button.attributes('title') === 'k005i')
    await dangerButton!.trigger('click')
    await body.get('[data-testid="admin-confirm"]').trigger('click')

    expect(deleteSticker).toHaveBeenCalledWith(1)
  })

  test('import report renders summary and expandable failures', async () => {
    vi.mocked(getStickers).mockResolvedValue([])
    vi.mocked(importStickerPack).mockResolvedValue({
      imported: 3,
      skipped: 1,
      failed: [{ name: 'bad.txt', reason: 'invalidImage' }, { name: 'oops.png', reason: 'tooLarge' }],
    })
    const { body } = mountPage()
    await flushPromises()

    await clickHeaderButton('k00vgi')
    const importButtons = () => body.findAll('button').filter(button => button.text().includes('k00uh'))

    // 无文件时先拦截
    await importButtons()[0].trigger('click')
    expect(importStickerPack).not.toHaveBeenCalled()

    const fileInput = body.get('input[type="file"]')
    const file = new File(['zip'], 'pack.zip', { type: 'application/zip' })
    Object.defineProperty(fileInput.element, 'files', { value: [file], configurable: true })
    await fileInput.trigger('change')
    await importButtons()[0].trigger('click')
    await flushPromises()

    expect(importStickerPack).toHaveBeenCalledWith(file)
    expect(body.text()).toContain('"imported":3,"skipped":1,"failed":2')
    expect(body.text()).toContain('bad.txt')
    expect(body.text()).toContain('k00vgw')
  })
})
