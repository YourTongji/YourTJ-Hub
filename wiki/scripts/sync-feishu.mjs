#!/usr/bin/env node
/**
 * sync-feishu.mjs — 飞书 CMS 辅轨同步脚本（issue #170 完整实现）
 *
 * 作用：把飞书文档（docx）与多维表格（bitable）同步为
 * wiki/docs/feishu/ 下的 markdown，作为 git markdown 主轨之外的"辅轨"。
 *
 * 环境变量（.env 或 shell 注入，见 .env.example）：
 *  - FEISHU_APP_ID / FEISHU_APP_SECRET  : 自建应用凭据（必填）
 *  - FEISHU_DOC_TOKENS                  : 逗号分隔的 docx 文档 token 列表
 *  - FEISHU_BITABLE_APP_TOKEN + FEISHU_BITABLE_TABLE_ID : 多维表格（可选）
 *  - FEISHU_OUTPUT_DIR                  : 输出目录（默认 wiki/docs/feishu）
 *  - FEISHU_DRY_RUN=1                   : 只打印不写文件
 *
 * 退出码：成功 0；任何失败非零（带明确错误信息）。
 */

import { mkdir, writeFile, readdir } from 'node:fs/promises'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const WIKI_ROOT = join(__dirname, '..')

const APP_ID = process.env.FEISHU_APP_ID || ''
const APP_SECRET = process.env.FEISHU_APP_SECRET || ''
const DOC_TOKENS = (process.env.FEISHU_DOC_TOKENS || '')
  .split(',').map((s) => s.trim()).filter(Boolean)
const BITABLE_APP_TOKEN = process.env.FEISHU_BITABLE_APP_TOKEN || ''
const BITABLE_TABLE_ID = process.env.FEISHU_BITABLE_TABLE_ID || ''
const OUTPUT_DIR = process.env.FEISHU_OUTPUT_DIR || join(WIKI_ROOT, 'docs', 'feishu')
const DRY_RUN = process.env.FEISHU_DRY_RUN === '1'

const FEISHU_API = 'https://open.feishu.cn/open-apis'

// ---------------------------------------------------------------- HTTP 层

async function feishuFetch(path, { method = 'GET', token, body } = {}) {
  const headers = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`
  let res
  try {
    res = await fetch(`${FEISHU_API}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    })
  } catch (err) {
    throw new Error(`飞书 API 网络错误 ${method} ${path}: ${err.message}`)
  }
  const data = await res.json().catch(() => ({}))
  if (!res.ok || (data.code !== undefined && data.code !== 0)) {
    throw new Error(`飞书 API ${method} ${path} 失败: HTTP ${res.status} code=${data.code} msg=${data.msg || ''}`)
  }
  return data
}

// ------------------------------------------------------------- token 获取

let cachedToken = null
async function getTenantToken() {
  if (cachedToken) return cachedToken
  const data = await feishuFetch('/auth/v3/tenant_access_token/internal', {
    method: 'POST',
    body: { app_id: APP_ID, app_secret: APP_SECRET },
  })
  if (!data.tenant_access_token) {
    throw new Error('获取 tenant_access_token 失败: 响应缺少 token（请检查 app_id/app_secret 与应用状态）')
  }
  cachedToken = data.tenant_access_token
  return cachedToken
}

// ---------------------------------------------------------------- docx 拉取

async function getDocumentMeta(token, docId) {
  try {
    const data = await feishuFetch(`/docx/v1/documents/${docId}`, { token })
    return (data.document || {}).title || ''
  } catch {
    return '' // 拿不到标题就退回用 token 命名
  }
}

/** 分页拉取文档全部块（按文档顺序返回）。 */
async function fetchAllBlocks(token, docId) {
  const blocks = []
  let pageToken = ''
  do {
    const qs = `?page_size=500${pageToken ? `&page_token=${encodeURIComponent(pageToken)}` : ''}`
    const data = await feishuFetch(`/docx/v1/documents/${docId}/blocks${qs}`, { token })
    blocks.push(...(data.items || []))
    pageToken = data.has_more ? data.page_token : ''
  } while (pageToken)
  return blocks
}

// ------------------------------------------------------- 块 → Markdown 转换

// 飞书 docx block_type 枚举（官方文档）
const BT = {
  PAGE: 1, TEXT: 2, HEADING1: 3, HEADING2: 4, HEADING3: 5,
  HEADING4: 6, HEADING5: 7, HEADING6: 8,
  BULLET: 9, ORDERED: 10, CODE: 11, QUOTE: 12, TODO: 13,
  BITABLE: 14, CALLOUT: 15, CHAT_CARD: 16, DIAGRAM: 17, DIVIDER: 18,
  FILE: 19, GRID: 20, GRID_COLUMN: 21, EMBED: 22, IMAGE: 23, ISV: 24,
  SHEET: 25, TABLE: 26, TABLE_CELL: 27, VIEW: 28, QUOTE_CONTAINER: 29,
  TASK: 30, DOCUMENT: 35, INFO: 41, NOTE: 42, VOTE: 44,
}

