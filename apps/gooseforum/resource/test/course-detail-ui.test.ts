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
    // 详情页三处弹层：举报评审/模板选择器/撰写评价前置确认（随弹层增补同步维护）
    expect(detailSource.match(/<Transition name="gf-modal">/g)).toHaveLength(3)
    expect(templateSelectorSource.match(/<Transition name="gf-modal">/g)).toHaveLength(1)
  })
})

// 提取指定函数体（从 `function name(` 到首个顶级右花括号），用于断言语句顺序。
function functionBody(source: string, name: string): string {
  const start = source.indexOf(`function ${name}(`)
  expect(start).toBeGreaterThanOrEqual(0)
  return source.slice(start, source.indexOf('\n}', start))
}

describe('课程详情页弹窗键盘可访问性', () => {
  test('删除确认弹窗具备 Esc 关闭、焦点陷阱与打开即聚焦（对齐举报弹窗）', () => {
    // Esc + Tab 焦点陷阱：复用举报弹窗 onReportKeydown 的处理模式。
    expect(detailSource).toContain('@keydown="onDeleteKeydown"')
    // 打开即聚焦：aria-modal="true" 声明的模态承诺必须在运行时兑现。
    expect(functionBody(detailSource, 'askRemoveReview')).toContain(
      'nextTick(() => deleteFocusableEls()[0]?.focus())',
    )
  })

  test('分享弹窗聚焦发生在 openShare 异步完成之后（sharePreview 先渲染再聚焦）', () => {
    const body = functionBody(detailSource, 'openShareDialog')
    expect(body).toContain('await openShare(review)')
    // 聚焦的 nextTick 必须排在 await 之后：openShare 置 sharePreview 前，
    // 弹窗（v-if="sharePreview"）尚未挂载，querySelector 落空即 no-op。
    expect(body.indexOf('await openShare(review)')).toBeLessThan(body.indexOf('nextTick'))
  })
})
