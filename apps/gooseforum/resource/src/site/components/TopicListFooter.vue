<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronLeft, Loader2 } from '@lucide/vue'

const props = defineProps<{
  pagination: {
    page: number
    nextPage: number
    hasNext: boolean
    nextUrl: string
  }
  loadingMore: boolean
  hasTopics: boolean
  loadError: string
}>()

const emit = defineEmits<{
  loadMore: []
}>()

const { t } = useI18n()

const previousUrl = computed(() => {
  if (props.pagination.page <= 1 || typeof window === 'undefined') return ''
  const url = new URL(window.location.href)
  const previousPage = props.pagination.page - 1
  if (previousPage <= 1) {
    url.searchParams.delete('page')
  } else {
    url.searchParams.set('page', String(previousPage))
  }
  return `${url.pathname}${url.search}${url.hash}`
})
</script>

<template>
  <footer class="border-t border-line bg-base-200/50 p-3 text-center">
    <button
      v-if="pagination.hasNext"
      type="button"
      class="gf-button gf-button-sm gf-button-ghost gap-2 disabled:cursor-wait"
      :disabled="loadingMore"
      @click="emit('loadMore')"
    >
      <Loader2 v-if="loadingMore" class="h-4 w-4 animate-spin" />
      {{ loadingMore ? t('common.loadingShort') : t('common.loadMore') }}
    </button>
    <p v-else-if="hasTopics" class="text-xs font-medium text-base-content/55">{{ t('topicList.allShown') }}</p>
    <p v-if="loadError" class="mt-2 text-xs text-error">{{ t('topicList.autoLoadFailed') }}</p>
    <a v-if="pagination.hasNext" :href="pagination.nextUrl" rel="next" class="sr-only">{{ t('common.nextPage') }}</a>
    <a
      v-if="previousUrl"
      :href="previousUrl"
      rel="prev"
      class="sr-only"
    >
      <ChevronLeft class="h-4 w-4" />
      {{ t('common.previousPage') }}
    </a>
  </footer>
</template>
