<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useUnreadStatus } from '@/runtime/unread-status'
import GfSwitch from './GfSwitch.vue'

const { t } = useI18n()
const unread = useUnreadStatus()
</script>

<template>
  <div class="space-y-2 border-b border-line py-3">
    <div class="flex items-center justify-between gap-4">
      <label for="unread-style-numbers" class="min-w-0 text-sm font-medium">{{ t('notifications.unreadStyleNumbers') }}</label>
      <GfSwitch
        id="unread-style-numbers"
        :model-value="unread.badgeStyle.value === 'number'"
        :disabled="!unread.checked.value || unread.badgeStyleSaving.value"
        :label="t('notifications.unreadStyleNumbers')"
        describedby="unread-style-hint"
        @update:model-value="unread.updateBadgeStyle($event ? 'number' : 'dot')"
      />
    </div>
    <p id="unread-style-hint" class="text-xs leading-relaxed text-base-content/60">{{ t('notifications.unreadStyleHint') }}</p>
    <p v-if="unread.badgeStyleSaving.value" role="status" class="text-xs text-base-content/60">{{ t('notifications.unreadStyleSaving') }}</p>
    <p v-if="unread.badgeStyleError.value" role="alert" class="text-sm text-error">{{ unread.badgeStyleError.value }}</p>
  </div>
</template>
