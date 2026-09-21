// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { PostPayload, PostWindowPayload, ViewerPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import PostStream from '../src/site/components/PostStream.vue'

// 同 post-stream-reply-preserves-pages.test.ts：stub 重型 PostComposer，透出 submit 事件
// 驱动 submitPost → revealCreatedPost 链路。
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

const TOPIC_ID = 318
const INNER_HEIGHT = 900
const SCROLL_HEIGHT = 2000
const MAX_SCROLL_Y = SCROLL_HEIGHT - INNER_HEIGHT
// 新回复的绝对位置：滚到底（scrollY = MAX_SCROLL_Y）时距视口顶 302px，
// 距 160px 舒适线还差 142px 且下方没有更多内容可滚——舒适位置不可达。
const CREATED_POST_ABSOLUTE_TOP = MAX_SCROLL_Y + 302
const CREATED_POST_ID = 9001

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

async function raf(times: number) {
  for (let i = 0; i < times; i += 1) {
    await new Promise((resolve) => window.requestAnimationFrame(resolve))
  }
}

// issue #745 回归：回复定位的钉扎目标超出文档可达范围（末楼下方内容不足）时，
// settle 循环必须立即收敛（进入 idle），不能每帧 scrollBy 撞上浏览器底部钳制、
// 依赖 2600ms 硬超时退出，并在超时窗口内持续夺回用户的滚动位置。
describe('PostStream 回复后钉扎目标不可达时收敛（issue #745 回归）', () => {
  let wrapper: VueWrapper | null = null
  let rectSpy: ReturnType<typeof vi.spyOn>
  let fakeScrollY = 0
  let scrollToMock: ReturnType<typeof vi.fn>
  let scrollByMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    getPostWindowMock.mockReset()
    createPostMock.mockReset()
    fakeScrollY = 0
    scrollToMock = vi.fn((arg: ScrollToOptions | number) => {
      const top = typeof arg === 'number' ? arg : (arg?.top ?? 0)
      fakeScrollY = Math.min(Math.max(0, top), MAX_SCROLL_Y)
    })
    scrollByMock = vi.fn((arg: ScrollToOptions | number) => {
      const top = typeof arg === 'number' ? arg : (arg?.top ?? 0)
      fakeScrollY = Math.min(Math.max(0, fakeScrollY + top), MAX_SCROLL_Y)
    })
    // 模拟浏览器滚动约束：scrollTo/scrollBy 只能在 [0, MAX_SCROLL_Y] 内生效。
    Object.defineProperty(window, 'scrollY', { configurable: true, get: () => fakeScrollY })
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: INNER_HEIGHT })
    Object.defineProperty(document.documentElement, 'scrollHeight', {
      configurable: true,
      get: () => SCROLL_HEIGHT,
    })
    window.scrollTo = scrollToMock as unknown as typeof window.scrollTo
    window.scrollBy = scrollByMock as unknown as typeof window.scrollBy
    // happy-dom 无真实布局：所有元素共享一个随 scrollY 联动的矩形，
    // 且高度足够大（bottom 超出 innerHeight - 120），让新回复被判定为“非基本可见”，
    // 走 scrollPostIntoComfortView + resumePostRailSyncWhenSettled 钉扎路径。
    rectSpy = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(() => {
      const top = CREATED_POST_ABSOLUTE_TOP - fakeScrollY
      return {
        top,
        bottom: top + 600,
        left: 0,
        right: 300,
        width: 300,
        height: 600,
        x: 0,
        y: top,
        toJSON: () => ({}),
      } as DOMRect
    })
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    rectSpy.mockRestore()
    vi.restoreAllMocks()
  })

  test('文档底部钳制下 settle 循环收敛，不夺回用户上滑位置', async () => {
    wrapper = mount(PostStream, {
      props: {
        topicId: TOPIC_ID,
        topicTitle: '钉扎不可达回归',
        contentType: 0,
        initialPostStream: {
          posts: page(1, 20),
          hasBefore: false,
          hasAfter: false,
          maxPostNo: 21,
          total: 21,
        } satisfies PostWindowPayload,
        viewer,
        canPost: true,
      },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    ;(wrapper.vm as unknown as { openFloatingPostComposer(): void }).openFloatingPostComposer()
    await flushPromises()
    createPostMock.mockResolvedValueOnce({ id: CREATED_POST_ID })
    getPostWindowMock.mockImplementationOnce(async () => ({
      posts: [...page(1, 20), makePost(21, CREATED_POST_ID)],
      replyTargets: [],
      hasBefore: false,
      hasAfter: false,
      maxPostNo: 21,
      total: 21,
    }))
    await wrapper.get('[data-test="composer-submit"]').trigger('click')
    await flushPromises()

    // 场景有效性守卫：新回复确实走了舒适定位（element 非基本可见 → scrollTo 被调用）。
    await raf(4)
    expect(scrollToMock).toHaveBeenCalled()

    // 给足收敛帧数：修复后钉扎目标不可达应立即视为收敛，8 帧稳定后进入 idle。
    await raf(20)
    await flushPromises()

    // 探针：用户上滑 300px，再跑若干帧，钉扎不得夺回位置。
    scrollByMock.mockClear()
    fakeScrollY -= 300
    await raf(6)
    await flushPromises()
    expect(scrollByMock).not.toHaveBeenCalled()
    expect(fakeScrollY).toBe(MAX_SCROLL_Y - 300)
  })
})
