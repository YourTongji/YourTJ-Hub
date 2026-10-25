// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import type { StickerItem } from '../packages/client/src/contracts/sticker'

const getForumStickers = vi.fn<() => Promise<StickerItem[]>>()
const saveMySticker = vi.fn<(name: string) => Promise<StickerItem>>()

vi.mock('@/runtime/api', () => {
  class ApiResponseError extends Error {}
  return {
    ApiResponseError,
    getForumStickers: () => getForumStickers(),
    saveMySticker: (name: string) => saveMySticker(name),
  }
})

async function mountCard(signedIn = true) {
  vi.resetModules()
  const StickerCard = (await import('../src/site/components/StickerCard.vue')).default
  const wrapper = mount(StickerCard, {
    props: { signedIn },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
  return wrapper
}

function card() {
  return document.querySelector<HTMLElement>('[data-test="sticker-card"]')
}

function saveButton() {
  return document.querySelector<HTMLButtonElement>('[data-test="sticker-card-save"]')!
}

beforeEach(() => {
  i18n.global.locale.value = 'zh'
  vi.clearAllMocks()
  document.body.innerHTML = ''
  getForumStickers.mockResolvedValue([{ name: 'smile', displayName: '微笑', url: '/smile.png' }])
})

describe('StickerCard 正文贴纸小窗', () => {
  test('放大显示贴纸与名称，收藏后原地确认', async () => {
    saveMySticker.mockResolvedValue({ name: 'smile', url: '/smile.png' })
    const wrapper = await mountCard()
    ;(wrapper.vm as unknown as { open: (s: { name: string, src: string }) => void }).open({ name: 'smile', src: '/smile.png' })
    await flushPromises()

    expect(card()).not.toBeNull()
    expect(card()!.querySelector('img')!.getAttribute('src')).toBe('/smile.png')
    expect(card()!.textContent).toContain('微笑')
    expect(saveButton().textContent).toContain('收藏到我的表情')

    saveButton().click()
    await flushPromises()
    expect(saveMySticker).toHaveBeenCalledWith('smile')
    expect(saveButton().textContent).toContain('已在我的表情')
    expect(saveButton().getAttribute('aria-disabled')).toBe('true')

    saveButton().click()
    await flushPromises()
    expect(saveMySticker).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  test('个人贴纸显示通用名称，失败时保留小窗并可重试', async () => {
    saveMySticker.mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ name: 'u_x', url: '/u.png' })
    const wrapper = await mountCard()
    const name = `u_${'a'.repeat(48)}`
    ;(wrapper.vm as unknown as { open: (s: { name: string, src: string }) => void }).open({ name, src: '/u.png' })
    await flushPromises()
    expect(card()!.textContent).toContain('自定义表情')

    saveButton().click()
    await flushPromises()
    expect(document.querySelector('[data-test="sticker-card-error"]')!.textContent).toContain('收藏未完成，请重试')

    saveButton().click()
    await flushPromises()
    expect(document.querySelector('[data-test="sticker-card-error"]')).toBeNull()
    expect(saveButton().textContent).toContain('已在我的表情')
    wrapper.unmount()
  })

  test('未登录时引导登录而不发请求', async () => {
    const wrapper = await mountCard(false)
    ;(wrapper.vm as unknown as { open: (s: { name: string, src: string }) => void }).open({ name: 'smile', src: '/smile.png' })
    await flushPromises()
    expect(saveButton().textContent).toContain('登录后收藏')
    saveButton().click()
    await flushPromises()
    expect(saveMySticker).not.toHaveBeenCalled()
    expect(wrapper.emitted('signIn')).toHaveLength(1)
    wrapper.unmount()
  })
})
