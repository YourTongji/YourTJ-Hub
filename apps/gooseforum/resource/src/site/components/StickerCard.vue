<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bookmark, BookmarkCheck, X } from '@lucide/vue'
import { DialogClose, DialogContent, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { ApiResponseError, saveMySticker } from '@/runtime/api'
import { useStickerLibrary } from '@/site/composables/useStickerLibrary'
import type { MarkdownSticker } from '@/runtime/markdown-image-preview'

// 正文贴纸的轻量小窗：放大查看、显示名称、收藏到我的表情。与普通图片灯箱分开，
// 不进入相册翻页；收藏成功后原地确认，不弹额外提示。
const props = defineProps<{ signedIn: boolean }>()
const emit = defineEmits<{ signIn: [] }>()
const { t } = useI18n()
const { stickers, ensureStickers } = useStickerLibrary()

const sticker = ref<MarkdownSticker | null>(null)
const open = ref(false)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
// 同一次浏览内已收藏过的贴纸再次打开时直接显示已收藏。
const savedNames = new Set<string>()

const label = computed(() => {
  const name = sticker.value?.name || ''
  if (/^u_[0-9a-f]{48}$/.test(name)) return t('stickers.custom')
  return stickers.value.find((item) => item.name === name)?.displayName || name
})

function show(next: MarkdownSticker) {
  sticker.value = next
  saved.value = savedNames.has(next.name)
  saving.value = false
  error.value = ''
  open.value = true
  void ensureStickers()
}

async function save() {
  const current = sticker.value
  if (!current || saving.value || saved.value) return
  if (!props.signedIn) {
    emit('signIn')
    return
  }
  saving.value = true
  error.value = ''
  try {
    await saveMySticker(current.name)
    savedNames.add(current.name)
    if (sticker.value === current) saved.value = true
  } catch (cause) {
    if (sticker.value === current) {
      error.value = cause instanceof ApiResponseError ? cause.message : t('stickers.saveFailed')
    }
  } finally {
    if (sticker.value === current) saving.value = false
  }
}

defineExpose({ open: show })
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2100] bg-black/30 duration-150 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
      <DialogContent
        v-if="sticker"
        data-test="sticker-card"
        :aria-describedby="undefined"
        class="gf-sticker-card fixed left-1/2 top-1/2 z-[2101] w-[min(18rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-3xl bg-base-100 p-5 text-center text-base-content outline-none duration-150 ease-out data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
      >
        <DialogClose
          class="absolute right-2.5 top-2.5 flex size-8 items-center justify-center rounded-full text-base-content/55 transition-colors hover:bg-base-200 hover:text-base-content focus-visible:outline-2 focus-visible:outline-primary"
          :aria-label="t('common.close')"
        >
          <X class="size-4" />
        </DialogClose>
        <img
          :src="sticker.src"
          alt=""
          draggable="false"
          class="mx-auto mt-3 size-40 object-contain"
        />
        <DialogTitle class="mt-4 truncate text-base font-semibold leading-6">{{ label }}</DialogTitle>
        <button
          type="button"
          data-test="sticker-card-save"
          class="gf-button gf-button-md mt-5 w-full active:scale-[0.96]"
          :class="saved ? 'gf-button-secondary pointer-events-none' : 'gf-button-primary'"
          :disabled="saving"
          :aria-disabled="saved"
          :aria-busy="saving"
          @click="save"
        >
          <BookmarkCheck v-if="saved" class="size-4" />
          <Bookmark v-else class="size-4" />
          {{ saved ? t('stickers.saved') : props.signedIn ? t('stickers.save') : t('stickers.signInToSave') }}
        </button>
        <p
          v-if="error"
          data-test="sticker-card-error"
          role="alert"
          class="mt-2 text-xs leading-5 text-error"
        >
          {{ error }}
        </p>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
.gf-sticker-card {
  box-shadow:
    0 0 0 1px oklch(0 0 0 / 0.06),
    0 12px 32px -8px oklch(0 0 0 / 0.22);
}

:global([data-theme='gf-dark']) .gf-sticker-card {
  box-shadow:
    0 0 0 1px oklch(1 0 0 / 0.08),
    0 12px 32px -8px oklch(0 0 0 / 0.6);
}
</style>
