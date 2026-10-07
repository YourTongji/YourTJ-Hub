import { shallowRef } from 'vue'
import type { HomeProps, PagePayload } from '@gooseforum/client'
import type { PreparedPage } from './router'

type Anchor = { id: number; top: number; following: number[]; preceding: number[] }
type Session = { page: PreparedPage; bytes: number; touched: number; anchor?: Anchor }
const sessions = new Map<string, Session>()
const maxBytes = 512 * 1024
let owner = 0
let shell: PreparedPage | undefined
let shellBytes = 0
export const activeForYouSession = shallowRef('')
export const forYouSessionLost = shallowRef(false)

export function setSessionOwner(id: number) {
  if (owner === id) return
  owner = id
  sessions.clear()
  shell = undefined
  shellBytes = 0
  activeForYouSession.value = ''
  forYouSessionLost.value = false
}
export function isForYou(payload: PagePayload) {
  return payload.component === 'home.index' &&
    ((payload.props as HomeProps).actualSort ?? (payload.props as HomeProps).sort) === 'for_you'
}
export function historySessionKey(position: unknown) { return `${owner}:${String(position)}` }

function prune() {
  const now = Date.now()
  for (const [key, session] of sessions) if (now - session.touched >= 30 * 60 * 1000) sessions.delete(key)
  let bytes = shellBytes + [...sessions.values()].reduce((sum, session) => sum + session.bytes, 0)
  while (sessions.size > 2 || bytes > maxBytes) {
    const oldest = [...sessions.entries()].sort((a, b) => a[1].touched - b[1].touched)[0]
    if (!oldest) break
    sessions.delete(oldest[0]); bytes -= oldest[1].bytes
  }
}
export function saveForYouSession(key: string, page: PreparedPage) {
  if (!key || !isForYou(page.payload) || !owner) return
  const props = page.payload.props as HomeProps
  if (props.topics.length > 120) return
  const shellPayload = { ...page.payload, props: { ...props, topics: [], seenProofs: [], snapshotId: '',
    pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' } } }
  const shellJson = JSON.stringify(shellPayload)
  shellBytes = new TextEncoder().encode(shellJson).length
  shell = shellBytes <= maxBytes ? { component: page.component, payload: JSON.parse(shellJson) } : undefined
  if (!shell) shellBytes = 0
  const json = JSON.stringify(page.payload)
  const bytes = new TextEncoder().encode(json).length
  if (bytes > maxBytes) { sessions.delete(key); prune(); return }
  const anchor = sessions.get(key)?.anchor
  sessions.set(key, { page: { component: page.component, payload: JSON.parse(json) }, bytes, touched: Date.now(), anchor })
  prune()
}
export function updateForYouSession(props: HomeProps) {
  const key = activeForYouSession.value
  const session = sessions.get(key)
  if (session) saveForYouSession(key, { ...session.page, payload: { ...session.page.payload, props } })
}
export function restoreForYouSession(key: string): PreparedPage | undefined {
  prune()
  const session = sessions.get(key)
  if (!session) return
  session.touched = Date.now()
  return { component: session.page.component, payload: JSON.parse(JSON.stringify(session.page.payload)) }
}
export function captureFeedAnchor(root: ParentNode = document): Anchor | undefined {
  const rows = [...root.querySelectorAll<HTMLElement>('[data-feed-id]')]
  const index = rows.findIndex((row) => row.getBoundingClientRect().bottom > 64)
  const row = rows[index]
  if (!row) return
  return { id: Number(row.dataset.feedId), top: row.getBoundingClientRect().top,
    following: rows.slice(index + 1).map((r) => Number(r.dataset.feedId)),
    preceding: rows.slice(0, index).reverse().map((r) => Number(r.dataset.feedId)) }
}
export function restoreFeedAnchor(anchor?: Anchor) {
  if (!anchor) return
  for (const id of [anchor.id, ...anchor.following, ...anchor.preceding]) {
    const row = document.querySelector<HTMLElement>(`[data-feed-id="${id}"]`)
    if (row) { window.scrollBy({ top: row.getBoundingClientRect().top - anchor.top, behavior: 'instant' }); return }
  }
}
export function saveSessionAnchor() {
  const session = sessions.get(activeForYouSession.value)
  if (session) { session.anchor = captureFeedAnchor(); session.touched = Date.now() }
}
export function sessionAnchor(key: string) { return sessions.get(key)?.anchor }

export function lostForYouPage(url: string): PreparedPage | undefined {
  if (!shell) return
  const page = { component: shell.component, payload: JSON.parse(JSON.stringify(shell.payload)) as PagePayload }
  page.payload.url = url
  ;(page.payload.props as HomeProps).sessionLost = true
  return page
}
