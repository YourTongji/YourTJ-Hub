import { reactive, watch, onMounted, onBeforeUnmount } from 'vue'
import { getPrivateNotes, setPrivateNote } from './api'
import type { ViewerPayload, PrivateNote } from '@gooseforum/client'

// Memory only: never enter public user cards, page payloads, drafts or offline caches.
export function createPrivateNoteStore(load = () => getPrivateNotes(), save = (id: number, note: string) => setPrivateNote(id, note)) {
  const state = reactive({ ownerId: 0, notes: new Map<number, PrivateNote>() })
  let generation = 0
  let request = 0
  function setOwner(ownerId: number) {
    if (state.ownerId === ownerId) return
    generation++
    state.ownerId = ownerId
    state.notes.clear()
  }
  async function refresh() {
    if (!state.ownerId) return
    const epoch = generation, seq = ++request, owner = state.ownerId
    try {
      const result = await load()
      if (epoch !== generation || seq !== request) return
      if (result.ownerId !== owner) { setOwner(0); return }
      state.notes = new Map(result.notes.map(note => [note.targetUserId, note]))
    } catch { /* Notes never block reading. Explicit saves still report failures. */ }
  }
  async function update(targetUserId: number, username: string, note: string) {
    const owner = state.ownerId, epoch = generation
    if (!owner) throw new Error('Authentication required')
    await save(targetUserId, note)
    if (epoch !== generation || state.ownerId !== owner) return
    ++request // A read started before this write cannot overwrite the new value.
    const normalized = note.trim()
    if (normalized) state.notes.set(targetUserId, { targetUserId, username, note: normalized })
    else state.notes.delete(targetUserId)
  }
  function name(id: number | undefined, username: string, nickname?: string | null) {
    const item = state.notes.get(id ?? 0)
    return item ? `${item.note}(${username || item.username})` : nickname || username
  }
  return { state, setOwner, refresh, update, name }
}
export const privateNotes = createPrivateNoteStore()
export const userDisplayName = privateNotes.name

export function usePrivateNotesSession(viewer: () => ViewerPayload) {
  watch(() => [viewer().id, viewer().isAuthenticated] as const, ([id, authenticated]) => {
    privateNotes.setOwner(authenticated ? id : 0)
    void privateNotes.refresh()
  }, { immediate: true })
  const clear = () => privateNotes.setOwner(0)
  const refresh = () => { void privateNotes.refresh() }
  onMounted(() => { window.addEventListener('focus', refresh); window.addEventListener('goose:session-cleared', clear) })
  onBeforeUnmount(() => { window.removeEventListener('focus', refresh); window.removeEventListener('goose:session-cleared', clear); clear() })
}
