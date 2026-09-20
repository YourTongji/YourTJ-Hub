import { spawnSync } from 'node:child_process'

// Netlify runs this from apps/status before installing dependencies.
// Its ignore convention is 0 = skip, 1 = build (including uncertain comparisons).
const { CONTEXT: context, CACHED_COMMIT_REF: cached, COMMIT_REF: current } = process.env
console.log(`[status-build] context=${context || 'unknown'} cached=${cached || 'none'} commit=${current || 'none'}`)

function build(reason) {
  console.log(`[status-build] Build: ${reason}`)
  process.exit(1)
}

// A previously built preview is not evidence that production has been published.
if (context === 'production') build('production releases always build')
if (!['deploy-preview', 'branch-deploy'].includes(context)) build('unknown deployment context')

// Without a cache Netlify can report the current commit as the cached commit.
if (!cached || !current || cached === current) build('no distinct cached commit to compare')

const comparison = spawnSync('git', ['diff', '--quiet', cached, current, '--', '.'])
if (comparison.error || comparison.status !== 0) build('status changes or comparison unavailable')

console.log('[status-build] Skip: status directory unchanged between distinct commits')
process.exit(0)
