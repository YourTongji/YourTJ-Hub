import { h } from 'vue'
import { Building2 } from '@lucide/vue'
import { createBrowserApp as createApp } from './create-app'
import CourseCatalogMultiSelect from '../../../src/site/components/CourseCatalogMultiSelect.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

await setLocale('zh')
document.documentElement.dataset.theme = 'gf-light'
const marginTop = new URLSearchParams(location.search).get('side') === 'up' ? `${innerHeight - 180}px` : '48px'
const form = h('form', { id: 'course-more-filters', action: '/courses', method: 'get' }, [
  h('section', { class: 'gf-card overflow-hidden', style: `height: 112px; margin: ${marginTop} 16px 1200px; padding: 16px` }, [
    h(CourseCatalogMultiSelect, {
      modelName: 'department',
      form: 'course-more-filters',
      label: '全部院系',
      options: [{ value: 'CS', label: '计算机科学与技术' }],
      selected: [],
      icon: Building2,
    }),
  ]),
])

createApp({ render: () => form }).use(i18n).mount('#app')
document.documentElement.dataset.ready = 'true'
