#!/usr/bin/env node
/**
 * sync-feishu.mjs — 飞书 CMS 辅轨同步脚本骨架（issue #170）
 *
 * 作用：把飞书文档库中的文档同步为 wiki/docs/feishu/ 下的 markdown，
 * 作为 git markdown 主轨之外的"辅轨"内容源。
 *
 * 设计（最小可用实现）：
 *  1. 用飞书开放平台 API（tenant_access_token + 文档 API）拉取指定
 *     文档库/文件夹下的文档。
 *  2. 转成 markdown（可选用 feishu2md / docsify 等转换工具，或用飞书
 *     export API 导出 docx 再转换）。
 *  3. 写入 wiki/docs/feishu/<doc-token>.md，并生成一个索引文件
 *     wiki/docs/feishu/README.md（列出所有已同步文档）。
 *  4. 由外部 cron（GitHub Actions schedule 或服务器 crontab）触发：
 *     pnpm sync:feishu && git commit -m "chore(wiki): sync feishu docs" && git push
 *
 * 环境变量（全部可选；不配置时进入 dry-run 演示模式，只打印将同步的文档）：
 *  - FEISHU_APP_ID / FEISHU_APP_SECRET : 自建应用的凭据（必填才可真实同步）
 *  - FEISHU_FOLDER_TOKEN               : 要同步的文档库/文件夹 token
 *  - WIKI_OUTPUT_DIR                   : 输出目录（默认 wiki/docs/feishu）
 *
 * 注意：这是骨架实现，真实同步需要按你的飞书应用权限（文档读写、
 * 云文档导出）补齐 API 调用细节。生产使用前请先在小范围文件夹验证。
 */

import { mkdir, writeFile, readdir } from 'node:fs/promises'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const WIKI_ROOT = join(__dirname, '..')
const OUTPUT_DIR = process.env.WIKI_OUTPUT_DIR || join(WIKI_ROOT, 'docs', 'feishu')
const APP_ID = process.env.FEISHU_APP_ID || ''
const APP_SECRET = process.env.FEISHU_APP_SECRET || ''
const FOLDER_TOKEN = process.env.FEISHU_FOLDER_TOKEN || ''

/**
 * 获取飞书 tenant_access_token。
 * 真实实现：POST https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal
 *   body: { app_id, app_secret }
 * 返回: { tenant_access_token, expire }
 */
async function getTenantToken() {
  if (!APP_ID || !APP_SECRET) return ''
  // TODO(issue #170): 实现真实 token 获取 + 缓存（token 有效期 2h）
  return ''
}

/**
 * 列出文件夹下的文档。
 * 真实实现：GET https://open.feishu.cn/open-apis/drive/v1/files
 *   query: folder_token, page_size
 *   header: Authorization: Bearer <tenant_access_token>
 * 返回文档数组 [{ token, name, type }]
 */
async function listFolderDocs(_token) {
  // TODO(issue #170): 实现真实列表拉取
  return [
    { token: FOLDER_TOKEN || 'demo-doc-1', name: '示例文档 1', type: 'docx' },
    { token: 'demo-doc-2', name: '示例文档 2', type: 'docx' },
  ]
}

/**
 * 把单篇文档导出为 markdown。
 * 真实实现（二选一）：
 *  a) 飞书 export API（docx -> markdown 需先导出 docx 再转换）
 *  b) 社区工具 feishu2md（https://github.com/Leizhenpeng/feishu2md）
 */
async function exportDocMarkdown(_token, _name) {
  // TODO(issue #170): 实现真实导出
  return `# 示例文档\n\n> 由飞书同步（骨架占位内容）。\n`
}

async function main() {
  const real = Boolean(APP_ID && APP_SECRET && FOLDER_TOKEN)
  console.log(`[sync-feishu] mode=${real ? 'real' : 'dry-run'} output=${OUTPUT_DIR}`)

  const token = await getTenantToken()
  const docs = await listFolderDocs(token)
  console.log(`[sync-feishu] ${docs.length} doc(s) to sync`)

  for (const doc of docs) {
    const md = await exportDocMarkdown(doc.token, doc.name)
    const file = join(OUTPUT_DIR, `${doc.token}.md`)
    await mkdir(dirname(file), { recursive: true })
    await writeFile(file, md, 'utf8')
    console.log(`[sync-feishu] wrote ${file}`)
  }

  // 生成索引 README
  const files = (await readdir(OUTPUT_DIR).catch(() => []))
    .filter((f) => f.endsWith('.md') && f !== 'README.md')
  const index = [
    '# 飞书同步文档',
    '',
    '> 本目录由 `pnpm sync:feishu` 自动生成，请勿手改。',
    '',
    ...files.map((f) => `- [${f.replace(/\.md$/, '')}](./${f})`),
    '',
  ].join('\n')
  await writeFile(join(OUTPUT_DIR, 'README.md'), index, 'utf8')
  console.log(`[sync-feishu] index updated (${files.length} doc(s))`)
}

main().catch((err) => {
  console.error('[sync-feishu] failed:', err)
  process.exit(1)
})
