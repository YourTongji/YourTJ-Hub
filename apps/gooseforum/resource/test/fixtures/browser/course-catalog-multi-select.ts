import { defineComponent, h } from 'vue'
import { Building2 } from '@lucide/vue'
import { createBrowserApp } from './create-app'
import CourseCatalogMultiSelect from '../../../src/site/components/CourseCatalogMultiSelect.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

const side = new URLSearchParams(location.search).get('side') === 'up' ? 'up' : 'down'
const selected: string[] = []
const root = defineComponent({
  render: () => h('main', { class: 'min-h-[1200px] bg-base-100 p-4 text-base-content' }, [
    h('form', { id: 'course-more-filters' }, [
      h('section', { class: 'gf-card relative h-[100px] overflow-hidden', style: { marginTop: side === 'up' ? '300px' : '40px' } }, [
        h('div', { style: { position: 'absolute', left: '16px', width: '260px', ...(side === 'up' ? { bottom: '8px' } : { top: '8px' }) } }, [
          h(CourseCatalogMultiSelect, {
            modelName: 'department',
            form: 'course-more-filters',
            label: '全部院系',
            options: [{ value: 'math', label: '数学科学学院' }, { value: 'physics', label: '物理科学与工程学院' }],
            selected,
            icon: Building2,
          }),
        ]),
      ]),
    ]),
  ]),
})

await setLocale('zh')
createBrowserApp(root).use(i18n).mount('#app')
