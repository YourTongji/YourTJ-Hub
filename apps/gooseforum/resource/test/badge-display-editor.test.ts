// @vitest-environment happy-dom
import { beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import type { UserBadgePayload } from '@gooseforum/client'
import BadgeDisplayEditor from '../src/site/components/BadgeDisplayEditor.vue'
import { displayBadges } from '../src/runtime/api'
vi.mock('../src/runtime/api', () => ({ displayBadges: vi.fn() }))
const badges = Array.from({length:6}, (_,i) => ({code:`b${i}`,name:`Badge ${i}`,color:'blue',level:'normal',iconType:'image',iconUrl:'/badge.svg',isEnabled:true}) as UserBadgePayload)
const render = (selected = badges.slice(0,2)) => mount(BadgeDisplayEditor,{props:{badges,selected},global:{plugins:[i18n]}})
beforeEach(() => vi.resetAllMocks())
test('persists the selected order independently of wearing a badge',async () => {
 const wrapper=render()
 await wrapper.find('li button:last-child').trigger('click')
 expect(wrapper.findAll('li > span').map(li => li.text())).toEqual(['1. Badge 1','2. Badge 0'])
 await wrapper.findAll('input')[2].setValue(true)
 await wrapper.find('section > button').trigger('click');await flushPromises()
 expect(displayBadges).toHaveBeenCalledWith(['b1','b0','b2'])
})
test('allows explicit empty display and enforces five selections',async () => {
 const wrapper=render(badges.slice(0,5))
 expect(wrapper.findAll('input')[5].attributes('disabled')).toBeDefined()
 for (const input of wrapper.findAll('input').slice(0,5)) await input.setValue(false)
 await wrapper.find('section > button').trigger('click');await flushPromises()
 expect(displayBadges).toHaveBeenCalledWith([])
 expect(wrapper.findAll('input')[5].attributes('disabled')).toBeUndefined()
})
test('failed save retains the selection and allows retry',async () => {
 vi.mocked(displayBadges).mockRejectedValueOnce(new Error('offline'))
 const wrapper=render()
 await wrapper.find('section > button').trigger('click');await flushPromises()
 expect(wrapper.find('[role=alert]').text()).toBe('offline')
 expect(wrapper.findAll('li')).toHaveLength(2)
 await wrapper.find('section > button').trigger('click');await flushPromises()
 expect(displayBadges).toHaveBeenCalledTimes(2)
 expect(wrapper.find('[role=alert]').exists()).toBe(false)
})
