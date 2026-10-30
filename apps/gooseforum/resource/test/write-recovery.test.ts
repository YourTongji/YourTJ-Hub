// @vitest-environment happy-dom
import { test, expect } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import { useWriteRecovery } from '../src/site/composables/useWriteRecovery'
import WriteRecoveryActions from '../src/site/components/WriteRecoveryActions.vue'

for (const locale of ['zh','en','ja','de'] as const) {
 test(`write recovery is code-based, local and transient in ${locale}`, async () => {
  await setLocale(locale)
  let show!: (value: unknown, message: string) => void
  let replaceMessage!: () => void
  const view = mount(defineComponent({setup(){
   const message=ref('')
   const recovery=useWriteRecovery(message)
   show=(error,text)=>{message.value=text;recovery.capture(error)}
   replaceMessage=()=>{message.value='a different local error'}
   return()=>h(WriteRecoveryActions,{code:recovery.code.value})
  }}),{global:{plugins:[i18n]}})
  show(new Error('permission.emailRequired'),'text only')
  await view.vm.$nextTick(); expect(view.find('a').exists()).toBe(false)
  show({messageCode:'permission.emailRequired'},'email gate')
  await view.vm.$nextTick(); expect(view.get('a').attributes('href')).toBe('/settings?tab=profile')
  expect(view.get('a').text()).toBe(i18n.global.t('auth.writeRecovery.email'))
  replaceMessage();await view.vm.$nextTick();expect(view.find('a').exists()).toBe(false)
  show({messageCode:'auth.required',params:{redirect:'https://evil.test'}},'session gate')
  await view.vm.$nextTick();expect(view.get('a').attributes('href')).toMatch(/^\/login\?redirect=%2F/)
  expect(view.get('a').attributes('href')).not.toContain('evil')
  expect(view.get('a').text()).toBe(i18n.global.t('auth.writeRecovery.login'))
  show({messageCode:'permission.userFrozen'},'frozen gate')
  await view.vm.$nextTick();expect(view.find('a').exists()).toBe(false)
  view.unmount()
 })
}
