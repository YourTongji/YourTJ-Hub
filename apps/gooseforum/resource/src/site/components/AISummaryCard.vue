<script setup lang="ts">
import { ref, watch } from 'vue'
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
let debounceTimer: ReturnType<typeof setTimeout> | undefined

// 展开/收起触发防抖加载：快速点击不重复请求；已生成（cached/generated）与
// disabled 视为终态不重复请求（仅 refresh 按钮强制重取）。
watch(expanded, (open) => {
  if (!open) {
    // 折叠时取消挂起的防抖请求，避免卡片已关闭仍触发生成/消耗限流。
    clearTimeout(debounceTimer)
    debounceTimer = undefined
    return
  }
  if (
    lastStatus.value === 'cached' ||
    lastStatus.value === 'generated' ||
    lastStatus.value === 'disabled'
  )
    return
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(load, 500)
})

async function load(refresh = false) {
  clearTimeout(debounceTimer)
  debounceTimer = undefined
  if (state.value.kind === 'loading') return
  state.value = { kind: 'loading' }
  let result: CourseSummaryResult
  try {
    result = await getCourseSummary(props.courseId, refresh)
  } catch {
    state.value = { kind: 'error' }
    return
  }
  lastStatus.value = result.status
  switch (result.status) {
    case 'cached':
    case 'generated':
      if (result.summary) {
        state.value = { kind: 'ready', payload: result.summary, generatedAt: result.generatedAt, model: result.model }
      } else {
        state.value = { kind: 'error' }
      }
      break
    case 'insufficient_data':
      state.value = { kind: 'insufficient' }
      break
    case 'disabled':
      state.value = { kind: 'disabled' }
      break
    case 'rateLimited':
      state.value = { kind: 'rateLimited', retryAfterSeconds: result.retryAfterSeconds }
      break
    default:
      state.value = { kind: 'error' }
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
    <button
      type="button"
      class="flex w-full items-center justify-between gap-2 px-4 py-3 text-left"
      :aria-expanded="expanded"
      @click="onToggle"
    >
      <span class="inline-flex items-center gap-2 text-sm font-semibold text-base-content">
        <Sparkles class="h-4 w-4 text-primary" />
        {{ t('courseSummary.title') }}
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span v-if="state.kind === 'ready'" class="gf-badge gf-badge-muted text-xs">
          {{ t('courseSummary.generated') }}
        </span>
        <ChevronDown class="h-4 w-4 text-base-content/45 transition-transform" :class="{ 'rotate-180': expanded }" />
      </span>
    </button>

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

        <div class="mt-3 flex items-center justify-between gap-2">
          <p class="text-xs leading-relaxed text-base-content/40">
            {{ t('courseSummary.disclaimer') }}
          </p>
          <button
            type="button"
            class="gf-button gf-button-ghost gf-button-sm shrink-0"
            @click="onRefresh"
          >
            <RefreshCw class="h-3.5 w-3.5" />
            {{ t('courseSummary.refresh') }}
          </button>
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
