import { createApp } from 'vue'
import App from './App.vue'
import { i18n, setLocale, type Locale } from './runtime/i18n'
import './styles/app.css'
const query = new URLSearchParams(location.search)
const selected = query.get('lang') ?? navigator.language.slice(0, 2)
setLocale((['zh','en','ja','de'].includes(selected) ? selected : 'en') as Locale)
const requested = query.get('theme')
document.documentElement.dataset.theme = requested === 'gf-dark' || requested === 'gf-light' ? requested : matchMedia('(prefers-color-scheme: dark)').matches ? 'gf-dark' : 'gf-light'
createApp(App).use(i18n).mount('#app')
