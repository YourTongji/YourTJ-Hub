// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AdminSettingsPage from '../src/admin/pages/AdminSettingsPage.vue'
import { i18n, setLocale } from '../src/runtime/i18n'
import { getAnnouncement } from '@/admin/runtime/api'

vi.mock('@/admin/runtime/api')
vi.mock('@/admin/runtime/toast', () => ({
  adminToast: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
  },
}))

let wrapper: ReturnType<typeof mount> | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.resetAllMocks()
})

beforeEach(async () => {
  await setLocale('zh')
})

// issue #1322 回归：公告卡片曾在整卡（含标题 Input 与正文 Textarea）上挂
// draggable="true"，在输入框里按下即触发整卡拖拽，无法选中文字/定位光标。
// 排序改由 vuedraggable + 专用手柄承担，卡片内不得再出现原生 draggable。
test('公告卡片只允许通过手柄拖拽排序，输入框不参与拖拽', async () => {
  vi.mocked(getAnnouncement).mockResolvedValue({
    enabled: true,
    content: '',
    items: [
      { id: 'ann-a', title: '第一条', content: '内容一', enabled: true },
      { id: 'ann-b', title: '第二条', content: '内容二', enabled: true },
    ],
  } as never)

  wrapper = mount(AdminSettingsPage, {
    props: {
      kind: 'announcement',
      payload: {
        layout: {
          sidebar: { categories: [] },
          site: { name: 'YourTJHub' },
          viewer: { username: 'admin', avatarUrl: '' },
        },
      } as never,
    },
    global: { plugins: [i18n] },
  })
  await flushPromises()

  const form = wrapper.element.querySelector('form')
  expect(form, 'announcement form should render').not.toBeNull()
  const scoped = form as HTMLElement

  // AC-1：两条公告的标题输入框与正文文本域都存在
  const titles = [...scoped.querySelectorAll<HTMLInputElement>('input')]
  const contents = [...scoped.querySelectorAll<HTMLTextAreaElement>('textarea')]
  expect(titles).toHaveLength(2)
  expect(contents).toHaveLength(2)
  expect(titles.map(input => input.value)).toEqual(['第一条', '第二条'])

  // AC-2：输入框与整卡都不再是原生可拖拽元素
  for (const control of [...titles, ...contents]) {
    expect(control.closest('[draggable="true"]'), 'text control must not sit inside a draggable card').toBeNull()
  }
  expect(scoped.querySelectorAll('[draggable="true"]')).toHaveLength(0)

  // Draggable 只把手柄当拖拽起点（避免整卡/输入框参与拖拽）
  expect(wrapper.findComponent({ name: 'draggable' }).vm.$attrs.handle).toBe('.js-announcement-handle')

  // AC-5：每张卡片都有可聚焦的手柄，且带无障碍名称
  const handles = [...scoped.querySelectorAll<HTMLElement>('.js-announcement-handle')]
  expect(handles).toHaveLength(2)
  for (const handle of handles) {
    expect(handle.tagName.toLowerCase()).toBe('button')
    expect(handle.getAttribute('type')).toBe('button')
    const label = handle.getAttribute('aria-label') || handle.getAttribute('title') || ''
    expect(label.trim(), 'drag handle needs an accessible name').not.toBe('')
  }
})
