import { createApp, h, ref } from 'vue'
import VditorOfficial from '../../../src/site/components/VditorOfficial.vue'
import ExternalLinkGuard from '../../../src/site/components/ExternalLinkGuard.vue'
import { i18n } from '../../../src/runtime/i18n'
import { externalLinkGuardState, cancelExternalLinkGuard } from '../../../src/runtime/external-link-guard'
import { enhanceLinkPreviews } from '../../../src/runtime/content-enhancements/link-preview'
import '../../../src/styles/resource.css'

i18n.global.locale.value = 'zh'
const markdown = ref('https://old.example/article')
const editor = ref<InstanceType<typeof VditorOfficial> | null>(null)
const root = document.createElement('section')
root.innerHTML = '<a id="external" href="https://outside.example/article">Outside</a> <p><a id="blocked" href="https://blocked.example/">https://blocked.example/</a></p> <a id="suspicious" href="https://outside.example/again">https://bank.example/login</a> <a id="internal" href="#inside">Inside</a>'
document.body.append(root)
void enhanceLinkPreviews(root)
createApp({ render: () => h('main', { style: 'max-width:720px;margin:auto' }, [
  h(VditorOfficial, { ref: editor, placeholder: 'Write', modelValue: markdown.value, 'onUpdate:modelValue': (value: string) => { markdown.value = value } }),
  h(ExternalLinkGuard),
]) }).use(i18n).mount('#app')
Object.assign(window, { linkFixture: {
  ready: () => editor.value?.editorReady,
  getValue: () => editor.value?.getValue(),
  model: () => markdown.value,
  setValue: (value: string) => { markdown.value = value },
  pending: () => externalLinkGuardState.pending,
  cancel: cancelExternalLinkGuard,
} })
