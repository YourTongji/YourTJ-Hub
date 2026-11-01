import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import StickerPicker from '../../../src/site/components/StickerPicker.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

await setLocale('zh')

// 锚点贴着视口底部：placement=below 放不进视口，面板应翻转到 above。
createApp({
  render: () => h('div', { class: 'relative', style: 'margin-top: 520px; height: 120px' }, [
    h(StickerPicker, { open: true, placement: 'below' }),
  ]),
}).use(i18n).mount('#app')
