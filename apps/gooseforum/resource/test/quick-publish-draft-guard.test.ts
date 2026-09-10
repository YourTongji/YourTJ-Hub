// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import QuickPublishModal from '../src/site/components/QuickPublishModal.vue'
import { i18n } from '../src/runtime/i18n'
import { useQuickPublish } from '../src/site/composables/useQuickPublish'
import { readQuickPublishDraft, writeQuickPublishDraft } from '../src/site/utils/quick-publish-draft'
import * as api from '../src/runtime/api'
import type { LayoutPayload } from '@gooseforum/client'

// happy-dom 20 把 localStorage 定义为原型 getter，vitest 的 populateGlobal 只复制
// 自有属性，导致测试环境缺失 localStorage（dev 上既有 7 个测试文件同样受影响）。
// 这里在测试内提供最小内存实现，仅服务于本文件的暂存断言。
const memoryStorage = new Map<string, string>()
Object.defineProperty(globalThis, 'localStorage', {
  value: {
    getItem: (key: string) => memoryStorage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      memoryStorage.set(key, String(value))
    },
    removeItem: (key: string) => {
      memoryStorage.delete(key)
    },
    clear: () => {
      memoryStorage.clear()
    },
    key: (index: number) => Array.from(memoryStorage.keys())[index] ?? null,
    get length() {
      return memoryStorage.size
    },
  },
  configurable: true,
})

const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/', component: { template: '<div>home</div>' } }],
})

const mockLayout: LayoutPayload = {
  site: { name: 'GooseForum', brandType: 'default', brandText: 'GooseForum' } as any,
  viewer: { isAuthenticated: true, id: 1, username: 'Tester', avatarUrl: '/avatar.png' } as any,
  sidebar: {
    categories: [
      { id: 101, label: '学术讨论', color: '#10b981', url: '/c/101' },
      { id: 102, label: '日常生活', color: '#3b82f6', url: '/c/102' },
    ],
    activeKey: '',
  },
  footer: { links: [], primary: [] },
  unread: { notifications: 0, messages: 0 },
  posting: { maxTitleLength: 100 },
  theme: { enabled: true, current: 'gf-light', themeColor: '#3b82f6' },
  insightFlareEnabled: false,
}

async function mountModal(type: 0 | 1 | 2 = 2) {
  const { openQuickPublish, closeQuickPublish, quickPublishOpen, quickPublishEditPayload } = useQuickPublish()
  openQuickPublish(type)
  const wrapper = mount(QuickPublishModal, {
    props: { layout: mockLayout },
    global: { plugins: [i18n, router] },
    attachTo: document.body,
  })
  await flushPromises()
  return { wrapper, vm: wrapper.vm as any, closeQuickPublish, quickPublishOpen, quickPublishEditPayload }
}

function buttonByText(text: string): HTMLButtonElement | null {
  return Array.from(document.body.querySelectorAll('button')).find((b) => b.textContent?.includes(text)) ?? null
}

