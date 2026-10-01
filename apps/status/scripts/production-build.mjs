import { spawnSync } from 'node:child_process'

// A Git tree identifies the app's content across merge commits. Only immutable
// Git objects are compared; build-cache commits may belong to a preview.
export function statusTree(ref = 'HEAD') {
  if (!/^(HEAD|[a-f\d]{40,64})$/.test(ref)) return null
  const result = spawnSync('git', ['rev-parse', `${ref}:apps/status`], { encoding: 'utf8' })
  const tree = result.stdout?.trim()
  return result.status === 0 && /^[a-f\d]{40,64}$/.test(tree) ? tree : null
}

export async function publishedTreeMatches(siteUrl, tree, fetcher = fetch) {
  if (!tree) return false
  try {
    const url = new URL(siteUrl)
    if (url.protocol !== 'https:' || url.username || url.password || url.pathname !== '/' || url.search || url.hash) return false
    const response = await fetcher(new URL('/status-build.json', url), {
      signal: AbortSignal.timeout(4000), redirect: 'error', cache: 'no-store',
    })
    if (!response.ok || !response.body) return false
    const reader = response.body.getReader()
    let body = ''
    const decoder = new TextDecoder()
    try {
      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        body += decoder.decode(value, { stream: true })
        if (body.length > 2048) return false
      }
    } finally { await reader.cancel(); reader.releaseLock() }
    const manifest = JSON.parse(body + decoder.decode())
    return manifest?.schema === 1 && manifest.context === 'production' && manifest.tree === tree
  } catch { return false }
}
