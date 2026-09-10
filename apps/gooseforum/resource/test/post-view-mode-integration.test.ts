// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { PostPayload, PostWindowPayload, ReplyTargetPayload, ViewerPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'

vi.mock('@/site/components/PostComposer.vue', () => ({
  __esModule: true,
  default: {
    name: 'PostComposerStub',
    props: ['open', 'mode'],
    template: '<div data-test="post-composer" />',
  },
}))

const TOPIC_ID = 300

function makePost(overrides: Partial<PostPayload> & { id: number; postNo: number; replyToPostId?: number }): PostPayload {
  return {
    topicId: TOPIC_ID,
    content: `楼层正文 ${overrides.postNo}`,
    renderedContent: `<p>楼层正文 ${overrides.postNo}</p>`,
    processStatus: 0,
    isHidden: false,
    isAuthorDeleted: false,
    isModeratorRemoved: false,
    canModerate: false,
    author: { id: 4, username: 'walker', avatarUrl: '' },
    createdAt: '2026-09-06 10:00:00',
    isOwnPost: false,
    updatedAt: '2026-09-06 10:00:00',
    revisionCount: 0,
    likeCount: 0,
    isLiked: false,
    isBookmarked: false,
    ...overrides,
  }
}

const viewer: ViewerPayload = {
  id: 4,
  username: 'walker',
  email: 'walker@example.com',
  avatarUrl: '',
  isAuthenticated: true,
  canAccessAdmin: false,
  isModerator: false,
  requiresEmailVerification: false,
  adminPermissions: [],
}

function streamOf(posts: PostPayload[], replyTargets: ReplyTargetPayload[] = []): PostWindowPayload {
  return { posts, replyTargets, hasBefore: false, hasAfter: false }
}

