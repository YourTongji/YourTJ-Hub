// @vitest-environment happy-dom
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import VditorOfficial from '../src/site/components/VditorOfficial.vue'
import { i18n } from '../src/runtime/i18n'

describe('移动视图下 Vditor 编辑器工具栏架构与交互规范', () => {
  test('文章模式在移动端（!simple && !compact）保持单行流线型布局，主行9项高频+more优雅收纳', async () => {
    // 模拟移动端视口 (max-width: 520px)
    const originalMatchMedia = window.matchMedia
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query.includes('520px'),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))

    const wrapper = mount(VditorOfficial, {
      props: {
        simple: false,
        compact: false,
        outline: true,
      },
      global: { plugins: [i18n] },
    })

    // 获取内部 resolveToolbar 的配置结果
    const toolbarList = (wrapper.vm as any).resolveToolbar()
    const itemNames = toolbarList.map((item: any) => (typeof item === 'string' ? item : item.name))

    // 文章模式主行必须精炼为单行9项（绝不换行臃肿成两行）：
    // 标题、加粗、斜体、引用、列表、代码块、图片、链接、更多
    expect(itemNames).toEqual([
      'headings',
      'bold',
      'italic',
      'quote',
      'list',
      'code',
      'upload',
      'link',
      'more',
    ])

    // 检查 more 子菜单内收纳的完整丰富排版能力：表格、表情、撤销/重做、删除线、任务列表、公式等
    const moreItem = toolbarList.find((item: any) => typeof item === 'object' && item.name === 'more')
    expect(moreItem).toBeDefined()
    expect(moreItem.toolbar).toContain('table')
    expect(moreItem.toolbar).toContain('emoji')
    expect(moreItem.toolbar).toContain('undo')
    expect(moreItem.toolbar).toContain('redo')
    expect(moreItem.toolbar).toContain('strike')
    expect(moreItem.toolbar).toContain('ordered-list')
    expect(moreItem.toolbar).toContain('check')
    expect(moreItem.toolbar).toContain('inline-code')
    expect(moreItem.toolbar).toContain('line')
    expect(moreItem.toolbar).toContain('fullscreen')
    expect(moreItem.toolbar).toContain('preview')

    wrapper.unmount()
    window.matchMedia = originalMatchMedia
  })

  test('回复模式在移动端（:compact="true"）解析聚焦高频交流与引用的 COMPACT_MOBILE_TOOLBAR，消除偏侧', async () => {
    const originalMatchMedia = window.matchMedia
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query.includes('520px'),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))

    const wrapper = mount(VditorOfficial, {
      props: {
        compact: true,
        height: 320,
      },
      global: { plugins: [i18n] },
    })

    const toolbarList = (wrapper.vm as any).resolveToolbar()
    const itemNames = toolbarList.map((item: any) => (typeof item === 'string' ? item : item.name))

    // 回复模式主行精准覆盖社区交流与回复高频工具：表情、图片、引用、加粗、代码、列表、链接、撤销、重做、更多
    expect(itemNames).toContain('emoji')
    expect(itemNames).toContain('upload')
    expect(itemNames).toContain('quote') // 引用他人发言核心功能
    expect(itemNames).toContain('bold')
    expect(itemNames).toContain('code')  // 代码与排错核心功能
    expect(itemNames).toContain('list')
    expect(itemNames).toContain('link')
    expect(itemNames).toContain('undo')
    expect(itemNames).toContain('redo')
    expect(itemNames).toContain('more')

    // 严禁包含令普通用户困惑的 edit-mode / insert-after
    expect(itemNames).not.toContain('edit-mode')
    expect(itemNames).not.toContain('insert-after')

    // 检查高度：compact 模式在移动端尊重 props.height，不强占半屏
    const resolvedHeight = (wrapper.vm as any).resolveHeight()
    expect(resolvedHeight).toBe(320)

    wrapper.unmount()
    window.matchMedia = originalMatchMedia
  })

  test('短文/极简模式在移动端（:simple="true"）解析轻量纯粹的 SIMPLE_MOBILE_TOOLBAR', async () => {
    const originalMatchMedia = window.matchMedia
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query.includes('520px'),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))

    const wrapper = mount(VditorOfficial, {
      props: {
        simple: true,
      },
      global: { plugins: [i18n] },
    })

    const toolbarList = (wrapper.vm as any).resolveToolbar()
    const itemNames = toolbarList.map((item: any) => (typeof item === 'string' ? item : item.name))

    expect(itemNames).toContain('upload')
    expect(itemNames).toContain('emoji')
    expect(itemNames).toContain('bold')
    expect(itemNames).toContain('italic')
    expect(itemNames).toContain('list')
    expect(itemNames).toContain('link')
    expect(itemNames).toContain('undo')
    expect(itemNames).toContain('redo')

    wrapper.unmount()
    window.matchMedia = originalMatchMedia
  })

  test('窄容器课评模式（:slim-mobile + :compact）移动端主行 7 项，≤320px 视口单行不裁剪', async () => {
    // 课评表单嵌套层级多（vw-64）：320px 视口下容器仅 ~254px，
    // compact 移动端 10 项（325px）会溢出裁掉 more 门；slim 预设收敛到 7 项（229px）。
    const originalMatchMedia = window.matchMedia
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query.includes('520px'),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))

    const wrapper = mount(VditorOfficial, {
      props: {
        compact: true,
        slimMobile: true,
        height: 300,
      },
      global: { plugins: [i18n] },
    })

    const toolbarList = (wrapper.vm as any).resolveToolbar()
    const itemNames = toolbarList.map((item: any) => (typeof item === 'string' ? item : item.name))

    // 主行恰好 7 项（7×32+5=229px ≤ 320px 视口下的 254px 容器），单行完整；
    // 标题放左一（对齐官方默认工具栏），代码块等低频项收进 more
    expect(itemNames).toEqual([
      'headings',
      'emoji',
      'upload',
      'bold',
      'quote',
      'list',
      'more',
    ])

    // 撤销/重做与低频排版能力收进 more，不丢失功能入口；标题已在主行不得重复
    const moreItem = toolbarList.find((item: any) => typeof item === 'object' && item.name === 'more')
    expect(moreItem).toBeDefined()
    expect(moreItem.toolbar).not.toContain('headings')
    expect(moreItem.toolbar).toContain('undo')
    expect(moreItem.toolbar).toContain('redo')
    expect(moreItem.toolbar).toContain('code')
    expect(moreItem.toolbar).toContain('italic')
    expect(moreItem.toolbar).toContain('link')
    expect(moreItem.toolbar).toContain('ordered-list')
    expect(moreItem.toolbar).toContain('table')
    expect(moreItem.toolbar).toContain('fullscreen')

    wrapper.unmount()
    window.matchMedia = originalMatchMedia
  })

  test('slimMobile 桌面端：compact 同构但标题提升左一；纯 compact 宿主不受影响', async () => {
    const originalMatchMedia = window.matchMedia
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))

    const slimWrapper = mount(VditorOfficial, {
      props: {
        compact: true,
        slimMobile: true,
      },
      global: { plugins: [i18n] },
    })

    const slimList = (slimWrapper.vm as any).resolveToolbar()
    const slimNames = slimList.map((item: any) => (typeof item === 'string' ? item : item.name))
    // 标题左一（对齐官方默认工具栏与移动端 slim），其余保持 compact 主行顺序
    expect(slimNames[0]).toBe('headings')
    expect(slimNames[1]).toBe('bold')
    const slimMore = slimList.find((item: any) => typeof item === 'object' && item.name === 'more')
    expect(slimMore.toolbar).not.toContain('headings')
    expect(slimMore.toolbar).toContain('table')
    slimWrapper.unmount()

    // 纯 compact 宿主（回复浮窗等）完全不受 slimMobile 预设影响
    const compactWrapper = mount(VditorOfficial, {
      props: { compact: true },
      global: { plugins: [i18n] },
    })
    const compactList = (compactWrapper.vm as any).resolveToolbar()
    const compactNames = compactList.map((item: any) => (typeof item === 'string' ? item : item.name))
    expect(compactNames[0]).toBe('bold')
    const compactMore = compactList.find((item: any) => typeof item === 'object' && item.name === 'more')
    expect(compactMore.toolbar).toContain('headings')
    compactWrapper.unmount()

    window.matchMedia = originalMatchMedia
  })

  test('slim 编辑器面板避让：slimMobile 挂 slim 类 + 标题/表情面板左锚定 CSS（对齐 QuickPublishModal 方案）', async () => {
    // 楼层回复编辑器表情为左一、面板默认向右展开故无溢出；课评 slim 把标题提到左一后，
    // vditor 对左侧按钮面板加 --left（right:0）→ 宽面板伸向视口外左侧。
    // 修复采用纯 CSS（同 QuickPublishModal 的 emoji 避让）：左锚定 + 视口宽度封顶。
    const componentSource = readFileSync(
      resolve(process.cwd(), 'src/site/components/VditorOfficial.vue'),
      'utf8',
    )

    // 根节点仅在 slimMobile 时挂避让作用域类，其他编辑器不受影响
    expect(componentSource).toContain(`:class="{ 'vditor-official--slim': slimMobile }"`)

    // 标题面板（.vditor-hint，无 .vditor-panel 类）与表情面板均左锚定且按视口封顶
    expect(componentSource).toContain(
      ".vditor-official--slim .vditor-toolbar__item:has(> [data-type='headings']) > .vditor-hint",
    )
    expect(componentSource).toContain(
      ".vditor-official--slim .vditor-toolbar__item:has(> [data-type='emoji']) > .vditor-panel",
    )
    expect(componentSource).toContain('max-width: min(320px, calc(100vw - 48px)) !important')

    // 表情在第二格：面板左移 32px 与标题面板对齐编辑器左缘；箭头随左锚定复位
    expect(componentSource).toContain('left: -32px !important')
    expect(componentSource).toContain('left: 12px !important')
    expect(componentSource).toContain('right: auto !important')
  })

  test('编辑器图片不启用 Vditor 原生灯箱，避免预览阻断内容编辑', () => {
    const componentSource = readFileSync(
      resolve(process.cwd(), 'src/site/components/VditorOfficial.vue'),
      'utf8',
    )

    expect(componentSource).toMatch(/image:\s*\{\s*isPreview:\s*false,\s*\}/)
  })
})
