import { h } from 'vue'
import { createBrowserApp } from './create-app'
import GlobalFlash from '../../../src/site/components/GlobalFlash.vue'
import { hydrateFlashMessages } from '../../../src/runtime/flash-message'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

await setLocale('zh')
createBrowserApp({ render: () => h(GlobalFlash) }).use(i18n).mount('#app')
hydrateFlashMessages()
document.documentElement.dataset.ready = 'true'
