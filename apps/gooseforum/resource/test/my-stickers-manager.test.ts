// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import type { StickerItem } from '../packages/client/src/contracts/sticker'

const getMyStickers = vi.fn<() => Promise<StickerItem[]>>()
const saveMyStickerEntry = vi.fn<(request: { stickerName?: string, fileName?: string, displayName?: string }) => Promise<StickerItem>>()
const deleteMySticker = vi.fn<(name: string) => Promise<void>>()
const orderMyStickers = vi.fn<(names: string[]) => Promise<void>>()
const uploadImageFile = vi.fn<(file: Blob, name: string) => Promise<string>>()

vi.mock('@/runtime/api', () => ({
  ApiResponseError: class ApiResponseError extends Error {},
  getMyStickers: () => getMyStickers(),
  saveMyStickerEntry: (request: { stickerName?: string, fileName?: string, displayName?: string }) => saveMyStickerEntry(request),
  deleteMySticker: (name: string) => deleteMySticker(name),
  orderMyStickers: (names: string[]) => orderMyStickers(names),
  uploadImageFile: (file: Blob, name: string) => uploadImageFile(file, name),
}))

const smile: StickerItem = { name: 'smile', url: '/smile.png', displayName: '微笑' }
const wave: StickerItem = { name: 'wave', url: '/wave.png' }
const pending: StickerItem = { name: 'u_' + 'b'.repeat(48), url: '/u.png', isOfficial: false, isEnabled: false, reviewStatus: 'pending' }

// Module-level cache: every case starts from a fresh import.
async function mountManager() {
  vi.resetModules()
  const Manager = (await import('../src/site/components/MyStickersManager.vue')).default
  const wrapper = mount(Manager, { global: { plugins: [i18n] }, attachTo: document.body })
  await flushPromises()
  return wrapper
}

const t = (key: string, params?: Record<string, unknown>) => i18n.global.t(key, params ?? {})

function labels(wrapper: Awaited<ReturnType<typeof mountManager>>) {
  return wrapper.findAll('[data-test="my-sticker-list"] li').map((item) => item.find('p').text())
}

function byLabel(wrapper: Awaited<ReturnType<typeof mountManager>>, label: string) {
  return wrapper.findAll('button').find((button) => button.attributes('aria-label') === label)!
}

beforeEach(() => {
  i18n.global.locale.value = 'zh'
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('MyStickersManager', () => {
  test('lists the library with review state and an empty state when nothing is saved', async () => {
    getMyStickers.mockResolvedValueOnce([])
    const empty = await mountManager()
    expect(empty.text()).toContain(t('stickers.mine.emptyTitle'))
    empty.unmount()

    getMyStickers.mockResolvedValueOnce([smile, wave, pending])
    const wrapper = await mountManager()
    expect(labels(wrapper)).toEqual(['微笑', 'wave', t('stickers.custom')])
    expect(wrapper.text()).toContain(t('stickers.mine.pending'))
    expect(wrapper.text()).toContain(t('stickers.mine.count', { count: 3, limit: 200 }))
    wrapper.unmount()
  })

  test('uploads images one by one and reports files it cannot take', async () => {
    getMyStickers.mockResolvedValue([smile])
    uploadImageFile.mockResolvedValue('/file/img/owned.png')
    saveMyStickerEntry.mockResolvedValue(pending)
    const wrapper = await mountManager()

    const good = new File(['png'], 'happy cat.png', { type: 'image/png' })
    const text = new File(['hi'], 'notes.txt', { type: 'text/plain' })
    const big = new File([new Uint8Array((4 << 20) + 1)], 'huge.png', { type: 'image/png' })
    const input = wrapper.get('[data-test="my-sticker-upload"]')
    Object.defineProperty(input.element, 'files', { value: [good, text, big], configurable: true })
    await input.trigger('change')
    await flushPromises()

    expect(uploadImageFile).toHaveBeenCalledTimes(1)
    expect(uploadImageFile).toHaveBeenCalledWith(good, 'happy cat.png')
    expect(saveMyStickerEntry).toHaveBeenCalledWith({ fileName: '/file/img/owned.png', displayName: 'happy cat' })
    expect(labels(wrapper)).toEqual(['微笑', t('stickers.custom')])
    const alert = wrapper.get('[role="alert"]').text()
    expect(alert).toContain(t('stickers.mine.notImage', { name: 'notes.txt' }))
    expect(alert).toContain(t('stickers.mine.tooLarge', { name: 'huge.png' }))
    wrapper.unmount()
  })

  test('renames, reorders and removes after confirmation', async () => {
    getMyStickers.mockResolvedValue([smile, wave])
    saveMyStickerEntry.mockResolvedValue({ ...wave, displayName: '挥手' })
    orderMyStickers.mockResolvedValue()
    deleteMySticker.mockResolvedValue()
    const wrapper = await mountManager()

    await byLabel(wrapper, t('stickers.mine.rename', { name: 'wave' })).trigger('click')
    const field = wrapper.get<HTMLInputElement>('input[maxlength="64"]')
    expect(document.activeElement).toBe(field.element)
    await field.setValue('  挥手 ')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(saveMyStickerEntry).toHaveBeenCalledWith({ stickerName: 'wave', displayName: '挥手' })
    expect(labels(wrapper)).toEqual(['微笑', '挥手'])

    await byLabel(wrapper, t('stickers.mine.moveEarlier', { name: '挥手' })).trigger('click')
    await flushPromises()
    expect(orderMyStickers).toHaveBeenCalledWith(['wave', 'smile'])
    expect(labels(wrapper)).toEqual(['挥手', '微笑'])

    await byLabel(wrapper, t('stickers.mine.remove', { name: '微笑' })).trigger('click')
    expect(deleteMySticker).not.toHaveBeenCalled()
    await wrapper.findAll('button').find((button) => button.text() === t('stickers.mine.removeConfirm'))!.trigger('click')
    await flushPromises()
    expect(deleteMySticker).toHaveBeenCalledWith('smile')
    expect(labels(wrapper)).toEqual(['挥手'])
    wrapper.unmount()
  })

  test('a rejected reorder reloads the server order and says so', async () => {
    getMyStickers.mockResolvedValueOnce([smile, wave]).mockResolvedValueOnce([wave, smile])
    orderMyStickers.mockRejectedValue(new Error(''))
    const wrapper = await mountManager()

    await byLabel(wrapper, t('stickers.mine.moveLater', { name: '微笑' })).trigger('click')
    await flushPromises()
    expect(getMyStickers).toHaveBeenCalledTimes(2)
    expect(labels(wrapper)).toEqual(['wave', '微笑'])
    expect(wrapper.get('[role="alert"]').text()).toBe(t('stickers.mine.orderFailed'))
    wrapper.unmount()
  })
})
