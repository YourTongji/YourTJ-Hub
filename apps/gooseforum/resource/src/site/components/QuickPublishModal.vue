<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Draggable from 'vuedraggable'
import { Check, ChevronDown, ChevronLeft, ChevronRight, HelpCircle, Loader2, Plus, Sparkles, X } from '@lucide/vue'
import {
  DialogClose,
  DialogContent,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  PopoverContent,
  PopoverPortal,
  PopoverRoot,
  PopoverTrigger,
} from 'reka-ui'
import type { LayoutPayload } from '@gooseforum/client'
import { submitTopic, uploadImage } from '@/runtime/api'
import { processImageFile, validateImageFile } from '@/runtime/image'
import { useCaptchaChallenge } from '@/site/composables/useCaptchaChallenge'
import { useQuickPublish } from '@/site/composables/useQuickPublish'
import VditorOfficial from '@/site/components/VditorOfficial.vue'

interface UploadedImageItem {
  id: string
  url: string
  alt: string
  uploading?: boolean
}

const MAX_IMAGE_COUNT = 9
const UPLOAD_CONCURRENCY = 3

const props = defineProps<{
  layout: LayoutPayload
}>()

const { t } = useI18n()
const router = useRouter()
const { quickPublishOpen, quickPublishType, quickPublishEditPayload, closeQuickPublish } = useQuickPublish()
const isEditing = computed(() => Boolean(quickPublishEditPayload.value && quickPublishEditPayload.value.topicId > 0))

const {
  captchaRequired,
  captchaId,
  captchaImg,
  captchaCode,
  captchaLoading,
  loadCaptcha,
  clearCaptcha,
  challengeFromError,
} = useCaptchaChallenge()

const title = ref('')
const content = ref('')
const categoryIds = ref<number[]>([])
const categoryPickerOpen = ref(false)
const titleFocused = ref(false)
const submitting = ref(false)
const uploading = ref(false)
const uploadTotal = ref(0)
const uploadDone = ref(0)
const errorMessage = ref('')
const validationAttempted = ref(false)
const titleInput = ref<HTMLInputElement | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const editor = ref<InstanceType<typeof VditorOfficial> | null>(null)
const uploadedImages = ref<UploadedImageItem[]>([])

const categories = computed(() => props.layout?.sidebar?.categories || [])

const selectedCategory = computed(() => {
  if (categoryIds.value.length === 0) return null
  return categories.value.find((c) => c.id === categoryIds.value[0]) || null
})

const titleMaxLength = computed(() => (quickPublishType.value === 2 ? 30 : 120))

const typeMeta = computed(() => {
  const isEdit = isEditing.value
  switch (quickPublishType.value) {
    case 1:
      return {
        title: isEdit ? t('publish.contentTypesEdit.question') : t('publish.contentTypesAction.question'),
        badgeClass: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20',
        icon: HelpCircle,
        placeholder: t('publish.modal.questionTitlePlaceholder'),
        editorPlaceholder: t('publish.modal.contentPlaceholder'),
      }
    case 2:
    default:
      return {
        title: isEdit ? t('publish.contentTypesEdit.thought') : t('publish.contentTypesAction.thought'),
        badgeClass: 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20',
        icon: Sparkles,
        placeholder: t('publish.modal.thoughtTitlePlaceholder'),
        editorPlaceholder: t('publish.modal.contentPlaceholder'),
      }
  }
})

watch(
  quickPublishOpen,
  (open) => {
    if (open) {
      errorMessage.value = ''
      validationAttempted.value = false
      categoryPickerOpen.value = false
      titleFocused.value = false
      clearCaptcha()

      if (quickPublishEditPayload.value) {
        // 编辑模式：回显原话题标题、正文、分类与图片
        title.value = quickPublishEditPayload.value.title || ''
        content.value = quickPublishEditPayload.value.content || ''
        categoryIds.value = quickPublishEditPayload.value.categoryIds?.length
          ? [...quickPublishEditPayload.value.categoryIds]
          : (categories.value.length > 0 ? [categories.value[0].id] : [])
        uploadedImages.value = (quickPublishEditPayload.value.images || []).map((url, idx) => ({
          id: `edit-${idx}-${Date.now()}`,
          url,
          alt: '',
        }))
      } else {
        // 新建模式：重置为空
        title.value = ''
        content.value = ''
        categoryIds.value = []
        uploadedImages.value = []
        if (categories.value.length > 0) {
          categoryIds.value = [categories.value[0].id]
        }
      }

      void nextTick(() => {
        titleInput.value?.focus()
        if (editor.value && content.value) {
          editor.value.setValue?.(content.value)
        }
      })
    } else {
      title.value = ''
      content.value = ''
      categoryIds.value = []
      uploadedImages.value = []
    }
  },
  { immediate: true },
)

