<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Ban, Eye, Flag, Loader2, X } from '@lucide/vue'
import {
  fetchModerationCourseReviewReports,
  moderationCourseReviewStatus,
  revealCourseReviewAuthor,
  updateModerationReportStatus,
  type ModerationCourseReviewReportItem,
} from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
import EmptyState from '@/site/components/EmptyState.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import UserAvatar from '@/site/components/UserAvatar.vue'
import type { CourseReviewModerationPageProps, LayoutPayload } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: CourseReviewModerationPageProps
}>()
const { t, te } = useI18n()

const isAdmin = page.layout.viewer.adminPermissions.includes(0)

// ---- 举报队列 ----
const reportStatus = ref<'open' | 'resolved' | 'rejected'>('open')
const reportItems = ref<ModerationCourseReviewReportItem[]>([])
const reportNextCursor = ref(0)
const reportHasNext = ref(true)
const reportLoading = ref(false)
const reportLoaded = ref(false)
const reportError = ref('')
const reportBusyIds = ref<number[]>([])

async function loadReports(reset = false) {
  if (reportLoading.value) return
  reportLoading.value = true
  reportError.value = ''
  try {
    const payload = await fetchModerationCourseReviewReports(reportStatus.value, reset ? 0 : reportNextCursor.value, 20)
    reportItems.value = reset ? payload.items : mergeReports(reportItems.value, payload.items)
    reportNextCursor.value = payload.nextCursor
    reportHasNext.value = payload.hasNext
    reportLoaded.value = true
  } catch (error) {
    reportError.value = error instanceof Error ? error.message : t('api.moderationCourseReviewReportsFailed')
  } finally {
    reportLoading.value = false
  }
}

function mergeReports(current: ModerationCourseReviewReportItem[], incoming: ModerationCourseReviewReportItem[]) {
  const seen = new Set(current.map((item) => item.id))
  return [...current, ...incoming.filter((item) => !seen.has(item.id))]
}

function switchReportStatus(status: 'open' | 'resolved' | 'rejected') {
  if (reportStatus.value === status) return
  reportStatus.value = status
  reportItems.value = []
  reportNextCursor.value = 0
  reportHasNext.value = true
  reportLoaded.value = false
  void loadReports(true)
}

function reportBusy(id: number) {
  return reportBusyIds.value.includes(id)
}

function reasonLabel(item: ModerationCourseReviewReportItem) {
  const key = `courseReviewModeration.reasons.${item.reason}`
  return te(key) ? t(key) : item.reason
}

async function hideReview(item: ModerationCourseReviewReportItem) {
  if (reportBusy(item.id)) return
  reportBusyIds.value = [...reportBusyIds.value, item.id]
  reportError.value = ''
  try {
    await moderationCourseReviewStatus(item.reviewId, 'hide')
    await updateModerationReportStatus(item.id, 'ban')
    reportItems.value = reportItems.value.filter((report) => report.id !== item.id)
  } catch (error) {
    reportError.value = error instanceof Error ? error.message : t('api.moderationActionFailed')
  } finally {
    reportBusyIds.value = reportBusyIds.value.filter((id) => id !== item.id)
  }
}

// ---- 身份揭示（Admin only） ----
const revealTarget = ref<ModerationCourseReviewReportItem | null>(null)
const revealReason = ref('')
const revealSubmitting = ref(false)
const revealError = ref('')
const revealResult = ref('')

function openReveal(item: ModerationCourseReviewReportItem) {
  if (!isAdmin) return
  revealTarget.value = item
  revealReason.value = ''
  revealError.value = ''
  revealResult.value = ''
}

function closeReveal() {
  revealTarget.value = null
  revealError.value = ''
  revealResult.value = ''
}

async function submitReveal() {
  if (!revealTarget.value) return
  if (!revealReason.value.trim()) {
    revealError.value = t('courseReviewModeration.revealReasonRequired')
    return
  }
  revealSubmitting.value = true
  revealError.value = ''
  revealResult.value = ''
  try {
    const payload = await revealCourseReviewAuthor(revealTarget.value.reviewId, revealReason.value.trim())
    if (payload.authorUserId && payload.authorUserId > 0) {
      const author = payload.username || payload.nickname || `#${payload.authorUserId}`
      revealResult.value = t(
        payload.isAnonymous ? 'courseReviewModeration.revealResultAnonymous' : 'courseReviewModeration.revealResultPublic',
        { author },
      )
    } else {
      revealResult.value = t('courseReviewModeration.revealLegacy')
    }
  } catch (error) {
    revealError.value = error instanceof Error ? error.message : t('api.moderationCourseReviewRevealFailed')
  } finally {
    revealSubmitting.value = false
  }
}

