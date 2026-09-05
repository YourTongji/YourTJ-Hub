<script setup lang="ts">
import { computed, nextTick, ref, useAttrs, watch, type ComponentPublicInstance } from 'vue'
import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'
import { i18n } from '@/runtime/i18n'
import { Check, ChevronDown, X } from '@lucide/vue'
import {
  SelectContent,
  SelectItem,
  SelectItemIndicator,
  SelectItemText,
  SelectPortal,
  SelectRoot,
  SelectTrigger,
  SelectValue,
  SelectViewport,
} from 'reka-ui'

type SelectOption = {
  value: string
  label: string
}

const props = defineProps<{
  modelValue: string
  options: SelectOption[]
  placeholder?: string
  /** 字段名（如 t('schedule.major')）：作为 combobox 的可访问名称，
   *  避免 aria-label 覆盖外层 label 的字段语义（issue #227）。 */
  label?: string
  /** 启用后在下拉顶部显示本地选项过滤输入框。 */
  searchable?: boolean
  searchPlaceholder?: string
  emptyText?: string
  align?: 'start' | 'center' | 'end'
  /** 启用后当有选中项时在 hover 状态显示一键清除按钮（复用箭头位置，零额外宽度占用）。 */
  clearable?: boolean
  /** 清除按钮的无障碍标签 / title 提示文本。 */
  clearLabel?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

function t(key: string): string {
  try {
    const res = i18n.global.t(key)
    return typeof res === 'string' && res !== key ? res : 'Clear'
  } catch {
    return 'Clear'
  }
}

// SelectRoot 是 renderless 组件（inheritAttrs: false），调用方 attrs（如 class 间距/宽度）
// 必须显式透传到 SelectTrigger，否则被整体丢弃（SettingsPage 语言选择/字号预设回归）。
// class 先用 clsx 规范化 Vue 支持的 string/array/object 形式，再交 twMerge 合并：
// 调用方宽度（如 w-44）覆盖默认 w-full，避免 CSS 产物顺序导致覆盖失败。
defineOptions({ inheritAttrs: false })

// reka-ui@2.9.8 的 SelectContentImpl 对 Tab 无条件 preventDefault 且不关闭 Select
// （SelectContentImpl.js handleKeyDown），补明确的 Tab 关闭 + 焦点回到 trigger。
const open = ref(false)
const triggerRef = ref<ComponentPublicInstance | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const searchQuery = ref('')

const attrs = useAttrs()
const { class: callerClass, ...restAttrs } = attrs
const triggerClass = computed(() =>
  twMerge(
    'gf-input group flex w-full items-center justify-between gap-2 text-left cursor-pointer transition-colors duration-150 select-none',
    open.value ? 'border-primary ring-2 ring-primary/20' : '',
    clsx(callerClass as ClassValue),
  ),
)

const filteredOptions = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  if (!query) return props.options
  return props.options.filter((option) => option.label.toLocaleLowerCase().includes(query))
})

watch(open, (isOpen) => {
  if (!isOpen) {
    searchQuery.value = ''
    return
  }
  if (props.searchable) {
    void nextTick(() => searchInputRef.value?.focus())
  }
})

function handleSearchKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    event.stopPropagation()
    const firstOption = document.querySelector<HTMLElement>('[data-site-select-option]')
    firstOption?.focus()
    return
  }

  // 保留 Tab 给 SelectContent 的关闭和归还焦点逻辑，Escape 给 reka-ui 的 dismiss layer。
  if (event.key === 'Tab' || event.key === 'Escape') return

  // reka-ui 会把单字符按键解释为 Select 的 typeahead，进而把焦点移到选项上。
  // 搜索输入框应独占普通字符输入，组合键（复制、全选等）则维持浏览器默认行为。
  if (!event.ctrlKey && !event.altKey && !event.metaKey && event.key.length === 1) {
    event.stopPropagation()
  }
}
function handleContentKeydown(event: KeyboardEvent) {
  if (event.key !== 'Tab') return
  event.preventDefault()
  open.value = false
  // reka-ui 组件 ref 通过 useForwardExpose 暴露 $el，先取 DOM 元素再聚焦。
  void nextTick(() => {
    const el = triggerRef.value?.$el
    if (el instanceof HTMLElement) el.focus()
  })
}
</script>

