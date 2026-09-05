// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import ScheduleCoursePicker from '../src/site/components/schedule/ScheduleCoursePicker.vue'

// 选课弹窗打开时会加载字典（getPkCampuses/getPkFaculties），mock 掉避免真实 fetch。
vi.mock('../src/runtime/pk-api', () => ({
  getPkCampuses: vi.fn(async () => []),
  getPkFaculties: vi.fn(async () => []),
  getPkCoursesByMajor: vi.fn(async () => []),
  getPkOptionalTypes: vi.fn(async () => []),
  getPkCoursesByNature: vi.fn(async () => []),
  searchPkCourses: vi.fn(async () => []),
  getPkCourseDetails: vi.fn(async () => []),
}))

describe('ScheduleCoursePicker Dialog 可访问性', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh'
  })

  test('aria-describedby 指向存在的描述元素（P2-6 建议项）', async () => {
    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    const dialog = document.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()
    const describedBy = dialog?.getAttribute('aria-describedby')
    expect(describedBy).toBeTruthy()
    const desc = document.getElementById(describedBy!)
    expect(desc).not.toBeNull()
    expect(desc?.textContent).toContain('课程')

    wrapper.unmount()
  })

  test('高级检索的校区和开课学院选择框启用关键词搜索', async () => {
    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    const tabs = [...document.querySelectorAll<HTMLElement>('[role="tab"]')]
    expect(tabs).toHaveLength(3)
    tabs[2].click()
    await flushPromises()
    const selects = [...document.querySelectorAll<HTMLElement>('[role="combobox"]')]
    expect(selects).toHaveLength(2)

    selects[0].dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, button: 0, pageX: 10, pageY: 10 }))
    await flushPromises()
    expect(document.querySelector('[data-testid="site-select-search-input"]')).not.toBeNull()
    wrapper.unmount()
  })

  test('快捷搜索框支持实时模糊过滤课程名称并支持一键清空', async () => {
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.setCompulsoryCourses([
      {
        courseCode: '121001',
        courseName: '高等数学A',
        courseNameReserved: '高等数学A',
        courseType: '必',
        credit: 5,
        faculty: '数学科学学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
      {
        courseCode: '122002',
        courseName: '大学物理B',
        courseNameReserved: '大学物理B',
        courseType: '必',
        credit: 4,
        faculty: '物理科学与工程学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    // 初始状态包含高等数学和大学物理
    expect(document.body.textContent).toContain('高等数学A')
    expect(document.body.textContent).toContain('大学物理B')

    // 搜索“数学”
    const searchInput = document.querySelector<HTMLInputElement>('input[placeholder*="搜索当前列表"]')
    expect(searchInput).not.toBeNull()
    searchInput!.value = '数学'
    searchInput!.dispatchEvent(new Event('input'))
    await flushPromises()

    expect(document.body.textContent).toContain('高等数学A')
    expect(document.body.textContent).not.toContain('大学物理B')

    // 搜索课程代码“122002”
    searchInput!.value = '122002'
    searchInput!.dispatchEvent(new Event('input'))
    await flushPromises()

    expect(document.body.textContent).not.toContain('高等数学A')
    expect(document.body.textContent).toContain('大学物理B')

    // 点击清空搜索按钮
    const clearSearchBtn = document.querySelector<HTMLButtonElement>('button[title*="清空搜索"]')
    expect(clearSearchBtn).not.toBeNull()
    clearSearchBtn!.click()
    await flushPromises()

    expect(document.body.textContent).toContain('高等数学A')
    expect(document.body.textContent).toContain('大学物理B')

    wrapper.unmount()
  })

  test('通识选修课支持横向分类胶囊导航与分类筛选', async () => {
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.setOptionalCourses([
      {
        courseCode: '001001',
        courseName: '音乐鉴赏',
        courseNameReserved: '音乐鉴赏',
        courseType: '选',
        credit: 2,
        courseNature: ['人文与艺术'],
        status: 0,
        teacher: [],
        courseDetail: [],
      },
      {
        courseCode: '002002',
        courseName: '人工智能导论',
        courseNameReserved: '人工智能导论',
        courseType: '选',
        credit: 2,
        courseNature: ['科学与技术'],
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    // 切换到通识选修 tab
    const tabs = [...document.querySelectorAll<HTMLElement>('[role="tab"]')]
    tabs[1].click()
    await flushPromises()

    // 检查分类切分 Tab：全部、人文与艺术、科学与技术
    expect(document.body.textContent).toContain('全部')
    expect(document.body.textContent).toContain('人文与艺术')
    expect(document.body.textContent).toContain('科学与技术')
    // 初始默认选中首个具体分类（人文与艺术），故初始仅展示该分类课程
    expect(document.body.textContent).toContain('音乐鉴赏')
    expect(document.body.textContent).not.toContain('人工智能导论')

    // 点击「科学与技术」分类切分 Tab
    const categoryButtons = [...document.querySelectorAll<HTMLButtonElement>('button')]
    const techBtn = categoryButtons.find((btn) => btn.textContent?.includes('科学与技术'))
    expect(techBtn).toBeDefined()
    techBtn!.click()
    await flushPromises()

    expect(document.body.textContent).not.toContain('音乐鉴赏')
    expect(document.body.textContent).toContain('人工智能导论')

    // 点击「全部」Tab
    const allBtn = categoryButtons.find((btn) => btn.textContent?.includes('全部'))
    expect(allBtn).toBeDefined()
    allBtn!.click()
    await flushPromises()

    expect(document.body.textContent).toContain('音乐鉴赏')
    expect(document.body.textContent).toContain('人工智能导论')

    wrapper.unmount()
  })

  test('高级检索表单提供重置条件功能与结果统计', async () => {
    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    // 切换到高级检索 tab
    const tabs = [...document.querySelectorAll<HTMLElement>('[role="tab"]')]
    tabs[2].click()
    await flushPromises()

    // 输入课程名称
    const courseNameInput = document.querySelector<HTMLInputElement>('input[placeholder*="请输入课程代码或课程名称"]')
    expect(courseNameInput).not.toBeNull()
    courseNameInput!.value = '操作系统'
    courseNameInput!.dispatchEvent(new Event('input'))
    await flushPromises()
    expect(courseNameInput!.value).toBe('操作系统')

    // 点击重置条件
    const resetBtn = [...document.querySelectorAll<HTMLButtonElement>('button')].find((btn) =>
      btn.textContent?.includes('重置条件'),
    )
    expect(resetBtn).toBeDefined()
    resetBtn!.click()
    await flushPromises()

    expect(courseNameInput!.value).toBe('')

    wrapper.unmount()
  })

  test('选课底栏展示已选统计与学分并在点击清空勾选时全部重置', async () => {
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.setCompulsoryCourses([
      {
        courseCode: '121001',
        courseName: '高等数学A',
        courseNameReserved: '高等数学A',
        courseType: '必',
        credit: 5,
        faculty: '数学科学学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    expect(document.body.textContent).toContain('暂未勾选课程')

    // 勾选第一门课程
    const checkbox = document.querySelector<HTMLInputElement>('input[type="checkbox"]')
    expect(checkbox).not.toBeNull()
    checkbox!.click()
    await flushPromises()

    // 检查底栏统计
    expect(document.body.textContent).toContain('已选 1 门课程')
    expect(document.body.textContent).toContain('5 学分')
    expect(document.body.textContent).toContain('加入备选课程 (1)')

    // 点击清空勾选
    const clearSelectedBtn = [...document.querySelectorAll<HTMLButtonElement>('button')].find((btn) =>
      btn.textContent?.includes('清空勾选'),
    )
    expect(clearSelectedBtn).toBeDefined()
    clearSelectedBtn!.click()
    await flushPromises()

    expect(document.body.textContent).toContain('暂未勾选课程')

    wrapper.unmount()
  })

  test('校区徽章使用 text-info 与 border-info/30 满足高对比度与无障碍规范', async () => {
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.setCompulsoryCourses([
      {
        courseCode: '121001',
        courseName: '高等数学A',
        courseNameReserved: '高等数学A',
        courseType: '必',
        credit: 5,
        faculty: '数学科学学院',
        campus: ['四平路校区', '嘉定校区'],
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    const campusBadge = [...document.querySelectorAll('span')].find((el) =>
      el.textContent?.includes('四平路校区、嘉定校区'),
    )
    expect(campusBadge).toBeDefined()
    expect(campusBadge?.className).toContain('text-info')
    expect(campusBadge?.className).toContain('border-info/30')
    expect(campusBadge?.className).toContain('bg-info/10')
    expect(campusBadge?.className).not.toContain('text-info-content')

    wrapper.unmount()
  })

  test('通识选修课横向分类轨道支持滚轮及左右滑动导航辅助', async () => {
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.setOptionalCourses([
      {
        courseCode: '001001',
        courseName: '艺术概论',
        courseNameReserved: '艺术概论',
        courseType: '选',
        credit: 2,
        courseNature: ['人文经典与审美素养'],
        status: 0,
        teacher: [],
        courseDetail: [],
      },
      {
        courseCode: '002002',
        courseName: '科技创新',
        courseNameReserved: '科技创新',
        courseType: '选',
        credit: 2,
        courseNature: ['科学探索与生命关怀'],
        status: 0,
        teacher: [],
        courseDetail: [],
      },
      {
        courseCode: '003003',
        courseName: '社会调查',
        courseNameReserved: '社会调查',
        courseType: '选',
        credit: 2,
        courseNature: ['社会发展与国际视野'],
        status: 0,
        teacher: [],
        courseDetail: [],
      },
      {
        courseCode: '004004',
        courseName: '工程设计',
        courseNameReserved: '工程设计',
        courseType: '选',
        credit: 2,
        courseNature: ['工程能力与创新思维'],
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    // 切换到通识选修 tab
    const tabs = [...document.querySelectorAll<HTMLElement>('[role="tab"]')]
    tabs[1].click()
    await flushPromises()

    const catRail = document.querySelector<HTMLElement>('[role="tablist"][aria-label="通识选修分类"]')
    expect(catRail).not.toBeNull()

    // 触发横向滚轮
    const wheelEvent = new WheelEvent('wheel', { deltaY: 100, deltaX: 0, bubbles: true })
    catRail!.dispatchEvent(wheelEvent)
    await flushPromises()

    wrapper.unmount()
  })

  test('激活的课程分类 Tab 上的计数徽章使用高对比度 text-neutral-content 与 bg-neutral-content/20', async () => {
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.setCompulsoryCourses([
      {
        courseCode: '121001',
        courseName: '高等数学A',
        courseNameReserved: '高等数学A',
        courseType: '必',
        credit: 5,
        faculty: '数学科学学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])
    store.setOptionalCourses([
      {
        courseCode: '001001',
        courseName: '艺术概论',
        courseNameReserved: '艺术概论',
        courseType: '选',
        credit: 2,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    // 默认激活第一个 Tab（计划内课程）
    const activeTab = document.querySelector('[role="tab"][aria-selected="true"]')
    expect(activeTab).not.toBeNull()
    expect(activeTab?.textContent).toContain('计划内课程')

    // 计数徽标应处于激活态高对比度样式：text-neutral-content 与 bg-neutral-content/20，严禁 text-primary 隐蔽色
    const activeBadge = activeTab?.querySelector('span.rounded-full')
    expect(activeBadge).not.toBeNull()
    expect(activeBadge?.textContent?.trim()).toBe('1')
    expect(activeBadge?.className).toContain('text-neutral-content')
    expect(activeBadge?.className).toContain('bg-neutral-content/20')
    expect(activeBadge?.className).not.toContain('text-primary')
    expect(activeBadge?.className).not.toContain('bg-primary/15')

    wrapper.unmount()
  })

  test('一键选择一整学期计划内课程支持全选、反选与半选状态联动，并自动跳过已在备选课程中的课程', async () => {
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.state.commonLists.stagedCourses = []
    store.setCompulsoryCourses([
      {
        courseCode: '121001',
        courseName: '高等数学A',
        courseNameReserved: '高等数学A',
        courseType: '必',
        credit: 5,
        faculty: '数学科学学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
      {
        courseCode: '121002',
        courseName: '线性代数',
        courseNameReserved: '线性代数',
        courseType: '必',
        credit: 3,
        faculty: '数学科学学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
      {
        courseCode: '121003',
        courseName: '大学英语',
        courseNameReserved: '大学英语',
        courseType: '必',
        credit: 2,
        faculty: '外国语学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])

    // 将大学英语预先加入备选课程
    store.pushStagedCourse({
      courseCode: '121003',
      courseName: '大学英语(121003)',
      courseNameReserved: '大学英语',
      credit: 2,
      courseType: '必',
      teacher: [],
      status: 0,
      courseDetail: [],
    })

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    // 找到一键全选复选框与文字
    const selectAllLabel = [...document.querySelectorAll('label')].find((el) =>
      el.textContent?.includes('一键全选本学期计划内课程'),
    )
    expect(selectAllLabel).toBeDefined()
    const selectAllCheckbox = selectAllLabel?.querySelector<HTMLInputElement>('input[type="checkbox"]')
    expect(selectAllCheckbox).not.toBeNull()

    // 初始状态：未勾选，总可勾选数为 2 门（排除大学英语）
    expect(selectAllCheckbox!.checked).toBe(false)
    expect(document.body.textContent).toContain('已选 0/2 门')

    // 手动勾选第 1 门课（高等数学A）
    const courseCheckboxes = document.querySelectorAll<HTMLInputElement>('li input[type="checkbox"]')
    expect(courseCheckboxes.length).toBe(3)
    // 第 3 门课在备选课程中应被禁用
    expect(courseCheckboxes[2].disabled).toBe(true)

    // 勾选第一门
    courseCheckboxes[0].click()
    await flushPromises()

    // 此时处于半选 (indeterminate)
    expect(selectAllCheckbox!.indeterminate).toBe(true)
    expect(document.body.textContent).toContain('已选 1/2 门')

    // 点击一键全选 -> 补齐全部可选课程
    selectAllCheckbox!.click()
    await flushPromises()

    expect(selectAllCheckbox!.checked).toBe(true)
    expect(selectAllCheckbox!.indeterminate).toBe(false)
    expect(document.body.textContent).toContain('已选 2/2 门')
    expect(document.body.textContent).toContain('加入备选课程 (2)')

    // 再次点击一键全选 -> 取消全选
    selectAllCheckbox!.click()
    await flushPromises()

    expect(selectAllCheckbox!.checked).toBe(false)
    expect(document.body.textContent).toContain('已选 0/2 门')

    // 勾选后通过清除按钮清空
    courseCheckboxes[0].click()
    await flushPromises()
    expect(document.body.textContent).toContain('已选 1/2 门')

    const clearBtn = [...document.querySelectorAll('button')].find(
      (btn) => btn.textContent?.trim() === '清除',
    )
    expect(clearBtn).toBeDefined()
    clearBtn!.click()
    await flushPromises()
    expect(document.body.textContent).toContain('已选 0/2 门')

    wrapper.unmount()
  })

  test('跨多年级计划课程分组支持单年级独立批选与取消', async () => {
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.state.commonLists.stagedCourses = []
    store.setCompulsoryCourses([
      {
        courseCode: '121001',
        courseName: '高等数学A',
        courseNameReserved: '高等数学A',
        courseType: '必',
        credit: 5,
        faculty: '数学科学学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
      {
        courseCode: '122002',
        courseName: '理论力学',
        courseNameReserved: '理论力学',
        courseType: '必',
        credit: 4,
        faculty: '航空航天与力学学院',
        grade: 2023,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    // 跨年级时应出现「选择该年级」操作按钮
    const gradeToggleBtns = [...document.querySelectorAll('button')].filter((btn) =>
      btn.textContent?.includes('选择该年级'),
    )
    expect(gradeToggleBtns.length).toBe(2)

    // 点击 2024级「选择该年级」
    gradeToggleBtns[0].click()
    await flushPromises()

    expect(document.body.textContent).toContain('已选 1/2 门')
    expect(document.body.textContent).toContain('取消该年级')

    wrapper.unmount()
  })

  test('顶栏快捷搜索过滤时，一键全选动态切换为「全选当前筛选课程」并仅勾选可见匹配项', async () => {
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.state.commonLists.stagedCourses = []
    store.setCompulsoryCourses([
      {
        courseCode: '121001',
        courseName: '高等数学A',
        courseNameReserved: '高等数学A',
        courseType: '必',
        credit: 5,
        faculty: '数学科学学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
      {
        courseCode: '121002',
        courseName: '线性代数',
        courseNameReserved: '线性代数',
        courseType: '必',
        credit: 3,
        faculty: '数学科学学院',
        grade: 2024,
        status: 0,
        teacher: [],
        courseDetail: [],
      },
    ])

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    // 初始显示全选本学期计划内课程
    expect(document.body.textContent).toContain('一键全选本学期计划内课程')
    expect(document.body.textContent).toContain('已选 0/2 门')

    // 在快捷搜索框中输入 "高等数学"
    const quickSearchInput = document.querySelector<HTMLInputElement>('input[placeholder*="搜索当前列表中的课程"]')
    expect(quickSearchInput).not.toBeNull()
    quickSearchInput!.value = '高等数学'
    quickSearchInput!.dispatchEvent(new Event('input'))
    await flushPromises()

    // 文案应动态切换为 "全选当前筛选课程"，且比例变为 "已选 0/1 门"
    expect(document.body.textContent).toContain('全选当前筛选课程')
    expect(document.body.textContent).toContain('已选 0/1 门')

    // 点击一键全选
    const selectAllCheckbox = document.querySelector<HTMLInputElement>('label input[type="checkbox"]')
    expect(selectAllCheckbox).not.toBeNull()
    selectAllCheckbox!.click()
    await flushPromises()

    expect(document.body.textContent).toContain('已选 1/1 门')

    // 清空快捷搜索
    quickSearchInput!.value = ''
    quickSearchInput!.dispatchEvent(new Event('input'))
    await flushPromises()

    // 两门课程均恢复展示，处于半选状态（已选 1/2 门）
    expect(document.body.textContent).toContain('一键全选本学期计划内课程')
    expect(document.body.textContent).toContain('已选 1/2 门')
    expect(selectAllCheckbox!.indeterminate).toBe(true)

    wrapper.unmount()
  })

  test('高级检索支持教师工号输入并在搜索时将 teacherCode 提交至后端 API，重置时正确清空', async () => {
    const { searchPkCourses } = await import('../src/runtime/pk-api')
    const { useScheduleStore } = await import('../src/site/composables/useScheduleStore')
    const store = useScheduleStore()
    store.state.majorSelected.calendarId = 123

    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    // 切换至「高级检索」Tab
    const tabs = [...document.querySelectorAll<HTMLElement>('[role="tab"]')]
    tabs[2].click()
    await flushPromises()

    // 检查「教师工号」输入框渲染与 placeholder
    const teacherCodeInput = document.querySelector<HTMLInputElement>('input[placeholder*="20231234"]')
    expect(teacherCodeInput).not.toBeNull()
    expect(document.body.textContent).toContain('教师工号')

    // 模拟输入教师工号与教师姓名
    teacherCodeInput!.value = '102488'
    teacherCodeInput!.dispatchEvent(new Event('input'))
    await flushPromises()

    const teacherNameInput = document.querySelector<HTMLInputElement>('input[placeholder*="教师姓名"]')
    expect(teacherNameInput).not.toBeNull()
    teacherNameInput!.value = '李教授'
    teacherNameInput!.dispatchEvent(new Event('input'))
    await flushPromises()

    // 点击「搜索」按钮
    const searchBtn = [...document.querySelectorAll<HTMLButtonElement>('button')].find((btn) =>
      btn.textContent?.includes('搜索') && !btn.textContent?.includes('高级'),
    )
    expect(searchBtn).toBeDefined()
    searchBtn!.click()
    await flushPromises()

    // 校验 searchPkCourses 收到 teacherCode: '102488' 和 teacherName: '李教授'
    expect(searchPkCourses).toHaveBeenCalledWith(
      expect.objectContaining({
        teacherCode: '102488',
        teacherName: '李教授',
      }),
    )

    // 点击「重置条件」按钮
    const resetBtn = [...document.querySelectorAll<HTMLButtonElement>('button')].find((btn) =>
      btn.textContent?.includes('重置条件'),
    )
    expect(resetBtn).toBeDefined()
    resetBtn!.click()
    await flushPromises()

    // 验证工号输入框与姓名输入框已被清空
    expect(teacherCodeInput!.value).toBe('')
    expect(teacherNameInput!.value).toBe('')

    wrapper.unmount()
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })
})
