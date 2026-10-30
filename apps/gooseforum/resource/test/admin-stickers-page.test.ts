import { inflateRawSync } from 'node:zlib'
// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import StickersManagementPage from '../src/admin/pages/management/StickersManagementPage.vue'
import {
  batchStickers,
  deleteSticker,
  deleteStickerPack,
  getStickerPacks,
  getStickers,
  importStickerPack,
  saveSticker,
  saveStickerPack,
} from '../src/admin/runtime/api'
import { uploadImageFile } from '../src/runtime/api'
import type { AdminPayload, AdminSticker, AdminStickerPack } from '../src/admin/types'

vi.mock('../src/runtime/api', () => ({ uploadImageFile: vi.fn() }))
vi.mock('../src/admin/runtime/api', () => ({
  batchStickers: vi.fn(),
  deleteSticker: vi.fn(),
  deleteStickerPack: vi.fn(),
  getStickerPacks: vi.fn(),
  getStickers: vi.fn(),
  importStickerPack: vi.fn(),
  saveSticker: vi.fn(),
  saveStickerPack: vi.fn(),
}))
vi.mock('../src/admin/runtime/i18n-text', () => ({
  adminText: (key: string, params?: Record<string, unknown>) => key + (params ? JSON.stringify(params) : ''),
}))

const sticker = (overrides: Partial<AdminSticker> = {}): AdminSticker => ({
  id: 1,
  name: 'chiikawa',
  fileName: 'stickers/abc.png',
  url: '/f/stickers/abc.png',
  pack: 'official',
  sortOrder: 1000,
  isEnabled: true,
  createdBy: 7,
  aliases: [],
  ...overrides,
})

const pack = (overrides: Partial<AdminStickerPack> = {}): AdminStickerPack => ({
  key: 'official',
  name: '默认',
  sortOrder: 0,
  coverStickerId: 0,
  coverUrl: '/f/stickers/abc.png',
  stickerCount: 1,
  enabledCount: 1,
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

function seed(stickers: AdminSticker[], packs: AdminStickerPack[] = [pack({ stickerCount: stickers.length, enabledCount: stickers.length })]) {
  vi.mocked(getStickers).mockResolvedValue(stickers)
  vi.mocked(getStickerPacks).mockResolvedValue(packs)
}

function mountPage() {
  // reka-ui Dialog teleports to document.body, so modal content is asserted through DOMWrapper(document.body).
  wrapper = mount(StickersManagementPage, { props: { payload }, attachTo: document.body, global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } } })
  return { page: wrapper, body: new DOMWrapper(document.body) }
}

function buttonWithText(root: { findAll: VueWrapper['findAll'] }, text: string) {
  return root.findAll('button').find((item) => item.text().includes(text))!
}

function cardTitles(page: VueWrapper) {
  return page.findAll('li p.font-medium').map((item) => item.text())
}

function byLabel(root: { findAll: VueWrapper['findAll'] }, label: string) {
  return root.findAll('[aria-label]').find((item) => item.attributes('aria-label') === label)!
}

function pickFiles(input: DOMWrapper<Element>, files: File[]) {
  Object.defineProperty(input.element, 'files', { value: files, configurable: true })
  return input.trigger('change')
}

