// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SiteChromeManagementPage from '../src/admin/pages/management/SiteChromeManagementPage.vue'
import { i18n, setLocale } from '../src/runtime/i18n'

vi.mock('../src/admin/runtime/toast', () => ({
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
  vi.unstubAllGlobals()
})

test('mounts and loads site chrome when randomUUID is unavailable', async () => {
  await setLocale('zh')
  const originalCrypto = globalThis.crypto
  vi.stubGlobal('crypto', {
    getRandomValues: (bytes: Uint8Array) => originalCrypto.getRandomValues(bytes),
  })
  const chrome = {
    header: [],
    mainMenu: [],
    resources: [],
    sidebarGroups: [],
    footerInfo: { primary: [], list: [] },
  }
  const fetchMock = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
    void init
    const result = String(url) === '/api/admin/site-chrome' ? chrome : true
    return new Response(JSON.stringify({ code: 0, result }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  })
  vi.stubGlobal('fetch', fetchMock)

  wrapper = mount(SiteChromeManagementPage, {
    props: {
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

  expect(wrapper.exists()).toBe(true)
  expect(fetchMock).toHaveBeenCalledWith('/api/admin/site-chrome', expect.objectContaining({
    headers: { Accept: 'application/json' },
  }))

  const addMenuButton = wrapper.findAll('button').find(button => button.text().includes('添加菜单'))
  expect(addMenuButton, 'add-menu action should render').toBeDefined()
  await addMenuButton!.trigger('click')
  const labelInput = document.querySelector<HTMLInputElement>('input[placeholder="例如 友情链接"]')
  const urlInput = document.querySelector<HTMLInputElement>('input[placeholder="/links 或 https://example.com"]')
  expect(labelInput, 'label input should render').not.toBeNull()
  expect(urlInput, 'URL input should render').not.toBeNull()
  labelInput!.value = 'HTTP menu item'
  labelInput!.dispatchEvent(new Event('input', { bubbles: true }))
  urlInput!.value = '/http-menu-item'
  urlInput!.dispatchEvent(new Event('input', { bubbles: true }))
  await flushPromises()
  const itemForm = labelInput!.closest('form')
  expect(itemForm, 'item form should render').not.toBeNull()
  itemForm!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  await flushPromises()

  const savePageButton = wrapper.findAll('button').find(button => button.text().includes('保存分类'))
  expect(savePageButton, 'save action should render').toBeDefined()
  await savePageButton!.trigger('click')
  await flushPromises()
  const saveCall = fetchMock.mock.calls.find(([url]) => String(url) === '/api/admin/save-site-chrome')
  expect(saveCall, 'save request should be sent').toBeDefined()
  const saved = JSON.parse(String(saveCall?.[1]?.body)).settings
  expect(saved.mainMenu[0]).toMatchObject({
    label: 'HTTP menu item',
    url: '/http-menu-item',
  })
  expect(saved.mainMenu[0].id).toMatch(
    /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
  )
})
