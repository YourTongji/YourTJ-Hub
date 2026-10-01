// @vitest-environment happy-dom
import { afterEach, describe, expect, test } from 'vitest'
import { mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import PostStream from '../src/site/components/PostStream.vue'

const viewer = {
  id: 9,
  username: 'viewer',
  isAuthenticated: true,
  canAccessAdmin: false,
  isModerator: false,
  requiresEmailVerification: false,
  adminPermissions: [],
}

let wrapper: ReturnType<typeof mount> | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
})

describe('topic detail action authority', () => {
  test('first-post actions and overview do not repeat topic actions or facts', async () => {
    wrapper = mount(PostStream, {
      attachTo: document.body,
      props: {
        topicId: 42,
        topicTitle: 'Topic title',
        initialPostStream: {
          posts: [{
            id: 101,
            topicId: 42,
            postNo: 1,
            content: 'First post',
            renderedContent: '<p>First post</p>',
            isHidden: false,
            isOwnPost: false,
            isAuthorDeleted: false,
            isModeratorRemoved: false,
            canModerate: false,
            processStatus: 0,
            createdAt: '2026-09-24T00:00:00Z',
            updatedAt: '2026-09-24T00:00:00Z',
            revisionCount: 1,
            likeCount: 1,
            isLiked: false,
            isBookmarked: false,
            author: { id: 7, username: 'author', avatarUrl: '' },
          }],
          hasBefore: false,
          hasAfter: false,
          total: 1,
          maxPostNo: 1,
        },
        viewer,
        canPost: true,
        topicActions: {
          likeCount: 8,
          isLiked: false,
          isBookmarked: false,
          isWatched: false,
          processStatus: 0,
          authorDeleted: false,
          moderatorRemoved: false,
          isOwnTopic: false,
          canModerateTopic: false,
          createdAt: '2026-09-24T00:00:00Z',
          updatedAt: '2026-09-24T00:00:00Z',
          replyCount: 12,
          viewCount: 345,
          maxPostNo: 1,
          participants: [],
          author: { id: 7, username: 'author', avatarUrl: '' },
          description: 'First post',
        },
      } as any,
      global: {
        plugins: [i18n],
        directives: {
          'code-copy': () => {},
          'code-highlight': () => {},
          'math-render': () => {},
          'content-enhancements': () => {},
        },
      },
    })

    const firstPost = wrapper.get('article[data-post-no="1"]')
    const firstPostActions = firstPost.get('[data-test="post-action-strip"]')
    for (const key of ['topic.reply', 'topic.like', 'topic.bookmark', 'topic.share', 'topic.report']) {
      expect(firstPostActions.find(`button[title="${i18n.global.t(key)}"]`).exists(), key).toBe(false)
    }

    const topicActions = firstPost.get('[data-test="topic-actions"]')
    const desktopActions = topicActions.get('[data-test="desktop-topic-actions"]')
    const mobileActions = topicActions.get('[data-test="mobile-topic-actions"]')
    expect(desktopActions.text()).toContain(i18n.global.t('topic.reply'))
    expect(desktopActions.text()).toContain(i18n.global.t('topic.report'))
    expect(desktopActions.classes()).toEqual(expect.arrayContaining(['hidden', 'sm:flex']))
    expect(mobileActions.text()).not.toContain(i18n.global.t('topic.reply'))
    expect(mobileActions.classes()).toEqual(expect.arrayContaining(['flex', 'sm:hidden']))
    const mobileReply = document.body.querySelector('[data-test="mobile-topic-reply"]')
    expect(mobileReply).not.toBeNull()
    expect(mobileReply?.className).toContain('sm:hidden')

    const overview = wrapper.get('aside dl')
    expect(overview.text()).not.toContain('12')
    expect(overview.text()).not.toContain('345')
    expect(overview.text()).toContain(i18n.global.t('topic.participants'))

    const floating = document.body.querySelector('[class*="bottom-4"]')!
    for (const key of ['topic.like', 'topic.bookmark', 'topic.watch']) {
      expect(floating.querySelector(`button[title="${i18n.global.t(key)}"]`), key).toBeNull()
    }

    await wrapper.setProps({ viewer: { ...viewer, isAuthenticated: false } } as any)
    expect(firstPostActions.find(`button[title="${i18n.global.t('topic.report')}"]`).exists()).toBe(false)
    expect(desktopActions.text()).toContain(i18n.global.t('topic.report'))
    await mobileActions.get(`button[title="${i18n.global.t('topic.more')}"]`).trigger('click')
    expect(document.body.querySelector('[role="menu"] button[role="menuitem"]')?.textContent).toContain(i18n.global.t('topic.report'))
  })
})
