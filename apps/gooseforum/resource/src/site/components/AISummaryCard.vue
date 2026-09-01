<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown, Loader2, RefreshCw, Sparkles } from '@lucide/vue'
import { getCourseSummary, type CourseSummaryPayload, type CourseSummaryResult, type CourseSummaryStatus } from '@/runtime/api'

const props = defineProps<{
  courseId: number
}>()

const { t } = useI18n()

type CardState =
  | { kind: 'idle' }
  | { kind: 'loading' }
  | { kind: 'ready'; payload: CourseSummaryPayload; generatedAt?: string; model?: string }
  | { kind: 'insufficient' }
  | { kind: 'disabled' }
  | { kind: 'rateLimited'; retryAfterSeconds?: number }
  | { kind: 'error' }

const expanded = ref(false)
const state = ref<CardState>({ kind: 'idle' })
const lastStatus = ref<CourseSummaryStatus | null>(null)
// refreshing 独立于 state.kind：ready 态刷新时 state 仍为 ready（保留内容），
// 但 refreshing 阻止重叠请求（review P2：双击/连点刷新会并发消耗全局生成配额）。
const refreshing = ref(false)
// refreshNotice 刷新失败/限流的瞬态提示（review：keepContent 保留旧内容时，
// state 不切换，必须用独立 ref 给用户反馈）。
const refreshNotice = ref<{ kind: 'error' } | { kind: 'rateLimited'; retryAfterSeconds?: number } | null>(null)
let debounceTimer: ReturnType<typeof setTimeout> | undefined

// 终态：已生成 / 已禁用，展开时不重复请求。
function isTerminal(status: CourseSummaryStatus | null): boolean {
  return status === 'cached' || status === 'generated' || status === 'disabled'
}

// 挂载时 check 预检（只读、不消耗限流）：已有总结 → 自动展开展示；
// insufficient/none → 保持折叠（none 首次展开才触发生成）；disabled → 不渲染。
// 网络失败（fetch reject）保持 idle：折叠卡片静默失败可接受（review 建议），
// 但必须捕获避免 unhandled rejection。
onMounted(async () => {
  try {
    const result = await getCourseSummary(props.courseId, false, true)
    lastStatus.value = result.status
    switch (result.status) {
      case 'cached':
        if (result.summary) {
          state.value = { kind: 'ready', payload: result.summary, generatedAt: result.generatedAt, model: result.model }
          expanded.value = true
        } else {
          // 行存在但 payload 为空/损坏：与后端 CheckAiSummary 的 none 语义一致，
          // 保持折叠，首次展开走生成流程（review nit：统一无内容语义）。
          state.value = { kind: 'idle' }
        }
        break
      case 'insufficient_data':
        state.value = { kind: 'insufficient' }
        break
      case 'disabled':
        state.value = { kind: 'disabled' }
        break
      case 'none':
        // 从未生成过：保持折叠，首次展开才触发生成。
        state.value = { kind: 'idle' }
        break
      default:
        state.value = { kind: 'idle' }
    }
  } catch {
    // 预检网络失败：保持折叠（idle），避免 unhandled rejection。
    state.value = { kind: 'idle' }
  }
})

// 展开/收起触发防抖加载：快速点击不重复请求；终态不重复请求（仅 refresh 按钮强制重取）。
watch(expanded, (open) => {
  if (!open) {
    // 折叠时取消挂起的防抖请求，避免卡片已关闭仍触发生成/消耗限流。
    clearTimeout(debounceTimer)
    debounceTimer = undefined
    return
  }
  if (isTerminal(lastStatus.value)) return
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => load(false), 500)
})

// applyResult 把服务端结果应用到卡片状态。
// keepContent=true 时仅对 error/rateLimited 保留已有内容，并通过 refreshNotice
// 给出瞬态提示（review：保留旧内容不等于静默——用户必须看到刷新失败/限流）；
// 成功结果（含 insufficient_data——评价被隐藏/删除后刷新）必须替换旧内容
// （review P2：不能继续展示引用已不可见评价的过期摘要）。
function applyResult(result: CourseSummaryResult, keepContent = false) {
  lastStatus.value = result.status
  switch (result.status) {
    case 'cached':
    case 'generated':
      if (result.summary) {
        state.value = { kind: 'ready', payload: result.summary, generatedAt: result.generatedAt, model: result.model }
        expanded.value = true
      } else if (!keepContent) {
        state.value = { kind: 'error' }
      }
      refreshNotice.value = null
      break
    case 'insufficient_data':
      // 成功语义：无论之前是否有内容，都切换为 insufficient（替换过期摘要）。
      state.value = { kind: 'insufficient' }
      refreshNotice.value = null
      break
    case 'disabled':
      state.value = { kind: 'disabled' }
      refreshNotice.value = null
      break
    case 'rateLimited':
      if (keepContent) {
        refreshNotice.value = { kind: 'rateLimited', retryAfterSeconds: result.retryAfterSeconds }
      } else {
        state.value = { kind: 'rateLimited', retryAfterSeconds: result.retryAfterSeconds }
        refreshNotice.value = null
      }
      break
    default:
      if (keepContent) {
        refreshNotice.value = { kind: 'error' }
      } else {
        state.value = { kind: 'error' }
        refreshNotice.value = null
      }
  }
}

