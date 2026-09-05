// @vitest-environment happy-dom
import { describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
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

const TOPIC_ID = 160

function makePost(overrides: Partial<PostPayload> & { id: number; postNo: number }): PostPayload {
  return {
    topicId: TOPIC_ID,
    content: '瞬间正文内容',
    renderedContent: '<p><img src="/uploads/img1.webp" /></p><p>瞬间正文文字描述</p>',
    processStatus: 0,
    isHidden: false,
    isAuthorDeleted: false,
    isModeratorRemoved: false,
    canModerate: false,
    author: { id: 4, username: 'walkerkiller', avatarUrl: '' },
    createdAt: '2026-09-04 01:19:00',
    isOwnPost: true,
    updatedAt: '2026-09-04 01:19:00',
    revisionCount: 0,
    likeCount: 0,
    isLiked: false,
    isBookmarked: false,
    ...overrides,
  }
}

const viewer: ViewerPayload = {
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

describe('瞬间（短文 contentType: 2）在 PostStream 中的架构完整性', () => {
  const sampleImages = ['/uploads/img1.webp', '/uploads/img2.webp']

  test('瞬间类型完整保留 aside 侧边栏（内容概览、活跃参与者）及吸底悬浮条，并嵌入置顶轮播视窗', async () => {
    const post1 = makePost({ id: 201, postNo: 1 })
    const initialStream: PostWindowPayload = {
      posts: [post1],
      hasBefore: false,
      hasAfter: false,
    }

    const wrapper = mount(PostStream, {
      props: {
        topicId: TOPIC_ID,
        topicTitle: '测试瞬间',
        contentType: 2, // 瞬间
        topicImages: sampleImages,
        initialPostStream: initialStream,
        viewer,
        canPost: true,
        topicActions: {
          likeCount: 5,
          isLiked: false,
          isBookmarked: false,
          isWatched: false,
          processStatus: 0,
          authorDeleted: false,
          moderatorRemoved: false,
          isOwnTopic: true,
          canModerateTopic: false,
          createdAt: '2026-09-04 01:19:00',
          updatedAt: '2026-09-04 01:19:00',
          replyCount: 0,
          viewCount: 12,
          maxPostNo: 1,
          participants: [post1.author],
          author: post1.author,
          description: '测试瞬间',
        },
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

    await flushPromises()

    // 1. 验证 aside 右侧栏存在（长文沿革核心要素），且包含内容概览信息
    const aside = wrapper.find('aside')
    expect(aside.exists()).toBe(true)
    expect(aside.text()).toContain(i18n.global.t('topic.overview'))
    expect(aside.text()).toContain(i18n.global.t('topic.activeParticipants'))

    // 2. 验证首楼操作工具栏保留（作者姓名、#1、编辑按钮、分享按钮）
    expect(wrapper.text()).toContain('#1')
    expect(wrapper.find(`button[title="${i18n.global.t('common.edit')}"]`).exists()).toBe(true)
    expect(wrapper.find(`button[title="${i18n.global.t('topic.share')}"]`).exists()).toBe(true)

    // 3. 验证首楼成功嵌入 TopicImageGallery 轮播窗口，并显示 1/2 徽章
    const gallery = wrapper.findComponent({ name: 'TopicImageGallery' })
    expect(gallery.exists()).toBe(true)
    expect(gallery.text()).toContain('1/2')

    // 4. 验证正文中与轮播重复的 <img> 已被剔除，保留正文文字
    const prose = wrapper.find('.gf-prose-thought')
    expect(prose.exists()).toBe(true)
    expect(prose.text()).toContain('瞬间正文文字描述')
    expect(prose.find('img[src="/uploads/img1.webp"]').exists()).toBe(false)

    // 5. 验证瞬间首楼存在醒目的“回复”主按钮与瞬间 Badge
    expect(wrapper.text()).toContain(i18n.global.t('publish.contentTypes.thought'))
    const replyButtons = wrapper.findAll('button').filter((b) => b.text().includes(i18n.global.t('topic.reply')))
    expect(replyButtons.length).toBeGreaterThan(0)
  })

  test('长文类型（contentType: 0 讨论）100% 保持图文穿插，不渲染置顶轮播图窗且保留正文中所有图片', async () => {
    const post1 = makePost({
      id: 202,
      postNo: 1,
      renderedContent: '<p>第一段文字</p><p><img src="/uploads/art1.png" alt="配图1" /></p><p>第二段分析</p><p><img src="/uploads/art2.png" alt="配图2" /></p>',
    })
    const initialStream: PostWindowPayload = {
      posts: [post1],
      hasBefore: false,
      hasAfter: false,
    }

    const wrapper = mount(PostStream, {
      props: {
        topicId: TOPIC_ID,
        topicTitle: '长文测试讨论',
        contentType: 0, // 长文讨论
        topicImages: ['/uploads/art1.png', '/uploads/art2.png'],
        initialPostStream: initialStream,
        viewer,
        canPost: true,
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

    await flushPromises()

    // 1. 验证长文不渲染置顶轮播窗口
    const gallery = wrapper.findComponent({ name: 'TopicImageGallery' })
    expect(gallery.exists()).toBe(false)

    // 2. 验证正文内原汁原味保留全部穿插图片
    const prose = wrapper.find('.gf-prose-post')
    expect(prose.exists()).toBe(true)
    expect(prose.find('img[src="/uploads/art1.png"]').exists()).toBe(true)
    expect(prose.find('img[src="/uploads/art2.png"]').exists()).toBe(true)
    expect(prose.text()).toContain('第一段文字')
    expect(prose.text()).toContain('第二段分析')
  })

  test('文章类型（contentType: 3）首楼清晰呈现文章专属 Badge', async () => {
    const post1 = makePost({
      id: 303,
      postNo: 1,
      renderedContent: '<p>深度长文正文内容</p>',
    })
    const initialStream: PostWindowPayload = {
      posts: [post1],
      hasBefore: false,
      hasAfter: false,
    }

    const wrapper = mount(PostStream, {
      props: {
        topicId: TOPIC_ID,
        topicTitle: '深度技术文章',
        contentType: 3, // 文章
        initialPostStream: initialStream,
        viewer,
        canPost: false,
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

    await flushPromises()

    // 验证文章首楼带有文章专属徽章
    expect(wrapper.text()).toContain(i18n.global.t('publish.contentTypes.article'))
  })

  test('瞬间（contentType: 2）点击首楼编辑按钮打开快捷弹层编辑器，回显标题正文与分类', async () => {
    const { quickPublishOpen, quickPublishEditPayload, closeQuickPublish } = await import('../src/site/composables/useQuickPublish').then(m => m.useQuickPublish())
    closeQuickPublish()

    const post1 = makePost({
      id: 501,
      postNo: 1,
      content: '瞬间正文测试',
      renderedContent: '<p>瞬间正文测试</p>',
      isOwnPost: true,
    })
    const initialStream: PostWindowPayload = {
      posts: [post1],
      hasBefore: false,
      hasAfter: false,
    }

    const wrapper = mount(PostStream, {
      props: {
        topicId: 500,
        topicTitle: '测试瞬间标题',
        contentType: 2, // 瞬间
        topicImages: ['/img/photo.png'],
        categories: [{ id: 10, name: '日常' }],
        initialPostStream: initialStream,
        viewer: { ...viewer, id: 1 },
        canPost: true,
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

    await flushPromises()

    // 找到首楼编辑按钮（PencilLine）
    const editBtn = wrapper.find('button[title="' + i18n.global.t('common.edit') + '"]')
    expect(editBtn.exists()).toBe(true)

    await editBtn.trigger('click')
    await flushPromises()

    // 验证弹层被打开，且装载了话题数据
    expect(quickPublishOpen.value).toBe(true)
    expect(quickPublishEditPayload.value).not.toBeNull()
    expect(quickPublishEditPayload.value?.topicId).toBe(500)
    expect(quickPublishEditPayload.value?.title).toBe('测试瞬间标题')
    expect(quickPublishEditPayload.value?.content).toBe('瞬间正文测试')
    expect(quickPublishEditPayload.value?.contentType).toBe(2)
    expect(quickPublishEditPayload.value?.images).toEqual(['/img/photo.png'])
    expect(quickPublishEditPayload.value?.categoryIds).toEqual([10])

    closeQuickPublish()
  })

  test('首楼顶部右侧精简收纳，不重复堆叠社媒互动按钮与冗余正文徽章', async () => {
    const post1 = makePost({
      id: 601,
      postNo: 1,
      content: '首楼正文内容',
      renderedContent: '<p>首楼正文内容</p>',
    })
    const post2 = makePost({
      id: 602,
      postNo: 2,
      content: '回复楼层内容',
      renderedContent: '<p>回复楼层内容</p>',
    })
    const initialStream: PostWindowPayload = {
      posts: [post1, post2],
      hasBefore: false,
      hasAfter: false,
    }

    const wrapper = mount(PostStream, {
      props: {
        topicId: 600,
        topicTitle: '移动端布局优化测试',
        contentType: 2, // 瞬间
        initialPostStream: initialStream,
        viewer: { ...viewer, id: 1 },
        canPost: true,
        topicActions: {
          likeCount: 5,
          isLiked: false,
          isBookmarked: false,
          isWatched: false,
          processStatus: 0,
          authorDeleted: false,
          moderatorRemoved: false,
          isOwnTopic: true,
          canModerateTopic: false,
          createdAt: '2026-09-04T00:00:00Z',
          updatedAt: '2026-09-04T00:00:00Z',
          replyCount: 1,
          viewCount: 10,
          maxPostNo: 2,
          participants: [],
          author: viewer as any,
          description: '',
        },
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

    await flushPromises()

    const firstPostArticle = wrapper.get('article[data-post-no="1"]')
    const secondPostArticle = wrapper.get('article[data-post-no="2"]')

    // 首楼顶部标题行只展示瞬间徽章与编辑，不冗余显示“正文”徽章
    const firstHeader = firstPostArticle.get('.mb-1\\.5')
    expect(firstHeader.text()).toContain(i18n.global.t('publish.contentTypes.thought'))
    expect(firstHeader.text()).not.toContain(i18n.global.t('topic.originalPost'))

    // 首楼顶部右侧在移动端隐藏社媒图标（hidden sm:inline-flex），仅在桌面端快捷保留
    const firstLikeBtn = firstHeader.find('button[title="' + i18n.global.t('topic.like') + '"]')
    expect(firstLikeBtn.exists()).toBe(true)
    expect(firstLikeBtn.classes()).toContain('hidden')
    expect(firstLikeBtn.classes()).toContain('sm:inline-flex')

    // 回复楼层（非首楼）顶部无条件展示快捷互动按钮
    const secondHeader = secondPostArticle.get('.mb-1\\.5')
    const secondLikeBtn = secondHeader.find('button[title="' + i18n.global.t('topic.like') + '"]')
    expect(secondLikeBtn.exists()).toBe(true)
    expect(secondLikeBtn.classes()).not.toContain('hidden')

    // 桌面端操作栏（.border-t .hidden.sm:flex）：完整平铺展开，不收纳
    const desktopBar = firstPostArticle.get('.border-t .hidden.sm\\:flex')
    expect(desktopBar.text()).toContain(i18n.global.t('topic.reply'))
    expect(desktopBar.text()).toContain('5') // likeCount
    expect(desktopBar.text()).toContain(i18n.global.t('topic.bookmark'))
    expect(desktopBar.text()).toContain(i18n.global.t('topic.watch'))
    expect(desktopBar.find('button[title="' + i18n.global.t('topic.share') + '"]').exists()).toBe(true)

    // 移动端操作栏（.border-t .flex.sm:hidden）：单行流线型布局 + 更多 Popover
    const mobileBar = firstPostArticle.get('.border-t .flex.sm\\:hidden')
    expect(mobileBar.text()).toContain(i18n.global.t('topic.reply'))
    expect(mobileBar.text()).toContain('5') // likeCount
    expect(mobileBar.find('button[title="' + i18n.global.t('topic.share') + '"]').exists()).toBe(true)
    expect(mobileBar.find('button[title="' + i18n.global.t('topic.more') + '"]').exists()).toBe(true)
  })

  test('“问题”、“文章”和“瞬间”贴一楼尾部回复/写回答按钮具有醒目的 Solid Primary 样式类名且不显示计数', async () => {
    const post1 = makePost({ id: 701, postNo: 1, content: '文章/问题/瞬间内容' })
    const initialStream: PostWindowPayload = {
      posts: [post1],
      hasBefore: false,
      hasAfter: false,
    }

    // 1. 测试文章类型（contentType: 3）
    const wrapperArticle = mount(PostStream, {
      props: {
        topicId: 701,
        topicTitle: '文章测试',
        contentType: 3, // 文章
        initialPostStream: initialStream,
        viewer,
        canPost: true,
        topicActions: {
          likeCount: 0,
          isLiked: false,
          isBookmarked: false,
          isWatched: false,
          processStatus: 0,
          authorDeleted: false,
          moderatorRemoved: false,
          isOwnTopic: true,
          canModerateTopic: false,
          createdAt: '2026-09-04T00:00:00Z',
          updatedAt: '2026-09-04T00:00:00Z',
          replyCount: 3,
          viewCount: 10,
          maxPostNo: 1,
          participants: [],
          author: viewer as any,
          description: '',
        },
      },
      global: {
        plugins: [i18n],
        directives: { 'code-copy': () => {}, 'code-highlight': () => {}, 'math-render': () => {} },
      },
    })
    await flushPromises()

    const articleReplyBtn = wrapperArticle.find('article[data-post-no="1"] .border-t button')
    expect(articleReplyBtn.text()).toContain(i18n.global.t('topic.reply'))
    expect(articleReplyBtn.text()).not.toContain('3') // 按钮上不要计数
    expect(articleReplyBtn.classes()).toContain('bg-primary')
    expect(articleReplyBtn.classes()).toContain('text-primary-content')
    wrapperArticle.unmount()

    // 2. 测试瞬间类型（contentType: 2）
    const wrapperThought = mount(PostStream, {
      props: {
        topicId: 702,
        topicTitle: '瞬间测试',
        contentType: 2, // 瞬间
        initialPostStream: initialStream,
        viewer,
        canPost: true,
        topicActions: {
          likeCount: 0,
          isLiked: false,
          isBookmarked: false,
          isWatched: false,
          processStatus: 0,
          authorDeleted: false,
          moderatorRemoved: false,
          isOwnTopic: true,
          canModerateTopic: false,
          createdAt: '2026-09-04T00:00:00Z',
          updatedAt: '2026-09-04T00:00:00Z',
          replyCount: 5,
          viewCount: 10,
          maxPostNo: 1,
          participants: [],
          author: viewer as any,
          description: '',
        },
      },
      global: {
        plugins: [i18n],
        directives: { 'code-copy': () => {}, 'code-highlight': () => {}, 'math-render': () => {} },
      },
    })
    await flushPromises()

    const thoughtReplyBtn = wrapperThought.find('article[data-post-no="1"] .border-t button')
    expect(thoughtReplyBtn.text()).toContain(i18n.global.t('topic.reply'))
    expect(thoughtReplyBtn.text()).not.toContain('5') // 按钮上不要计数
    expect(thoughtReplyBtn.classes()).toContain('bg-primary')
    expect(thoughtReplyBtn.classes()).toContain('text-primary-content')
    wrapperThought.unmount()

    // 3. 测试问题类型（contentType: 1）
    const wrapperQuestion = mount(PostStream, {
      props: {
        topicId: 703,
        topicTitle: '问题测试',
        contentType: 1, // 问题
        initialPostStream: initialStream,
        viewer,
        canPost: true,
        topicActions: {
          likeCount: 0,
          isLiked: false,
          isBookmarked: false,
          isWatched: false,
          processStatus: 0,
          authorDeleted: false,
          moderatorRemoved: false,
          isOwnTopic: true,
          canModerateTopic: false,
          createdAt: '2026-09-04T00:00:00Z',
          updatedAt: '2026-09-04T00:00:00Z',
          replyCount: 2,
          viewCount: 10,
          maxPostNo: 1,
          participants: [],
          author: viewer as any,
          description: '',
        },
      },
      global: {
        plugins: [i18n],
        directives: { 'code-copy': () => {}, 'code-highlight': () => {}, 'math-render': () => {} },
      },
    })
    await flushPromises()

    const questionReplyBtn = wrapperQuestion.find('article[data-post-no="1"] .border-t button')
    expect(questionReplyBtn.text()).toContain(i18n.global.t('topic.writeAnswer'))
    expect(questionReplyBtn.text()).not.toContain('2') // 按钮上不要计数
    expect(questionReplyBtn.classes()).toContain('bg-primary')
    expect(questionReplyBtn.classes()).toContain('text-primary-content')
    wrapperQuestion.unmount()
  })

  test('移动视图下短文类型首楼有图片时，图窗置顶，头像栏在次，正文在后', async () => {
    const post1 = makePost({ id: 801, postNo: 1, content: '短文测试正文' })
    const initialStream: PostWindowPayload = {
      posts: [post1],
      hasBefore: false,
      hasAfter: false,
    }

    const wrapper = mount(PostStream, {
      props: {
        topicId: 801,
        topicTitle: '多图短文',
        contentType: 2, // 瞬间
        topicImages: ['/img/photo1.jpg', '/img/photo2.jpg'],
        initialPostStream: initialStream,
        viewer,
        canPost: true,
        topicActions: {
          likeCount: 1,
          isLiked: false,
          isBookmarked: false,
          isWatched: false,
          processStatus: 0,
          authorDeleted: false,
          moderatorRemoved: false,
          isOwnTopic: true,
          canModerateTopic: false,
          createdAt: '2026-09-04T00:00:00Z',
          updatedAt: '2026-09-04T00:00:00Z',
          replyCount: 0,
          viewCount: 1,
          maxPostNo: 1,
          participants: [],
          author: viewer as any,
          description: '',
        },
      },
      global: {
        plugins: [i18n],
        directives: { 'code-copy': () => {}, 'code-highlight': () => {}, 'math-render': () => {} },
      },
    })
    await flushPromises()

    const article = wrapper.find('article[data-post-no="1"]')
    // 包含移动端 flex flex-col 样式类
    expect(article.classes()).toContain('flex')
    expect(article.classes()).toContain('flex-col')

    // 包含移动端置顶图窗容器与作者信息栏
    const mobileGallery = article.find('.block.sm\\:hidden')
    expect(mobileGallery.exists()).toBe(true)

    const mobileAuthorBar = article.find('.flex.sm\\:hidden.items-center.justify-between')
    expect(mobileAuthorBar.exists()).toBe(true)
    expect(mobileAuthorBar.text()).toContain(viewer.username)

    wrapper.unmount()
  })
})
