<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Building2, CalendarDays, UsersRound } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import type { CourseDetailPageProps, LayoutPayload } from '@gooseforum/client'

defineProps<{
  layout: LayoutPayload
  props: CourseDetailPageProps
}>()
const { t } = useI18n()

function formatCredit(creditX10: number) {
  if (!creditX10) return ''
  return (creditX10 / 10).toFixed(1).replace(/\.0$/, '')
}
</script>

<template>
  <div class="pb-12">
    <a href="/courses" class="mb-3 inline-flex items-center gap-1 text-[13px] text-base-content/55 hover:text-primary">
      <ArrowLeft class="h-3.5 w-3.5" />
      {{ t('courseDetailPage.backToList') }}
    </a>

    <PageHeader :title="props.course.name">
      <template #badge>
        <span class="gf-badge gf-badge-muted">{{ props.course.primaryCode }}</span>
      </template>
      <template #meta>
        <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-[13px] text-base-content/60">
          <span class="inline-flex items-center gap-1">
            <Building2 class="h-3.5 w-3.5" />
            {{ props.course.department }}
          </span>
          <span v-if="formatCredit(props.course.creditX10)" class="inline-flex items-center gap-1">
            <span class="h-1 w-1 rounded-full bg-base-content/30" />
            {{ t('courseDetailPage.credit') }}：{{ formatCredit(props.course.creditX10) }}
          </span>
        </div>
      </template>
    </PageHeader>

    <div v-if="props.course.aliases?.length" class="mb-4 flex flex-wrap items-center gap-1.5">
      <span class="text-[12px] text-base-content/45">{{ t('courseDetailPage.aliases') }}：</span>
      <span v-for="alias in props.course.aliases" :key="alias" class="gf-badge gf-badge-ghost text-[11px]">
        {{ alias }}
      </span>
    </div>

    <section class="gf-panel">
      <h2 class="mb-3 text-base font-semibold text-base-content">
        {{ t('courseDetailPage.offeringsTitle') }}
      </h2>
      <EmptyState
        v-if="!props.course.offerings?.length"
        :icon="CalendarDays"
        :title="t('courseDetailPage.noOfferings')"
      />
      <ul v-else class="space-y-3">
        <li
          v-for="offering in props.course.offerings"
          :key="offering.id"
          class="rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 sm:bg-base-100"
        >
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span class="gf-badge gf-badge-muted">{{ offering.termCode }}</span>
            <span v-if="offering.campus" class="text-[12px] text-base-content/55">{{ offering.campus }}</span>
            <span v-if="offering.faculty" class="text-[12px] text-base-content/55">{{ offering.faculty }}</span>
          </div>
          <div v-if="offering.instructors?.length" class="mt-2 flex items-center gap-1.5 text-[13px] text-base-content/75">
            <UsersRound class="h-3.5 w-3.5 text-base-content/45" />
            {{ t('courseDetailPage.instructors') }}：{{ offering.instructors.join('、') }}
          </div>
        </li>
      </ul>
    </section>
  </div>
</template>
