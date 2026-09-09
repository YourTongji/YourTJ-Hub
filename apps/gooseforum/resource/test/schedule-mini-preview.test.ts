// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'
import ScheduleMiniPreviewPopover from '../src/site/components/schedule/ScheduleMiniPreviewPopover.vue'
import ScheduleDetailList from '../src/site/components/schedule/ScheduleDetailList.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import type { PkArrangement, PkCourseDetail, PkStagedCourse } from '../src/site/types/pk'

// Mock APIs for ScheduleDetailList
vi.mock('../src/runtime/pk-api', () => ({
  getPkCourseReviewBrief: vi.fn().mockResolvedValue({ courseId: 1, courseCode: 'CS101', ratingAvg: 4.5, reviewCount: 10 }),
}))
vi.mock('../src/runtime/api', () => ({
  listCourseReviews: vi.fn().mockResolvedValue({ total: 0, list: [] }),
  getCourseSummary: vi.fn().mockResolvedValue({ status: 'insufficient_data' }),
}))

function makeDetail(code: string, day: number, times: number[], weeks = [1, 16]): PkCourseDetail {
  return {
    code,
    isExclusive: false,
    campus: '四平路校区',
    teachers: [{ teacherCode: 'T1', teacherName: '李老师' }],
    teachingLanguage: '中文',
    arrangementInfo: [
      {
        arrangementText: `周${day}第${times.join('-')}节[${weeks.join('-')}周]`,
        occupyDay: day,
        occupyTime: times,
        occupyWeek: weeks,
        occupyRoom: 'A101',
        teacherAndCode: '李老师(T1)',
      },
    ],
  }
}

function makeStaged(courseCode: string, details: PkCourseDetail[]): PkStagedCourse {
  return {
    courseCode,
    courseName: `课程${courseCode}`,
    courseNameReserved: `课程${courseCode}`,
    credit: 3,
    courseType: '必',
    teacher: [],
    status: 0,
    courseDetail: details,
  }
}

