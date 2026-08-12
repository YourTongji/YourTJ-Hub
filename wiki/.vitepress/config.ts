import { defineConfig } from 'vitepress'
import { chineseSearchOptimize, pagefindPlugin } from 'vitepress-plugin-pagefind'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: 'YourTJ Wiki',
  description: '同济大学校园社区平台 YourTJ 知识库 —— 使用指南、部署运维与开发文档',
  lang: 'zh-CN',
  base: '/',
  // 内容源目录（review：缺失时 VitePress 以项目根为 srcDir，首页被构建到
  // dist/docs/index.html，根路径与导航全部 404，且 README.md 被误收为内容源）
  srcDir: 'docs',

  head: [
    ['link', { rel: 'icon', href: '/favicon.svg' }],
    ['meta', { name: 'theme-color', content: '#06b6d4' }],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:locale', content: 'zh_CN' }],
    ['meta', { name: 'og:site_name', content: 'YourTJ Wiki' }],
  ],

  // Pagefind 离线全文搜索（中文切词优化，基于浏览器 Intl.Segmenter）。
  // 注意：安卓 WebView 的 Intl.Segmenter 支持不完整可能导致中文结果减少
  // （pagefind issue #1176），落地后需在安卓端实测。
  vite: {
    plugins: [
      pagefindPlugin({
        customSearchQuery: chineseSearchOptimize,
        placeholder: '搜索 YourTJ Wiki…',
      }),
    ],
  },

  themeConfig: {
    logo: '/favicon.svg',

    nav: [
      { text: '首页', link: '/' },
      { text: '指南', link: '/guide/getting-started' },
      { text: '部署', link: '/deployment/' },
      { text: '飞书同步', link: '/feishu/' },
    ],

    sidebar: {
      '/guide/': [
        {
          text: '指南',
          items: [
            { text: '快速开始', link: '/guide/getting-started' },
            { text: '内容规范', link: '/guide/content' },
          ],
        },
      ],
      '/deployment/': [
        {
          text: '部署',
          items: [
            { text: '部署总览', link: '/deployment/' },
            { text: 'CF Pages 发布', link: '/deployment/cloudflare-pages' },
            { text: 'Waline 评论服务', link: '/deployment/waline' },
            { text: 'OAuth Center 与 Hub OIDC', link: '/deployment/oauth-center-oidc' },
          ],
        },
      ],
      '/feishu/': [
        {
          text: '飞书 CMS 辅轨',
          items: [
            { text: '同步机制', link: '/feishu/' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/YourTongji/YourTJ-Hub' },
    ],

    footer: {
      message: 'YourTJ Wiki · 同济大学校园社区平台',
      copyright: 'MIT Licensed',
    },

    // 搜索由 pagefindPlugin 接管（vitepress 会自动禁用内置搜索并挂 Pagefind UI）
  },
})
