<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { FolderOpen, Search, UsersRound } from '@lucide/vue'
import { formatNumber } from '@/runtime/format'
import EmptyState from '@/site/components/EmptyState.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import TopicList from '@/site/components/TopicList.vue'
import type { LayoutPayload, SearchPageProps } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: SearchPageProps
  pageUrl: string
}>()
const { t } = useI18n()

const query = ref(page.props.query)
const scope = computed(() => page.props.scope || 'all')
const topics = computed(() => page.props.topics || [])
const users = computed(() => page.props.users || [])
const categories = computed(() => page.props.categories || [])
const hasQuery = computed(() => (page.props.query || '').trim().length > 0)
const hasResults = computed(() => topics.value.length > 0 || users.value.length > 0 || categories.value.length > 0)
const hasTopicResults = computed(() => topics.value.length > 0)
const hasUserResults = computed(() => users.value.length > 0)
const hasCategoryResults = computed(() => categories.value.length > 0)
const searchUnavailable = computed(() => page.props.searchUnavailable === true)
const failedScopes = computed(() => page.props.failedScopes || [])
const hasPartialFailure = computed(() => failedScopes.value.length > 0 && !searchUnavailable.value)
const scopeLabelMap: Record<string, string> = {
  topics: 'scopeTopics',
  users: 'scopeUsers',
  categories: 'scopeCategories',
}
const scopeLabels = computed(() => failedScopes.value.map((s) => (scopeLabelMap[s] ? t(`searchPage.${scopeLabelMap[s]}`) : s)).join(', '))
const searchDescription = computed(() => {
  if (!hasQuery.value) return t('searchPage.emptyPrompt')
  const count = scope.value === 'users' ? page.props.usersTotal : scope.value === 'categories' ? page.props.categoriesTotal : page.props.total
  return `${page.props.query} · ${t('searchPage.resultCount', { count: formatNumber(count) })}`
})

const scopeTabs = computed(() => {
  const base = '/search'
  const scopeParam = (value: string) => (value === 'all' || !value ? '' : `&scope=${value}`)
  return [
    { key: 'all', label: t('searchPage.scopeAll'), url: `${base}?q=${encodeURIComponent(page.props.query)}`, active: scope.value === 'all' },
    { key: 'topics', label: t('searchPage.scopeTopics'), url: `${base}?q=${encodeURIComponent(page.props.query)}${scopeParam('topics')}`, active: scope.value === 'topics' },
    { key: 'users', label: t('searchPage.scopeUsers'), url: `${base}?q=${encodeURIComponent(page.props.query)}${scopeParam('users')}`, active: scope.value === 'users' },
    { key: 'categories', label: t('searchPage.scopeCategories'), url: `${base}?q=${encodeURIComponent(page.props.query)}${scopeParam('categories')}`, active: scope.value === 'categories' },
  ]
})

// 分页链接保留当前 scope（与后端 Pagination.NextURL 一致）
function pageURL(nextPage: number) {
  const params = new URLSearchParams()
  if (page.props.query) params.set('q', page.props.query)
  if (scope.value !== 'all') params.set('scope', scope.value)
  if (nextPage > 1) params.set('page', String(nextPage))
  const qs = params.toString()
  return qs ? `/search?${qs}` : '/search'
}

function userUrl(user: { id: number }) {
  return `/u/${user.id}`
}
function categoryUrl(cat: { slug: string; id: number }) {
  return `/c/${cat.slug}/${cat.id}`
}

watch(
  () => page.props.query,
  () => {
    query.value = page.props.query
  },
)
</script>

