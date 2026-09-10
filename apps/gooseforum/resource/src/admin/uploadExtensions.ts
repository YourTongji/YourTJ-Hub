// 上传扩展名白名单的受支持集合（issue #408）。
// 与后端 app/bundles/imagepolicy 的 canonical 集合保持同步：
// 服务端校验是权威，此处集合只用于管理端交互层的即时提示。
export const SUPPORTED_UPLOAD_EXTENSIONS = ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp'] as const

export function normalizeExtensionToken(raw: string): string {
  const token = raw.trim().toLowerCase()
  if (token && !token.startsWith('.')) return `.${token}`
  return token
}

export function isSupportedUploadExtension(raw: string): boolean {
  const token = normalizeExtensionToken(raw)
  return (SUPPORTED_UPLOAD_EXTENSIONS as readonly string[]).includes(token)
}
