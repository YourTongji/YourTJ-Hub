// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import UserDecorationsManagementPage from '../src/admin/pages/management/UserDecorationsManagementPage.vue'
import { changeUserDecoration, getUserDecorations } from '../src/admin/runtime/api'
import type { DecorationState } from '@gooseforum/client'
vi.mock('../src/admin/runtime/api', () => ({ getUserDecorations: vi.fn(), changeUserDecoration: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key, locale: { value: 'en' } }) }))
vi.mock('../src/admin/runtime/i18n-text', () => ({ adminText: (key: string) => key }))
vi.mock('../src/admin/components/global-layout', () => ({ BasicPage: { template: '<div><slot /></div>' } }))
vi.mock('../src/admin/components/AdminSelect.vue', () => ({ default: {
  props: ['modelValue', 'options', 'disabled'], emits: ['update:modelValue'],
  template: '<select :value="modelValue" :disabled="disabled" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
} }))
vi.mock('../src/admin/components/AdminConfirmDialog.vue', () => ({ default: {
  props: ['open', 'loading', 'description'], emits: ['confirm', 'update:open'],
  template: '<div v-if="open" role="dialog">{{ description }}<button data-testid="confirm-revoke" :disabled="loading" @click="$emit(\'confirm\')">Confirm</button><button data-testid="cancel-revoke" @click="$emit(\'update:open\', false)">Cancel</button></div>',
} }))
vi.mock('../src/admin/components/DecorationStylePreview.vue', () => ({ default: {
  props: ['style'], template: '<div data-testid="style-preview">{{ style.name }}</div>',
} }))
const state: DecorationState = { catalog: [{ code: 'frame-demo', name: 'Frame demo', kind: 'avatarFrame', lightColors: ['#166534'], darkColors: ['#86efac'], isEnabled: true, sortOrder: 1 }], ownedCodes: [], equipped: { avatarFrame: '', nicknameStyle: '' }, audits: [{ id: 1, requestKey: 'old', userId: 7, code: 'frame-demo', action: 'grant', source: 'manual', actorId: 1, reason: 'Original grant', createdAt: '2026-10-13T00:00:00Z' }] }
afterEach(() => vi.resetAllMocks())
it('does not change the previously loaded account after entering a different user ID', async () => {
  vi.mocked(getUserDecorations).mockResolvedValue(structuredClone(state))
  const wrapper = mount(UserDecorationsManagementPage)
  await wrapper.get('input').setValue(7); await wrapper.get('form').trigger('submit'); await flushPromises()
  await wrapper.get('select').setValue('frame-demo'); await wrapper.get('textarea').setValue('Helpful member')
  await wrapper.get('input').setValue(8)
  const grant = wrapper.findAll('button').find(button => button.text() === 'decoration.grant')!
  expect(grant.attributes('disabled')).toBeDefined()
  await grant.trigger('click'); await flushPromises()
  expect(changeUserDecoration).not.toHaveBeenCalled()
  wrapper.unmount()
})
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
  expect(wrapper.get('[role="alert"]').text()).toContain('Audit read failed')
  expect(wrapper.get('[role="alert"]').text()).toContain('decoration.grantsAdmin.refreshFailed')
  expect(wrapper.get('[role="status"]').text()).toContain('decoration.grantsAdmin.grantSuccess')
  expect(grant.attributes('disabled')).toBeDefined()
  expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
  wrapper.unmount()
})

it('keeps the request key when retrying an ambiguous write failure, including after a read refresh', async () => {
  vi.mocked(getUserDecorations).mockResolvedValue(structuredClone(state))
  vi.mocked(changeUserDecoration).mockRejectedValueOnce(new Error('Network interrupted')).mockResolvedValueOnce({ ...state, ownedCodes: ['frame-demo'] })
  const wrapper = mount(UserDecorationsManagementPage)
  await wrapper.get('input').setValue(7); await wrapper.get('form').trigger('submit'); await flushPromises()
  await wrapper.get('select').setValue('frame-demo'); await wrapper.get('textarea').setValue(' Helpful member ')
  await wrapper.get('[data-testid="grant"]').trigger('click'); await flushPromises()
  const firstBody = vi.mocked(changeUserDecoration).mock.calls[0][1]
  expect(firstBody.reason).toBe('Helpful member')
  expect(firstBody.requestKey).toBeTruthy()
  expect(wrapper.get('[role="alert"]').text()).toContain('Network interrupted')
  expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe(' Helpful member ')
  await wrapper.get('form').trigger('submit'); await flushPromises()
  await wrapper.get('[data-testid="grant"]').trigger('click'); await flushPromises()
  expect(changeUserDecoration).toHaveBeenCalledTimes(2)
  expect(vi.mocked(changeUserDecoration).mock.calls[1][1]).toEqual(firstBody)
  wrapper.unmount()
})

