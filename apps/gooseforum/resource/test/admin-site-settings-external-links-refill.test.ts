import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'

// 回归（issue #1244）：站点信息保存会净化 externalLinks，保存成功后必须用落库
// 值回填该字段，否则管理员看到的是被净化的输入而不是真实存储值；回填只写
// externalLinks，不整体覆盖表单（避免回填覆盖未保存的其它编辑）。

const pageSrc = readFileSync(resolve(__dirname, '../src/admin/pages/AdminSettingsPage.vue'), 'utf8')

function extractFunction(name: string, async = false): string {
  const signature = async ? `async function ${name}(` : `function ${name}(`
  const start = pageSrc.indexOf(signature)
  expect(start, `function ${name} should exist`).toBeGreaterThanOrEqual(0)
  const end = pageSrc.indexOf('\n}', start)
  expect(end, `function ${name} should terminate`).toBeGreaterThan(start)
  return pageSrc.slice(start, end)
}

function siteInfoBranch(): string {
  const saveFn = extractFunction('save', true)
  const start = saveFn.indexOf("kind === 'site-info') {")
  expect(start, "site-info 保存分支 should exist").toBeGreaterThanOrEqual(0)
  const end = saveFn.indexOf("kind === 'mail'", start)
  expect(end, "site-info 分支应以 mail 分支结束").toBeGreaterThan(start)
  return saveFn.slice(start, end)
}

describe('站点设置 externalLinks 保存后回填(回归: 保存净化值未回显)', () => {
  test('site-info 分支保存后调用既有加载函数回填 externalLinks', () => {
    const branch = siteInfoBranch()
    expect(branch).toContain('await saveSiteSettings(normalizeSite(siteForm))')
    expect(branch).toContain('siteForm.externalLinks = normalizeSite(await getSiteSettings()).externalLinks')
  })

  test('回填只写 externalLinks，不整体覆盖表单', () => {
    const branch = siteInfoBranch()
    expect(branch).not.toContain('Object.assign(siteForm')
    expect(branch).not.toContain('siteForm =')
  })
})
