import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import AnonymousIdentitySettings from '../../../src/site/components/AnonymousIdentitySettings.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
await setLocale('zh')
createApp({ render: () => h('main', { class: 'mx-auto max-w-3xl p-3' }, h(AnonymousIdentitySettings)) }).use(i18n).mount('#app')
