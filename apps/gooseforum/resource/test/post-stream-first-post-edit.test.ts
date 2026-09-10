// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { PostPayload, PostWindowPayload, ViewerPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import PostStream from '../src/site/components/PostStream.vue'

// PostComposer 是重型异步组件（vditor/prosemirror），与本回归无关，stub 掉以隔离。
// 断言依赖其 props（mode 表示就地编辑态），故 stub 需透出 props。
vi.mock('@/site/components/PostComposer.vue', () => ({
  // Vue 的 async loader 依赖 __esModule 决定取 mod.default 作为组件；
  // 缺省时会把整个 mock 命名空间当组件，test-utils 对 vitest 代理访问未声明属性会抛错。
  __esModule: true,
  default: {
    name: 'PostComposerStub',
    props: ['open', 'mode'],
    template: '<div data-test="post-composer" />',
  },
}))

const TOPIC_ID = 42
const ORIGINAL_HREF = `http://localhost:5234/p/topic/${TOPIC_ID}`

let locationMock: { href: string; pathname: string; hash: string }
const originalLocationDescriptor = Object.getOwnPropertyDescriptor(window, 'location')!

function makePost(overrides: Partial<PostPayload> & { id: number; postNo: number }): PostPayload {
  return {
    topicId: TOPIC_ID,
    content: '正文内容',
    renderedContent: '<p>正文内容</p>',
    processStatus: 0,
    isHidden: false,
    isAuthorDeleted: false,
    isModeratorRemoved: false,
    canModerate: false,
    author: { id: 7, username: 'author', avatarUrl: '' },
    createdAt: '2026-09-02 10:00:00',
    isOwnPost: true,
    updatedAt: '2026-09-02 10:00:00',
    revisionCount: 0,
    likeCount: 0,
    isLiked: false,
    isBookmarked: false,
    ...overrides,
  }
}

const viewer: ViewerPayload = {
  id: 7,
  username: 'author',
  email: 'author@example.com',
  avatarUrl: '',
  isAuthenticated: true,
  canAccessAdmin: false,
  isModerator: false,
  requiresEmailVerification: false,
  adminPermissions: [],
}

const editButtonTitle = i18n.global.t('common.edit')

function mountStream() {
  const postStream: PostWindowPayload = {
    posts: [
      makePost({ id: 101, postNo: 1 }),
      makePost({ id: 102, postNo: 2 }),
    ],
    replyTargets: [],
    hasBefore: false,
    hasAfter: false,
    total: 2,
    maxPostNo: 2,
  }
  return mount(PostStream, {
    props: {
      topicId: TOPIC_ID,
      topicTitle: '测试话题',
      initialPostStream: postStream,
      viewer,
      canPost: true,
      syncUrl: false,
    },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
}

// 首楼点击编辑应导航到 /publish?id=<topicId>（发布页编辑态），回复楼层不跳转、走就地编辑。
// 回归：PR #217（b8cb1ff0）误删首楼分支，导致首楼只能就地改正文、无法改标题/分类（issue #379）。
describe('PostStream 首楼编辑跳发布页（issue #379 回归）', () => {
  beforeEach(() => {
    locationMock = { href: ORIGINAL_HREF, pathname: `/p/topic/${TOPIC_ID}`, hash: '' }
    Object.defineProperty(window, 'location', { configurable: true, value: locationMock })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', originalLocationDescriptor)
    vi.restoreAllMocks()
  })

  test('首楼（postNo === 1）点击编辑笔跳转 /publish?id=<topicId>', async () => {
    const wrapper = mountStream()
    await flushPromises()

    const firstPostArticle = wrapper.get('article[data-post-no="1"]')
    await firstPostArticle.get(`button[title="${editButtonTitle}"]`).trigger('click')
    await flushPromises()

    expect(locationMock.href).toBe(`/publish?id=${TOPIC_ID}`)
    wrapper.unmount()
  })

  test('回复楼层（postNo > 1）点击编辑笔不跳转，进入就地编辑态', async () => {
    const wrapper = mountStream()
    await flushPromises()

    const replyArticle = wrapper.get('article[data-post-no="2"]')
    await replyArticle.get(`button[title="${editButtonTitle}"]`).trigger('click')
    await flushPromises()

    expect(locationMock.href).toBe(ORIGINAL_HREF)
    const composer = wrapper.findComponent({ name: 'PostComposerStub' })
    expect(composer.exists()).toBe(true)
    expect(composer.props('open')).toBe(true)
    expect(composer.props('mode')).toBe('edit')
    wrapper.unmount()
  })
})