function selectCategory(catId: number) {
  categoryIds.value = [catId]
  categoryPickerOpen.value = false
  errorMessage.value = ''
}

function handleTitleEnter() {
  editor.value?.focus()
}

function imageAlt(filename: string) {
  return filename.replace(/\.[^.]+$/, '').replace(/[[\]\n\r]/g, ' ').trim() || 'image'
}

function triggerUpload() {
  fileInput.value?.click()
}

function handleFileInputChange(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    void uploadImageFiles(Array.from(input.files))
    input.value = ''
  }
}

function removeImage(index: number) {
  uploadedImages.value.splice(index, 1)
}

function moveImage(index: number, direction: 'left' | 'right') {
  const targetIndex = direction === 'left' ? index - 1 : index + 1
  if (targetIndex < 0 || targetIndex >= uploadedImages.value.length) return
  const [moved] = uploadedImages.value.splice(index, 1)
  uploadedImages.value.splice(targetIndex, 0, moved)
}

async function uploadImageFiles(files: File[]) {
  if (!files.length || uploading.value) return
  // 数量上限：在选择/粘贴入口统一提前拦截，未达上限时仅处理可容纳的张数
  const remainingSlots = Math.max(0, MAX_IMAGE_COUNT - uploadedImages.value.length)
  const accepted = remainingSlots > 0 ? files.slice(0, remainingSlots) : []
  uploading.value = true
  uploadTotal.value = accepted.length
  uploadDone.value = 0
  errorMessage.value = ''
  if (accepted.length < files.length) {
    errorMessage.value = t('publish.modal.maxImageCount', { count: MAX_IMAGE_COUNT })
  }

  try {
    // 有界并发上传（默认 3 张在途）：按入参顺序抢占游标，失败即中止后续调度，
    // 成功项按占位入列顺序落位，最终图片列表顺序与选择顺序一致
    let cursor = 0
    let failed = false
    const worker = async () => {
      while (!failed && cursor < accepted.length) {
        const file = accepted[cursor++]
        const validation = validateImageFile(file)
        if (validation) {
          errorMessage.value = `${file.name}: ${validation}`
          continue
        }
        const tempId = `img_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`
        uploadedImages.value.push({
          id: tempId,
          url: '',
          alt: imageAlt(file.name),
          uploading: true,
        })

        try {
          const processed = await processImageFile(file)
          const url = await uploadImage(processed.file)
          const item = uploadedImages.value.find((i) => i.id === tempId)
          if (item) {
            item.url = url
            item.uploading = false
          }
          // 发瞬间图片作为顶部媒体独立管理，不必插入正文
          uploadDone.value += 1
        } catch (err) {
          failed = true
          uploadedImages.value = uploadedImages.value.filter((i) => i.id !== tempId)
          errorMessage.value = err instanceof Error ? err.message : t('api.imageUploadFailed')
        }
      }
    }

    const runnerCount = Math.min(UPLOAD_CONCURRENCY, accepted.length)
    await Promise.all(Array.from({ length: runnerCount }, () => worker()))
  } finally {
    uploading.value = false
    uploadTotal.value = 0
    uploadDone.value = 0
  }
}

function handleEditorError(err: Error) {
  errorMessage.value = err.message
}

