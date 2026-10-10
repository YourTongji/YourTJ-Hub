<script setup lang="ts" generic="T extends string | number">
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/admin/components/ui/select'

// Design-system replacement for native <select>. Options are keyed by index, so empty
// strings and numbers round-trip unchanged and `v-model` keeps the caller's value type.
export interface AdminSelectOption<V> { value: V; label: string; disabled?: boolean }

const props = withDefaults(defineProps<{
  modelValue?: T | null
  options: AdminSelectOption<T>[]
  disabled?: boolean
  size?: 'sm' | 'default'
  id?: string
  ariaLabel?: string
  placeholder?: string
  class?: HTMLAttributes['class']
}>(), { size: 'default' })
const emit = defineEmits<{ 'update:modelValue': [value: T]; change: [value: T] }>()

const selected = computed(() => {
  const index = props.options.findIndex((option) => option.value === props.modelValue)
  return index < 0 ? undefined : String(index)
})
function pick(key: unknown) {
  const option = props.options[Number(key)]
  if (!option || option.value === props.modelValue) return
  emit('update:modelValue', option.value)
  emit('change', option.value)
}
</script>

<template>
  <Select :model-value="selected" :disabled="disabled" @update:model-value="pick">
    <SelectTrigger :id="id" :size="size" :aria-label="ariaLabel" :class="props.class">
      <SelectValue :placeholder="placeholder" />
    </SelectTrigger>
    <SelectContent>
      <SelectItem v-for="(option, index) in options" :key="index" :value="String(index)" :disabled="option.disabled">
        {{ option.label }}
      </SelectItem>
    </SelectContent>
  </Select>
</template>
