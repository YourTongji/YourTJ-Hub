// @vitest-environment happy-dom
import { expect, test } from 'vitest'
import { mount } from '@vue/test-utils'
import type { TopicPayload } from '@gooseforum/client'
import TopicList from '../src/site/components/TopicList.vue'
import { i18n } from '../src/runtime/i18n'

function moment(id: number, description: string): TopicPayload {
  return {
    id, title: '', description, url: `/p/${id}`, pinWeight: 1,
    processStatus: 0, contentType: 2, author: { id: 1, username: 'alice', avatarUrl: '' },
    participants: [], categories: [], replyCount: 0, viewCount: 0,
    activityText: '', lastUpdateTime: '',
  }
}

for (const feedMode of ['table', 'card'] as const) {
  test(`untitled pinned moments retain readable links in ${feedMode} mode`, async () => {
    const wrapper = mount(TopicList, {
      props: {
        topics: [moment(42, '校园里的新发现'), moment(43, '')],
        home: true, showPinned: true, feedMode,
      },
      global: { plugins: [i18n] },
    })
    try {
      const first = wrapper.get('a[href="/p/42"]')
      expect(first.text()).toBe('校园里的新发现')
      expect(first.attributes('title')).toBe('校园里的新发现')
      await wrapper.get('button[aria-expanded]').trigger('click')
      const second = wrapper.get('a[href="/p/43"]')
      const label = second.get('span:last-child').text()
      expect(label).not.toBe('')
      expect(second.attributes('title')).toBe(label)
    } finally {
      wrapper.unmount()
    }
  })
}