<template>
  <main class="min-w-0 pb-8">
    <PageHeader :title="t('searchPage.title')" :description="searchDescription" compact>
      <template #badge>
        <span class="gf-badge gf-badge-muted h-5 text-[11px] uppercase">{{ t('searchPage.label') }}</span>
      </template>
      <template #actions>
        <form action="/search" method="GET" class="w-full sm:w-80 lg:w-96">
          <input v-if="scope !== 'all'" type="hidden" name="scope" :value="scope" />
          <label class="flex h-10 items-center gap-2 rounded-field border border-line bg-base-100 px-3 text-sm text-base-content/55 transition focus-within:border-primary focus-within:ring-4 focus-within:ring-primary/20">
            <Search class="h-4 w-4 shrink-0" />
            <input v-model="query" name="q" class="min-w-0 flex-1 bg-transparent text-base-content outline-none placeholder:text-base-content/55" :placeholder="t('searchPage.inputPlaceholder')" />
            <button type="submit" class="gf-button gf-button-sm gf-button-neutral shrink-0">{{ t('common.search') }}</button>
          </label>
        </form>
        </template>
      </PageHeader>

      <div v-if="hasQuery && !searchUnavailable" class="mb-3 flex flex-wrap items-center gap-1">
        <a
          v-for="tab in scopeTabs"
          :key="tab.key"
          :href="tab.url"
          class="rounded-full px-3 py-1 text-sm transition"
          :class="tab.active ? 'bg-primary text-primary-content' : 'bg-base-200 text-base-content/70 hover:bg-base-300'"
          :aria-pressed="tab.active"
        >
          {{ tab.label }}
        </a>
      </div>

      <div v-if="hasPartialFailure" class="mb-3 rounded-field border border-warning/30 bg-warning/5 px-4 py-3 text-sm text-base-content/70">
        <p class="font-medium text-base-content/80">{{ t('searchPage.partialFailureTitle') }}</p>
        <p>{{ t('searchPage.partialFailureDescription', { scopes: scopeLabels }) }}</p>
      </div>

      <section class="gf-card overflow-hidden">
        <template v-if="hasResults">
          <div v-if="hasUserResults && (scope === 'all' || scope === 'users')" class="border-b border-line">
            <h2 class="flex items-center gap-2 px-4 pt-3 text-sm font-semibold text-base-content/70">
              <UsersRound class="h-4 w-4" />
              {{ t('searchPage.usersSection') }}
              <span class="text-xs font-normal text-base-content/45">{{ t('searchPage.usersCount', { count: formatNumber(page.props.usersTotal) }) }}</span>
            </h2>
            <ul class="divide-y divide-line">
              <li v-for="user in users" :key="user.id">
                <a :href="userUrl(user)" class="flex items-center gap-3 px-4 py-3 transition hover:bg-base-200/60">
                  <img :src="user.avatarUrl || undefined" :alt="user.username" class="h-10 w-10 shrink-0 rounded-full bg-base-300 object-cover" />
                  <div class="min-w-0">
                    <p class="truncate text-sm font-medium text-base-content">{{ user.nickname || user.username }}</p>
                    <p class="truncate text-xs text-base-content/55">@{{ user.username }}</p>
                    <p v-if="user.bio" class="mt-0.5 truncate text-xs text-base-content/45">{{ user.bio }}</p>
                  </div>
                </a>
              </li>
            </ul>
          </div>

          <div v-if="hasTopicResults && (scope === 'all' || scope === 'topics')" class="border-b border-line">
            <h2 v-if="scope === 'all'" class="flex items-center gap-2 px-4 pt-3 text-sm font-semibold text-base-content/70">
              {{ t('searchPage.topicsSection') }}
              <span class="text-xs font-normal text-base-content/45">{{ t('searchPage.topicsCount', { count: formatNumber(page.props.total) }) }}</span>
            </h2>
            <TopicList :topics="topics" />
            <footer v-if="(scope === 'topics' || scope === 'all') && page.props.totalPages > 1" class="flex flex-col gap-3 border-t border-line bg-base-200/50 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="text-sm text-base-content/55">
                {{ t('common.page', { page: page.props.pagination.page, total: page.props.totalPages }) }}
              </div>
              <div class="flex items-center gap-2">
                <a
                  v-if="page.props.pagination.page > 1"
                  :href="pageURL(page.props.pagination.page - 1)"
                  class="gf-button gf-button-sm gf-button-secondary"
                >
                  {{ t('common.previousPage') }}
                </a>
                <a
                  v-if="page.props.pagination.hasNext"
                  :href="page.props.pagination.nextUrl"
                  class="gf-button gf-button-sm gf-button-secondary"
                  rel="next"
                >
                  {{ t('common.nextPage') }}
                </a>
              </div>
            </footer>
          </div>

          <div v-if="hasCategoryResults && (scope === 'all' || scope === 'categories')" class="border-b border-line">
            <h2 class="flex items-center gap-2 px-4 pt-3 text-sm font-semibold text-base-content/70">
              <FolderOpen class="h-4 w-4" />
              {{ t('searchPage.categoriesSection') }}
              <span class="text-xs font-normal text-base-content/45">{{ t('searchPage.categoriesCount', { count: formatNumber(page.props.categoriesTotal) }) }}</span>
            </h2>
            <ul class="divide-y divide-line">
              <li v-for="cat in categories" :key="cat.id">
                <a :href="categoryUrl(cat)" class="flex items-center gap-3 px-4 py-3 transition hover:bg-base-200/60">
                  <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-field text-base" :style="{ backgroundColor: cat.color || undefined }">{{ cat.icon || '#' }}</span>
                  <div class="min-w-0">
                    <p class="truncate text-sm font-medium text-base-content">{{ cat.name }}</p>
                    <p v-if="cat.desc" class="truncate text-xs text-base-content/55">{{ cat.desc }}</p>
                  </div>
                </a>
              </li>
            </ul>
          </div>
        </template>

        <EmptyState v-else-if="searchUnavailable" :icon="Search" :title="t('searchPage.unavailableTitle')" :description="t('searchPage.unavailableDescription')" />

        <EmptyState v-else-if="hasQuery" :icon="UsersRound" :title="t('searchPage.noResultsTitle')" :description="t('searchPage.noResultsDescription')" />

        <EmptyState v-else :icon="Search" :title="t('searchPage.startTitle')" :description="t('searchPage.startDescription')" />
      </section>
    </main>
</template>
