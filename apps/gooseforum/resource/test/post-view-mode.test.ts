import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

describe('usePostViewMode', () => {
  const localStorage = {
    getItem: vi.fn<[], string | null>(() => null),
    setItem: vi.fn(),
  }

  // 模块级共享状态：每个用例通过 resetModules + 动态导入获得干净状态。
  async function loadModule() {
    vi.resetModules()
    return import('../src/runtime/post-view-mode')
  }

  beforeEach(() => {
    localStorage.getItem.mockReset()
    localStorage.getItem.mockReturnValue(null)
    localStorage.setItem.mockReset()
    vi.stubGlobal('window', { localStorage })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('默认扁平：未手动选择时所有内容类型统一为全局默认（issue #580）', async () => {
    const { usePostViewMode } = await loadModule()

    expect(usePostViewMode().viewMode.value).toBe('flat')
  })

  test('手动选择全局生效并持久化为单值，所有实例即时跟随', async () => {
    const { usePostViewMode } = await loadModule()

    const qa = usePostViewMode()
    qa.setViewMode('tree')

    expect(qa.viewMode.value).toBe('tree')
    expect(localStorage.setItem).toHaveBeenCalledWith('goose:post-view-mode', 'tree')

    // 后续实例（不论所在话题的内容类型）读到的都是同一全局状态
    expect(usePostViewMode().viewMode.value).toBe('tree')
  })

  test('已存储的全局单值覆盖默认（含切回扁平）', async () => {
    localStorage.getItem.mockReturnValue('tree')
    const { usePostViewMode } = await loadModule()
    expect(usePostViewMode().viewMode.value).toBe('tree')

    localStorage.getItem.mockReturnValue('flat')
    const mod = await loadModule()
    expect(mod.usePostViewMode().viewMode.value).toBe('flat')
  })

  test('旧版按内容类型 map：各类型选择一致时继承为全局偏好', async () => {
    // 多类型且一致
    localStorage.getItem.mockReturnValue(JSON.stringify({ '1': 'tree', '3': 'tree' }))
    let mod = await loadModule()
    expect(mod.usePostViewMode().viewMode.value).toBe('tree')

    // 仅单个类型有记录同样继承
    localStorage.getItem.mockReturnValue(JSON.stringify({ '3': 'tree' }))
    mod = await loadModule()
    expect(mod.usePostViewMode().viewMode.value).toBe('tree')
  })

  test('旧版 map 类型间冲突或值非法时回落默认，不猜测用户偏好', async () => {
    // 类型间选择冲突（{ flat, tree }）无法判断全局倾向 → 默认扁平
    localStorage.getItem.mockReturnValue(JSON.stringify({ '1': 'flat', '3': 'tree' }))
    let mod = await loadModule()
    expect(mod.usePostViewMode().viewMode.value).toBe('flat')

    // map 中无合法值
    localStorage.getItem.mockReturnValue(JSON.stringify({ '1': 'banana' }))
    mod = await loadModule()
    expect(mod.usePostViewMode().viewMode.value).toBe('flat')
  })

  test('非法 JSON、非法单值与数组静默回落默认', async () => {
    localStorage.getItem.mockReturnValue('{not-json')
    let mod = await loadModule()
    expect(mod.usePostViewMode().viewMode.value).toBe('flat')

    localStorage.getItem.mockReturnValue('banana')
    mod = await loadModule()
    expect(mod.usePostViewMode().viewMode.value).toBe('flat')

    localStorage.getItem.mockReturnValue(JSON.stringify(['tree']))
    mod = await loadModule()
    expect(mod.usePostViewMode().viewMode.value).toBe('flat')

    localStorage.getItem.mockReturnValue('null')
    mod = await loadModule()
    expect(mod.usePostViewMode().viewMode.value).toBe('flat')
  })

  test('SSR 无 window 时走默认且 setViewMode 不崩溃（内存态仍生效）', async () => {
    vi.unstubAllGlobals()
    const { usePostViewMode } = await loadModule()

    const { viewMode, setViewMode } = usePostViewMode()
    expect(viewMode.value).toBe('flat')
    expect(() => setViewMode('tree')).not.toThrow()
    expect(viewMode.value).toBe('tree')
  })
})
