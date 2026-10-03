import { cacheKey, configured, type Config } from './config'
import { collectOne, type SnapshotStore, type Stored } from './snapshots'
import type { StatusRange, StatusServerRange } from '../src/types'

type Records = Record<string, Stored<unknown>>
const ranges: StatusRange[] = ['24h', '7d', '30d']
const serverRanges: StatusServerRange[] = ['1h', '6h', '24h', '7d']
const viewKey = (range: StatusRange, serverRange: StatusServerRange) => `views/v1/${range}/${serverRange}`

// Assemble selected public records outside Workers: an API miss reads one view
// plus one device object instead of five R2 objects. Original timestamps and
// failure flags stay intact; the reader still ages each source independently.
export async function publishViews(store: SnapshotStore, config: Config, now = Date.now) {
  const started = now()
  const keys: Promise<string>[] = []
  if (configured(config, 'komari')) keys.push(cacheKey(config, 'komari', 'current'), ...serverRanges.map(r => cacheKey(config, 'komari', `history-${r}`)))
  if (configured(config, 'uptime')) keys.push(cacheKey(config, 'uptime', 'current'))
  if (configured(config, 'umami')) keys.push(...ranges.map(r => cacheKey(config, 'umami', r)))
  const records = new Map(await Promise.all((await Promise.all(keys)).map(async key => [key, (await store.read(key))?.value] as const)))
  const writes = ranges.flatMap(range => serverRanges.map(async serverRange => {
    const selected: Records = {}
    for (const key of await Promise.all([cacheKey(config, 'komari', 'current'), cacheKey(config, 'komari', `history-${serverRange}`), cacheKey(config, 'uptime', 'current'), cacheKey(config, 'umami', range)])) {
      const value = records.get(key)
      if (value) selected[key] = value
    }
    await collectOne(store, viewKey(range, serverRange), async () => selected, () => started)
  }))
  if ((await Promise.allSettled(writes)).some(r => r.status === 'rejected')) throw new Error('Public views could not be saved')
}

export function viewStore(store: SnapshotStore, range: StatusRange, serverRange: StatusServerRange): SnapshotStore {
  let view: Promise<Records | null> | undefined
  return {
    async read<T>(key: string) {
      if (key.split('/')[2]?.startsWith('devices-')) return store.read<T>(key)
      view ??= store.read<Records>(viewKey(range, serverRange)).then(result => result?.value.data ?? null)
      const record = (await view)?.[key]
      // Scope fingerprints remain the lookup keys, so changed/revoked sources
      // cannot expose old data even before the next collector publishes a view.
      return record ? { value: record as Stored<T>, etag: 'read-only-view' } : null
    },
    async write() { throw new Error('Public views are read-only') },
  }
}
