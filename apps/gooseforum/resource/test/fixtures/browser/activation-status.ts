import { h } from 'vue'
import { createBrowserApp } from './create-app'
import ActivationEmailAction from '../../../src/site/components/ActivationEmailAction.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
const params = new URLSearchParams(location.search)
document.documentElement.dataset.theme = params.get('theme') === 'dark' ? 'gf-dark' : 'gf-light'
await setLocale(params.get('locale') === 'de' ? 'de' : 'zh')
createBrowserApp({ render: () => h('main', { class: 'min-h-screen bg-base-100 p-4 text-base-content' }, [h(ActivationEmailAction, { accountId: 7 })]) }).use(i18n).mount('#app')