async function handleSubmit() {
  validationAttempted.value = true
  content.value = editor.value?.syncValue() ?? content.value

  let finalTitle = title.value.trim()
  if (quickPublishType.value === 2 && !finalTitle) {
    const cleanContent = content.value.replace(/[#*`~>[\]()\n]/g, ' ').trim()
    finalTitle = cleanContent.slice(0, 30) || (uploadedImages.value.length > 0 ? t('publish.modal.imageOnlyTitle') : t('publish.contentTypesAction.thought'))
  }

  let finalContent = content.value.trim()
  if (quickPublishType.value === 2 && !finalContent && uploadedImages.value.length > 0) {
    finalContent = t('publish.modal.imageOnlyContent')
  }

  if (!finalTitle || !finalContent || categoryIds.value.length === 0) {
    errorMessage.value = t('publish.validation.requiredFields')
    return
  }

  if (submitting.value || uploading.value) return
  submitting.value = true
  errorMessage.value = ''

  try {
    const targetTopicId = quickPublishEditPayload.value ? quickPublishEditPayload.value.topicId : 0
    const topicId = await submitTopic({
      topicId: targetTopicId,
      title: finalTitle,
      content: finalContent,
      categoryId: categoryIds.value,
      topicStatus: 1,
      contentType: quickPublishType.value,
      images: uploadedImages.value.filter((i) => !i.uploading && i.url).map((i) => i.url),
      captchaId: captchaId.value || undefined,
      captchaCode: captchaCode.value || undefined,
    })

    closeQuickPublish()
    if (targetTopicId > 0) {
      if (typeof window !== 'undefined') {
        if (window.location.pathname.includes(`/p/post/${targetTopicId}`)) {
          window.location.reload()
        } else {
          try {
            await router.push(`/p/post/${targetTopicId}`)
          } catch {
            window.location.href = `/p/post/${targetTopicId}`
          }
        }
      }
    } else if (topicId) {
      try {
        await router.push(`/p/post/${topicId}`)
      } catch {
        window.location.href = `/p/post/${topicId}`
      }
    }
  } catch (err) {
    if (challengeFromError(err)) {
      errorMessage.value = t('server.auth.captcha.invalid')
    } else {
      errorMessage.value = err instanceof Error ? err.message : t('publish.saveFailed')
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <DialogRoot :open="quickPublishOpen" @update:open="(val) => !val && closeQuickPublish()">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[80] bg-black/50 backdrop-blur-xs duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[90] w-[96vw] sm:w-[92vw] max-w-2xl -translate-x-1/2 -translate-y-1/2 rounded-2xl sm:rounded-3xl border border-line/80 bg-base-100 shadow-2xl outline-none duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 h-[88vh] sm:h-auto min-h-[520px] sm:min-h-0 max-h-[95vh] flex flex-col overflow-hidden"
        :aria-describedby="undefined"
        @keydown.meta.enter="handleSubmit"
        @keydown.ctrl.enter="handleSubmit"
      >
        <!-- 弹层顶栏：类型徽章与关闭按钮（具有平滑悬停微交互） -->
        <div class="flex items-center justify-between px-4 sm:px-6 pt-3.5 sm:pt-4 pb-2 shrink-0 border-b border-line/40">
          <div class="flex items-center gap-2">
            <DialogTitle class="sr-only">
              {{ typeMeta.title }}
            </DialogTitle>
            <span
              class="inline-flex items-center gap-1.5 rounded-full border px-2.5 sm:px-2.5 py-0.5 text-xs font-semibold transition-transform duration-200 hover:scale-105 shadow-2xs"
              :class="typeMeta.badgeClass"
            >
              <component :is="typeMeta.icon" class="h-3.5 w-3.5" />
              <span>{{ typeMeta.title }}</span>
            </span>
            <span v-if="!isEditing" class="text-xs sm:text-sm font-semibold text-base-content/80">
              {{ t('publish.modal.quickPublish') }}
            </span>
          </div>

          <DialogClose
            class="rounded-full p-1.5 text-base-content/40 hover:bg-base-200 hover:text-base-content hover:rotate-90 transition-all duration-200 active:scale-[0.92] focus-visible:ring-2 focus-visible:ring-primary/40 outline-none"
            :aria-label="t('publish.modal.close')"
          >
            <X class="h-5 w-5 transition-transform duration-200" />
          </DialogClose>
        </div>

        <!-- 弹层主体：参考用户截图（首行快捷传图 -> 填写标题 -> 添加正文铺满 -> 底部工具栏与添加分区） -->
        <div class="flex-1 min-h-0 flex flex-col px-4 sm:px-6 py-2.5 sm:py-3 gap-2.5 sm:gap-3 overflow-hidden sm:overflow-y-auto">
          <!-- 首行入口：快捷传图与已上传图片预览横向滑动流（支持滑动预览与拖拽重排，移动端比例适度放大） -->
          <div class="gf-image-scroll-track shrink-0 flex items-center gap-3 overflow-x-auto pb-2 pt-0.5">
            <!-- 已上传/正在上传的图片缩略图卡片列表（支持桌面拖拽与移动端触控拖拽排序）
                 触控点按/拖拽降级（issue #455）：✕/左右移按钮豁免拖拽热区（filter），
                 触屏需长按 180ms 且位移超过阈值才进入拖拽，轻点按钮可正常触发点击 -->
            <Draggable
              v-model="uploadedImages"
              item-key="id"
              :animation="200"
              ghost-class="opacity-35"
              class="flex items-center gap-3 shrink-0"
              filter=".gf-image-card-btn"
              :prevent-on-filter="false"
              :delay="180"
              :delay-on-touch-only="true"
              :touch-start-threshold="10"
            >
              <template #item="{ element: img, index: idx }">
                <div
                  class="group relative h-[86px] w-[86px] sm:h-20 sm:w-20 shrink-0 overflow-hidden rounded-2xl sm:rounded-xl border border-line bg-base-200/50 shadow-xs transition-transform duration-150 cursor-grab active:cursor-grabbing select-none"
                >
                  <div v-if="img.uploading" class="absolute inset-0 flex items-center justify-center bg-base-200/80 pointer-events-none">
                    <Loader2 class="h-5 w-5 animate-spin text-primary" />
                  </div>
                  <img v-else :src="img.url" :alt="img.alt" class="h-full w-full object-cover pointer-events-none" />

                  <!-- 左下角：轮播次序角标（1, 2, 3...） -->
                  <span
                    v-if="!img.uploading"
                    class="absolute bottom-1.5 left-1.5 flex h-5 min-w-5 items-center justify-center rounded-md bg-black/70 backdrop-blur-xs px-1 text-[11px] font-mono font-bold text-white shadow-xs select-none pointer-events-none"
                  >
                    {{ idx + 1 }}
                  </span>

                  <!-- 调序操作按钮（向左/向右移动：支持点击微调与键盘/无障碍辅助） -->
                  <div
                    v-if="!img.uploading && uploadedImages.length > 1"
                    class="absolute bottom-1.5 right-1.5 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity z-10"
                  >
                    <button
                      v-if="idx > 0"
                      type="button"
                      class="gf-image-card-btn flex h-5.5 w-5.5 items-center justify-center rounded-md bg-black/75 backdrop-blur-xs text-white hover:bg-black active:scale-90 transition-all cursor-pointer shadow-xs"
                      :title="t('publish.modal.moveImageLeft')"
                      :aria-label="t('publish.modal.moveImageLeft')"
                      @click.stop="moveImage(idx, 'left')"
                    >
                      <ChevronLeft class="h-4 w-4" />
                    </button>
                    <button
                      v-if="idx < uploadedImages.length - 1"
                      type="button"
                      class="gf-image-card-btn flex h-5.5 w-5.5 items-center justify-center rounded-md bg-black/75 backdrop-blur-xs text-white hover:bg-black active:scale-90 transition-all cursor-pointer shadow-xs"
                      :title="t('publish.modal.moveImageRight')"
                      :aria-label="t('publish.modal.moveImageRight')"
                      @click.stop="moveImage(idx, 'right')"
                    >
                      <ChevronRight class="h-4 w-4" />
                    </button>
                  </div>

                  <!-- 右上角删除按钮 -->
                  <button
                    v-if="!img.uploading"
                    type="button"
                    class="gf-image-card-btn absolute top-1.5 right-1.5 flex h-6 w-6 items-center justify-center rounded-full bg-black/70 backdrop-blur-xs text-white hover:bg-error transition-all active:scale-90 shadow-xs cursor-pointer z-10"
                    :aria-label="t('publish.modal.deleteImage')"
                    @click.stop="removeImage(idx)"
                  >
                    <X class="h-3.5 w-3.5 stroke-[2.5]" />
                  </button>
                </div>
              </template>
            </Draggable>

            <!-- 快捷传图 + 方形虚线卡片 -->
            <button
              type="button"
              class="flex h-[86px] w-[86px] sm:h-20 sm:w-20 shrink-0 flex-col items-center justify-center rounded-2xl sm:rounded-xl border-2 border-dashed border-line/80 bg-base-200/40 hover:bg-base-200/80 hover:border-primary/60 text-base-content/55 hover:text-primary transition-all active:scale-[0.96] cursor-pointer"
              :aria-label="t('publish.modal.addImages')"
              @click="triggerUpload"
            >
              <Plus class="h-6 w-6 sm:h-6 sm:w-6" />
              <span v-if="uploadedImages.length === 0" class="mt-1.5 text-[11px] font-medium leading-tight">
                {{ t('publish.modal.addImages') }}
              </span>
            </button>

            <!-- 空图片提示 -->
            <div v-if="uploadedImages.length === 0" class="flex flex-col justify-center pl-1 text-xs select-none">
              <span class="font-medium text-base-content/65">{{ t('publish.modal.uploadTip') }}</span>
              <span class="text-[11px] text-base-content/40">{{ t('publish.modal.uploadTipDetail') }}</span>
            </div>

            <!-- 隐藏的通用文件上传控件 -->
            <input
              ref="fileInput"
              type="file"
              accept="image/*"
              multiple
              class="hidden"
              @change="handleFileInputChange"
            />
          </div>

          <!-- 第二行：标题输入与字数计数器（如 0/30） -->
          <div class="shrink-0 pt-0.5">
            <div class="relative flex items-center justify-between gap-3">
              <input
                ref="titleInput"
                v-model="title"
                type="text"
                class="w-full text-base sm:text-lg font-bold placeholder:text-base-content/35 border-none bg-transparent outline-none focus:outline-none focus:ring-0 px-0 text-base-content transition"
                :placeholder="quickPublishType === 2 ? t('publish.modal.thoughtTitlePlaceholder') : typeMeta.placeholder"
                :maxlength="titleMaxLength"
                @focus="titleFocused = true"
                @blur="titleFocused = false"
                @keydown.enter.prevent="handleTitleEnter"
              />
              <span class="shrink-0 text-xs font-mono text-base-content/40 select-none">
                {{ title.length }}/{{ titleMaxLength }}
              </span>
            </div>
          </div>

          <!-- 极轻细微过渡线：聚焦时柔和过渡到主色调 -->
          <div
            class="shrink-0 h-px w-full transition-all duration-300"
            :class="titleFocused ? 'bg-primary/50 ring-1 ring-primary/20' : 'bg-line/40'"
          />

          <!-- 第三行：正文编辑器（弹性填满剩余空间，工具栏移到底部大拇指触控区，隐去传图按钮） -->
          <div class="gf-modal-editor relative flex-1 min-h-0 flex flex-col">
            <VditorOfficial
              ref="editor"
              v-model="content"
              :simple="true"
              :hide-upload="true"
              :placeholder="t('publish.modal.contentPlaceholder')"
              @upload="uploadImageFiles"
              @error="handleEditorError"
            />
          </div>

          <!-- 第四行：参考截图放置在正文/工具栏下方的“+ 添加分区及话题”药丸胶囊 -->
          <div class="shrink-0 flex items-center gap-2 pt-0.5">
            <PopoverRoot v-model:open="categoryPickerOpen">
              <PopoverTrigger as-child>
                <button
                  type="button"
                  class="group inline-flex items-center gap-1.5 sm:gap-2 rounded-full border border-line/80 bg-base-200/50 hover:bg-base-200/80 hover:border-primary/40 px-3 py-1.5 text-xs font-medium text-base-content/85 transition-all duration-150 active:scale-[0.96] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 cursor-pointer"
                  :aria-label="t('publish.modal.addCategoryAndTopic')"
                >
                  <Plus v-if="!selectedCategory" class="h-3.5 w-3.5 opacity-60 group-hover:opacity-100 transition-opacity" />
                  <span
                    v-if="selectedCategory"
                    class="h-2 w-2 rounded-full shrink-0 ring-1 ring-black/10 dark:ring-white/10 group-hover:scale-110 transition-transform duration-150"
                    :style="{ backgroundColor: selectedCategory.color || 'var(--gf-color-primary)' }"
                  />
                  <span>{{ selectedCategory ? selectedCategory.label : t('publish.modal.addCategoryAndTopic') }}</span>
                  <ChevronDown
                    class="h-3.5 w-3.5 opacity-60 group-hover:opacity-100 transition-transform duration-200 ease-out"
                    :class="{ 'rotate-180': categoryPickerOpen }"
                  />
                </button>
              </PopoverTrigger>

              <PopoverPortal>
                <PopoverContent
                  align="start"
                  :side-offset="6"
                  class="z-[100] w-56 max-w-[calc(100vw-2rem)] max-h-64 overflow-y-auto rounded-2xl border border-line/80 bg-base-100/98 p-1.5 shadow-xl backdrop-blur-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 duration-150 ease-out"
                >
                  <div class="px-2.5 py-1.5 text-[11px] font-semibold text-base-content/50 uppercase tracking-wider">
                    {{ t('publish.modal.selectCategory') }}
                  </div>
                  <button
                    v-for="cat in categories"
                    :key="cat.id"
                    type="button"
                    class="flex w-full items-center justify-between gap-2.5 rounded-xl px-2.5 py-2 text-xs text-left transition-all duration-150 hover:bg-base-200/80 active:scale-[0.98]"
                    :class="{ 'font-semibold text-primary bg-primary/8': categoryIds.includes(cat.id) }"
                    @click="selectCategory(cat.id)"
                  >
                    <div class="flex items-center gap-2 min-w-0">
                      <span
                        class="h-2 w-2 shrink-0 rounded-full"
                        :style="{ backgroundColor: cat.color || 'var(--gf-color-primary)' }"
                      />
                      <span class="truncate">{{ cat.label }}</span>
                    </div>
                    <Check v-if="categoryIds.includes(cat.id)" class="h-3.5 w-3.5 text-primary shrink-0" />
                  </button>
                </PopoverContent>
              </PopoverPortal>
            </PopoverRoot>
          </div>

          <!-- 验证码卡片（触发风控时展示） -->
          <div
            v-if="captchaRequired"
            class="shrink-0 gf-card flex flex-wrap items-center gap-2.5 sm:gap-3 p-2.5 sm:p-3 rounded-2xl border border-line bg-base-200/40 animate-in fade-in-0 duration-200"
          >
            <button
              type="button"
              class="relative h-9 w-24 shrink-0 overflow-hidden rounded-lg border border-line active:scale-95 transition-transform"
              :disabled="captchaLoading"
              @click="loadCaptcha()"
            >
              <Loader2 v-if="captchaLoading || !captchaImg" class="mx-auto h-4 w-4 animate-spin text-base-content/55" />
              <img v-else :src="captchaImg" :alt="t('auth.captchaAlt')" class="h-full w-full object-cover" />
            </button>
            <input
              v-model="captchaCode"
              type="text"
              maxlength="8"
              class="gf-input h-9 w-28 sm:w-32 px-2.5 text-sm rounded-lg"
              :placeholder="t('auth.captcha')"
              @keydown.enter.prevent="handleSubmit"
            />
          </div>

          <!-- 错误状态反馈 -->
          <p v-if="errorMessage" class="shrink-0 text-xs text-error font-medium px-1 animate-in fade-in-0 slide-in-from-top-1 duration-150">
            {{ errorMessage }}
          </p>
        </div>

        <!-- 弹层底部操作栏：弱底色、圆角平滑过渡、主次操作分明 -->
        <div class="px-4 sm:px-6 py-2.5 sm:py-3 border-t border-line/60 bg-base-200/20 flex items-center justify-between shrink-0">
          <!-- 左侧：图片处理提示与快捷键提示 -->
          <div class="text-xs text-base-content/50 flex items-center gap-2 min-w-0">
            <span v-if="uploading" class="inline-flex items-center gap-1.5 text-primary font-medium animate-pulse">
              <Loader2 class="h-3.5 w-3.5 animate-spin" />
              {{ t('publish.processingImage') }}
            </span>
            <kbd
              v-else
              class="hidden sm:inline-flex items-center gap-1 rounded-md border border-line/60 bg-base-200/60 px-1.5 py-0.5 text-[10px] font-mono text-base-content/50 select-none"
            >
              ⌘ + Enter / Ctrl + Enter
            </kbd>
          </div>

          <!-- 右侧：取消与立即发布按钮（严格遵循 active:scale-[0.96] 微反馈） -->
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="gf-button gf-button-secondary rounded-xl text-xs px-3.5 py-1.5 sm:px-4 sm:py-2 transition-all duration-150 hover:bg-base-200/80 active:scale-[0.96]"
              @click="closeQuickPublish()"
            >
              {{ t('publish.modal.cancel') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-primary rounded-xl text-xs px-4 py-1.5 sm:px-5 sm:py-2 inline-flex items-center gap-1.5 shadow-sm hover:shadow-md hover:brightness-105 active:scale-[0.96] transition-all duration-150 disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
              :disabled="submitting || uploading"
              @click="handleSubmit"
            >
              <Loader2 v-if="submitting || uploading" class="h-3.5 w-3.5 animate-spin" />
              <span>{{ submitting ? t('common.saving') : (isEditing ? t('common.save') : t('publish.modal.submit')) }}</span>
            </button>
          </div>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style>
/* 弹层内的编辑器去除生硬的多重外边框，让工具栏与正文完全融入画布 */
.gf-modal-editor {
  display: flex !important;
  flex-direction: column !important;
}
.gf-modal-editor .vditor {
  border: none !important;
  border-radius: 0 !important;
  background-color: transparent !important;
  display: flex !important;
  flex-direction: column !important;
  flex: 1 1 auto !important;
  min-height: 0 !important;
  height: 100% !important;
}
.gf-modal-editor .vditor-content {
  background-color: transparent !important;
  flex: 1 1 auto !important;
  min-height: 120px !important;
  overflow-y: auto !important;
  -webkit-overflow-scrolling: touch !important;
}
.gf-modal-editor .vditor-reset {
  padding: 8px 0 !important;
  background-color: transparent !important;
  min-height: 100px !important;
}
.gf-modal-editor .vditor-toolbar {
  border-top: 1px solid var(--color-line, rgba(0, 0, 0, 0.08)) !important;
  border-bottom: none !important;
  background-color: transparent !important;
  padding-left: 0 !important;
  padding-right: 0 !important;
  order: 2 !important;
  margin-top: auto !important;
}
.gf-modal-editor .vditor-toolbar--pin {
  background-color: var(--color-base-100, #fff) !important;
}
.gf-modal-editor .vditor-content {
  order: 1 !important;
}
.gf-modal-editor .vditor-toolbar__item > .vditor-panel {
  z-index: 100 !important;
  top: auto !important;
  bottom: calc(100% + 8px) !important;
}
.gf-modal-editor .vditor-toolbar__item > .vditor-panel.vditor-panel--arrow:before {
  top: auto !important;
  bottom: -14px !important;
  border-top-color: var(--panel-background-color, #fff) !important;
  border-bottom-color: transparent !important;
}
/* 桌面端短文弹层：emoji 位于工具栏靠右侧（SIMPLE_TOOLBAR），向左对齐展开，防止被右侧容器边界裁切 */
@media (min-width: 641px) {
  .gf-modal-editor .vditor-panel:has(.vditor-emojis),
  .gf-modal-editor .vditor-toolbar__item:has(> [data-type="emoji"]) > .vditor-panel,
  .gf-modal-editor [data-type="emoji"] ~ .vditor-panel {
    right: 0 !important;
    left: auto !important;
    max-width: min(340px, calc(100vw - 32px)) !important;
  }

  .gf-modal-editor .vditor-panel:has(.vditor-emojis).vditor-panel--arrow:before,
  .gf-modal-editor .vditor-toolbar__item:has(> [data-type="emoji"]) > .vditor-panel.vditor-panel--arrow:before,
  .gf-modal-editor [data-type="emoji"] ~ .vditor-panel.vditor-panel--arrow:before {
    left: auto !important;
    right: 12px !important;
  }
}

/* 移动端短文弹层：emoji 位于工具栏最左侧（SIMPLE_MOBILE_TOOLBAR 首项），向右对齐展开，杜绝向左溢出屏幕边界 */
@media (max-width: 640px) {
  .gf-modal-editor .vditor-panel:has(.vditor-emojis),
  .gf-modal-editor .vditor-toolbar__item:has(> [data-type="emoji"]) > .vditor-panel,
  .gf-modal-editor [data-type="emoji"] ~ .vditor-panel {
    left: 0 !important;
    right: auto !important;
    max-width: min(320px, calc(100vw - 32px)) !important;
  }

  .gf-modal-editor .vditor-panel:has(.vditor-emojis).vditor-panel--arrow:before,
  .gf-modal-editor .vditor-toolbar__item:has(> [data-type="emoji"]) > .vditor-panel.vditor-panel--arrow:before,
  .gf-modal-editor [data-type="emoji"] ~ .vditor-panel.vditor-panel--arrow:before {
    left: 10px !important;
    right: auto !important;
  }
}

.gf-modal-editor .vditor-emojis {
  max-height: 160px !important;
}

/* 移动端专属针对性优化：模仿参考截图将工具栏移到正文下方（更符合用户单手持握直觉），正文区弹性占满避免多余空白，单行精简工具，去除悬浮气泡 */
@media (max-width: 640px) {
  .gf-modal-editor {
    flex: 1 1 auto !important;
    min-height: 0 !important;
  }
  .gf-modal-editor .vditor {
    height: 100% !important;
    overflow: visible !important;
  }
  .gf-modal-editor .vditor-content {
    flex: 1 1 auto !important;
    min-height: 140px !important;
  }
  .gf-modal-editor .vditor-toolbar {
    display: flex !important;
    flex-wrap: wrap !important;
    overflow: visible !important;
    gap: 4px !important;
    padding: 6px 0 !important;
    align-items: center !important;
  }
  .gf-modal-editor .vditor-toolbar__item {
    flex: 0 0 auto !important;
    min-width: 36px !important;
    width: 36px !important;
    height: 36px !important;
    margin: 0 !important;
    padding: 0 !important;
  }
  .gf-modal-editor .vditor-toolbar__item .vditor-tooltipped {
    width: 36px !important;
    height: 36px !important;
    min-width: 36px !important;
    max-width: 36px !important;
    padding: 6px !important;
    display: flex !important;
    align-items: center !important;
    justify-content: center !important;
    border-radius: 8px !important;
    box-sizing: border-box !important;
  }
  .gf-modal-editor .vditor-toolbar__item .vditor-tooltipped svg {
    width: 18px !important;
    height: 18px !important;
  }
}

/* 桌面端编辑器给足基础舒适高度 */
@media (min-width: 641px) {
  .gf-modal-editor {
    min-height: 240px;
  }
  .gf-modal-editor .vditor {
    min-height: 240px;
  }
}

/* 移动触控设备上关闭悬浮 tooltip 黑气泡，避免遮挡按钮 */
@media (hover: none) {
  .gf-modal-editor .vditor-toolbar__item .vditor-tooltipped::after,
  .gf-modal-editor .vditor-toolbar__item .vditor-tooltipped::before {
    display: none !important;
  }
}

/* 传图预览横向滑动指示条：多图时清晰可见，触控/鼠标滑动舒适 */
.gf-image-scroll-track {
  scrollbar-width: thin;
  scrollbar-color: color-mix(in oklch, var(--gf-color-base-content) 25%, transparent) transparent;
}

.gf-image-scroll-track::-webkit-scrollbar {
  height: 5px;
}

.gf-image-scroll-track::-webkit-scrollbar-track {
  background: transparent;
  border-radius: 9999px;
}

.gf-image-scroll-track::-webkit-scrollbar-thumb {
  background: color-mix(in oklch, var(--gf-color-base-content) 25%, transparent);
  border-radius: 9999px;
}

.gf-image-scroll-track::-webkit-scrollbar-thumb:hover {
  background: color-mix(in oklch, var(--gf-color-base-content) 45%, transparent);
}
</style>
