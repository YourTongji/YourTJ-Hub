// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import PostStream from '../src/site/components/PostStream.vue'
import { likeTopic } from '../src/runtime/api'

vi.mock('../src/runtime/api', async (importOriginal) => ({
  ...await importOriginal<typeof import('../src/runtime/api')>(),
  likeTopic: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('@/site/components/PostComposer.vue', () => ({
  __esModule: true,
  default: {
    name: 'PostComposerStub',
    props: ['open', 'mode'],
    template: '<div data-test="post-composer" />',
  },
}))

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
  vi.clearAllMocks()
})

const topicActionsProps = {
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
  updatedAt: '2026-09-24T01:00:00Z',
  replyCount: 98,
  viewCount: 345,
  maxPostNo: 99,
  participants: [],
  author: { id: 7, username: 'author', avatarUrl: '' },
  description: 'First post',
}

function makePost(overrides: Record<string, unknown>) {
  return {
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
    ...overrides,
  }
}

function mountStream(props: Record<string, unknown>) {
  return mount(PostStream, {
    attachTo: document.body,
    props: {
      topicId: 42,
      topicTitle: 'Topic title',
      viewer,
      canPost: true,
      ...props,
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
}

describe('topic detail action authority', () => {
  test('first-post actions and overview do not repeat topic actions or facts', async () => {
    wrapper = mountStream({
      initialPostStream: {
        posts: [makePost({})],
        hasBefore: false,
        hasAfter: false,
        total: 1,
        maxPostNo: 1,
      },
      topicActions: { ...topicActionsProps, replyCount: 12, maxPostNo: 1 },
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

    const overview = wrapper.get('aside dl')
    expect(overview.text()).not.toContain('12')
    expect(overview.text()).not.toContain('345')
    expect(overview.text()).toContain(i18n.global.t('topic.participants'))

    // 悬浮胶囊与首楼操作栏镜像：同一组话题级操作、同一状态来源
    const floating = document.body.querySelector('[data-test="floating-controls"]')!
    const floatingPill = floating.querySelector('.gf-floating-surface')!
    // 胶囊挂载在楼层流之外，任何宽度都应渲染（不得为空壳）
    expect(floatingPill.className).not.toContain('xl:hidden')
    for (const key of ['topic.like', 'topic.bookmark', 'topic.watch']) {
      expect(floatingPill.querySelector(`button[title="${i18n.global.t(key)}"]`), key).not.toBeNull()
    }
    const mobileReply = floatingPill.querySelector('[data-test="mobile-topic-reply"]')
    expect(mobileReply).not.toBeNull()
    expect(mobileReply?.className).not.toContain('sm:hidden')

    // 游客：举报需要登录（后端 /api/forum/report 要求会话），话题级举报入口不可见；
    // 胶囊镜像操作需登录不显示，但保留带点赞数的点赞入口（点击去登录）与登录参与讨论入口
    await wrapper.setProps({ viewer: { ...viewer, isAuthenticated: false } } as any)
    expect(firstPostActions.find(`button[title="${i18n.global.t('topic.report')}"]`).exists()).toBe(false)
    expect(desktopActions.text()).not.toContain(i18n.global.t('topic.report'))
    expect(mobileActions.find(`button[title="${i18n.global.t('topic.more')}"]`).exists()).toBe(false)
    const guestLike = floatingPill.querySelector(`button[title="${i18n.global.t('topic.like')}"]`)
    expect(guestLike).not.toBeNull()
    expect(guestLike?.textContent).toContain('8')
    for (const key of ['topic.bookmark', 'topic.watch']) {
      expect(floatingPill.querySelector(`button[title="${i18n.global.t(key)}"]`), key).toBeNull()
    }
    const guestReply = floatingPill.querySelector('[data-test="mobile-topic-reply"]')!
    expect(guestReply.textContent).toContain(i18n.global.t('topic.loginToJoinDiscussion'))
  })

  test('window without the first post keeps topic actions reachable via the floating capsule', async () => {
    wrapper = mountStream({
      initialPostStream: {
        posts: [makePost({ id: 199, postNo: 99, content: 'Reply 99', renderedContent: '<p>Reply 99</p>', likeCount: 0, createdAt: '2026-09-24T01:00:00Z', updatedAt: '2026-09-24T01:00:00Z' })],
        hasBefore: true,
        hasAfter: false,
        total: 99,
        maxPostNo: 99,
      },
      topicActions: topicActionsProps,
    })

    // 首楼不在窗口内：楼层流内没有话题操作栏，胶囊镜像仍提供话题级操作（dev 兜底行为）
    expect(wrapper.find('article [data-test="topic-actions"]').exists()).toBe(false)
    const floating = document.body.querySelector('[data-test="floating-controls"]')!
    const floatingPill = floating.querySelector('.gf-floating-surface')!
    for (const key of ['topic.like', 'topic.bookmark', 'topic.watch', 'topic.share', 'topic.report']) {
      expect(floatingPill.querySelector(`button[title="${i18n.global.t(key)}"]`), key).not.toBeNull()
    }

    await wrapper.setProps({ topicActions: { ...topicActionsProps, isOwnTopic: true } } as any)
    expect(floatingPill.querySelector(`button[title="${i18n.global.t('topic.report')}"]`)).toBeNull()
    await wrapper.setProps({ topicActions: { ...topicActionsProps, authorDeleted: true } } as any)
    expect(floatingPill.querySelector(`button[title="${i18n.global.t('topic.report')}"]`)).toBeNull()
    expect(floatingPill.querySelector(`button[title="${i18n.global.t('topic.watch')}"]`)).toBeNull()
    expect((floatingPill.querySelector(`button[title="${i18n.global.t('topic.like')}"]`) as HTMLButtonElement).disabled).toBe(true)
    await wrapper.setProps({ topicActions: topicActionsProps } as any)

    // 胶囊回复入口呼出编辑器
    await (floatingPill.querySelector('[data-test="mobile-topic-reply"]') as HTMLButtonElement).click()
    await flushPromises()
    expect(document.body.querySelector('[data-test="post-composer"]')).not.toBeNull()
  })
  test('bar and capsule share successful like state and failed-action rollback', async () => {
    wrapper = mountStream({
      initialPostStream: { posts: [makePost({})], hasBefore: false, hasAfter: false },
      topicActions: topicActionsProps,
    })
    const barLike = wrapper.get('[data-test="desktop-topic-actions"]').findAll('button')
      .find((button) => button.text() === i18n.global.t('topic.like'))!
    const capsuleLike = document.body.querySelector(`[data-test="floating-controls"] button[title="${i18n.global.t('topic.like')}"]`) as HTMLButtonElement
    await barLike.trigger('click')
    await flushPromises()
    expect(likeTopic).toHaveBeenCalledWith(42, 1)
    expect(capsuleLike.querySelector('svg')?.getAttribute('fill')).toBe('currentColor')
    vi.mocked(likeTopic).mockRejectedValueOnce(new Error('Request failed'))
    capsuleLike.click()
    await flushPromises()
    expect(likeTopic).toHaveBeenLastCalledWith(42, 2)
    expect(barLike.get('svg').attributes('fill')).toBe('currentColor')
    expect(capsuleLike.querySelector('svg')?.getAttribute('fill')).toBe('currentColor')
    expect(wrapper.get('[data-test="topic-actions"]').text()).toContain('Request failed')
  })

})
