<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { AlertTriangle, Check, FileText, ListChecks, Loader2, Send, X } from '@lucide/vue'
import { submitTopic, uploadImage } from '@/runtime/api'
import { processImageFile, validateImageFile } from '@/runtime/image'
import { useUnsavedDraftGuard } from '@/site/composables/useUnsavedDraftGuard'
import { useCaptchaChallenge } from '@/site/composables/useCaptchaChallenge'
import PageHeader from '@/site/components/PageHeader.vue'
import VditorOfficial from '@/site/components/VditorOfficial.vue'
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
const uploadTotal = ref(0)
const uploadDone = ref(0)
const message = ref('')
const website = ref('')
const error = ref('')
const validationAttempted = ref(false)
const titleInput = ref<HTMLInputElement | null>(null)
const categorySection = ref<HTMLElement | null>(null)
const headerSection = ref<HTMLElement | null>(null)
const editorHost = ref<HTMLElement | null>(null)
const categoryPickerOpen = ref(false)
const categoryPickerRoot = ref<HTMLElement | null>(null)
const bodySection = ref<HTMLElement | null>(null)
const editor = ref<InstanceType<typeof VditorOfficial> | null>(null)

const isValid = computed(() => Boolean(title.value.trim() && content.value.trim() && categoryIds.value.length > 0))
const categoryMissing = computed(() => validationAttempted.value && categoryIds.value.length === 0)
const validationError = computed(() => validationAttempted.value && !isValid.value ? t('publish.validation.requiredFields') : '')
const draftSaveable = computed(() => isValid.value && !submitting.value && !uploading.value)
const titleFilled = computed(() => Boolean(title.value.trim()))
const bodyCharCount = computed(() => content.value.trim().length)
const bodyFilled = computed(() => bodyCharCount.value > 0)
const selectedCategories = computed(() => page.props.categories.filter((category) => categoryIds.value.includes(category.id)))
const categoriesFull = computed(() => categoryIds.value.length >= 3)
/** 向上扩展：折叠标题/分类区，让编辑区占满 */
const headerCollapsed = ref(false)
const collapsedHeaderHeight = ref(0)
const uploadText = computed(() => {
  if (!uploading.value) return ''
  return uploadTotal.value > 1 ? t('publish.processingImages', { done: uploadDone.value, total: uploadTotal.value }) : t('publish.processingImage')
})
const savedSnapshot = ref(editorSnapshot())
const hasUnsavedChanges = computed(() => editorSnapshot() !== savedSnapshot.value)
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

/** 分类下拉：点击外部关闭（better-ui：popover 走文档级 pointerdown 收敛） */
function handleCategoryPickerPointerDown(event: PointerEvent) {
  const target = event.target
  if (target instanceof Node && categoryPickerRoot.value?.contains(target)) return
  categoryPickerOpen.value = false
}

function handleCategoryPickerKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    categoryPickerOpen.value = false
    return
  }
  if (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    categoryPickerOpen.value = true
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', handleCategoryPickerPointerDown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleCategoryPickerPointerDown)
})

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

/** 工具栏「向上扩展」：折叠/展开标题+分类区，并把高度让给编辑器 */
function toggleHeaderCollapsed() {
  const header = headerSection.value
  if (header) {
    collapsedHeaderHeight.value = header.offsetHeight
  }
  headerCollapsed.value = !headerCollapsed.value
  void nextTick(() => {
    if (editor.value) {
      // 折叠：编辑器高度 = 默认 480 + 头部高度 + 间距；展开：回到 480
      const target = headerCollapsed.value ? 480 + collapsedHeaderHeight.value + 20 : 480
      editor.value.setHeight(target)
    }
  })
}

function handleEditorError(editorError: Error) {
  error.value = editorError.message
}

function imageAlt(filename: string) {
  return filename.replace(/\.[^.]+$/, '').replace(/[[\]\n\r]/g, ' ').trim() || 'image'
}

