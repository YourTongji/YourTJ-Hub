import { h, ref } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import VditorOfficial from '../../../src/site/components/VditorOfficial.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

await setLocale('zh')

const value = ref('')
const editor = ref<{ insertMarkdown: (markdown: string) => void } | null>(null)

// 测试钩子：宿主设置正文（含既有贴纸 token）走 modelValue，等价于草稿/回填路径
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
    }),
    // 与三个宿主编辑器插入贴纸的调用一致（element.click() 不移焦，光标仍在编辑区）
    h('button', {
      id: 'insert-sticker',
      type: 'button',
      onClick: () => editor.value?.insertMarkdown('[:sticker:huaji:]'),
    }, 'insert sticker'),
    h('pre', { id: 'model-value' }, value.value),
  ]),
}).use(i18n).mount('#app')
