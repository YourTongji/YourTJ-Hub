import { createApp, h } from 'vue'
import LoginPage from '../../../src/site/pages/LoginPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { LayoutPayload, LoginPageProps } from '@gooseforum/client'
import '../../../src/styles/resource.css'

const params = new URLSearchParams(location.search)
await setLocale((params.get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
const props: LoginPageProps = {
  initialMode: params.get('mode') === 'register' ? 'register' : 'login',
  redirectUrl: '/campus', githubUrl: '/api/auth/github', googleUrl: '/api/auth/google', googleReady: true,
  tongjiReady: true, tongjiUrl: '/api/auth/tongji?redirect=%2Fcampus',
  termsOfServiceEnabled: true, privacyPolicyEnabled: true, allowedDomains: ['tongji.edu.cn'], oauthNotice: false,
}
const layout = {
  site: { name: 'yourtj', brandType: 'text' },
  viewer: { isAuthenticated: false, id: 0 },
} as LayoutPayload
createApp({ render: () => h(LoginPage, { layout, props }) }).use(i18n).mount('#app')
document.documentElement.dataset.ready = 'true'
