import { getStore, type Store } from '@netlify/blobs'
import type { Context } from '@netlify/functions'
import type { SnapshotStore, Stored } from './snapshots'
export function storeName(deploy: Pick<Context['deploy'], 'context' | 'id' | 'published'>): string {
  if (deploy.context === 'production' && deploy.published) return 'status-v1'
  if (!deploy.id) return 'status-local-v1'
  return `status-preview-v1-${deploy.id}`
}
export function adaptStore(store: Pick<Store, 'getWithMetadata' | 'setJSON' | 'list'>): SnapshotStore {
  async function version(key: string) {
    const result = await store.list({ prefix: key })
    return result.blobs.find(blob => blob.key === key)?.etag
  }
  return {
    async read<T>(key: string) {
      let result = await store.getWithMetadata(key, { type: 'json', consistency: 'strong' })
      if (!result) return null
      if (result.etag) return { value: result.data as Stored<T>, etag: result.etag }
      // Netlify's local Blobs server omits GET ETags, but lists them. Bracket
      // the payload read with matching versions; never attach a newer ETag to old data.
      for (let retry = 0; retry < 3; retry++) {
        const before = await version(key)
        result = await store.getWithMetadata(key, { type: 'json', consistency: 'strong' })
        const after = await version(key)
        if (!before && !result && !after) return null
        if (before && before === after && result) return { value: result.data as Stored<T>, etag: after }
      }
      throw new Error('Snapshot version unavailable')
    },
    async write<T>(key: string, value: Stored<T>, etag?: string) {
      const result = await store.setJSON(key, value, etag ? { onlyIfMatch: etag } : { onlyIfNew: true })
      return result.modified
    },
  }
}
export function snapshotStore(context: Context): SnapshotStore {
  return adaptStore(getStore({ name: storeName(context.deploy), consistency: 'strong' }))
}
