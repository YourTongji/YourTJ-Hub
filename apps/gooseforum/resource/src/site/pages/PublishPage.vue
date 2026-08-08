<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { ListChecks, Loader2, Send, X } from '@lucide/vue'
import { submitTopic, uploadImage } from '@/runtime/api'
import { processImageFile, validateImageFile } from '@/runtime/image'
import { markdownFromClipboard } from '@/runtime/rich-paste'
import { useUnsavedDraftGuard } from '@/site/composables/useUnsavedDraftGuard'
import { useCaptchaChallenge } from '@/site/composables/useCaptchaChallenge'
import PageHeader from '@/site/components/PageHeader.vue'
import VditorEditor from '@/site/components/VditorEditor.vue'
import type { LayoutPayload, PublishPageProps } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'

const page = defineProps<{
  layout: LayoutPayload
  props: PublishPageProps
}>()

const { t } = useI18n()
const {
  captchaRequired: captchaRequired,
  captchaId: captchaId,
  captchaImg: captchaImg,
  captchaCode: captchaCode,
  captchaLoading: captchaLoading,
  loadCaptcha: loadCaptcha,
  clearCaptcha: clearCaptcha,
  challengeFromError: challengeFromError,
} = useCaptchaChallenge()

const title = ref(page.props.topic.title || '')
const content = ref(page.props.topic.content || '')
const categoryIds = ref<number[]>([...(page.props.topic.categoryIds || [])])
const currentTopicId = ref(page.props.topicId)
const submitting = ref(false)
const uploading = ref(false)
const dragOver = ref(false)
const uploadTotal = ref(0)
const uploadDone = ref(0)
const message = ref('')
const website = ref('')
const error = ref('')
const validationAttempted = ref(false)
const titleInput = ref<HTMLInputElement | null>(null)
const categorySection = ref<HTMLElement | null>(null)
const bodySection = ref<HTMLElement | null>(null)
const editor = ref<InstanceType<typeof VditorEditor> | null>(null)

const isValid = computed(() => Boolean(title.value.trim() && content.value.trim() && categoryIds.value.length > 0))
const categoryMissing = computed(() => validationAttempted.value && categoryIds.value.length === 0)
const validationError = computed(() => validationAttempted.value && !isValid.value ? t('publish.validation.requiredFields') : '')
const selectedCategories = computed(() => page.props.categories.filter((category) => categoryIds.value.includes(category.id)))
const draftSaveable = computed(() => isValid.value && !submitting.value && !uploading.value)
const savedSnapshot = ref(editorSnapshot())
const hasUnsavedChanges = computed(() => editorSnapshot() !== savedSnapshot.value)
const uploadText = computed(() => {
  if (!uploading.value) return ''
  return uploadTotal.value > 1 ? t('publish.processingImages', { done: uploadDone.value, total: uploadTotal.value }) : t('publish.processingImage')
})
const {
  leavePromptOpen,
  forceNextNavigation,
  closeLeavePrompt,
  discardAndLeave,
  saveDraftAndLeave,
} = useUnsavedDraftGuard({
  hasUnsavedChanges,
  canSaveDraft: draftSaveable,
  saveDraftBeforeLeave: () => persistDraft(undefined, false),
})

function editorSnapshot() {
  return JSON.stringify({
    title: title.value.trim(),
    // content is kept in sync with the editor through the v-model/input
    // pipeline, so reading it directly avoids a full DOM→Markdown
    // serialization on every unsaved-changes check.
    content: content.value.trim(),
    categoryIds: [...categoryIds.value].sort((a, b) => a - b),
  })
}

function syncSavedSnapshot() {
  savedSnapshot.value = editorSnapshot()
}

function toggleCategory(id: number) {
  if (categoryIds.value.includes(id)) {
    categoryIds.value = categoryIds.value.filter((item) => item !== id)
    return
  }
  if (categoryIds.value.length >= 3) return
  categoryIds.value = [...categoryIds.value, id]
}

async function validateRequiredFields() {
  content.value = editor.value?.syncValue() ?? content.value
  if (isValid.value) return true
  validationAttempted.value = true
  await nextTick()

  if (!title.value.trim()) {
    titleInput.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    titleInput.value?.focus({ preventScroll: true })
  } else if (!categoryIds.value.length) {
    categorySection.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  } else {
    bodySection.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    editor.value?.focus()
  }
  return false
}

function imageAlt(filename: string) {
  return filename.replace(/\.[^.]+$/, '').replace(/[[\]\n\r]/g, ' ').trim() || 'image'
}

function insertMarkdownBlock(text: string) {
  editor.value?.insertMarkdown(text)
}

function handleEditorError(editorError: Error) {
  error.value = editorError.message
}

