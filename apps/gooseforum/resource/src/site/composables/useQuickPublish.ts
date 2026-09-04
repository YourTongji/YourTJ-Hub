import { ref } from 'vue'

export interface QuickPublishEditPayload {
  topicId: number
  contentType: 0 | 1 | 2 | 3
  title: string
  content: string
  categoryIds: number[]
  images?: string[]
}

const quickPublishOpen = ref(false)
const quickPublishType = ref<0 | 1 | 2>(0) // 0: regular, 1: question, 2: thought
const quickPublishEditPayload = ref<QuickPublishEditPayload | null>(null)

export function useQuickPublish() {
  function openQuickPublish(type: 0 | 1 | 2 = 0) {
    quickPublishEditPayload.value = null
    quickPublishType.value = type
    quickPublishOpen.value = true
  }

  function openQuickPublishEdit(payload: QuickPublishEditPayload) {
    quickPublishEditPayload.value = payload
    quickPublishType.value = payload.contentType === 1 ? 1 : 2
    quickPublishOpen.value = true
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
