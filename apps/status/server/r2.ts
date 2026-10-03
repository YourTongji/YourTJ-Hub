import type { R2Bucket } from '@cloudflare/workers-types'
import type { SnapshotStore, Stored } from './snapshots'

// R2 conditional puts preserve the snapshot ordering described in MADR 0057.
export function r2Store(bucket: Pick<R2Bucket, 'get' | 'put'>): SnapshotStore {
  return {
    async read<T>(key: string) {
      const object = await bucket.get(key)
      return object ? { value: await object.json<Stored<T>>(), etag: object.etag } : null
    },
    async write<T>(key: string, value: Stored<T>, etag?: string) {
      return await bucket.put(key, JSON.stringify(value), {
        onlyIf: etag ? { etagMatches: etag } : { etagDoesNotMatch: '*' },
        httpMetadata: { contentType: 'application/json' },
      }) !== null
    },
  }
}
