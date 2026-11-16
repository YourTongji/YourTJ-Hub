<script setup lang="ts">
import type { SelectValueProps } from "reka-ui"
import { SelectValue } from "reka-ui"

const props = defineProps<SelectValueProps>()
</script>

<template>
  <SelectValue
    data-slot="select-value"
    v-bind="props"
    v-slot="{ selectedLabel }"
  >
    <!-- Keep truncation on a block child while the value root stays flex for its contents. -->
    <span class="block min-w-0 flex-1 truncate">
      <slot>{{ selectedLabel.length ? selectedLabel.join(', ') : props.placeholder }}</slot>
    </span>
  </SelectValue>
</template>
