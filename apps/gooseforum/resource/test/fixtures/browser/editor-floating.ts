import { h, ref } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import VditorOfficial from '../../../src/site/components/VditorOfficial.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import { applyThemeTemporary, type SiteTheme } from '../../../src/runtime/site-theme'
import '../../../src/styles/resource.css'

const params = new URLSearchParams(location.search)
await setLocale('zh')
applyThemeTemporary((params.get('theme') || 'gf-light') as SiteTheme)

const art = '/assets/test/fixtures/browser/topic-feed-art.svg'
const value = ref(params.get('value') ?? `第一段文字，用来测试选区格式条。\n\n![封面](${art})\n\n![左](${art}?a#w=50) ![右](${art}?b#w=50)\n\n最后一段。`)
;(window as unknown as { testValue: () => string }).testValue = () => value.value

const host = params.get('host')
// dialog：带 transform 的对话框（fixed 浮层的包含块变成对话框，需要按实测修正偏差）
const hostProps = host === 'dialog'
  ? { role: 'dialog', 'aria-modal': 'true', class: 'fixed inset-x-6 top-16 bottom-6 overflow-auto rounded-box bg-base-100 p-4', style: 'transform: translateZ(0)' }
  : { class: host === 'scroll' ? 'h-[480px] overflow-auto p-4' : 'mx-auto max-w-3xl p-4' }

createApp({
  render: () => h('div', hostProps, [
    h(VditorOfficial, {
      modelValue: value.value,
      'onUpdate:modelValue': (next: string) => { value.value = next },
      placeholder: '说点什么',
      profile: 'post',
    }),
  ]),
}).use(i18n).mount('#app')
