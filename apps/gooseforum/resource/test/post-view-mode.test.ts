import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { ref } from 'vue'

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

  test('contentType 1 默认树状，其余类型默认扁平', async () => {
    const { usePostViewMode } = await loadModule()

    expect(usePostViewMode(() => 1).viewMode.value).toBe('tree')
    expect(usePostViewMode(() => 0).viewMode.value).toBe('flat')
    expect(usePostViewMode(() => 2).viewMode.value).toBe('flat')
    expect(usePostViewMode(() => 3).viewMode.value).toBe('flat')
    expect(usePostViewMode(() => undefined).viewMode.value).toBe('flat')
  })

  test('手动选择按内容类型独立记忆并持久化', async () => {
    const { usePostViewMode } = await loadModule()

    const qa = usePostViewMode(() => 1)
    qa.setViewMode('flat')

    expect(qa.viewMode.value).toBe('flat')
    expect(localStorage.setItem).toHaveBeenCalledWith(
      'goose:post-view-mode',
      JSON.stringify({ '1': 'flat' }),
    )

    // 同类型其他实例共享状态
    expect(usePostViewMode(() => 1).viewMode.value).toBe('flat')
    // 其他类型不受影响，仍走各自默认
    expect(usePostViewMode(() => 3).viewMode.value).toBe('flat')
    expect(usePostViewMode(() => 2).viewMode.value).toBe('flat')
  })

  test('已存储的偏好覆盖内容类型默认', async () => {
    localStorage.getItem.mockReturnValue(JSON.stringify({ '3': 'tree', '1': 'flat' }))
    const { usePostViewMode } = await loadModule()

    expect(usePostViewMode(() => 3).viewMode.value).toBe('tree')
    expect(usePostViewMode(() => 1).viewMode.value).toBe('flat')
    // 未存储的类型仍走默认
    expect(usePostViewMode(() => 2).viewMode.value).toBe('flat')
  })

  test('非法 JSON 与非法值静默回落默认', async () => {
    localStorage.getItem.mockReturnValue('{not-json')
    let mod = await loadModule()
    expect(mod.usePostViewMode(() => 1).viewMode.value).toBe('tree')

    localStorage.getItem.mockReturnValue(JSON.stringify({ '1': 'banana', '2': 'tree' }))
    mod = await loadModule()
    expect(mod.usePostViewMode(() => 1).viewMode.value).toBe('tree')
    expect(mod.usePostViewMode(() => 2).viewMode.value).toBe('tree')

    localStorage.getItem.mockReturnValue(JSON.stringify(['tree']))
    mod = await loadModule()
    expect(mod.usePostViewMode(() => 1).viewMode.value).toBe('tree')
  })

  test('contentType 变化时视图随共享状态联动', async () => {
    const { usePostViewMode } = await loadModule()

    const contentType = ref<number | undefined>(1)
    const { viewMode, setViewMode } = usePostViewMode(() => contentType.value)

    expect(viewMode.value).toBe('tree')
    contentType.value = 3
    expect(viewMode.value).toBe('flat')

    setViewMode('tree')
    expect(viewMode.value).toBe('tree')
    contentType.value = 1
    // 类型 1 未手动选择过，回落默认树状（与类型 3 的手动选择互不影响）
    expect(viewMode.value).toBe('tree')
    expect(localStorage.setItem).toHaveBeenLastCalledWith(
      'goose:post-view-mode',
      JSON.stringify({ '3': 'tree' }),
    )
  })

  test('SSR 无 window 时走默认且 setViewMode 不崩溃', async () => {
    vi.unstubAllGlobals()
    const { usePostViewMode } = await loadModule()

    const { viewMode, setViewMode } = usePostViewMode(() => 1)
    expect(viewMode.value).toBe('tree')
    expect(() => setViewMode('flat')).not.toThrow()
    expect(viewMode.value).toBe('flat')
  })
})
