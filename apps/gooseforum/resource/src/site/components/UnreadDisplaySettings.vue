<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { UnreadBadgeCategory } from '@gooseforum/client'
import { unreadBadgeCategories, useUnreadStatus } from '@/runtime/unread-status'
import GfSwitch from './GfSwitch.vue'

const { t } = useI18n()
const unread = useUnreadStatus()
function toggleCategory(category: UnreadBadgeCategory, enabled: boolean) {
  const selected = new Set(unread.badgeCategories.value)
  if (enabled) selected.add(category)
  else selected.delete(category)
  void unread.updateBadgeCategories(unreadBadgeCategories.filter(item => selected.has(item)))
}
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
    <fieldset :disabled="!unread.checked.value || unread.badgeStyleSaving.value" class="space-y-1" aria-describedby="unread-categories-hint">
      <legend class="text-sm font-medium">{{ t('notifications.unreadCategories') }}</legend>
      <label v-for="category in unreadBadgeCategories" :key="category" class="flex min-h-11 items-center gap-3 text-sm">
        <input type="checkbox" class="checkbox checkbox-sm" :checked="unread.badgeCategories.value.includes(category)" @change="toggleCategory(category, ($event.target as HTMLInputElement).checked)">
        {{ t(`notifications.tabs.${category}`) }}
      </label>
      <p id="unread-categories-hint" class="text-xs leading-relaxed text-base-content/60">{{ t('notifications.unreadCategoriesHint') }}</p>
    </fieldset>
    <p v-if="unread.badgeStyleSaving.value" role="status" class="text-xs text-base-content/60">{{ t('notifications.unreadStyleSaving') }}</p>
    <p v-if="unread.badgeStyleError.value" role="alert" class="text-sm text-error">{{ unread.badgeStyleError.value }}</p>
  </div>
</template>
