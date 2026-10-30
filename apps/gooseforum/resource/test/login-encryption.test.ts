import { constants, generateKeyPairSync, privateDecrypt, webcrypto } from 'node:crypto'
import { afterEach, expect, test, vi } from 'vitest'

vi.mock('../src/runtime/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }))
import { login } from '../src/runtime/api'

const { publicKey, privateKey } = generateKeyPairSync('rsa', {
  modulusLength: 2048,
  publicKeyEncoding: { type: 'spki', format: 'pem' },
  privateKeyEncoding: { type: 'pkcs8', format: 'pem' },
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

for (const path of ['webcrypto', 'no-subtle', 'subtle-failure'] as const) {
  test(`password login encrypts RSA-OAEP SHA-256 through ${path}`, async () => {
    const password = '中文密码A123'
    const ts = 1793289600000
    const crypto = path === 'no-subtle' ? {} : webcrypto
    if (path === 'subtle-failure') vi.spyOn(webcrypto.subtle, 'importKey').mockRejectedValue(new Error('unavailable'))
    const encrypted = path === 'webcrypto' ? vi.spyOn(webcrypto.subtle, 'encrypt') : undefined
    vi.stubGlobal('window', { crypto, atob, btoa, location: { origin: 'https://forum.test', href: 'https://forum.test/login' } })
    const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
      if (url === '/api/login-public-key') return Response.json({ code: 0, result: { publicKey, serverTs: ts, algorithm: 'RSA-OAEP-256' } })
      expect(url).toBe('/api/login')
      const request = JSON.parse(String(init?.body))
      const plain = privateDecrypt({ key: privateKey, padding: constants.RSA_PKCS1_OAEP_PADDING, oaepHash: 'sha256' }, Buffer.from(request.encryptedPassword, 'base64'))
      expect(JSON.parse(plain.toString('utf8'))).toEqual({ password, ts })
      expect(request.password).toBeUndefined()
      return Response.json({ code: 0, result: 'signed in' })
    })
    vi.stubGlobal('fetch', fetchMock)
    await expect(login('student', password, '', '')).resolves.toEqual({ twoFactorRequired: false })
    expect(fetchMock).toHaveBeenCalledTimes(2)
    if (encrypted) expect(encrypted).toHaveBeenCalledOnce()
  })
}
