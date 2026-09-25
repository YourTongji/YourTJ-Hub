// @vitest-environment happy-dom
import { beforeEach, expect, it, vi } from 'vitest'
import { DOMWrapper, mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import PrivateNoteEditor from '../src/site/components/PrivateNoteEditor.vue'
import { privateNotes } from '../src/runtime/private-notes'
const mocks = vi.hoisted(() => ({ save: vi.fn(), load: vi.fn() }))
vi.mock('../src/runtime/api', () => ({ setPrivateNote: mocks.save, getPrivateNotes: mocks.load }))
const plugin = () => createI18n({ legacy: false, locale: 'en', messages: { en: { privateNote: { edit: 'Edit', label: 'Note', hint: 'Private' }, common: { save: 'Save', cancel: 'Cancel' }, api: { operationFailed: 'Failed' } } } })
const dialog = () => new DOMWrapper(document.body.querySelector('[role="dialog"]')!)
beforeEach(async () => { vi.clearAllMocks(); privateNotes.setOwner(0); privateNotes.setOwner(1); mocks.save.mockResolvedValue(true); mocks.load.mockResolvedValue({ownerId:1,notes:[]}); await privateNotes.refresh() })
it('edits and clears the note without changing usernames', async () => {
  const wrapper = mount(PrivateNoteEditor, { props: { userId: 2, username: 'alice' }, global: { plugins: [plugin()] } })
  await wrapper.get('[data-testid="private-note-edit"]').trigger('click'); await dialog().get('input').setValue('My note'); await dialog().get('form').trigger('submit'); await flushPromises()
  expect(mocks.save).toHaveBeenCalledWith(2, 'My note'); expect(privateNotes.name(2, 'alice')).toBe('My note(alice)')
  await wrapper.get('[data-testid="private-note-edit"]').trigger('click'); await dialog().get('input').setValue(''); await dialog().get('form').trigger('submit'); await flushPromises()
  expect(privateNotes.name(2, 'alice')).toBe('alice'); wrapper.unmount()
})
it('retains failed input and hides editing for self or signed-out viewers', async () => {
  mocks.save.mockRejectedValueOnce(new Error('offline'))
  const wrapper = mount(PrivateNoteEditor, { props: { userId: 2, username: 'alice' }, global: { plugins: [plugin()] } })
  await wrapper.get('[data-testid="private-note-edit"]').trigger('click'); await dialog().get('input').setValue('retry'); await dialog().get('form').trigger('submit'); await flushPromises()
  expect(dialog().get('input').element.value).toBe('retry'); expect(dialog().get('[role="alert"]').text()).toBe('offline')
  await wrapper.setProps({ userId: 1 }); expect(wrapper.find('[data-testid="private-note-edit"]').exists()).toBe(false)
  privateNotes.setOwner(0); await wrapper.setProps({ userId: 2 }); expect(wrapper.find('[data-testid="private-note-edit"]').exists()).toBe(false); wrapper.unmount()
})
it('waits for loaded notes before editing and supports retry after a failed read', async () => {
  let finish!: (value: unknown) => void
  mocks.load.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
  const pending = privateNotes.refresh()
  const wrapper = mount(PrivateNoteEditor, { props: { userId: 2, username: 'alice' }, global: { plugins: [plugin()] } })
  expect(wrapper.get('[data-testid="private-note-edit"]').attributes('disabled')).toBeDefined()
  await wrapper.get('[data-testid="private-note-edit"]').trigger('click'); expect(document.body.querySelector('[role="dialog"] input')).toBeNull()
  finish({ ownerId: 1, notes: [{ targetUserId: 2, username: 'alice', note: 'Original' }] }); await pending; await flushPromises()
  await wrapper.get('[data-testid="private-note-edit"]').trigger('click'); expect(dialog().get('input').element.value).toBe('Original')
  wrapper.unmount()
  privateNotes.setOwner(0); privateNotes.setOwner(1); mocks.load.mockRejectedValueOnce(new Error('offline')); await privateNotes.refresh()
  const retry = mount(PrivateNoteEditor, { props: { userId: 2, username: 'alice' }, global: { plugins: [plugin()] } })
  expect(document.body.querySelector('[role="dialog"] input')).toBeNull()
  mocks.load.mockResolvedValueOnce({ ownerId: 1, notes: [{ targetUserId: 2, username: 'alice', note: 'Original' }] })
  await retry.get('[data-testid="private-note-retry"]').trigger('click'); await flushPromises()
  await retry.get('[data-testid="private-note-edit"]').trigger('click'); expect(dialog().get('input').element.value).toBe('Original')
  expect(mocks.save).not.toHaveBeenCalled(); retry.unmount()
})
