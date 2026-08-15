// Wiki 管理工具函数（与后端 wikiservice.ValidateNamespace 的判定保持一致，
// 见 apps/gooseforum/app/service/wikiservice/path.go）。
// 命名空间名称：GitHub 仓库顶层目录名，与文件系统目录名兼容，支持中文等
// Unicode 字符；不做小写归一。约束：trim 后按字符计数 1..64；拒绝 "." / ".." /
// 点开头 / 空白（含全角）与首尾空格 / 控制字符 / 保留字符 / \ : * ? " < > |。

/** 命名空间名称最大长度（与后端 maxNamespaceLen 对齐，见 wikiservice/path.go）。 */
export const MAX_NAMESPACE_NAME_LENGTH = 64

/** 文件系统保留字符（Windows 保留，跨平台目录名安全）。 */
const RESERVED_PATH_CHARS = /[/\\:*?"<>|]/

/** 空白与控制字符（与后端 unicode.IsControl/IsSpace 对齐）。 */
const CONTROL_OR_SPACE = /[\p{C}\p{Z}]/u

/**
 * 校验单个路径段（命名空间或 slug）。
 * 与后端 validSegment 对齐：非空、非 "." / ".."、非点开头、
 * 长度 ≤64（按码点）、无空白/控制字符/保留字符。
 */
function isValidSegment(segment: string): boolean {
  if (segment === '' || segment === '.' || segment === '..' || segment.startsWith('.')) {
    return false
  }
  if ([...segment].length > MAX_NAMESPACE_NAME_LENGTH) {
    return false
  }
  return !CONTROL_OR_SPACE.test(segment) && !RESERVED_PATH_CHARS.test(segment)
}

/**
 * 校验命名空间名称（前端镜像后端规则：先 trim，再按段校验，不做小写归一）。
 * 返回 false 时调用方应提示「不支持空白/保留字符/点开头」。
 */
export function isValidNamespaceName(name: string): boolean {
  if (typeof name !== 'string') return false
  const normalized = name.trim()
  if (normalized === '' || [...normalized].length > MAX_NAMESPACE_NAME_LENGTH) {
    return false
  }
  return isValidSegment(normalized)
}

// 与后端 wikiservice.ValidatePath 的判定保持一致，
// 见 apps/gooseforum/app/service/wikiservice/path.go。
// 页面路径：namespace/slug[/slug...]，每段与目录名兼容（支持中文），
// 至少 2 段、每段 ≤64（按码点）、总长 ≤255。

const MAX_SLUG_LENGTH = 64
const MAX_PATH_LENGTH = 255

/**
 * 校验 wiki 页面路径（前端镜像后端规则：先 trim，再按 '/' 分段检查每段格式与长度，
 * 不做小写归一）。返回 false 时调用方应提示「路径格式：namespace/slug」。
 */
export function isValidWikiPath(path: string): boolean {
  if (typeof path !== 'string') return false
  const normalized = path.trim()
  if (normalized === '' || normalized.length > MAX_PATH_LENGTH) {
    return false
  }
  const segments = normalized.split('/')
  if (segments.length < 2) {
    return false // 至少 namespace + 一个 slug 段
  }
  return segments.every(isValidSegment)
}
