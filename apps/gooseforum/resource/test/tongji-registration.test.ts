// @vitest-environment happy-dom
import { beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import TongjiRegistration from '../src/site/components/TongjiRegistration.vue'
import { ApiResponseError, getTongjiRegistration, completeTongjiRegistration } from '../src/runtime/api'
vi.mock('../src/runtime/api', () => ({ ApiResponseError: class extends Error { constructor(message: string, public messageCode?: string) { super(message) } }, getTongjiRegistration: vi.fn(), completeTongjiRegistration: vi.fn() }))
beforeEach(() => {
 vi.resetAllMocks()
 vi.mocked(getTongjiRegistration).mockResolvedValue({ csrfToken: 'proof', email:'1234567@tongji.edu.cn', expiresAt:'2030-01-01T00:00:00Z' })
})
const render = (terms = false) => mount(TongjiRegistration,{props:{terms},global:{plugins:[i18n]}})
async function fill(wrapper: ReturnType<typeof render>, confirmation = 'Password123') {
 await flushPromises()
 await wrapper.find('input[autocomplete=username]').setValue('chosen_student')
 const passwords = wrapper.findAll('input[autocomplete=new-password]')
 await passwords[0].setValue('Password123')
 await passwords[1].setValue(confirmation)
}
test('school proof alone never creates an account; username starts empty',async () => {
 const wrapper=render(); await flushPromises()
 expect((wrapper.find('input[autocomplete=username]').element as HTMLInputElement).value).toBe('')
 expect(wrapper.text()).toContain('1234567@tongji.edu.cn')
 expect(completeTongjiRegistration).not.toHaveBeenCalled()
 expect(wrapper.findAll('input[autocomplete=new-password]')).toHaveLength(2)
})
test('requires confirmation and published terms; sends only chosen fields and CSRF',async () => {
 const wrapper=render(true); await fill(wrapper,'different')
 await wrapper.find('form').trigger('submit'); expect(completeTongjiRegistration).not.toHaveBeenCalled()
 await wrapper.findAll('input[autocomplete=new-password]')[1].setValue('Password123')
 await wrapper.find('form').trigger('submit'); expect(completeTongjiRegistration).not.toHaveBeenCalled()
 await wrapper.find('input[type=checkbox]').setValue(true)
 vi.mocked(completeTongjiRegistration).mockRejectedValueOnce(new Error('Username unavailable'))
 await wrapper.find('form').trigger('submit'); await flushPromises()
 expect(completeTongjiRegistration).toHaveBeenCalledWith('chosen_student','Password123','proof')
 expect(wrapper.find('[role=alert]').text()).toBe('Username unavailable')
 expect((wrapper.find('input[autocomplete=username]').element as HTMLInputElement).value).toBe('chosen_student')
 expect(wrapper.find('button[type=submit]').attributes('disabled')).toBeUndefined()
})
test('expired proof offers a fresh login without a usable registration form',async () => {
 vi.mocked(getTongjiRegistration).mockRejectedValue(new Error('expired'))
 const wrapper=render(); await flushPromises()
 expect(wrapper.find('form').exists()).toBe(false)
 expect(wrapper.find('a').attributes('href')).toBe('/login')
 expect(wrapper.find('[role=alert]').exists()).toBe(true)
})

test('proof expiring during input gives a fresh login action',async () => {
 const wrapper=render(); await fill(wrapper)
 vi.mocked(completeTongjiRegistration).mockRejectedValueOnce(new ApiResponseError('expired','auth.required'))
 await wrapper.find('form').trigger('submit'); await flushPromises()
 expect(wrapper.find('form').exists()).toBe(false)
 expect(wrapper.find('a').attributes('href')).toBe('/login')
})
