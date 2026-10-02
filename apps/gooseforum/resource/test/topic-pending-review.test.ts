// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import TopicPage from '../src/site/pages/TopicPage.vue'
import zh from '../src/locales/zh'

const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh } })

function mountTopic(processStatus: number, isOwnTopic: boolean) {
  const topic = {
    id: 975, title: '待审话题', contentType: 3, createdAt: '2026-10-02 10:00:00', processStatus,
    replyCount: 0, viewCount: 0, likeCount: 0, isLiked: false, isBookmarked: false,
    author: { id: 1, username: 'author', nickname: '作者', avatarUrl: '' }, categories: [],
  }
  return mount(TopicPage, {
    props: {
      layout: { viewer: null } as any,
      props: { topic, postStream: { posts: [] }, hotTopics: [], permissions: { canPost: false, isOwnTopic } } as any,
    },
    global: { plugins: [i18n], stubs: { UserAvatar: true, PostStream: true, Breadcrumb: true, PostStreamFloatingActions: true } },
  })
}

describe('pending topic banner (issue #975)', () => {
  test('authors learn who can see their pending topic', async () => {
    const wrapper = mountTopic(2, true)
    await flushPromises()
    expect(wrapper.find('[data-test="topic-pending-review"]').text()).toBe('这篇内容正在审核，目前只有你和审核员能看到。通过后所有人可见。')
  })

  test('reviewers see a neutral notice', async () => {
    const wrapper = mountTopic(2, false)
    await flushPromises()
    expect(wrapper.find('[data-test="topic-pending-review"]').text()).toBe('这篇内容正在等待审核，其他读者暂时看不到。')
  })

  test('published topics show no banner', async () => {
    const wrapper = mountTopic(0, true)
    await flushPromises()
    expect(wrapper.find('[data-test="topic-pending-review"]').exists()).toBe(false)
  })
})
