import { onActivated, onBeforeUnmount, onDeactivated, onMounted } from 'vue'

// The same-origin session cookie authenticates this owner-only stream. REST
// reconciles on hello/reconnect; no private body is carried in an event.
export function useContentUpdates(refresh: () => void, enabled: () => boolean = () => true) {
  let stream: EventSource | undefined
  let active = false
  let queued = false
  const reconcile = () => {
    if (!active || document.hidden) { queued = true; return }
    queued = false
    refresh()
  }
  const visibility = () => { if (!document.hidden && queued) reconcile() }
  const start = () => {
    if (active || !enabled() || typeof EventSource === 'undefined') return
    active = true
    stream = new EventSource('/api/forum/events', { withCredentials: true })
    stream.addEventListener('content.changed', reconcile)
    stream.addEventListener('hello', reconcile)
    stream.addEventListener('session.invalidated', stop)
    document.addEventListener('visibilitychange', visibility)
  }
  const stop = () => { active = false; stream?.close(); stream = undefined; document.removeEventListener('visibilitychange', visibility) }
  onMounted(start)
  onActivated(start)
  onDeactivated(stop)
  onBeforeUnmount(stop)
}
