<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useId } from 'vue'
import { Check, ChevronDown } from '@lucide/vue'

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

const listboxId = useId()
const open = ref(false)
const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
/** 列表展开时的焦点索引；默认聚焦当前选中项（无则首项）。 */
const highlightIndex = ref(-1)

const selectedOption = computed(() => props.options.find(option => option.value === props.modelValue))
const triggerLabel = computed(() => selectedOption.value?.label || props.placeholder || '')

function selectOption(value: string) {
  emit('update:modelValue', value)
  open.value = false
  // 选中后焦点回到触发按钮，符合 combobox 交互（Tab 继续导航表单）。
  trigger.value?.focus()
}

function openList(initialIndex?: number) {
  open.value = true
  const selectedIndex = props.options.findIndex((option) => option.value === props.modelValue)
  const base = selectedIndex >= 0 ? selectedIndex : 0
  highlightIndex.value = initialIndex ?? base
  // 打开后立即把焦点移入高亮 option（roving tabindex 模式），
  // 后续 ArrowDown/Enter 由 listbox handler 处理（issue #235 review P1）。
  requestAnimationFrame(() => focusOption(highlightIndex.value))
}

function focusOption(index: number) {
  root.value?.querySelectorAll<HTMLElement>('[role="option"]')[index]?.focus()
}

function handleDocumentPointerDown(event: PointerEvent) {
  const target = event.target
  if (target instanceof Node && root.value?.contains(target)) return
  open.value = false
}

function handleTriggerKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    open.value = false
    return
  }
  if (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    if (open.value) {
      // 焦点已移入 option（openList 后），此处仅在焦点被外部拉回 trigger 时兜底：
      // 重新聚焦当前高亮项，让列表键盘流程继续。
      focusOption(highlightIndex.value)
      return
    }
    openList()
    return
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    if (!open.value) {
      // 上方向键打开：高亮定位到当前选中项的前一项（已选中则取末项）。
      const selectedIndex = props.options.findIndex((option) => option.value === props.modelValue)
      const previous = (selectedIndex - 1 + props.options.length) % props.options.length
      openList(selectedIndex >= 0 ? previous : 0)
      return
    }
    event.stopPropagation()
    const next = (highlightIndex.value - 1 + props.options.length) % props.options.length
    highlightIndex.value = next
    focusOption(next)
  }
}

function handleListKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    open.value = false
    trigger.value?.focus()
    return
  }
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    const delta = event.key === 'ArrowDown' ? 1 : -1
    const next = (highlightIndex.value + delta + props.options.length) % props.options.length
    highlightIndex.value = next
    focusOption(next)
    return
  }
  if (event.key === 'Home') {
    event.preventDefault()
    highlightIndex.value = 0
    focusOption(0)
    return
  }
  if (event.key === 'End') {
    event.preventDefault()
    highlightIndex.value = props.options.length - 1
    focusOption(props.options.length - 1)
    return
  }
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    const option = props.options[highlightIndex.value]
    if (option) selectOption(option.value)
    return
  }
  if (event.key === 'Tab') {
    open.value = false
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown)
})
</script>

<template>
  <div ref="root" class="relative">
    <button
      ref="trigger"
      type="button"
      role="combobox"
      class="gf-input flex w-full items-center justify-between gap-2 text-left"
      aria-haspopup="listbox"
      :aria-expanded="open"
      :aria-controls="listboxId"
      :aria-activedescendant="open ? `${listboxId}-opt-${highlightIndex}` : undefined"
      :aria-label="props.label || triggerLabel"
      @click="open ? (open = false) : openList()"
      @keydown="handleTriggerKeydown"
    >
      <span class="min-w-0 truncate" :class="selectedOption ? 'text-base-content' : 'text-base-content/45'">
        {{ triggerLabel }}
      </span>
      <ChevronDown class="h-4 w-4 shrink-0 text-base-content/45 transition-transform" :class="{ 'rotate-180': open }" />
    </button>

    <Transition name="gf-menu">
      <div
        v-if="open"
        :id="listboxId"
        role="listbox"
        class="gf-menu-surface absolute left-0 right-0 top-[calc(100%+0.375rem)] z-30 overflow-hidden p-1"
        @keydown="handleListKeydown"
      >
        <button
          v-for="(option, optionIndex) in options"
          :key="option.value"
          :id="`${listboxId}-opt-${optionIndex}`"
          type="button"
          role="option"
          :aria-selected="option.value === modelValue"
          class="flex h-9 w-full items-center gap-2 rounded-md px-2.5 text-left text-sm font-medium text-base-content hover:bg-base-200"
          :class="option.value === modelValue ? 'bg-primary/10 text-primary' : ''"
          @click="selectOption(option.value)"
        >
          <span class="min-w-0 flex-1 truncate">{{ option.label }}</span>
          <Check v-if="option.value === modelValue" class="h-4 w-4 shrink-0" />
        </button>
      </div>
    </Transition>
  </div>
</template>
