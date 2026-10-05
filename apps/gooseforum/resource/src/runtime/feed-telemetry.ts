import type { TopicPayload } from '@gooseforum/client'

type Context = { trace: string; position: number; topic: number }
type Patch = { trace: string; visibleMask: number; dwell: Record<number, number> }
let selected: Context | undefined
const patches = new Map<string, Patch>()
let account = 0
let retry = false
let flushing = false
let timer: ReturnType<typeof setInterval> | undefined
let listenersBound = false
let detail:
  | { context: Context; started: number; elapsed: number; running: boolean; reported: number }
  | undefined

export function resetFeedAccount(id: number) {
  if (id === account) return
  retry = false
  if (timer) {
    clearInterval(timer)
    timer = undefined
  }
  account = id
  selected = undefined
  detail = undefined
  patches.clear()
}

export function selectFeedTopic(topic: TopicPayload) {
  selected =
    topic.feedTrace && topic.feedPosition != null
      ? { trace: topic.feedTrace, position: topic.feedPosition, topic: topic.id }
      : undefined
}

export function feedRequestHeaders(path: string | URL): Record<string, string> {
  if (!selected) return {}
  const url = new URL(path.toString(), window.location.origin)
  if (url.origin !== window.location.origin) return {}
  if (url.pathname === `/p/post/${selected.topic}` || url.pathname.startsWith('/api/forum/')) {
    return {
      'X-Goose-Feed-Trace': selected.trace,
      'X-Goose-Feed-Position': String(selected.position),
      'X-Goose-Feed-Topic': String(selected.topic),
    }
  }
  return {}
}

export function feedFetch(input: string | URL, init: RequestInit = {}) {
  const headers = new Headers(init.headers)
  for (const [key, value] of Object.entries(feedRequestHeaders(input))) headers.set(key, value)
  return fetch(input, { ...init, headers }).then((response) => {
    if (
      response.ok &&
      typeof window !== 'undefined' &&
      /\/api\/(logout|forum\/account-close)$/.test(new URL(input.toString(), window.location.origin).pathname)
    )
      resetFeedAccount(0)
    return response
  })
}

function patchFor(context: Context) {
  let patch = patches.get(context.trace)
  if (!patch) {
    if (patches.size >= 50) return undefined
    patch = { trace: context.trace, visibleMask: 0, dwell: {} }
    patches.set(context.trace, patch)
  }
  return patch
}

function startTimer() {
  if (timer) return
  timer = setInterval(() => {
    updateDwell()
    void flushFeedEvents()
  }, 5000)
  if (listenersBound) return
  listenersBound = true
  window.addEventListener('pagehide', () => {
    updateDwell()
    void flushFeedEvents(true)
  })
  document.addEventListener('visibilitychange', () => {
    updateDwell()
    if (detail) {
      detail.running = !document.hidden
      detail.started = performance.now()
    }
    if (document.hidden) void flushFeedEvents(true)
  })
}

export function beginFeedDetail(topic: number) {
  endFeedDetail()
  if (selected?.topic !== topic) return
  detail = {
    context: selected,
    started: performance.now(),
    elapsed: 0,
    running: !document.hidden,
    reported: 0,
  }
  startTimer()
}

function updateDwell() {
  if (!detail) return
  const now = performance.now()
  if (detail.running) detail.elapsed += Math.max(0, now - detail.started)
  detail.started = now
  const seconds = Math.min(600, Math.floor(detail.elapsed / 5000) * 5)
  if (seconds > detail.reported) {
    detail.reported = seconds
    const patch = patchFor(detail.context)
    if (patch)
      patch.dwell[detail.context.position] = Math.max(patch.dwell[detail.context.position] ?? 0, seconds)
  }
}

export function endFeedDetail() {
  updateDwell()
  detail = undefined
}

