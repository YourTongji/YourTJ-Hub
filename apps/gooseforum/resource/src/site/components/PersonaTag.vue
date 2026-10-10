<script setup lang="ts">
import { useI18n } from 'vue-i18n'

// Marks persona authorship next to an author name. `own` is set only from the
// viewer's private ownership flag, so the owner-only wording never reaches others.
defineProps<{ own?: boolean }>()
const { t } = useI18n()
</script>

<template>
  <span
    class="inline-flex shrink-0 items-center rounded-full px-1.5 py-0.5 text-[11px] font-medium leading-none"
    :class="own ? 'bg-primary/10 text-primary' : 'bg-base-content/8 text-base-content/60'"
    :title="own ? t('anonymous.tagOwnHint') : undefined"
    data-test="persona-tag"
  >{{ own ? t('anonymous.tagOwn') : t('anonymous.tag') }}</span>
</template>
