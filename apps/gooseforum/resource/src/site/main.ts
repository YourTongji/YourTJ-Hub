import { createApp, h, shallowRef } from 'vue'
import App from '@/site/App.vue'
import '@/styles/resource.css'
import { readInitialPayload, updateDocumentMeta } from '@/runtime/payload'
import { fetchPage, installNavigation, preparePayload } from '@/runtime/router'
import { currentLocale, i18n } from '@/runtime/i18n'
import { hydrateFlashMessages } from '@/runtime/flash-message'
import { applySiteThemePayload, applyStoredTheme, initSystemThemeListener } from '@/runtime/site-theme'
import { applyStoredAppearanceSettings } from '@/runtime/appearance-settings'
import { installBaTouchEffect } from '@/runtime/ba-touch-effect'
import { registerAppNavigator } from '@/runtime/app-navigation'
import PayloadRouteView from '@/site/components/PayloadRouteView.vue'
import { codeHighlightDirective } from '@/runtime/code-highlight-directive'
import { mathRenderDirective } from '@/runtime/math-render-directive'
import { codeCopyDirective } from '@/runtime/code-copy-directive'
import { contentEnhancementsDirective } from '@/runtime/content-enhancements'

const initialPayload = readInitialPayload()
const initialPage = await preparePayload(initialPayload)
const currentPage = shallowRef(initialPage)
const navigationEntry = typeof window !== 'undefined'
  ? performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined
  : undefined
const isReloadNavigation = navigationEntry?.type === 'reload'

document.documentElement.lang = currentLocale()
applySiteThemePayload(initialPayload.layout.theme)
applyStoredTheme()
initSystemThemeListener()
applyStoredAppearanceSettings()
installBaTouchEffect()

// 加载 Noto Serif SC（wiki/正文衬线字体）。用国内可直连的 Google Fonts 镜像
// （fonts.googleapis.cn）加速，标签带预连接；请求失败不阻塞页面渲染。
function installNotoSerifSc() {
  const preconnect = document.createElement('link')
  preconnect.rel = 'preconnect'
  preconnect.href = 'https://fonts.googleapis.cn'
  preconnect.crossOrigin = ''
  document.head.appendChild(preconnect)

  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = 'https://fonts.googleapis.cn/css2?family=Noto+Serif+SC:wght@400;500;600;700&display=swap'
  link.onerror = () => link.remove()
  document.head.appendChild(link)
}
installNotoSerifSc()

function commitPage(nextPage: typeof initialPage) {
  if (currentPage.value.payload.layout.insightFlareEnabled !== nextPage.payload.layout.insightFlareEnabled) {
    window.location.reload()
    return
  }
  currentPage.value = nextPage
  applySiteThemePayload(nextPage.payload.layout.theme)
  updateDocumentMeta(nextPage.payload)
}

const router = installNavigation(initialPage, PayloadRouteView, (nextPage) => {
  commitPage(nextPage)
})
// 注册 SPA 导航桥：runtime 模块（如 browser-notification 通知点击）经此走 vue-router，
// 避免整页跳转；需在应用启动时注册一次。
registerAppNavigator((path) => {
  void router.push(path)
})

const app = createApp({
  setup() {
    return () => h(App, {
      page: currentPage.value,
    })
  },
})

app.use(i18n)
app.use(router)
app.directive('code-highlight', codeHighlightDirective)
app.directive('math-render', mathRenderDirective)
app.directive('code-copy', codeCopyDirective)
app.directive('content-enhancements', contentEnhancementsDirective)
await router.isReady()
app.mount('#goose-app')

if (isReloadNavigation && typeof window !== 'undefined' && !window.location.hash) {
  requestAnimationFrame(() => {
    window.scrollTo({ top: 0, left: 0, behavior: 'auto' })
  })
}

hydrateFlashMessages()

window.addEventListener('goose:page', async (event) => {
  const nextPayload = event instanceof CustomEvent ? event.detail : undefined
  if (!nextPayload) return
  commitPage(await preparePayload(nextPayload))
})

// 隐私政策配置可能在当前标签页打开后改变。每分钟拉取一个轻量公共页面
// payload，并在切回前台/bfcache 恢复时立即检查；状态变化就整页刷新，使 SDK 的
// history、visibilitychange 和 performance 监听及时卸载或按新配置加载。
let insightFlareStateCheckInFlight = false
async function checkInsightFlareState() {
  if (insightFlareStateCheckInFlight) return
  insightFlareStateCheckInFlight = true
  try {
    const payload = await fetchPage(new URL('/privacy', window.location.origin))
    if (currentPage.value.payload.layout.insightFlareEnabled !== payload.layout.insightFlareEnabled) {
      window.location.reload()
    }
  } catch {
    // 配置探测失败不打断当前页面；下一次轮询或恢复事件会重试。
  } finally {
    insightFlareStateCheckInFlight = false
  }
}
window.setInterval(() => void checkInsightFlareState(), 60_000)
document.addEventListener('visibilitychange', () => void checkInsightFlareState())
window.addEventListener('pageshow', () => void checkInsightFlareState())
