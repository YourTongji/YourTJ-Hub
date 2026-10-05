// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent } from 'vue'
import { i18n } from '../src/runtime/i18n'
import PublishPage from '../src/site/pages/PublishPage.vue'

const submit = vi.hoisted(() => vi.fn())
vi.mock('@/runtime/api', async importOriginal => ({
  ...await importOriginal<typeof import('../src/runtime/api')>(),
  submitTopicResult: submit,
}))
vi.mock('@/site/components/VditorOfficial.vue', () => ({
  default: defineComponent({
    props: { modelValue: String }, emits: ['update:modelValue', 'input'],
    setup(props, { expose }) { expose({ syncValue: () => props.modelValue, getMentionContext: () => null }) },
    template: `<textarea data-test="editor" :value="modelValue" @input="$emit('update:modelValue', $event.target.value); $emit('input')" />`,
  }),
}))
let wrapper: VueWrapper | undefined
async function mountPage(contentType = 3) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
  await router.push('/')
  await router.isReady()
  submit.mockReset().mockRejectedValue(new Error('save failed'))
  wrapper = mount(PublishPage, {
    props: {
      layout: { viewer: { id: 7, isAuthenticated: true }, posting: {} } as any,
      props: { topicId: 17, isEditing: true, categories: [{ id: 1, name: '分类', color: '#666' }], topic: {
        title: '被拒后修改', content: '正文 ![旧图](/file/img/old.png)', categoryIds: [1], contentType,
        images: ['/file/img/old.png', '/file/img/gallery.png'],
      } } as any,
    },
    global: { plugins: [i18n, router], stubs: { PageHeader: true, MentionCandidates: true, StickerPicker: true } },
  })
  await flushPromises()
  return wrapper
}
afterEach(() => { wrapper?.unmount(); vi.restoreAllMocks() })

describe('发布页编辑图片', () => {
  test('内容管理的瞬间编辑使用服务端类型并保持瞬间提交', async () => {
    const view = await mountPage(2)
    expect((view.vm as any).contentType).toBe(2)
    await (view.vm as any).save()
    expect(submit).toHaveBeenCalledWith(expect.objectContaining({ topicId: 17, contentType: 2 }))
  })

  test.each(['save', 'saveDraft'])('%s 不带回已从正文删除的图片，保留独立图库和新正文图片', async action => {
    const view = await mountPage()
    await view.get('[data-test="editor"]').setValue('修改后正文 ![新图](/file/img/new.png)')
    await (view.vm as any)[action]()
    expect(submit).toHaveBeenCalledWith(expect.objectContaining({
      content: '修改后正文 ![新图](/file/img/new.png)',
      images: ['/file/img/gallery.png', '/file/img/new.png'],
      topicStatus: action === 'save' ? 1 : 0,
    }))
    expect((view.get('[data-test="editor"]').element as HTMLTextAreaElement).value).toContain('修改后正文')
  })

  test('额外图库可删除，删空后保存草稿提交空数组', async () => {
    const view = await mountPage()
    expect(view.findAll('[data-test="gallery-image"]')).toHaveLength(1)
    await view.get(`[aria-label="${i18n.global.t('publish.modal.deleteImage')}"]`).trigger('click')
    await view.get('[data-test="editor"]').setValue('已经删除所有图片的正文')
    await (view.vm as any).saveDraft()
    expect(submit).toHaveBeenCalledWith(expect.objectContaining({ images: [], topicStatus: 0 }))
  })
})