function imageFilesFromList(files: FileList | File[] | null | undefined) {
  return Array.from(files || []).filter((file) => file.type.startsWith('image/'))
}

function imageFilesFromDataTransfer(dataTransfer: DataTransfer | null) {
  if (!dataTransfer) return []
  return imageFilesFromList(dataTransfer.files)
}

function hasImageDataTransfer(dataTransfer: DataTransfer | null) {
  if (!dataTransfer) return false
  if (Array.from(dataTransfer.items || []).some((item) => item.kind === 'file' && item.type.startsWith('image/'))) return true
  return imageFilesFromList(dataTransfer.files).length > 0
}

function imageFilesFromClipboard(data: DataTransfer | null) {
  if (!data) return []
  return Array.from(data.items || [])
    .filter((item) => item.kind === 'file' && item.type.startsWith('image/'))
    .map((item) => item.getAsFile())
    .filter((file): file is File => Boolean(file))
}

async function uploadImageFiles(files: File[]) {
  if (!files.length || uploading.value) return

  uploading.value = true
  uploadTotal.value = files.length
  uploadDone.value = 0
  message.value = ''
  error.value = ''

  const markdownImages: string[] = []
  const failed: string[] = []

  try {
    for (const file of files) {
      const validation = validateImageFile(file)
      if (validation) {
        failed.push(`${file.name}: ${validation}`)
        uploadDone.value += 1
        continue
      }

      try {
        const optimized = await processImageFile(file)
        const url = await uploadImage(optimized.file)
        markdownImages.push(`![${imageAlt(file.name)}](${url})`)
      } catch (err) {
        failed.push(`${file.name}: ${err instanceof Error ? err.message : t('api.imageUploadFailed')}`)
      } finally {
        uploadDone.value += 1
      }
    }

    if (markdownImages.length) {
      insertMarkdownBlock(markdownImages.join('\n'))
      message.value = markdownImages.length > 1 ? t('publish.imagesInserted', { count: markdownImages.length }) : t('publish.imageInserted')
    }

    if (failed.length) {
      error.value = failed.slice(0, 3).join(t('punctuation.semicolon')) + (failed.length > 3 ? t('publish.moreImageFailures', { count: failed.length - 3 }) : '')
    } else if (!markdownImages.length) {
      error.value = t('publish.noUploadableImages')
    }
  } finally {
    uploading.value = false
    uploadTotal.value = 0
    uploadDone.value = 0
  }
}

async function handlePaste(event: ClipboardEvent) {
  const files = imageFilesFromClipboard(event.clipboardData)
  if (files.length) {
    event.preventDefault()
    await uploadImageFiles(files)
    return
  }

  const markdown = markdownFromClipboard(event.clipboardData)
  if (!markdown) return
  event.preventDefault()
  insertMarkdownBlock(markdown)
}

async function handleDrop(event: DragEvent) {
  dragOver.value = false
  const files = imageFilesFromDataTransfer(event.dataTransfer)
  if (!files.length) return
  event.preventDefault()
  await uploadImageFiles(files)
}

function handleDragOver(event: DragEvent) {
  if (!hasImageDataTransfer(event.dataTransfer)) return
  event.preventDefault()
  dragOver.value = true
}

async function save() {
  if (submitting.value || uploading.value || !(await validateRequiredFields())) return
  submitting.value = true
  error.value = ''
  message.value = ''
  try {
    const id = await submitTopic({
      topicId: currentTopicId.value,
      title: title.value.trim(),
      content: content.value.trim(),
      categoryId: categoryIds.value,
      topicStatus: 1,
      website: website.value,
      captchaId: captchaId.value,
      captchaCode: captchaCode.value,
    })
    clearCaptcha()
    currentTopicId.value = id
    syncSavedSnapshot()
    forceNextNavigation()
    message.value = page.props.isEditing ? t('publish.topicUpdated') : t('publish.topicPublished')
    window.location.href = `/p/post/${id}`
  } catch (err) {
    if (challengeFromError(err)) {
      error.value = t('auth.captcha.invalid')
    } else {
      error.value = err instanceof Error ? err.message : t('publish.saveFailed')
    }
  } finally {
    submitting.value = false
  }
}

async function saveDraft() {
  if (submitting.value || uploading.value || !(await validateRequiredFields())) return
  await persistDraft('/drafts')
}

