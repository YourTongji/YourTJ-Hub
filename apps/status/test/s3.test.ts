import { expect, it, vi } from 'vitest'
import { GetObjectCommand, PutObjectCommand, type S3Client } from '@aws-sdk/client-s3'
import { s3Store } from '../server/s3'
import { cacheKey, loadConfig } from '../server/config'
import { createHash } from 'node:crypto'
import { hash } from '../server/hash'

const value = { attemptedAt: 1000, fetchedAt: new Date(1000).toISOString(), failed: false, data: { cpu: 0 } }
function harness() {
  const send = vi.fn()
  return { send, store: s3Store({ send } as unknown as S3Client, 'private-bucket') }
}
it('keeps native Web Crypto fingerprints identical to existing Node snapshot keys', async () => {
  for (const value of ['', 'public-source', '账号范围:v1/设备🔐']) {
    expect(await hash(value)).toBe(createHash('sha256').update(value).digest('hex'))
  }
})
it('reads R2 JSON with its ETag and distinguishes missing objects from failed storage', async () => {
  const { send, store } = harness()
  send.mockResolvedValueOnce({ Body: { transformToString: async () => JSON.stringify(value) }, ETag: '"etag"' })
  expect(await store.read('key')).toEqual({ value, etag: '"etag"' })
  expect(send.mock.calls[0][0]).toBeInstanceOf(GetObjectCommand)
  expect(send.mock.calls[0][0].input).toEqual({ Bucket: 'private-bucket', Key: 'key' })
  send.mockRejectedValueOnce({ $metadata: { httpStatusCode: 404 } })
  expect(await store.read('missing')).toBeNull()
  send.mockRejectedValueOnce({ $metadata: { httpStatusCode: 403 }, message: 'private error' })
  await expect(store.read('denied')).rejects.toThrow('Cannot read snapshot storage')
  send.mockResolvedValueOnce({ Body: { transformToString: async () => '{}' } })
  await expect(store.read('invalid')).rejects.toThrow('Cannot read snapshot storage')
})
it('uses conditional writes and exposes conflicts separately from permission or network errors', async () => {
  const { send, store } = harness()
  send.mockResolvedValue({})
  expect(await store.write('key', value)).toBe(true)
  expect(send.mock.calls[0][0]).toBeInstanceOf(PutObjectCommand)
  expect(send.mock.calls[0][0].input).toMatchObject({ Bucket: 'private-bucket', IfNoneMatch: '*', Body: JSON.stringify(value) })
  await store.write('key', value, '"etag"')
  expect(send.mock.calls[1][0].input.IfMatch).toBe('"etag"')
  expect(send.mock.calls[1][0].input.IfNoneMatch).toBeUndefined()
  send.mockRejectedValueOnce({ $metadata: { httpStatusCode: 412 } })
  expect(await store.write('key', value)).toBe(false)
  send.mockRejectedValueOnce(new Error('private connection error'))
  await expect(store.write('key', value)).rejects.toThrow('Cannot write snapshot storage')
})
it('lets a credential-free Worker read the collector device scope and invalidates revoked scopes', async () => {
  const vars = { STATUS_ENABLED: 'true', UMAMI_URL: 'https://analytics.example.com', UMAMI_SHARE_ID: 'share', UMAMI_DEVICE_REVISION: 'v1' }
  const reader = loadConfig(vars)
  const collector = loadConfig({ ...vars, UMAMI_USERNAME: 'readonly', UMAMI_PASSWORD: 'private' })
  expect(await cacheKey(reader, 'umami', 'devices-7d')).toBe(await cacheKey(collector, 'umami', 'devices-7d'))
  expect(await cacheKey(reader, 'umami', 'devices-7d')).not.toBe(await cacheKey(loadConfig({ ...vars, UMAMI_DEVICE_REVISION: 'v2' }), 'umami', 'devices-7d'))
})
