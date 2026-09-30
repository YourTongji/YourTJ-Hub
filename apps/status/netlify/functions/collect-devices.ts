import type { Config, Context } from '@netlify/functions'
import { snapshotStore } from '../../server/blobs'
import { loadConfig } from '../../server/config'
import { collect } from '../../server/collect'
export default async (_request: Request, context: Context) => {
  await collect('devices', snapshotStore(context), loadConfig())
}
// Offset expensive reports from history collection at the top of the hour.
export const config: Config = { schedule: '7 * * * *' }