async function load(refresh = false) {
  clearTimeout(debounceTimer)
  debounceTimer = undefined
  if (refreshing.value) return
  // 已有内容时刷新：保留内容，仅更新状态（失败/限流不丢旧内容）。
  const keepContent = state.value.kind === 'ready'
  if (!keepContent) state.value = { kind: 'loading' }
  refreshing.value = true
  try {
    const result = await getCourseSummary(props.courseId, refresh)
    applyResult(result, keepContent)
  } catch {
    // 网络错误：必须退出 loading（review P2：否则折叠再展开被 loading guard
    // 挡住，卡片永远停留在骨架屏）；keepContent 时同样给出瞬态提示
    // （review：保留旧内容不等于静默）。
    if (keepContent) {
      refreshNotice.value = { kind: 'error' }
    } else {
      state.value = { kind: 'error' }
      refreshNotice.value = null
    }
  } finally {
    refreshing.value = false
  }
}

function onRefresh() {
  load(true)
}

function onToggle() {
  expanded.value = !expanded.value
}

// 五档口碑徽章（后端 consensus 枚举映射到本地化文案）
type ConsensusLevel = 'strong_recommend' | 'recommend' | 'neutral' | 'cautious' | 'not_recommend'

const consensusLevels: ConsensusLevel[] = ['strong_recommend', 'recommend', 'neutral', 'cautious', 'not_recommend']

function consensusLevel(payload: CourseSummaryPayload): ConsensusLevel | undefined {
  return consensusLevels.find((level) => payload.consensus === level)
}

function consensusBadgeClass(payload: CourseSummaryPayload): string {
  switch (consensusLevel(payload)) {
    case 'strong_recommend':
      return 'gf-badge gf-badge-success'
    case 'recommend':
      return 'gf-badge gf-badge-info'
    case 'neutral':
      return 'gf-badge gf-badge-muted'
    case 'cautious':
      return 'gf-badge gf-badge-warning'
    case 'not_recommend':
      return 'gf-badge gf-badge-error'
    default:
      return 'gf-badge gf-badge-muted'
  }
}

function sentimentLabel(sentiment: string): string {
  switch (sentiment) {
    case 'positive':
      return t('courseSummary.sentimentPositive')
    case 'negative':
      return t('courseSummary.sentimentNegative')
    default:
      return t('courseSummary.sentimentNeutral')
  }
}

function sentimentBadgeClass(sentiment: string): string {
  switch (sentiment) {
    case 'positive':
      return 'gf-badge gf-badge-success'
    case 'negative':
      return 'gf-badge gf-badge-error'
    default:
      return 'gf-badge gf-badge-muted'
  }
}
</script>

