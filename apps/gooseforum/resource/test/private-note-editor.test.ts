// @vitest-environment happy-dom
import { beforeEach, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import PrivateNoteEditor from '../src/site/components/PrivateNoteEditor.vue'
import { privateNotes } from '../src/runtime/private-notes'
const mocks = vi.hoisted(() => ({ save: vi.fn(), load: vi.fn() }))
vi.mock('../src/runtime/api', () => ({ setPrivateNote: mocks.save, getPrivateNotes: mocks.load }))
const plugin = () => createI18n({ legacy: false, locale: 'en', messages: { en: { privateNote: { edit: 'Edit', label: 'Note', hint: 'Private' }, common: { save: 'Save', cancel: 'Cancel' }, api: { operationFailed: 'Failed' } } } })
beforeEach(async () => { vi.clearAllMocks(); privateNotes.setOwner(0); privateNotes.setOwner(1); mocks.save.mockResolvedValue(true); mocks.load.mockResolvedValue({ownerId:1,notes:[]}); await privateNotes.refresh() })
it('edits and clears the note without changing usernames', async () => {
  const wrapper = mount(PrivateNoteEditor, { props: { userId: 2, username: 'alice' }, global: { plugins: [plugin()] } })
  await wrapper.get('button').trigger('click'); await wrapper.get('input').setValue('My note'); await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(mocks.save).toHaveBeenCalledWith(2, 'My note'); expect(privateNotes.name(2, 'alice')).toBe('My note(alice)')
  await wrapper.get('button').trigger('click'); await wrapper.get('input').setValue(''); await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(privateNotes.name(2, 'alice')).toBe('alice'); wrapper.unmount()
})
it('retains failed input and hides editing for self or signed-out viewers', async () => {
  mocks.save.mockRejectedValueOnce(new Error('offline'))
  const wrapper = mount(PrivateNoteEditor, { props: { userId: 2, username: 'alice' }, global: { plugins: [plugin()] } })
  await wrapper.get('button').trigger('click'); await wrapper.get('input').setValue('retry'); await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(wrapper.get('input').element.value).toBe('retry'); expect(wrapper.get('[role="alert"]').text()).toBe('offline')
  await wrapper.setProps({ userId: 1 }); expect(wrapper.find('button').exists()).toBe(false)
  privateNotes.setOwner(0); await wrapper.setProps({ userId: 2 }); expect(wrapper.find('button').exists()).toBe(false); wrapper.unmount()
})
it('waits for loaded notes before editing and supports retry after a failed read', async () => {
  let finish!: (value: unknown) => void
  mocks.load.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
  const pending = privateNotes.refresh()
  const wrapper = mount(PrivateNoteEditor, { props: { userId: 2, username: 'alice' }, global: { plugins: [plugin()] } })
  expect(wrapper.get('button').attributes('disabled')).toBeDefined()
  await wrapper.get('button').trigger('click'); expect(wrapper.find('input').exists()).toBe(false)
  finish({ ownerId: 1, notes: [{ targetUserId: 2, username: 'alice', note: 'Original' }] }); await pending; await flushPromises()
  await wrapper.get('button').trigger('click'); expect(wrapper.get('input').element.value).toBe('Original')
  wrapper.unmount()
  privateNotes.setOwner(0); privateNotes.setOwner(1); mocks.load.mockRejectedValueOnce(new Error('offline')); await privateNotes.refresh()
  const retry = mount(PrivateNoteEditor, { props: { userId: 2, username: 'alice' }, global: { plugins: [plugin()] } })
  expect(retry.find('input').exists()).toBe(false)
  mocks.load.mockResolvedValueOnce({ ownerId: 1, notes: [{ targetUserId: 2, username: 'alice', note: 'Original' }] })
  await retry.get('[data-testid="private-note-retry"]').trigger('click'); await flushPromises()
  await retry.get('button').trigger('click'); expect(retry.get('input').element.value).toBe('Original')
  expect(mocks.save).not.toHaveBeenCalled(); retry.unmount()
})
