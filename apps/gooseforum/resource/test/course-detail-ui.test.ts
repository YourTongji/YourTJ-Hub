import { readFileSync } from 'node:fs'
import { describe, expect, test } from 'vitest'

const detailSource = readFileSync(
  new URL('../src/site/pages/CourseDetailPage.vue', import.meta.url),
  'utf8',
)
const templateSelectorSource = readFileSync(
  new URL('../src/site/components/CourseReviewTemplateSelector.vue', import.meta.url),
  'utf8',
)

describe('课程详情页 UI 结构', () => {
  test('桌面端（xl+）评价列表为主列，评分分布/开课记录/相关课程收纳右栏', () => {
    expect(detailSource).toContain(
      'xl:grid-cols-[minmax(0,1fr)_minmax(0,340px)]',
    )
    expect(detailSource).toContain('xl:order-1')
    expect(detailSource).toContain('xl:order-2')
  })

  test('写评价入口只保留在评价列表标题处', () => {
    expect(detailSource.match(/@click="openCreateForm"/g)).toHaveLength(1)
  })

  test('写评表单和页面弹层使用全局过渡', () => {
    expect(detailSource.match(/<Transition name="gf-local-expand">/g)).toHaveLength(1)
    expect(detailSource.match(/<Transition name="gf-modal">/g)).toHaveLength(2)
    expect(templateSelectorSource.match(/<Transition name="gf-modal">/g)).toHaveLength(1)
  })
})
