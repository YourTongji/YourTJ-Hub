// safe-url：管理员可配置 URL 的前端渲染防线（issue #409）。
// 与后端 app/bundles/urlutil 的策略一致，用于历史脏配置/绕过 API 的配置在
// 进入 href/src 前的最后降级：危险 scheme（javascript:/data:/vbscript:/file:）、
// 协议相对 //、控制字符/HTML 实体混淆一律返回空串（调用方降级为占位链接）。
// 仅 http(s) 绝对 URL 或站内相对路径通过；contact 额外放行 mailto:。
// 纯函数、无 DOM/vue 依赖，可在 vitest 直接单测。

export type SafeUrlPolicy = 'site-link' | 'external' | 'image' | 'contact'

const schemeRe = /^([a-zA-Z][a-zA-Z0-9+.-]*):/
const numericEntityRe = /&#(?:x([0-9a-fA-F]+)|([0-9]+));/g
const maxUrlLength = 2048

/** 最小 HTML 实体解码：覆盖管理端可能保存的十进制/十六进制/命名实体混淆。
 *  返回 hasInvalid=true 表示值里含越界/代理区/非法的数值实体（损坏配置）。 */
function decodeEntities(value: string): { text: string; hasInvalid: boolean } {
  let hasInvalid = false
  const text = value
    .replace(numericEntityRe, (_, hex: string | undefined, dec: string | undefined) => {
      const codePoint = hex !== undefined ? Number.parseInt(hex, 16) : Number.parseInt(dec ?? '', 10)
      // 越界码点（如 &#1114112;）、代理区（如 &#xD800;）与非法数字既不能产生
      // scheme 字符也不该出现：整个值视为损坏配置交给 safeUrl 降级为空串
      // （占位链接），而不是 String.fromCodePoint 抛 RangeError 挂掉渲染。
      // 合法码点与命名实体照常解码，不误伤。
      if (Number.isNaN(codePoint) || codePoint > 0x10ffff || (codePoint >= 0xd800 && codePoint <= 0xdfff)) {
        hasInvalid = true
        return '\ufffd'
      }
      return String.fromCodePoint(codePoint)
    })
    .replace(/&quot;/g, '"')
    .replace(/&#39;|&apos;/g, "'")
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&amp;/g, '&')
  return { text, hasInvalid }
}

function hasControl(value: string): boolean {
  // eslint-disable-next-line no-control-regex
  return /[\u0000-\u001f\u007f]/.test(value)
}

/** 镜像后端 urlutil.Clean：返回可安全用于 href/src 的值，非法返回空串。 */
export function safeUrl(raw: string | null | undefined, policy: SafeUrlPolicy = 'site-link'): string {
  if (raw == null) return ''
  const value = raw.trim()
  if (value === '' || value.length > maxUrlLength) return ''
  const decoded = decodeEntities(value)
  if (decoded.hasInvalid || decoded.text.length > maxUrlLength || hasControl(decoded.text)) return ''
  if (decoded.text.startsWith('//')) return ''

  const match = schemeRe.exec(decoded.text)
  const scheme = match ? match[1].toLowerCase() : ''
  if (scheme === '') {
    // 无 scheme = 站内相对路径；仅 external 要求绝对 http(s)。
    return policy === 'external' ? '' : value
  }
  if (scheme === 'http' || scheme === 'https') {
    // 要求是带 host 的绝对 URL；"https:" / "https://" 这类残缺值解析失败即拒绝。
    try {
      const parsed = new URL(decoded.text)
      return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? value : ''
    } catch {
      return ''
    }
  }
  if (scheme === 'mailto' && policy === 'contact' && decoded.text.length > scheme.length + 1) {
    return value
  }
  return ''
}
