<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronLeft, ChevronRight, ImagePlus, Loader2, Pencil, RotateCcw, Trash2 } from '@lucide/vue'
import type { StickerItem } from '@gooseforum/client'
import { isStickerUsable, MAX_MY_STICKERS, useMyStickers } from '@/site/composables/useMyStickers'
import { isPersonalStickerName } from '@/site/utils/sticker-token'

/**
 * Manage "My stickers": upload images, rename the private label, reorder and
 * remove. Used by the settings tab and by the dialog the picker opens. Uploads
 * wait for review; until then they are listed but cannot be inserted.
 */
const MAX_UPLOAD_BYTES = 4 << 20

const { t } = useI18n()
const mine = useMyStickers()
const { items, ready, loading, failed, signedOut, busy } = mine
const error = ref('')
const uploading = ref(0)
const renaming = ref<string | null>(null)
const renameDraft = ref('')
const confirmingRemove = ref<string | null>(null)
const dragFrom = ref<number | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const full = computed(() => items.value.length >= MAX_MY_STICKERS)

onMounted(() => {
  void mine.load(true)
})

function label(item: StickerItem) {
  if (item.displayName) return item.displayName
  return isPersonalStickerName(item.name) ? t('stickers.custom') : item.name
}

function statusText(item: StickerItem) {
  if (item.reviewStatus === 'pending') return t('stickers.mine.pending')
  if (item.reviewStatus === 'rejected') return t('stickers.mine.rejected')
  if (item.isEnabled === false) return t('stickers.mine.unavailable')
  return ''
}

async function run(action: () => Promise<unknown>, fallbackKey = 'stickers.mine.actionFailed') {
  error.value = ''
  try {
    await action()
    return true
  } catch (cause) {
    error.value = mine.errorText(cause, t(fallbackKey))
    return false
  }
}

async function onFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  const problems: string[] = []
  // One at a time: the server serializes library writes per account anyway.
  for (const file of files) {
    if (!file.type.startsWith('image/')) {
      problems.push(t('stickers.mine.notImage', { name: file.name }))
      continue
    }
    if (file.size > MAX_UPLOAD_BYTES) {
      problems.push(t('stickers.mine.tooLarge', { name: file.name }))
      continue
    }
    uploading.value += 1
    try {
      await mine.upload(file, file.name.replace(/\.[^.]+$/, '').slice(0, 64))
    } catch (cause) {
      problems.push(`${file.name}: ${mine.errorText(cause, t('stickers.mine.uploadFailed'))}`)
    } finally {
      uploading.value -= 1
    }
  }
  error.value = problems.join('\n')
}

async function startRename(item: StickerItem) {
  confirmingRemove.value = null
  renaming.value = item.name
  renameDraft.value = item.displayName || ''
  await nextTick()
  const field = document.getElementById(renameFieldId(item)) as HTMLInputElement | null
  field?.focus()
  field?.select()
}

function renameFieldId(item: StickerItem) {
  return `my-sticker-label-${item.name}`
}

async function submitRename(item: StickerItem) {
  if (renaming.value !== item.name) return
  const draft = renameDraft.value.trim()
  if (draft === (item.displayName || '')) {
    renaming.value = null
    return
  }
  if (await run(() => mine.rename(item, draft))) renaming.value = null
}

function cancelRename() {
  renaming.value = null
}

async function remove(item: StickerItem) {
  if (await run(() => mine.remove(item))) confirmingRemove.value = null
}

function move(from: number, to: number) {
  void run(() => mine.move(from, to), 'stickers.mine.orderFailed')
}

function onDrop(to: number) {
  const from = dragFrom.value
  dragFrom.value = null
  if (from !== null && from !== to) move(from, to)
}
</script>

