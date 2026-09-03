<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronLeft, ChevronRight } from '@lucide/vue'

export interface CategoryRailItem {
  id: number
  label: string
  url: string
  /** 后台配置的分类图标（emoji/短文本），空值回退为名称首字。 */
  icon?: string
  color: string
}

const props = defineProps<{
  categories: CategoryRailItem[]
  /** 当前激活分类（侧栏 activeKey 格式：category_<id>）。 */
  activeCategory?: string
}>()

const { t } = useI18n()

const railElement = ref<HTMLElement | null>(null)
const canScrollStart = ref(false)
const canScrollEnd = ref(false)
let scrollObserver: ResizeObserver | null = null

/**
 * 分类色驱动：icon 底用「原色 → 同色加深」的对角渐变（白字保证对比度），
 * active 外圈与文字用原色。颜色解析交给浏览器 color-mix，浅/深色主题通用。
 */
function categoryColorVar(color: string) {
  return { '--category-color': color || 'var(--gf-color-primary)' }
}

function isActive(item: CategoryRailItem) {
  return props.activeCategory === `category_${item.id}`
}

/** icon 内容：后台配置的 icon 优先；为空时回退名称首字（中文）或前两个字母（拉丁）。 */
function iconGlyph(item: CategoryRailItem) {
  const configured = (item.icon || '').trim()
  if (configured) return configured
  const label = item.label.trim()
  if (!label) return '?'
  return /^[\u4e00-\u9fa5]/.test(label) ? label.slice(0, 1) : label.slice(0, 2).toUpperCase()
}

/** 双缘渐隐遮罩：仅在对应方向仍有更多内容时显示，同时暗示可横向滚动。 */
const showStartFade = computed(() => canScrollStart.value)
const showEndFade = computed(() => canScrollEnd.value)

function updateScrollState() {
  const rail = railElement.value
  if (!rail) {
    canScrollStart.value = false
    canScrollEnd.value = false
    return
  }
  // 两个方向各自判断：留 1px 容差，滚到头后对应遮罩/箭头消失。
  canScrollStart.value = rail.scrollLeft > 1
  canScrollEnd.value =
    rail.scrollWidth > rail.clientWidth + 1 &&
    rail.scrollLeft + rail.clientWidth < rail.scrollWidth - 1
}

function onScroll() {
  updateScrollState()
}

/** 箭头按钮：按可视区约 70% 的步长平滑滚动。 */
function scrollByDirection(direction: 1 | -1) {
  const rail = railElement.value
  if (!rail) return
  rail.scrollBy({ left: direction * rail.clientWidth * 0.7, behavior: 'smooth' })
}

onMounted(() => {
  updateScrollState()
  const rail = railElement.value
  if (rail) {
    scrollObserver = new ResizeObserver(updateScrollState)
    scrollObserver.observe(rail)
  }
  // 当前分类不在可视区时自动滚到中间，让选中状态一眼可见。
  void nextTick(() => {
    const activeItem = rail?.querySelector<HTMLElement>('.gf-category-link-active')
    activeItem?.scrollIntoView({ inline: 'center', block: 'nearest' })
    updateScrollState()
  })
})

onBeforeUnmount(() => {
  scrollObserver?.disconnect()
})
</script>

<template>
    <div v-if="categories.length" class="gf-category-rail" role="navigation" :aria-label="t('shell.railLabel')">
    <div class="gf-category-rail-viewport">
      <ul
        ref="railElement"
        class="gf-category-rail-track"
        @scroll.passive="onScroll"
      >
        <li v-for="item in categories" :key="item.id" class="gf-category-item">
          <a
            :href="item.url"
            class="gf-category-link"
            :class="{ 'gf-category-link-active': isActive(item) }"
            :style="categoryColorVar(item.color)"
            :aria-current="isActive(item) ? 'page' : undefined"
          >
            <span class="gf-category-icon" aria-hidden="true">{{ iconGlyph(item) }}</span>
            <span class="gf-category-label">{{ item.label }}</span>
          </a>
        </li>
      </ul>
      <div v-if="showStartFade" class="gf-category-rail-fade gf-category-rail-fade-start" aria-hidden="true" />
      <div v-if="showEndFade" class="gf-category-rail-fade gf-category-rail-fade-end" aria-hidden="true" />
      <button
        v-if="canScrollStart"
        type="button"
        class="gf-category-rail-arrow gf-category-rail-arrow-start"
        :aria-label="t('shell.railScrollPrev')"
        @click="scrollByDirection(-1)"
      >
        <ChevronLeft class="h-5 w-5" />
      </button>
      <button
        v-if="canScrollEnd"
        type="button"
        class="gf-category-rail-arrow gf-category-rail-arrow-end"
        :aria-label="t('shell.railScrollNext')"
        @click="scrollByDirection(1)"
      >
        <ChevronRight class="h-5 w-5" />
      </button>
    </div>
  </div>
