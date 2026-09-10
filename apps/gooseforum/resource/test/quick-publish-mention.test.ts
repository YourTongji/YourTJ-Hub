// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { defineComponent, h, shallowRef } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { i18n } from '../src/runtime/i18n'
import QuickPublishModal from '../src/site/components/QuickPublishModal.vue'
import { searchForumUsers } from '../src/runtime/api'
import { useQuickPublish } from '../src/site/composables/useQuickPublish'
import type { LayoutPayload } from '@gooseforum/client'

vi.mock('@/runtime/api', () => ({
  submitTopic: vi.fn(async () => 1),
  uploadImage: vi.fn(async () => ''),
  sensitiveWordsFromError: vi.fn(() => []),
  searchForumUsers: vi.fn(),
}))

// Vditor 在 happy-dom 下无法真实初始化：用可控 stub 提供 getMentionContext/replaceMentionToken，
// 驱动快速发布弹层的 mention 会话（issue #590：发帖编辑器 @ 补全与回复编辑器对齐）。
type StubContext = { prefix: string; rect?: DOMRect | null; inCode?: boolean; element?: HTMLElement | null }
interface StubApi {
  setMentionContext: (ctx: StubContext | null) => void
  getInserted: () => Array<{ length: number; replacement: string }>
}
let currentStubApi: StubApi | null = null

vi.mock('@/site/components/VditorOfficial.vue', () => ({
  default: defineComponent({
    name: 'VditorOfficialStub',
    props: {
      modelValue: { type: String, default: '' },
      placeholder: { type: String, default: '' },
      simple: { type: Boolean, default: false },
      hideUpload: { type: Boolean, default: false },
      sensitiveWords: { type: Array, default: undefined },
    },
    emits: ['update:modelValue', 'input', 'upload', 'error'],
    setup(props, { emit, expose }) {
      const mentionContext = shallowRef<{
        prefix: string
        rect: DOMRect | null
        inCode: boolean
        element: HTMLElement | null
      } | null>(null)
      const inserted: Array<{ length: number; replacement: string }> = []
      const api: StubApi = {
        setMentionContext: (ctx) => {
          mentionContext.value = ctx ? { prefix: ctx.prefix, rect: ctx.rect ?? null, inCode: ctx.inCode ?? false, element: ctx.element ?? null } : null
        },
        getInserted: () => inserted,
      }
      currentStubApi = api
      expose({
        editorReady: shallowRef(true),
        editorFailed: shallowRef(false),
        focus: () => {},
        getValue: () => props.modelValue ?? '',
        setValue: () => {},
        insertMarkdown: () => {},
        setHeight: () => {},
        syncValue: () => props.modelValue ?? '',
        getMentionContext: () => mentionContext.value,
        // 模拟真实插入：token 原位替换 + 尾部空格，随后 input 事件让宿主重检
        replaceMentionToken: (length: number, replacement: string) => {
          inserted.push({ length, replacement })
          const current = mentionContext.value
          if (current) {
            mentionContext.value = { ...current, prefix: `${current.prefix.slice(0, current.prefix.length - length)}${replacement} ` }
          }
          emit('update:modelValue', `${props.modelValue ?? ''}${replacement} `)
          emit('input')
          return true
        },
      })
      return () => h('div', { class: 'vditor-stub' }, String(props.modelValue ?? ''))
    },
  }),
}))

const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/', component: { template: '<div>home</div>' } }],
})

const mockLayout = {
  viewer: { isAuthenticated: true, id: 1, username: 'Tester', avatarUrl: '/avatar.png' },
  sidebar: {
    categories: [
      { id: 101, label: '学术讨论', color: '#10b981', url: '/c/101' },
    ],
    activeKey: '',
  },
  footer: { links: [], primary: [] },
  unread: { notifications: 0, messages: 0 },
  posting: { maxTitleLength: 100 },
  theme: { enabled: true, current: 'gf-light', themeColor: '#3b82f6' },
  insightFlareEnabled: false,
} as unknown as LayoutPayload

function searchUser(id: number, username: string, nickname?: string) {
  return { id, username, nickname: nickname ?? username, avatarUrl: `/a${id}.png`, bio: '' }
}

type Deferred<T> = { promise: Promise<T>; resolve: (value: T) => void }

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((r) => {
    resolve = r
  })
  return { promise, resolve }
}

const searchPending = new Map<string, Deferred<ReturnType<typeof searchUser>[]>>()

const stubEditor = (wrapper: VueWrapper) => wrapper.findComponent({ name: 'VditorOfficialStub' })

