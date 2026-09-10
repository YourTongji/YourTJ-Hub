// @vitest-environment happy-dom
// 隔离模块单例：vi.resetModules() + 动态 import 重新求值 useQuickPublish。
// mock 工厂首次抛错模拟 chunk 加载失败（瞬态网络中断 / 发版替换旧 hash），
// 验证 Codex review 契约：被拒绝的 promise 不能被永久复用——失败后缓存清空、
// 首开锁存复位、下一次调用重新发起 import。
//
// 说明：只保留单一用例。vitest 对 mock 工厂的成功结果按模块缓存（跨
// resetModules 生效），同一文件内无法第二次模拟「失败→成功」；而预热与
// 正式打开调用的是同一个 loadQuickPublishModal()，无独立代码路径，
// 单用例已覆盖两者的失败恢复语义。
import { beforeEach, describe, expect, test, vi } from 'vitest'

const attempts = vi.hoisted(() => ({ count: 0 }))

vi.mock('@/site/components/QuickPublishModal.vue', () => {
  attempts.count += 1
  if (attempts.count === 1) {
    throw new Error('chunk load failed (network outage)')
  }
  return { default: { name: 'QuickPublishModalStub', template: '<div />' } }
})

beforeEach(() => {
  attempts.count = 0
  vi.resetModules()
})

describe('QuickPublish 弹层 chunk 加载失败恢复', () => {
  test('失败后清空缓存并复位首开锁存，下次调用重新发起 import 并成功', async () => {
    const { loadQuickPublishModal, useEverOpenedQuickPublish } = await import(
      '../src/site/composables/useQuickPublish'
    )
    const everOpened = useEverOpenedQuickPublish()
    everOpened.value = true // 模拟此前已成功打开过

    // 第一次调用（预热或打开）：chunk 加载失败 → promise 拒绝透传给调用方
    // （vitest 会包装 mock 工厂抛出的错误，这里只断言拒绝发生；
    //   「确实重新发起 import」由下方 attempts 计数证明）
    await expect(loadQuickPublishModal()).rejects.toThrow()
    expect(attempts.count).toBe(1)
    // 首开锁存复位：AppShell 的 v-if 卸载处于错误态的异步组件，
    // 下一次打开时重新挂载并重新触发 import
    expect(everOpened.value).toBe(false)

    // 网络恢复后的下一次交互：缓存已清空 → 重新发起 import（第 2 次工厂执行）并成功
    const mod = await loadQuickPublishModal()
    expect(attempts.count).toBe(2)
    expect(mod.default).toEqual({ name: 'QuickPublishModalStub', template: '<div />' })
  })
})
