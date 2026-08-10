import { createApp, h, shallowRef } from 'vue'
import App from '@/site/App.vue'
import '@/styles/resource.css'
import { readInitialPayload, updateDocumentMeta } from '@/runtime/payload'
import { installNavigation, preparePayload } from '@/runtime/router'
import { currentLocale, i18n } from '@/runtime/i18n'
import { hydrateFlashMessages } from '@/runtime/flash-message'
import { applySiteThemePayload, applyStoredTheme } from '@/runtime/site-theme'
import { applyStoredAppearanceSettings } from '@/runtime/appearance-settings'
import { installBaTouchEffect } from '@/runtime/ba-touch-effect'
import PayloadRouteView from '@/site/components/PayloadRouteView.vue'
import { codeHighlightDirective } from '@/runtime/code-highlight-directive'
import { mathRenderDirective } from '@/runtime/math-render-directive'
import { codeCopyDirective } from '@/runtime/code-copy-directive'

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
applyStoredAppearanceSettings()
installBaTouchEffect()

function commitPage(nextPage: typeof initialPage) {
  currentPage.value = nextPage
  applySiteThemePayload(nextPage.payload.layout.theme)
  updateDocumentMeta(nextPage.payload)
}

const router = installNavigation(initialPage, PayloadRouteView, (nextPage) => {
  commitPage(nextPage)
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
