// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { uploadAvatar } from '../src/runtime/api'
afterEach(() => { vi.unstubAllGlobals() })
test('avatar upload sends the bot ID only when editing a bot', async () => {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify({ code: 0, result: '/avatar.webp' }), { headers: { 'Content-Type': 'application/json' } }))
  vi.stubGlobal('fetch', fetchMock)
  const file = new File(['image'], 'avatar.webp', { type: 'image/webp' })
  await uploadAvatar([file, file], 9)
  const form = (fetchMock.mock.calls[0] as unknown as [string, RequestInit])[1].body as FormData
  expect(form.get('agentId')).toBe('9')
  expect(form.get('avatar')).toBeInstanceOf(File)
  expect(form.get('avatarMedium')).toBeInstanceOf(File)
  await uploadAvatar(file)
  const viewerForm = (fetchMock.mock.calls[1] as unknown as [string, RequestInit])[1].body as FormData
  expect(viewerForm.has('agentId')).toBe(false)
})
