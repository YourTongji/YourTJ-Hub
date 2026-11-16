import { h, ref } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import MarkdownImportDialog from '../../../src/site/components/MarkdownImportDialog.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import { applyThemeTemporary, type SiteTheme } from '../../../src/runtime/site-theme'
import '../../../src/styles/resource.css'
await setLocale('zh')
applyThemeTemporary((new URLSearchParams(location.search).get('theme') || 'gf-light') as SiteTheme)
const open = ref(true)
createApp({
  render: () => h(MarkdownImportDialog, { open: open.value, 'onUpdate:open': (value: boolean) => (open.value = value), bodyEmpty: true, allowTitle: true, titleEmpty: true }),
}).use(i18n).mount('#app')
