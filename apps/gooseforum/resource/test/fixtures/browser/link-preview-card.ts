import { createApp, h } from 'vue'
import LinkPreviewCard from '../../../src/site/components/LinkPreviewCard.vue'
import { i18n } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { LinkPreview } from '@gooseforum/client'

/**
 * LinkPreviewCard 的布局夹具：卡片宽度由视口决定，所以这里不写死宽高，
 * 只负责把四种真实形状摆出来，让 link-preview-card.browser.mjs 在多个
 * 视口下量真实几何（截断行数、封面比例、贴边规则、卡片高度预算）。
 *
 * 封面/图标由测试用 page.route 注入 SVG，夹具里只放 URL：safeUrl 只放行
 * http(s)，data: 会被自己挡掉，不能用内联图片。
 */
document.documentElement.dataset.theme = new URLSearchParams(location.search).get('theme') === 'light' ? 'gf-light' : 'gf-dark'

const bilibiliDescription = '-，视频播放量 2280、弹幕量 0、点赞数 75、投硬币枚数 13、收藏人数 250、转发人数 14, 视频作者 Ning的笔记, 作者简介 一名什么都不想学的算法科研工作者，相关视频：不是AI做不到，只是你不知道技术的名字——Anime.js，PlanWeave：把项目计划变成 Agent 可执行的工作流，你的 AGENTS.md 不该手写，该训练；backpass 用 7 类会话反向更新记忆文件，如何使用 AI 做出顶级的 UI 设计，我们做了一个未来的输入法，Astra 完成的3D建筑拼接游戏，不是AI做不出来，是你不知道这个技术名字'

const cases: Array<{ id: string; preview: LinkPreview }> = [
  {
    id: 'long',
    preview: {
      requestedUrl: 'https://www.bilibili.com/video/BV1w5Yy67Eie/',
      kind: 'external',
      status: 'ready',
      url: 'https://www.bilibili.com/video/BV1w5Yy67Eie/',
      displayHost: 'www.bilibili.com',
      registrableDomain: 'bilibili.com',
      siteName: '哔哩哔哩',
      title: '不是AI做不到，是你不知道技术的名称——Chroma.js_哔哩哔哩_bilibili',
      description: bilibiliDescription,
      imageUrl: 'https://cdn.test/cover-wide.png',
      faviconUrl: 'https://cdn.test/icon.png',
    },
  },
  {
    id: 'dedup',
    preview: {
      requestedUrl: 'https://example.com',
      kind: 'external',
      status: 'ready',
      url: 'https://example.com',
      displayHost: 'example.com',
      registrableDomain: 'example.com',
      siteName: 'example.com',
      title: 'Example Domain',
    },
  },
  {
    id: 'internal',
    preview: {
      requestedUrl: '/topics/95',
      kind: 'internal',
      status: 'ready',
      url: '/topics/95',
      displayHost: 'forum.test',
      registrableDomain: 'forum.test',
      siteName: 'YourTJ',
      title: '论坛现在同时支持 KaTeX 数学公式 与 代码高亮。',
      description: bilibiliDescription,
    },
  },
  {
    id: 'tall',
    preview: {
      requestedUrl: 'https://design.test/poster',
      kind: 'external',
      status: 'ready',
      url: 'https://design.test/poster',
      displayHost: 'design.test',
      registrableDomain: 'design.test',
      siteName: 'Design Weekly',
      title: '一张竖版海报：object-cover 必须按框裁切，不能把卡片撑高',
      description: '竖图是最容易把卡片拉长的输入。',
      imageUrl: 'https://cdn.test/cover-tall.png',
    },
  },
  {
    // 校园网卡片：服务端只给 campus 标记（名字来自部署配置），配置没给名字时
    // title / description 为空，兜底文案由客户端 i18n 出。
    id: 'campus',
    preview: {
      requestedUrl: 'https://agent.tongji.edu.cn/chat',
      kind: 'external',
      status: 'ready',
      url: 'https://agent.tongji.edu.cn/chat',
      displayHost: 'agent.tongji.edu.cn',
      registrableDomain: 'tongji.edu.cn',
      siteName: 'agent.tongji.edu.cn',
      campus: true,
    },
  },
  {
    // 配置给了名字时以服务端值为准，客户端不得用兜底文案覆盖。
    id: 'campus-named',
    preview: {
      requestedUrl: 'https://1.tongji.edu.cn/',
      kind: 'external',
      status: 'ready',
      url: 'https://1.tongji.edu.cn/',
      displayHost: '1.tongji.edu.cn',
      registrableDomain: 'tongji.edu.cn',
      siteName: '1.tongji.edu.cn',
      title: '同济大学教学管理系统',
      campus: true,
    },
  },
]

// 卡片自带文案（校园网卡片的兜底标题/描述由客户端出），而夹具是独立挂载的
// 小应用：必须像 content-enhancements/link-preview.ts 那样装上共享 i18n，
// 并把语言固定成 zh，避免断言随浏览器语言漂移。
;(i18n.global.locale as unknown as { value: string }).value = 'zh'

createApp({
  render: () => h('div', { class: 'px-3 py-4' }, [
    h('div', { class: 'gf-prose gf-prose-post mx-auto max-w-[934px]' },
      cases.map(item => h('div', { id: `case-${item.id}` }, [h(LinkPreviewCard, { preview: item.preview })]))),
  ]),
}).use(i18n).mount('#app')
