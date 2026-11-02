import { h, ref } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import VditorOfficial from '../../../src/site/components/VditorOfficial.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

await setLocale('zh')

const value = ref(new URLSearchParams(location.search).get('value') ?? '')
const editor = ref<{ insertMarkdown: (markdown: string) => void } | null>(null)

;(window as unknown as { testSetValue: (next: string) => void }).testSetValue = (next) => {
  value.value = next
}

createApp({
  render: () => h('div', { class: 'p-4' }, [
    h(VditorOfficial, {
      ref: editor,
      modelValue: value.value,
      'onUpdate:modelValue': (next: string) => { value.value = next },
      placeholder: '说点什么',
      stickerPicker: true,
    }),
    h('button', {
      id: 'insert-sticker',
      type: 'button',
      onClick: () => editor.value?.insertMarkdown('[:sticker:huaji:]'),
    }, 'insert sticker'),
    h('pre', { id: 'model-value' }, value.value),
  ]),
}).use(i18n).mount('#app')
