// @vitest-environment happy-dom
import { describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { PostPayload, PostWindowPayload, ViewerPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import PostStream from '../src/site/components/PostStream.vue'

vi.mock('@/site/components/PostComposer.vue', () => ({
  __esModule: true,
  default: {
    name: 'PostComposerStub',
    props: ['open', 'mode'],
    template: '<div data-test="post-composer" />',
  },
}))

const TOPIC_ID = 900

function makePost(): PostPayload {
  return {
    topicId: TOPIC_ID,
    content: '首楼正文内容',
    renderedContent: '<p>首楼正文内容</p>',
    processStatus: 0,
    isHidden: false,
    isAuthorDeleted: false,
    isModeratorRemoved: false,
    canModerate: false,
    author: { id: 4, username: 'walkerkiller', avatarUrl: '' },
    createdAt: '2026-09-04 01:19:00',
    isOwnPost: false,
    updatedAt: '2026-09-04 01:19:00',
    revisionCount: 0,
    likeCount: 0,
    isLiked: false,
    isBookmarked: false,
  }
}

const signedInViewer: ViewerPayload = {
  id: 4,
  username: 'walkerkiller',
  email: 'walkerkiller@example.com',
  avatarUrl: '',
  isAuthenticated: true,
  canAccessAdmin: false,
  isModerator: false,
  requiresEmailVerification: false,
  adminPermissions: [],
}

const guestViewer: ViewerPayload = {
  ...signedInViewer,
  id: 0,
  username: '',
  email: '',
  isAuthenticated: false,
}

function mountStream(viewer: ViewerPayload, canPost: boolean): VueWrapper {
  return mount(PostStream, {
    props: {
      topicId: TOPIC_ID,
      topicTitle: '回复门控测试',
      contentType: 0,
      initialPostStream: {
        posts: [makePost()],
        hasBefore: false,
        hasAfter: false,
      } as PostWindowPayload,
      viewer,
      canPost,
    },
    global: {
      plugins: [i18n],
      directives: {
        'code-copy': () => {},
        'code-highlight': () => {},
        'math-render': () => {},
      },
    },
  })
}

async function openFloatingComposer(wrapper: VueWrapper) {
  await flushPromises()
  ;(wrapper.vm as { openFloatingPostComposer(): void }).openFloatingPostComposer()
  await flushPromises()
}

// composerMounted 只在首次打开时置 true 且永不重置：编辑器元素出现与否
// 就是「入口是否放行」的直接观测信号。
const composerAppeared = (wrapper: VueWrapper) =>
  wrapper.find('[data-test="post-composer"]').exists()

describe('悬浮回复编辑器 canPost 门控（issue #707 深链入口）', () => {
  test('已登录但 canPost=false 时深链/浮层入口不再打开编辑器', async () => {
    const wrapper = mountStream(signedInViewer, false)
    await openFloatingComposer(wrapper)
    expect(composerAppeared(wrapper)).toBe(false)
    wrapper.unmount()
  })

  test('访客仍可打开编辑器，由 PostComposer 内置登录门引导', async () => {
    const wrapper = mountStream(guestViewer, false)
    await openFloatingComposer(wrapper)
    expect(composerAppeared(wrapper)).toBe(true)
    wrapper.unmount()
  })

  test('已登录且 canPost=true 时正常打开编辑器', async () => {
    const wrapper = mountStream(signedInViewer, true)
    await openFloatingComposer(wrapper)
    expect(composerAppeared(wrapper)).toBe(true)
    wrapper.unmount()
  })
})
