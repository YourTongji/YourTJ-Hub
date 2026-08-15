<script setup lang="ts">
// 无限滚动页脚：sentinel 进入视口自动触发 load-more，同时保留手动按钮兜底。
// 复用论坛首页帖子流的 IntersectionObserver 模式（rootMargin 480px 预加载）。
// 错误态停止自动触发，仅提供手动重试，避免请求风暴。
import { onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loader2 } from '@lucide/vue'

const props = defineProps<{
  /** 是否还有下一页可加载。 */
  hasNext: boolean
  /** 是否加载中（由页面传入，双保险防重复请求）。 */
  loading: boolean
  /** 加载错误文案；为空则不渲染错误态（错误由页面其它位置展示）。 */
  error: string
  /** 是否已有数据（决定「已全部加载」提示是否展示）。 */
  hasItems: boolean
  /** 手动兜底按钮文案，默认 t('common.loadMore')。 */
  loadLabel?: string
}>()

const emit = defineEmits<{
  loadMore: []
}>()

const { t } = useI18n()
const sentinel = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | undefined

function observe() {
  observer?.disconnect()
  if (!props.hasNext || props.loading || props.error || !sentinel.value || !('IntersectionObserver' in window)) return
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) emit('loadMore')
    },
    { rootMargin: '480px 0px' },
  )
  observer.observe(sentinel.value)
}

watch(
  () => [props.hasNext, props.loading, props.error] as const,
  () => observe(),
)

onMounted(() => observe())
onActivated(() => observe())
onDeactivated(() => observer?.disconnect())
onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <footer class="border-t border-line bg-base-200/50 p-3 text-center">
    <!-- 加载中 -->
    <p v-if="loading" class="inline-flex items-center gap-2 text-sm text-base-content/60">
      <Loader2 class="h-4 w-4 animate-spin" />
      {{ t('common.loadingShort') }}
    </p>
    <!-- 加载失败：停止自动触发，仅手动重试 -->
    <div v-else-if="error" class="space-y-1">
      <p class="text-xs text-error">{{ error }}</p>
      <button type="button" class="gf-button gf-button-sm gf-button-ghost" @click="emit('loadMore')">
        {{ t('common.retry') }}
      </button>
    </div>
    <!-- 还有更多：sentinel + 手动兜底按钮 -->
    <div v-else-if="hasNext">
      <div ref="sentinel" aria-hidden="true" class="h-px" />
      <button type="button" class="gf-button gf-button-sm gf-button-ghost" :disabled="loading" @click="emit('loadMore')">
        {{ loadLabel ?? t('common.loadMore') }}
      </button>
    </div>
    <!-- 已全部加载 -->
    <p v-else-if="hasItems" class="text-xs font-medium text-base-content/55">{{ t('common.allShown') }}</p>
  </footer>
</template>
