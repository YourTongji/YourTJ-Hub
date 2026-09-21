import { describe, expect, it, vi } from 'vitest'
import { createPrivateNoteStore } from '../src/runtime/private-notes'
import type { PrivateNotesPayload } from '@gooseforum/client'

const snapshot = (ownerId = 1, note = '实验搭档'): PrivateNotesPayload => ({ ownerId, notes: [{ targetUserId: 2, username: 'alice', note }] })
const deferred = () => { let resolve!: (value: PrivateNotesPayload) => void; const promise = new Promise<PrivateNotesPayload>(r => { resolve = r }); return { promise, resolve } }
describe('private user display notes', () => {
  it('uses a private note with canonical username without changing public data', async () => {
    const data = snapshot(); const store = createPrivateNoteStore(async () => data)
    expect(store.name(2, 'alice', '昵称')).toBe('昵称')
    store.setOwner(1); await store.refresh()
    expect(store.name(2, 'alice', '昵称')).toBe('实验搭档(alice)')
    expect(store.name(3, 'bob')).toBe('bob')
    expect(store.name(2, 'renamed')).toBe('实验搭档(renamed)')
    expect(store.name(2, '', '昵称')).toBe('实验搭档(alice)')
    expect(data.notes[0]?.note).toBe('实验搭档')
    store.setOwner(0); expect(store.name(2, 'alice', '昵称')).toBe('昵称')
  })
  it('discards old account responses and does not fetch for anonymous viewers', async () => {
    const pending = deferred(); const load = vi.fn(() => pending.promise); const store = createPrivateNoteStore(load)
    await store.refresh(); expect(load).not.toHaveBeenCalled()
    store.setOwner(1); const request = store.refresh(); store.setOwner(3)
    pending.resolve(snapshot()); await request
    expect(store.state.notes.size).toBe(0)
  })
  it('clears state when the cookie belongs to a different owner', async () => {
    const store = createPrivateNoteStore(async () => snapshot(3)); store.setOwner(1)
    await store.refresh(); expect(store.state.ownerId).toBe(0)
  })
  it('protects a saved value from an earlier read and allows clearing', async () => {
    const pending = deferred(); const save = vi.fn(async () => true)
    const store = createPrivateNoteStore(() => pending.promise, save); store.setOwner(1)
    const read = store.refresh(); await store.update(2, 'alice', '新备注')
    pending.resolve(snapshot()); await read
    expect(store.name(2, 'alice')).toBe('新备注(alice)')
    await store.update(2, 'alice', '  '); expect(store.name(2, 'alice')).toBe('alice')
  })
  it('preserves the previous value on failure and rejects an old account write response', async () => {
    let resolve!: (value: boolean) => void
    const store = createPrivateNoteStore(async () => snapshot(), () => new Promise(r => { resolve = r }))
    store.setOwner(1); await store.refresh()
    const write = store.update(2, 'alice', '旧账号'); store.setOwner(3); resolve(true); await write
    expect(store.state.notes.size).toBe(0)
    const failing = createPrivateNoteStore(async () => snapshot(), async () => { throw new Error('offline') })
    failing.setOwner(1); await failing.refresh()
    await expect(failing.update(2, 'alice', 'no')).rejects.toThrow('offline')
    expect(failing.name(2, 'alice')).toBe('实验搭档(alice)')
  })
})