describe('stickers management page', () => {
  test('groups stickers by pack and filters by pack, keyword and previous name', async () => {
    seed(
      [
        sticker(),
        sticker({ id: 2, name: '悲伤蛙', pack: 'memes', isEnabled: false, aliases: ['sad_frog'] }),
        sticker({ id: 3, name: 'usagi', pack: 'memes' }),
      ],
      [pack(), pack({ key: 'memes', name: '程序员梗', coverUrl: '', stickerCount: 2, enabledCount: 1 })],
    )
    const { page } = mountPage()
    await flushPromises()

    expect(cardTitles(page)).toEqual(['chiikawa', '悲伤蛙', 'usagi'])
    expect(page.text()).toContain('stickerLibrarySummary{"total":3,"packs":2}')
    // The sidebar shows enabled / total when they differ.
    expect(buttonWithText(page, '程序员梗').text()).toContain('1/2')

    await buttonWithText(page, '程序员梗').trigger('click')
    expect(cardTitles(page)).toEqual(['悲伤蛙', 'usagi'])
    expect(page.text()).toContain('stickerPackSummary{"total":2,"enabled":1}')

    await page.get('input[aria-label="stickerSearch"]').setValue('sad')
    expect(cardTitles(page)).toEqual(['悲伤蛙'])

    await page.get('input[aria-label="stickerSearch"]').setValue('nothing')
    expect(page.text()).toContain('stickerEmptyFiltered')
    await buttonWithText(page, 'stickerClearFilters').trigger('click')
    expect(cardTitles(page)).toEqual(['悲伤蛙', 'usagi'])
  })

  test('create dialog uploads the image and saves into the current pack', async () => {
    seed([], [pack({ stickerCount: 0, enabledCount: 0 }), pack({ key: 'memes', name: '程序员梗', stickerCount: 0, enabledCount: 0 })])
    vi.mocked(saveSticker).mockResolvedValue(undefined)
    const { page, body } = mountPage()
    await flushPromises()
    expect(page.text()).toContain('stickerEmptyTitle')

    await buttonWithText(page, '程序员梗').trigger('click')
    await buttonWithText(page, 'k00vgb').trigger('click')
    const form = body.get('form')

    // An empty name and a missing image are both stopped before saving.
    await form.trigger('submit')
    expect(saveSticker).not.toHaveBeenCalled()
    await form.findAll('input').find((input) => !input.attributes('type'))!.setValue('chiikawa')
    await form.trigger('submit')
    expect(saveSticker).not.toHaveBeenCalled()

    const image = new File(['png'], 'smile.png', { type: 'image/png' })
    await pickFiles(body.get('input[type="file"][accept="image/*"]'), [image])
    vi.mocked(uploadImageFile).mockResolvedValue('/file/img/owned.png')
    await form.trigger('submit')
    await flushPromises()
    expect(uploadImageFile).toHaveBeenCalledWith(image, 'smile.png')
    expect(saveSticker).toHaveBeenCalledWith({ id: 0, name: 'chiikawa', fileName: '/file/img/owned.png', pack: 'memes', sortOrder: 1000, isEnabled: true })
    expect(getStickers).toHaveBeenCalledTimes(2)
  })

  test('editing keeps the pack and shows previous names', async () => {
    seed([sticker({ aliases: ['old_smile'] })])
    vi.mocked(saveSticker).mockRejectedValue(Object.assign(new Error('reserved'), { messageCode: 'admin.sticker.nameReserved' }))
    const { page, body } = mountPage()
    await flushPromises()

    await byLabel(page, 'stickerEditOne{"name":"chiikawa"}').trigger('click')
    expect(body.text()).toContain('stickerRenameHint')
    expect(body.text()).toContain('stickerAliases{"names":"old_smile"}')
    await body.get('form').trigger('submit')
    await flushPromises()
    expect(saveSticker).toHaveBeenCalledWith({ id: 1, name: 'chiikawa', pack: 'official', sortOrder: 1000, isEnabled: true })
  })

  test('card switch toggles enabled state and the edit dialog deletes', async () => {
    seed([sticker({ id: 9, name: 'usagi', isEnabled: false })])
    vi.mocked(saveSticker).mockResolvedValue(undefined)
    vi.mocked(deleteSticker).mockResolvedValue(undefined)
    const { page, body } = mountPage()
    await flushPromises()

    await page.get('[role="switch"]').trigger('click')
    expect(saveSticker).toHaveBeenCalledWith({ id: 9, name: 'usagi', pack: 'official', sortOrder: 1000, isEnabled: true })

    await byLabel(page, 'stickerEditOne{"name":"usagi"}').trigger('click')
    await buttonWithText(body, 'k005i').trigger('click')
    await body.get('[data-testid="admin-confirm"]').trigger('click')
    expect(deleteSticker).toHaveBeenCalledWith(9)
  })

  test('selection drives the batch toolbar', async () => {
    seed([sticker(), sticker({ id: 2, name: 'usagi' }), sticker({ id: 3, name: 'hachiware' })])
    vi.mocked(batchStickers).mockResolvedValue({ changed: 2 })
    const { page, body } = mountPage()
    await flushPromises()
    expect(page.find('[role="toolbar"]').exists()).toBe(false)

    await byLabel(page, 'stickerSelectOne{"name":"chiikawa"}').trigger('click')
    await byLabel(page, 'stickerSelectOne{"name":"usagi"}').trigger('click')
    const toolbar = page.get('[role="toolbar"]')
    expect(toolbar.text()).toContain('stickerSelectedCount{"count":2}')

    await buttonWithText(toolbar, 'stickerBatchDisable').trigger('click')
    await flushPromises()
    expect(batchStickers).toHaveBeenCalledWith({ ids: [1, 2], action: 'disable' })

    await buttonWithText(page.get('[role="toolbar"]'), 'stickerBatchDelete').trigger('click')
    expect(body.text()).toContain('stickerBatchDeleteTitle{"count":2}')
    await body.get('[data-testid="admin-confirm"]').trigger('click')
    await flushPromises()
    expect(batchStickers).toHaveBeenLastCalledWith({ ids: [1, 2], action: 'delete' })
    expect(page.find('[role="toolbar"]').exists()).toBe(false)
  })

  test('creating a pack continues into the upload queue for that pack', async () => {
    seed([sticker()])
    vi.mocked(saveStickerPack).mockResolvedValue({ key: 'p-1a2b3c4d' })
    vi.mocked(importStickerPack).mockResolvedValue({ imported: 3, skipped: 1, failed: [{ name: 'bad.png', reason: 'invalidImage' }] })
    const { page, body } = mountPage()
    await flushPromises()

    await page.get('button[aria-label="stickerPackCreate"]').trigger('click')
    const form = body.get('form')
    await form.trigger('submit')
    expect(saveStickerPack).not.toHaveBeenCalled()

    vi.mocked(getStickerPacks).mockResolvedValue([pack(), pack({ key: 'p-1a2b3c4d', name: '校园', sortOrder: 10, stickerCount: 0, enabledCount: 0 })])
    await body.get('input[maxlength="64"]').setValue('  校园 ')
    await form.trigger('submit')
    await flushPromises()
    expect(saveStickerPack).toHaveBeenCalledWith({ name: '校园', sortOrder: 10 })
    expect(body.text()).toContain('stickerUploadTitle')

    const archive = new File(['zip'], 'pack.zip', { type: 'application/zip' })
    const image = new File(['png'], 'one.png', { type: 'image/png' })
    await pickFiles(body.get('[data-testid="sticker-upload-archives"]'), [archive])
    await pickFiles(body.get('[data-testid="sticker-upload-images"]'), [image])
    await buttonWithText(body, 'stickerUploadStart').trigger('click')
    await flushPromises()

    expect(importStickerPack).toHaveBeenCalledTimes(2)
    expect(importStickerPack).toHaveBeenCalledWith(archive, 'p-1a2b3c4d')
    expect(importStickerPack).toHaveBeenCalledWith(image, 'p-1a2b3c4d')
    expect(body.text()).toContain('stickerUploadSummary{"imported":6,"skipped":2,"failed":2}')
    expect(body.text()).toContain('bad.png')
    expect(body.text()).toContain('k00vgw')
  })

  test('partial ZIP retry sends only failed entries and retains successful totals', async () => {
    seed([sticker()])
    vi.mocked(importStickerPack)
      .mockResolvedValueOnce({ imported: 1, skipped: 0, failed: [{ name: 'nested/bad.png', reason: 'saveFailed' }] })
      .mockResolvedValueOnce({ imported: 1, skipped: 0, failed: [] })
    const { page, body } = mountPage()
    await flushPromises()
    await buttonWithText(page, 'stickerUploadAction').trigger('click')
    const archive = new File([Uint8Array.from(atob('UEsDBBQAAAAIABGWXl2QAxiDBQAAAAMAAAANAAAAb3RoZXIvYmFkLnBuZyvISwcAUEsDBBQAAAAIABGWXl2QAxiDBQAAAAMAAAAOAAAAbmVzdGVkL2JhZC5wbmcryEsHAFBLAQIUAxQAAAAIABGWXl2QAxiDBQAAAAMAAAANAAAAAAAAAAAAAACAAQAAAABvdGhlci9iYWQucG5nUEsBAhQDFAAAAAgAEZZeXZADGIMFAAAAAwAAAA4AAAAAAAAAAAAAAIABMAAAAG5lc3RlZC9iYWQucG5nUEsFBgAAAAACAAIAdwAAAGEAAAAAAA=='), c => c.charCodeAt(0))], 'pack.zip', { type: 'application/zip' })
    await pickFiles(body.get('[data-testid="sticker-upload-archives"]'), [archive])
    await buttonWithText(body, 'stickerUploadStart').trigger('click')
    await flushPromises()
    await buttonWithText(body, 'stickerUploadRetry').trigger('click')
    await flushPromises()
    expect(importStickerPack).toHaveBeenCalledTimes(2)
    const retry = vi.mocked(importStickerPack).mock.calls[1][0]
    const bytes = new Uint8Array(await retry.arrayBuffer())
    const text = new TextDecoder().decode(bytes)
    expect(text).toContain('nested/bad.png')
    expect(text).not.toContain('other/bad.png')
    const view = new DataView(bytes.buffer)
    const central = view.getUint32(bytes.length - 6, true)
    expect(view.getUint32(central, true)).toBe(0x02014b50)
    expect(view.getUint32(central + 42, true)).toBe(0)
    expect(view.getUint16(bytes.length - 12, true)).toBe(1)
    const dataStart = 30 + view.getUint16(26, true) + view.getUint16(28, true)
    expect(inflateRawSync(bytes.subarray(dataStart, dataStart + view.getUint32(18, true))).toString()).toBe('png')
    expect(body.text()).toContain('stickerUploadSummary{"imported":2,"skipped":0,"failed":0}')
  })

  test('the upload queue rejects unsupported and oversized files before uploading', async () => {
    seed([sticker()])
    const { page, body } = mountPage()
    await flushPromises()

    await buttonWithText(page, 'stickerUploadAction').trigger('click')
    const big = new File([new Uint8Array((4 << 20) + 1)], 'huge.png', { type: 'image/png' })
    const text = new File(['hello'], 'notes.txt', { type: 'text/plain' })
    await pickFiles(body.get('[data-testid="sticker-upload-images"]'), [big, text])

    expect(body.text()).toContain('stickerUploadImageTooLarge')
    expect(body.text()).toContain('stickerUploadUnsupported')
    expect(buttonWithText(body, 'stickerUploadStart').attributes('disabled')).toBeDefined()
    expect(importStickerPack).not.toHaveBeenCalled()
  })

  test('deleting a pack moves its stickers to the default pack', async () => {
    seed(
      [sticker(), sticker({ id: 2, name: 'usagi', pack: 'memes' })],
      [pack(), pack({ key: 'memes', name: '程序员梗', stickerCount: 1, enabledCount: 1 })],
    )
    vi.mocked(deleteStickerPack).mockResolvedValue({ moved: 1 })
    const { page, body } = mountPage()
    await flushPromises()

    // The default pack cannot be deleted.
    await buttonWithText(page, '默认').trigger('click')
    expect(page.findAll('button').some((item) => item.text().includes('stickerPackDelete'))).toBe(false)

    await buttonWithText(page, '程序员梗').trigger('click')
    await buttonWithText(page, 'stickerPackDelete').trigger('click')
    expect(body.text()).toContain('stickerPackDeleteDescription{"count":1}')
    await buttonWithText(body, 'stickerPackDeleteConfirm').trigger('click')
    await flushPromises()
    expect(deleteStickerPack).toHaveBeenCalledWith('memes', 'official')
  })
})
