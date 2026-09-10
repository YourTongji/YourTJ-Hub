import { createApp, h, ref } from 'vue'
import VditorOfficial from '../../../src/site/components/VditorOfficial.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import { enhanceMathText } from '../../../src/runtime/math-render-directive'
import { renderMarkdownPreview } from '../../../src/runtime/markdown'
import '../../../src/styles/resource.css'
await setLocale('zh')
const editor = ref<InstanceType<typeof VditorOfficial>>()
const input = new URLSearchParams(location.search).get('source') || ''
createApp({ render: () => h(VditorOfficial, { ref: editor, modelValue: input, height: 400, placeholder: 'Math audit' }) }).use(i18n).mount('#app')
Object.assign(window, { mathAudit: { editor, enhanceMathText, renderMarkdownPreview } })
