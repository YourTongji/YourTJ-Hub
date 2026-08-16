<script setup lang="ts">
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
</script>

<template>
  <SelectRoot
    :model-value="props.modelValue"
    @update:model-value="(value) => emit('update:modelValue', String(value))"
  >
    <SelectTrigger
      class="gf-input flex w-full items-center justify-between gap-2 text-left"
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
        class="gf-menu-surface z-50 min-w-[var(--reka-select-trigger-width)] overflow-hidden p-1"
        position="popper"
        :side-offset="6"
        align="start"
      >
        <SelectViewport class="max-h-64">
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
