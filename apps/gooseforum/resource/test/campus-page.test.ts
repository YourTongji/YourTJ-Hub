// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { createI18n } from 'vue-i18n'
import type { CampusDataset, CampusDatasetKey, CampusStatus, LayoutPayload } from '@gooseforum/client'
import { CampusError } from '../src/runtime/campus-api'
import CampusPage from '../src/site/pages/CampusPage.vue'

const api = vi.hoisted(() => ({ status: vi.fn(), dataset: vi.fn(), message: vi.fn(), confirm: vi.fn(), start: vi.fn() }))
vi.mock('@/runtime/campus-api', async original => ({
  ...await original<typeof import('../src/runtime/campus-api')>(), campusAPI: api,
}))
beforeEach(() => { sessionStorage.clear(); history.replaceState(null, '', '/campus') })
afterEach(() => { vi.restoreAllMocks(); vi.resetAllMocks() })
const status: CampusStatus = { enabled: true, binding: { maskedId: '***01', revision: 'first', needsAuthorization: false }, candidate: null }
function dataset(key: CampusDatasetKey): CampusDataset {
  return { key, status: 'ready', updatedAt: '2026-09-19T00:00:00Z', metrics: [], columns: [], rows: [], events: [], series: [] }
}
function setup(candidate: CampusStatus['candidate'] = null, userId = 42, revision = 'first') {
  api.status.mockResolvedValue({ ...status, binding: {...status.binding!, revision}, candidate })
  api.dataset.mockImplementation(async (key: CampusDatasetKey) => ({ ...dataset(key),
    metrics: key === 'summary' ? [{ label: '综合 GPA', value: '3.72', unit: '' }] : key === 'profile' ? [{ label: '姓名', value: '测试同学', unit: '' }] : key === 'calendar' ? [{ label: '教学周', value: '3', unit: '周' }] : [],
    messages: key === 'messages' ? Array.from({ length: 7 }, (_, i) => ({ id: String(i + 1), title: `校园通知 ${i + 1}`, publisher: '学校', publishedAt: '2026-09-19T10:00:00+08:00' })) : undefined,
  }))
  return mount(CampusPage, {
    props: { layout: { viewer: { isAuthenticated: true, id: userId } } as LayoutPayload, props: {} }, attachTo: document.body,
    global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh: { common: { loadingShort: '加载中' } } } })] },
  })
}
async function clickTab(wrapper: ReturnType<typeof setup>, label: string) {
  await wrapper.findAll('nav[aria-label="校园内容"] button').find(b => b.text() === label)!.trigger('click')
  await flushPromises()
}

test('home shows the personal day and latest messages without loading academic records or tool links', async () => {
  const wrapper = setup()
  try {
    await flushPromises()
    expect(wrapper.text()).toContain('测试同学')
    expect(wrapper.text()).toContain('第 3 周')
    expect(wrapper.text()).not.toContain('GPA')
    expect(wrapper.text()).not.toContain('学分进度')
    expect(wrapper.find('a[href="/schedule"]').exists()).toBe(false)
    expect(api.dataset.mock.calls.map(c => c[0])).not.toContain('grades')
    expect(api.dataset.mock.calls.map(c => c[0])).not.toContain('summary')
    expect(wrapper.text()).toContain('校园通知 5')
    expect(wrapper.text()).not.toContain('校园通知 6')
    await clickTab(wrapper, '学业记录')
    expect(wrapper.text()).toContain('3.72')
  } finally { wrapper.unmount() }
})

test('refresh after another tab changes identity clears the old identity even when new data fails', async () => {
  const wrapper = setup()
  try {
    await flushPromises()
    await clickTab(wrapper, '学业记录')
    expect(wrapper.text()).toContain('3.72')
    api.status.mockResolvedValue({ ...status, binding: { ...status.binding, revision: 'second', maskedId: '***02' } })
    api.dataset.mockRejectedValue(new Error('学校服务暂不可用'))
    await wrapper.get('button[aria-label="刷新校园数据"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('***02')
    expect(wrapper.text()).not.toContain('3.72')
  } finally { wrapper.unmount() }
})

test('message details render as text and are cleared when closed', async () => {
  const wrapper = setup()
  api.message.mockResolvedValue({ id: '1', title: '校园通知 1', publisher: '学校', publishedAt: '2026-09-19T10:00:00+08:00', content: '<script>unsafe()</script>正文', links: [] })
  try {
    await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('校园通知 1'))!.trigger('click')
    await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.textContent).toContain('<script>unsafe()</script>正文')
    expect(dialog.querySelector('script')).toBeNull()
    ;(dialog.querySelector('button[aria-label="关闭消息"]') as HTMLButtonElement).click()
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
  } finally { wrapper.unmount() }
})


test('closing a message ignores its late response', async () => {
  const wrapper = setup()
  let resolve!: (value: unknown) => void
  api.message.mockImplementation(() => new Promise(done => { resolve = done }))
  try {
    await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('校园通知 1'))!.trigger('click')
    await flushPromises()
    ;(document.querySelector('button[aria-label="关闭消息"]') as HTMLButtonElement).click()
    await flushPromises()
    resolve({ id: '1', content: '迟到的私密正文', links: [] })
    await flushPromises()
    expect(document.body.textContent).not.toContain('迟到的私密正文')
    expect(document.querySelector('[role="dialog"]')).toBeNull()
  } finally { wrapper.unmount() }
})


