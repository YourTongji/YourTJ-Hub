import { GetObjectCommand, PutObjectCommand, S3Client } from '@aws-sdk/client-s3'
import type { SnapshotStore, Stored } from './snapshots'

// The external collector uses bucket-scoped S3 credentials. Workers reads the
// same private bucket through its binding and holds no upstream/S3 credentials.
export function s3Store(client: S3Client, bucket: string): SnapshotStore {
  return {
    async read<T>(key: string) {
      try {
        const object = await client.send(new GetObjectCommand({ Bucket: bucket, Key: key }))
        if (!object.Body || !object.ETag) throw new Error('Invalid stored snapshot')
        return { value: JSON.parse(await object.Body.transformToString()) as Stored<T>, etag: object.ETag }
      } catch (error) {
        if ((error as { $metadata?: { httpStatusCode?: number } }).$metadata?.httpStatusCode === 404) return null
        throw new Error('Cannot read snapshot storage')
      }
    },
    async write<T>(key: string, value: Stored<T>, etag?: string) {
      try {
        await client.send(new PutObjectCommand({
          Bucket: bucket, Key: key, Body: JSON.stringify(value), ContentType: 'application/json',
          ...(etag ? { IfMatch: etag } : { IfNoneMatch: '*' }),
        }))
        return true
      } catch (error) {
        if ((error as { $metadata?: { httpStatusCode?: number } }).$metadata?.httpStatusCode === 412) return false
        throw new Error('Cannot write snapshot storage')
      }
    },
  }
}
