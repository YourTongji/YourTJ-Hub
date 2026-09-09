// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi, type MockInstance } from 'vitest'
import { defineComponent, h, shallowRef } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import PostComposer from '../src/site/components/PostComposer.vue'
import { searchForumUsers } from '../src/runtime/api'
import type { MentionUser } from '../src/runtime/mention'

vi.mock('@/runtime/api', () => ({
  uploadImage: vi.fn(async () => ''),
  searchForumUsers: vi.fn(),
}))

// Vditor 在 happy-dom 下无法真实初始化：用可控 stub 提供 getMentionContext/replaceMentionToken，
// 驱动 PostComposer 的 mention 会话（issue #564：键盘操作、stale response、移动布局验收）。
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
      height: { type: [Number, String], default: undefined },
      compact: { type: Boolean, default: false },
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
        syncValue: () => {},
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

type Deferred<T> = { promise: Promise<T>; resolve: (value: T) => void }

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((r) => {
    resolve = r
  })
  return { promise, resolve }
}

function searchUser(id: number, username: string, nickname?: string) {
  return { id, username, nickname: nickname ?? username, avatarUrl: `/a${id}.png`, bio: '' }
}

const searchPending = new Map<string, Deferred<ReturnType<typeof searchUser>[]>>()

function typePrefix(wrapper: VueWrapper, prefix: string) {
  currentStubApi?.setMentionContext({ prefix })
  const stub = wrapper.findComponent({ name: 'VditorOfficialStub' })
  stub.vm.$emit('input')
}

