import { reactive, watch, onMounted, onBeforeUnmount } from 'vue'
import { getPrivateNotes, setPrivateNote } from './api'
import type { ViewerPayload, PrivateNote } from '@gooseforum/client'

// Memory only: never enter public user cards, page payloads, drafts or offline caches.
export function createPrivateNoteStore(load = () => getPrivateNotes(), save = (id: number, note: string) => setPrivateNote(id, note)) {
  const state = reactive({ ownerId: 0, loaded: false, loading: false, failed: false, notes: new Map<number, PrivateNote>() })
  let generation = 0
  let request = 0
  function setOwner(ownerId: number) {
    if (state.ownerId === ownerId) return
    generation++
    state.ownerId = ownerId
    state.notes.clear()
    state.loaded = false; state.loading = false; state.failed = false
  }
  async function refresh() {
    if (!state.ownerId) return
    const epoch = generation, seq = ++request, owner = state.ownerId
    state.loading = true; state.failed = false
    try {
      const result = await load()
      if (epoch !== generation || seq !== request) return
      if (result.ownerId !== owner) { setOwner(0); return }
      state.notes = new Map(result.notes.map(note => [note.targetUserId, note]))
      state.loaded = true
    } catch { if (epoch === generation && seq === request) state.failed = true }
    finally { if (epoch === generation && seq === request) state.loading = false }
  }
  async function update(targetUserId: number, username: string, note: string) {
    const owner = state.ownerId, epoch = generation
    if (!owner) throw new Error('Authentication required')
    if (!state.loaded || state.loading || state.failed) throw new Error('Load notes before editing')
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