it('confirms revocation before writing and shows the updated inventory and equipment', async () => {
  const ownedState = { ...structuredClone(state), ownedCodes: ['frame-demo'], equipped: { avatarFrame: 'frame-demo', nicknameStyle: '' } }
  vi.mocked(getUserDecorations).mockResolvedValueOnce(ownedState).mockResolvedValueOnce(structuredClone(state))
  vi.mocked(changeUserDecoration).mockResolvedValue(structuredClone(state))
  const wrapper = mount(UserDecorationsManagementPage)
  await wrapper.get('input').setValue(7); await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(wrapper.text()).toContain('decoration.grantsAdmin.equipped')
  await wrapper.get('select').setValue('frame-demo'); await wrapper.get('textarea').setValue('Accidental grant')
  expect(wrapper.get('[data-testid="style-preview"]').text()).toBe('Frame demo')
  expect(wrapper.get('[data-testid="grant"]').attributes('disabled')).toBeDefined()
  await wrapper.get('[data-testid="revoke"]').trigger('click'); await flushPromises()
  expect(wrapper.find('[role="dialog"]').exists()).toBe(true)
  expect(changeUserDecoration).not.toHaveBeenCalled()
  await wrapper.get('[data-testid="cancel-revoke"]').trigger('click')
  expect(changeUserDecoration).not.toHaveBeenCalled()
  await wrapper.get('[data-testid="revoke"]').trigger('click')
  await wrapper.get('[data-testid="confirm-revoke"]').trigger('click'); await flushPromises()
  expect(changeUserDecoration).toHaveBeenCalledWith('revoke', expect.objectContaining({ userId: 7, code: 'frame-demo', reason: 'Accidental grant' }))
  expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  expect(wrapper.text()).toContain('decoration.grantsAdmin.inventoryEmpty')
  expect(wrapper.text()).not.toContain('decoration.grantsAdmin.equipped')
  expect(wrapper.get('[role="status"]').text()).toContain('decoration.grantsAdmin.revokeSuccess')
  wrapper.unmount()
})

it('shows readable audit actions and dates and blocks granting a disabled decoration', async () => {
  vi.mocked(getUserDecorations).mockResolvedValue({ ...structuredClone(state), catalog: [{ ...state.catalog[0], isEnabled: false }] })
  const wrapper = mount(UserDecorationsManagementPage)
  await wrapper.get('input').setValue(7); await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(wrapper.get('time').text()).not.toBe('2026-10-13T00:00:00Z')
  expect(wrapper.get('time').attributes('datetime')).toBe('2026-10-13T00:00:00Z')
  expect(wrapper.get('tbody').text()).toContain('decoration.grant')
  expect(wrapper.get('tbody').text()).toContain('decoration.grantsAdmin.manualSource')
  await wrapper.get('select').setValue('frame-demo'); await wrapper.get('textarea').setValue('Helpful member')
  expect(wrapper.text()).toContain('decoration.grantsAdmin.disabledHint')
  expect(wrapper.get('[data-testid="grant"]').attributes('disabled')).toBeDefined()
  wrapper.unmount()
})

it('shows empty and failed account states with a read-only retry', async () => {
  vi.mocked(getUserDecorations).mockRejectedValueOnce(new Error('Account not found')).mockResolvedValueOnce({ ...state, catalog: [], audits: [] })
  const wrapper = mount(UserDecorationsManagementPage)
  expect(wrapper.text()).toContain('decoration.grantsAdmin.noAccount')
  await wrapper.get('input').setValue(7); await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(wrapper.get('[role="alert"]').text()).toContain('Account not found')
  await wrapper.findAll('button').find(button => button.text() === 'decoration.retry')!.trigger('click'); await flushPromises()
  expect(wrapper.text()).toContain('decoration.grantsAdmin.catalogEmpty')
  expect(wrapper.text()).toContain('decoration.grantsAdmin.auditEmpty')
  expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  expect(changeUserDecoration).not.toHaveBeenCalled()
  wrapper.unmount()
})
