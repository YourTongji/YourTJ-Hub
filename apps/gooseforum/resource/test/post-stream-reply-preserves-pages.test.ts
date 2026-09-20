// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { PostPayload, PostWindowPayload, ViewerPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import PostStream from '../src/site/components/PostStream.vue'

// PostComposer 是重型异步组件（vditor/prosemirror），与本回归无关，stub 掉并透出
// v-model 与 submit 事件，用于驱动 submitPost → revealCreatedPost 链路。
vi.mock('@/site/components/PostComposer.vue', () => ({
  __esModule: true,
  default: {
    name: 'PostComposerStub',
    props: ['open', 'mode', 'modelValue'],
    emits: ['update:modelValue', 'submit'],
    template: `<button type="button" data-test="composer-submit" @click="$emit('update:modelValue', '新回复内容'); $emit('submit')">submit</button>`,
  },
}))

const getPostWindowMock = vi.hoisted(() => vi.fn())
const createPostMock = vi.hoisted(() => vi.fn())

vi.mock('@/runtime/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../src/runtime/api')>()
  return {
    ...actual,
    getPostWindow: getPostWindowMock,
    createPost: createPostMock,
  }
})

const TOPIC_ID = 317

function makePost(postNo: number, id = 1000 + postNo): PostPayload {
  return {
    topicId: TOPIC_ID,
    content: `第 ${postNo} 楼内容`,
    renderedContent: `<p>第 ${postNo} 楼内容</p>`,
    processStatus: 0,
    isHidden: false,
    isAuthorDeleted: false,
    isModeratorRemoved: false,
    canModerate: false,
    author: { id: 7, username: 'author', avatarUrl: '' },
    createdAt: '2026-09-10 10:00:00',
    isOwnPost: true,
    updatedAt: '2026-09-10 10:00:00',
    revisionCount: 0,
    likeCount: 0,
    isLiked: false,
    isBookmarked: false,
    id,
    postNo,
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

function page(from: number, to: number): PostPayload[] {
  const posts: PostPayload[] = []
  for (let postNo = from; postNo <= to; postNo += 1) posts.push(makePost(postNo))
  return posts
}

async function mountStream(initialPostStream: PostWindowPayload): Promise<VueWrapper> {
  const wrapper = mount(PostStream, {
    props: {
      topicId: TOPIC_ID,
      topicTitle: '分页保留回归',
      contentType: 0,
      initialPostStream,
      viewer,
      canPost: true,
    },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}
function loadedPostNos(wrapper: VueWrapper) {
  return wrapper.findAll('[data-post-no]').map((node) => Number(node.attributes('data-post-no')))
}

async function submitReply(wrapper: VueWrapper) {
  ;(wrapper.vm as unknown as { openFloatingPostComposer(): void }).openFloatingPostComposer()
  await flushPromises()
  await wrapper.get('[data-test="composer-submit"]').trigger('click')
  await flushPromises()
  // revealCreatedPost 内部等待 requestAnimationFrame 稳帧定位新楼层（happy-dom 的
  // rAF 基于真实定时器），等待其完成后再断言。
  await new Promise((resolve) => setTimeout(resolve, 400))
  await flushPromises()
}

// issue #717 回归：往下分页累积多页后提交回复，已加载楼层必须保留（append 合并），
// 不能因锚点窗口整体 replace 而清空分页进度、跳回顶部。
describe('PostStream 回复后保留已加载分页（issue #717 回归）', () => {
  let wrapper: VueWrapper | null = null
  let rectSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    getPostWindowMock.mockReset()
    createPostMock.mockReset()
    // happy-dom 的 getBoundingClientRect 恒为 0，会让新楼层被判为“不可见”而进入
    // 滚动钉住循环；统一给一个稳定且“基本可见”的矩形，走正常 idle 收尾路径。
    rectSpy = vi
      .spyOn(HTMLElement.prototype, 'getBoundingClientRect')
      .mockReturnValue({ top: 200, bottom: 400, left: 0, right: 300, width: 300, height: 200 } as DOMRect)
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    rectSpy.mockRestore()
    vi.restoreAllMocks()
  })

  test('相邻新楼层走 append：分页加载两页后回复，已加载 1..40 楼全部保留', async () => {
    wrapper = await mountStream({
      posts: page(1, 20),
      hasBefore: false,
      hasAfter: true,
      maxPostNo: 41,
      total: 41,
    })
    expect(loadedPostNos(wrapper)).toHaveLength(20)

    // 向下加载一页：21..40 楼 append 进来（总楼层 41，仍有一页 hasAfter）。
    getPostWindowMock.mockImplementationOnce(async (params: { afterPostNo?: number }) => {
      expect(params.afterPostNo).toBe(20)
      return {
        posts: page(21, 40),
        replyTargets: [],
        hasBefore: true,
        hasAfter: true,
        maxPostNo: 41,
        total: 41,
      }
    })
    const loadMore = wrapper
      .findAll('button')
      .find((button) => button.text() === i18n.global.t('topic.loadMoreReplies'))
    expect(loadMore).toBeTruthy()
    await loadMore!.trigger('click')
    await flushPromises()
    expect(loadedPostNos(wrapper)).toHaveLength(40)
    expect(loadedPostNos(wrapper)).toContain(1)
    expect(loadedPostNos(wrapper)).toContain(40)

    // 提交回复：新楼层 #42 与已加载尾部（#40）相邻 → 锚点窗口 23..42 走 append。
    createPostMock.mockResolvedValueOnce({ id: 9001 })
    getPostWindowMock.mockImplementationOnce(async () => ({
      posts: [...page(23, 41), makePost(42, 9001)],
      replyTargets: [],
      hasBefore: true,
      hasAfter: false,
      maxPostNo: 42,
      total: 42,
    }))
    await submitReply(wrapper)

    expect(createPostMock).toHaveBeenCalledTimes(1)
    expect(getPostWindowMock).toHaveBeenLastCalledWith(
      expect.objectContaining({ topicId: TOPIC_ID, anchorPostId: 9001, limit: 20 }),
    )
    const postNos = loadedPostNos(wrapper)
    expect(postNos).toHaveLength(42)
    expect(postNos).toContain(1)
    expect(postNos).toContain(40)
    expect(postNos).toContain(42)
    expect(new Set(postNos).size).toBe(42)
    // 分页边界同步：全部楼层已加载，不再出现“加载更多”。
    expect(wrapper.text()).toContain(i18n.global.t('topic.allRepliesShown'))
  })

  test('远端锚点仍走 replace：深链/超长话题场景的导航语义保持不变', async () => {
    wrapper = await mountStream({
      posts: page(1, 20),
      hasBefore: false,
      hasAfter: true,
      maxPostNo: 100,
      total: 100,
    })
    expect(loadedPostNos(wrapper)).toHaveLength(20)

    // 话题共 100 楼、未向下翻页（尾部 #20），新楼层 #101 距尾部超过一窗 → replace。
    createPostMock.mockResolvedValueOnce({ id: 9002 })
    getPostWindowMock.mockImplementationOnce(async () => ({
      posts: [...page(82, 100), makePost(101, 9002)],
      replyTargets: [],
      hasBefore: true,
      hasAfter: false,
      maxPostNo: 101,
      total: 101,
    }))
    await submitReply(wrapper)

    const postNos = loadedPostNos(wrapper)
    expect(postNos).toHaveLength(20)
    expect(postNos).not.toContain(1)
    expect(postNos).toContain(82)
    expect(postNos).toContain(101)
  })
})
