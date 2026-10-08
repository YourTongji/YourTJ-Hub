import { shallowRef } from 'vue'
import { setSessionOwner } from './for-you-sessions'
import type { SeenPatch, SeenProof, TopicPayload } from '@gooseforum/client'

type Context = { trace: string; position: number; topic: number }
type Patch = { trace: string; visibleMask: number; dwell: Record<number, number> }
let selected: Context | undefined
const patches = new Map<string, Patch>()
const seen = new Map<string, { patch: SeenPatch; expiresAt: number }>()
const received = new Map<string, number>()
let seenCount = 0
const invalidSeenProofs = new Map<string, number>()
export const seenConfirmationLost = shallowRef(false)

export function pendingSeenPatches(): SeenPatch[] {
  const result: SeenPatch[] = []
  for (const { patch, expiresAt } of seen.values()) {
    if (expiresAt <= Date.now()) {
      seen.delete(patch.proof)
      seenCount -= Object.keys(patch.seen).length
      seenConfirmationLost.value = true
      continue
    }
    result.push({ proof: patch.proof, seen: { ...patch.seen } })
  }
  return result
}
export function acknowledgeSeen(rows: SeenPatch[]) {
  for (const row of rows) {
    const pending = seen.get(row.proof)
    if (!pending) continue
    for (const [pos, elapsed] of Object.entries(row.seen)) {
      if (pending.patch.seen[Number(pos)] === elapsed) {
        delete pending.patch.seen[Number(pos)]
        seenCount--
      }
    }
    if (!Object.keys(pending.patch.seen).length) seen.delete(row.proof)
  }
}
// Invalid process-epoch proofs cannot be repaired by retiming old observations.
export function invalidateSeen(rows: SeenPatch[]) {
  for (const row of rows) {
    const pending = seen.get(row.proof)
    if (!pending) continue
    invalidSeenProofs.set(row.proof, pending.expiresAt)
    seenCount -= Object.keys(pending.patch.seen).length
    seen.delete(row.proof)
  }
  while (invalidSeenProofs.size > 120) invalidSeenProofs.delete(invalidSeenProofs.keys().next().value!)
  seenConfirmationLost.value = true
}
function recordSeen(proof: SeenProof, position: number) {
  for (const [token, expiresAt] of invalidSeenProofs) if (expiresAt <= Date.now()) invalidSeenProofs.delete(token)
  if (seenCount >= 120 || proof.expiresAt <= Date.now() || invalidSeenProofs.has(proof.token)) return false
  let row = seen.get(proof.token)
  if (!row) { row = { patch: { proof: proof.token, seen: {} }, expiresAt: proof.expiresAt }; seen.set(proof.token, row) }
  if (row.patch.seen[position] == null) {
    row.patch.seen[position] = Math.max(1000, Math.floor(performance.now() - (received.get(proof.token) ?? performance.now())))
    seenCount++
  }
  return true
}
export async function confirmPendingSeen() {
  while (seenCount) await flushFeedEvents(false, true)
}
let account = 0
let accountRevision = 0
let retry = false
let flushing = false
let timer: ReturnType<typeof setInterval> | undefined
let listenersBound = false
let detail:
  | { context: Context; started: number; elapsed: number; running: boolean; reported: number }
  | undefined

export function feedAccount() { return account }
export function feedAccountRevision() { return accountRevision }
export class StaleFeedAccountError extends Error {}

export function resetFeedAccount(id: number) {
  if (id === account) return
  accountRevision++
  retry = false
  if (timer) {
    clearInterval(timer)
    timer = undefined
  }
  account = id
  setSessionOwner(id)
  selected = undefined
  detail = undefined
  patches.clear()
  seen.clear()
  received.clear()
  seenCount = 0
  invalidSeenProofs.clear()
  seenConfirmationLost.value = false
}

export function selectFeedTopic(topic: TopicPayload) {
  selected =
    topic.feedTrace && topic.feedPosition != null
      ? { trace: topic.feedTrace, position: topic.feedPosition, topic: topic.id }
      : undefined
}

