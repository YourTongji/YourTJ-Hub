// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import MarkdownImageViewer from '@/components/MarkdownImageViewer.vue'
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
  test('traps focus and restores the trigger and body overflow after Escape', async () => {
    const trigger = document.createElement('button')
    document.body.append(trigger)
    trigger.focus()
    document.body.style.overflow = 'auto'
    const wrapper = mount(MarkdownImageViewer, { global: { plugins: [i18n] }, attachTo: document.body })
    wrapper.vm.open([{ src: '/uploads/focus.png', alt: '' }], 0)
    await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.contains(document.activeElement)).toBe(true)
    expect(document.body.style.overflow).toBe('hidden')
    const buttons = dialog.querySelectorAll('button')
    const last = buttons[buttons.length - 1]!
    last.focus()
    last.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(buttons[0])
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    expect(document.activeElement).toBe(trigger)
    expect(document.body.style.overflow).toBe('auto')
    wrapper.unmount()
    trigger.remove()
    document.body.style.overflow = ''
  })

  test('supports touch swipes, thumbnail selection and backdrop close', async () => {
    const wrapper = mount(MarkdownImageViewer, { global: { plugins: [i18n] }, attachTo: document.body })
    wrapper.vm.open([{ src: '/a.png', alt: 'A' }, { src: '/b.png', alt: 'B' }], 0)
    await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    const touch = (type: string, key: string, x: number, y: number) => {
      const event = new Event(type, { bubbles: true })
      Object.defineProperty(event, key, { value: [{ clientX: x, clientY: y }] })
      dialog.dispatchEvent(event)
    }
    touch('touchstart', 'touches', 100, 100)
    touch('touchend', 'changedTouches', 100, 200)
    await flushPromises()
    expect(dialog.querySelector('img')?.getAttribute('src')).toBe('/a.png')
    touch('touchstart', 'touches', 100, 100)
    touch('touchend', 'changedTouches', 20, 105)
    await flushPromises()
    expect(dialog.querySelector('img')?.getAttribute('src')).toBe('/b.png')
    ;(dialog.querySelector('button[aria-pressed="false"]') as HTMLButtonElement).click()
    await flushPromises()
    expect(dialog.querySelector('img')?.getAttribute('src')).toBe('/a.png')
    dialog.querySelector('img')!.parentElement!.click()
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    wrapper.unmount()
  })

})
