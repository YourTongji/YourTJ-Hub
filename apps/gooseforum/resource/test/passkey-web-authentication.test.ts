// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

const { feedFetch } = vi.hoisted(() => ({ feedFetch: vi.fn() }))

vi.mock('../src/runtime/feed-telemetry', () => ({
  feedFetch,
  feedTopicHeaders: vi.fn(),
}))

import { i18n } from '../src/runtime/i18n'
import { loginWithPasskey } from '../src/runtime/api'

const buffer = (bytes: number[]) => new Uint8Array(bytes).buffer
const envelope = (result: unknown) => new Response(JSON.stringify({ code: 0, result }), { status: 200 })

describe('WebAuthn passkey login', () => {
  let getCredential: ReturnType<typeof vi.fn>
  let credentialsDescriptor: PropertyDescriptor | undefined

  beforeEach(() => {
    feedFetch.mockReset()
    getCredential = vi.fn()
    credentialsDescriptor = Object.getOwnPropertyDescriptor(navigator, 'credentials')
    Object.defineProperty(navigator, 'credentials', {
      configurable: true,
      value: { get: getCredential, create: vi.fn() } as unknown as CredentialsContainer,
    })
    Object.defineProperty(window, 'isSecureContext', { configurable: true, value: true })
    vi.stubGlobal('PublicKeyCredential', class PublicKeyCredentialStub {})
  })

  afterEach(() => {
    if (credentialsDescriptor) {
      Object.defineProperty(navigator, 'credentials', credentialsDescriptor)
    } else {
      Reflect.deleteProperty(navigator, 'credentials')
    }
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  test('requests a discoverable credential and submits the standard assertion response', async () => {
    feedFetch
      .mockResolvedValueOnce(envelope({
        ceremonyId: 'ceremony-1',
        publicKey: { challenge: 'AQI', rpId: 'f.yourtj.de', userVerification: 'required' },
      }))
      .mockResolvedValueOnce(envelope({}))
    getCredential.mockResolvedValue({
      id: 'AQ',
      rawId: buffer([1]),
      type: 'public-key',
      authenticatorAttachment: 'platform',
      response: {
        clientDataJSON: buffer([2]),
        authenticatorData: buffer([3]),
        signature: buffer([4]),
        userHandle: null,
      },
    })

    await loginWithPasskey()

    expect(getCredential).toHaveBeenCalledWith({
      publicKey: expect.objectContaining({
        challenge: new Uint8Array([1, 2]),
        rpId: 'f.yourtj.de',
        userVerification: 'required',
      }),
    })
    expect(feedFetch).toHaveBeenCalledTimes(2)
    const finish = feedFetch.mock.calls[1][1] as RequestInit
    expect(JSON.parse(String(finish.body))).toEqual({
      ceremonyId: 'ceremony-1',
      credential: {
        id: 'AQ',
        rawId: 'AQ',
        type: 'public-key',
        authenticatorAttachment: 'platform',
        response: {
          clientDataJSON: 'Ag',
          authenticatorData: 'Aw',
          signature: 'BA',
          userHandle: null,
        },
      },
    })
  })

  test('maps user cancellation to the localized retry message', async () => {
    feedFetch.mockResolvedValueOnce(envelope({
      ceremonyId: 'ceremony-1',
      publicKey: { challenge: 'AQ', rpId: 'f.yourtj.de', userVerification: 'required' },
    }))
    getCredential.mockRejectedValue(new DOMException('dismissed', 'NotAllowedError'))

    await expect(loginWithPasskey()).rejects.toThrow(i18n.global.t('auth.passkeyCancelled'))
    expect(feedFetch).toHaveBeenCalledTimes(1)
  })

  test('does not start a ceremony outside a secure context', async () => {
    Object.defineProperty(window, 'isSecureContext', { configurable: true, value: false })

    await expect(loginWithPasskey()).rejects.toThrow(i18n.global.t('auth.passkeyUnavailable'))
    expect(feedFetch).not.toHaveBeenCalled()
  })
})
