<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { BookOpen, Building2, CalendarDays, MapPin, Search } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import type { CourseCatalogPageProps, LayoutPayload } from '@gooseforum/client'

defineProps<{
  layout: LayoutPayload
  props: CourseCatalogPageProps
}>()
const { t } = useI18n()

function formatRating(avg: number) {
  return avg.toFixed(1)
}
</script>

<template>
  <div class="pb-12">
    <PageHeader :title="t('coursesPage.title')" :description="t('coursesPage.subtitle')">
      <template #badge>
        <span class="gf-badge gf-badge-muted">{{ props.pagination.page }}</span>
      </template>
    </PageHeader>

    <div class="space-y-4">
      <form
        class="gf-panel space-y-3 p-4"
        action="/courses"
        method="get"
        role="search"
      >
        <div class="flex flex-col gap-2 sm:flex-row">
          <label class="sr-only" for="course-keyword">{{ t('coursesPage.search') }}</label>
          <div class="relative flex-1">
            <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
            <input
              id="course-keyword"
              name="keyword"
              type="search"
              :value="props.query.keyword"
              :placeholder="t('coursesPage.searchPlaceholder')"
              class="gf-input gf-input-md w-full pl-9"
            />
          </div>
          <button type="submit" class="gf-button gf-button-md gf-button-primary">
            {{ t('coursesPage.search') }}
          </button>
        </div>
        <div class="grid gap-2 sm:grid-cols-3">
          <label class="sr-only" for="course-department">{{ t('coursesPage.department') }}</label>
          <div class="relative">
            <Building2 class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
            <input
              id="course-department"
              name="department"
              type="text"
              :value="props.query.department"
              :placeholder="t('coursesPage.department')"
              class="gf-input gf-input-md w-full pl-9"
            />
          </div>
          <label class="sr-only" for="course-term">{{ t('coursesPage.term') }}</label>
          <div class="relative">
            <CalendarDays class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
            <input
              id="course-term"
              name="term"
              type="text"
              :value="props.query.term"
              :placeholder="t('coursesPage.term')"
              class="gf-input gf-input-md w-full pl-9"
            />
          </div>
          <label class="sr-only" for="course-campus">{{ t('coursesPage.campus') }}</label>
          <div class="relative">
            <MapPin class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/40" />
            <input
              id="course-campus"
              name="campus"
              type="text"
              :value="props.query.campus"
              :placeholder="t('coursesPage.campus')"
              class="gf-input gf-input-md w-full pl-9"
            />
          </div>
        </div>
      </form>
      <EmptyState
        v-if="!props.courses.length"
        class="gf-panel"
        :icon="BookOpen"
        :title="t('coursesPage.noResult')"
        :description="t('coursesPage.emptyDescription')"
      />

      <ul v-else class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <li v-for="course in props.courses" :key="course.id">
          <a
            :href="`/courses/${course.id}`"
            class="group block rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 transition hover:border-primary/25 hover:bg-info/10 sm:bg-base-100"
          >
            <div class="flex items-start justify-between gap-2">
              <h2 class="min-w-0 truncate text-[15px] font-semibold text-base-content group-hover:text-primary">
                {{ course.name }}
              </h2>
              <span class="gf-badge gf-badge-muted shrink-0 text-[11px]">{{ course.primaryCode }}</span>
            </div>
            <p class="mt-1 truncate text-[12px] text-base-content/55">{{ course.department }}</p>
            <div v-if="course.instructors?.length" class="mt-2 truncate text-[12px] text-base-content/75">
              {{ t('coursesPage.instructors', { names: course.instructors.join('、') }) }}
            </div>
            <div v-if="course.recentTerms?.length" class="mt-1 truncate text-[11px] text-base-content/45">
              {{ t('coursesPage.terms') }}：{{ course.recentTerms.join(' / ') }}
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-1.5">
              <span v-if="course.aliases?.length" class="gf-badge gf-badge-ghost text-[11px]">
                {{ course.aliases[0] }}
              </span>
              <span v-if="course.creditX10" class="text-[11px] text-base-content/45">
                {{ t('coursesPage.credit', { credit: (course.creditX10 / 10).toFixed(1).replace(/\.0$/, '') }) }}
              </span>
              <span v-if="course.ratingAvg !== undefined" class="text-[11px]">
                <span class="font-semibold tabular-nums text-warning">{{ formatRating(course.ratingAvg) }}</span>
                <span class="text-base-content/45"> · {{ t('coursesPage.reviewCount', { count: course.reviewCount ?? 0 }, course.reviewCount ?? 0) }}</span>
              </span>
            </div>
          </a>
        </li>
      </ul>

      <nav v-if="props.pagination.hasNext" class="flex justify-center pt-2">
        <a
          :href="props.pagination.nextUrl"
          class="gf-button gf-button-md gf-button-outline"
        >
          {{ t('common.loadMore') }}
        </a>
      </nav>
    </div>
  </div>
</template>
