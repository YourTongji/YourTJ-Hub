<script setup lang="ts">
import { computed } from 'vue'
import type { ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { BadgeCheck, Info, OctagonX, TriangleAlert, X } from '@lucide/vue'
import {
  dismiss,
  pauseFlashDismiss,
  resumeFlashDismiss,
  useFlashMessages,
  type FlashMessageType,
} from '@/runtime/flash-message'

const { messages } = useFlashMessages()
const { t } = useI18n()

const visibleMessages = computed(() => messages.value)

/** 类型标签：短、可扫读，与正文形成层级 */
function typeLabel(type: FlashMessageType) {
  switch (type) {
    case 'success':
      return t('flash.type.success')
    case 'warning':
      return t('flash.type.warning')
    case 'error':
      return t('flash.type.error')
    default:
      return t('flash.type.info')
  }
}

function iconFor(type: FlashMessageType) {
  switch (type) {
    case 'success':
      return BadgeCheck
    case 'warning':
      return TriangleAlert
    case 'error':
      return OctagonX
    default:
      return Info
  }
}

function toneClass(type: FlashMessageType) {
  switch (type) {
    case 'success':
      return 'gf-flash-banner--success'
    case 'warning':
      return 'gf-flash-banner--warning'
    case 'error':
      return 'gf-flash-banner--error'
    default:
      return 'gf-flash-banner--info'
  }
}

function liveRole(type: FlashMessageType) {
  return type === 'error' ? 'alert' : 'status'
}

function onBannerEnter(id: number) {
  pauseFlashDismiss(id)
}

function onBannerLeave(id: number) {
  resumeFlashDismiss(id)
}

/**
 * Toast 退场溶解层：流光碎屑 + 波点剥落 + 柔光晕开。
 * 从卡片表面多点生成（非单点爆破），呼应 leave 上浮/回落方向。
 * 装饰性、一次性；reduced-motion 下跳过；fixed 于视口，不受卡片 overflow 裁切。
 */
interface BannerBounds {
  left: number
  top: number
  width: number
  height: number
}

function prefersReducedMotion(): boolean {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function isMobileViewport(): boolean {
  return window.matchMedia('(max-width: 639px)').matches
}

/** 解析挂载时缓存的卡片几何；失败则回退空边界 */
function readBannerBounds(raw: string | undefined): BannerBounds | null {
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as Partial<BannerBounds>
    if (
      typeof parsed.left !== 'number' ||
      typeof parsed.top !== 'number' ||
      typeof parsed.width !== 'number' ||
      typeof parsed.height !== 'number'
    ) {
      return null
    }
    return {
      left: parsed.left,
      top: parsed.top,
      width: parsed.width,
      height: parsed.height,
    }
  } catch {
    return null
  }
}

/** 创建一枚固定定位的装饰粒子，动画结束后自清理 */
function createDissolveParticle(styles: string): HTMLSpanElement {
  const particle = document.createElement('span')
  particle.className = 'gf-flash-dissolve-particle'
  particle.setAttribute('aria-hidden', 'true')
  particle.style.cssText = styles
  document.body.appendChild(particle)
  return particle
}

/**
 * 三层溶解：
 * 1) 波点 — 表面网格圆点，先浮出再轻漂淡出（主体）
 * 2) 流光 — 横向细丝曳尾，像表面流光被抽走
 * 3) 柔晕 — 大光斑扩开溶解，像雾在玻璃上散开
 *
 * Design read（design-taste + better-ui）：
 * 产品微交互 / 玻璃 Toast；VARIANCE≈4、MOTION≈4（退场比入场更静）。
 * 粒子贴表面剥落，不做爆炸式散射；exit 不抢注意力。
 */
function spawnDissolveEffects(accent: string, bounds: BannerBounds) {
  if (!accent || prefersReducedMotion()) return

  const mobile = isMobileViewport()
  // 桌面 Toast 在右上：轻上漂；移动端在底部：轻下沉。水平几乎不偏，避免「右缘碎屑」
  const verticalBias = mobile ? 1 : -1

  // 内缩取样：粒子从内容区表面起，而不是边框外
  const padX = Math.min(22, bounds.width * 0.1)
  const padY = Math.min(16, bounds.height * 0.14)
  const innerLeft = bounds.left + padX
  const innerTop = bounds.top + padY
  const innerWidth = Math.max(12, bounds.width - padX * 2)
  const innerHeight = Math.max(12, bounds.height - padY * 2)

  const easeOutSoft = 'cubic-bezier(0.16, 1, 0.3, 1)'
  const easeDissolve = 'cubic-bezier(0.22, 0.61, 0.36, 1)'

  // —— 层 1：波点（主体，像水珠/墨点从表面剥落）——
  const dotCount = 14
  for (let index = 0; index < dotCount; index++) {
    const column = index % 5
    const row = Math.floor(index / 5)
    const gridX = (column + 0.5) / 5
    const gridY = (row + 0.5) / 3
    const originX = innerLeft + (gridX + (Math.random() - 0.5) * 0.16) * innerWidth
    const originY = innerTop + (gridY + (Math.random() - 0.5) * 0.2) * innerHeight

    const size = 3.2 + Math.random() * 4.2
    const isSoftHalo = index % 3 === 0
    const delay = 12 + index * 14 + Math.random() * 28
    // 与 leave ~0.56s 对齐，略长一点让中段仍可见
    const duration = 620 + Math.random() * 220

    // 克制漂移：主要沿退场竖直方向，水平只微抖
    const driftX = (Math.random() - 0.5) * 14
    const driftY = verticalBias * (10 + Math.random() * 22) + (Math.random() - 0.5) * 8
    const midScale = isSoftHalo ? 1.15 : 1.05
    const endScale = isSoftHalo ? 0.2 : 0.35 + Math.random() * 0.2

    const particle = createDissolveParticle(
      [
        'position:fixed',
        `left:${originX}px`,
        `top:${originY}px`,
        `width:${size.toFixed(1)}px`,
        `height:${size.toFixed(1)}px`,
        'border-radius:9999px',
        `background:${accent}`,
        isSoftHalo
          ? `box-shadow:0 0 ${10 + Math.random() * 12}px color-mix(in oklab, ${accent} 70%, transparent), 0 0 2px ${accent}`
          : `box-shadow:0 0 6px color-mix(in oklab, ${accent} 55%, transparent), 0 0 1px ${accent}`,
        'pointer-events:none',
        'z-index:200',
        'will-change:transform,opacity',
      ].join(';'),
    )

    particle
      .animate(
        [
          {
            opacity: isSoftHalo ? 0.55 : 0.75,
            transform: 'translate(-50%,-50%) scale(0.75)',
            offset: 0,
          },
          {
            // 中段保持可读：先「浮出」再溶解，而非立刻飘走
            opacity: isSoftHalo ? 0.85 : 1,
            transform: `translate(calc(-50% + ${driftX * 0.25}px), calc(-50% + ${driftY * 0.25}px)) scale(${midScale})`,
            offset: 0.28,
          },
          {
            opacity: isSoftHalo ? 0.55 : 0.7,
            transform: `translate(calc(-50% + ${driftX * 0.65}px), calc(-50% + ${driftY * 0.65}px)) scale(${(midScale + endScale) / 2})`,
            offset: 0.62,
          },
          {
            opacity: 0,
            transform: `translate(calc(-50% + ${driftX}px), calc(-50% + ${driftY}px)) scale(${endScale})`,
            offset: 1,
          },
        ],
        { duration, delay, easing: easeDissolve, fill: 'forwards' },
      )
      .finished.then(() => particle.remove())
      .catch(() => particle.remove())
  }

  // —— 层 2：流光丝（横向曳尾，像表面流光被抽走）——
  const streakCount = 5
  for (let index = 0; index < streakCount; index++) {
    const originX = innerLeft + (0.12 + Math.random() * 0.76) * innerWidth
    const originY = innerTop + (0.22 + Math.random() * 0.5) * innerHeight
    // 横丝：宽 > 高，读作流光而非竖条
    const streakWidth = 22 + Math.random() * 30
    const streakHeight = 2 + Math.random() * 2.2
    const tilt = (mobile ? 12 : -18) + (Math.random() - 0.5) * 14
    const delay = 40 + index * 28 + Math.random() * 30
    const duration = 540 + Math.random() * 200

    const driftX = (mobile ? 0 : 1) * (8 + Math.random() * 16) + (Math.random() - 0.5) * 10
    const driftY = verticalBias * (14 + Math.random() * 22)

    const particle = createDissolveParticle(
      [
        'position:fixed',
        `left:${originX}px`,
        `top:${originY}px`,
        `width:${streakWidth.toFixed(1)}px`,
        `height:${streakHeight.toFixed(1)}px`,
        'border-radius:9999px',
        `background:linear-gradient(90deg, transparent 0%, color-mix(in oklab, ${accent} 55%, white) 28%, ${accent} 52%, color-mix(in oklab, white 40%, ${accent}) 72%, transparent 100%)`,
        `box-shadow:0 0 14px color-mix(in oklab, ${accent} 65%, transparent), 0 0 2px ${accent}`,
        'pointer-events:none',
        'z-index:201',
        'will-change:transform,opacity',
        'opacity:0',
      ].join(';'),
    )

    particle
      .animate(
        [
          {
            opacity: 0,
            transform: `translate(-50%,-50%) scaleX(0.45) rotate(${tilt}deg)`,
          },
          {
            opacity: 0.9,
            transform: `translate(calc(-50% + ${driftX * 0.2}px), calc(-50% + ${driftY * 0.2}px)) scaleX(1) rotate(${tilt}deg)`,
            offset: 0.2,
          },
          {
            opacity: 0.55,
            transform: `translate(calc(-50% + ${driftX * 0.55}px), calc(-50% + ${driftY * 0.55}px)) scaleX(0.85) rotate(${tilt + (mobile ? 4 : -6)}deg)`,
            offset: 0.55,
          },
          {
            opacity: 0,
            transform: `translate(calc(-50% + ${driftX}px), calc(-50% + ${driftY}px)) scaleX(0.3) rotate(${tilt + (mobile ? 8 : -10)}deg)`,
          },
        ],
        { duration, delay, easing: easeOutSoft, fill: 'forwards' },
      )
      .finished.then(() => particle.remove())
      .catch(() => particle.remove())
  }

  // —— 层 3：柔晕（大光斑扩开淡出，像雾/墨在玻璃上散开）——
  const glowCount = 4
  for (let index = 0; index < glowCount; index++) {
    const originX = innerLeft + (0.18 + index * 0.2 + (Math.random() - 0.5) * 0.08) * innerWidth
    const originY = innerTop + (0.28 + (Math.random() - 0.5) * 0.22) * innerHeight
    const size = 28 + Math.random() * 32
    const delay = 8 + index * 36
    const duration = 680 + Math.random() * 160
    const driftX = (Math.random() - 0.5) * 10
    const driftY = verticalBias * (6 + Math.random() * 12)

    const particle = createDissolveParticle(
      [
        'position:fixed',
        `left:${originX}px`,
        `top:${originY}px`,
        `width:${size.toFixed(1)}px`,
        `height:${size.toFixed(1)}px`,
        'border-radius:9999px',
        `background:radial-gradient(circle, color-mix(in oklab, ${accent} 70%, white) 0%, color-mix(in oklab, ${accent} 40%, transparent) 38%, color-mix(in oklab, ${accent} 16%, transparent) 62%, transparent 78%)`,
        'pointer-events:none',
        'z-index:199',
        'will-change:transform,opacity',
        'filter:blur(1.5px)',
      ].join(';'),
    )

    particle
      .animate(
        [
          {
            opacity: 0.4,
            transform: 'translate(-50%,-50%) scale(0.5)',
            offset: 0,
          },
          {
            opacity: 0.72,
            transform: `translate(calc(-50% + ${driftX * 0.3}px), calc(-50% + ${driftY * 0.3}px)) scale(1)`,
            offset: 0.3,
          },
          {
            opacity: 0,
            transform: `translate(calc(-50% + ${driftX}px), calc(-50% + ${driftY}px)) scale(1.85)`,
            offset: 1,
          },
        ],
        { duration, delay, easing: easeDissolve, fill: 'forwards' },
      )
      .finished.then(() => particle.remove())
      .catch(() => particle.remove())
  }
}

/**
 * 挂载时（正常布局）捕获整卡几何，供退场溶解层取样。
 * 退场瞬间元素会转为 absolute，此时读取的 rect 可能变窄/偏移。
 */
function captureBannerOrigin(_id: number, el: Element | ComponentPublicInstance | null) {
  if (!(el instanceof Element)) return
  const r = el.getBoundingClientRect()
  if (r.width < 2 || r.height < 2) return
  const bounds: BannerBounds = {
    left: r.left,
    top: r.top,
    width: r.width,
    height: r.height,
  }
  ;(el as HTMLElement).dataset.origin = JSON.stringify(bounds)
}

/** TransitionGroup 退场钩子：与卡片 leave 同步启动溶解层 */
function onFlashLeave(el: Element, done: () => void) {
  const banner = el as HTMLElement
  const accent = getComputedStyle(banner).getPropertyValue('--gf-flash-accent').trim()
  // 优先用挂载时缓存的稳定几何；live 仅在缓存缺失时兜底
  const cachedBounds = readBannerBounds(banner.dataset.origin)
  const liveRect = banner.getBoundingClientRect()
  const liveBounds: BannerBounds | null =
    liveRect.width >= 24 && liveRect.height >= 24
      ? {
          left: liveRect.left,
          top: liveRect.top,
          width: liveRect.width,
          height: liveRect.height,
        }
      : null
  // 若 live 明显比缓存窄（absolute 塌缩），仍用缓存
  const bounds =
    cachedBounds && liveBounds && liveBounds.width < cachedBounds.width * 0.75
      ? cachedBounds
      : (cachedBounds ?? liveBounds)
  if (!bounds) {
    done()
    return
  }

  // 略晚于 leave 起步：卡片先开始软化，再剥落波点（溶解而非爆炸）
  window.setTimeout(() => spawnDissolveEffects(accent, bounds), 40)
  done()
}

</script>

<template>
  <!--
    玻璃态 Toast 栈：桌面右上、移动端底部安全区。
    z 高于设置弹层，媒体校验提示始终可见。
  -->
  <div
    class="pointer-events-none fixed inset-x-0 bottom-[max(1rem,env(safe-area-inset-bottom))] z-[140] px-4 sm:inset-x-auto sm:bottom-auto sm:right-5 sm:top-[4.75rem] sm:px-0 lg:right-8"
    aria-live="polite"
    aria-relevant="additions text"
  >
    <TransitionGroup
      name="gf-flash"
      tag="div"
      class="gf-flash-stack mx-auto flex w-full max-w-[22.5rem] flex-col items-stretch gap-3.5 sm:mx-0 sm:max-w-[22rem] sm:items-end"
      @leave="onFlashLeave"
    >
      <div
        v-for="item in visibleMessages"
        :key="item.id"
        :ref="(el) => captureBannerOrigin(item.id, el)"
        class="gf-flash-banner pointer-events-auto relative w-full"
        :class="toneClass(item.type)"
        :role="liveRole(item.type)"
        :style="{ '--gf-flash-duration': `${item.durationMs}ms` }"
        @mouseenter="onBannerEnter(item.id)"
        @mouseleave="onBannerLeave(item.id)"
        @focusin="onBannerEnter(item.id)"
        @focusout="onBannerLeave(item.id)"
      >
        <div class="gf-flash-banner__body grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-x-3 px-3.5 pt-3.5 pb-3">
          <!-- 类型图标：无圆底容器，现代线条图标直接呈现 -->
          <span class="gf-flash-banner__icon mt-0.5 flex shrink-0" aria-hidden="true">
            <component
              :is="iconFor(item.type)"
              class="gf-flash-banner__glyph h-5 w-5"
              stroke-width="2"
            />
          </span>

          <div class="gf-flash-banner__copy min-w-0 pt-0.5">
            <p class="gf-flash-banner__label">{{ typeLabel(item.type) }}</p>
            <p class="gf-flash-banner__message">{{ item.message }}</p>
          </div>

          <button
            type="button"
            class="gf-flash-banner__close -mr-0.5 mt-0 inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full"
            :aria-label="t('flash.close')"
            @click="dismiss(item.id)"
          >
            <X class="h-3.5 w-3.5" stroke-width="2" aria-hidden="true" />
          </button>
        </div>

        <!-- 剩余时间：亏月彗星环，弧长即剩余时长 -->
        <div class="gf-flash-banner__moon" aria-hidden="true" />
      </div>
    </TransitionGroup>
  </div>
</template>
