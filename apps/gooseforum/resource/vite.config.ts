import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  base: '/assets/',
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: 'static/dist',
    assetsDir: 'assets',
    manifest: true,
    emptyOutDir: true,
    // 页面级 CSP（securityHeaders.go buildPageCSP）的 script-src/style-src 均不放行 data:。
    // 默认 4096B 内联阈值会把 ?url 导入的小资产（vditor 语言包、content-theme/hljs 主题 CSS）
    // 内联成 data:text/javascript|css 而被 CSP 拦截：语言包报错令编辑器无法挂载，主题 CSS 静默丢失
    // （issue #461）。必须保持 0，让所有 ?url 运行时资产落成 /assets 同源文件；dev server 恒不内联，
    // 此约束只在生产构建生效。
    assetsInlineLimit: 0,
    rollupOptions: {
      input: {
        site: 'src/site/main.ts',
        admin: 'src/admin/main.ts',
      },
    },
  },
  server: {
    port: 3010,
    // 不固定 dev server 的来源地址：页面 CSP 为 script-src 'self'，固定后 `?url` 运行时资源
    // 生成绝对 URL 会被判为跨域脚本静默拦截（issue #453）。保持相对路径经后端 /assets 同源代理。
  },
})
