<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { Gauge, Star } from '@lucide/vue'

// 评分仪表卡：均分渐变环 + 五档星条（同色系醇度阶梯）。
// 桌面端（×1）置于详情页右侧信息栏顶部，移动端以 xl:hidden 实例挂在页头下方；
// 双实例由父页控制，本组件只负责呈现（纯 HTML/CSS/SVG，无交互）。
const props = defineProps<{
  ratingAvg: number | null
  reviewCount: number
  /** 各档计数，index 0 = 1 星 … index 4 = 5 星。 */
  distribution: number[]
}>()

const { t } = useI18n()

// useId 保证双实例下 SVG 渐变 defs 的 id 唯一（同页两个副本同时渲染）。
const gradientId = `gf-rating-arc-${useId()}`

const RING_CENTER = 52
const RING_RADIUS = 46
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS

const ratio = computed(() => {
  if (props.ratingAvg == null || props.ratingAvg <= 0) return 0
  return Math.min(1, props.ratingAvg / 5)
})

// 均分星级簇：逐星半星粒度取整（round 到 0.5），每槽位输出 [0, 1] 的填充比例，
// 避免整层按百分比裁切时因星间 gap 累积错位（缺星被切成怪异碎片）。
// ratingAvg 为 0~5 分制，即 5 颗星槽位的填充单位总和。
const starFillUnits = computed(() => {
  if (props.ratingAvg == null || props.ratingAvg <= 0) return [0, 0, 0, 0, 0]
  const rounded = Math.round(props.ratingAvg * 2) / 2
  return [0, 1, 2, 3, 4].map((index) => Math.min(Math.max(rounded - index, 0), 1))
})

const arcOffset = computed(() => RING_CIRCUMFERENCE * (1 - ratio.value))

// 弧末端光点的圆上坐标（角度从顶部 -90° 起顺时针）。
const endDot = computed(() => {
  const angle = ratio.value * 2 * Math.PI - Math.PI / 2
  return {
    x: RING_CENTER + RING_RADIUS * Math.cos(angle),
    y: RING_CENTER + RING_RADIUS * Math.sin(angle),
  }
})

const distributionMax = computed(() => Math.max(...props.distribution, 1))

const rows = computed(() =>
  [5, 4, 3, 2, 1].map((star) => ({
    star,
    count: props.distribution[star - 1] ?? 0,
    widthPct: ((props.distribution[star - 1] ?? 0) / distributionMax.value) * 100,
  })),
)

// 5★ → 1★ 同色系醇度阶梯：高分满亮、低分窄暗，语义与量感同时传达。
const ROW_OPACITY = [0.95, 0.72, 0.5, 0.34, 0.24]

function rowOpacity(index: number) {
  return ROW_OPACITY[index] ?? 0.4
}
</script>

<template>
  <div
    class="gf-panel p-5"
    role="img"
    :aria-label="`${ratingAvg != null ? ratingAvg.toFixed(1) : '—'}${t('courseDetailPage.ratingOutOf')}，${t('courseDetailPage.reviewCountLabel', { count: reviewCount }, reviewCount)}`"
  >
    <div class="mb-4 flex items-center justify-between gap-2">
      <h2 class="inline-flex items-baseline gap-2 text-sm font-semibold text-base-content">
        <Gauge class="h-4 w-4 -translate-y-px self-center text-base-content/45" aria-hidden="true" />
        {{ t('courseDetailPage.ratingTitle') }}
        <span class="text-[11px] font-normal tabular-nums text-base-content/45">
          {{ t('courseDetailPage.reviewCountLabel', { count: reviewCount }, reviewCount) }}
        </span>
      </h2>
      <!-- 均分星级簇：逐星槽位填充（空星=描边淡灰、满星=实心 warning、半星=槽内裁切），
           半星粒度取整，无 gap 累积错位 -->
      <div class="relative flex shrink-0 gap-0.5" role="img" :aria-label="`${ratingAvg != null ? ratingAvg.toFixed(1) : '—'} / 5.0`">
        <span v-for="(fill, index) in starFillUnits" :key="index" class="relative inline-block h-3.5 w-3.5">
          <Star class="h-3.5 w-3.5 stroke-[1.5] text-base-content/25" fill="none" />
          <span
            v-if="fill > 0"
            class="absolute inset-0 overflow-hidden transition-[width] duration-300 ease-out"
            :style="{ width: `${fill * 100}%` }"
          >
            <Star class="h-3.5 w-3.5 text-warning" fill="currentColor" />
          </span>
        </span>
      </div>
    </div>

    <div class="flex items-center gap-5">
      <!-- 渐变仪表环：进度 = 均分 / 5 -->
      <div class="relative h-[104px] w-[104px] shrink-0" aria-hidden="true">
        <svg viewBox="0 0 104 104" class="h-full w-full -rotate-90">
          <defs>
            <linearGradient :id="gradientId" x1="0%" y1="100%" x2="100%" y2="0%">
              <stop offset="0%" stop-color="var(--gf-color-warning)" />
              <stop offset="100%" stop-color="var(--gf-color-primary)" />
            </linearGradient>
          </defs>
          <circle
            :cx="RING_CENTER"
            :cy="RING_CENTER"
            :r="RING_RADIUS"
            fill="none"
            stroke="var(--gf-color-base-300)"
            stroke-opacity="0.45"
            stroke-width="8"
          />
          <circle
            :cx="RING_CENTER"
            :cy="RING_CENTER"
            :r="RING_RADIUS"
            fill="none"
            :stroke="`url(#${gradientId})`"
            stroke-width="8"
            stroke-linecap="round"
            :stroke-dasharray="RING_CIRCUMFERENCE"
            :stroke-dashoffset="arcOffset"
          />
          <circle
            v-if="ratio > 0"
            :cx="endDot.x"
            :cy="endDot.y"
            r="4.5"
            fill="var(--gf-color-primary)"
            stroke="var(--gf-color-base-100)"
            stroke-width="2.5"
          />
        </svg>
        <div class="absolute inset-0 flex flex-col items-center justify-center gap-1">
          <Star class="h-3.5 w-3.5 text-warning" fill="currentColor" />
          <span class="text-[24px] font-bold leading-none tracking-tight tabular-nums text-base-content">
            {{ ratingAvg != null ? ratingAvg.toFixed(1) : '—' }}
          </span>
          <span class="text-[10px] leading-none text-base-content/45">{{ t('courseDetailPage.ratingOutOf') }}</span>
        </div>
      </div>

      <!-- 五档星条：宽度按数量比例，同色系醇度阶梯 -->
      <div class="min-w-0 flex-1 space-y-2">
        <div
          v-for="(row, index) in rows"
          :key="row.star"
          class="flex items-center gap-2"
          :aria-label="`${row.star} ${t('courseDetailPage.stars')} ${row.count} ${t('courseDetailPage.reviewCountLabel', { count: row.count }, row.count)}`"
        >
          <span class="inline-flex w-9 shrink-0 items-center gap-1 text-[11px] tabular-nums leading-none text-base-content/55">
            <Star class="h-3 w-3 fill-current text-base-content/30" />
            {{ row.star }}
          </span>
          <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-base-300/45">
            <div
              class="h-full rounded-full bg-warning transition-[width] duration-500 ease-out"
              :style="{ width: `${row.widthPct}%`, opacity: rowOpacity(index) }"
            />
          </div>
          <span class="w-5 shrink-0 text-right text-[11px] tabular-nums text-base-content/45">{{ row.count }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
