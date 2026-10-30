<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UnreadBadgeStyle } from '@gooseforum/client'

const props = defineProps<{ unread: boolean; count?: number; badgeStyle: UnreadBadgeStyle }>()
const { t } = useI18n()
const number = computed(() => props.badgeStyle === 'number' && props.count !== undefined && Number.isSafeInteger(props.count) && props.count > 0 ? props.count : undefined)
const label = computed(() => number.value === undefined ? t('notifications.unreadIndicator') : `${t('notifications.unreadIndicator')}: ${number.value}`)
</script>

<template>
  <span
    v-if="unread"
    data-unread-indicator
    :data-unread-style="number === undefined ? 'dot' : 'number'"
    :aria-label="label"
    role="img"
    class="inline-flex shrink-0 items-center justify-center rounded-full bg-error text-error-content"
    :class="number === undefined ? 'h-2.5 w-2.5' : 'min-h-4 min-w-4 px-1 text-[11px] font-semibold leading-tight tabular-nums'"
  ><span v-if="number !== undefined" aria-hidden="true">{{ number > 99 ? '99+' : number }}</span></span>
</template>
