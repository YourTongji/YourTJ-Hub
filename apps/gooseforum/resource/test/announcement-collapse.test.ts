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

// issue #1322：公告卡片进入页面时全部折叠，只留排名/标题/启用/删除，
// 点标题行展开正文。新增公告默认展开，省一次点击。
async function mountAnnouncementPage() {
  vi.mocked(getAnnouncement).mockResolvedValue({
    enabled: true,
    content: '',
    items: [
      { id: 'ann-1', title: '第一条公告', content: '内容一', enabled: true },
      { id: 'ann-2', title: '第二条公告', content: '内容二', enabled: false },
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
  return form as HTMLElement
}

function bodies(root: HTMLElement) {
  return {
    first: root.querySelector<HTMLElement>('#announcement-ann-1'),
    second: root.querySelector<HTMLElement>('#announcement-ann-2'),
  }
}

function toggles(root: HTMLElement) {
  return [...root.querySelectorAll<HTMLButtonElement>('button[aria-controls^="announcement-"]')]
}

test('公告卡片默认全部折叠，只保留头部摘要', async () => {
  const root = await mountAnnouncementPage()
  const { first, second } = bodies(root)
  expect(first, 'collapsed body container must still exist (v-show)').not.toBeNull()
  expect(second).not.toBeNull()
  expect(first!.style.display).toBe('none')
  expect(second!.style.display).toBe('none')

  // 标题输入框随正文一起收起
  const titleInputs = [...root.querySelectorAll<HTMLInputElement>('input')]
  expect(titleInputs).toHaveLength(2)
  for (const input of titleInputs) expect(input.closest<HTMLElement>('[id^="announcement-"]')!.style.display).toBe('none')

  // 折叠开关：button + aria-expanded/aria-controls + 键盘可聚焦（非 tabindex=-1）
  const buttons = toggles(root)
  expect(buttons).toHaveLength(2)
  for (const button of buttons) {
    expect(button.getAttribute('aria-expanded')).toBe('false')
    expect(button.getAttribute('tabindex')).not.toBe('-1')
  }
  expect(buttons.map(button => button.getAttribute('aria-controls'))).toEqual([
    'announcement-ann-1',
    'announcement-ann-2',
  ])

  // 折叠态头部仍有排名、标题、启用开关、删除按钮和拖拽手柄
  expect(buttons[0]!.textContent).toContain('#1')
  expect(buttons[0]!.textContent).toContain('第一条公告')
  expect(buttons[1]!.textContent).toContain('#2')
  expect(buttons[1]!.textContent).toContain('第二条公告')
  const cards = buttons.map(button => button.closest('.rounded-lg') as HTMLElement)
  for (const card of cards) {
    expect(card.querySelectorAll('[data-slot="switch"]')).toHaveLength(1)
    expect(card.querySelector('button .sr-only')?.textContent).toContain('删除公告')
    expect(card.querySelector('.js-announcement-handle')).not.toBeNull()
  }
})

test('点折叠开关展开单条公告，另一条保持折叠', async () => {
  const root = await mountAnnouncementPage()
  const buttons = toggles(root)
  buttons[0]!.click()
  await flushPromises()

  const { first, second } = bodies(root)
  expect(first!.style.display).not.toBe('none')
  expect(second!.style.display).toBe('none')
  expect(toggles(root)[0]!.getAttribute('aria-expanded')).toBe('true')
  expect(toggles(root)[1]!.getAttribute('aria-expanded')).toBe('false')

  // 再次点击收起
  toggles(root)[0]!.click()
  await flushPromises()
  expect(bodies(root).first!.style.display).toBe('none')
})

test('添加公告后新卡片默认展开，已有卡片保持折叠', async () => {
  const root = await mountAnnouncementPage()
  const addButton = [...root.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent?.includes('添加公告'))
  expect(addButton, 'add announcement button should render').toBeDefined()
  addButton!.click()
  await flushPromises()

  const buttons = toggles(root)
  expect(buttons).toHaveLength(3)
  expect(buttons.map(button => button.getAttribute('aria-expanded'))).toEqual(['false', 'false', 'true'])
  const added = buttons[2]!.getAttribute('aria-controls')!
  expect(root.querySelector<HTMLElement>(`#${added}`)!.style.display).not.toBe('none')
})
