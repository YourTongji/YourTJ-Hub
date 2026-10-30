// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import StickerMarketManagementPage from '../src/admin/pages/management/StickerMarketManagementPage.vue'
import {
  createStickerMarketPack,
  getStickerMarket,
  getStickerMarketPack,
  importStickerMarketPack,
  saveStickerMarketItem,
  saveStickerMarketPack,
  syncStickerMarket,
  uploadStickerMarketFile,
} from '../src/admin/runtime/api'
import { adminToast } from '../src/admin/runtime/toast'
import type { AdminPayload, AdminStickerMarketPack, AdminStickerMarketPage } from '../src/admin/types'

vi.mock('../src/admin/runtime/api', () => ({
  createStickerMarketPack: vi.fn(),
  getStickerMarket: vi.fn(),
  getStickerMarketPack: vi.fn(),
  importStickerMarketPack: vi.fn(),
  saveStickerMarketItem: vi.fn(),
  saveStickerMarketPack: vi.fn(),
  syncStickerMarket: vi.fn(),
  uploadStickerMarketFile: vi.fn(),
}))
vi.mock('../src/admin/runtime/toast', () => ({
  adminToast: { success: vi.fn(), error: vi.fn(), info: vi.fn(), warning: vi.fn() },
}))
vi.mock('../src/admin/runtime/i18n-text', () => ({
  adminText: (key: string, params?: Record<string, unknown>) => key + (params ? JSON.stringify(params) : ''),
}))

const pack = (overrides: Partial<AdminStickerMarketPack> = {}): AdminStickerMarketPack => ({
  id: 9201,
  upstreamId: 'group-1',
  source: 'fluxdo',
  name: 'Pack One',
  topic: 'anime',
  upstreamCount: 3,
  archived: false,
  missing: false,
  status: 'partial',
  published: true,
  sortOrder: 0,
  itemCount: 2,
  failedCount: 1,
  importTotal: 3,
  lastError: '',
  coverUrl: '/file/img/stickers/market/a.png',
  iconUrl: '',
  iconText: '',
  syncedAt: '2026-10-29T08:00:00Z',
  importedAt: '2026-10-29T08:05:00Z',
  ...overrides,
})

const marketPage = (packs: AdminStickerMarketPack[]): AdminStickerMarketPage => ({
  enabled: true,
  source: 'fluxdo',
  lastSyncedAt: '2026-10-29T08:00:00Z',
  topics: [{ id: 'anime', label: '动画' }],
  packs,
  total: packs.length,
  page: 1,
  pageSize: 20,
})

const payload = { component: 'manage-home', props: {}, meta: { title: 'admin' }, layout: {}, url: '/admin/sticker-market', version: 'test' } as unknown as AdminPayload

let wrapper: VueWrapper | undefined
afterEach(() => {
  wrapper?.unmount()
  document.body.innerHTML = ''
  vi.useRealTimers()
  vi.resetAllMocks()
})

function mountPage() {
  wrapper = mount(StickerMarketManagementPage, { props: { payload }, attachTo: document.body })
  return { page: wrapper, body: new DOMWrapper(document.body) }
}

function buttonWithText(text: string) {
  return wrapper!.findAll('button').find(button => button.text().includes(text))!
}