onMounted(() => {
  void loadReports(true)
})
</script>

<template>
  <main class="min-w-0 pb-8">
    <PageHeader
      :title="t('courseReviewModeration.title')"
      :description="t('courseReviewModeration.description')"
      compact
      class="border-b-0 !mb-2 sm:!mb-2 !pb-2 sm:!pb-2"
    />

    <section class="space-y-3">
      <p v-if="reportError" class="rounded border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">
        {{ reportError }}
      </p>

      <div class="gf-card overflow-hidden">
        <div class="flex items-center gap-1 border-b border-line bg-base-200/60 p-2">
          <button
            type="button"
            class="gf-tab"
            :class="reportStatus === 'open' ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
            @click="switchReportStatus('open')"
          >
            {{ t('courseReviewModeration.statusTabs.open') }}
          </button>
          <button
            type="button"
            class="gf-tab"
            :class="reportStatus === 'resolved' ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
            @click="switchReportStatus('resolved')"
          >
            {{ t('courseReviewModeration.statusTabs.resolved') }}
          </button>
          <button
            type="button"
            class="gf-tab"
            :class="reportStatus === 'rejected' ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
            @click="switchReportStatus('rejected')"
          >
            {{ t('courseReviewModeration.statusTabs.rejected') }}
          </button>
        </div>

        <div v-if="reportItems.length" class="divide-y divide-line">
          <article
            v-for="item in reportItems"
            :key="item.id"
            class="grid grid-cols-[32px_minmax(0,1fr)] gap-3 px-3 py-2.5 transition hover:bg-base-200/70 lg:grid-cols-[32px_minmax(260px,1fr)_130px_150px_190px] lg:items-center lg:gap-4"
          >
            <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded bg-base-200 text-warning">
              <Flag class="h-4 w-4" />
            </div>

            <div class="min-w-0">
              <div class="flex min-w-0 items-center gap-1.5 text-[15px] leading-5 text-base-content/80">
                <span class="shrink-0 text-base-content/45">{{ t('courseReviewModeration.reviewLabel') }} #{{ item.reviewId }}</span>
                <span v-if="item.reportCount > 1" class="gf-badge gf-badge-ghost text-[11px]">
                  {{ t('courseReviewModeration.reportCount', { count: item.reportCount }) }}
                </span>
              </div>
              <p class="mt-0.5 line-clamp-2 text-[13px] leading-5 text-base-content/55">{{ item.excerpt }}</p>
              <p v-if="item.note" class="mt-0.5 line-clamp-1 text-[12px] leading-5 text-base-content/45">
                {{ t('courseReviewModeration.noteLabel') }} {{ item.note }}
              </p>
              <div class="mt-1 flex flex-wrap items-center gap-x-2.5 gap-y-1 text-xs text-base-content/50 lg:hidden">
                <a
                  :href="`/u/${item.reporter.id}`"
                >
                  <UserAvatar :src="item.reporter.avatarUrl" alt="" class="h-4 w-4 rounded-full object-cover ring-1 ring-line" />
                  <span class="shrink-0">{{ t('courseReviewModeration.reporterLabel') }}</span>
                  <span class="max-w-28 truncate font-medium text-base-content/65">{{ item.reporter.username }}</span>
                </a>
                <time>{{ formatDateTime(item.createdAt) }}</time>
              </div>
            </div>

            <div class="hidden min-w-0 text-[13px] text-base-content/55 lg:block">
              <div class="flex items-center gap-1.5">
                <span class="shrink-0 text-base-content/40">{{ t('courseReviewModeration.reasonLabel') }}</span>
                <span class="min-w-0 truncate font-medium text-base-content/70">{{ reasonLabel(item) }}</span>
              </div>
            </div>

            <div class="hidden min-w-0 text-[13px] text-base-content/55 lg:block">
              <a
                :href="`/u/${item.reporter.id}`"
                class="flex min-w-0 items-center gap-1.5 hover:text-primary"
              >
                <UserAvatar :src="item.reporter.avatarUrl" alt="" class="h-5 w-5 rounded-full object-cover ring-1 ring-line" />
                <span class="shrink-0">{{ t('courseReviewModeration.reporterLabel') }}</span>
                <span class="min-w-0 truncate font-medium text-base-content/65">{{ item.reporter.username }}</span>
              </a>
              <time class="mt-0.5 block text-xs tabular-nums text-base-content/45">{{ formatDateTime(item.createdAt) }}</time>
            </div>

            <div class="col-start-2 mt-1 flex flex-wrap items-center gap-2 lg:col-start-auto lg:mt-0 lg:justify-end">
              <template v-if="reportStatus === 'open'">
                <button
                  type="button"
                  class="gf-button gf-button-sm gf-button-danger shrink-0 whitespace-nowrap text-xs"
                  :disabled="reportBusy(item.id)"
                  @click="hideReview(item)"
                >
                  <Ban class="h-4 w-4" />
                  {{ t('courseReviewModeration.hide') }}
                </button>
                <button
                  v-if="isAdmin"
                  type="button"
                  class="gf-button gf-button-sm gf-button-outline shrink-0 whitespace-nowrap text-xs"
                  :disabled="reportBusy(item.id)"
                  @click="openReveal(item)"
                >
                  <Eye class="h-4 w-4" />
                  {{ t('courseReviewModeration.reveal') }}
                </button>
              </template>
            </div>
          </article>
        </div>

        <EmptyState
          v-else-if="reportLoading"
          :icon="Flag"
          :title="t('courseReviewModeration.loading')"
          loading
        />
        <EmptyState
          v-else
          :icon="Flag"
          :title="t('courseReviewModeration.emptyTitle')"
          :description="t('courseReviewModeration.emptyDescription')"
        />

        <footer v-if="reportLoaded && (reportItems.length || reportHasNext)" class="border-t border-line px-4 py-3 text-center text-xs font-semibold text-base-content/55">
          <button
            v-if="reportHasNext"
            type="button"
            class="gf-button gf-button-sm gf-button-ghost"
            :disabled="reportLoading"
            @click="loadReports(false)"
          >
            {{ reportLoading ? t('courseReviewModeration.loading') : t('courseReviewModeration.loadMore') }}
          </button>
          <span v-else-if="reportItems.length" class="text-xs text-base-content/45">{{ t('courseReviewModeration.noMore') }}</span>
        </footer>
      </div>
    </section>

    <!-- 身份揭示弹窗（Admin only） -->
    <Teleport to="body">
      <div
        v-if="revealTarget"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
        role="dialog"
        aria-modal="true"
        @click.self="closeReveal"
      >
        <div class="w-full max-w-md rounded-[var(--gf-radius-box)] bg-base-100 p-5 shadow-lg ring-1 ring-line">
          <div class="flex items-start justify-between gap-3">
            <h2 class="text-base font-bold text-base-content">{{ t('courseReviewModeration.revealTitle') }}</h2>
            <button
              type="button"
              class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75"
              @click="closeReveal"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
          <p class="mt-1 text-[13px] leading-5 text-base-content/55">{{ t('courseReviewModeration.revealDescription') }}</p>

          <div class="mt-4">
            <textarea
              v-model="revealReason"
              class="gf-textarea min-h-24"
              maxlength="500"
              :placeholder="t('courseReviewModeration.revealReasonPlaceholder')"
            />
          </div>

          <p v-if="revealError" class="mt-3 text-sm text-error">{{ revealError }}</p>
          <p v-if="revealResult" class="mt-3 rounded border border-success/25 bg-success/10 px-3 py-2 text-sm text-base-content/75">
            {{ revealResult }}
          </p>

          <div class="mt-4 flex justify-end gap-2">
            <button
              type="button"
              class="gf-button gf-button-md gf-button-muted"
              :disabled="revealSubmitting"
              @click="closeReveal"
            >
              {{ t('common.cancel') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-md gf-button-primary"
              :disabled="revealSubmitting"
              @click="submitReveal"
            >
              <Loader2 v-if="revealSubmitting" class="h-4 w-4 animate-spin" />
              {{ revealSubmitting ? t('common.loadingShort') : t('courseReviewModeration.confirmReveal') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </main>
</template>
