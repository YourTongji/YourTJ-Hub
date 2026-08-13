/**
 * 课程评价计数口径统一（CourseDetailPage 使用，issue #178）。
 *
 * 顶部统计卡评论数：评价列表加载完成后以客户端实时计数（reviewTotal）为准，
 * 未加载时回退 SSR 的 reviewCount，避免统计卡与评价区标题计数分叉。
 * 创建/删除成功后通过以下纯函数同步总数，保证写入后立即反映新值而无需刷新。
 */

/** 创建评价成功后：总数 +1（与列表 unshift 同步）。 */
export function nextReviewTotalOnCreate(total: number): number {
  return total + 1
}

/** 删除评价成功后：总数 -1，下限 0（与列表 filter 同步）。 */
export function nextReviewTotalOnDelete(total: number): number {
  return Math.max(0, total - 1)
}

/** 统计卡评论数：加载完成后用客户端实时总数，未加载时回退 SSR 的 reviewCount。 */
export function resolveStatsReviewCount(loaded: boolean, clientTotal: number, ssrCount: number): number {
  return loaded ? clientTotal : ssrCount
}
