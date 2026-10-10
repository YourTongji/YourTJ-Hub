<script setup lang="ts">
import { SwitchRoot, SwitchThumb } from 'reka-ui'

// Site switch on the design tokens; replaces native checkboxes for on/off settings.
// Callers may keep the confirmed value controlled and commit it only after a save succeeds.
defineProps<{ modelValue: boolean; disabled?: boolean; label?: string; labelledby?: string; describedby?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<template>
  <SwitchRoot
    :model-value="modelValue"
    :disabled="disabled"
    :aria-label="label"
    :aria-labelledby="labelledby"
    :aria-describedby="describedby"
    class="relative inline-flex h-6 w-10 shrink-0 cursor-pointer items-center rounded-full p-0.5 outline-none transition-colors duration-150 ease-out focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-base-100 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=unchecked]:bg-base-content/20"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <!-- Expand the hit area to 44px without changing the visual track. -->
    <span aria-hidden="true" class="absolute -inset-x-1 -inset-y-2.5" />
    <SwitchThumb
      class="pointer-events-none block h-5 w-5 rounded-full bg-white shadow-[0_1px_2px_oklch(0_0_0/0.2),0_0_0_0.5px_oklch(0_0_0/0.06)] transition-transform duration-150 ease-out data-[state=checked]:translate-x-4 rtl:data-[state=checked]:-translate-x-4"
    />
  </SwitchRoot>
</template>
