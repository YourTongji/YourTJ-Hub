import { h, ref } from 'vue'
import AdminSelect from '../../../src/admin/components/AdminSelect.vue'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../../src/admin/components/ui/select'
import ScheduleTimeGrid from '../../../src/site/components/schedule/ScheduleTimeGrid.vue'
import type { PkCourseOnTable } from '../../../src/site/types/pk'
import type { GridLayout } from '../../../src/site/utils/timetableGrid'
import { createBrowserApp } from './create-app'
import { i18n, setLocale } from '../../../src/runtime/i18n'

const params = new URLSearchParams(location.search)
const requestedSurface = params.get('surface')
const surface = requestedSurface === 'select' ? 'select' : requestedSurface === 'probe' ? 'probe' : 'schedule'
const theme = params.get('theme') === 'dark' ? 'dark' : 'light'
const viewportIsMobile = window.matchMedia('(max-width: 767px)').matches
document.documentElement.dataset.theme = `gf-${theme}`
document.documentElement.classList.toggle('dark', theme === 'dark' && surface === 'select')
document.body.dataset.surface = surface
document.body.dataset.theme = theme
await setLocale('zh')

const longCourse: PkCourseOnTable = {
  showText: '',
  courseName: '高等代数与应用数学课程综合实践'.repeat(10),
  code: 'TJ-ALG-2030',
  occupyTime: [1, 2],
  occupyDay: 1,
  occupyWeek: [1, 2, 3, 4],
  teacherAndCode: '李老师 12345',
  arrangementText: '',
  occupyRoom: '济事楼 A101',
}
const shortCourse: PkCourseOnTable = {
  ...longCourse,
  courseName: '概率论',
  code: 'MATH-101',
}
const columns: PkCourseOnTable[][] = Array.from({ length: 7 }, () => [])
columns[0] = [longCourse]
columns[1] = [shortCourse]
const nextRow: PkCourseOnTable[][] = Array.from({ length: 7 }, () => [])
const grid: GridLayout = {
  cellCourses: [columns, nextRow],
  cellSpans: [[2, 2, 1, 1, 1, 1, 1], [1, 1, 1, 1, 1, 1, 1]],
  occupiedGrid: [
    [false, false, false, false, false, false, false],
    [true, true, false, false, false, false, false],
  ],
}

const options = [
  { value: 'short', label: '短选项' },
  { value: 'long', label: '学院、专业与课程组合筛选的超长管理端选项，用于验证单行省略和触发器宽度' },
]
const selectedValue = ref(surface === 'select' ? 'long' : 'short')
const directSelectedValue = ref('long')
const directSelect = () => h(Select, {
  modelValue: directSelectedValue.value,
  'onUpdate:modelValue': value => { directSelectedValue.value = String(value) },
}, {
  default: () => [
    h(SelectTrigger, { class: 'w-full', 'aria-label': '直接共享课程筛选' }, {
      default: () => h(SelectValue, { placeholder: '请选择课程' }),
    }),
    h(SelectContent, {}, {
      default: () => options.map(option => h(SelectItem, { value: option.value }, { default: () => option.label })),
    }),
  ],
})

const app = createBrowserApp({
  render: () => h('main', [
    surface === 'schedule'
      ? h('section', { id: 'schedule-panel' }, [
          h('h2', '周视图课程表'),
          h('div', { id: 'schedule-scroll' }, [
            h(ScheduleTimeGrid, {
              grid,
              sectionTimes: [],
              isMobile: viewportIsMobile,
              interactive: false,
              showCourseCodes: true,
            }),
          ]),
        ])
      : null,
    surface === 'probe'
      ? h('section', { id: 'probe-panel' }, [
          h('h2', '通用断言自证'),
          // 静态门禁盲区①：跨属性拆分（静态 class 与动态 class 分写，运行时合并成
          // "line-clamp-2 block"）——源码逐串扫描不合并多属性，浏览器裁决必须抓到。
          h('p', { id: 'probe-split', class: ['line-clamp-2', 'block'] }, '跨属性拆分探针'),
          // 静态门禁盲区②：可叠加变体组合（dark:clamp + hover:block 由运行时状态裁决）。
          h('p', { id: 'probe-legal', class: 'line-clamp-2 text-sm' }, '合法截断探针'),
          // line-clamp-none 是显式取消截断，不应被断言误报。
          h('p', { id: 'probe-none', class: 'line-clamp-none block' }, '显式取消截断探针'),
        ])
      : null,
    surface === 'select'
      ? h('section', { id: 'select-panel' }, [
          h('h2', '管理端共享下拉框'),
          h('div', { id: 'select-target' }, [
            h(AdminSelect, {
              modelValue: selectedValue.value,
              options,
              ariaLabel: '管理端课程筛选',
              class: 'w-full',
              'onUpdate:modelValue': value => { selectedValue.value = String(value) },
            }),
          ]),
          h('div', { id: 'select-target-direct' }, [directSelect()]),
        ])
      : null,
  ]),
}).use(i18n)
app.mount('#app')
document.documentElement.dataset.ready = 'true'