<template>
  <SelectRoot
    v-model:open="open"
    :model-value="props.modelValue"
    @update:model-value="(value) => emit('update:modelValue', String(value))"
  >
    <SelectTrigger
      ref="triggerRef"
      v-bind="restAttrs"
      :class="triggerClass"
      :aria-label="props.label || undefined"
    >
      <!-- SelectValue 根 span 是 trigger(flex) 的子项：flex-1 撑开可用宽度，
           min-w-0 允许其收缩到内容宽度以下，否则长学期名会把 trigger 撑破溢出。 -->
      <SelectValue :placeholder="props.placeholder ?? ''" class="min-w-0 flex-1">
        <template #default="{ selectedLabel }">
          <!-- 内层文本 span 必须是 block：inline 元素上 text-overflow: ellipsis 不生效 -->
          <span
            class="block truncate"
            :class="selectedLabel.length ? 'text-base-content' : 'text-base-content/45'"
            :title="selectedLabel[0] ?? props.placeholder ?? ''"
          >
            {{ selectedLabel[0] ?? props.placeholder ?? '' }}
          </span>
        </template>
      </SelectValue>
      <!-- 右侧指示器：支持 clearable 时在 hover 时替换为清除图标，零额外宽度开销 -->
      <div v-if="props.clearable && props.modelValue" class="relative flex h-4 w-4 shrink-0 items-center justify-center">
        <button
          type="button"
          class="hidden group-hover:flex h-4 w-4 items-center justify-center rounded-sm text-base-content/45 hover:text-error transition-colors cursor-pointer"
          :title="props.clearLabel || t('schedule.clear')"
          :aria-label="props.clearLabel || t('schedule.clear')"
          @click.stop.prevent="emit('update:modelValue', '')"
        >
          <X class="h-3.5 w-3.5" />
        </button>
        <ChevronDown
          class="h-4 w-4 shrink-0 text-base-content/45 transition-transform duration-200 group-hover:hidden"
          :class="open ? 'rotate-180 text-primary' : ''"
        />
      </div>
      <ChevronDown
        v-else
        class="h-4 w-4 shrink-0 text-base-content/45 transition-transform duration-200"
        :class="open ? 'rotate-180 text-primary' : ''"
      />
    </SelectTrigger>

    <SelectPortal>
      <SelectContent
        class="gf-menu-surface z-[2100] min-w-[var(--reka-select-trigger-width)] max-w-[min(28rem,calc(100vw-2rem))] overflow-hidden rounded-xl border border-line/80 bg-base-100/98 p-1 shadow-2xl backdrop-blur-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-1.5 data-[side=top]:slide-in-from-bottom-1.5 duration-150 ease-out origin-[var(--reka-popper-transform-origin)]"
        position="popper"
        :side-offset="6"
        :collision-padding="8"
        :align="props.align ?? 'start'"
        :body-lock="false"
        :disable-outside-pointer-events="false"
        @keydown="handleContentKeydown"
      >
        <div v-if="props.searchable" class="border-b border-line/60 p-1">
          <input
            ref="searchInputRef"
            v-model="searchQuery"
            type="search"
            class="gf-input gf-input-sm w-full"
            :placeholder="props.searchPlaceholder ?? ''"
            :aria-label="props.searchPlaceholder ?? props.label ?? undefined"
            data-testid="site-select-search-input"
            @keydown="handleSearchKeydown"
          />
        </div>
        <div
          v-if="props.searchable && filteredOptions.length === 0"
          class="px-2.5 py-3 text-center text-sm text-base-content/55"
          data-testid="site-select-empty"
          role="status"
        >
          {{ props.emptyText ?? '' }}
        </div>
        <SelectViewport class="gf-scrollbar-thin max-h-[min(18rem,calc(var(--reka-popper-available-height,18rem)-1.5rem))] overflow-y-auto overscroll-contain">
          <SelectItem
            v-for="option in filteredOptions"
            :key="option.value"
            :value="option.value"
            :title="option.label"
            class="flex h-9 w-full cursor-pointer items-center gap-2 rounded-lg px-2.5 text-left text-sm font-medium text-base-content outline-none select-none transition-colors duration-150 hover:bg-base-200/80 data-[highlighted]:bg-primary/10 data-[highlighted]:text-primary"
            :class="option.value === props.modelValue ? 'bg-primary/10 text-primary font-semibold' : ''"
            data-site-select-option
          >
            <SelectItemText class="min-w-0 flex-1 truncate">{{ option.label }}</SelectItemText>
            <SelectItemIndicator>
              <Check class="h-4 w-4 shrink-0 text-primary" />
            </SelectItemIndicator>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
