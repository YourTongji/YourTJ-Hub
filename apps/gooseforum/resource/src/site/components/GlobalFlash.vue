<script setup lang="ts">
import { computed } from 'vue'
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
 * 退场粒子弥散：彗星环收卷殆尽时，类型色微粒从环位迸发、向四周飘散渐隐。
 * 装饰性动画，reduced-motion 下不生成；粒子 fixed 于视口，飞离卡片边界无裁切。
 */
function spawnParticles(accent: string, originX: number, originY: number) {
  if (!accent || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return

  const particleCount = 12
  for (let i = 0; i < particleCount; i++) {
    const particle = document.createElement('span')
    particle.setAttribute('aria-hidden', 'true')
    const size = i % 3 === 0 ? '4.5' : (2.5 + Math.random() * 2).toFixed(1)
    particle.style.cssText = [
      `position:fixed`,
      `left:${originX}px`,
      `top:${originY}px`,
      `width:${size}px`,
      `height:${size}px`,
      'border-radius:9999px',
      `background:${accent}`,
      'pointer-events:none',
      'z-index:200',
    ].join(';')
    document.body.appendChild(particle)

    // 七成粒子向上半区飘散（呼应上浮退场），其余全向
    const upward = Math.random() < 0.7
    const angle = upward
      ? -Math.PI * (0.25 + Math.random() * 0.5)
      : Math.random() * Math.PI * 2
    const distance = (upward ? 52 + Math.random() * 48 : 30 + Math.random() * 34)
    const dx = Math.cos(angle) * distance
    const dy = Math.sin(angle) * distance
    const duration = 460 + Math.random() * 220

    particle
      .animate(
        [
          { opacity: 1, transform: 'translate(-50%,-50%) scale(1)' },
          { opacity: 0, transform: `translate(calc(-50% + ${dx}px), calc(-50% + ${dy}px)) scale(0.4)` },
        ],
        { duration, easing: 'cubic-bezier(0.16, 1, 0.3, 1)', fill: 'forwards' },
      )
      .finished.then(() => particle.remove())
  }
}

/**
 * 挂载时（正常布局）捕获彗星环中心坐标，供退场粒子使用。
 * 退场瞬间元素会转为 absolute，此时读取的 rect 不可靠。
 */
function captureBannerOrigin(_id: number, el: Element | null) {
  if (!el) return
  const r = el.getBoundingClientRect()
  const originX = r.left + r.width / 2
  const originY = r.bottom - 19 // 彗星环中心
  ;(el as HTMLElement).dataset.origin = `${originX},${originY}`
}

/** TransitionGroup 退场钩子：消散中段迸发粒子，随后结束退场 */
function onFlashLeave(el: Element, done: () => void) {
  const banner = el as HTMLElement
  const accent = getComputedStyle(banner).getPropertyValue('--gf-flash-accent').trim()
  const [originX, originY] = (banner.dataset.origin ?? '0,0').split(',').map(Number)
  window.setTimeout(() => spawnParticles(accent, originX, originY), 140)
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
