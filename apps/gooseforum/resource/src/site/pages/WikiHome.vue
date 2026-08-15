<script setup lang="ts">
import { Clock, ExternalLink, FileText, Library } from '@lucide/vue'
import { formatDateTime } from '@/runtime/format'
import type { LayoutPayload, WikiHomeProps } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'

const page = defineProps<{
  layout: LayoutPayload
  props: WikiHomeProps
}>()

const { t } = useI18n()
</script>

<template>
  <div class="min-w-0">
    <div class="min-w-0">
      <!-- hero -->
      <section class="gf-card overflow-hidden">
        <div class="px-4 py-8 sm:px-8 sm:py-10">
          <div class="flex flex-wrap items-center gap-3">
            <span class="inline-flex h-10 w-10 items-center justify-center rounded-[var(--gf-radius-field)] bg-info/10 text-primary">
              <Library class="h-5 w-5" aria-hidden="true" />
            </span>
            <h1 class="text-2xl font-bold tracking-tight text-base-content sm:text-3xl">{{ t('wiki.homeTitle') }}</h1>
          </div>
          <p class="mt-3 max-w-2xl text-sm leading-6 text-base-content/60 sm:text-base">
            {{ t('wiki.homeSubtitle') }}
          </p>
          <div v-if="props.canManage" class="mt-5">
            <!-- GitHub SSOT：管理入口直达 Wiki 管理页（同步面板/命名空间）。 -->
            <a href="/admin/wiki" class="gf-button gf-button-md gf-button-neutral">
              {{ t('wiki.goToAdmin') }}
              <ExternalLink class="h-4 w-4" aria-hidden="true" />
            </a>
          </div>
        </div>
      </section>

      <!-- namespace 卡片列表 -->
      <section class="mt-4">
        <h2 class="px-1 text-sm font-semibold text-base-content/55">{{ t('wiki.namespaces') }}</h2>
        <div v-if="props.namespaces.length" class="mt-2 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <a
            v-for="namespace in props.namespaces"
            :key="namespace.name"
            :href="namespace.firstPagePath ? `/wiki/${namespace.firstPagePath}` : undefined"
            class="gf-card group block p-4 transition-colors hover:border-primary/40"
            :aria-disabled="!namespace.firstPagePath"
          >
            <div class="flex items-center justify-between gap-2">
              <h3 class="min-w-0 truncate font-semibold text-base-content group-hover:text-primary">{{ namespace.name }}</h3>
              <span class="inline-flex shrink-0 items-center gap-1 rounded bg-base-200 px-1.5 py-0.5 text-[11px] font-semibold text-base-content/55">
                <FileText class="h-3 w-3" aria-hidden="true" />
                {{ namespace.pageCount }}
              </span>
            </div>
            <p v-if="namespace.description" class="mt-1.5 line-clamp-2 text-[13px] leading-5 text-base-content/55">
              {{ namespace.description }}
            </p>
            <div v-if="namespace.updatedAt" class="mt-3 flex items-center gap-1.5 text-xs text-base-content/45">
              <Clock class="h-3 w-3" aria-hidden="true" />
              {{ formatDateTime(namespace.updatedAt) }}
            </div>
          </a>
        </div>
        <div v-else class="mt-2 rounded border border-dashed border-line bg-base-100/60 px-4 py-8 text-center text-sm text-base-content/55">
          {{ t('wiki.namespacesEmpty') }}
        </div>
      </section>

      <!-- 最近更新 -->
      <section class="mt-6">
        <h2 class="px-1 text-sm font-semibold text-base-content/55">{{ t('wiki.recentUpdates') }}</h2>
        <div v-if="props.recent.length" class="mt-2 overflow-hidden rounded-[var(--gf-radius-box)] border border-line bg-base-100">
          <a
            v-for="(item, index) in props.recent"
            :key="item.pageId"
            :href="`/wiki/${item.path}`"
            class="flex min-w-0 items-center gap-3 px-4 py-3 transition-colors hover:bg-base-200/60"
            :class="{ 'border-t border-line': index > 0 }"
          >
            <div class="min-w-0 flex-1">
              <div class="truncate text-sm font-semibold text-base-content hover:text-primary">{{ item.title }}</div>
              <div class="mt-0.5 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-base-content/55">
                <span class="truncate">{{ item.path }}</span>
                <span v-if="item.editorName">@{{ item.editorName }}</span>
                <span class="inline-flex items-center gap-1">
                  <Clock class="h-3 w-3" aria-hidden="true" />
                  {{ formatDateTime(item.updatedAt) }}
                </span>
              </div>
            </div>
          </a>
        </div>
        <div v-else class="mt-2 rounded border border-dashed border-line bg-base-100/60 px-4 py-8 text-center text-sm text-base-content/55">
          {{ t('wiki.recentEmpty') }}
        </div>
      </section>
    </div>
  </div>
</template>
