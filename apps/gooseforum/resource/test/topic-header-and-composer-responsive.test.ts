// @vitest-environment happy-dom
import { describe, it, expect } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TopicPage from '../src/site/pages/TopicPage.vue'
import PostComposer from '../src/site/components/PostComposer.vue'
import { createI18n } from 'vue-i18n'
import zh from '../src/locales/zh'

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: { zh },
})

describe('TopicPage 移动端与桌面端元数据自适应布局', () => {
  it('渲染桌面端完整横排元数据栏与移动端两行优化元数据栏', async () => {
    const topic = {
      id: 101,
      title: '测试响应式元数据',
      contentType: 3, // 文章
      createdAt: '2026-08-06 17:03:00',
      replyCount: 12,
      viewCount: 345,
      likeCount: 67,
      isLiked: false,
      isBookmarked: false,
      author: {
        id: 1,
        username: 'testuser',
        nickname: '测试作者',
        avatarUrl: '/avatar.png',
      },
      categories: [
        { id: 2, name: '闲聊茶馆', color: '#f59e0b', url: '/c/chat' },
      ],
    }

    const wrapper = mount(TopicPage, {
      props: {
        layout: { viewer: null } as any,
        props: {
          topic,
          posts: [],
          totalPosts: 1,
          replyTarget: null,
          postWindow: null,
          postStream: { posts: [] },
          hotTopics: [],
          permissions: { canPost: true },
        } as any,
      },
      global: {
        plugins: [i18n],
        stubs: {
          UserAvatar: true,
          PostStream: true,
          Breadcrumb: true,
          PostStreamFloatingActions: true,
        },
      },
    })
    await flushPromises()

    // 桌面端容器存在（hidden sm:flex）
    const desktopBar = wrapper.find('.hidden.sm\\:flex')
    expect(desktopBar.exists()).toBe(true)
    expect(desktopBar.text()).toContain('闲聊茶馆')
    expect(desktopBar.text()).toContain('345')

    // 移动端容器存在（sm:hidden flex flex-col）
    const mobileBar = wrapper.find('.sm\\:hidden.flex.flex-col')
    expect(mobileBar.exists()).toBe(true)

    // 移动端第一行包含作者和分区
    const row1 = mobileBar.find('.flex.items-center.justify-between')
    expect(row1.exists()).toBe(true)
    expect(row1.text()).toContain('闲聊茶馆')

    // 移动端第二行包含时间与回复、浏览、点赞三项计数项（同排）
    const row2 = mobileBar.findAll('.flex.items-center.justify-between')[1]
    expect(row2.exists()).toBe(true)
    expect(row2.text()).toContain('12')
    expect(row2.text()).toContain('345')
    expect(row2.text()).toContain('67')

    wrapper.unmount()
  })
})

describe('PostComposer 独立类名防护与移动端防压缩高度保护', () => {
  it('dialog 具备 gf-composer-surface 类名与 min-h-[160px] 保护，避免污染通用浮层', async () => {
    const wrapper = mount(PostComposer, {
      props: {
        open: true,
        authenticated: true,
        submitting: false,
        errorMessage: '',
        successMessage: '',
      },
      global: {
        plugins: [i18n],
        stubs: {
          VditorOfficial: {
            template: '<div class="vditor-mock" />',
            methods: { focus() {} },
          },
          TurnstileWidget: true,
        },
      },
      attachTo: document.body,
    })
    await flushPromises()

    const dialog = document.body.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.className).toContain('gf-composer-surface')

    const editorArea = dialog?.querySelector('.min-h-\\[160px\\]')
    expect(editorArea).not.toBeNull()

    const resizeHandle = dialog?.querySelector('.composer-resize-handle')
    expect(resizeHandle).not.toBeNull()
    expect(resizeHandle?.getAttribute('role')).toBe('separator')

    wrapper.unmount()
  })
})
