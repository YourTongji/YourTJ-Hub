// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import UserAvatar from '../src/site/components/UserAvatar.vue'
import DecorationEditor from '../src/site/components/DecorationEditor.vue'
import { equipDecorations, getDecorations } from '../src/runtime/api'
import type { DecorationState, DecorationStyle } from '@gooseforum/client'
vi.mock('../src/runtime/api', () => ({ getDecorations: vi.fn(), equipDecorations: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const frame: DecorationStyle = { code: 'frame-owned', name: 'Owned frame', kind: 'avatarFrame', lightColors: ['#166534'], darkColors: ['#86efac'], isEnabled: true, sortOrder: 1 }
const state: DecorationState = { catalog: [frame, { ...frame, code: 'frame-locked', name: 'Locked frame' }, { ...frame, code: 'frame-disabled', name: 'Disabled frame', isEnabled: false }], ownedCodes: ['frame-owned', 'frame-disabled'], equipped: { avatarFrame: '', nicknameStyle: '' } }
afterEach(() => vi.clearAllMocks())
describe('decoration equipment', () => {
  it('shows preview but locks unowned and disabled definitions; saves both selected slots', async () => {
    vi.mocked(getDecorations).mockResolvedValue(structuredClone(state))
    vi.mocked(equipDecorations).mockResolvedValue({ ...state, equipped: { avatarFrame: 'frame-owned', nicknameStyle: '' } })
    const wrapper = mount(DecorationEditor, { props: { avatarUrl: '/avatar.webp', name: 'Member' }, global: { stubs: { UserAvatar: true, UserNickname: true } } })
    await flushPromises()
    expect(wrapper.get('input[value="frame-locked"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('input[value="frame-disabled"]').attributes('disabled')).toBeDefined()
    await wrapper.get('input[value="frame-owned"]').setValue(true)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(equipDecorations).toHaveBeenCalledWith({ avatarFrame: 'frame-owned', nicknameStyle: '' })
    expect(wrapper.emitted('changed')?.[0]?.[0]).toEqual({ avatarFrame: frame, nicknameStyle: undefined })
    wrapper.unmount()
  })
  it('retains the pending selection and shows inline error when saving fails', async () => {
    vi.mocked(getDecorations).mockResolvedValue(structuredClone(state))
    vi.mocked(equipDecorations).mockRejectedValue(new Error('Equipment denied'))
    const wrapper = mount(DecorationEditor, { props: { avatarUrl: '/avatar.webp', name: 'Member' }, global: { stubs: { UserAvatar: true, UserNickname: true } } })
    await flushPromises(); await wrapper.get('input[value="frame-owned"]').setValue(true); await wrapper.get('button').trigger('click'); await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toBe('Equipment denied')
    expect((wrapper.get('input[value="frame-owned"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.emitted('changed')).toBeUndefined()
    wrapper.unmount()
  })
})

describe('avatar frame layout', () => {
  it('preserves plain avatar markup and overlays an enabled frame without replacing the image', () => {
    const plain = mount(UserAvatar, { props: { src: '/avatar.webp', alt: 'Member' }, attrs: { class: 'h-10 w-10 rounded-full' } })
    expect(plain.element.tagName).toBe('IMG')
    plain.unmount()
    const framed = mount(UserAvatar, { props: { src: '/avatar.webp', alt: 'Member', frame }, attrs: { class: 'h-10 w-10 rounded-full' } })
    expect(framed.get('.gf-avatar-frame').attributes('aria-hidden')).toBe('true')
    expect(framed.get('img').attributes('src')).toBe('/avatar_medium.webp')
    expect(framed.get('img').attributes('alt')).toBe('Member')
    expect(framed.classes()).toContain('h-10')
    framed.unmount()
  })
})

describe('image avatar frame', () => {
  it('uses only bundled artwork, leaves portrait accessible, and falls back to a gradient on image failure', async () => {
    const wrapper = mount(UserAvatar, { props: { src: '/avatar.webp', alt: 'Member', frame: { ...frame, assetUrl: '/static/decorations/tea-garden-v1.png' } }, attrs: { class: 'h-10 w-10 rounded-full' } })
    const artwork = wrapper.get('.gf-avatar-frame-image')
    expect(artwork.attributes('alt')).toBe('')
    expect(artwork.attributes('aria-hidden')).toBe('true')
    expect(wrapper.get('img[alt="Member"]').attributes('src')).toBe('/avatar_medium.webp')
    expect(wrapper.find('.gf-avatar-frame').exists()).toBe(true)
    await artwork.trigger('load')
    expect(wrapper.find('.gf-avatar-frame').exists()).toBe(false)
    await artwork.trigger('error')
    expect(wrapper.find('.gf-avatar-frame-image').exists()).toBe(false)
    expect(wrapper.find('.gf-avatar-frame').exists()).toBe(true)
    await wrapper.setProps({ frame: { ...frame, assetUrl: 'https://untrusted.example/frame.png' } })
    expect(wrapper.find('.gf-avatar-frame-image').exists()).toBe(false)
    expect(wrapper.find('.gf-avatar-frame').exists()).toBe(true)
    wrapper.unmount()
  })
})

it('scales frame artwork around the avatar center and falls back for invalid scale', async () => {
  const wrapper = mount(UserAvatar, { props: { src: '/avatar.webp', alt: 'Member', frame: { ...frame, assetUrl: '/static/decorations/tea-garden-v1.png', assetScale: 1.6 } }, attrs: { class: 'h-10 w-10 rounded-full' } })
  expect((wrapper.element as HTMLElement).style.getPropertyValue('--avatar-frame-size')).toBe('160%')
  expect((wrapper.element as HTMLElement).style.getPropertyValue('--avatar-frame-offset')).toBe('-30%')
  await wrapper.setProps({ frame: { ...frame, assetUrl: '/static/decorations/tea-garden-v1.png', assetScale: 0.75 } })
  expect((wrapper.element as HTMLElement).style.getPropertyValue('--avatar-frame-offset')).toBe('12.5%')
  await wrapper.setProps({ frame: { ...frame, assetUrl: '/static/decorations/tea-garden-v1.png', assetScale: 20 } })
  expect((wrapper.element as HTMLElement).style.getPropertyValue('--avatar-frame-size')).toBe('125%')
  wrapper.unmount()
})
