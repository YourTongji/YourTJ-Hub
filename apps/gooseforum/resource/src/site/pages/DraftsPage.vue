<script setup lang="ts">
import { AlertTriangle, FileText, Loader2, PenSquare, ShieldAlert, Trash2 } from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { deleteTopic, reportContentEvent } from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
import { queueFlashMessage, useFlashMessages } from '@/runtime/flash-message'
import EmptyState from '@/site/components/EmptyState.vue'
import PageHeader from '@/site/components/PageHeader.vue'
import type { DraftPayload, DraftsPageProps, LayoutPayload } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: DraftsPageProps
}>()

const { t } = useI18n()
const { push: flash } = useFlashMessages()
const DRAFT_WINDOW_SIZE = 100
const drafts = ref([...page.props.drafts])
const total = ref(page.props.total)
const draftToDelete = ref<DraftPayload | null>(null)
const deletingDraftId = ref<number | null>(null)
const deleteError = ref('')
const cancelDeleteButton = ref<HTMLButtonElement | null>(null)
const newDraftAction = ref<HTMLAnchorElement | null>(null)
const draftDeleteButtons = new Map<number, HTMLButtonElement>()
const focusAfterDialogClose = ref<HTMLElement | null>(null)

function isBlocked(draft: DraftPayload) {
  return draft.processStatus === 1
}

function setDeleteButtonRef(id: number, element: unknown) {
  if (element instanceof HTMLButtonElement) draftDeleteButtons.set(id, element)
  else draftDeleteButtons.delete(id)
}

function requestDeleteDraft(draft: DraftPayload) {
  if (deletingDraftId.value !== null) return
  deleteError.value = ''
  draftToDelete.value = draft
  void reportContentEvent('content_delete_clicked', 'topic', draft.id)
}

function closeDeleteDialog() {
  if (deletingDraftId.value !== null) return
  draftToDelete.value = null
  deleteError.value = ''
}

function handleDialogOpenChange(open: boolean) {
  if (open) return
  closeDeleteDialog()
}

function focusCancelButton(event: Event) {
  event.preventDefault()
  cancelDeleteButton.value?.focus()
}

function handleCloseAutoFocus(event: Event) {
  if (focusAfterDialogClose.value) event.preventDefault()
}

async function confirmDeleteDraft() {
  const draft = draftToDelete.value
  if (!draft || deletingDraftId.value !== null) return

  deletingDraftId.value = draft.id
  deleteError.value = ''
  void reportContentEvent('content_delete_confirmed', 'topic', draft.id)

  let refreshWindow = false
  try {
    const deleted = await deleteTopic(draft.id)
    if (!deleted) throw new Error(t('drafts.deleteFailed'))

    const index = drafts.value.findIndex(item => item.id === draft.id)
    const nextDraft = drafts.value[index + 1] ?? drafts.value[index - 1]
    refreshWindow = drafts.value.length >= DRAFT_WINDOW_SIZE
    if (!refreshWindow) {
      focusAfterDialogClose.value = nextDraft
        ? draftDeleteButtons.get(nextDraft.id) ?? newDraftAction.value
        : newDraftAction.value
    }
    drafts.value = drafts.value.filter(item => item.id !== draft.id)
    total.value = refreshWindow
      ? Math.max(0, total.value - 1)
      : drafts.value.length
    draftToDelete.value = null
    // The server page is a bounded 100-item window. Reload to fill the slot
    // with the next draft instead of leaving the list one item short.
    if (refreshWindow) {
      queueFlashMessage(t('drafts.deleteSuccess'), 'success')
      window.location.reload()
    } else {
      flash(t('drafts.deleteSuccess'), 'success')
    }
  } catch (error) {
    deleteError.value = error instanceof Error && error.message.trim()
      ? error.message
      : t('drafts.deleteFailed')
  } finally {
    deletingDraftId.value = null
    if (focusAfterDialogClose.value) {
      await nextTick()
      focusAfterDialogClose.value.focus()
      focusAfterDialogClose.value = null
    }
  }
}
</script>