export function feedTopicHeaders(topic: TopicPayload): Record<string, string> {
  if (!topic.feedTrace || topic.feedPosition == null) return {}
  return {
    'X-Goose-Feed-Trace': topic.feedTrace,
    'X-Goose-Feed-Position': String(topic.feedPosition),
    'X-Goose-Feed-Topic': String(topic.id),
  }
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
  for (const [key, value] of Object.entries(feedRequestHeaders(input))) {
    if (!headers.has(key)) headers.set(key, value)
  }
  return fetch(input, { ...init, headers }).then((response) => {
    if (
      response.ok &&
      typeof window !== 'undefined' &&
      /\/api\/(logout|forum\/user\/account-close)$/.test(new URL(input.toString(), window.location.origin).pathname)
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
  const context = selected
  endFeedDetail()
  if (context?.topic !== topic) return
  selected = context
  detail = {
    context,
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
  selected = undefined
}

let currentFlush: Promise<void> | undefined
export async function flushFeedEvents(keepalive = false, required = false): Promise<void> {
  if (currentFlush) { await currentFlush; if (required && seenCount) return flushFeedEvents(keepalive, required); return }
  currentFlush = sendFeedEvents(keepalive, required)
  try { await currentFlush } finally { currentFlush = undefined }
}
async function sendFeedEvents(keepalive: boolean, required: boolean) {
  if (flushing || (!patches.size && !seenCount)) return
  const currentAccount = account
  const currentRevision = accountRevision
  const data: Patch[] = []
  const seenData: SeenPatch[] = []
  try {
    for (const row of pendingSeenPatches()) {
      if (seenData.length >= 50) break
      if (new TextEncoder().encode(JSON.stringify({ patches: [], seenPatches: [...seenData, row] })).length > 30000) break
      seenData.push(row)
    }
  } catch (error) { if (required) throw error }
  for (const patch of patches.values()) {
    if (seenData.length + data.length >= 50) break
    const row = { ...patch, dwell: { ...patch.dwell } }
    if (new TextEncoder().encode(JSON.stringify({ patches: [...data, row], seenPatches: seenData })).length > 32768) break
    data.push(row)
  }
  if (!data.length && !seenData.length) { if (required && seenCount) throw new Error("seen confirmation unavailable"); return }
  for (const row of data) patches.delete(row.trace)
  flushing = true
  try {
    const response = await fetch('/api/forum/feed/events', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ patches: data, ...(seenData.length ? { seenPatches: seenData } : {}) }),
      keepalive,
    })
    if (response.status === 400 && currentRevision === accountRevision && seenData.length) invalidateSeen(seenData)
    if (!response.ok) throw new Error('feed events unavailable')
    if (seenData.length) {
      const body = await response.json()
      if (body.code !== 0 || !body.result?.seenConfirmed) throw new Error('seen confirmation unavailable')
      if (currentAccount === account && currentRevision === accountRevision) acknowledgeSeen(seenData)
    }
    retry = false
  } catch (error) {
    // One bounded in-memory retry. Never persist traces or block navigation.
    if (currentAccount === account && currentRevision === accountRevision && !retry) {
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
    if (required) throw error
  } finally {
    flushing = false
  }
}

/** Each mounted row may be reused: cancel its one-second timer whenever its
 * trace/position changes, leaves the viewport, or the document goes backstage. */
export function observeFeedRows(root: HTMLElement, topics: () => TopicPayload[], proofs: () => SeenProof[] = () => []) {
  startTimer()
  const pending = new Map<Element, ReturnType<typeof setTimeout>>()
  const visible = new Set<Element>()
  const reported = new Set<string>()
  const cancel = (element: Element) => {
    const t = pending.get(element)
    if (t) clearTimeout(t)
    pending.delete(element)
  }
  const proofFor = (id: number) => {
    for (const proof of proofs()) {
      if (!received.has(proof.token)) {
        if (received.size >= 120) received.delete(received.keys().next().value!)
        received.set(proof.token, performance.now())
      }
      const position = proof.topicIds.indexOf(id)
      if (position >= 0) return { proof, position }
    }
  }
  const observerAccount = account
  const observerRevision = accountRevision
  const active = () => account === observerAccount && accountRevision === observerRevision && account > 0 && !document.hidden && root.isConnected &&
    !root.closest('[inert], [aria-hidden="true"]') &&
    !document.querySelector('dialog[open], [role="dialog"][aria-modal="true"]')
  const begin = (element: Element) => {
    cancel(element)
    if (!active()) return
    const id = Number((element as HTMLElement).dataset.feedId)
    const topic = topics().find((t) => t.id === id)
    if (!topic) return
    const claim = proofFor(id)
    const context = topic.feedTrace && topic.feedPosition != null
      ? { trace: topic.feedTrace, position: topic.feedPosition, topic: id } : undefined
    const statsKey = context ? `${context.trace}:${context.position}` : ''
    const seenKey = claim ? `seen:${claim.proof.token}:${claim.position}` : ''
    if ((!statsKey || reported.has(statsKey)) && (!seenKey || reported.has(seenKey))) return
    pending.set(element, setTimeout(() => {
      pending.delete(element)
      const current = topics().find((t) => t.id === id)
      if (!active() || !visible.has(element) || !current) return
      if (context && current.feedTrace === context.trace && current.feedPosition === context.position) {
        const patch = patchFor(context)
        if (patch) { patch.visibleMask |= 1 << context.position; reported.add(statsKey) }
      }
      if (claim && proofFor(id)?.proof.token === claim.proof.token) {
        if (recordSeen(claim.proof, claim.position)) reported.add(seenKey)
        else if (seenCount >= 120) pending.set(element, setTimeout(() => begin(element), 1000))
      }
    }, 1000))
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
    { threshold: [0, 0.5], rootMargin: `-${Math.max(0, document.querySelector('header')?.getBoundingClientRect().bottom ?? 0)}px 0px 0px` },
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
      if (!active()) cancel(element)
      else begin(element)
    }
  }
  const overlays = new MutationObserver(visibility)
  overlays.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ['open', 'aria-modal', 'inert', 'aria-hidden'] })
  const click = (event: Event) => {
    const mouse = event as MouseEvent
    const link = (event.target as Element)?.closest<HTMLAnchorElement>('a[href]')
    if (!link || link.target === '_blank' || mouse.button !== 0 || mouse.metaKey || mouse.ctrlKey || mouse.shiftKey || mouse.altKey) return
    const row = link.closest('[data-feed-id]')
    if (!row) return
    const topic = topics().find((t) => t.id === Number((row as HTMLElement).dataset.feedId))
    const url = new URL(link.href, window.location.origin)
    if (topic && url.origin === window.location.origin && url.pathname === `/p/post/${topic.id}`) selectFeedTopic(topic)
  }
  root.addEventListener('click', click, true)
  document.addEventListener('visibilitychange', visibility)
  scan()
  return () => {
    observer.disconnect()
    mutation.disconnect()
    overlays.disconnect()
    for (const e of pending.keys()) cancel(e)
    visible.clear()
    root.removeEventListener('click', click, true)
    document.removeEventListener('visibilitychange', visibility)
    void flushFeedEvents()
  }
}
