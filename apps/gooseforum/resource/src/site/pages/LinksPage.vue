<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ExternalLink, Link, Send, ShieldCheck } from '@lucide/vue'
import EmptyState from '@/site/components/EmptyState.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import type { LayoutPayload, LinksPageProps } from '@gooseforum/client'
import { safeUrl } from '@/runtime/safe-url'

const page = defineProps<{
  layout: LayoutPayload
  props: LinksPageProps
}>()
const { t } = useI18n()

// 渲染防线（issue #409）：历史脏配置/绕过 API 的链接降级为 '#'，不进入 href。
function linkHref(url: string) {
  return safeUrl(url, 'external') || '#'
}

function logoHref(url: string) {
  return safeUrl(url, 'image')
}
</script>

<template>
    <div class="pb-12">
      <PageHeader :title="t('linksPage.title')" :description="t('linksPage.subtitle')" compact>
        <template #badge>
          <span class="gf-badge gf-badge-muted">{{ props.totalCount }}</span>
        </template>
      </PageHeader>

      <div class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_260px]">
        <div class="space-y-6">
          <section v-for="group in props.groups" :key="group.name" class="space-y-2.5">
            <div class="flex items-center justify-between gap-3">
              <h2 class="flex min-w-0 items-center gap-2 text-base font-bold text-base-content">
                <span
                  class="flex h-7 w-7 shrink-0 items-center justify-center rounded-[var(--gf-radius-field)] bg-base-200 text-sm"
                  :style="{ color: group.color || 'var(--gf-color-base-content)' }"
                >
                  {{ group.emoji || '↗' }}
                </span>
                <span class="truncate">{{ group.name }}</span>
              </h2>
              <span class="gf-badge gf-badge-muted text-[11px]">{{ group.links.length }}</span>
            </div>

            <div class="grid grid-cols-2 gap-2 md:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-5">
              <a
                v-for="link in group.links"
                :key="`${group.name}-${link.url}`"
                :href="linkHref(link.url)"
                target="_blank"
                rel="noopener noreferrer"
                class="group relative rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-2 transition hover:border-primary/25 hover:bg-info/10 sm:bg-base-100"
              >
                <div class="flex items-center gap-2">
                  <div class="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-[var(--gf-radius-field)] border border-line bg-base-100 sm:bg-base-200">
                    <img
                      v-if="logoHref(link.logoUrl)"
                      :src="logoHref(link.logoUrl)"
                      :alt="link.name"
                      class="h-full w-full object-cover"
                      loading="lazy"
                    />
                    <Link v-else class="h-4 w-4 text-base-content/55" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="flex min-w-0 items-center gap-1.5">
                      <h3 class="truncate text-[13px] font-semibold text-base-content group-hover:text-primary">{{ link.name }}</h3>
                      <ExternalLink class="h-3 w-3 shrink-0 text-base-content/35 group-hover:text-primary" />
                    </div>
                    <p class="mt-0.5 truncate text-[11px] leading-4 text-base-content/55">{{ link.desc || link.url }}</p>
                  </div>
                </div>
                <!-- 描述被 truncate 截断时（issue #471），hover/focus 在卡片上方展示完整描述：
                     与卡片同宽（inset-x-0）、紧贴上沿（bottom-full），高度随内容自适配。
                     配色用 base 主题变量（随深浅色正确翻转）；neutral 是对比色变量，会反向。
                     完整文本本就在 DOM 中（truncate 仅视觉裁剪），tooltip 对读屏重复，故 aria-hidden -->
                <span
                  v-if="link.desc"
                  aria-hidden="true"
                  class="pointer-events-none absolute inset-x-0 bottom-full z-10 break-words rounded-[var(--gf-radius-field)] border border-line bg-base-100 px-2 py-1 text-left text-[11px] font-medium leading-4 text-base-content shadow-md opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100"
                >{{ link.desc }}</span>
              </a>
            </div>
          </section>

          <EmptyState v-if="!props.groups.length" class="gf-panel" :icon="Link" :title="t('linksPage.emptyTitle')" :description="t('linksPage.emptyDescription')" />
        </div>

        <aside class="space-y-3">
          <div class="rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 sm:bg-base-100">
            <h2 class="text-sm font-semibold text-base-content">{{ t('linksPage.applyTitle') }}</h2>
            <p class="mt-2 text-sm leading-6 text-base-content/55">{{ t('linksPage.applyDescription') }}</p>
            <a href="/publish" class="gf-button gf-button-md gf-button-primary mt-4">
              <Send class="h-4 w-4" />
              {{ t('linksPage.applyAction') }}
            </a>
          </div>

          <div class="rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 sm:bg-base-100">
            <h2 class="text-sm font-semibold text-base-content">{{ t('linksPage.principlesTitle') }}</h2>
            <div class="mt-3 space-y-2 text-sm text-base-content/75">
              <div class="flex gap-2">
                <ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-success" />
                <span>{{ t('linksPage.principles.healthy') }}</span>
              </div>
              <div class="flex gap-2">
                <ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-success" />
                <span>{{ t('linksPage.principles.relevant') }}</span>
              </div>
              <div class="flex gap-2">
                <ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-success" />
                <span>{{ t('linksPage.principles.stable') }}</span>
              </div>
            </div>
          </div>
        </aside>
      </div>
    </div>
</template>
