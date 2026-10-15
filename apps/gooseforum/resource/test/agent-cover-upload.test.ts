// @vitest-environment happy-dom
import { expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import AgentProfileEditor from '../src/site/components/AgentProfileEditor.vue'
import { useCoverCropUpload } from '../src/site/composables/useCoverCropUpload'
import { updateAgent } from '../src/admin/runtime/api'
import { i18n, setLocale } from '../src/runtime/i18n'
import type { UserCardPayload } from '@gooseforum/client'
vi.mock('../src/admin/runtime/api', () => ({ updateAgent: vi.fn() }))
vi.mock('../src/site/composables/useCoverCropUpload', () => ({ useCoverCropUpload: vi.fn() }))
test('cover crop can be confirmed before saving the bot profile', async () => {
  const priorLocale = i18n.global.locale.value
  await setLocale('en')
  const cropOpen = ref(true)
  const upload = vi.fn().mockResolvedValue('/file/img/bot-cover.webp')
  vi.mocked(useCoverCropUpload).mockReturnValue({ coverInput: ref(null), uploadingCover: ref(false), coverCropOpen: cropOpen, coverImageUrl: ref('blob:cover'), coverCropError: ref(''), chooseCover: vi.fn(), handleCoverChange: vi.fn(), closeCoverCrop: () => { cropOpen.value = false }, saveCoverFromCanvas: upload })
  const editor = defineComponent({ setup(_, { expose, emit }) { expose({ save: () => emit('save', document.createElement('canvas')) }); return () => null } })
  const wrapper = mount(AgentProfileEditor, { props: { user: { userId: 9, nickname: 'Bot', avatarUrl: '/static/pic/1.webp' } as UserCardPayload }, global: { plugins: [i18n], stubs: { CoverImageEditor: editor } }, attachTo: document.body })
  try {
    await wrapper.find('button').trigger('click')
    await flushPromises()
    const body = new DOMWrapper(document.body)
    const confirm = body.findAll('button').find(button => button.text() === 'Save')!
    expect(confirm).toBeDefined()
    await confirm.trigger('click')
    await flushPromises()
    expect(upload).toHaveBeenCalledOnce()
    expect(cropOpen.value).toBe(false)
    expect(body.find<HTMLInputElement>('input[name="profileCoverUrl"]').element.validity.valid).toBe(true)
    vi.mocked(updateAgent).mockResolvedValue({ agentId: 9 } as never)
    await body.find('form').trigger('submit')
    await flushPromises()
    expect(updateAgent).toHaveBeenCalledWith(expect.objectContaining({ agentId: 9, profileCoverUrl: '/file/img/bot-cover.webp' }))
  } finally { wrapper.unmount(); document.body.innerHTML = ''; i18n.global.locale.value = priorLocale }
})