/** 提取元素数组的纯文本（text_run / mention_doc / equation 等）。 */
function elementsToText(elements) {
  if (!Array.isArray(elements)) return ''
  return elements.map((el) => {
    if (el.text_run && el.text_run.content !== undefined) return el.text_run.content
    if (el.mention_doc) return `[文档:${el.mention_doc.title || el.mention_doc.token || ''}]`
    if (el.mention_user) return `@${el.mention_user.user_name || '用户'}`
    if (el.equation) return `$${el.equation.content || ''}$`
    if (el.mention) return `@${el.mention.user_name || ''}`
    return ''
  }).join('')
}

/** code 块语言数字枚举 → 常用语言名（尽力而为，未知省略）。 */
const CODE_LANG = {
  1: '', 31: 'go', 34: 'html', 36: 'java', 37: 'javascript',
  38: 'json', 44: 'markdown', 45: 'matlab', 50: 'python', 51: 'ruby',
  52: 'scala', 53: 'shell', 54: 'sql', 55: 'swift', 56: 'typescript',
  60: 'yaml', 61: 'c', 62: 'cpp', 63: 'css',
}

/** 把文本转成安全文件名片段。 */
function sanitizeName(name) {
  const s = String(name || '')
    .replace(/[\\/:*?"<>|\u0000-\u001f]/g, '-')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
  return s || 'untitled'
}

/**
 * 渲染一棵块树。blocksByParent: Map<parentId, Block[]>（API 顺序）。
 * renderBlock 返回 markdown 行数组。
 */
function renderBlocks(blocksByParent, parentId) {
  const out = []
  const children = blocksByParent.get(parentId) || []
  for (const block of children) {
    out.push(...renderBlock(blocksByParent, block))
  }
  return out
}

function renderBlock(blocksByParent, block) {
  const type = block.block_type
  const text = (field) => elementsToText((block[field] || {}).elements)
  const childLines = () => renderBlocks(blocksByParent, block.block_id)
  const prefix = () => (block.block_type === BT.GRID_COLUMN ? '  ' : '')

  switch (type) {
    case BT.PAGE:
    case BT.GRID:
    case BT.GRID_COLUMN:
    case BT.VIEW:
    case BT.QUOTE_CONTAINER:
      return childLines().map((l) => prefix() + l)

    case BT.TEXT: {
      const t = text('text')
      return t ? [t] : []
    }

    case BT.HEADING1:
    case BT.HEADING2:
    case BT.HEADING3:
    case BT.HEADING4:
    case BT.HEADING5:
    case BT.HEADING6: {
      const level = type - BT.HEADING1 + 1
      const t = text(`heading${level}`)
      return t ? [`${'#'.repeat(level)} ${t}`] : []
    }

    case BT.BULLET: {
      const t = text('bullet')
      return t ? [`- ${t}`] : []
    }

    case BT.ORDERED: {
      const t = text('ordered')
      return t ? [`1. ${t}`] : []
    }

    case BT.TODO: {
      const done = (block.todo || {}).style?.done ? '[x]' : '[ ]'
      const t = text('todo')
      return t ? [`- ${done} ${t}`] : []
    }

    case BT.QUOTE: {
      const t = text('quote')
      return t ? [`> ${t}`] : []
    }

    case BT.CALLOUT: {
      const t = text('callout')
      return t ? [`> 💡 ${t}`] : []
    }

    case BT.NOTE:
    case BT.INFO: {
      const t = text(type === BT.NOTE ? 'note' : 'info')
      return t ? [`> ℹ️ ${t}`] : []
    }

    case BT.CODE: {
      const t = text('code')
      const lang = CODE_LANG[(block.code || {}).style?.language] || ''
      return [`\`\`\`${lang}`, ...t.split('\n'), '```']
    }

    case BT.DIVIDER:
      return ['---']

    case BT.TABLE: {
      const cells = (block.table || {}).cells || []
      // cells 是 [[{ block_id }]] 的二维数组；table_cell 块在扁平块列表里
      const byId = new Map()
      for (const [, list] of blocksByParent) {
        for (const b of list) byId.set(b.block_id, b)
      }
      const rows = cells.map((row) =>
        row.map((cell) => {
          const cb = byId.get(cell.block_id)
          const t = cb ? elementsToText((cb.table_cell || {}).text?.elements) : ''
          return t.replace(/\|/g, '\\|').replace(/\n/g, ' ')
        }),
      )
      if (rows.length === 0) return []
      const headerSep = `| ${rows[0].map(() => '---').join(' | ')} |`
      return [`| ${rows[0].join(' | ')} |`, headerSep, ...rows.slice(1).map((r) => `| ${r.join(' | ')} |`), '']
    }

    case BT.IMAGE:
      return [`<!-- 图片 token=${(block.image || {}).token || '?'}（需下载接口，未内嵌） -->`]

    case BT.FILE:
      return [`<!-- 文件 token=${(block.file || {}).token || '?'} -->`]

    case BT.SHEET:
      return [`<!-- 电子表格 token=${(block.sheet || {}).token || '?'} -->`]

    case BT.DIAGRAM:
      return [`<!-- 流程图 token=${(block.diagram || {}).token || '?'} -->`]

    case BT.EMBED:
      return [`<!-- 内嵌网页 ${(block.embed || {}).url || ''} -->`]

    case BT.BITABLE:
      return [`<!-- 内嵌多维表格 app_token=${(block.bitable || {}).app_token || '?'} -->`]

    case BT.CHAT_CARD:
    case BT.ISV:
    case BT.TASK:
    case BT.VOTE:
    default:
      return [`<!-- 未支持的块类型 ${type} -->`]
  }
}

/** docx 文档 → markdown 全文。 */
async function docxToMarkdown(token, docId) {
  const blocks = await fetchAllBlocks(token, docId)
  const blocksByParent = new Map()
  for (const b of blocks) {
    const pid = b.parent_id || ''
    if (!blocksByParent.has(pid)) blocksByParent.set(pid, [])
    blocksByParent.get(pid).push(b)
  }
  const lines = renderBlocks(blocksByParent, '')
  // 去掉首尾多余空行
  while (lines.length && lines[0] === '') lines.shift()
  while (lines.length && lines[lines.length - 1] === '') lines.pop()
  return lines.join('\n') + '\n'
}

// ------------------------------------------------------------ bitable 拉取

function fieldToString(v) {
  if (v === null || v === undefined) return ''
  if (typeof v === 'string') return v
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  if (Array.isArray(v)) return v.map(fieldToString).join(', ')
  if (typeof v === 'object') {
    if (typeof v.text === 'string') return v.text
    if (typeof v.name === 'string') return v.name
    if (typeof v.en_name === 'string') return v.en_name
    if (typeof v.link === 'string') return v.link
    if (typeof v.value === 'string') return v.value
    return JSON.stringify(v)
  }
  return String(v)
}

async function fetchBitableRecords(token) {
  const records = []
  let pageToken = ''
  do {
    const qs = `?page_size=100${pageToken ? `&page_token=${encodeURIComponent(pageToken)}` : ''}`
    const data = await feishuFetch(
      `/bitable/v1/apps/${BITABLE_APP_TOKEN}/tables/${BITABLE_TABLE_ID}/records${qs}`,
      { token },
    )
    records.push(...(data.items || []))
    pageToken = data.has_more ? data.page_token : ''
  } while (pageToken)
  return records
}

/** 多维表格记录 → markdown（表格形式）。 */
function bitableToMarkdown(records) {
  const fieldsOrder = []
  for (const rec of records) {
    for (const key of Object.keys(rec.fields || {})) {
      if (!fieldsOrder.includes(key)) fieldsOrder.push(key)
    }
  }
  if (fieldsOrder.length === 0) return '# 多维表格\n\n（无记录或字段为空）\n'
  const esc = (s) => String(s).replace(/\|/g, '\\|').replace(/\n/g, ' ')
  const header = `| ${fieldsOrder.map(esc).join(' | ')} |`
  const sep = `| ${fieldsOrder.map(() => '---').join(' | ')} |`
  const rows = records.map((rec) =>
    `| ${fieldsOrder.map((k) => esc(fieldToString(rec.fields?.[k]))).join(' | ')} |`)
  return ['# 多维表格记录', '', header, sep, ...rows, ''].join('\n')
}

// ------------------------------------------------------------------- 主流程

function buildFrontmatter({ title, source, docToken, fetchedAt }) {
  return [
    '---',
    `title: "${String(title).replace(/"/g, '\\"')}"`,
    `source: ${source}`,
    `doc_token: "${docToken}"`,
    `fetched_at: "${fetchedAt}"`,
    '---',
    '',
  ].join('\n')
}

async function writeOrPrint(file, content) {
  if (DRY_RUN) {
    const preview = content.slice(0, 500)
    console.log(`[sync-feishu][dry-run] would write ${file}\n---\n${preview}\n---\n`)
    return
  }
  await mkdir(dirname(file), { recursive: true })
  await writeFile(file, content, 'utf8')
  console.log(`[sync-feishu] wrote ${file} (${content.length} chars)`)
}

async function main() {
  const hasCreds = Boolean(APP_ID && APP_SECRET)
  const hasSource = DOC_TOKENS.length > 0 || Boolean(BITABLE_APP_TOKEN && BITABLE_TABLE_ID)

  if (!hasCreds) {
    if (DRY_RUN) {
      console.log('[sync-feishu] 演示模式：未配置 FEISHU_APP_ID/FEISHU_APP_SECRET，' +
        '仅打印流程（复制 .env.example 为 .env 并填写后启用真实同步）。')
      console.log(`[sync-feishu] 将同步 ${DOC_TOKENS.length} 个 docx + bitable(${BITABLE_APP_TOKEN ? 'on' : 'off'}) -> ${OUTPUT_DIR}`)
      return
    }
    throw new Error('缺少必填环境变量 FEISHU_APP_ID / FEISHU_APP_SECRET（见 wiki/.env.example）')
  }
  if (!hasSource) {
    throw new Error('缺少内容源：请设置 FEISHU_DOC_TOKENS（docx token 列表）或 FEISHU_BITABLE_APP_TOKEN+FEISHU_BITABLE_TABLE_ID')
  }

  console.log(`[sync-feishu] mode=${DRY_RUN ? 'dry-run' : 'real'} output=${OUTPUT_DIR}`)
  const token = await getTenantToken()
  console.log(`[sync-feishu] tenant_access_token 获取成功（${token.slice(0, 8)}…）`)

  const fetchedAt = new Date().toISOString()
  let totalDocs = 0
  let totalChars = 0
  const writtenFiles = []

  // --- docx 文档 ---
  for (const docId of DOC_TOKENS) {
    const title = (await getDocumentMeta(token, docId)) || docId
    const md = await docxToMarkdown(token, docId)
    const front = buildFrontmatter({ title, source: 'feishu-docx', docToken: docId, fetchedAt })
    const file = join(OUTPUT_DIR, `${sanitizeName(title)}.md`)
    await writeOrPrint(file, front + md)
    writtenFiles.push(file)
    totalDocs++
    totalChars += md.length
    console.log(`[sync-feishu] docx ${docId} -> ${title} (${md.length} chars)`)
  }

  // --- 多维表格 ---
  if (BITABLE_APP_TOKEN && BITABLE_TABLE_ID) {
    const records = await fetchBitableRecords(token)
    const md = bitableToMarkdown(records)
    const front = buildFrontmatter({
      title: `bitable-${BITABLE_TABLE_ID}`,
      source: 'feishu-bitable',
      docToken: `${BITABLE_APP_TOKEN}/${BITABLE_TABLE_ID}`,
      fetchedAt,
    })
    const file = join(OUTPUT_DIR, `bitable-${sanitizeName(BITABLE_TABLE_ID)}.md`)
    await writeOrPrint(file, front + md)
    writtenFiles.push(file)
    totalDocs++
    totalChars += md.length
    console.log(`[sync-feishu] bitable ${BITABLE_TABLE_ID} -> ${records.length} records (${md.length} chars)`)
  }

  // --- 索引 README ---
  const existing = (await readdir(OUTPUT_DIR).catch(() => []))
    .filter((f) => f.endsWith('.md') && f !== 'README.md' && f !== 'index.md')
  const index = [
    '# 飞书同步文档',
    '',
    '> 本目录由 `pnpm sync:feishu` 自动生成（含 frontmatter 源信息），请勿手改。',
    '',
    ...existing.map((f) => `- [${f.replace(/\.md$/, '')}](./${f})`),
    '',
  ].join('\n')
  await writeOrPrint(join(OUTPUT_DIR, 'README.md'), index)

  console.log(`[sync-feishu] 完成：${totalDocs} 篇文档，共 ${totalChars} 字符`)
}

const isMain = process.argv[1] &&
  import.meta.url === new URL(`file://${process.argv[1]}`).href
if (isMain) {
  main().catch((err) => {
    console.error(`[sync-feishu] 失败: ${err.message}`)
    process.exit(1)
  })
}

// 导出供单元测试（node --test）复用转换逻辑
export {
  elementsToText,
  renderBlocks,
  renderBlock,
  docxToMarkdown,
  bitableToMarkdown,
  sanitizeName,
  buildFrontmatter,
  BT,
}
