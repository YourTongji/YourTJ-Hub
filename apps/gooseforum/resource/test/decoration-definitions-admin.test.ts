// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DecorationsManagementPage from '../src/admin/pages/management/DecorationsManagementPage.vue'
import { getDecorations, saveDecoration } from '../src/admin/runtime/api'
import type { DecorationStyle } from '@gooseforum/client'

vi.mock('../src/admin/runtime/api', () => ({ getDecorations: vi.fn(), saveDecoration: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('../src/admin/runtime/i18n-text', () => ({ adminText: (key: string) => key }))
vi.mock('../src/admin/components/global-layout', () => ({ BasicPage: { template: '<div><slot name="actions" /><slot /></div>' } }))
vi.mock('../src/admin/components/ui/dialog', () => ({
  Dialog: { props: ['open'], template: '<div v-if="open"><slot /></div>' },
  DialogContent: { template: '<div><slot /></div>' },
  DialogHeader: { template: '<div><slot /></div>' },
  DialogTitle: { template: '<h2><slot /></h2>' },
  DialogDescription: { template: '<p><slot /></p>' },
  DialogFooter: { template: '<div><slot /></div>' },
}))
vi.mock('../src/admin/components/AdminSelect.vue', () => ({ default: {
  props: ['modelValue', 'options'], emits: ['update:modelValue'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :value="option.value">{{ option.label }}</option></select>',
} }))
const style: DecorationStyle = { code: 'tea-garden', name: 'Tea garden', kind: 'avatarFrame', lightColors: ['#166534'], darkColors: ['#86EFAC'], isEnabled: true, sortOrder: 100 }
afterEach(() => vi.resetAllMocks())

it('uses the confirmed save response without risking a second list read', async () => {
  vi.mocked(getDecorations).mockResolvedValueOnce([]).mockRejectedValueOnce(new Error('List unavailable'))
  vi.mocked(saveDecoration).mockResolvedValue(style)
  const wrapper = mount(DecorationsManagementPage, { global: { stubs: { UserAvatar: true, UserNickname: true } } })
  await flushPromises()
  await wrapper.findAll('button').find(button => button.text() === 'decoration.new')!.trigger('click')
  await wrapper.get('#decoration-code').setValue(style.code)
  await wrapper.get('#decoration-name').setValue(style.name)
  await wrapper.get('form').trigger('submit')
  await flushPromises()
  expect(saveDecoration).toHaveBeenCalledTimes(1)
  expect(wrapper.text()).toContain('Tea garden')
  expect(wrapper.text()).toContain('decoration.saved')
  expect(getDecorations).toHaveBeenCalledTimes(1)
  wrapper.unmount()
})

async function editor(items: DecorationStyle[] = []) {
  vi.mocked(getDecorations).mockResolvedValue(items)
  const wrapper = mount(DecorationsManagementPage, { global: { stubs: { UserAvatar: true, UserNickname: true } } })
  await flushPromises()
  const text = items.length ? 'decoration.configAdmin.edit' : 'decoration.new'
  await wrapper.findAll('button').find(button => button.text() === text)!.trigger('click')
  return wrapper
}
it('does not mutate saved palettes, locks identity, and retains failed drafts for retry', async () => {
  const original = structuredClone(style)
  const wrapper = await editor([original])
  expect(wrapper.get('#decoration-code').attributes('disabled')).toBeDefined()
  expect(wrapper.get('#decoration-kind').attributes('disabled')).toBeDefined()
  await wrapper.get('#decoration-lightColors-0').setValue('#115E59')
  await wrapper.get('#decoration-name').setValue('Updated garden')
  expect(original.lightColors).toEqual(['#166534'])
  vi.mocked(saveDecoration).mockRejectedValueOnce(new Error('Save unavailable')).mockResolvedValueOnce({ ...style, name: 'Updated garden' })
  await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(wrapper.get('[role="alert"]').text()).toBe('Save unavailable')
  expect((wrapper.get('#decoration-name').element as HTMLInputElement).value).toBe('Updated garden')
  await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(saveDecoration).toHaveBeenCalledTimes(2)
  expect(wrapper.text()).toContain('Updated garden')
  wrapper.unmount()
})
it('rejects unsafe artwork and unreadable nickname colors, clearing artwork on kind change', async () => {
  const wrapper = await editor()
  await wrapper.get('#decoration-code').setValue(style.code)
  await wrapper.get('#decoration-name').setValue(style.name)
  await wrapper.get('#decoration-asset').setValue('https://external.invalid/frame.png')
  await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(saveDecoration).not.toHaveBeenCalled()
  expect(wrapper.text()).toContain('decoration.configAdmin.assetInvalid')
  await wrapper.get('#decoration-kind').setValue('nicknameStyle')
  await wrapper.get('#decoration-lightColors-0').setValue('#FFFFFF')
  await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(wrapper.text()).toContain('decoration.configAdmin.contrastInvalid')
  expect(saveDecoration).not.toHaveBeenCalled()
  await wrapper.get('#decoration-lightColors-0').setValue('#166534')
  await wrapper.get('#decoration-lightColors-1').setValue('#115E59')
  vi.mocked(saveDecoration).mockResolvedValue({ ...style, kind: 'nicknameStyle' })
  await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(saveDecoration).toHaveBeenCalledWith(expect.objectContaining({ kind: 'nicknameStyle', assetUrl: '' }))
  wrapper.unmount()
})
it('retains catalog rows and offers retry when a refresh fails', async () => {
  vi.mocked(getDecorations).mockResolvedValueOnce([style]).mockRejectedValueOnce(new Error('Refresh unavailable')).mockResolvedValueOnce([style])
  const wrapper = mount(DecorationsManagementPage, { global: { stubs: { UserAvatar: true, UserNickname: true } } })
  await flushPromises()
  await wrapper.findAll('button').find(button => button.text() === 'decoration.configAdmin.refresh')!.trigger('click'); await flushPromises()
  expect(wrapper.text()).toContain('Refresh unavailable')
  await wrapper.get('input').setValue('tea')
  await wrapper.findAll('button').find(button => button.text() === 'k002w')!.trigger('click'); await flushPromises()
  expect(wrapper.text()).toContain('Tea garden')
  wrapper.unmount()
})
it('previews and saves artwork scale, retaining it when editing an existing frame', async () => {
  const scaled = { ...style, assetUrl: '/static/decorations/tea-garden-v1.png', assetScale: 1.4 }
  const wrapper = await editor([scaled])
  expect((wrapper.get('#decoration-scale').element as HTMLInputElement).value).toBe('140')
  await wrapper.get('#decoration-scale').setValue(110)
  vi.mocked(saveDecoration).mockResolvedValue({ ...scaled, assetScale: 1.1 })
  await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(saveDecoration).toHaveBeenCalledWith(expect.objectContaining({ assetScale: 1.1 }))
  wrapper.unmount()
})

it('clears invalid hidden artwork scale when switching to a nickname style', async () => {
  const wrapper = await editor()
  await wrapper.get('#decoration-code').setValue('nickname-owned')
  await wrapper.get('#decoration-name').setValue('Nickname')
  const percent = wrapper.findAll('input[type="number"]').find(input => input.attributes('aria-label') === 'decoration.configAdmin.scale')!
  await percent.setValue('')
  await wrapper.get('#decoration-kind').setValue('nicknameStyle')
  await wrapper.get('#decoration-lightColors-1').setValue('#115E59')
  vi.mocked(saveDecoration).mockResolvedValue({ ...style, kind: 'nicknameStyle' })
  await wrapper.get('form').trigger('submit'); await flushPromises()
  expect(saveDecoration).toHaveBeenCalledWith(expect.objectContaining({ kind: 'nicknameStyle', assetScale: 1.25 }))
  wrapper.unmount()
})
