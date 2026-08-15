// Wiki 管理工具函数（与后端 wikiservice.ValidateNamespace 的判定保持一致，
// 见 apps/gooseforum/app/service/wikiservice/path.go:47）。
// 命名空间名称：小写字母、数字、连字符，段间以单个连字符分隔，最长 64 字符。
//
// 已知差异（有意保持更严格）：Go strings.ToLower 采用 Unicode 简单映射，会把
// U+0130(İ) 折为 ASCII 'i'，故后端接受 "İstanbul"→"istanbul"；而 JS
// String.prototype.toLowerCase 采用全量映射，将其折为 "i"+组合符点（非 ASCII），
// 本函数随之拒绝。该方向安全（本函数通过 ⇒ 后端必接受），且命名空间按产品语义
// 本就是 ASCII slug，故不模拟 Go 的简单映射。

const NAMESPACE_NAME_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/

/** 命名空间名称最大长度（与后端 maxNamespaceLen 对齐，见 wikiservice/path.go:40）。 */
export const MAX_NAMESPACE_NAME_LENGTH = 64

/**
 * 校验命名空间名称（前端镜像后端规则：先 trim + 小写归一，再匹配正则与长度上限）。
 * 返回 false 时调用方应提示「仅限小写字母、数字和连字符」。
 */
export function isValidNamespaceName(name: string): boolean {
  if (typeof name !== 'string') return false
  const normalized = name.trim().toLowerCase()
  return normalized.length <= MAX_NAMESPACE_NAME_LENGTH && NAMESPACE_NAME_RE.test(normalized)
}
