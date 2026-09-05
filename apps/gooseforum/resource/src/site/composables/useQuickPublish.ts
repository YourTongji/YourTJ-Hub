import { ref } from 'vue'

export interface QuickPublishEditPayload {
  topicId: number
  contentType: 0 | 1 | 2 | 3
  title: string
  content: string
  categoryIds: number[]
  images?: string[]
}

// 模块级缓存的动态 import：悬停/聚焦预热与正式打开共享同一个 promise，
// Vite 对相同 specifier 的动态 import 去重到同一 chunk，不会二次加载。
let modalLoader: Promise<typeof import('@/site/components/QuickPublishModal.vue')> | null = null

export function loadQuickPublishModal() {
  if (!modalLoader) {
    modalLoader = import('@/site/components/QuickPublishModal.vue')
  }
  return modalLoader
}

// 首开锁存：弹层首次打开后保持 true、永不复位，让 AppShell 首次打开后一直挂载
// 异步组件（匹配 PostStream 对 PostComposer 的「首次打开后保持挂载」先例），
// 退场动画与 250ms payload 清理语义不受懒加载影响。
const everOpenedQuickPublish = ref(false)

export function useEverOpenedQuickPublish() {
  return everOpenedQuickPublish
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
