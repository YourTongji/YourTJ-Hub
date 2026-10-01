import { spawnSync } from 'node:child_process'
import { statusTree, publishedTreeMatches } from './production-build.mjs'

// Netlify runs this from apps/status before installing dependencies.
// Its ignore convention is 0 = skip, 1 = build (including uncertain comparisons).
const { CONTEXT: context, CACHED_COMMIT_REF: cached, COMMIT_REF: current } = process.env
console.log(`[status-build] context=${context || 'unknown'} cached=${cached || 'none'} commit=${current || 'none'}`)

function build(reason) {
  console.log(`[status-build] Build: ${reason}`)
  process.exit(1)
}

if (process.env.STATUS_FORCE_BUILD === 'true') build('explicit rebuild requested')
if (context === 'production') {
  // Clearing the cache / rebuilding the same commit must apply environment-only
  // changes as well. The cache is never used as proof of a published version.
  if (!cached || !current || cached === current) build('fresh cache or same-commit rebuild')
  if (!await publishedTreeMatches(process.env.URL, statusTree(current))) build('production content changed or published version unavailable')
  console.log('[status-build] Skip: status source tree already published in production')
  process.exit(0)
}
if (!['deploy-preview', 'branch-deploy'].includes(context)) build('unknown deployment context')

// Without a cache Netlify can report the current commit as the cached commit.
if (!cached || !current || cached === current) build('no distinct cached commit to compare')

const comparison = spawnSync('git', ['diff', '--quiet', cached, current, '--', '.'])
if (comparison.error || comparison.status !== 0) build('status changes or comparison unavailable')

console.log('[status-build] Skip: status directory unchanged between distinct commits')
process.exit(0)