async function persistDraft(nextUrl?: string, redirect = true): Promise<boolean> {
  content.value = editor.value?.syncValue() ?? content.value
  submitting.value = true
  error.value = ''
  message.value = ''
  try {
    const id = await submitTopic({
      topicId: currentTopicId.value,
      title: title.value.trim(),
      content: content.value.trim(),
      categoryId: categoryIds.value,
      topicStatus: 0,
      website: website.value,
      captchaId: captchaId.value,
      captchaCode: captchaCode.value,
    })
    clearCaptcha()
    currentTopicId.value = id
    syncSavedSnapshot()
    forceNextNavigation()
    if (redirect) window.location.href = nextUrl || '/drafts'
    return true
  } catch (err) {
    if (!challengeFromError(err)) {
      error.value = err instanceof Error ? err.message : t('publish.draftSaveFailed')
    }
    return false
  } finally {
    submitting.value = false
  }
}
</script>

<template>
    <main class="min-w-0 pb-8">
      <PageHeader :title="props.isEditing ? t('publish.editTitle') : t('publish.createTitle')" :description="t('publish.subtitle')" />

      <div class="grid gap-3 xl:grid-cols-[minmax(0,1fr)_280px]">
        <section class="gf-card p-4 sm:p-5">
          <div class="space-y-5">
            <label class="block">
              <span class="text-sm font-semibold text-base-content/75">{{ t('publish.fields.title') }}</span>
              <input
                ref="titleInput"
                v-model="title"
                class="mt-1 h-11 w-full rounded-md border border-line px-3 text-lg font-semibold outline-none transition focus:border-primary focus:ring-4 focus:ring-primary/20"
                :placeholder="t('publish.titlePlaceholder')"
              />
            </label>

            <div ref="categorySection">
              <div class="mb-2 flex items-center justify-between">
                <div class="flex min-w-0 items-baseline gap-2">
                  <span class="text-sm font-semibold text-base-content/75">
                    {{ t('publish.fields.category') }}
                  </span>
                  <span v-if="categoryMissing" class="truncate text-xs text-error/80">
                    {{ t('publish.validation.categoryRequired') }}
                  </span>
                </div>
                <span class="text-xs text-base-content/55">{{ t('publish.maxCategories') }}</span>
              </div>
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="category in props.categories"
                  :key="category.id"
                  type="button"
                  class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-40"
                  :class="categoryIds.includes(category.id) ? 'border-primary bg-info/10 text-primary' : 'border-line text-base-content/75 hover:border-line hover:bg-base-200'"
                  :disabled="!categoryIds.includes(category.id) && categoryIds.length >= 3"
                  @click="toggleCategory(category.id)"
                >
                  <span class="h-2 w-2 rounded-[3px]" :style="{ backgroundColor: category.color }" />
                  {{ category.name }}
                </button>
              </div>
            </div>

            <div ref="bodySection">
              <div class="mb-2 flex items-center gap-2">
                <span class="text-sm font-semibold text-base-content/75">{{ t('publish.fields.body') }}</span>
                <span v-if="uploadText" class="gf-badge gf-badge-info rounded">{{ uploadText }}</span>
              </div>

              <div
                :class="[
                  'min-h-80 bg-transparent',
                  dragOver ? 'bg-info/10 ring-1 ring-inset ring-primary shadow-[0_0_0_4px_rgba(59,130,246,0.12)]' : '',
                ]"
              >
                <div class="relative">
                  <VditorEditor
                    ref="editor"
                    v-model="content"
                    :placeholder="t('publish.visualPlaceholder')"
                    :uploading="uploading"
                    @paste="handlePaste"
                    @drop="handleDrop"
                    @dragover="handleDragOver"
                    @dragleave="dragOver = false"
                    @upload="uploadImageFiles"
                    @error="handleEditorError"
                  />
                  <div
                    v-if="dragOver"
                    class="pointer-events-none absolute inset-3 grid place-items-center rounded-lg border-2 border-dashed border-primary/60 bg-info/10 text-sm font-semibold text-primary"
                  >
                    {{ t('publish.dropToUpload') }}
                  </div>
                </div>
              </div>
            </div>

            <p v-if="validationError" class="gf-status-message gf-status-message-error">{{ validationError }}</p>
            <p v-if="error" class="gf-status-message gf-status-message-error">{{ error }}</p>
            <p v-if="message" class="gf-status-message gf-status-message-success">{{ message }}</p>

            <div v-if="captchaRequired" class="gf-card flex flex-wrap items-center gap-3 p-3">
              <button
                type="button"
                class="relative h-10 w-28 shrink-0 overflow-hidden rounded-md border border-line"
                :disabled="captchaLoading"
                @click="loadCaptcha()"
              >
                <Loader2 v-if="captchaLoading || !captchaImg" class="mx-auto h-5 w-5 animate-spin text-base-content/55" />
                <img v-else :src="captchaImg" :alt="t('auth.captchaAlt')" class="h-full w-full object-cover" />
              </button>
              <input
                v-model="captchaCode"
                class="h-10 min-w-0 flex-1 rounded-md border border-line px-3 text-sm outline-none focus:border-primary"
                :placeholder="t('auth.captcha')"
                maxlength="8"
              />
              <span class="text-xs text-base-content/55">{{ t('auth.validation.captchaLoadFailed') }}</span>
            </div>
            <input v-model="website" type="text" class="hidden" tabindex="-1" autocomplete="off" aria-hidden="true" />

            <div class="flex items-center justify-end gap-2 border-t border-line pt-4">
              <a href="/" class="gf-button gf-button-lg gf-button-muted">{{ t('common.cancel') }}</a>
              <button
                type="button"
                class="gf-button gf-button-lg gf-button-secondary"
                :disabled="submitting || uploading"
                @click="saveDraft"
              >
                {{ submitting ? t('common.saving') : t('publish.saveDraft') }}
              </button>
              <button
                type="button"
                class="gf-button gf-button-lg gf-button-primary"
                :disabled="submitting || uploading"
                @click="save"
              >
                <Send class="h-4 w-4" />
                {{ submitting ? t('common.saving') : props.isEditing ? t('publish.updateTopic') : t('publish.publishTopic') }}
              </button>
            </div>
          </div>
        </section>

        <aside class="space-y-3">
          <section class="gf-card p-4">
            <div class="flex items-center gap-2">
              <ListChecks class="h-4 w-4 text-base-content/55" />
              <h2 class="text-sm font-semibold text-base-content">{{ t('publish.checklist.title') }}</h2>
            </div>
            <ul class="mt-3 space-y-2 text-sm text-base-content/75">
              <li class="flex items-center justify-between gap-3"><span>{{ t('publish.fields.title') }}</span><span :class="title.trim() ? 'text-success' : 'text-base-content/55'">{{ title.trim() ? t('publish.checklist.done') : t('publish.checklist.pending') }}</span></li>
              <li class="flex items-center justify-between gap-3"><span>{{ t('publish.fields.category') }}</span><span :class="categoryIds.length ? 'text-success' : 'text-base-content/55'">{{ categoryIds.length }}/3</span></li>
              <li class="flex items-center justify-between gap-3"><span>{{ t('publish.fields.body') }}</span><span :class="content.trim() ? 'text-success' : 'text-base-content/55'">{{ t('publish.checklist.characters', { count: content.trim().length }) }}</span></li>
            </ul>
          </section>

          <section v-if="selectedCategories.length" class="gf-card p-4">
            <h2 class="text-sm font-semibold text-base-content">{{ t('publish.selectedCategories') }}</h2>
            <div class="mt-3 flex flex-wrap gap-2">
              <button
                v-for="category in selectedCategories"
                :key="category.id"
                type="button"
                class="inline-flex items-center gap-1.5 rounded-md border border-line px-2 py-1 text-sm text-base-content/75 hover:bg-base-200"
                @click="toggleCategory(category.id)"
              >
                <span class="h-2 w-2 rounded-[3px]" :style="{ backgroundColor: category.color }" />
                {{ category.name }}
                <X class="h-3 w-3" />
              </button>
            </div>
          </section>
        </aside>
      </div>

      <div v-if="leavePromptOpen" class="fixed inset-0 z-[100] flex items-center justify-center bg-neutral/50 px-4 backdrop-blur-sm" role="dialog" aria-modal="true">
        <div class="gf-menu-surface w-full max-w-md overflow-hidden">
          <div class="border-b border-line px-5 py-4">
            <h2 class="text-base font-semibold text-base-content">{{ t('publish.leaveTitle') }}</h2>
            <p class="mt-1 text-sm leading-6 text-base-content/55">
              {{ t('publish.leaveDescription') }}
            </p>
          </div>

          <div v-if="!isValid" class="border-b border-warning/20 bg-warning/10 px-5 py-3 text-sm font-medium text-warning">
            {{ t('publish.draftRequirement') }}
          </div>

          <div class="flex flex-wrap items-center justify-end gap-2 bg-base-200 px-5 py-4">
            <button type="button" class="gf-button gf-button-lg gf-button-muted" @click="closeLeavePrompt">
              {{ t('publish.continueEditing') }}
            </button>
            <button type="button" class="gf-button gf-button-lg gf-button-secondary" @click="discardAndLeave">
              {{ t('publish.leaveWithoutSaving') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-lg gf-button-primary min-w-28"
              :disabled="!draftSaveable"
              @click="saveDraftAndLeave"
            >
              {{ submitting ? t('common.saving') : t('publish.saveDraft') }}
            </button>
          </div>
        </div>
      </div>
    </main>
</template>
