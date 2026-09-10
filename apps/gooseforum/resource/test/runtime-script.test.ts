import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { loadRuntimeScript } from '../src/runtime/runtime-script'

class FakeScriptElement extends EventTarget {
  async = false
  dataset: Record<string, string> = {}
  id = ''
  isConnected = false
  src = ''

  constructor(private readonly elements: Map<string, FakeScriptElement>) {
    super()
  }

  remove() {
    this.isConnected = false
    if (this.elements.get(this.id) === this) this.elements.delete(this.id)
  }
}

describe('loadRuntimeScript', () => {
  const elements = new Map<string, FakeScriptElement>()
  const appended: FakeScriptElement[] = []

  beforeEach(() => {
    vi.useFakeTimers()
    elements.clear()
    appended.length = 0

    const documentStub = {
      createElement: () => new FakeScriptElement(elements),
      getElementById: (id: string) => elements.get(id) || null,
      head: {
        appendChild(script: FakeScriptElement) {
          script.isConnected = true
          elements.set(script.id, script)
          appended.push(script)
          return script
        },
      },
    }
    vi.stubGlobal('document', documentStub)
    vi.stubGlobal('window', globalThis)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  test('resolves an existing script already marked as loaded', async () => {
    const script = new FakeScriptElement(elements)
    script.id = 'asset'
    script.dataset.loaded = 'true'
    elements.set(script.id, script)

    await expect(loadRuntimeScript('/asset.js', script.id)).resolves.toBeUndefined()
    expect(appended).toHaveLength(0)
  })

  test('rebuilds an unmarked script whose load event may have already fired', async () => {
    const stale = new FakeScriptElement(elements)
    stale.id = 'asset'
    stale.isConnected = true
    elements.set(stale.id, stale)

    const loading = loadRuntimeScript('/asset.js', stale.id)
    expect(stale.isConnected).toBe(false)
    expect(appended).toHaveLength(1)

    appended[0].dispatchEvent(new Event('load'))
    await expect(loading).resolves.toBeUndefined()
    expect(appended[0].dataset.loaded).toBe('true')
  })

  test('shares one in-flight load between concurrent callers', async () => {
    const first = loadRuntimeScript('/asset.js', 'asset')
    const second = loadRuntimeScript('/asset.js', 'asset')

    expect(second).toBe(first)
    expect(appended).toHaveLength(1)

    appended[0].dispatchEvent(new Event('load'))
    await expect(Promise.all([first, second])).resolves.toEqual([undefined, undefined])
  })

  test('times out, removes the failed node, and allows a retry', async () => {
    const failed = loadRuntimeScript('/asset.js', 'asset')
    const rejection = expect(failed).rejects.toThrow('Timed out loading runtime asset')

    await vi.advanceTimersByTimeAsync(15_000)
    await rejection
    expect(elements.has('asset')).toBe(false)

    const retry = loadRuntimeScript('/asset.js', 'asset')
    expect(appended).toHaveLength(2)
    appended[1].dispatchEvent(new Event('load'))
    await expect(retry).resolves.toBeUndefined()
  })
})