<template>
  <!-- 功能未启用时不渲染整个卡片（避免空 accordion 与重复请求） -->
  <section v-if="state.kind !== 'disabled'" class="gf-panel">
    <div class="flex w-full items-center justify-between gap-2 px-4 py-3">
      <button
        type="button"
        class="flex min-w-0 flex-1 items-center justify-between gap-2 text-left"
        :aria-expanded="expanded"
        @click="onToggle"
      >
        <span class="inline-flex items-center gap-2 text-sm font-semibold text-base-content">
          <Sparkles class="h-4 w-4 text-primary" />
          {{ t('courseSummary.title') }}
        </span>
        <span class="inline-flex shrink-0 items-center gap-1.5">
          <span v-if="state.kind === 'ready'" class="gf-badge gf-badge-muted text-xs">
            {{ t('courseSummary.generated') }}
          </span>
          <ChevronDown class="h-4 w-4 text-base-content/45 transition-transform" :class="{ 'rotate-180': expanded }" />
        </span>
      </button>
      <!-- 右上角手动刷新：已有内容时显示；刷新期间禁用防重叠（review P2） -->
      <button
        v-if="state.kind === 'ready'"
        type="button"
        class="gf-button gf-button-ghost gf-button-sm shrink-0"
        :class="{ 'pointer-events-none opacity-50': refreshing }"
        :title="t('courseSummary.refresh')"
        :aria-label="t('courseSummary.refresh')"
        :disabled="refreshing"
        @click.stop="onRefresh"
      >
        <RefreshCw class="h-3.5 w-3.5" :class="{ 'motion-safe:animate-spin': refreshing }" />
      </button>
    </div>

    <div v-if="expanded" class="border-t border-line/70 px-4 py-3">
      <!-- loading 骨架 -->
      <div v-if="state.kind === 'loading'" class="space-y-3">
        <div class="h-4 w-3/4 motion-safe:animate-pulse rounded bg-base-300/70" />
        <div class="h-4 w-full motion-safe:animate-pulse rounded bg-base-300/70" />
        <div class="h-4 w-1/2 motion-safe:animate-pulse rounded bg-base-300/70" />
      </div>

      <!-- 就绪：五档徽章 + 关键词 + pros/cons + 代表性评价 -->
      <div v-else-if="state.kind === 'ready'">
        <div class="mb-3 flex flex-wrap items-center gap-2">
          <span :class="consensusBadgeClass(state.payload)">
            {{ t(`courseSummary.consensus.${consensusLevel(state.payload) ?? 'neutral'}`) }}
          </span>
          <span v-if="state.generatedAt" class="text-xs text-base-content/45">
            {{ t('courseSummary.generatedAt', { time: state.generatedAt }) }}
          </span>
        </div>

        <p class="mb-3 text-sm leading-relaxed text-base-content/85">{{ t(`courseSummary.consensusText.${consensusLevel(state.payload) ?? 'neutral'}`) }}</p>

        <div v-if="state.payload.keywords?.length" class="mb-3 flex flex-wrap items-center gap-1.5">
          <span class="text-[12px] text-base-content/55">{{ t('courseSummary.keywords') }}：</span>
          <span
            v-for="kw in state.payload.keywords.slice(0, 5)"
            :key="kw"
            class="rounded-full border border-line/70 bg-base-200/60 px-2 py-0.5 text-xs text-base-content/70"
          >
            {{ kw }}
          </span>
        </div>

        <div class="mb-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div v-if="state.payload.pros?.length">
            <h4 class="mb-1.5 text-[12px] font-semibold text-success">{{ t('courseSummary.pros') }}</h4>
            <ul class="space-y-1 text-[13px] text-base-content/80">
              <li v-for="item in state.payload.pros.slice(0, 4)" :key="item" class="flex gap-1.5">
                <span class="mt-0.5 text-success">+</span>{{ item }}
              </li>
            </ul>
          </div>
          <div v-if="state.payload.cons?.length">
            <h4 class="mb-1.5 text-[12px] font-semibold text-error">{{ t('courseSummary.cons') }}</h4>
            <ul class="space-y-1 text-[13px] text-base-content/80">
              <li v-for="item in state.payload.cons.slice(0, 4)" :key="item" class="flex gap-1.5">
                <span class="mt-0.5 text-error">−</span>{{ item }}
              </li>
            </ul>
          </div>
        </div>

        <div v-if="state.payload.representativeReviews?.length" class="space-y-2">
          <h4 class="text-[12px] font-semibold text-base-content/70">{{ t('courseSummary.representativeReviews') }}</h4>
          <div
            v-for="(review, index) in state.payload.representativeReviews.slice(0, 3)"
            :key="index"
            class="rounded-[var(--gf-radius-box)] border border-line/60 bg-base-200/40 p-3"
          >
            <div class="mb-1 flex items-center justify-between gap-2">
              <span :class="sentimentBadgeClass(review.sentiment)">{{ sentimentLabel(review.sentiment) }}</span>
            </div>
            <p class="text-[13px] leading-relaxed text-base-content/80">{{ review.excerpt }}</p>
          </div>
        </div>

        <div class="mt-3">
          <p class="text-xs leading-relaxed text-base-content/40">
            {{ t('courseSummary.disclaimer') }}
          </p>
        </div>

        <!-- 刷新失败/限流的瞬态提示：保留旧内容时 state 不切换，必须显式反馈
             （review：keepContent 不等于静默失败） -->
        <div v-if="refreshNotice" class="mt-3 flex items-center gap-1.5 text-xs">
          <p v-if="refreshNotice?.kind === 'rateLimited'" class="text-warning">
            {{ t('courseSummary.rateLimited', { seconds: refreshNotice.retryAfterSeconds ?? 0 }) }}
          </p>
          <p v-else class="text-error">
            {{ t('courseSummary.loadFailed') }}
          </p>
        </div>
      </div>

      <!-- 评价不足占位 -->
      <div v-else-if="state.kind === 'insufficient'" class="flex items-center gap-2 text-sm text-base-content/55">
        <Sparkles class="h-4 w-4 text-base-content/35" />
        {{ t('courseSummary.insufficientData') }}
      </div>

      <!-- 429 限流 -->
      <div v-else-if="state.kind === 'rateLimited'" class="flex items-center gap-2 text-sm text-base-content/55">
        <Loader2 class="h-4 w-4 motion-safe:animate-spin text-warning" />
        {{ t('courseSummary.rateLimited', { seconds: state.retryAfterSeconds ?? 0 }) }}
      </div>

      <!-- 失败态：不影响课程页主流程 -->
      <div v-else-if="state.kind === 'error'" class="flex items-center justify-between gap-2">
        <span class="text-sm text-base-content/55">{{ t('courseSummary.loadFailed') }}</span>
        <button type="button" class="gf-button gf-button-ghost gf-button-sm" @click="onRefresh">
          <RefreshCw class="h-3.5 w-3.5" />
          {{ t('courseSummary.retry') }}
        </button>
      </div>
    </div>
  </section>
</template>
