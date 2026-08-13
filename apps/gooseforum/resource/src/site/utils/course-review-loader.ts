import type { ReviewPayload } from '@/runtime/api'

/** 评价列表一页的返回结构（与 runtime `ReviewPage` 对齐）。 */
export interface ReviewPageResult {
  list: ReviewPayload[]
  total: number
  nextCursor?: string
}

export type ReviewPageFetcher = (
  offeringId: number,
  cursor: string,
) => Promise<ReviewPageResult>

/**
 * 带请求版本号（epoch）的列表加载协调器。
 *
 * 竞态场景（issue #178 review P1）：`onMounted` 立即发起首屏 GET，而「写评价」在
 * `reviewLoading` 期间仍可提交。若该 GET 读到创建前的旧快照、却在 POST 成功后才返回，
 * 会无条件把旧 `list`/`total` 写回，覆盖用户刚发布成功的评价。这里用 epoch 版本号规避：
 *
 * - `load()` 发起请求时递增 epoch 并记录快照版本；
 * - 写操作（创建/删除）成功后调用 `invalidate()` 递增 epoch，使所有进行中的请求过期；
 * - 请求返回时若版本号已变，返回 `null` 表示过期，调用方丢弃该响应（含过期请求的异常）。
 */
export function createReviewPageLoader(fetchPage: ReviewPageFetcher) {
  let epoch = 0

  return {
    /** 发起一次加载；若期间发生写操作（invalidate），返回 null 表示过期，调用方应丢弃。 */
    async load(offeringId: number, cursor: string): Promise<ReviewPageResult | null> {
      const myEpoch = ++epoch
      try {
        const page = await fetchPage(offeringId, cursor)
        return myEpoch === epoch ? page : null
      } catch (error) {
        // 过期请求的异常同样丢弃：写操作已发生，旧请求的错误不应干扰当前状态。
        if (myEpoch !== epoch) return null
        throw error
      }
    },

    /** 写操作（创建/删除）成功后调用：使所有进行中的加载过期。 */
    invalidate(): void {
      epoch += 1
    },
  }
}
