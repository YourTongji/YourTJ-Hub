// @vitest-environment happy-dom
import { expect, test } from 'vitest'
import { mount } from '@vue/test-utils'
import UserAvatar from '../src/site/components/UserAvatar.vue'

test('comment-sized persona avatars keep the public SVG URL', () => {
  for (const src of ['/a/opaque-uid/avatar.svg', 'https://forum.example/a/opaque-uid/avatar.svg?v=1']) {
    const wrapper = mount(UserAvatar, { props: { src, alt: '躲进云里的猫' } })
    expect(wrapper.get('img').attributes('src')).toBe(src)
  }
})

test('member raster avatars still use the medium variant', () => {
  for (const [src, expected] of [
    ['/file/avatar/opaque/avatar.webp', '/file/avatar/opaque/avatar_medium.webp'],
    ['/static/pic/default-avatar.webp', '/static/pic/default-avatar_medium.webp'],
  ]) {
    const wrapper = mount(UserAvatar, { props: { src, alt: 'member' } })
    expect(wrapper.get('img').attributes('src')).toBe(expected)
  }
})
