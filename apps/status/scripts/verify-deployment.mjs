import { statusTree, publishedTreeMatches } from './production-build.mjs'

const origin = process.argv[2]
const tree = statusTree(process.env.COMMIT_REF || 'HEAD')
if (!await publishedTreeMatches(origin, tree)) throw new Error('Published status content does not match this deployment')
const response = await fetch(new URL('/api/status', origin), { signal: AbortSignal.timeout(10_000), redirect: 'error' })
if (!response.ok || (await response.json()).code !== 0) throw new Error('Published status API is unavailable')
console.log('Published status source and API verified; scheduled source freshness requires a separate check.')
