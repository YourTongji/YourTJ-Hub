import { createApp, h, ref } from 'vue'
import AgentReplySetting from '../../../src/site/components/AgentReplySetting.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
await setLocale('zh')
createApp({ setup() {
  const disabled = ref(false)
  return () => h('main', { class: 'mx-auto max-w-2xl p-4' }, [
    h('section', { id: 'author' }, [h('h1', '作者设置'), h(AgentReplySetting, {
      topicId: 42, canManage: true, disabled: disabled.value,
      onChanged: (value: boolean) => { disabled.value = value },
    })]),
    h('section', { id: 'reader' }, [h('h2', '访客看到的状态'), h(AgentReplySetting, {
      topicId: 42, canManage: false, disabled: disabled.value,
    })]),
  ])
}}).use(i18n).mount('#app')
