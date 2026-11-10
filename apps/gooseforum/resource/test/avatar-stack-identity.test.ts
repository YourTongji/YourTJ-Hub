// @vitest-environment happy-dom
import { expect, test, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AvatarStack from '../src/site/components/AvatarStack.vue'

test('mixed participants link to distinct public identities and preserve member cards', async () => {
 const card = vi.fn()
 window.addEventListener('goose:user-card-show', card)
 const users = [
  { id: 0, publicUid: 'persona-one', username: 'one', avatarUrl: '' },
  { id: 0, publicUid: 'persona-two', username: 'two', avatarUrl: '' },
  { id: 42, username: 'member', avatarUrl: '' },
 ]
 const wrapper = mount(AvatarStack, { props: { users }, global: { stubs: { UserAvatar: true } } })
 try {
  expect(wrapper.findAll('a').map(a => a.attributes('href'))).toEqual(['/a/persona-one', '/a/persona-two', '/u/42'])
  await wrapper.findAll('a')[0]!.trigger('click')
  expect(card).not.toHaveBeenCalled()
  await wrapper.findAll('a')[2]!.trigger('click')
  expect(card).toHaveBeenCalledOnce()
  await wrapper.setProps({ users: [users[1]!, users[0]!, users[2]!] })
  expect(wrapper.findAll('a').map(a => a.attributes('href'))).toEqual(['/a/persona-two', '/a/persona-one', '/u/42'])
 } finally { wrapper.unmount(); window.removeEventListener('goose:user-card-show', card) }
})