// post-view-mode 使用模块级共享状态（与 home-feed-mode 同模式），
// 每次挂载前 resetModules 让组件拿到全新模块并重新读取 localStorage。
async function mountPostStream(
  contentType: 1 | 3,
  posts: PostPayload[],
  replyTargets: ReplyTargetPayload[] = [],
): Promise<VueWrapper> {
  vi.resetModules()
  const { default: PostStream } = await import('../src/site/components/PostStream.vue')
  const wrapper = mount(PostStream, {
    props: {
      topicId: TOPIC_ID,
      topicTitle: '测试话题',
      contentType,
      initialPostStream: streamOf(posts, replyTargets),
      viewer,
      canPost: true,
    },
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
  await flushPromises()
  return wrapper
}

function findCapsule(wrapper: VueWrapper) {
  const group = wrapper
    .findAll('[role="group"]')
    .find((candidate) => candidate.text().includes(i18n.global.t('topic.viewFlat')))
  expect(group, '视图切换胶囊应渲染').toBeTruthy()
  return group!
}

function capsuleButtons(wrapper: VueWrapper) {
  return findCapsule(wrapper).findAll('button')
}

describe('楼层流树状/扁平双视图（组件级集成）', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  test('提问话题默认树状：链内回复真实父子嵌套，主流层只渲染根节点，内容零丢失', async () => {
    const posts = [
      makePost({ id: 1, postNo: 1 }),
      makePost({ id: 2, postNo: 2, replyToPostId: 1, isAnswer: true }),
      makePost({ id: 3, postNo: 3, replyToPostId: 2 }),
      makePost({ id: 4, postNo: 4, replyToPostId: 3 }),
    ]
    const wrapper = await mountPostStream(1, posts)

    const buttons = capsuleButtons(wrapper)
    expect(buttons[0]!.attributes('aria-pressed')).toBe('false')
    expect(buttons[1]!.attributes('aria-pressed')).toBe('true')

    // 主流层 = 森林根节点（#1 提问 + #2 回答）；#3/#4 嵌套在 #2 卡片内的树状块
    expect(wrapper.find('article[data-post-no="1"]').exists()).toBe(true)
    expect(wrapper.find('article[data-post-no="2"]').exists()).toBe(true)
    expect(wrapper.find('article[data-post-no="3"]').exists()).toBe(false)
    expect(wrapper.find('article[data-post-no="4"]').exists()).toBe(false)

    const row3 = wrapper.find('[data-post-no="3"]')
    expect(row3.exists()).toBe(true)
    expect(wrapper.find('article[data-post-no="2"]').element.contains(row3.element)).toBe(true)
    // 内容零丢失：最深层的 #4 仍然可见
    expect(wrapper.find('[data-post-no="4"]').exists()).toBe(true)

    // #520：树状视图父子关系由嵌套缩进表达，不再渲染全文引用条
    expect(wrapper.find('article[data-post-no="2"]').find('aside').exists()).toBe(false)
    expect(row3.find('aside').exists()).toBe(false)
    expect(wrapper.find('[data-test="reply-context-hint"]').exists()).toBe(false)

    // #515：树状行头部时间戳在窄屏隐藏，避免深层缩进行头部撑破视口宽度
    expect(row3.find('time').classes()).toContain('hidden')
  })

  test('文章话题默认扁平：链内回复平铺为独立顶层楼层', async () => {
    const posts = [
      makePost({ id: 1, postNo: 1 }),
      makePost({ id: 2, postNo: 2, replyToPostId: 1 }),
      makePost({ id: 3, postNo: 3, replyToPostId: 2 }),
    ]
    const wrapper = await mountPostStream(3, posts)

    const buttons = capsuleButtons(wrapper)
    expect(buttons[0]!.attributes('aria-pressed')).toBe('true')
    expect(buttons[1]!.attributes('aria-pressed')).toBe('false')
    expect(wrapper.find('article[data-post-no="3"]').exists()).toBe(true)
  })

  test('胶囊切换即时生效并按内容类型持久化，类型间互不影响', async () => {
    const posts = [
      makePost({ id: 1, postNo: 1 }),
      makePost({ id: 2, postNo: 2, replyToPostId: 1 }),
      makePost({ id: 3, postNo: 3, replyToPostId: 2 }),
    ]

    // #519：Q&A（默认树状）切到扁平 = 全量平铺，链内子回复不再堆叠进链根卡片
    const qa = await mountPostStream(1, posts)
    const qaButtons = capsuleButtons(qa)
    await qaButtons[0]!.trigger('click')
    await flushPromises()
    expect(qaButtons[0]!.attributes('aria-pressed')).toBe('true')
    expect(qa.find('article[data-post-no="3"]').exists()).toBe(true)
    const qaRow3 = qa.find('[data-post-no="3"]')
    expect(qaRow3.exists()).toBe(true)
    expect(qa.find('article[data-post-no="2"]').element.contains(qaRow3.element)).toBe(false)
    // 与普通话题一致：回复非首楼时由引用条承接上下文
    expect(qaRow3.find('aside').exists()).toBe(true)
    expect(JSON.parse(window.localStorage.getItem('goose:post-view-mode')!)).toEqual({ '1': 'flat' })

    // 文章话题不受 Q&A 选择影响，仍走默认扁平
    const article = await mountPostStream(3, posts)
    const articleButtons = capsuleButtons(article)
    expect(articleButtons[0]!.attributes('aria-pressed')).toBe('true')
    expect(JSON.parse(window.localStorage.getItem('goose:post-view-mode')!)).toEqual({ '1': 'flat' })

    // 文章切到树状并持久化，两种类型的选择互不覆盖
    await articleButtons[1]!.trigger('click')
    await flushPromises()
    expect(articleButtons[1]!.attributes('aria-pressed')).toBe('true')
    expect(JSON.parse(window.localStorage.getItem('goose:post-view-mode')!)).toEqual({ '1': 'flat', '3': 'tree' })
    expect(article.find('article[data-post-no="3"]').exists()).toBe(false)
    expect(article.find('[data-post-no="3"]').exists()).toBe(true)

    // 刷新场景：重新挂载后 Q&A 仍记住「扁平」选择（#519：保持全量平铺）
    const qaReloaded = await mountPostStream(1, posts)
    const reloadedButtons = capsuleButtons(qaReloaded)
    expect(reloadedButtons[0]!.attributes('aria-pressed')).toBe('true')
    expect(qaReloaded.find('article[data-post-no="3"]').exists()).toBe(true)
    expect(qaReloaded.find('article[data-post-no="2"]').element.contains(qaReloaded.find('[data-post-no="3"]').element)).toBe(false)
  })

  test('树状折叠：折叠有子链的节点隐藏其后代，再点展开恢复', async () => {
    const posts = [
      makePost({ id: 1, postNo: 1 }),
      makePost({ id: 2, postNo: 2, replyToPostId: 1 }),
      makePost({ id: 3, postNo: 3, replyToPostId: 2 }),
      makePost({ id: 4, postNo: 4, replyToPostId: 3 }),
    ]
    const wrapper = await mountPostStream(1, posts)

    expect(wrapper.find('[data-post-no="4"]').exists()).toBe(true)
    const collapseButtonTitle = i18n.global.t('topic.collapseReply')
    const row3 = wrapper.find('[data-post-no="3"]')
    const collapseButton = row3.find(`button[title="${collapseButtonTitle}"]`)
    expect(collapseButton.exists()).toBe(true)

    await collapseButton.trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-post-no="4"]').exists()).toBe(false)
    // 被折叠节点自身仍在
    expect(wrapper.find('[data-post-no="3"]').exists()).toBe(true)

    const expandButton = wrapper
      .find('[data-post-no="3"]')
      .find(`button[title="${i18n.global.t('topic.expandReply')}"]`)
    await expandButton.trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-post-no="4"]').exists()).toBe(true)
  })

  test('树状视图目标未加载时兜底为根楼层平铺（深链零丢失）', async () => {
    // 窗口只含回复链中的一段：#5 回复未加载的 #4
    const posts = [
      makePost({ id: 5, postNo: 5, replyToPostId: 4 }),
    ]
    const wrapper = await mountPostStream(1, posts)

    // 兜底为根节点 → 渲染为主流顶层楼层，内容不丢
    expect(wrapper.find('article[data-post-no="5"]').exists()).toBe(true)
  })

  test('#520 树状孤根弱提示：目标未加载的根楼层保留「回复了 @user #N」短提示而非全文引用条', async () => {
    const posts = [
      makePost({ id: 1, postNo: 1 }),
      makePost({ id: 5, postNo: 5, replyToPostId: 4 }),
    ]
    // 深链场景：目标楼层在窗口外但仍可见 → 后端下发带作者与楼号的 replyTargets 摘要
    // （buildReplyTargetPayload available 分支；unavailable=false 经 omitempty 省略）
    const wrapper = await mountPostStream(1, posts, [
      { id: 4, postNo: 4, author: { id: 9, username: 'alice', avatarUrl: '' } },
    ])

    // 兜底孤根仍渲染为主流顶层楼层
    const orphanRoot = wrapper.find('article[data-post-no="5"]')
    expect(orphanRoot.exists()).toBe(true)
    // 全文引用条（aside/PostReplyReference）不再渲染
    expect(orphanRoot.find('aside').exists()).toBe(false)
    // 弱化短提示兜底上下文
    const hint = wrapper.find('[data-test="reply-context-hint"]')
    expect(hint.exists()).toBe(true)
    expect(hint.text()).toContain('@alice')
    expect(hint.text()).toContain('#4')
  })

  test('#520 树状孤根目标不可见时降级为「原回复不可见」短提示（不泄漏作者与楼号）', async () => {
    const posts = [
      makePost({ id: 1, postNo: 1 }),
      makePost({ id: 5, postNo: 5, replyToPostId: 4 }),
    ]
    // 生产真实形态：目标被隐藏/删除/清理时后端只下发 { id, unavailable }（作者/楼号为零值，
    // 见 app/http/controllers/forum/payload.go buildReplyTargetPayload 早退分支）
    const wrapper = await mountPostStream(1, posts, [
      { id: 4, author: { id: 0, username: '', avatarUrl: '' }, unavailable: true },
    ])

    const orphanRoot = wrapper.find('article[data-post-no="5"]')
    expect(orphanRoot.exists()).toBe(true)
    // 全文引用条不渲染，短提示兜底上下文
    expect(orphanRoot.find('aside').exists()).toBe(false)
    const hint = wrapper.find('[data-test="reply-context-hint"]')
    expect(hint.exists()).toBe(true)
    expect(hint.text()).toContain(i18n.global.t('topic.replyTargetUnavailable'))
    // 不泄漏被隐藏目标的作者与楼号
    expect(hint.text()).not.toContain('@')
    expect(hint.text()).not.toContain('#')
  })
})