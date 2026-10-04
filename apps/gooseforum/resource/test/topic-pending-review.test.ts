// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import type { PostPayload, PostWindowPayload } from '@gooseforum/client'
import TopicPage from '../src/site/pages/TopicPage.vue'
import PostStream from '../src/site/components/PostStream.vue'
import PostReplyRow from '../src/site/components/PostReplyRow.vue'
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

function pendingPost(isOwnPost: boolean, content: string): PostPayload {
  return {
    id: 9751, topicId: 975, postNo: 2, content, renderedContent: content ? `<p>${content}</p>` : '',
    processStatus: 2, isHidden: true, isAuthorDeleted: false, isModeratorRemoved: false, canModerate: false,
    author: { id: 1, username: 'author', avatarUrl: '' }, createdAt: '2026-10-02 10:00:00', updatedAt: '2026-10-02 10:00:00',
    isOwnPost, revisionCount: 0, likeCount: 0, isLiked: false, isBookmarked: false,
  } as PostPayload
}

function mountStream(post: PostPayload) {
  const first = { ...pendingPost(true, ''), id: 9750, postNo: 1, processStatus: 0, isHidden: false, content: '首楼', renderedContent: '<p>首楼</p>' }
  return mount(PostStream, {
    props: {
      topicId: 975, topicTitle: '待审话题', contentType: 0,
      initialPostStream: { posts: post.postNo === 1 ? [post] : [first, post], hasBefore: false, hasAfter: false } as PostWindowPayload,
      viewer: { id: 1, username: 'author', email: '', avatarUrl: '', isAuthenticated: true, canAccessAdmin: false, isModerator: false, requiresEmailVerification: false, adminPermissions: [] }, canPost: false,
    },
    global: { plugins: [i18n], directives: { 'code-copy': () => {}, 'code-highlight': () => {}, 'math-render': () => {}, 'content-enhancements': () => {} } },
  })
}

function mountReplyRow(post: PostPayload) {
  return mount(PostReplyRow, {
    props: { post, authenticated: true, canPost: false, actionState: { likeCount: 0, isLiked: false, isBookmarked: false, actingLike: false, actingBookmark: false } },
    global: { plugins: [i18n], stubs: { UserAvatar: true }, directives: { 'code-copy': () => {}, 'code-highlight': () => {}, 'math-render': () => {}, 'content-enhancements': () => {} } },
  })
}

describe('pending post content (issue #975)', () => {
  test('authors can edit pending content without enabling public interactions', async () => {
    const post = pendingPost(true, '再次编辑待审内容')
    for (const wrapper of [mountStream({ ...post, postNo: 1 }), mountStream(post), mountReplyRow(post)]) {
      await flushPromises()
      const target = wrapper.find('#post-9751')
      const content = target.exists() ? target : wrapper
      expect(content.find('[title="编辑"]').exists()).toBe(true)
      expect(content.find('[title="赞"]').exists()).toBe(false)
      wrapper.unmount()
    }
  })

  test('authors read their own pending post in full, with a review badge', async () => {
    for (const wrapper of [mountStream(pendingPost(true, '刚写完的回复')), mountReplyRow(pendingPost(true, '刚写完的回复'))]) {
      await flushPromises()
      expect(wrapper.text()).toContain('刚写完的回复')
      expect(wrapper.text()).not.toContain('该回复已被处理')
      expect(wrapper.find('[data-test="post-pending-review"]').exists()).toBe(true)
      wrapper.unmount()
    }
  })

  test('other readers still see the hidden placeholder', async () => {
    for (const wrapper of [mountStream(pendingPost(false, '')), mountReplyRow(pendingPost(false, ''))]) {
      await flushPromises()
      expect(wrapper.text()).toContain('该回复已被处理')
      wrapper.unmount()
    }
  })
})