</template>

<style scoped>
/*
 * Rail 面板：与 toolbar / feed 卡片同栅格的内嵌背板。
 * 16px 外边距对齐 toolbar 的 px-4（sm+ 为 20px），避免「贴边裸排」的松散感。
 */
.gf-category-rail {
  position: relative;
  margin: 12px 16px;
  border: var(--gf-border) solid var(--gf-color-line);
  border-radius: var(--gf-radius-box);
  background: var(--gf-color-base-100);
  box-shadow: 0 2px 12px rgb(0 0 0 / calc(var(--gf-depth) * 0.04));
}

.gf-category-rail-viewport {
  position: relative;
  overflow: hidden;
  border-radius: inherit;
}

/*
 * 横向滚动轨道：禁止换行，滚轮/触控板原生横向浏览。
 * 隐藏滚动条但保留滚动能力；scroll-padding 与内边距一致，
 * 保证 scrollIntoView 后 active 项完整可见。
 */
.gf-category-rail-track {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  padding: 10px 14px;
  scrollbar-width: none;
  -ms-overflow-style: none;
  scroll-padding-inline: 14px;
}

.gf-category-rail-track::-webkit-scrollbar {
  display: none;
}

@media (min-width: 640px) {
  .gf-category-rail {
    margin: 14px 20px;
  }

  .gf-category-rail-track {
    gap: 9px;
    padding: 12px 18px;
    scroll-padding-inline: 18px;
  }
}

.gf-category-rail-fade {
  position: absolute;
  inset-block: 0;
  width: 52px;
  pointer-events: none;
}

.gf-category-rail-fade-start {
  left: 0;
  background: linear-gradient(to right, var(--gf-color-base-100), transparent);
}

.gf-category-rail-fade-end {
  right: 0;
  background: linear-gradient(to right, transparent, var(--gf-color-base-100));
}

/*
 * 边缘指示箭头（无容器裸箭头）：平时隐藏并隐藏命中，鼠标在 rail 上驻留后才浮现，
 * 避免扫过即闪造成误触；drop-shadow 让箭头在渐隐遮罩上浮起（无圆底/边框）。
 * 颜色走语义 token（默认 base-content，hover 强调 primary）；
 * 移动端依赖手势滑动，箭头仅 sm+ 显示。
 */
.gf-category-rail-arrow {
  display: none;
  position: absolute;
  top: 50%;
  z-index: 2;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--gf-color-base-content);
  filter: drop-shadow(0 1px 3px rgb(0 0 0 / 0.18));
  visibility: hidden;
  opacity: 0;
  transform: translateY(-50%);
  transition:
    opacity 150ms ease,
    transform 150ms ease,
    color 150ms ease,
    filter 150ms ease,
    visibility 0s linear 150ms;
}

.gf-category-rail-arrow-start {
  left: 6px;
  transform: translateY(-50%) translateX(-6px);
}

.gf-category-rail-arrow-end {
  right: 6px;
  transform: translateY(-50%) translateX(6px);
}

@media (min-width: 640px) {
  .gf-category-rail-arrow {
    display: inline-flex;
  }
}

/* 浮现：驻留 150ms 才滑入（防扫过误触），从所属一侧以 6px 滑入 + 淡入；收起反向淡出。 */
@media (hover: hover) {
  .gf-category-rail:hover .gf-category-rail-arrow,
  .gf-category-rail-arrow:hover {
    visibility: visible;
    opacity: 1;
    transform: translateY(-50%);
    transition-delay: 150ms;
  }
}

