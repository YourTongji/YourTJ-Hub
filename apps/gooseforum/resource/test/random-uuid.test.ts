import { afterEach, describe, expect, test, vi } from 'vitest'
import { createUuidV4 } from '../src/runtime/uuid'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('createUuidV4', () => {
  test('prefers the native randomUUID implementation', () => {
    const uuid = '00000000-0000-4000-8000-000000000001'
    const randomUUID = vi.fn(() => uuid)
    const getRandomValues = vi.fn()
    vi.stubGlobal('crypto', { randomUUID, getRandomValues })

    expect(createUuidV4()).toBe(uuid)
    expect(uuid).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
    expect(randomUUID).toHaveBeenCalledOnce()
    expect(getRandomValues).not.toHaveBeenCalled()
  })

  test('uses getRandomValues to produce an RFC 4122 version 4 UUID', () => {
    const getRandomValues = vi.fn((bytes: Uint8Array) => {
      bytes.fill(0xff)
      return bytes
    })
    vi.stubGlobal('crypto', { getRandomValues })

    const uuid = createUuidV4()

    expect(getRandomValues).toHaveBeenCalledOnce()
    expect(getRandomValues.mock.calls[0][0]).toHaveLength(16)
    expect(uuid).toBe('ffffffff-ffff-4fff-bfff-ffffffffffff')
    expect(uuid).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
  })

  test('returns unique UUID v4 strings for a large batch of secure random values', () => {
    const originalCrypto = globalThis.crypto
    vi.stubGlobal('crypto', {
      getRandomValues: originalCrypto.getRandomValues.bind(originalCrypto),
    })

    const uuids = Array.from({ length: 10_000 }, () => createUuidV4())

    expect(new Set(uuids).size).toBe(uuids.length)
    expect(uuids.every(uuid => /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(uuid))).toBe(true)
  })

  test('fails explicitly when secure randomness is unavailable', () => {
    vi.stubGlobal('crypto', undefined)

    expect(() => createUuidV4()).toThrow(/secure random/i)
    vi.stubGlobal('crypto', {})
    expect(() => createUuidV4()).toThrow(/secure random/i)
  })
})
