// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { register, forgotPassword, resetPassword, ApiResponseError } from '../src/runtime/api'
import { i18n, setLocale } from '../src/runtime/i18n'
vi.mock('../src/runtime/feed-telemetry', () => ({ feedFetch: (...args: unknown[]) => fetch(...args as Parameters<typeof fetch>), feedTopicHeaders: () => ({}) }))
afterEach(() => { vi.unstubAllGlobals() })
const operations = [
  ['register', () => register('username','user@example.test','Password123','','')],
  ['forgot', () => forgotPassword('user@example.test','','')],
  ['reset', () => resetPassword('synthetic-token','Password123')],
] as const
for (const [name, operation] of operations) for (const locale of ['zh','en','ja','de'] as const) {
 test(`${name} preserves safe structured HTTP failure and rate-limit metadata in ${locale}`, async () => {
  await setLocale(locale)
  const fetch = vi.fn()
  vi.stubGlobal('fetch', fetch)
  fetch.mockResolvedValue(new Response(JSON.stringify({code:1,messageCode:'auth.signupDisabled',result:'private storage error'}),{status:403}))
  await expect(operation()).rejects.toMatchObject({messageCode:'auth.signupDisabled',message:i18n.global.t('server.auth.signupDisabled')})
  fetch.mockResolvedValue(new Response(JSON.stringify({code:1,messageCode:'common.rateLimited',params:{retryAfterSeconds:12},message:'secret provider error'}),{status:429,headers:{'Retry-After':'37'}}))
  const error = await operation().catch(e => e)
  expect(error).toMatchObject({messageCode:'common.rateLimited',retryAfterSeconds:37,params:{retryAfterSeconds:37}})
  expect(error).toBeInstanceOf(ApiResponseError)
  expect(error.message).toContain('37')
  expect(error.message).not.toContain('secret')
 })
}

test('unknown errors and malformed responses stay generic; successful params remain localized', async () => {
 await setLocale('en')
 const fetch = vi.fn(); vi.stubGlobal('fetch',fetch)
 for (const response of [new Response('<html>private proxy details</html>',{status:503}),new Response(JSON.stringify({code:1,messageCode:'private.unknown',result:'private SQL details',message:'private SMTP details'}),{status:400})]) {
  fetch.mockResolvedValueOnce(response)
  await expect(register('username','user@example.test','Password123','','')).rejects.toThrow(i18n.global.t('api.registerFailed'))
 }
 fetch.mockResolvedValueOnce(new Response(JSON.stringify({code:0,result:'never display this string',messageCode:'auth.login.success'}),{status:200}))
 await expect(register('username','user@example.test','Password123','','')).resolves.toBe(i18n.global.t('server.auth.login.success'))
})