.gf-category-rail-arrow:focus-visible {
  visibility: visible;
  opacity: 1;
  transform: translateY(-50%);
  transition-delay: 0s;
  outline: 2px solid var(--gf-color-primary);
  outline-offset: 1px;
}

.gf-category-rail-arrow:hover {
  color: var(--gf-color-primary);
  filter: drop-shadow(0 2px 4px rgb(0 0 0 / 0.22));
}

.gf-category-rail-arrow:active {
  transform: translateY(-50%) scale(0.92);
}

@media (hover: hover) and (max-width: 639px) {
  .gf-category-rail:hover .gf-category-rail-arrow {
    visibility: hidden;
    opacity: 0;
  }
}

/*
 * 分区入口：左 icon 右文字的胶囊（参考站点头部快捷入口的横排样式），
 * 靠自身边框/圆角形成节奏，不再依赖大间距。
 */
.gf-category-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px 5px 6px;
  border-radius: 9999px;
  border: 1px solid transparent;
  background: color-mix(in srgb, var(--category-color) 6%, var(--gf-color-base-100));
  -webkit-tap-highlight-color: transparent;
  transition: background 150ms ease, border-color 150ms ease, box-shadow 150ms ease;
}

.gf-category-link:hover {
  border-color: color-mix(in srgb, var(--category-color) 30%, transparent);
  background: color-mix(in srgb, var(--category-color) 11%, var(--gf-color-base-100));
}

/*
 * active：同色描边 + 分类色 10% 底 + 轻投影，一眼可见且与渐变 icon 同源。
 */
.gf-category-link-active {
  border-color: color-mix(in srgb, var(--category-color) 50%, transparent);
  background: color-mix(in srgb, var(--category-color) 10%, var(--gf-color-base-100));
  box-shadow: 0 2px 10px -4px color-mix(in srgb, var(--category-color) 45%, transparent);
}

/*
 * icon 块：分类色对角渐变实底 + 白字（glyph 场景），emoji 直接原色显示；
 * 后台 icon 过长（>视口）时截断，不影响胶囊高度。
 */
.gf-category-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  min-width: 24px;
  max-width: 26px;
  overflow: hidden;
  border-radius: 9999px;
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
  color: rgb(255 255 255 / 0.96);
  background:
    linear-gradient(
      135deg,
      color-mix(in srgb, var(--category-color) 88%, white) 0%,
      var(--category-color) 45%,
      color-mix(in srgb, var(--category-color) 82%, black) 100%
    );
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.22),
    0 1px 2px rgb(0 0 0 / 0.08);
  transition: transform 150ms ease, box-shadow 150ms ease;
}

.gf-category-link:hover .gf-category-icon {
  transform: translateY(-1px) scale(1.03);
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.22),
    0 6px 16px -6px color-mix(in srgb, var(--category-color) 55%, transparent);
}

.gf-category-link:active .gf-category-icon {
  transform: translateY(0) scale(0.97);
}

.gf-category-link-active .gf-category-icon {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.22),
    0 4px 12px -4px color-mix(in srgb, var(--category-color) 40%, transparent);
}

.gf-category-label {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.25;
  color: var(--gf-color-base-content);
  opacity: 0.85;
  transition: color 150ms ease, opacity 150ms ease;
}

.gf-category-link:hover .gf-category-label {
  opacity: 1;
}

.gf-category-link-active .gf-category-label {
  color: var(--category-color);
  opacity: 1;
  font-weight: 600;
}

@media (prefers-reduced-motion: reduce) {
  .gf-category-icon,
  .gf-category-label,
  .gf-category-rail-arrow {
    transition: none;
  }

  .gf-category-link:hover .gf-category-icon {
    transform: none;
  }
}

/* 焦点可达性：键盘 Tab 时给出与 active 同级的可视反馈。 */
.gf-category-link:focus-visible {
  outline: 2px solid var(--gf-color-primary);
  outline-offset: 2px;
}
</style>
