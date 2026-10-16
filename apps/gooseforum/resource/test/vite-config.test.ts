import { describe, expect, test } from 'vitest'
import { createServer } from 'vite'
import { resolve } from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import config from '../vite.config'

const resourceRoot = fileURLToPath(new URL('../', import.meta.url))
const twemojiPublicPath = fileURLToPath(
  new URL('../../../mobile/packages/ui_kit/assets/twemoji-public', import.meta.url),
)
const repositoryRoot = resolve(resourceRoot, '../../..')

/**
 * 页面级 CSP（app/http/middleware/securityHeaders.go buildPageCSP）为 script-src 'self'：
 * dev 模式若设置 Vite server 的 origin，`?url` 运行时资源（vditor i18n/lute/icons、katex 等）
 * 会生成为指向 dev server 的绝对 URL，对后端端口而言是跨域脚本而被 CSP 静默拦截，
 * 编辑器无法初始化（issue #453）。删除后资源为 /assets 相对路径，经后端 dev 代理同源返回，
 * 与生产构建行为一致。上游同步时勿再带入该配置。
 */
describe('vite 配置约束', () => {
  test('dev server 不得固定 origin，保证 ?url 运行时资源为同源相对路径', () => {
    const server = (config as { server?: { origin?: string } }).server
    expect(server?.origin).toBeUndefined()
  })

  /**
   * 页面级 CSP 的 script-src 'self' 与 style-src 均不放行 data:（issue #461）。
   * Vite 构建默认把小于 4096B 的 ?url 资产内联为 data: URL：vditor 四个语言包
   * （data:text/javascript）会因 CSP 拦截导致编辑器无法挂载，content-theme 与
   * hljs 主题四个小 CSS（data:text/css）静默丢失。assetsInlineLimit 必须为 0，
   * 强制所有 ?url 运行时资产落成 /assets 同源文件。dev server 恒不内联，
   * 因此本约束只在生产构建生效，上游同步或后续改动勿再放宽。
   */
  test('构建不得内联运行时资产为 data: URL，assetsInlineLimit 必须为 0', () => {
    const build = (config as { build?: { assetsInlineLimit?: number | boolean } }).build
    expect(build?.assetsInlineLimit).toBe(0)
  })

  test('Vite 只允许访问前端工程与 Twemoji 静态资源目录', () => {
    const server = (config as { server?: { fs?: { allow?: string[] } } }).server
    expect(server?.fs?.allow).toEqual([resourceRoot, twemojiPublicPath])
    expect(server?.fs?.allow).not.toContain(repositoryRoot)
  })

  test('开发服务器提供前端与 Twemoji 资源并拒绝仓库外部文件', async () => {
    const vite = await createServer({
      ...config,
      configFile: false,
      root: resourceRoot,
      plugins: [],
      optimizeDeps: { noDiscovery: true },
      server: {
        ...config.server,
        host: '127.0.0.1',
        port: 0,
        strictPort: false,
        open: false,
      },
      logLevel: 'silent',
    })

    try {
      await vite.listen()
      const address = vite.httpServer?.address()
      if (!address || typeof address === 'string') throw new Error('Vite did not open a TCP listener')
      const origin = `http://127.0.0.1:${address.port}`

      const appModulePath = resolve(resourceRoot, 'src/site/main.ts')
      const appModule = await fetch(`${origin}/assets/@fs${appModulePath}?raw`)
      expect(appModule.status).toBe(200)
      expect(await appModule.text()).toContain('createApp')

      const sharedClientPath = resolve(resourceRoot, 'packages/client/src/index.ts')
      const sharedClient = await fetch(`${origin}/assets/@fs${sharedClientPath}?raw`)
      expect(sharedClient.status).toBe(200)

      const twemoji = await fetch(`${origin}/assets/twemoji/72x72/1f600.png`)
      expect(twemoji.status).toBe(200)
      expect(twemoji.headers.get('content-type')).toContain('image/png')

      const privateFile = resolve(repositoryRoot, 'docs/README.md')
      const denied = await fetch(`${origin}/assets/@fs${privateFile}`)
      expect(denied.status).toBe(403)
    } finally {
      await vite.close()
    }
  }, 30_000)
})
