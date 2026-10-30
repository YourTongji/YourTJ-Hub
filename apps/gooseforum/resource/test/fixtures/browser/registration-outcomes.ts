import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import LoginPage from '../../../src/site/pages/LoginPage.vue'
import GlobalFlash from '../../../src/site/components/GlobalFlash.vue'
import { hydrateFlashMessages } from '../../../src/runtime/flash-message'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { LayoutPayload, LoginPageProps } from '@gooseforum/client'
import '../../../src/styles/resource.css'
const params = new URLSearchParams(location.search)
await setLocale((params.get('lang') || 'en') as 'zh' | 'en' | 'ja' | 'de')
document.documentElement.dataset.theme = params.get('theme') || 'gf-light'
const landing = new URL(location.href)
landing.searchParams.set('landed', 'true')
const props: LoginPageProps = {
  initialMode: 'register', captchaRequired: false, redirectUrl: landing.pathname + landing.search,
  githubUrl: '', githubReady: false, googleReady: false,
  termsOfServiceEnabled: false, privacyPolicyEnabled: false, allowedDomains: [], oauthNotice: false,
}
const layout = { site: { name: 'yourtj', brandType: 'text' }, viewer: { id: 0, isAuthenticated: false } } as LayoutPayload
if (params.has('landed')) hydrateFlashMessages()
createApp({ render: () => params.has('landed') ? h(GlobalFlash) : h(LoginPage, { layout, props }) }).use(i18n).mount('#app')
document.documentElement.dataset.ready = 'true'
