// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { defineComponent, h, ref, watch } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { PostPayload, ViewerPayload } from '@gooseforum/client'
import PostStream from '../src/site/components/PostStream.vue'
import PostComposer from '../src/site/components/PostComposer.vue'
import { i18n } from '../src/runtime/i18n'
import { readReplyDraft, writeReplyDraft } from '../src/site/utils/reply-draft'

// Exercise the real stream + composer boundary. The editor stub deliberately
// leaves its input callback pending, as Vditor does while debouncing input.
vi.mock('@/site/components/VditorOfficial.vue', () => ({
  default: defineComponent({
    name: 'DelayedEditor',
    props: { modelValue: { type: String, default: '' } },
    emits: ['update:modelValue'],
    setup(props, { emit, expose }) {
      const documentValue = ref(props.modelValue)
      watch(() => props.modelValue, value => { documentValue.value = value })
      expose({
        editorReady: true,
        editorFailed: false,
        focus() {},
        setHeight() {},
        getMentionContext: () => null,
        syncValue() {
          emit('update:modelValue', documentValue.value)
          return documentValue.value
        },
      })
      return () => h('textarea', {
        'data-test': 'delayed-editor', value: documentValue.value,
        onInput: (event: Event) => { documentValue.value = (event.target as HTMLTextAreaElement).value },
      })
    },
  }),
}))

const records = new Map<string, string>()
const storage = {
  getItem: (key: string) => records.get(key) ?? null,
  setItem: (key: string, value: string) => records.set(key, String(value)),
  removeItem: (key: string) => records.delete(key),
}
Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: storage })
Object.defineProperty(window, 'localStorage', { configurable: true, value: storage })

const viewer: ViewerPayload = {
  id: 7, username: 'writer', email: 'writer@example.test', avatarUrl: '',
  isAuthenticated: true, canAccessAdmin: false, isModerator: false,
  requiresEmailVerification: false, adminPermissions: [],
}
const reply: PostPayload = {
  id: 902, postNo: 2, topicId: 900, content: 'published reply', renderedContent: '<p>published reply</p>',
  processStatus: 0, isHidden: false, isAuthorDeleted: false, isModeratorRemoved: false,
  canModerate: false, author: { id: 7, username: 'writer', avatarUrl: '' },
  createdAt: '2026-11-07 01:00:00', updatedAt: '2026-11-07 01:00:00',
  isOwnPost: true, revisionCount: 0, likeCount: 0, isLiked: false, isBookmarked: false,
}
let wrapper: VueWrapper | undefined
async function openStream(initial = '') {
  if (initial) writeReplyDraft(7, 900, { content: initial, identity: 'persona', targetPostId: 901 })
  wrapper = mount(PostStream, {
    props: {
      topicId: 900, topicTitle: 'reply snapshots', contentType: 0, viewer, canPost: true,
      initialPostStream: { posts: [reply], hasBefore: false, hasAfter: false },
    },
    global: { plugins: [i18n], stubs: { StickerPicker: true, IdentityPicker: true }, directives: { 'code-copy': () => {}, 'code-highlight': () => {}, 'math-render': () => {}, 'content-enhancements': () => {} } },
    attachTo: document.body,
  })
  ;(wrapper.vm as unknown as { openFloatingPostComposer(): void }).openFloatingPostComposer()
  await flushPromises()
  return wrapper
}
async function typeDocument(value: string) {
  const input = document.querySelector<HTMLTextAreaElement>('[data-test="delayed-editor"]')!
  expect(input).not.toBeNull()
  input.value = value
  input.dispatchEvent(new Event('input', { bubbles: true }))
  await flushPromises()
}
beforeEach(() => { records.clear(); i18n.global.locale.value = 'zh' })
afterEach(() => { wrapper?.unmount(); wrapper = undefined; document.body.innerHTML = ''; vi.restoreAllMocks() })

for (const [name, initial, current] of [
  ['first input', '', 'first confirmed reply'],
  ['latest tail', 'old reply', 'old reply with latest tail'],
  ['explicit clear', 'old reply', ''],
]) {
  test(`pagehide persists the live document: ${name}`, async () => {
    await openStream(initial)
    await typeDocument(current)
    window.dispatchEvent(new Event('pagehide'))
    expect(readReplyDraft(7, 900)?.content ?? '').toBe(current)
  })
}

test('component unmount persists pending editor input', async () => {
  await openStream('old reply')
  await typeDocument('last confirmed reply')
  wrapper!.unmount()
  wrapper = undefined
  expect(readReplyDraft(7, 900)?.content).toBe('last confirmed reply')
})

test('editing another reply and cancelling restores the live unsent reply', async () => {
  const stream = await openStream('old reply')
  await typeDocument('last confirmed reply')
  ;(stream.vm as any).startEditPost(reply)
  await flushPromises()
  expect(stream.findComponent(PostComposer).props('modelValue')).toBe('published reply')
  ;(stream.vm as any).cancelEditPost()
  await flushPromises()
  expect(stream.findComponent(PostComposer).props('modelValue')).toBe('last confirmed reply')
  expect(readReplyDraft(7, 900)?.content).toBe('last confirmed reply')
})

for (const scope of ['owner', 'topic'] as const) {
  test(`changing ${scope} never copies the previous editor document into the new draft`, async () => {
    const stream = await openStream('old saved reply')
    await typeDocument('old private input')
    await stream.setProps(scope === 'owner' ? { viewer: { ...viewer, id: 8 } } : { topicId: 901 })
    await flushPromises()
    window.dispatchEvent(new Event('pagehide'))
    expect(readReplyDraft(scope === 'owner' ? 8 : 7, scope === 'topic' ? 901 : 900)).toBeNull()
    expect(document.querySelector<HTMLTextAreaElement>('[data-test="delayed-editor"]')!.value).toBe('')
  })
}

test('pagehide during reply editing does not overwrite the unsent reply draft', async () => {
  const stream = await openStream('old reply')
  await typeDocument('unsent current reply')
  ;(stream.vm as any).startEditPost(reply)
  await flushPromises()
  await typeDocument('unsubmitted edit to published reply')
  window.dispatchEvent(new Event('pagehide'))
  expect(readReplyDraft(7, 900)?.content).toBe('unsent current reply')
})

test('model-driven clear does not reintroduce submitted content', async () => {
  const stream = await openStream('old reply')
  await typeDocument('submitted content')
  ;(stream.vm as any).postContent = ''
  await flushPromises()
  window.dispatchEvent(new Event('pagehide'))
  expect(readReplyDraft(7, 900)).toBeNull()
})
