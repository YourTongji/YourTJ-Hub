// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ManagedContent from '../src/site/components/ManagedContent.vue'
import { useQuickPublish } from '../src/site/composables/useQuickPublish'
import { i18n } from '../src/runtime/i18n'
import { fetchPage } from '../src/runtime/router'
vi.mock('../src/runtime/router', () => ({ fetchPage: vi.fn() }))
afterEach(() => { useQuickPublish().closeQuickPublish(); vi.clearAllMocks() })
for (const contentType of [1, 2]) {
 test(`content management opens short editor for type ${contentType} using latest candidate`, async () => {
  const topic = { id: 42, contentType, title: '新版标题', content: '新版正文', categoryIds: [3], images: ['/file/candidate.png'] }
  vi.mocked(fetchPage).mockResolvedValue({ component: 'publish.index', props: { topic } } as any)
  const wrapper = mount(ManagedContent, { props: { item: { id: 42, contentType: 'topic', title: '旧版', content: '旧文', processStatus: 1 } as any }, global: { plugins: [i18n] } })
  const edit = wrapper.findAll('a,button').find(el => el.text() === i18n.global.t('contentReview.retry'))!
  await edit.trigger('click')
  await flushPromises()
  expect(useQuickPublish().quickPublishOpen.value).toBe(true)
  expect(useQuickPublish().quickPublishEditPayload.value).toEqual({ topicId: 42, contentType, title: topic.title, content: topic.content, categoryIds: [3], images: topic.images })
  wrapper.unmount()
 })
}
