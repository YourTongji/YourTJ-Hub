// @vitest-environment happy-dom

import { mount } from '@vue/test-utils'
import { describe, expect, test } from 'vitest'
import WikiSidebarNode from '../src/site/components/WikiSidebarNode.vue'

describe('WikiSidebarNode', () => {
  test('renders nested directories, exposes the active page, and collapses descendants', async () => {
    const wrapper = mount(WikiSidebarNode, {
      props: {
        depth: 0,
        node: {
          kind: 'directory',
          pageId: 0,
          path: 'admission',
          title: 'admission',
          active: false,
          children: [
            {
              kind: 'page',
              pageId: 12,
              path: 'guide/admission/process',
              title: '流程',
              active: true,
              children: [],
            },
          ],
        },
      },
    })

    expect(wrapper.get('button').attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('a').attributes('href')).toBe('/wiki/guide/admission/process')
    expect(wrapper.get('a').classes()).toContain('text-primary')

    await wrapper.get('button').trigger('click')
    expect(wrapper.get('button').attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('a').exists()).toBe(false)
  })
})
