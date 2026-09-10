import { ref } from 'vue'

export interface QuickPublishEditPayload {
  topicId: number
  contentType: 0 | 1 | 2 | 3
  title: string
  content: string
  categoryIds: number[]
  images?: string[]
}

// 首开锁存：弹层首次交互打开后保持 true（closeQuickPublish 不复位），让 AppShell
// 首次打开后一直挂载异步组件（匹配 PostStream 对 PostComposer 的「首次打开后
// 保持挂载」先例），退场动画与 250ms payload 清理语义不受懒加载影响；唯一例外
// 是 chunk 加载失败时复位（见 loadQuickPublishModal），允许下一次打开重新挂载重试。
const everOpenedQuickPublish = ref(false)

export function useEverOpenedQuickPublish() {
  return everOpenedQuickPublish
}

// 模块级缓存的动态 import：悬停/聚焦预热与正式打开共享同一个 promise，
// Vite 对相同 specifier 的动态 import 去重到同一 chunk，不会二次加载。
// 瞬态失败（网络抖动、发版替换旧 hash chunk 导致 404）时清空缓存并复位
// 首开锁存，让下一次交互重新发起 import 并重新挂载组件——否则被拒绝的
// promise 会被永久复用，发布入口在网络恢复后仍打不开，只能整页刷新。
let modalLoader: Promise<typeof import('@/site/components/QuickPublishModal.vue')> | null = null

export function loadQuickPublishModal() {
  if (!modalLoader) {
    const loader = import('@/site/components/QuickPublishModal.vue').catch((error) => {
      if (modalLoader === loader) {
        modalLoader = null
      }
      everOpenedQuickPublish.value = false
      throw error
    })
    modalLoader = loader
  }
  return modalLoader
}

const quickPublishOpen = ref(false)
const quickPublishType = ref<0 | 1 | 2>(0) // 0: regular, 1: question, 2: thought
const quickPublishEditPayload = ref<QuickPublishEditPayload | null>(null)

export function useQuickPublish() {
  function openQuickPublish(type: 0 | 1 | 2 = 0) {
    quickPublishEditPayload.value = null
    quickPublishType.value = type
    quickPublishOpen.value = true
    everOpenedQuickPublish.value = true
  }

  function openQuickPublishEdit(payload: QuickPublishEditPayload) {
    quickPublishEditPayload.value = payload
    quickPublishType.value = payload.contentType === 1 ? 1 : 2
    quickPublishOpen.value = true
    everOpenedQuickPublish.value = true
  }

  function closeQuickPublish() {
    quickPublishOpen.value = false
    // 延迟 250ms 在弹层完全退场淡出后再清空 payload，防止退场动画期间文案瞬间闪回“快速发布”
    setTimeout(() => {
      if (!quickPublishOpen.value) {
        quickPublishEditPayload.value = null
      }
    }, 250)
  }

  return {
    quickPublishOpen,
    quickPublishType,
    quickPublishEditPayload,
    openQuickPublish,
    openQuickPublishEdit,
    closeQuickPublish,
  }
}
