// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import IdentityPicker from '../src/site/components/IdentityPicker.vue'
import { confirmName, generateNames, getIdentityState, setProfileContent, type IdentityState } from '../src/runtime/anonymous-identity'

vi.mock('../src/runtime/anonymous-identity', () => ({
  getIdentityState: vi.fn(),
  generateNames: vi.fn(),
  confirmName: vi.fn(),
  disableIdentity: vi.fn(),
  setProfileContent: vi.fn(),
}))
let state: IdentityState
let wrapper: ReturnType<typeof mount>
beforeEach(async () => {
  vi.resetAllMocks()
  await setLocale('zh')
  state = {
    persona: null,
    nameSelectedAt: null,
    nameChangeAvailableAt: null,
    disabled: false,
    governanceDisabled: false,
    showContent: true,
    day: '2026-10-07',
    remaining: 10,
    resetsAt: '2026-10-07T16:00:00Z',
    batches: [],
    lexiconVersion: 'test',
  }
  vi.mocked(getIdentityState).mockImplementation(async () => structuredClone(state))
})
afterEach(() => {
  wrapper?.unmount()
  document.body.innerHTML = ''
})
// Menu options carry a title and a subtitle; match either the whole text or its last line.
// Decorative aria-hidden parts (slip numbers) are not part of the label.
function visibleText(item: Element) {
  const copy = item.cloneNode(true) as Element
  copy.querySelectorAll('[aria-hidden="true"]').forEach((node) => node.remove())
  return copy.textContent?.trim()
}
function button(label: string) {
  const found = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
    (item) => visibleText(item) === label || item.lastElementChild?.lastElementChild?.textContent?.trim() === label,
  )
  expect(found, `button ${label}`).toBeTruthy()
  return found!
}
test('sets up a missing persona in place and selects it only after explicit confirmation', async () => {
  const originalURL = location.href
  wrapper = mount(IdentityPicker, {
    props: { viewer: { username: 'yzxoi', avatarUrl: '' } },
    attachTo: document.body,
    global: { plugins: [i18n] },
  })
  await flushPromises()
  expect(wrapper.find('a[href="/settings#anonymous-identity"]').exists()).toBe(false)
  await wrapper.find('button[aria-label^="发布身份"]').trigger('click')
  await flushPromises()
  button('设置匿名身份').click()
  await flushPromises()
  expect(document.querySelector('[role="dialog"]')).toBeTruthy()
  vi.mocked(generateNames).mockImplementation(async () => {
    const batch = { id: 'batch1', day: state.day, words: ['星辰', '同学'], expiresAt: state.resetsAt, createdAt: '' }
    state.batches.push(batch)
    state.remaining--
    return batch
  })
  button('生成花名').click()
  await flushPromises()
  button('星辰').click()
  await flushPromises()
  expect(confirmName).not.toHaveBeenCalled()
  expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  expect(document.body.textContent).toContain('一年内不能换花名')
  vi.mocked(confirmName).mockResolvedValue({
    kind: 'persona',
    publicUid: 'p1',
    name: '星辰',
    avatarUrl: '/a/avatar.svg',
    profileUrl: '/a/p1',
  })
  button('使用这个花名').click()
  await flushPromises()
  expect(confirmName).toHaveBeenCalledWith('batch1', 0)
  expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['persona'])
  expect(document.querySelector('[role="dialog"]')).toBeNull()
  expect(location.href).toBe(originalURL)
})
test('loading failure keeps the selected identity and offers retry', async () => {
  vi.mocked(getIdentityState).mockRejectedValueOnce(new Error('offline'))
  wrapper = mount(IdentityPicker, {
    props: { modelValue: 'persona', viewer: { username: 'yzxoi', avatarUrl: '' } },
    attachTo: document.body,
    global: { plugins: [i18n] },
  })
  await flushPromises()
  expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  expect(wrapper.find('button[aria-label="重试"]').exists()).toBe(true)
})

test('renders the member avatar frame in the trigger and identity menu', async () => {
  const viewer = {
    username: 'owner',
    avatarUrl: '/static/pic/1.webp',
    decoration: {
      avatarFrame: {
        code: 'gradient-frame',
        name: 'Gradient frame',
        kind: 'avatarFrame' as const,
        lightColors: ['#ff0000', '#0000ff'],
        darkColors: ['#ff8888', '#8888ff'],
        isEnabled: true,
        sortOrder: 1,
      },
    },
  }
  wrapper = mount(IdentityPicker, { props: { viewer }, attachTo: document.body, global: { plugins: [i18n] } })
  await flushPromises()

  expect(wrapper.find('.gf-avatar-frame').exists()).toBe(true)
  await wrapper.find('button[aria-label^="发布身份"]').trigger('click')
  await flushPromises()
  expect([...document.querySelectorAll('.gf-avatar-frame')]).toHaveLength(2)
})

test('renders image avatar frames for the member identity', async () => {
  const viewer = {
    username: 'owner',
    avatarUrl: '/static/pic/1.webp',
    decoration: {
      avatarFrame: {
        code: 'image-frame',
        name: 'Image frame',
        kind: 'avatarFrame' as const,
        lightColors: ['#ff0000'],
        darkColors: ['#ff8888'],
        isEnabled: true,
        sortOrder: 1,
        assetUrl: '/static/decorations/frame.webp',
      },
    },
  }
  wrapper = mount(IdentityPicker, { props: { viewer }, attachTo: document.body, global: { plugins: [i18n] } })
  await flushPromises()

  expect(wrapper.find('img.gf-avatar-frame-image').exists()).toBe(true)
  await wrapper.find('button[aria-label^="发布身份"]').trigger('click')
  await flushPromises()
  expect(document.querySelectorAll('img.gf-avatar-frame-image')).toHaveLength(2)
})