<template>
  <main class="min-w-0 pb-8">
    <PageHeader :title="t('drafts.title')" :description="t('drafts.summary')" compact>
      <template #badge>
        <span class="gf-badge gf-badge-muted h-5 tabular-nums text-base-content/75">
          {{ t('drafts.total', { count: total }) }}
        </span>
      </template>
      <template #actions>
        <a ref="newDraftAction" href="/publish" class="gf-button gf-button-sm gf-button-secondary min-h-10 w-fit text-xs">
          <PenSquare class="h-4 w-4" />
          {{ t('drafts.newDraft') }}
        </a>
      </template>
    </PageHeader>

    <section class="gf-card overflow-hidden">
      <div class="hidden grid-cols-[minmax(0,1fr)_152px_216px] gap-4 border-b border-line bg-base-200/60 px-4 py-2 text-[11px] font-bold uppercase text-base-content/75 md:grid">
        <div>{{ t('drafts.table.draft') }}</div>
        <div class="text-right">{{ t('drafts.table.updatedAt') }}</div>
        <div class="text-right">{{ t('drafts.table.action') }}</div>
      </div>

      <div v-if="drafts.length" class="divide-y divide-line">
        <article
          v-for="draft in drafts"
          :key="draft.id"
          class="grid gap-3 px-4 py-3 transition hover:bg-base-200/70 md:grid-cols-[minmax(0,1fr)_152px_216px] md:items-center md:gap-4"
        >
          <div class="min-w-0">
            <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
              <a :href="draft.editUrl" class="min-w-0 max-w-full truncate text-base font-semibold text-base-content hover:text-primary">
                {{ draft.title || t('drafts.untitled') }}
              </a>
              <span
                v-for="category in draft.categories"
                :key="category.id"
                class="inline-flex shrink-0 items-center gap-1 rounded-full border border-line px-2 py-0.5 text-[11px] font-medium text-base-content/75"
              >
                <span class="h-1.5 w-1.5 rounded-full" :style="{ backgroundColor: category.color }" />
                {{ category.name }}
              </span>
              <span
                v-if="isBlocked(draft)"
                class="gf-badge gf-badge-error shrink-0 text-[11px]"
              >
                <ShieldAlert class="h-3.5 w-3.5" />
                {{ t('drafts.blocked') }}
              </span>
            </div>
            <p class="mt-1.5 line-clamp-1 text-sm leading-6 text-base-content/55">
              {{ draft.description || t('drafts.emptyDescription') }}
            </p>
            <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-base-content/55">
              <span>{{ t('drafts.meta.createdAt') }} {{ formatDateTime(draft.createdAt) }}</span>
              <span>{{ t('drafts.meta.views') }} {{ draft.viewCount }}</span>
              <span>{{ t('drafts.meta.replies') }} {{ draft.replyCount }}</span>
            </div>
          </div>
          <time class="text-xs font-medium text-base-content/55 md:text-right">{{ formatDateTime(draft.updatedAt) }}</time>
          <div class="flex flex-wrap items-center gap-2 md:flex-nowrap md:justify-end">
            <a
              :href="draft.editUrl"
              class="gf-button gf-button-sm gf-button-primary min-h-10 shrink-0 text-xs"
            >
              <FileText class="h-4 w-4" />
              {{ t('drafts.edit') }}
            </a>
            <button
              :ref="element => setDeleteButtonRef(draft.id, element)"
              type="button"
              class="gf-button gf-button-sm gf-button-danger min-h-10 shrink-0 text-xs"
              :aria-label="t('drafts.deleteLabel', { title: draft.title || t('drafts.untitled') })"
              :disabled="deletingDraftId !== null"
              @click="requestDeleteDraft(draft)"
            >
              <Trash2 class="h-4 w-4" aria-hidden="true" />
              {{ t('drafts.delete') }}
            </button>
          </div>
        </article>
      </div>

      <EmptyState v-else :icon="FileText" :title="t('drafts.emptyTitle')" :description="t('drafts.emptyHint')">
        <a href="/publish" class="gf-button gf-button-md gf-button-primary px-4">
          {{ t('drafts.newDraft') }}
        </a>
      </EmptyState>
    </section>

    <DialogRoot :open="draftToDelete !== null" @update:open="handleDialogOpenChange">
      <DialogPortal>
        <DialogOverlay class="fixed inset-0 z-[120] bg-neutral/50 backdrop-blur-sm" />
        <DialogContent
          role="alertdialog"
          class="fixed left-1/2 top-1/2 z-[120] max-h-[90vh] w-[calc(100%-2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 overflow-y-auto outline-none"
          @open-auto-focus="focusCancelButton"
          @close-auto-focus="handleCloseAutoFocus"
        >
          <div class="gf-menu-surface w-full p-4 sm:p-5">
            <div class="flex items-start gap-3">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-error/10 text-error ring-1 ring-error/15">
                <AlertTriangle class="h-5 w-5" aria-hidden="true" />
              </div>
              <div class="min-w-0 flex-1">
                <DialogTitle class="text-base font-bold text-base-content">{{ t('drafts.deleteTitle') }}</DialogTitle>
                <DialogDescription class="mt-1 break-words text-sm leading-6 text-base-content/60">
                  {{ t('drafts.deleteDescription', { title: draftToDelete?.title || t('drafts.untitled') }) }}
                </DialogDescription>
              </div>
            </div>

            <p v-if="deleteError" class="mt-4 rounded-lg border border-error/20 bg-error/10 px-3 py-2 text-sm leading-5 text-error" role="alert">
              {{ deleteError }}
            </p>

            <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <button
                ref="cancelDeleteButton"
                type="button"
                class="gf-button gf-button-lg gf-button-muted"
                :disabled="deletingDraftId !== null"
                data-test="cancel-delete"
                @click="closeDeleteDialog"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-lg gf-button-danger"
                :disabled="deletingDraftId !== null"
                :aria-busy="deletingDraftId !== null"
                data-test="confirm-delete"
                @click="confirmDeleteDraft"
              >
                <Loader2 v-if="deletingDraftId !== null" class="h-4 w-4 animate-spin" aria-hidden="true" />
                <Trash2 v-else class="h-4 w-4" aria-hidden="true" />
                {{ deletingDraftId !== null ? t('drafts.deletePending') : t('drafts.deleteConfirm') }}
              </button>
            </div>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </main>
</template>
