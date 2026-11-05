// @vitest-environment happy-dom
import { vi } from 'vitest'
import { afterEach, describe, expect, test } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { i18n } from '../src/runtime/i18n'
import SearchPage from '../src/site/pages/SearchPage.vue'
import type { SearchPageProps } from '@gooseforum/client'

function emptyProps(overrides: Partial<SearchPageProps> = {}): SearchPageProps {
  return {
    query: '',
    scope: 'all',
    topics: [],
    users: [],
    categories: [],
    courses: [],
    total: 0,
    usersTotal: 0,
    categoriesTotal: 0,
    coursesTotal: 0,
    totalPages: 0,
    pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
    ...overrides,
  }
}

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
}

async function mountPage(router: Router, props: SearchPageProps): Promise<VueWrapper> {
  const wrapper = mount(SearchPage, {
    props: { layout: {}, props, pageUrl: '/search' },
    global: { plugins: [i18n, router] },
    attachTo: document.body,
  })
  return wrapper
}

// 回归测试（issue：搜索提交整页导航导致短暂白屏）：
// 页面 body 为空壳、内容由 Vue 客户端渲染，原生 GET 提交会先卸载整页再渲染，
// 期间画面空白。表单必须拦截 submit 走客户端路由（router.push），
// 保留 action/method 作为无 JS 环境的原生回退。
describe('SearchPage 搜索表单提交', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  test('提交查询走客户端路由跳转（不再整页导航）', async () => {
    const router = makeRouter()
    const wrapper = await mountPage(router, emptyProps())
    await wrapper.get('input[name="q"]').setValue('Vue 白屏')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/search')
    expect(router.currentRoute.value.query.q).toBe('Vue 白屏')
    wrapper.unmount()
  })

  test('当前 scope 非 all 时提交保留 scope 参数', async () => {
    const router = makeRouter()
    const wrapper = await mountPage(router, emptyProps({ scope: 'topics' }))
    expect(wrapper.get('input[name="scope"]').attributes('value')).toBe('topics')
    await wrapper.get('input[name="q"]').setValue('食堂')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/search')
    expect(router.currentRoute.value.query).toEqual({ q: '食堂', scope: 'topics' })
    wrapper.unmount()
  })

  test('查询词首尾空白被裁剪（与后端 buildSearchURL 语义一致）', async () => {
    const router = makeRouter()
    const wrapper = await mountPage(router, emptyProps())
    await wrapper.get('input[name="q"]').setValue('  选课  ')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(router.currentRoute.value.query.q).toBe('选课')
    wrapper.unmount()
  })

  test('空查询提交回到 /search 空状态页（不带多余参数）', async () => {
    const router = makeRouter()
    const wrapper = await mountPage(router, emptyProps())
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/search')
    wrapper.unmount()
  })

  test('已在 /search 空状态页再次提交不触发原生整页回退', async () => {
    const router = makeRouter()
    await router.push('/search')
    const wrapper = await mountPage(router, emptyProps())
    const form = wrapper.get('form').element as HTMLFormElement
    const submitSpy = vi.spyOn(form, 'submit').mockImplementation(() => {})
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    // duplicated navigation 属正常取消（非异常），绝不落入 form.submit() 原生回退
    expect(router.currentRoute.value.fullPath).toBe('/search')
    expect(submitSpy).not.toHaveBeenCalled()
    submitSpy.mockRestore()
    wrapper.unmount()
  })
})

describe('SearchPage scope counts', () => {
 test('all describes the combined total and does not label it as the topic total', async () => {
  const wrapper = await mountPage(makeRouter(), emptyProps({ query: '课程', total: 27, coursesTotal: 257 }))
  const tabs = wrapper.findAll('[role="group"] a')
  expect(tabs[0]!.find('span').exists()).toBe(false)
  expect(tabs[1]!.get('span').text()).toBe('27')
  expect(wrapper.text()).toContain('284')
  wrapper.unmount()
 })
 test.each(['topics', 'users', 'categories', 'courses'])('only queried %s has a known count after switching scope', async scope => {
  const wrapper = await mountPage(makeRouter(), emptyProps({ query: '课程', total: 27, coursesTotal: 257 }))
  await wrapper.setProps({ props: emptyProps({ query: '课程', scope, coursesTotal: scope === 'courses' ? 257 : 0 }) })
  const tabs = wrapper.findAll('[role="group"] a')
  expect(tabs.filter(tab => tab.find('span').exists())).toHaveLength(1)
  expect(tabs.find(tab => tab.attributes('aria-pressed') === 'true')!.get('span').text()).toBe(scope === 'courses' ? '257' : '0')
  wrapper.unmount()
 })
 test('a failed index has no misleading zero', async () => {
  const wrapper = await mountPage(makeRouter(), emptyProps({ query: '课程', failedScopes: ['courses'] }))
  expect(wrapper.findAll('[role="group"] a')[4]!.find('span').exists()).toBe(false)
  wrapper.unmount()
 })
})
