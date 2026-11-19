// @vitest-environment happy-dom
import { expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CategoryPage from '../src/site/pages/CategoryPage.vue'
import { i18n, setLocale } from '../src/runtime/i18n'
import type { CategoryPageProps, LayoutPayload, TopicPayload } from '@gooseforum/client'

vi.mock('@/runtime/router', () => ({ fetchPage: vi.fn() }))
import { fetchPage } from '@/runtime/router'

const topic = (id: number, pinWeight = 0): TopicPayload => ({
  id, pinWeight, title: `话题 ${id}`, url: `/p/${id}`, description: '',
  author: { id: 1, username: 'reader', avatarUrl: '' }, participants: [], categories: [],
  replyCount: 0, viewCount: 0, processStatus: 0, activityText: '', lastUpdateTime: '',
})

test('category latest exposes expandable pins and deduplicates subsequent pages', async () => {
  await setLocale('zh')
  const props: CategoryPageProps = {
    category: { id: 7, name: '校园生活', description: '', icon: '', color: '', url: '/c/7' },
    sort: 'latest', tabs: [], topics: [topic(1, 2), topic(2, 1), topic(3)],
    pagination: { page: 1, nextPage: 2, hasNext: true, nextUrl: '/c/7?page=2' },
  }
  const view = mount(CategoryPage, {
    props: { layout: {} as LayoutPayload, props, pageUrl: '/c/7' },
    global: { plugins: [i18n], stubs: { TopicRow: { props: ['topic'], template: '<p class="test-row">{{ topic.title }}</p>' } } },
  })
  try {
    expect(view.find('section[aria-label="置顶话题"]').exists()).toBe(true)
    expect(view.findAll('.test-row').map(row => row.text())).toEqual(['话题 3'])
    expect(view.find('a[href="/p/2"]').exists()).toBe(false)
    await view.get('button[aria-expanded]').trigger('click')
    expect(view.get('a[href="/p/2"]').text()).toContain('话题 2')
    vi.mocked(fetchPage).mockResolvedValue({ props: {
      ...props, topics: [topic(2, 1), topic(4)],
      pagination: { page: 2, nextPage: 0, hasNext: false, nextUrl: '' },
    } } as never)
    await view.get('footer button').trigger('click')
    await flushPromises()
    expect(view.findAll('a[href="/p/2"]')).toHaveLength(1)
    expect(view.findAll('.test-row').map(row => row.text())).toEqual(['话题 3', '话题 4'])

    await view.setProps({ props: { ...props, sort: 'new' }, pageUrl: '/c/7/new' })
    expect(view.find('button[aria-expanded]').exists()).toBe(false)
    expect(view.findAll('.test-row')).toHaveLength(3)
    await view.setProps({ props: { ...props, topics: [topic(3)] }, pageUrl: '/c/7?refresh=1' })
    expect(view.find('section[aria-label="置顶话题"]').exists()).toBe(false)
  } finally { view.unmount() }
})
