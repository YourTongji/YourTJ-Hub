import { describe, expect, test } from 'vitest'
import config from '../vite.config'

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
})