describe('sticker market management page', () => {
  test('lists packs with their import state and topic label', async () => {
    vi.mocked(getStickerMarket).mockResolvedValue(marketPage([pack(), pack({ id: 9202, upstreamId: 'group-2', name: 'Pack Two', status: 'new', published: false, itemCount: 0, failedCount: 0, coverUrl: '', importedAt: null })]))
    const { page } = mountPage()
    await flushPromises()

    const rows = page.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('Pack One')
    expect(rows[0].text()).toContain('动画')
    expect(rows[0].text()).toContain('stickerMarketStatusPartial')
    expect(rows[0].text()).toContain('stickerMarketFailedCount{"count":1}')
    // A pack with no imported stickers cannot be published yet.
    const publish = rows[1].get('[role="switch"]')
    expect(publish.attributes('disabled')).toBeDefined()
    expect(rows[1].text()).toContain('stickerMarketImport')
  })

  test('import queues the pack and shows live progress until the task finishes', async () => {
    vi.useFakeTimers()
    const fresh = pack({ id: 9202, status: 'new', published: false, itemCount: 0, failedCount: 0, importTotal: 0 })
    vi.mocked(getStickerMarket).mockResolvedValueOnce(marketPage([fresh]))
    vi.mocked(importStickerMarketPack).mockResolvedValue({ ...fresh, status: 'queued' })
    const { page } = mountPage()
    await flushPromises()

    await buttonWithText('stickerMarketImport').trigger('click')
    await flushPromises()
    expect(importStickerMarketPack).toHaveBeenCalledWith(9202)
    expect(adminToast.success).toHaveBeenCalledWith('stickerMarketImportQueued')
    expect(page.text()).toContain('stickerMarketStatusQueued')
    expect(page.text()).toContain('stickerMarketWaiting')

    vi.mocked(getStickerMarket).mockResolvedValueOnce(marketPage([{ ...fresh, status: 'importing', itemCount: 1, failedCount: 1, importTotal: 4 }]))
    await vi.advanceTimersByTimeAsync(1500)
    await flushPromises()
    const bar = page.get('[role="progressbar"]')
    expect(bar.attributes('aria-valuenow')).toBe('2')
    expect(bar.attributes('aria-valuemax')).toBe('4')
    expect(bar.attributes('aria-label')).toBe('stickerMarketProgressLabel{"name":"Pack One"}')
    expect(page.text()).toContain('stickerMarketProgress{"done":2,"total":4}')

    vi.mocked(getStickerMarket).mockResolvedValueOnce(marketPage([{ ...fresh, status: 'ready', itemCount: 2, failedCount: 0, importTotal: 2 }]))
    await vi.advanceTimersByTimeAsync(1500)
    await flushPromises()
    expect(getStickerMarket).toHaveBeenCalledTimes(3)
    expect(page.text()).toContain('stickerMarketStatusReady')
    expect(page.find('[role="progressbar"]').exists()).toBe(false)
  })

  test('shows a server-made cover before import and falls back when it fails', async () => {
    vi.mocked(getStickerMarket).mockResolvedValue(marketPage([
      pack({ id: 9203, status: 'new', coverUrl: '', iconUrl: '/api/admin/sticker-market-cover?id=9203' }),
      pack({ id: 9204, status: 'new', coverUrl: '', iconUrl: '/api/admin/sticker-market-cover?id=9204', iconText: '🌿' }),
      pack({ id: 9205, coverUrl: '/file/img/stickers/market/b.png', iconUrl: '/file/img/stickers/market/icon.png' }),
    ]))
    const { page } = mountPage()
    await flushPromises()
    // Emoji text covers render through Twemoji; only the cover images carry the size class.
    const covers = () => page.findAll('tbody img.size-10').map((image) => image.attributes('src'))
    expect(covers()).toEqual(['/api/admin/sticker-market-cover?id=9203', '/api/admin/sticker-market-cover?id=9204', '/file/img/stickers/market/b.png'])

    await page.findAll('tbody img.size-10')[1].trigger('error')
    expect(covers()).toEqual(['/api/admin/sticker-market-cover?id=9203', '/file/img/stickers/market/b.png'])
    expect(page.find('tbody img[src="/twemoji/72x72/1f33f.png"]').exists()).toBe(true)

    await page.findAll('tbody img.size-10')[0].trigger('error')
    expect(covers()).toEqual(['/file/img/stickers/market/b.png'])
  })

  test('publishing saves the pack and confirms the change', async () => {
    vi.mocked(getStickerMarket).mockResolvedValue(marketPage([pack({ published: false })]))
    vi.mocked(saveStickerMarketPack).mockResolvedValue(pack({ published: true }))
    const { page } = mountPage()
    await flushPromises()

    await page.get('tbody [role="switch"]').trigger('click')
    await flushPromises()
    expect(saveStickerMarketPack).toHaveBeenCalledWith({ id: 9201, published: true, sortOrder: 0 })
    expect(adminToast.success).toHaveBeenCalledWith('stickerMarketPublished{"name":"Pack One"}')
  })

  test('sync reports packs that left the upstream catalog and localizes failures', async () => {
    vi.mocked(getStickerMarket).mockResolvedValue(marketPage([]))
    vi.mocked(syncStickerMarket).mockResolvedValueOnce({ topics: 1, packs: 2, missing: 1 })
    mountPage()
    await flushPromises()

    await buttonWithText('stickerMarketSync').trigger('click')
    await flushPromises()
    expect(adminToast.success).toHaveBeenCalledWith('stickerMarketSyncedWithMissing{"packs":2,"missing":1}')

    vi.mocked(syncStickerMarket).mockRejectedValueOnce(Object.assign(new Error('x'), { messageCode: 'admin.sticker.market.unavailable' }))
    await buttonWithText('stickerMarketSync').trigger('click')
    await flushPromises()
    expect(adminToast.warning).toHaveBeenCalledWith('stickerMarketUnavailable')
  })

  test('withdrawing one sticker from the pack detail', async () => {
    vi.mocked(getStickerMarket).mockResolvedValue(marketPage([pack()]))
    vi.mocked(getStickerMarketPack).mockResolvedValue({
      pack: pack(),
      items: [
        { upstreamId: 'e1', stickerId: 9301, name: 'm_a', aliases: [], url: '/a.png', isEnabled: true, error: '' },
        { upstreamId: 'e3', stickerId: 0, name: '', aliases: [], url: '', isEnabled: false, error: 'blocked' },
      ],
    })
    vi.mocked(saveStickerMarketItem).mockResolvedValue(true)
    const { page, body } = mountPage()
    await flushPromises()

    await page.findAll('button').find(button => button.attributes('title') === 'stickerMarketViewStickers')!.trigger('click')
    await flushPromises()
    expect(body.text()).toContain('stickerMarketItemNotImported{"reason":"stickerMarketReasonBlocked"}')
    await body.get('[role="dialog"] [role="switch"]').trigger('click')
    await flushPromises()
    expect(saveStickerMarketItem).toHaveBeenCalledWith({ stickerId: 9301, isEnabled: false })
    expect(adminToast.success).toHaveBeenCalledWith('stickerMarketItemWithdrawn')
  })

  test('renaming a sticker inline keeps the generated name as a previous name', async () => {
    const generated = 'm_' + 'a'.repeat(40)
    vi.mocked(getStickerMarket).mockResolvedValue(marketPage([pack()]))
    vi.mocked(getStickerMarketPack).mockResolvedValue({
      pack: pack(),
      items: [{ upstreamId: 'e1', stickerId: 9301, name: generated, aliases: [], url: '/a.png', isEnabled: true, error: '' }],
    })
    vi.mocked(saveStickerMarketItem)
      .mockRejectedValueOnce(Object.assign(new Error('x'), { messageCode: 'admin.sticker.nameExists' }))
      .mockResolvedValueOnce(true)
    const { page, body } = mountPage()
    await flushPromises()
    await page.findAll('button').find(button => button.attributes('title') === 'stickerMarketViewStickers')!.trigger('click')
    await flushPromises()

    await body.get(`[role="dialog"] button[title="${generated}"]`).trigger('click')
    await flushPromises()
    const field = body.get<HTMLInputElement>('#sticker-market-name-9301')
    // A generated token is not a useful starting point; the field starts empty.
    expect(field.element.value).toBe('')
    expect(document.activeElement).toBe(field.element)

    await field.setValue('happy_cat')
    await body.get('[role="dialog"] form').trigger('submit')
    await flushPromises()
    expect(saveStickerMarketItem).toHaveBeenLastCalledWith({ stickerId: 9301, name: 'happy_cat', isEnabled: true })
    expect(body.get('[role="alert"]').text()).toBe('stickerMarketNameExists')
    expect(field.attributes('aria-invalid')).toBe('true')
    expect(adminToast.error).not.toHaveBeenCalled()

    await body.get('[role="dialog"] form').trigger('submit')
    await flushPromises()
    expect(adminToast.success).toHaveBeenCalledWith('stickerMarketRenamed{"name":"happy_cat"}')
    expect(body.find('#sticker-market-name-9301').exists()).toBe(false)
    expect(body.get('[role="dialog"] button[title="happy_cat"]').exists()).toBe(true)
    expect(body.text()).toContain('stickerMarketPreviousNames')
  })

  test('an uploaded pack offers uploads and info editing instead of import', async () => {
    const uploaded = pack({ id: 9401, source: 'upload', upstreamId: 'u0123456789abcdef', name: '校园日常', topic: '', status: 'ready', itemCount: 4, failedCount: 0 })
    vi.mocked(getStickerMarket).mockResolvedValue(marketPage([uploaded]))
    vi.mocked(saveStickerMarketPack).mockResolvedValue({ ...uploaded, name: '校园', topic: 'u-abc' })
    const { page, body } = mountPage()
    await flushPromises()

    const row = page.get('tbody tr')
    expect(row.text()).toContain('stickerMarketSourceUpload')
    expect(row.text()).toContain('stickerMarketUploadedCount{"count":4}')
    expect(row.text()).not.toContain('u0123456789abcdef')
    expect(row.text()).not.toContain('stickerMarketImport')
    expect(row.text()).toContain('stickerMarketUploadImages')

    await row.findAll('button').find((button) => button.attributes('title') === 'stickerMarketEditInfo')!.trigger('click')
    const form = body.get('[role="dialog"] form')
    // Without a topic the picker starts on "new category", which needs a label.
    await form.trigger('submit')
    expect(adminToast.warning).toHaveBeenCalledWith('stickerMarketTopicRequired')
    expect(saveStickerMarketPack).not.toHaveBeenCalled()

    const inputs = form.findAll('input')
    await inputs[0].setValue('校园')
    await inputs[1].setValue('校园生活')
    await form.trigger('submit')
    await flushPromises()
    expect(saveStickerMarketPack).toHaveBeenCalledWith({ id: 9401, published: true, sortOrder: 0, name: '校园', topicLabel: '校园生活' })
  })

  test('creating an uploaded pack continues into uploads and can publish from there', async () => {
    const created = pack({ id: 9402, source: 'upload', upstreamId: 'u1', name: '新包', status: 'new', published: false, itemCount: 0, failedCount: 0, coverUrl: '' })
    vi.mocked(getStickerMarket).mockResolvedValue(marketPage([]))
    vi.mocked(createStickerMarketPack).mockResolvedValue(created)
    vi.mocked(uploadStickerMarketFile).mockResolvedValue({ result: { imported: 2, skipped: 0, failed: [] }, pack: { ...created, status: 'ready', itemCount: 2 } })
    vi.mocked(saveStickerMarketPack).mockResolvedValue({ ...created, status: 'ready', itemCount: 2, published: true })
    const { body } = mountPage()
    await flushPromises()

    await buttonWithText('stickerMarketUploadPack').trigger('click')
    const form = body.get('[role="dialog"] form')
    await form.trigger('submit')
    expect(adminToast.warning).toHaveBeenCalledWith('stickerPackNameRequired')
    expect(createStickerMarketPack).not.toHaveBeenCalled()

    await form.get('input').setValue('  新包 ')
    await form.trigger('submit')
    await flushPromises()
    expect(createStickerMarketPack).toHaveBeenCalledWith({ name: '新包', topic: 'anime' })
    expect(body.text()).toContain('stickerMarketUploadTitle{"name":"新包"}')
    expect(body.text()).toContain('stickerMarketUploadHidden')
    const publish = body.findAll('[role="dialog"] button').find((button) => button.text().includes('stickerMarketPublishNow'))!
    expect(publish.attributes('disabled')).toBeDefined()

    const archive = new File(['zip'], 'set.zip', { type: 'application/zip' })
    const input = body.get('[data-testid="sticker-upload-archives"]')
    Object.defineProperty(input.element, 'files', { value: [archive], configurable: true })
    await input.trigger('change')
    await body.findAll('[role="dialog"] button').find((button) => button.text().includes('stickerUploadStart'))!.trigger('click')
    await flushPromises()
    expect(uploadStickerMarketFile).toHaveBeenCalledWith(9402, archive)
    expect(body.text()).toContain('stickerMarketUploadedCount{"count":2}')

    await body.findAll('[role="dialog"] button').find((button) => button.text().includes('stickerMarketPublishNow'))!.trigger('click')
    await flushPromises()
    expect(saveStickerMarketPack).toHaveBeenCalledWith({ id: 9402, published: true, sortOrder: 0 })
    expect(adminToast.success).toHaveBeenCalledWith('stickerMarketPublished{"name":"新包"}')
  })
})