function mountComposer(options: {
  mentionUsers?: MentionUser[]
  currentUserId?: number
  mobile?: boolean
}) {
  const matchMedia = vi.spyOn(window, 'matchMedia').mockReturnValue({
    matches: options.mobile ?? false,
    media: '(max-width: 640px)',
    addEventListener: () => {},
    removeEventListener: () => {},
  } as unknown as MediaQueryList)
  const wrapper = mount(PostComposer, {
    props: {
      authenticated: true,
      errorMessage: '',
      open: true,
      submitting: false,
      successMessage: '',
      mentionUsers: options.mentionUsers ?? [],
      currentUserId: options.currentUserId ?? 0,
    },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
  return { wrapper, matchMedia }
}

function mentionPanel() {
  return document.querySelector<HTMLElement>('#gf-mention-listbox')
}

const stubEditor = (wrapper: VueWrapper) => wrapper.findComponent({ name: 'VditorOfficialStub' })

beforeEach(() => {
  i18n.global.locale.value = 'zh'
  searchPending.clear()
  vi.useFakeTimers()
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
  typePrefix(wrapper, prefix)
  await vi.advanceTimersByTimeAsync(300)
  const d = searchPending.get(prefix.slice(1))
  if (d) d.resolve(results)
  await flushPromises()
}

describe('PostComposer @mention 会话（issue #564）', () => {
  test('输入 @ 后打开候选列表，展示服务端搜索结果', async () => {
    const { wrapper } = mountComposer({})
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery', '小薇')])
    const panel = mentionPanel()
    expect(panel).not.toBeNull()
    expect(panel!.getAttribute('role')).toBe('listbox')
    expect(panel!.textContent).toContain('@wavery')
    expect(panel!.textContent).toContain('小薇')
    wrapper.unmount()
  })

  test('仅输入 @：只展示本地上下文，不发起服务端请求', async () => {
    const mentionUsers: MentionUser[] = [
      { id: 1, username: 'target', nickname: '回复目标', avatarUrl: '/a1.png', tag: 'reply-target' },
      { id: 2, username: 'author', nickname: '主题作者', avatarUrl: '/a2.png', tag: 'topic-author' },
    ]
    const { wrapper } = mountComposer({ mentionUsers })
    typePrefix(wrapper, '@')
    await flushPromises()
    expect(searchForumUsers).not.toHaveBeenCalled()
    const panel = mentionPanel()
    expect(panel!.textContent).toContain('@target')
    expect(panel!.textContent).toContain('@author')
    expect(panel!.textContent).toContain('正在回复')
    expect(panel!.textContent).toContain('主题作者')
    wrapper.unmount()
  })

  test('仅输入 @ 且无本地上下文：提示继续输入而非空查询', async () => {
    const { wrapper } = mountComposer({})
    typePrefix(wrapper, '@')
    await flushPromises()
    expect(searchForumUsers).not.toHaveBeenCalled()
    expect(mentionPanel()!.textContent).toContain('继续输入以搜索用户')
    wrapper.unmount()
  })

  test('光标位于 code/pre/a 内不弹候选', async () => {
    const { wrapper } = mountComposer({})
    currentStubApi?.setMentionContext({ prefix: '@wa', inCode: true })
    wrapper.findComponent({ name: 'VditorOfficialStub' }).vm.$emit('input')
    await flushPromises()
    expect(mentionPanel()).toBeNull()
    expect(searchForumUsers).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  test('空格/终止标点后会话关闭', async () => {
    const { wrapper } = mountComposer({})
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery')])
    expect(mentionPanel()).not.toBeNull()
    typePrefix(wrapper, '@wa ')
    await flushPromises()
    expect(mentionPanel()).toBeNull()
    wrapper.unmount()
  })

  test('stale 响应不会覆盖更新 query 的结果', async () => {
    const { wrapper } = mountComposer({})
    // 输入 "@a" → 发起 A 请求（不 resolve）
    typePrefix(wrapper, '@a')
    await vi.advanceTimersByTimeAsync(300)
    expect(searchPending.has('a')).toBe(true)
    // 输入 "@b" → 发起 B 请求
    typePrefix(wrapper, '@b')
    await vi.advanceTimersByTimeAsync(300)
    expect(searchPending.has('b')).toBe(true)
    // B 先返回，A 后返回：展示必须停留在 B 的结果
    searchPending.get('b')!.resolve([searchUser(31, 'bee')])
    await flushPromises()
    expect(mentionPanel()!.textContent).toContain('@bee')
    searchPending.get('a')!.resolve([searchUser(30, 'ant')])
    await flushPromises()
    expect(mentionPanel()!.textContent).toContain('@bee')
    expect(mentionPanel()!.textContent).not.toContain('@ant')
    wrapper.unmount()
  })

  test('debounce 窗口内旧 query 的响应不会覆盖新 query 结果', async () => {
    const { wrapper } = mountComposer({})
    // 输入 "@a"：发起 A 请求（不 resolve），随后立即改为 "@ab"（debounce 尚未触发）
    typePrefix(wrapper, '@a')
    await vi.advanceTimersByTimeAsync(300)
    expect(searchPending.has('a')).toBe(true)
    typePrefix(wrapper, '@ab')
    // A 在 debounce 窗口内返回：不得覆盖（query 已变化）
    searchPending.get('a')!.resolve([searchUser(30, 'ant')])
    await flushPromises()
    expect(mentionPanel()!.textContent).not.toContain('@ant')
    // 新 query 的 debounce 触发并返回
    await vi.advanceTimersByTimeAsync(300)
    searchPending.get('ab')!.resolve([searchUser(31, 'abba')])
    await flushPromises()
    expect(mentionPanel()!.textContent).toContain('@abba')
    wrapper.unmount()
  })

  test('ArrowDown/ArrowUp 移动 active，Enter 选中插入', async () => {
    const { wrapper } = mountComposer({})
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery'), searchUser(22, 'wangwu')])
    const panel = mentionPanel()!
    expect(panel.querySelectorAll('[role="option"]')).toHaveLength(2)
    expect(panel.querySelector('[data-active="true"]')!.getAttribute('aria-label')).toContain('@wavery')

    const rawHandler = () => console.log('[debug] raw document keydown fired')
    document.addEventListener('keydown', rawHandler)
    const arrowDown = new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true })
    document.dispatchEvent(arrowDown)
    document.removeEventListener('keydown', rawHandler)
    console.log('[debug] arrowDown defaultPrevented:', arrowDown.defaultPrevented)
    await flushPromises()
    expect(panel.querySelector('[data-active="true"]')!.getAttribute('aria-label')).toContain('@wangwu')

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true, cancelable: true }))
    await flushPromises()
    expect(panel.querySelector('[data-active="true"]')!.getAttribute('aria-label')).toContain('@wavery')

    const enterEvent = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true })
    document.dispatchEvent(enterEvent)
    expect(enterEvent.defaultPrevented).toBe(true)
    // "@wa" token 长度 3，替换为 @wavery 并补空格；插入后会话关闭
    const inserted = currentStubApi!.getInserted()
    expect(inserted).toEqual([{ length: 3, replacement: '@wavery' }])
    await flushPromises()
    expect(mentionPanel()).toBeNull()
    wrapper.unmount()
  })

  test('Esc 只关闭候选不插入，Tab 不被劫持', async () => {
    const { wrapper } = mountComposer({})
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery')])
    const tabEvent = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    document.dispatchEvent(tabEvent)
    expect(tabEvent.defaultPrevented).toBe(false)
    expect(currentStubApi!.getInserted()).toHaveLength(0)

    const escEvent = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true })
    document.dispatchEvent(escEvent)
    expect(escEvent.defaultPrevented).toBe(true)
    await flushPromises()
    expect(mentionPanel()).toBeNull()
    expect(currentStubApi!.getInserted()).toHaveLength(0)
    wrapper.unmount()
  })

  test('当前用户不进入候选', async () => {
    const { wrapper } = mountComposer({ currentUserId: 21 })
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery'), searchUser(22, 'wangwu')])
    const panel = mentionPanel()!
    expect(panel.textContent).not.toContain('@wavery')
    expect(panel.textContent).toContain('@wangwu')
    wrapper.unmount()
  })

  test('移动端（≤640px）：候选停靠编辑区（is-docked），不渲染贴近 caret 浮层', async () => {
    const { wrapper } = mountComposer({ mobile: true })
    await openMentionWithSearch(wrapper, '@wa', [searchUser(21, 'wavery')])
    const panel = mentionPanel()!
    expect(panel.classList.contains('is-docked')).toBe(true)
    expect(panel.getAttribute('style') || '').not.toContain('position: absolute')
    wrapper.unmount()
  })
})