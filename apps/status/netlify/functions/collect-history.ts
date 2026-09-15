import type { Config, Context } from '@netlify/functions'
import { snapshotStore } from '../../server/blobs'
import { loadConfig } from '../../server/config'
import { collect } from '../../server/collect'
export default async (_request: Request, context: Context) => {
  await collect('history', snapshotStore(context), loadConfig())
}
export const config: Config = { schedule: '*/5 * * * *' }