function typePrefix(wrapper: VueWrapper, prefix: string) {
  currentStubApi?.setMentionContext({ prefix, element: stubEditor(wrapper).element as HTMLElement })
  stubEditor(wrapper).vm.$emit('input')
}

function mentionPanel() {
  return document.querySelector<HTMLElement>('#gf-mention-listbox')
}

function mountModal() {
  vi.spyOn(window, 'matchMedia').mockReturnValue({
    matches: false,
    media: '(max-width: 640px)',
    addEventListener: () => {},
    removeEventListener: () => {},
  } as unknown as MediaQueryList)
  const { openQuickPublish } = useQuickPublish()
  openQuickPublish(2) // 瞬间
  const wrapper = mount(QuickPublishModal, {
    props: { layout: mockLayout },
    global: { plugins: [i18n, router] },
    attachTo: document.body,
  })
  return wrapper
}

beforeEach(() => {
  i18n.global.locale.value = 'zh'
  searchPending.clear()
  vi.clearAllMocks()
  vi.mocked(searchForumUsers).mockImplementation((query: string) => {
    const d = deferred<ReturnType<typeof searchUser>[]>()
    searchPending.set(query, d)
    return d.promise
  })
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

async function openMentionWithSearch(wrapper: VueWrapper, prefix: string, results: Array<ReturnType<typeof searchUser>>) {
  // 先 flush 让 reka-ui Dialog 异步挂载完成（否则 findComponent 落空），再驱动 300ms debounce。
  // fake timers 会冻结 Dialog Presence 时序，这里统一用真实 timers（与 quick-publish-modal.test.ts 同风格）。
  await flushPromises()
  typePrefix(wrapper, prefix)
  await new Promise((resolve) => setTimeout(resolve, 350))
  const d = searchPending.get(prefix.slice(1))
  if (d) d.resolve(results)
  await flushPromises()
}

describe('QuickPublishModal @mention 会话（issue #590）', () => {
  test('输入 @ 后弹出候选列表：空 query 不打服务端，提示继续输入', async () => {
    const wrapper = mountModal()
    await flushPromises()
    typePrefix(wrapper, '@')
    await flushPromises()
    expect(searchForumUsers).not.toHaveBeenCalled()
    const panel = mentionPanel()
    expect(panel).not.toBeNull()
    expect(panel!.getAttribute('role')).toBe('listbox')
    expect(panel!.textContent).toContain('继续输入以搜索用户')
    wrapper.unmount()
  })

  test('输入 @wa 后服务端搜索结果进入候选列表', async () => {
    const wrapper = mountModal()
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery', '小薇')])
    const panel = mentionPanel()
    expect(panel).not.toBeNull()
    expect(searchForumUsers).toHaveBeenCalledWith('wa', expect.anything())
    expect(panel!.textContent).toContain('@wavery')
    expect(panel!.textContent).toContain('小薇')
    wrapper.unmount()
  })

  test('Enter 选中候选：替换 token 并补空格，会话关闭', async () => {
    const wrapper = mountModal()
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery')])
    const enterEvent = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true })
    stubEditor(wrapper).element.dispatchEvent(enterEvent)
    expect(enterEvent.defaultPrevented).toBe(true)
    expect(currentStubApi!.getInserted()).toEqual([{ length: 3, replacement: '@wavery' }])
    await flushPromises()
    expect(mentionPanel()).toBeNull()
    wrapper.unmount()
  })

  test('Esc 只关闭候选不插入，Tab 不被劫持', async () => {
    const wrapper = mountModal()
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery')])
    const tabEvent = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    stubEditor(wrapper).element.dispatchEvent(tabEvent)
    expect(tabEvent.defaultPrevented).toBe(false)
    expect(currentStubApi!.getInserted()).toHaveLength(0)

    const escEvent = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true })
    stubEditor(wrapper).element.dispatchEvent(escEvent)
    expect(escEvent.defaultPrevented).toBe(true)
    await flushPromises()
    expect(mentionPanel()).toBeNull()
    expect(currentStubApi!.getInserted()).toHaveLength(0)
    wrapper.unmount()
  })

  test('关闭弹层时结束 mention 会话并清除编辑器 ARIA 引用', async () => {
    const wrapper = mountModal()
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery')])
    const el = stubEditor(wrapper).element
    expect(el.getAttribute('aria-expanded')).toBe('true')
    expect(mentionPanel()).not.toBeNull()

    const { closeQuickPublish } = useQuickPublish()
    closeQuickPublish()
    await flushPromises()
    expect(mentionPanel()).toBeNull()
    expect(el.hasAttribute('aria-controls')).toBe(false)
    expect(el.hasAttribute('aria-activedescendant')).toBe(false)
    wrapper.unmount()
  })
})
