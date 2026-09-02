// 管理端「注册与安全」设置页（AdminSettingsPage.vue）批量粘贴导入的纯解析逻辑
// （Blueprint R5）：把粘贴文本拆成条目并与既有列表合并去重，预览后由页面并入表单。
// 纯函数实现，不依赖 Vue / DOM，便于单测覆盖。

/** 单次导入的最大条目数：解析到该上限后仍有多余条目则截断（文案 10,000 与此保持一致）。 */
export const BULK_IMPORT_LIMIT = 10_000

/** 弹层预览区最多渲染的新增 chips 数，超出部分只显示计数提示。 */
export const BULK_IMPORT_PREVIEW_LIMIT = 200

/** 条目分隔符（用户名/名单类）：换行与空白、半角/全角逗号、顿号、分号。 */
const TOKEN_SEPARATOR_RE = /[\s,;，；、]+/

/** 条目分隔符（敏感词类）：仅换行与逗号/顿号/分号——保留条目内部普通空格，
 *  使 "credit card cash out" 这类多词短语不被拆成多个独立敏感词。 */
const PHRASE_SEPARATOR_RE = /[\n\r,;，；、]+/

export interface BulkImportOptions {
  /** 敏感词列表用：为 true 时不把条目内部的普通空格当作分隔符（多词短语逐条保留）。 */
  preserveSpaces?: boolean
}

export interface BulkImportPreview {
  /** 按输入顺序去重后的新增条目：trim 后的原串，保留首个大小写形态（与现有 chips 显示一致）。 */
  added: string[]
  /** 与既有集合（existing 或本批先出现的条目）重复而被跳过的条数；不含截断丢弃的部分。 */
  skipped: number
  /** 输入条目超过 BULK_IMPORT_LIMIT，只解析了前 BULK_IMPORT_LIMIT 条。 */
  truncated: boolean
}

/**
 * 解析批量粘贴文本。
 * @param text 粘贴的原始文本。默认条目可用换行/空白/逗号/顿号/分号分隔；
 *        options.preserveSpaces 为 true（敏感词列表）时保留条目内部空格，
 *        仅按换行/逗号/顿号/分号分隔。
 * @param existing 当前列表，视为已 trim 的字符串集合；去重对大小写不敏感。
 */
export function parseImportText(text: string, existing: string[], options: BulkImportOptions = {}): BulkImportPreview {
  const separator = options.preserveSpaces ? PHRASE_SEPARATOR_RE : TOKEN_SEPARATOR_RE
  const seen = new Set(existing.map(item => item.toLowerCase()))
  const added: string[] = []
  let parsed = 0
  let skipped = 0
  let truncated = false
  for (const segment of text.split(separator)) {
    const item = segment.trim()
    if (!item) continue
    if (parsed >= BULK_IMPORT_LIMIT) {
      truncated = true
      break
    }
    parsed++
    const key = item.toLowerCase()
    if (seen.has(key)) {
      skipped++
      continue
    }
    seen.add(key)
    added.push(item)
  }
  return { added, skipped, truncated }
}
