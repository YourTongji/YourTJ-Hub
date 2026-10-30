import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import StickerPicker from '../../../src/site/components/StickerPicker.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

await setLocale('zh')

createApp({
  render: () => h('div', { class: 'relative p-4', style: 'height: 1600px' }, [
    h(StickerPicker, { open: true, placement: 'below' }),
  ]),
}).use(i18n).mount('#app')
