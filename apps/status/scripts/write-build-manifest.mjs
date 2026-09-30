import { writeFileSync } from 'node:fs'
import { statusTree } from './production-build.mjs'

const context = process.env.CONTEXT || 'dev'
const tree = statusTree(process.env.COMMIT_REF || 'HEAD')
if (context === 'production' && !tree) throw new Error('Cannot identify production status source tree')
// Uploaded with the site, so failed builds and previews cannot advance the
// production comparison. Local builds are explicitly ineligible for skipping.
writeFileSync('dist/status-build.json', JSON.stringify({ schema: 1, context, tree }))
