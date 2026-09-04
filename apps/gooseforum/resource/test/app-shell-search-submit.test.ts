// @vitest-environment happy-dom
import { vi } from 'vitest'
import { afterEach, describe, expect, test } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { i18n } from '../src/runtime/i18n'
import AppShell from '../src/site/components/AppShell.vue'
import type { LayoutPayload } from '@gooseforum/client'

function minimalLayout(): LayoutPayload {
  return {
    site: {
      name: 'yourtj',
      description: '',
      logo: '',
      favicon: '',
      brandType: 'text',
      brandText: 'yourtj',
      brandImage: '',
    },
    viewer: {
      id: 0,
      username: '',
      email: '',
      avatarUrl: '',
      isAuthenticated: false,
      canAccessAdmin: false,
      isModerator: false,
      requiresEmailVerification: false,
      adminPermissions: [],
    },
    sidebar: { main: [], resources: [], groups: [], categories: [], activeKey: 'topics' },
    footer: { links: [], primary: [] },
    unread: { notifications: false, messages: false, moderationReports: false },
    theme: { enabled: false, current: 'gf-light', themeColor: '#fbfdff' },
  }
}

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
}

async function mountShell(router: Router): Promise<VueWrapper> {
  const wrapper = mount(AppShell, {
    props: { layout: minimalLayout() },
    global: { plugins: [i18n, router] },
    attachTo: document.body,
  })
  return wrapper
}

// 回归测试（issue #446，与 SearchPage 提交同一体验问题的另一处实例）：
// navbar 搜索栏此前用 document.createElement('a').click() 触发全局 a[href]
// 点击拦截器转成 SPA 路由，但游离节点的事件路径只有它自己，冒泡不到
// document，拦截器永远收不到，浏览器执行默认行为——整页导航白屏
// （无顶部 loading bar）。提交必须走 router.push 的 X-Goose-Page JSON
// 拉取路径，与 SearchPage 对齐。
describe('AppShell navbar 搜索表单提交', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  test('回车提交走客户端路由跳转（不再整页导航）', async () => {
    const router = makeRouter()
    const wrapper = await mountShell(router)
    await wrapper.get('input[type="search"]').setValue('Vue 白屏')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/search')
    expect(router.currentRoute.value.query.q).toBe('Vue 白屏')
    wrapper.unmount()
  })

  test('查询词首尾空白被裁剪（与后端 buildSearchURL 语义一致）', async () => {
    const router = makeRouter()
    const wrapper = await mountShell(router)
    await wrapper.get('input[type="search"]').setValue('  选课  ')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(router.currentRoute.value.query.q).toBe('选课')
    wrapper.unmount()
  })

  test('空查询不导航、聚焦输入框', async () => {
    const router = makeRouter()
    const wrapper = await mountShell(router)
    const input = wrapper.get('input[type="search"]')
    const focusSpy = vi.spyOn(input.element, 'focus')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/')
    expect(focusSpy).toHaveBeenCalled()
    focusSpy.mockRestore()
    wrapper.unmount()
  })
})