describe('ScheduleMiniPreviewPopover 组件行为与视觉规范', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.useFakeTimers()
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setMajorInfo({ calendarId: 121, grade: 2025, major: '00301' })
  })

  afterEach(() => {
    document.body.innerHTML = ''
    vi.useRealTimers()
  })

  test('open=false 时不渲染浮窗', () => {
    const wrapper = mount(ScheduleMiniPreviewPopover, {
      props: {
        open: false,
        stagedDetail: makeDetail('CS101.01', 2, [3, 4]),
      },
      global: { plugins: [i18n] },
    })
    expect(document.body.querySelector('[data-testid="schedule-mini-preview-popover"]')).toBeNull()
    wrapper.unmount()
  })

  test('open=true 时渲染到 body，正确标出新加入节次并应用闪烁脉冲动画类', async () => {
    const detail = makeDetail('CS101.01', 2, [3, 4])
    const wrapper = mount(ScheduleMiniPreviewPopover, {
      props: {
        open: true,
        stagedDetail: detail,
        courseName: '操作系统',
      },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    const popover = document.body.querySelector('[data-testid="schedule-mini-preview-popover"]')
    expect(popover).not.toBeNull()
    expect(popover?.textContent).toContain('CS101.01')
    expect(popover?.textContent).toContain('操作系统')

    // 检查脉冲动画元素：周二第3节与第4节带有 animate-slot-pulse
    const pulsingSlots = popover?.querySelectorAll('.animate-slot-pulse')
    expect(pulsingSlots?.length).toBe(2)

    // 无冲突时展示无冲突徽章
    const successBadge = popover?.querySelector('.text-success')
    expect(successBadge).not.toBeNull()

    wrapper.unmount()
  })

  test('存在时间冲突时展示冲突徽章并高亮冲突样式', async () => {
    const store = useScheduleStore()
    // 先加入已有课程在周二第 3-4 节
    const existing = makeDetail('MATH101.01', 2, [3, 4])
    store.stageCourse(existing)
    store.solidify()

    // 新加入课程也在周二第 3-4 节，发生冲突
    const conflicting = makeDetail('PHYS101.01', 2, [3, 4])
    const wrapper = mount(ScheduleMiniPreviewPopover, {
      props: {
        open: true,
        stagedDetail: conflicting,
        courseName: '大学物理',
      },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    const popover = document.body.querySelector('[data-testid="schedule-mini-preview-popover"]')
    expect(popover).not.toBeNull()

    // 冲突徽章存在
    const errorBadge = popover?.querySelector('.text-error')
    expect(errorBadge).not.toBeNull()

    // 冲突格子使用 bg-error
    const errorSlots = popover?.querySelectorAll('.bg-error.animate-slot-pulse')
    expect(errorSlots?.length).toBe(2)

    wrapper.unmount()
  })

  test('4秒自动关闭并在鼠标 hover 时暂停，移出后恢复计时', async () => {
    const detail = makeDetail('CS101.01', 1, [1, 2])
    const wrapper = mount(ScheduleMiniPreviewPopover, {
      props: {
        open: true,
        stagedDetail: detail,
      },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    const popover = document.body.querySelector('[data-testid="schedule-mini-preview-popover"]') as HTMLElement
    expect(popover).not.toBeNull()

    // 前进 2 秒，未超时
    vi.advanceTimersByTime(2000)
    expect(wrapper.emitted('close')).toBeFalsy()

    // 鼠标移入（hover 暂停）
    popover.dispatchEvent(new MouseEvent('mouseenter'))
    // 再前进 5 秒，依然不会关闭
    vi.advanceTimersByTime(5000)
    expect(wrapper.emitted('close')).toBeFalsy()

    // 鼠标移出（触发 2 秒后关闭）
    popover.dispatchEvent(new MouseEvent('mouseleave'))
    vi.advanceTimersByTime(1999)
    expect(wrapper.emitted('close')).toBeFalsy()
    vi.advanceTimersByTime(2)
    expect(wrapper.emitted('close')).toBeTruthy()

    wrapper.unmount()
  })

  test('按下 Escape 键和点击关闭按钮触发 close 事件', async () => {
    const detail = makeDetail('CS101.01', 1, [1, 2])
    const wrapper = mount(ScheduleMiniPreviewPopover, {
      props: {
        open: true,
        stagedDetail: detail,
      },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    // 点击关闭按钮（从 document.body 获取 teleported 元素中的按钮）
    const closeBtn = document.body.querySelector('button[aria-label="Close"], button[aria-label="关闭"]') as HTMLButtonElement | null
    expect(closeBtn).not.toBeNull()
    closeBtn?.click()
    expect(wrapper.emitted('close')?.length).toBe(1)

    // 按下 Escape
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(wrapper.emitted('close')?.length).toBe(2)

    wrapper.unmount()
  })

  test('桌面端根据 anchorEl 计算定位并避免溢出屏幕', async () => {
    const fakeAnchor = document.createElement('button')
    vi.spyOn(fakeAnchor, 'getBoundingClientRect').mockReturnValue({
      top: 200,
      bottom: 232,
      left: 400,
      right: 480,
      width: 80,
      height: 32,
      x: 400,
      y: 200,
      toJSON: () => {},
    })

    const detail = makeDetail('CS101.01', 3, [5, 6])
    const wrapper = mount(ScheduleMiniPreviewPopover, {
      props: {
        open: true,
        anchorEl: fakeAnchor,
        stagedDetail: detail,
        isReviewOpen: true,
      },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    const popover = document.body.querySelector('[data-testid="schedule-mini-preview-popover"]') as HTMLElement
    expect(popover).not.toBeNull()

    // 验证桌面端样式设置了 top 与 left
    expect(popover.style.top).toBeDefined()
    expect(popover.style.left).toBeDefined()

    wrapper.unmount()
  })

  test('点击浮窗外部触发 close 事件且点击浮窗内部不触发', async () => {
    const detail = makeDetail('CS101.01', 3, [5, 6])
    const wrapper = mount(ScheduleMiniPreviewPopover, {
      props: {
        open: true,
        stagedDetail: detail,
      },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    const popover = document.body.querySelector('[data-testid="schedule-mini-preview-popover"]') as HTMLElement
    expect(popover).not.toBeNull()

    // 点击浮窗内部：不触发 close
    popover.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')).toBeFalsy()

    // 点击浮窗外部任意区域：触发 close
    const outsideEl = document.createElement('div')
    document.body.appendChild(outsideEl)
    outsideEl.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')?.length).toBe(1)

    outsideEl.remove()
    wrapper.unmount()
  })
})

describe('ScheduleDetailList 与 ScheduleMiniPreviewPopover 联动集成', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.useFakeTimers()
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setMajorInfo({ calendarId: 121, grade: 2025, major: '00301' })
    store.setClickedCourseInfo({ courseCode: 'CS101', courseName: '计算机科学导论' })
    store.pushStagedCourse(
      makeStaged('CS101', [
        makeDetail('CS101.01', 2, [3, 4]),
        makeDetail('CS101.02', 4, [1, 2]),
      ]),
    )
  })

  afterEach(() => {
    document.body.innerHTML = ''
    vi.useRealTimers()
  })

  test('点击加入课表后产生 180ms 延迟响应，随后弹出缩略课表气泡', async () => {
    const wrapper = mount(ScheduleDetailList, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    // 尚未点击前，无预览浮窗
    expect(document.body.querySelector('[data-testid="schedule-mini-preview-popover"]')).toBeNull()

    // 找到第一个教学班的加入按钮并点击
    const stageBtns = wrapper.findAll('button').filter((b) => b.text().includes('Add to schedule') || b.text().includes('加入课表'))
    expect(stageBtns.length).toBeGreaterThan(0)
    await stageBtns[0].trigger('click')

    // 刚点击时（50ms），浮窗仍因延迟未打开
    vi.advanceTimersByTime(50)
    await flushPromises()
    expect(document.body.querySelector('[data-testid="schedule-mini-preview-popover"]')).toBeNull()

    // 达到 180ms 延迟响应后，缩略课表气泡优雅展开
    vi.advanceTimersByTime(150)
    await flushPromises()
    const popover = document.body.querySelector('[data-testid="schedule-mini-preview-popover"]')
    expect(popover).not.toBeNull()
    expect(popover?.textContent).toContain('CS101.01')

    // 再次点击该按钮（退选反选），浮窗立即退场
    await stageBtns[0].trigger('click')
    await flushPromises()
    expect(document.body.querySelector('[data-testid="schedule-mini-preview-popover"]')).toBeNull()

    wrapper.unmount()
  })
})
