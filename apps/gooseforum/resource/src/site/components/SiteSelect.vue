<script setup lang="ts">
import { computed, nextTick, ref, useAttrs, type ComponentPublicInstance } from 'vue'
import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'
import { Check, ChevronDown } from '@lucide/vue'
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
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

// SelectRoot 是 renderless 组件（inheritAttrs: false），调用方 attrs（如 class 间距/宽度）
// 必须显式透传到 SelectTrigger，否则被整体丢弃（SettingsPage 语言选择/字号预设回归）。
// class 先用 clsx 规范化 Vue 支持的 string/array/object 形式，再交 twMerge 合并：
// 调用方宽度（如 w-44）覆盖默认 w-full，避免 CSS 产物顺序导致覆盖失败。
defineOptions({ inheritAttrs: false })

const attrs = useAttrs()
const { class: callerClass, ...restAttrs } = attrs
const triggerClass = computed(() =>
  twMerge('gf-input flex w-full items-center justify-between gap-2 text-left', clsx(callerClass as ClassValue)),
)

// reka-ui@2.9.8 的 SelectContentImpl 对 Tab 无条件 preventDefault 且不关闭 Select
// （SelectContentImpl.js handleKeyDown），补明确的 Tab 关闭 + 焦点回到 trigger。
const open = ref(false)
const triggerRef = ref<ComponentPublicInstance | null>(null)

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
      <SelectValue :placeholder="props.placeholder ?? ''">
        <template #default="{ selectedLabel }">
          <span
            class="min-w-0 truncate"
            :class="selectedLabel.length ? 'text-base-content' : 'text-base-content/45'"
          >
            {{ selectedLabel[0] ?? props.placeholder ?? '' }}
          </span>
        </template>
      </SelectValue>
      <ChevronDown class="h-4 w-4 shrink-0 text-base-content/45" />
    </SelectTrigger>

    <SelectPortal>
      <SelectContent
        class="gf-menu-surface z-[2100] min-w-[var(--reka-select-trigger-width)] overflow-hidden p-1"
        position="popper"
        :side-offset="6"
        align="start"
        :body-lock="false"
        :disable-outside-pointer-events="false"
        @keydown="handleContentKeydown"
      >
        <SelectViewport class="gf-scrollbar-thin max-h-64 overflow-y-auto overscroll-contain">
          <SelectItem
            v-for="option in props.options"
            :key="option.value"
            :value="option.value"
            class="flex h-9 w-full cursor-pointer items-center gap-2 rounded-md px-2.5 text-left text-sm font-medium text-base-content outline-none select-none hover:bg-base-200 data-[highlighted]:bg-primary/10 data-[highlighted]:text-primary"
            :class="option.value === props.modelValue ? 'bg-primary/10 text-primary' : ''"
          >
            <SelectItemText class="min-w-0 flex-1 truncate">{{ option.label }}</SelectItemText>
            <SelectItemIndicator>
              <Check class="h-4 w-4 shrink-0" />
            </SelectItemIndicator>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