export async function flushFeedEvents(keepalive = false) {
  if (flushing || !patches.size) return
  const currentAccount = account
  const data: Patch[] = []
  for (const patch of patches.values()) {
    const row = { ...patch, dwell: { ...patch.dwell } }
    if (new TextEncoder().encode(JSON.stringify({ patches: [...data, row] })).length > 32768) break
    data.push(row)
  }
  if (!data.length) {
    patches.clear()
    return
  }
  for (const row of data) patches.delete(row.trace)
  flushing = true
  try {
    const response = await fetch('/api/forum/feed/events', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ patches: data }),
      keepalive,
    })
    if (!response.ok) throw new Error('feed events unavailable')
    retry = false
  } catch {
    // One bounded in-memory retry. Never persist traces or block navigation.
    if (currentAccount === account && !retry) {
      retry = true
      for (const p of data) {
        const existing = patches.get(p.trace)
        if (existing) {
          existing.visibleMask |= p.visibleMask
          for (const [pos, seconds] of Object.entries(p.dwell))
            existing.dwell[Number(pos)] = Math.max(existing.dwell[Number(pos)] ?? 0, seconds)
        } else if (patches.size < 50) patches.set(p.trace, p)
      }
    }
  } finally {
    flushing = false
  }
}

/** Each mounted row may be reused: cancel its one-second timer whenever its
 * trace/position changes, leaves the viewport, or the document goes backstage. */
export function observeFeedRows(root: HTMLElement, topics: () => TopicPayload[]) {
  startTimer()
  const pending = new Map<Element, ReturnType<typeof setTimeout>>()
  const visible = new Set<Element>()
  const reported = new Set<string>()
  const cancel = (element: Element) => {
    const t = pending.get(element)
    if (t) clearTimeout(t)
    pending.delete(element)
  }
  const begin = (element: Element) => {
    cancel(element)
    if (document.hidden) return
    const id = Number((element as HTMLElement).dataset.feedId)
    const topic = topics().find((t) => t.id === id)
    if (!topic?.feedTrace || topic.feedPosition == null) return
    const context = { trace: topic.feedTrace, position: topic.feedPosition, topic: id }
    const key = `${context.trace}:${context.position}`
    if (reported.has(key)) return
    pending.set(
      element,
      setTimeout(() => {
        pending.delete(element)
        const current = topics().find((t) => t.id === id)
        if (
          !document.hidden &&
          visible.has(element) &&
          current?.feedTrace === context.trace &&
          current.feedPosition === context.position
        ) {
          const patch = patchFor(context)
          if (patch) {
            patch.visibleMask |= 1 << context.position
            reported.add(key)
          }
        }
      }, 1000),
    )
  }
  const observer = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (e.isIntersecting && e.intersectionRatio >= 0.5) {
          visible.add(e.target)
          begin(e.target)
        } else {
          visible.delete(e.target)
          cancel(e.target)
        }
      }
    },
    { threshold: [0, 0.5] },
  )
  const scan = () => {
    for (const element of root.querySelectorAll('[data-feed-id]')) {
      observer.observe(element)
      if (visible.has(element)) begin(element)
    }
    if (reported.size > 2400) reported.clear()
  }
  const mutation = new MutationObserver(scan)
  mutation.observe(root, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ['data-feed-id', 'data-feed-trace'],
  })
  const visibility = () => {
    for (const element of visible) {
      if (document.hidden) cancel(element)
      else begin(element)
    }
  }
  const click = (event: Event) => {
    const row = (event.target as Element)?.closest('[data-feed-id]')
    if (!row) return
    const topic = topics().find((t) => t.id === Number((row as HTMLElement).dataset.feedId))
    if (topic) selectFeedTopic(topic)
  }
  root.addEventListener('click', click, true)
  document.addEventListener('visibilitychange', visibility)
  scan()
  return () => {
    observer.disconnect()
    mutation.disconnect()
    for (const e of pending.keys()) cancel(e)
    visible.clear()
    root.removeEventListener('click', click, true)
    document.removeEventListener('visibilitychange', visibility)
    void flushFeedEvents()
  }
}
