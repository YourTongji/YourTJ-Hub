// @vitest-environment happy-dom
import { describe, expect, test, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import TopicImageGallery from '../src/site/components/TopicImageGallery.vue'
import { i18n } from '../src/runtime/i18n'

describe('TopicImageGallery 组件', () => {
  beforeEach(() => {
    // 画廊无障碍文案走 i18n（common.* / publish.gallery.*），固定英语便于稳定断言
    i18n.global.locale.value = 'en'
  })

  const sampleImages = [

    '/uploads/image1.jpg',
    '/uploads/image2.png',
    '/uploads/image3.webp',
  ]

  test('图片为空时优雅不渲染容器', () => {
    const wrapper = mount(TopicImageGallery, {
      props: { images: [] },
      global: { plugins: [i18n] },
    })
    expect(wrapper.find('.group').exists()).toBe(false)
  })

  test('正确渲染首张图片与高优先级加载 fetchpriority="high"', () => {
    const wrapper = mount(TopicImageGallery, {
      props: { images: sampleImages, title: '测试画廊' },
      global: { plugins: [i18n] },
    })

    const mainImg = wrapper.find('img[alt="测试画廊 - 1"]')
    expect(mainImg.exists()).toBe(true)
    expect(mainImg.attributes('src')).toBe('/uploads/image1.jpg')
    expect(mainImg.attributes('fetchpriority')).toBe('high')
  })

  test('多张图片时展示 1/N 计数徽章与左右切换控件', async () => {
    const wrapper = mount(TopicImageGallery, {
      props: { images: sampleImages },
      global: { plugins: [i18n] },
    })

    // 检查页码徽章
    expect(wrapper.text()).toContain('1/3')

    // 检查右翻页按钮
    const nextBtn = wrapper.find(`button[aria-label="${i18n.global.t('common.nextPage')}"]`)
    expect(nextBtn.exists()).toBe(true)

    // 点击翻页
    await nextBtn.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('2/3')
    const secondImg = wrapper.find('img[alt="Image 2"]')
    expect(secondImg.exists()).toBe(true)
    expect(secondImg.attributes('src')).toBe('/uploads/image2.png')

    // 点击左翻页
    const prevBtn = wrapper.find(`button[aria-label="${i18n.global.t('common.previousPage')}"]`)
    expect(prevBtn.exists()).toBe(true)
    await prevBtn.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('1/3')
  })

  test('点击放大按钮可以唤起全屏大图预览 Lightbox', async () => {
    const wrapper = mount(TopicImageGallery, {
      props: { images: sampleImages },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })

    const zoomBtn = wrapper.find(`button[aria-label="${i18n.global.t('publish.gallery.zoomIn')}"]`)
    expect(zoomBtn.exists()).toBe(true)

    await zoomBtn.trigger('click')
    await flushPromises()

    const closeBtn = document.body.querySelector(`button[aria-label="${i18n.global.t('common.close')}"]`)
    expect(closeBtn).not.toBeNull()

    // 关闭 Lightbox
    closeBtn?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()

    wrapper.unmount()
  })
})
