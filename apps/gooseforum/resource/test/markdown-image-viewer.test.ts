// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import MarkdownImageViewer from '@/site/components/MarkdownImageViewer.vue'
import { i18n } from '../src/runtime/i18n'
import { nextTick } from 'vue'

describe('MarkdownImageViewer', () => {
  test('打开单图时正常呈现图片与控制按钮，不渲染多图翻页器与缩略图', async () => {
    const wrapper = mount(MarkdownImageViewer, {
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    })

    // 初始关闭
    expect(document.querySelector('[role="dialog"]')).toBeNull()

    // 打开单张图片
    wrapper.vm.open([{ src: '/uploads/single.png', alt: '测试单图' }], 0)
    await nextTick()
    await flushPromises()

    const dialog = document.querySelector('[role="dialog"]')
    expect(dialog).toBeTruthy()
    expect(dialog?.textContent).toContain('测试单图')

    // 单图不应该有翻页计数器
    expect(dialog?.querySelector('.font-mono')).toBeNull()

    wrapper.unmount()
  })

  test('打开多图时呈现页码指示器、翻页按钮与底部缩略图序列', async () => {
    const wrapper = mount(MarkdownImageViewer, {
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    })

    const sampleImages = [
      { src: '/uploads/img1.png', alt: '图片一' },
      { src: '/uploads/img2.png', alt: '图片二' },
      { src: '/uploads/img3.png', alt: '图片三' },
    ]

    wrapper.vm.open(sampleImages, 0)
    await nextTick()
    await flushPromises()

    const dialog = document.querySelector('[role="dialog"]')
    expect(dialog).toBeTruthy()

    // 验证多图页码 1 / 3
    expect(dialog?.textContent).toContain('1 / 3')

    // 验证切换下一张
    wrapper.vm.showNext()
    await nextTick()
    expect(dialog?.textContent).toContain('2 / 3')

    // 验证切换上一张
    wrapper.vm.showPrevious()
    await nextTick()
    expect(dialog?.textContent).toContain('1 / 3')

    // 验证缩放切换
    expect(wrapper.vm.isZoomed).toBe(false)
    wrapper.vm.toggleZoom()
    expect(wrapper.vm.isZoomed).toBe(true)

    // 验证关闭
    wrapper.vm.close()
    await nextTick()
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()

    wrapper.unmount()
  })
})
