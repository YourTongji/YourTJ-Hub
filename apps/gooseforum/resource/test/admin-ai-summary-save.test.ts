import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'

// 从源码提取 ai-summary 表单的 normalize/save 逻辑，静态断言保存链路不回丢
// apiKey（回归：normalizeAiSummary 曾硬编码 apiKey: ''，用户填的 key 在保存时
// 被清空 → 后端按「留空=保留已存密文」处理 → 永远无法配置 → 线上 401）。

const pageSrc = readFileSync(resolve(__dirname, '../src/admin/pages/AdminSettingsPage.vue'), 'utf8')

function extractFunction(name: string, async = false): string {
  const signature = async ? `async function ${name}(` : `function ${name}(`
  const start = pageSrc.indexOf(signature)
  expect(start, `function ${name} should exist`).toBeGreaterThanOrEqual(0)
  const end = pageSrc.indexOf('\n}', start)
  expect(end, `function ${name} should terminate`).toBeGreaterThan(start)
  return pageSrc.slice(start, end)
}

describe('admin AI summary apiKey 保存链路(回归: 填了 key 保存后仍显示未配置)', () => {
  test('normalizeAiSummary 保留表单 apiKey（不硬编码清空）', () => {
    const fn = extractFunction('normalizeAiSummary')
    expect(fn).toContain("apiKey: settings.apiKey ?? ''")
    expect(fn).not.toContain("apiKey: ''")
  })

  test('ai-summary 保存成功后同步 apiKeyConfigured 徽标状态', () => {
    const saveFn = extractFunction('save', true)
    // save() 内 ai-summary 分支：保存后立即置位（依赖重新进入页面会误导用户）。
    const branch = saveFn.slice(saveFn.indexOf("kind === 'ai-summary'"))
    expect(branch).toContain('await saveAiSummarySettings(aiSummaryPayload())')
    expect(branch).toContain('aiSummaryForm.apiKeyConfigured = true')
  })

  test('apiKeyConfigured 为只读回显字段，不随保存请求回传（issue #324 安全模式）', () => {
    const payloadFn = extractFunction('aiSummaryPayload')
    expect(payloadFn).toContain('apiKeyConfigured: _configured')
    expect(payloadFn).toContain('...payload')
  })
})