test('completed school authorization can be confirmed inside the message dialog and resumes that message', async () => {
  const wrapper = setup({ mode: 'reauthorize', maskedId: '***01', expiresAt: '2026-09-19T12:00:00Z' })
  api.message.mockRejectedValueOnce(new CampusError('campus.messageAuthorizationRequired', '需要更新授权'))
    .mockResolvedValue({ id: '1', title: '校园通知 1', publisher: '学校', publishedAt: '', content: '更新授权后的正文', links: [] })
  api.confirm.mockResolvedValue(null)
  try {
    await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('校园通知 1'))!.trigger('click')
    await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.textContent).toContain('学校认证已完成')
    const confirm = Array.from(dialog.querySelectorAll('button')).find(b => b.textContent === '确认更新授权')
    expect(confirm).toBeDefined()
    api.status.mockResolvedValue({ ...status, binding: { ...status.binding, revision: 'updated' } })
    confirm!.click()
    await flushPromises()
    expect(api.confirm).toHaveBeenCalledOnce()
    expect(api.message).toHaveBeenCalledTimes(2)
    expect(document.querySelector('[role="dialog"]')?.textContent).toContain('更新授权后的正文')
  } finally { wrapper.unmount() }
})


test('a failed confirmation stays visible inside the message dialog', async () => {
  const wrapper = setup({ mode: 'reauthorize', maskedId: '***01', expiresAt: '2026-09-19T12:00:00Z' })
  api.message.mockRejectedValue(new CampusError('campus.messageAuthorizationRequired', '需要更新授权'))
  api.confirm.mockRejectedValue(new Error('暂时无法保存授权，请重试'))
  try {
    await flushPromises()
    await wrapper.findAll('button').find(b => b.text().includes('校园通知 1'))!.trigger('click')
    await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    Array.from(dialog.querySelectorAll('button')).find(b => b.textContent === '确认更新授权')!.click()
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')?.textContent).toContain('暂时无法保存授权，请重试')
    expect(api.message).toHaveBeenCalledOnce()
  } finally { wrapper.unmount() }
})

for (const scenario of ['present', 'removed', 'different-account', 'different-binding', 'expired', 'failed']) {
 test(`school round trip resumes only a valid message intent: ${scenario}`, async () => {
  vi.spyOn(window.location, 'assign').mockImplementation(() => {})
  api.start.mockResolvedValue({ url: 'https://school.test/authorize' })
  api.message.mockRejectedValue(new CampusError('campus.messageAuthorizationRequired', '需要更新授权'))
  const first = setup()
  await flushPromises()
  await clickTab(first, '校园消息')
  await first.findAll('button').find(b => b.text().includes('校园通知 1'))!.trigger('click')
  await flushPromises()
  Array.from(document.querySelectorAll('[role="dialog"] button')).find(b => b.textContent === '更新学校授权')!.dispatchEvent(new MouseEvent('click'))
  await flushPromises()
  expect(api.start).toHaveBeenCalledWith('reauthorize')
  const saved = sessionStorage.getItem('yourtj:campus-message-return')
  expect(saved).not.toBeNull()
  expect(saved).not.toContain('校园通知')
  first.unmount()
  if (scenario === 'expired') {
   const intent = JSON.parse(saved!)
   intent.expiresAt = Date.now() - 1
   sessionStorage.setItem('yourtj:campus-message-return', JSON.stringify(intent))
  }
  history.replaceState(null, '', `/campus?authorization=${scenario === 'failed' ? 'failed' : 'ready'}`)
  const second = setup({ mode: 'reauthorize', maskedId: '***01', expiresAt: 'later' }, scenario === 'different-account' ? 99 : 42, scenario === 'different-binding' ? 'other' : 'first')
  if (scenario === 'removed') api.dataset.mockImplementation(async (key: CampusDatasetKey) => ({...dataset(key), messages: []}))
  api.message.mockReset().mockResolvedValue({id:'1', title:'新标题', content:'恢复后的正文', links:[]})
  api.confirm.mockResolvedValue(null)
  try {
   await flushPromises()
   expect(sessionStorage.getItem('yourtj:campus-message-return')).toBeNull()
   api.status.mockResolvedValue({...status, binding: {...status.binding!, revision:'renewed'}})
   await second.findAll('button').find(b => b.text() === '确认更新授权')!.trigger('click')
   await flushPromises()
   if (scenario === 'present') {
    expect(second.find('nav button[aria-current="page"]').text()).toBe('校园消息')
    expect(api.message).toHaveBeenCalledWith('1', expect.any(AbortSignal))
    expect(document.querySelector('[role="dialog"]')?.textContent).toContain('恢复后的正文')
   } else { expect(api.message).not.toHaveBeenCalled() }
  } finally { second.unmount() }
 })
}
