import { describe, expect, test, vi } from 'vitest'
import {
  createReviewPageLoader,
  type ReviewPageResult,
} from '../src/site/utils/course-review-loader'

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

function page(overrides: Partial<ReviewPageResult> = {}): ReviewPageResult {
  return { list: [], total: 0, ...overrides }
}

describe('createReviewPageLoader（评价列表加载竞态保护）', () => {
  test('无写操作时透传页面结果', async () => {
    const result = page({ total: 3 })
    const fetchPage = vi.fn(async () => result)
    const loader = createReviewPageLoader(fetchPage)

    await expect(loader.load(0, '')).resolves.toBe(result)
  })

  test('写操作后返回的旧首屏响应被丢弃（返回 null）', async () => {
    const gate = deferred<ReviewPageResult>()
    const fetchPage = vi.fn(() => gate.promise)
    const loader = createReviewPageLoader(fetchPage)

    // onMounted 的首次 GET 挂起，快照版本 = 1
    const loadPromise = loader.load(0, '')
    // 创建评价成功，版本 = 2，使进行中的 GET 过期
    loader.invalidate()
    // 旧快照此刻才返回
    gate.resolve(page({ total: 1 }))

    await expect(loadPromise).resolves.toBeNull()
  })

  test('invalidate 之后发起的加载不受此前失效影响', async () => {
    const result = page({ total: 5 })
    const fetchPage = vi.fn(async () => result)
    const loader = createReviewPageLoader(fetchPage)

    loader.invalidate()
    await expect(loader.load(0, '')).resolves.toBe(result)
  })

  test('过期请求的异常同样被丢弃（不干扰写操作后的状态）', async () => {
    const gate = deferred<ReviewPageResult>()
    const fetchPage = vi.fn(() => gate.promise)
    const loader = createReviewPageLoader(fetchPage)

    const loadPromise = loader.load(0, '')
    loader.invalidate()
    gate.reject(new Error('stale fetch failed'))

    await expect(loadPromise).resolves.toBeNull()
  })

  test('未过期请求的异常照常抛出', async () => {
    const fetchPage = vi.fn(async () => {
      throw new Error('network down')
    })
    const loader = createReviewPageLoader(fetchPage)

    await expect(loader.load(0, '')).rejects.toThrow('network down')
  })
})
