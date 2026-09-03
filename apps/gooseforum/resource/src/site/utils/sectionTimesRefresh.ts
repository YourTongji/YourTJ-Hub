// 排课器节次作息「静默刷新」逻辑（与页面渲染解耦，便于单测）。
//
// 背景：/schedule 的节次作息只随 SSR payload（props.sectionTimes）注入一次。
// 管理端保存新作息后，停留在页面上的用户（尤其 bfcache/切后台恢复）仍看到旧时间。
// 本模块在页面恢复事件（pageshow persisted / visibilitychange visible）触发时
// 重新拉取当前页 payload 并比对，仅在作息表变化时回调 apply —— 避免无谓的网络
// 请求与表格重渲染。配合后端 HTML no-store，双端保证「保存即同步」。
import type { SectionTime } from '@/site/utils/sectionTimes'

export interface SectionTimesRefresherOptions {
  /** 最小刷新间隔（毫秒）。默认 5000：与服务端 scheduleSettings 5s 缓存 TTL 同量级。 */
  minIntervalMs?: number
  /** 取当前时间（可注入便于测试）。默认 Date.now。 */
  now?: () => number
  /** 页面挂载时已应用的作息表（SSR props）；首次拉取与之相同则不回调 apply。 */
  initial?: SectionTime[]
}

export interface SectionTimesRefresher {
  /** 触发一次刷新（幂等 + 节流 + 错误静默）。返回是否真正发起了拉取。 */
  refresh(): Promise<boolean>
  /** 从 payload 中提取节次作息（未携带时为 undefined）。 */
  extract(payload: unknown): SectionTime[] | undefined
  dispose(): void
}

export interface SectionTimesRefresherDeps {
  /** 拉取当前页 payload（注入 fetchPage，便于测试与静默失败）。 */
  fetchPayload: () => Promise<unknown>
  /** 作息表变化时回调（注入 store.setSectionTimeOverrides）。 */
  apply: (times: SectionTime[]) => void
}

/** 深比较两份节次作息表（顺序敏感；undefined 与空数组视为等价）。 */
export function sectionTimesEqual(a: SectionTime[] | undefined, b: SectionTime[] | undefined): boolean {
  const left = a ?? []
  const right = b ?? []
  if (left.length !== right.length) return false
  for (let i = 0; i < left.length; i++) {
    const l = left[i]
    const r = right[i]
    if (l.section !== r.section || l.start !== r.start || l.end !== r.end) return false
  }
  return true
}

/**
 * 创建作息表刷新器：
 * - refresh() 节流（minIntervalMs 内重复触发不拉取）；
 * - 拉取失败静默（网络/瞬时错误不打断排课器使用，下个恢复事件重试）；
 * - 作息表与当前应用值相同时不回调 apply（避免表格无谓重渲染）。
 */
export function createSectionTimesRefresher(deps: SectionTimesRefresherDeps, options?: SectionTimesRefresherOptions): SectionTimesRefresher {
  const minIntervalMs = options?.minIntervalMs ?? 5000
  const now = options?.now ?? Date.now
  let lastApplied: SectionTime[] | undefined = options?.initial
  let lastRefreshedAt = 0
  let disposed = false

  function extract(payload: unknown): SectionTime[] | undefined {
    if (!payload || typeof payload !== 'object') return undefined
    const props = (payload as { props?: { sectionTimes?: unknown } }).props
    if (!props || !Array.isArray(props.sectionTimes)) return undefined
    return props.sectionTimes as SectionTime[]
  }

  async function refresh(): Promise<boolean> {
    if (disposed) return false
    const nowAt = now()
    if (nowAt - lastRefreshedAt < minIntervalMs) return false
    lastRefreshedAt = nowAt
    let payload: unknown
    try {
      payload = await deps.fetchPayload()
    } catch {
      // 拉取失败静默：不打断排课器使用，下个恢复事件重试。
      return false
    }
    // dispose() 之后才返回的过期响应必须丢弃：页面已卸载（可能已重新挂载并应用了
    // 更新的作息），用旧响应覆盖会重新引入「不同步」（review P2 stale 竞态）。
    if (disposed) return false
    const times = extract(payload)
    if (times === undefined || sectionTimesEqual(times, lastApplied)) return false
    lastApplied = times
    deps.apply(times)
    return true
  }

  return {
    refresh,
    extract,
    dispose() {
      disposed = true
    },
  }
}
