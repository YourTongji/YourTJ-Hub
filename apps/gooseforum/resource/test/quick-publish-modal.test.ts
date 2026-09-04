// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import QuickPublishModal from '../src/site/components/QuickPublishModal.vue'
import { i18n } from '../src/runtime/i18n'
import { useQuickPublish } from '../src/site/composables/useQuickPublish'
import type { LayoutPayload } from '@gooseforum/client'

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
  theme: { enabled: true, current: 'gf-light', themeColor: '#3b82f6' },
}

describe('QuickPublishModal 组件', () => {
  test('当 quickPublishOpen 为 true 时弹出模态窗口并展示分类', async () => {
    const { openQuickPublish, closeQuickPublish } = useQuickPublish()
    openQuickPublish(1) // 提问类型

    const wrapper = mount(QuickPublishModal, {
      props: { layout: mockLayout },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })
    await flushPromises()

    const dialog = document.body.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()

    // 检查分类标签
    const categoryButtons = document.body.querySelectorAll('button')
    const categoryTexts = Array.from(categoryButtons).map((b) => b.textContent)
    expect(categoryTexts.some((t) => t?.includes('学术讨论'))).toBe(true)

    // 关闭弹层
    closeQuickPublish()
    await flushPromises()
    wrapper.unmount()
  })

  test('图片列表渲染 1, 2... 次序徽章并支持左右移动调整顺序', async () => {
    i18n.global.locale.value = 'zh'
    const { openQuickPublish, closeQuickPublish } = useQuickPublish()
    openQuickPublish(2) // 瞬间类型

    const wrapper = mount(QuickPublishModal, {
      props: { layout: mockLayout },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })
    await flushPromises()

    // 注入模拟图片
    const vm = wrapper.vm as any
    vm.uploadedImages = [
      { id: 'img_1', url: '/img/first.webp', alt: 'first' },
      { id: 'img_2', url: '/img/second.webp', alt: 'second' },
    ]
    await flushPromises()

    // 验证次序徽章渲染
    const badges = document.body.querySelectorAll('span.font-mono')
    const badgeTexts = Array.from(badges).map((b) => b.textContent?.trim())
    expect(badgeTexts).toContain('1')
    expect(badgeTexts).toContain('2')

    // 验证向右移动第一张图片
    const moveRightBtn = document.body.querySelector(`button[title="${i18n.global.t('publish.modal.moveImageRight')}"]`) as HTMLButtonElement | null
    expect(moveRightBtn).not.toBeNull()
    moveRightBtn?.click()
    await flushPromises()

    // 验证顺序已调换
    expect(vm.uploadedImages[0].url).toBe('/img/second.webp')
    expect(vm.uploadedImages[1].url).toBe('/img/first.webp')

    closeQuickPublish()
    await flushPromises()
    wrapper.unmount()
  })

  test('编辑模式下（openQuickPublishEdit）正确回显标题、正文、分类与图片，并呈现编辑与保存按钮', async () => {
    i18n.global.locale.value = 'zh'
    const { openQuickPublishEdit, closeQuickPublish } = useQuickPublish()
    openQuickPublishEdit({
      topicId: 888,
      contentType: 2, // 瞬间
      title: '原有瞬间标题',
      content: '原有瞬间正文内容',
      categoryIds: [102],
      images: ['/img/existing1.png'],
    })

    const wrapper = mount(QuickPublishModal, {
      props: { layout: mockLayout },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })
    await flushPromises()

    const dialog = document.body.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()

    // 验证标题回显
    const titleInput = dialog?.querySelector('input[type="text"]') as HTMLInputElement | null
    expect(titleInput?.value).toBe('原有瞬间标题')

    // 验证提交按钮文案显示“保存”
    const submitBtn = dialog?.querySelector('.gf-button-primary')
    expect(submitBtn?.textContent).toContain(i18n.global.t('common.save'))

    // 验证顶栏徽章显示“编辑瞬间”，且无多余重复的“编辑”后置文案
    const typeBadge = dialog?.querySelector('span.rounded-full.border')
    expect(typeBadge?.textContent).toContain('编辑瞬间')

    // 验证移动端缩略图卡片包含放大比例与圆角类名 h-[86px] w-[86px]
    const imageCard = dialog?.querySelector('.group.relative')
    expect(imageCard?.className).toContain('h-[86px]')
    expect(imageCard?.className).toContain('w-[86px]')
    expect(imageCard?.className).toContain('sm:h-20')
    expect(imageCard?.className).toContain('rounded-2xl')

    // 验证图片回显
    const vm = wrapper.vm as any
    expect(vm.uploadedImages.length).toBe(1)
    expect(vm.uploadedImages[0].url).toBe('/img/existing1.png')

    closeQuickPublish()
    await flushPromises()
    wrapper.unmount()
  })

  test('编辑问题模式下左上角徽章显示“编辑问题”', async () => {
    i18n.global.locale.value = 'zh'
    const { openQuickPublishEdit, closeQuickPublish } = useQuickPublish()
    openQuickPublishEdit({
      topicId: 999,
      contentType: 1, // 提问
      title: '原有问题标题',
      content: '原有问题内容',
      categoryIds: [101],
    })

    const wrapper = mount(QuickPublishModal, {
      props: { layout: mockLayout },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })
    await flushPromises()

    const dialog = document.body.querySelector('[role="dialog"]')
    const typeBadge = dialog?.querySelector('span.rounded-full.border')
    expect(typeBadge?.textContent).toContain('编辑问题')

    closeQuickPublish()
    await flushPromises()
    wrapper.unmount()
  })

  test('关闭弹层时平滑退场，editPayload 不会立即清空，避免退场动画闪回“快速发布”', async () => {
    const { openQuickPublishEdit, closeQuickPublish, quickPublishOpen, quickPublishEditPayload } = useQuickPublish()
    openQuickPublishEdit({
      topicId: 888,
      contentType: 2,
      title: '原有瞬间标题',
      content: '原有瞬间内容',
      categoryIds: [101],
    })

    expect(quickPublishOpen.value).toBe(true)
    expect(quickPublishEditPayload.value?.topicId).toBe(888)

    // 调用关闭时，弹层立即关闭，但 payload 在退场过渡期仍然存在
    closeQuickPublish()
    expect(quickPublishOpen.value).toBe(false)
    expect(quickPublishEditPayload.value?.topicId).toBe(888)

    // 等待退场过渡结束后被异步清空
    await new Promise((resolve) => setTimeout(resolve, 300))
    expect(quickPublishEditPayload.value).toBeNull()
  })
})
