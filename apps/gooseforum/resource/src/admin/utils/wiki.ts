// Wiki 管理工具函数（与后端 wikiservice.ValidateNamespace 的判定保持一致，
// 见 apps/gooseforum/app/service/wikiservice/path.go:47）。
// 命名空间名称：小写字母、数字、连字符，段间以单个连字符分隔，最长 64 字符。

const NAMESPACE_NAME_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/
const MAX_NAMESPACE_NAME_LENGTH = 64

/**
 * 校验命名空间名称（前端镜像后端规则：先 trim + 小写归一，再匹配正则与长度上限）。
 * 返回 false 时调用方应提示「仅限小写字母、数字和连字符」。
 */
export function isValidNamespaceName(name: string): boolean {
  const normalized = name.trim().toLowerCase()
  return normalized.length <= MAX_NAMESPACE_NAME_LENGTH && NAMESPACE_NAME_RE.test(normalized)
}

// 与后端 wikiservice.ValidatePath 的判定保持一致，
// 见 apps/gooseforum/app/service/wikiservice/path.go:53。
// 页面路径：namespace/slug[/slug...]，每段小写字母/数字/连字符，段间以单个连字符分隔，
// 至少 2 段、每段 ≤64、总长 ≤255。

const SLUG_SEGMENT_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/
const MAX_SLUG_LENGTH = 64
const MAX_PATH_LENGTH = 255

/**
 * 校验 wiki 页面路径（前端镜像后端规则：先 trim + 小写归一，
 * 再按 '/' 分段检查每段格式与长度）。返回 false 时调用方应提示
 * 「路径仅限小写字母、数字和连字符，格式：namespace/slug」。
 */
export function isValidWikiPath(path: string): boolean {
  const normalized = path.trim().toLowerCase()
  if (normalized === '' || normalized.length > MAX_PATH_LENGTH) {
    return false
  }
  const segments = normalized.split('/')
  if (segments.length < 2) {
    return false // 至少 namespace + 一个 slug 段
  }
  return segments.every(
    (segment) =>
      segment !== '' && segment !== '.' && segment !== '..' &&
      segment.length <= MAX_SLUG_LENGTH && SLUG_SEGMENT_RE.test(segment),
  )
}
