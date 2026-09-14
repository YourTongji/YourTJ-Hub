import type { Config, Context } from '@netlify/functions'
import { snapshotStore } from '../../server/blobs'
import { loadConfig } from '../../server/config'
import { serveSnapshot } from '../../server/snapshots'
export default async (request: Request, context: Context) => {
  try { return await serveSnapshot(request, snapshotStore(context), loadConfig()) }
  catch { return Response.json({ error: 'Snapshot storage unavailable' }, { status: 503, headers: { 'Cache-Control': 'no-store' } }) }
}
export const config: Config = { path: '/api/status' }