beforeEach(() => {
  i18n.global.locale.value = 'zh'
  localStorage.clear()
  const { quickPublishOpen, quickPublishEditPayload } = useQuickPublish()
  quickPublishOpen.value = false
  quickPublishEditPayload.value = null
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('QuickPublishModal 草稿与离开保护（issue #583）', () => {
  test('空字段关闭立即关闭，不弹离开确认', async () => {
    const { wrapper, vm, quickPublishOpen } = await mountModal(2)
    try {
      expect(quickPublishOpen.value).toBe(true)
      vm.requestClose()
      await nextTick()
      expect(quickPublishOpen.value).toBe(false)
      expect(vm.leavePromptOpen).toBe(false)
      expect(document.body.querySelector('[role="alertdialog"]')).toBeNull()
    } finally {
      wrapper.unmount()
    }
  })

  test('有输入内容时关闭弹出离开确认且弹层不关闭', async () => {
    const { wrapper, vm, quickPublishOpen } = await mountModal(2)
    try {
      vm.title = '测试标题'
      await nextTick()
      vm.requestClose()
      await nextTick()
      expect(quickPublishOpen.value).toBe(true)
      expect(vm.leavePromptOpen).toBe(true)
      expect(document.body.querySelector('[role="alertdialog"]')).not.toBeNull()
    } finally {
      wrapper.unmount()
    }
  })

  test('继续编辑关闭确认并保持弹层打开', async () => {
    const { wrapper, vm, quickPublishOpen } = await mountModal(2)
    try {
      vm.title = '测试标题'
      await nextTick()
      vm.requestClose()
      await nextTick()
      expect(vm.leavePromptOpen).toBe(true)

      buttonByText(i18n.global.t('publish.continueEditing'))?.click()
      await nextTick()
      expect(vm.leavePromptOpen).toBe(false)
      expect(quickPublishOpen.value).toBe(true)
    } finally {
      wrapper.unmount()
    }
  })

  test('不保存离开关闭弹层并清除本地暂存', async () => {
    writeQuickPublishDraft(1, 2, { title: '旧标题', content: '旧正文', categoryIds: [101], images: [] })
    const { wrapper, vm, quickPublishOpen } = await mountModal(2)
    try {
      expect(vm.title).toBe('旧标题')
      vm.title = '改过的标题'
      await nextTick()
      vm.requestClose()
      await nextTick()
      expect(vm.leavePromptOpen).toBe(true)

      buttonByText(i18n.global.t('publish.leaveWithoutSaving'))?.click()
      await nextTick()
      expect(quickPublishOpen.value).toBe(false)
      expect(readQuickPublishDraft(1, 2)).toBeNull()
    } finally {
      wrapper.unmount()
    }
  })

  test('输入自动暂存 localStorage，重新打开同类型恢复并提示', async () => {
    vi.useFakeTimers()
    const { openQuickPublish, closeQuickPublish, quickPublishOpen } = useQuickPublish()
    openQuickPublish(2)
    const wrapper = mount(QuickPublishModal, {
      props: { layout: mockLayout },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })
    try {
      await nextTick()
      const vm = wrapper.vm as any
      vm.title = '自动暂存标题'
      vm.content = '自动暂存正文'
      await nextTick()
      await vi.advanceTimersByTimeAsync(600)

      const stash = readQuickPublishDraft(1, 2)
      expect(stash?.title).toBe('自动暂存标题')
      expect(stash?.content).toBe('自动暂存正文')

      closeQuickPublish()
      await nextTick()
      expect(quickPublishOpen.value).toBe(false)

      openQuickPublish(2)
      await nextTick()
      expect(vm.title).toBe('自动暂存标题')
      expect(vm.content).toBe('自动暂存正文')
      expect(vm.draftRestored).toBe(true)
      expect(document.body.querySelector('[data-test="quick-publish-draft-restored"]')).not.toBeNull()
    } finally {
      wrapper.unmount()
    }
  })

  test('新建模式保存草稿：topicStatus 0 + contentType 2，弹层关闭且暂存清除', async () => {
    const submit = vi.spyOn(api, 'submitTopic').mockResolvedValue(123)
    const { wrapper, vm, quickPublishOpen } = await mountModal(2)
    try {
      vm.title = '草稿标题'
      vm.content = '草稿正文'
      vm.categoryIds = [101]
      await nextTick()

      const saveDraftBtn = buttonByText(i18n.global.t('publish.saveDraft'))
      expect(saveDraftBtn).not.toBeNull()
      saveDraftBtn?.click()
      await flushPromises()

      expect(submit).toHaveBeenCalledWith(expect.objectContaining({ topicStatus: 0, contentType: 2 }))
      expect(quickPublishOpen.value).toBe(false)
      expect(readQuickPublishDraft(1, 2)).toBeNull()
    } finally {
      wrapper.unmount()
    }
  })

  test('发布成功清除本地暂存', async () => {
    const submit = vi.spyOn(api, 'submitTopic').mockResolvedValue(55)
    writeQuickPublishDraft(1, 2, { title: '旧标题', content: '旧正文', categoryIds: [101], images: [] })
    const { wrapper, vm, quickPublishOpen } = await mountModal(2)
    try {
      vm.title = '发布标题'
      vm.content = '发布正文'
      vm.categoryIds = [101]
      await nextTick()
      await vm.handleSubmit()
      await flushPromises()

      expect(submit).toHaveBeenCalled()
      expect(quickPublishOpen.value).toBe(false)
      expect(readQuickPublishDraft(1, 2)).toBeNull()
    } finally {
      wrapper.unmount()
    }
  })

  test('beforeunload：脏且有内容时阻止，干净时不阻止', async () => {
    const { wrapper, vm } = await mountModal(2)
    try {
      const cleanEvent = new Event('beforeunload', { cancelable: true })
      window.dispatchEvent(cleanEvent)
      expect(cleanEvent.defaultPrevented).toBe(false)

      vm.title = '测试标题'
      await nextTick()
      const dirtyEvent = new Event('beforeunload', { cancelable: true })
      window.dispatchEvent(dirtyEvent)
      expect(dirtyEvent.defaultPrevented).toBe(true)
    } finally {
      wrapper.unmount()
    }
  })

  test('编辑模式：脏时弹确认，且确认面板与页脚均不渲染保存草稿按钮', async () => {
    const { openQuickPublishEdit, closeQuickPublish } = useQuickPublish()
    openQuickPublishEdit({
      topicId: 888,
      contentType: 2,
      title: '原有瞬间标题',
      content: '原有瞬间正文',
      categoryIds: [102],
      images: [],
    })
    const wrapper = mount(QuickPublishModal, {
      props: { layout: mockLayout },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })
    try {
      await flushPromises()
      const vm = wrapper.vm as any

      // 页脚无保存草稿按钮
      expect(buttonByText(i18n.global.t('publish.saveDraft'))).toBeNull()

      vm.title = '改过的标题'
      await nextTick()
      vm.requestClose()
      await nextTick()
      expect(vm.leavePromptOpen).toBe(true)

      const alert = document.body.querySelector('[role="alertdialog"]')
      expect(alert).not.toBeNull()
      const alertButtons = Array.from(alert!.querySelectorAll('button')).map((b) => b.textContent)
      expect(alertButtons.some((t) => t?.includes(i18n.global.t('publish.saveDraft')))).toBe(false)
    } finally {
      closeQuickPublish()
      await flushPromises()
      wrapper.unmount()
    }
  })
})

describe('draft isolation and flush regressions', () => {
  test('another account never restores the previous account draft', async () => {
    writeQuickPublishDraft(1, 2, { title: 'private account A', content: 'private', categoryIds: [101], images: [] })
    const previous = mockLayout.viewer
    mockLayout.viewer = { ...previous, id: 2 } as any
    const { wrapper, vm } = await mountModal()
    try { expect(vm.title).toBe('') } finally { wrapper.unmount(); mockLayout.viewer = previous }
  })

  test('refresh flushes the latest input before the debounce fires', async () => {
    vi.useFakeTimers()
    const { wrapper, vm } = await mountModal()
    try {
      vm.title = 'last keystroke'
      await nextTick()
      window.dispatchEvent(new Event('beforeunload', { cancelable: true }))
      expect(readQuickPublishDraft(1, 2)?.title).toBe('last keystroke')
    } finally { wrapper.unmount() }
  })

  test('clearing all fields removes the old stash', async () => {
    vi.useFakeTimers()
    const { wrapper, vm } = await mountModal()
    try {
      vm.title = 'old draft'
      await nextTick(); await vi.advanceTimersByTimeAsync(600)
      vm.title = ''
      await nextTick(); await vi.advanceTimersByTimeAsync(600)
      expect(readQuickPublishDraft(1, 2)).toBeNull()
    } finally { wrapper.unmount() }
  })

  test('restored unsaved text still needs a leave decision', async () => {
    writeQuickPublishDraft(1, 2, { title: 'unsaved', content: '', categoryIds: [], images: [] })
    const { wrapper, vm, quickPublishOpen } = await mountModal()
    try {
      vm.requestClose(); await nextTick()
      expect(quickPublishOpen.value).toBe(true)
      expect(vm.leavePromptOpen).toBe(true)
      expect(document.activeElement?.textContent).toContain(i18n.global.t('publish.continueEditing'))
    } finally { wrapper.unmount() }
  })

  test('upload-only edits cannot be closed without confirmation', async () => {
    const { wrapper, vm, quickPublishOpen } = await mountModal()
    try {
      vm.uploading = true
      vm.requestClose(); await nextTick()
      expect(quickPublishOpen.value).toBe(true)
      expect(vm.leavePromptOpen).toBe(true)
    } finally { wrapper.unmount() }
  })
})


test('expired and anonymous draft storage fails closed', () => {
  const stash = { title: 'private', content: '', categoryIds: [], images: [] }
  writeQuickPublishDraft(1, 2, stash)
  const key = [...memoryStorage.keys()][0]
  memoryStorage.set(key, JSON.stringify({ ...stash, updatedAt: Date.now() - 8 * 24 * 60 * 60 * 1000 }))
  expect(readQuickPublishDraft(1, 2)).toBeNull()
  writeQuickPublishDraft(0, 2, stash)
  expect(readQuickPublishDraft(0, 2)).toBeNull()
  expect(memoryStorage.size).toBe(0)
})

test('changing accounts closes an already open private draft', async () => {
  const { wrapper, vm, quickPublishOpen } = await mountModal()
  try {
    vm.title = 'private session'
    await wrapper.setProps({ layout: { ...mockLayout, viewer: { ...mockLayout.viewer, id: 2 } } as any })
    await nextTick()
    expect(quickPublishOpen.value).toBe(false)
    expect(vm.title).toBe('')
  } finally { wrapper.unmount() }
})