<template>
  <div class="grid gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <p class="text-sm tabular-nums text-base-content/70">{{ t('stickers.mine.count', { count: items.length, limit: MAX_MY_STICKERS }) }}</p>
        <p class="mt-0.5 text-pretty text-xs text-base-content/55">{{ t('stickers.mine.uploadHint') }}</p>
      </div>
      <button
        type="button"
        class="gf-button gf-button-sm gf-button-primary shrink-0"
        :disabled="signedOut || full || uploading > 0 || busy"
        :title="full ? t('stickers.mine.full', { limit: MAX_MY_STICKERS }) : undefined"
        @click="fileInput?.click()"
      >
        <Loader2 v-if="uploading > 0" class="size-4 animate-spin" />
        <ImagePlus v-else class="size-4" />
        {{ uploading > 0 ? t('stickers.mine.uploading') : t('stickers.mine.upload') }}
      </button>
      <input ref="fileInput" type="file" accept="image/*" multiple class="hidden" data-test="my-sticker-upload" @change="onFiles" />
    </div>

    <p v-if="error" role="alert" class="whitespace-pre-line rounded-field bg-error/10 px-3 py-2 text-sm text-error">{{ error }}</p>

    <div v-if="loading && !ready" class="flex items-center justify-center gap-2 py-10 text-sm text-base-content/55">
      <Loader2 class="size-4 animate-spin" />
      {{ t('common.loadingShort') }}
    </div>
    <p v-else-if="signedOut" class="rounded-box bg-base-200/60 px-4 py-8 text-center text-sm text-base-content/65">{{ t('stickers.mine.signedOut') }}</p>
    <div v-else-if="failed" class="flex items-center justify-center gap-2 py-8 text-sm text-error">
      {{ t('api.stickersLoadFailed') }}
      <button type="button" class="gf-icon-button size-8" :aria-label="t('common.retry')" @click="mine.load(true)">
        <RotateCcw class="size-4" />
      </button>
    </div>
    <div v-else-if="items.length === 0" class="rounded-box bg-base-200/60 px-4 py-10 text-center">
      <p class="font-medium">{{ t('stickers.mine.emptyTitle') }}</p>
      <p class="mx-auto mt-1 max-w-sm text-pretty text-sm text-base-content/60">{{ t('stickers.mine.emptyBody') }}</p>
    </div>

    <ul v-else class="grid grid-cols-[repeat(auto-fill,minmax(7.25rem,1fr))] gap-3" data-test="my-sticker-list">
      <li
        v-for="(item, index) in items"
        :key="item.name"
        class="group flex flex-col gap-2 rounded-box bg-base-200/50 p-2 transition-[background-color,box-shadow] duration-150"
        :class="dragFrom !== null && dragFrom !== index ? 'hover:shadow-[0_0_0_2px_var(--gf-color-primary)]' : ''"
        :draggable="renaming !== item.name && !busy"
        @dragstart="dragFrom = index"
        @dragend="dragFrom = null"
        @dragover.prevent
        @drop.prevent="onDrop(index)"
      >
        <div class="relative flex aspect-square items-center justify-center rounded-field bg-base-100 p-2">
          <img
            :src="item.url"
            :alt="label(item)"
            loading="lazy"
            draggable="false"
            class="max-h-full max-w-full object-contain"
            :class="!isStickerUsable(item) && 'opacity-45 grayscale'"
          />
          <span v-if="statusText(item)" class="gf-badge gf-badge-warning absolute inset-x-1.5 bottom-1.5 justify-center truncate text-[11px]">
            {{ statusText(item) }}
          </span>
        </div>

        <form v-if="renaming === item.name" class="grid gap-1" @submit.prevent="submitRename(item)">
          <input
            :id="renameFieldId(item)"
            v-model="renameDraft"
            maxlength="64"
            autocomplete="off"
            class="gf-input h-8 px-2 text-base sm:text-xs"
            :aria-label="t('stickers.mine.labelField')"
            :placeholder="label(item)"
            :disabled="busy"
            @keydown.esc.stop.prevent="cancelRename"
            @blur="submitRename(item)"
          />
        </form>
        <p v-else class="truncate px-1 text-center text-xs font-medium" :title="label(item)">{{ label(item) }}</p>

        <div v-if="confirmingRemove === item.name" class="grid grid-cols-2 gap-1">
          <button type="button" class="gf-button gf-button-xs gf-button-danger" :disabled="busy" @click="remove(item)">{{ t('stickers.mine.removeConfirm') }}</button>
          <button type="button" class="gf-button gf-button-xs gf-button-ghost" :disabled="busy" @click="confirmingRemove = null">{{ t('common.cancel') }}</button>
        </div>
        <div v-else class="flex items-center justify-between gap-0.5 opacity-80 transition-opacity duration-150 group-hover:opacity-100 group-focus-within:opacity-100">
          <button type="button" class="gf-icon-button size-7" :disabled="busy || index === 0" :aria-label="t('stickers.mine.moveEarlier', { name: label(item) })" :title="t('stickers.mine.moveEarlier', { name: label(item) })" @click="move(index, index - 1)">
            <ChevronLeft class="size-3.5 rtl:-scale-x-100" />
          </button>
          <button type="button" class="gf-icon-button size-7" :disabled="busy" :aria-label="t('stickers.mine.rename', { name: label(item) })" :title="t('stickers.mine.rename', { name: label(item) })" @click="startRename(item)">
            <Pencil class="size-3.5" />
          </button>
          <button type="button" class="gf-icon-button size-7 hover:text-error" :disabled="busy" :aria-label="t('stickers.mine.remove', { name: label(item) })" :title="t('stickers.mine.remove', { name: label(item) })" @click="confirmingRemove = item.name">
            <Trash2 class="size-3.5" />
          </button>
          <button type="button" class="gf-icon-button size-7" :disabled="busy || index === items.length - 1" :aria-label="t('stickers.mine.moveLater', { name: label(item) })" :title="t('stickers.mine.moveLater', { name: label(item) })" @click="move(index, index + 1)">
            <ChevronRight class="size-3.5 rtl:-scale-x-100" />
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