function insertMarkdownBlock(text: string) {
  editor.value?.insertMarkdown(text)
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

      <!-- 单栏全宽：发布检查并入页脚，正文区吃满主列宽度 -->
      <section class="gf-card p-4 sm:p-5">
        <div class="space-y-5">
          <!-- 标题 + 分类同一行（better-layout：主输入吃满、元数据靠右共享边缘；可整体折叠给编辑区腾空间） -->
          <div ref="headerSection" v-show="!headerCollapsed" class="flex flex-col gap-4 sm:flex-row sm:items-start sm:gap-4">
            <label class="block min-w-0 flex-1">
              <span class="text-sm font-semibold text-base-content/75">{{ t('publish.fields.title') }}</span>
              <input
                ref="titleInput"
                v-model="title"
                class="mt-1 h-11 w-full rounded-md border border-line px-3 text-lg font-semibold outline-none transition focus:border-primary focus:ring-4 focus:ring-primary/20"
                :placeholder="t('publish.titlePlaceholder')"
              />
            </label>

            <div ref="categorySection" class="shrink-0 sm:w-60">
              <div class="flex items-center justify-between gap-2">
                <span class="text-sm font-semibold text-base-content/75">{{ t('publish.fields.category') }}</span>
                <span class="text-xs text-base-content/55">{{ t('publish.maxCategories') }}</span>
              </div>
              <div ref="categoryPickerRoot" class="relative mt-1">
                <button
                  type="button"
                  class="gf-input flex h-11 w-full items-center gap-2 text-left"
                  :class="categoryMissing ? '!border-error' : ''"
                  :aria-expanded="categoryPickerOpen"
                  :aria-label="t('publish.fields.category')"
                  @click="categoryPickerOpen = !categoryPickerOpen"
                  @keydown="handleCategoryPickerKeydown"
                >
                  <span v-if="selectedCategories.length" class="flex min-w-0 flex-1 flex-wrap items-center gap-1.5 overflow-hidden">
                    <span
                      v-for="category in selectedCategories"
                      :key="category.id"
                      class="inline-flex max-w-full items-center gap-1.5 truncate rounded-md px-2 py-0.5 text-xs font-semibold"
                      :style="{ backgroundColor: category.color + '22', color: category.color }"
                    >
                      <span class="h-1.5 w-1.5 shrink-0 rounded-full" :style="{ backgroundColor: category.color }" />
                      <span class="truncate">{{ category.name }}</span>
                    </span>
                  </span>
                  <span v-else class="flex-1 text-sm text-base-content/45">{{ t('publish.selectCategory') }}</span>
                  <span class="text-xs tabular-nums text-base-content/55">{{ categoryIds.length }}/3</span>
                  <svg
                    class="h-4 w-4 shrink-0 text-base-content/45 transition-transform duration-150"
                    :class="{ 'rotate-180': categoryPickerOpen }"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    aria-hidden="true"
                  >
                    <path d="m6 9 6 6 6-6" />
                  </svg>
                </button>

                <Transition name="gf-menu">
                  <div
                    v-if="categoryPickerOpen"
                    class="gf-menu-surface absolute left-0 right-0 top-[calc(100%+0.375rem)] z-[300] max-h-64 overflow-y-auto p-1"
                  >
                    <button
                      v-for="category in props.categories"
                      :key="category.id"
                      type="button"
                      class="flex h-9 w-full items-center gap-2 rounded-md px-2.5 text-left text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-40"
                      :class="categoryIds.includes(category.id) ? 'bg-primary/10 text-primary' : 'text-base-content hover:bg-base-200'"
                      :disabled="!categoryIds.includes(category.id) && categoriesFull"
                      @click="toggleCategory(category.id)"
                    >
                      <span class="h-2 w-2 shrink-0 rounded-[3px]" :style="{ backgroundColor: category.color }" />
                      <span class="min-w-0 flex-1 truncate">{{ category.name }}</span>
                      <Check v-if="categoryIds.includes(category.id)" class="h-4 w-4 shrink-0" />
                    </button>
                    <p v-if="categoriesFull" class="px-2.5 py-1.5 text-xs text-base-content/55">{{ t('publish.maxCategories') }}</p>
                  </div>
                </Transition>
              </div>
              <p v-if="categoryMissing" class="mt-1 text-xs text-error/80">{{ t('publish.validation.categoryRequired') }}</p>
            </div>
          </div>

          <div ref="bodySection">
            <div class="mb-2 flex items-center gap-2">
              <span class="text-sm font-semibold text-base-content/75">{{ t('publish.fields.body') }}</span>
              <span v-if="uploadText" class="gf-badge gf-badge-info rounded">{{ uploadText }}</span>
            </div>

            <div ref="editorHost" class="relative">
              <VditorOfficial
                ref="editor"
                v-model="content"
                :height="480"
                :outline="true"
                :counter="true"
                :header-toggle="true"
                :header-collapsed="headerCollapsed"
                :placeholder="t('publish.visualPlaceholder')"
                @toggle-header="toggleHeaderCollapsed"
                @upload="uploadImageFiles"
                @error="handleEditorError"
              />
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

          <!-- 页脚：左侧发布检查状态条 + 右侧操作（better-layout：控制区合并、用间距分组） -->
          <div class="flex flex-col gap-4 border-t border-line pt-4 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
            <div
              class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-base-content/65"
              role="status"
              :aria-label="t('publish.checklist.title')"
            >
              <span class="inline-flex items-center gap-1.5 font-semibold text-base-content/75">
                <ListChecks class="h-3.5 w-3.5 shrink-0 text-base-content/55" aria-hidden="true" />
                {{ t('publish.checklist.title') }}
              </span>
              <span class="hidden h-3 w-px bg-line sm:inline-block" aria-hidden="true" />
              <span class="inline-flex items-center gap-1">
                <span>{{ t('publish.fields.title') }}</span>
                <span :class="titleFilled ? 'font-medium text-success' : 'text-base-content/55'">
                  {{ titleFilled ? t('publish.checklist.done') : t('publish.checklist.pending') }}
                </span>
              </span>
              <span class="text-base-content/35" aria-hidden="true">·</span>
              <span class="inline-flex items-center gap-1">
                <span>{{ t('publish.fields.category') }}</span>
                <span :class="categoryIds.length ? 'font-medium text-success' : 'text-base-content/55'">
                  {{ categoryIds.length }}/3
                </span>
              </span>
              <span class="text-base-content/35" aria-hidden="true">·</span>
              <span class="inline-flex items-center gap-1">
                <span>{{ t('publish.fields.body') }}</span>
                <span :class="bodyFilled ? 'font-medium text-success' : 'text-base-content/55'">
                  {{ t('publish.checklist.characters', { count: bodyCharCount }) }}
                </span>
              </span>
            </div>

            <div class="flex flex-wrap items-center justify-end gap-2 sm:shrink-0">
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
        </div>
      </section>

      <Transition name="gf-modal">
        <div
          v-if="leavePromptOpen"
          class="fixed inset-0 z-[100] overflow-y-auto bg-neutral/50 px-3 py-4 backdrop-blur-sm sm:px-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="leave-prompt-title"
          @click.self="closeLeavePrompt"
        >
          <div class="mx-auto flex min-h-full max-w-md items-center justify-center">
            <div class="gf-menu-surface w-full p-4 sm:p-5">
              <!-- 头部：警示图标 + 标题 + 描述 + 关闭（对齐全站确认弹窗模式） -->
              <div class="flex items-start gap-3">
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning">
                  <FileText class="h-5 w-5" />
                </div>
                <div class="min-w-0 flex-1">
                  <h2 id="leave-prompt-title" class="text-base font-bold text-base-content">{{ t('publish.leaveTitle') }}</h2>
                  <p class="mt-1 text-sm leading-6 text-base-content/55">{{ t('publish.leaveDescription') }}</p>
                </div>
                <button
                  type="button"
                  class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75"
                  :aria-label="t('common.close')"
                  @click="closeLeavePrompt"
                >
                  <X class="h-4 w-4" />
                </button>
              </div>

              <!-- 草稿条件提示：圆角块 + 图标（比整条底色条更精致） -->
              <div
                v-if="!isValid"
                class="mt-4 flex items-start gap-2.5 rounded-xl border border-warning/20 bg-warning/10 px-3.5 py-3 text-sm leading-5 text-warning/90"
              >
                <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" />
                <span>{{ t('publish.draftRequirement') }}</span>
              </div>

              <!-- 操作区：移动端主按钮置顶堆叠，桌面右对齐；危险操作（不保存离开）用 error 色 -->
              <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
                <button type="button" class="gf-button gf-button-lg gf-button-muted" @click="closeLeavePrompt">
                  {{ t('publish.continueEditing') }}
                </button>
                <button type="button" class="gf-button gf-button-lg gf-button-error" @click="discardAndLeave">
                  {{ t('publish.leaveWithoutSaving') }}
                </button>
                <button
                  type="button"
                  class="gf-button gf-button-lg gf-button-primary min-w-28"
                  :disabled="!draftSaveable"
                  @click="saveDraftAndLeave"
                >
                  <Loader2 v-if="submitting" class="h-4 w-4 animate-spin" />
                  <FileText v-else class="h-4 w-4" />
                  {{ submitting ? t('common.saving') : t('publish.saveDraft') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </main>
</template>
