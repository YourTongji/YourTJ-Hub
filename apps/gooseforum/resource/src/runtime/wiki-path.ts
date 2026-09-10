// GitHub SSOT：路径保留中文等 Unicode（不再小写归一），URL 按段编码。
// 只编码每段，保留 "/" 分隔符。
export function wikiHref(path: string | undefined | null): string | undefined {
  if (!path) return undefined
  return '/wiki/' + path.split('/').map((seg) => encodeURIComponent(seg)).join('/')
}
