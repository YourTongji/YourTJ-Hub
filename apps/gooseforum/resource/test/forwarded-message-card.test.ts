// @vitest-environment happy-dom
import { afterEach, expect, it } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { formatDateTime } from '../src/runtime/format'
import { i18n } from '../src/runtime/i18n'
import ForwardedMessageCard from '../src/site/components/ForwardedMessageCard.vue'
let wrapper: VueWrapper | undefined
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })
it('opens nested history without losing the sender avatars or parent dialog', async () => {
  wrapper = mount(ForwardedMessageCard, {
    attachTo: document.body,
    props: {
      bundle: { version: 1, messages: [{ senderName: 'Forwarder', avatarUrl: '/forwarder.png', content: '[Chat history]\nAlice: original', createdAt: '2026-09-28T10:00:00Z', msgType: 4,
        forwarded: { version: 1, messages: [{ senderName: 'Alice', avatarUrl: '/alice.png', content: 'original', createdAt: '2026-09-28T09:00:00Z', msgType: 1 }] },
      }] },
      stickerUrls: new Map(),
    }, global: { plugins: [i18n] },
  })
  await wrapper.get('button').trigger('click')
  await flushPromises()
  const parent = document.querySelector('[role="dialog"]')!
  expect(parent.querySelector('img[src="/forwarder.png"]')).not.toBeNull()
  const nestedCard = parent.querySelector('article button') as HTMLButtonElement
  expect(nestedCard).not.toBeNull()
  nestedCard.click()
  await flushPromises()
  const dialogs = document.querySelectorAll('[role="dialog"]')
  expect(dialogs).toHaveLength(2)
  expect(dialogs[1].textContent).toContain('original')
  expect(dialogs[1].querySelector('img[src="/alice.png"]')).not.toBeNull()
  ;(dialogs[1].querySelector('header button') as HTMLButtonElement).click()
  await flushPromises()
  expect(document.querySelectorAll('[role="dialog"]')).toHaveLength(1)
  expect(document.querySelector('[role="dialog"]')?.textContent).toContain('Forwarder')
})
it('opens a bounded snapshot with escaped author/content and resolved stickers', async () => {
  wrapper = mount(ForwardedMessageCard, {
    attachTo: document.body,
    props: {
      bundle: { version: 1, messages: [{ senderName: '<b>Alice</b>', avatarUrl: '/file/img/alice.png', content: '<script>bad()</script> [:sticker:smile:]', createdAt: '2026-09-28T01:00:00Z', msgType: 1 }] },
      stickerUrls: new Map([['smile', '/file/img/smile.png']]),
    }, global: { plugins: [i18n] },
  })
  expect(document.querySelector('[role="dialog"]')).toBeNull()
  await wrapper.get('button').trigger('click')
  await flushPromises()
  const dialog = document.querySelector('[role="dialog"]')!
  expect(dialog).not.toBeNull()
  expect(dialog.querySelector('time')?.textContent).toBe(formatDateTime('2026-09-28T01:00:00Z'))
  expect(dialog.textContent).toContain('<b>Alice</b>')
  expect(dialog.textContent).toContain('<script>bad()</script>')
  expect(dialog.querySelector('script, b, a')).toBeNull()
  expect(dialog.querySelector('img[src="/file/img/alice.png"]')).not.toBeNull()
  expect(dialog.querySelector('img[src="/file/img/smile.png"]')).not.toBeNull()
  ;(dialog.querySelector('button') as HTMLButtonElement).click()
  await flushPromises()
  expect(document.querySelector('[role="dialog"]')).toBeNull()
})
