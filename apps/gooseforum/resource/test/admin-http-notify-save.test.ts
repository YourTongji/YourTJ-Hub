import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'

// 回归（PR #1055 review S1）：全局保存按服务端全量替换端点，normalizeHttpNotify 会滤掉
// 没有地址的端点。已存飞书地址切换通道后 URL 留空时，必须拦截保存而不是静默删除端点。

const pageSrc = readFileSync(resolve(__dirname, '../src/admin/pages/AdminSettingsPage.vue'), 'utf8')
const actionPageSrc = readFileSync(resolve(__dirname, '../src/site/pages/ModerationActionPage.vue'), 'utf8')

function extractFunction(src: string, name: string, async = false): string {
  const signature = async ? `async function ${name}(` : `function ${name}(`
  const start = src.indexOf(signature)
  expect(start, `function ${name} should exist`).toBeGreaterThanOrEqual(0)
  const end = src.indexOf('\n}', start)
  expect(end, `function ${name} should terminate`).toBeGreaterThan(start)
  return src.slice(start, end)
}

describe('HTTP 通知全局保存不静默删除端点', () => {
  test('save() 在构造负载前校验表单里所有端点都有地址', () => {
    const saveFn = extractFunction(pageSrc, 'save', true)
    const guard = saveFn.indexOf('validateHttpEndpointUrls()')
    expect(guard).toBeGreaterThanOrEqual(0)
    expect(guard).toBeLessThan(saveFn.indexOf('validateHttpNotify(normalizeHttpNotify(httpNotifyForm))'))
  })

  test('校验针对未过滤的表单端点，且不区分启用状态', () => {
    const fn = extractFunction(pageSrc, 'validateHttpEndpointUrls')
    expect(fn).toContain('httpNotifyForm.endpoints.find(')
    expect(fn).toContain('!endpoint.url.trim() && !endpoint.urlConfigured')
    expect(fn).not.toContain('enabled')
    expect(fn).toContain("adminText('k00d3'")
    expect(fn).toContain('expandedHttpEndpoints.add(missing.id)')
  })
})

describe('HTTP 通知端点 UUID 兼容性', () => {
  test('新增与规范化路径用共享 UUID，保存与测试继续按端点 ID 操作', () => {
    expect(extractFunction(pageSrc, 'normalizeEndpoint')).toContain('id: endpoint.id || createUuidV4()')
    expect(extractFunction(pageSrc, 'addHttpEndpoint')).toContain('const id = createUuidV4()')
    expect(extractFunction(pageSrc, 'saveHttpEndpoint', true)).toContain('endpoint.id')
    expect(extractFunction(pageSrc, 'testHttpEndpoint', true)).toContain('endpoint.id')
    expect(pageSrc).not.toContain('crypto.randomUUID()')
  })
})

describe('快捷审批确认页加载失败时仍有出口', () => {
  test('预览失败（无 view）时显示打开版主工作台', () => {
    const fallback = actionPageSrc.slice(actionPageSrc.indexOf('<div v-else class="flex flex-wrap items-center gap-2 pt-2">'))
    expect(fallback.length).toBeGreaterThan(0)
    expect(fallback).toContain('href="/moderation"')
    expect(fallback).toContain("t('moderationAction.openWorkbench')")
  })
})
