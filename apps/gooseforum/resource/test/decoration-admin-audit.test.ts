// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import UserDecorationsManagementPage from '../src/admin/pages/management/UserDecorationsManagementPage.vue'
import { changeUserDecoration, getUserDecorations } from '../src/admin/runtime/api'
import type { DecorationState } from '@gooseforum/client'
vi.mock('../src/admin/runtime/api', () => ({ getUserDecorations: vi.fn(), changeUserDecoration: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('../src/admin/components/global-layout', () => ({ BasicPage: { template: '<div><slot /></div>' } }))
const state: DecorationState = { catalog: [{ code: 'frame-demo', name: 'Frame demo', kind: 'avatarFrame', lightColors: ['#166534'], darkColors: ['#86efac'], isEnabled: true, sortOrder: 1 }], ownedCodes: [], equipped: { avatarFrame: '', nicknameStyle: '' }, audits: [{ id: 1, requestKey: 'old', userId: 7, code: 'frame-demo', action: 'grant', source: 'manual', actorId: 1, reason: 'Original grant', createdAt: '2026-10-13T00:00:00Z' }] }
afterEach(() => vi.clearAllMocks())
it('refreshes audit history after grant; read failure keeps existing log and does not repeat the completed write', async () => {
  vi.mocked(getUserDecorations).mockResolvedValueOnce(structuredClone(state)).mockRejectedValueOnce(new Error('Audit read failed'))
  vi.mocked(changeUserDecoration).mockResolvedValue({ ...state, audits: undefined, ownedCodes: ['frame-demo'] })
  const wrapper = mount(UserDecorationsManagementPage)
  await wrapper.get('input').setValue(7); await wrapper.get('form').trigger('submit'); await flushPromises()
  await wrapper.get('select').setValue('frame-demo'); await wrapper.get('textarea').setValue('Helpful member')
  const grant = wrapper.findAll('button').find(button => button.text() === 'decoration.grant')!
  await grant.trigger('click'); await flushPromises()
  expect(getUserDecorations).toHaveBeenCalledTimes(2)
  expect(changeUserDecoration).toHaveBeenCalledTimes(1)
  expect(wrapper.text()).toContain('Original grant')
  expect(wrapper.get('[role="alert"]').text()).toBe('Audit read failed')
  expect(grant.attributes('disabled')).toBeDefined()
  expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
  wrapper.unmount()
})
