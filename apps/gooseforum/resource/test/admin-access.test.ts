import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'
import { AdminPermission, canVisitAdminPath, configureAdminAccess } from '../src/admin/runtime/access'

// 从源码提取侧边栏菜单 URL 与 settingsPages 路由 path,静态验证权限白名单
// 完整覆盖(防止「菜单/路由新增但白名单漏加 → 点击被守卫 fallback 到 /admin」类回归,
// 即 AI 课程总结跳转 bug 的根因)。

const adminSrc = resolve(__dirname, '../src/admin')

function sidebarMenuUrls(): string[] {
  const src = readFileSync(resolve(adminSrc, 'components/layout/AppSidebar.vue'), 'utf8')
  const urls = [...src.matchAll(/url: '(\/admin[^']*)'/g)].map((m) => m[1])
  return [...new Set(urls)]
}

function settingsRouterPaths(): string[] {
  const src = readFileSync(resolve(adminSrc, 'runtime/router.ts'), 'utf8')
  const paths = [...src.matchAll(/^  '(\/admin\/settings\/[^']*)':/gm)].map((m) => m[1])
  return [...new Set(paths)]
}

function whitelistPaths(): string[] {
  const src = readFileSync(resolve(adminSrc, 'runtime/access.ts'), 'utf8')
  const block = src.match(/const adminPathPermissions[^=]*= \{([\s\S]*?)\n\}/)?.[1] ?? ''
  return [...block.matchAll(/'(\/admin[^']*)':/g)].map((m) => m[1])
}

describe('admin 权限白名单与菜单/路由一致性(回归: AI 课程总结跳转 bug)', () => {
  test('侧边栏每个菜单项 URL 都在权限白名单中', () => {
    const urls = sidebarMenuUrls()
    const whitelist = whitelistPaths()
    expect(urls.length).toBeGreaterThan(0)
    for (const url of urls) {
      expect(whitelist, `菜单项 ${url} 应出现在 access.ts adminPathPermissions 白名单`).toContain(url)
    }
  })

  test('settingsPages 每个路由 path 都在权限白名单中', () => {
    const paths = settingsRouterPaths()
    const whitelist = whitelistPaths()
    expect(paths.length).toBeGreaterThan(0)
    for (const p of paths) {
      expect(whitelist, `settings 路由 ${p} 应出现在 access.ts adminPathPermissions 白名单`).toContain(p)
    }
  })

  test('授权 SiteManager 后可访问 AI 课程总结与限流设置页(修复目标)', () => {
    configureAdminAccess([AdminPermission.SiteManager])
    expect(canVisitAdminPath('/admin/settings/ai-summary')).toBe(true)
    expect(canVisitAdminPath('/admin/settings/rate-limit')).toBe(true)
  })

  test('未授权时 AI 课程总结路径被拒绝(守卫生效)', () => {
    configureAdminAccess([])
    expect(canVisitAdminPath('/admin/settings/ai-summary')).toBe(false)
    configureAdminAccess([AdminPermission.SiteManager])
    expect(canVisitAdminPath('/admin/settings/ai-summary')).toBe(true)
  })

  test('站点统计 /admin 不受影响: 需 Admin 权限', () => {
    configureAdminAccess([])
    expect(canVisitAdminPath('/admin')).toBe(false)
    configureAdminAccess([AdminPermission.Admin])
    expect(canVisitAdminPath('/admin')).toBe(true)
  })
})