test('keeps the member avatar undecorated when no frame is equipped', async () => {
  wrapper = mount(IdentityPicker, {
    props: { viewer: { username: 'owner', avatarUrl: '/avatar.svg' } },
    attachTo: document.body,
    global: { plugins: [i18n] },
  })
  await flushPromises()

  expect(wrapper.find('.gf-avatar-frame, .gf-avatar-frame-image').exists()).toBe(false)
  expect(wrapper.find('button[aria-label^="发布身份"] img[src="/avatar.svg"]').exists()).toBe(true)
})

test('does not expose the member frame when the persona identity is selected', async () => {
  state.persona = { kind: 'persona', publicUid: 'p1', name: '星辰', avatarUrl: '/persona.svg', profileUrl: '/a/p1' }
  wrapper = mount(IdentityPicker, {
    props: {
      modelValue: 'persona',
      viewer: {
        username: 'owner',
        avatarUrl: '/member.svg',
        decoration: {
          avatarFrame: {
            code: 'private-frame',
            name: 'Private frame',
            kind: 'avatarFrame',
            lightColors: ['#ff0000'],
            darkColors: ['#ff8888'],
            isEnabled: true,
            sortOrder: 1,
          },
          nicknameStyle: {
            code: 'private-name',
            name: 'Private name',
            kind: 'nicknameStyle',
            lightColors: ['#ff0000'],
            darkColors: ['#ff8888'],
            isEnabled: true,
            sortOrder: 2,
          },
        },
      },
    },
    attachTo: document.body,
    global: { plugins: [i18n] },
  })
  await flushPromises()

  const trigger = wrapper.find('button[aria-label^="发布身份"]')
  expect(trigger.find('img[src="/persona.svg"]').exists()).toBe(true)
  expect(trigger.find('.gf-avatar-frame, .gf-avatar-frame-image').exists()).toBe(false)
  expect(trigger.find('.gf-decorated-nickname').exists()).toBe(false)

  await trigger.trigger('click')
  await flushPromises()
  const memberOption = [...document.querySelectorAll<HTMLButtonElement>('button')]
    .find((item) => item.getAttribute('aria-pressed') === 'false' && item.textContent?.includes('主账号'))
  expect(memberOption?.querySelector('.gf-avatar-frame')).toBeTruthy()
  const personaOption = [...document.querySelectorAll<HTMLButtonElement>('button')]
    .find((item) => item.getAttribute('aria-pressed') === 'true')
  expect(personaOption?.querySelector('.gf-avatar-frame, .gf-avatar-frame-image')).toBeNull()
  expect(personaOption?.querySelector('.gf-decorated-nickname')).toBeNull()
})

test('profile privacy commits only after success and preserves the preference on failure', async () => {
  state.persona = { kind: 'persona', publicUid: 'p1', name: '躲进云里的猫', avatarUrl: '', profileUrl: '/a/p1' }
  state.nameChangeAvailableAt = '2099-01-01T00:00:00Z'
  wrapper = mount(IdentityPicker, { props: { modelValue: 'persona', viewer: { username: 'owner', avatarUrl: '' } }, attachTo: document.body, global: { plugins: [i18n] } })
  await flushPromises()
  await wrapper.find('button[aria-label^="发布身份"]').trigger('click')
  await flushPromises()
  button('管理匿名身份').click()
  await flushPromises()
  const toggle = () => document.querySelector<HTMLButtonElement>('[role="switch"][aria-labelledby="anonymous-show-content-label"]')!
  expect(toggle().getAttribute('aria-checked')).toBe('true')
  vi.mocked(setProfileContent).mockRejectedValueOnce(new Error('offline'))
  toggle().click(); await flushPromises()
  expect(toggle().getAttribute('aria-checked')).toBe('true')
  expect(document.body.textContent).toContain('offline')
  vi.mocked(setProfileContent).mockResolvedValue(true)
  toggle().click(); await flushPromises()
  expect(setProfileContent).toHaveBeenLastCalledWith(false)
  expect(toggle().getAttribute('aria-checked')).toBe('false')
})

test('ambiguous draw retries reuse the same request key and failed confirmation keeps the publishing identity', async () => {
  wrapper = mount(IdentityPicker, {
    props: { viewer: { username: 'yzxoi', avatarUrl: '' } },
    attachTo: document.body,
    global: { plugins: [i18n] },
  })
  await flushPromises()
  await wrapper.find('button[aria-label^="发布身份"]').trigger('click')
  await flushPromises()
  button('设置匿名身份').click()
  await flushPromises()
  const batch = { id: 'batch-retry', day: state.day, words: ['星辰'], expiresAt: state.resetsAt, createdAt: '' }
  vi.mocked(generateNames)
    .mockRejectedValueOnce(new Error('lost response'))
    .mockImplementationOnce(async () => {
      state.batches = [batch]
      state.remaining = 9
      return batch
    })
  button('生成花名').click()
  await flushPromises()
  button('生成花名').click()
  await flushPromises()
  expect(vi.mocked(generateNames).mock.calls[0][1]).toBe(vi.mocked(generateNames).mock.calls[1][1])
  button('星辰').click()
  await flushPromises()
  vi.mocked(confirmName).mockRejectedValueOnce(new Error('offline'))
  button('使用这个花名').click()
  await flushPromises()
  expect(document.querySelector('[role="alert"]')?.textContent).toContain('offline')
  expect(document.querySelector('[role="dialog"]')).toBeTruthy()
  expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  button('取消').click()
  await flushPromises()
  expect(document.querySelector('[role="dialog"]')).toBeNull()
  expect(wrapper.emitted('update:modelValue')).toBeUndefined()
})
