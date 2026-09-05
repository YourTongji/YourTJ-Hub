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
