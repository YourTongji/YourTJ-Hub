// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { i18n } from '../src/runtime/i18n'
import DraftsPage from '../src/site/pages/DraftsPage.vue'

const { deleteTopic, reportContentEvent, queueFlashMessage, pushFlashMessage } = vi.hoisted(() => ({
  deleteTopic: vi.fn(),
  reportContentEvent: vi.fn(),
  queueFlashMessage: vi.fn(),
  pushFlashMessage: vi.fn(),
}))

vi.mock('@/runtime/api', async importOriginal => ({
  ...await importOriginal<typeof import('../src/runtime/api')>(),
  deleteTopic,
  reportContentEvent,
}))
vi.mock('@/runtime/flash-message', () => ({ queueFlashMessage, useFlashMessages: () => ({ push: pushFlashMessage }) }))

const PageHeader = defineComponent({
  template: '<div><slot name="badge" /><slot name="actions" /></div>',
})
const EmptyState = defineComponent({
  props: { title: String, description: String },
  template: '<div data-test="empty-state"><h2>{{ title }}</h2><p>{{ description }}</p><slot /></div>',
})

let wrapper: VueWrapper | undefined

function makeDraft(id = 42, title = 'A saved draft') {
  return {
    id,
    title,
    description: 'Draft body preview',
    editUrl: `/publish?id=${id}`,
    replyCount: 0,
    viewCount: 0,
    processStatus: 0,
    updatedAt: '2026-10-16T00:00:00Z',
    createdAt: '2026-10-15T00:00:00Z',
    categories: [],
  }
}

async function mountPage(drafts = [makeDraft()]) {
  deleteTopic.mockReset().mockResolvedValue(true)
  reportContentEvent.mockReset().mockResolvedValue(true)
  queueFlashMessage.mockReset()
  pushFlashMessage.mockReset()
  wrapper = mount(DraftsPage, {
    props: {
      layout: {} as any,
      props: {
        drafts,
        total: drafts.length,
        pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
      } as any,
    },
    global: {
      plugins: [i18n],
      stubs: { PageHeader, EmptyState },
    },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

function deleteButton(title = 'A saved draft') {
  return document.body.querySelector<HTMLButtonElement>(`button[aria-label="${i18n.global.t('drafts.deleteLabel', { title })}"]`)
}

function dialogButton(testId: string) {
  return document.body.querySelector<HTMLButtonElement>(`[data-test="${testId}"]`)
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('Web 草稿箱删除', () => {
  test('每条草稿有与继续编辑区分的删除入口', async () => {
    const view = await mountPage()

    expect(view.text()).toContain(i18n.global.t('drafts.edit'))
    expect(deleteButton()).not.toBeNull()
    expect(deleteButton()?.textContent).toContain(i18n.global.t('drafts.delete'))
  })

  test('取消确认不会调用删除接口或移除草稿', async () => {
    const view = await mountPage()

    deleteButton()?.click()
    await flushPromises()
    expect(document.body.querySelector('[role="alertdialog"]')?.textContent).toContain('A saved draft')

    dialogButton('cancel-delete')?.click()
    await flushPromises()

    expect(deleteTopic).not.toHaveBeenCalled()
    expect(view.text()).toContain('A saved draft')
    expect(view.find('[data-test="empty-state"]').exists()).toBe(false)
  })

  test('确认后删除云端草稿并在只剩一条时显示空状态', async () => {
    const view = await mountPage()

    deleteButton()?.click()
    await flushPromises()
    dialogButton('confirm-delete')?.click()
    await flushPromises()

    expect(deleteTopic).toHaveBeenCalledOnce()
    expect(deleteTopic).toHaveBeenCalledWith(42)
    expect(view.find('[data-test="empty-state"]').exists()).toBe(true)
    expect(view.text()).not.toContain('A saved draft')
    expect(pushFlashMessage).toHaveBeenCalledExactlyOnceWith(i18n.global.t('drafts.deleteSuccess'), 'success')
    expect(queueFlashMessage).not.toHaveBeenCalled()
  })

  test('满 100 条窗口时仅排队一次成功提示并重载补齐列表', async () => {
    await mountPage(Array.from({ length: 100 }, (_, i) => makeDraft(i + 1, `Draft ${i + 1}`)))
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {})
    deleteButton('Draft 1')?.click()
    await flushPromises()
    dialogButton('confirm-delete')?.click()
    await flushPromises()
    expect(queueFlashMessage).toHaveBeenCalledExactlyOnceWith(i18n.global.t('drafts.deleteSuccess'), 'success')
    expect(reload).toHaveBeenCalledOnce()
    expect(queueFlashMessage.mock.invocationCallOrder[0]).toBeLessThan(reload.mock.invocationCallOrder[0]!)
    expect(pushFlashMessage).not.toHaveBeenCalled()
  })

  test('失败保留原草稿、显示错误并允许重试', async () => {
    const view = await mountPage()
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {})
    deleteTopic.mockRejectedValueOnce(new Error('You cannot delete this draft'))

    deleteButton()?.click()
    await flushPromises()
    dialogButton('confirm-delete')?.click()
    await flushPromises()

    expect(document.body.querySelector('[role="alert"]')?.textContent).toContain('You cannot delete this draft')
    expect(view.text()).toContain('A saved draft')
    expect(view.find('[data-test="empty-state"]').exists()).toBe(false)
    expect(dialogButton('confirm-delete')?.disabled).toBe(false)
    expect(queueFlashMessage).not.toHaveBeenCalled()
    expect(pushFlashMessage).not.toHaveBeenCalled()
    expect(reload).not.toHaveBeenCalled()

    dialogButton('confirm-delete')?.click()
    await flushPromises()
    expect(deleteTopic).toHaveBeenCalledTimes(2)
    expect(view.find('[data-test="empty-state"]').exists()).toBe(true)
  })

  test('删除被拒绝时保留列表并显示失败，不发成功提示或重载', async () => {
    const view = await mountPage()
    const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {})
    deleteTopic.mockResolvedValueOnce(false)
    deleteButton()?.click()
    await flushPromises()
    dialogButton('confirm-delete')?.click()
    await flushPromises()
    expect(view.text()).toContain('A saved draft')
    expect(document.body.querySelector('[role="alert"]')?.textContent).toContain(i18n.global.t('drafts.deleteFailed'))
    expect(queueFlashMessage).not.toHaveBeenCalled()
    expect(pushFlashMessage).not.toHaveBeenCalled()
    expect(reload).not.toHaveBeenCalled()
  })

  test('删除期间忽略重复确认提交', async () => {
    const view = await mountPage()
    let finishDelete!: (result: boolean) => void
    deleteTopic.mockImplementationOnce(() => new Promise<boolean>(resolve => { finishDelete = resolve }))

    deleteButton()?.click()
    await flushPromises()
    dialogButton('confirm-delete')?.click()
    await flushPromises()

    expect(deleteTopic).toHaveBeenCalledOnce()
    expect(dialogButton('confirm-delete')?.disabled).toBe(true)
    dialogButton('confirm-delete')?.click()
    expect(deleteTopic).toHaveBeenCalledOnce()

    finishDelete(true)
    await flushPromises()
    expect(view.find('[data-test="empty-state"]').exists()).toBe(true)
  })
})
