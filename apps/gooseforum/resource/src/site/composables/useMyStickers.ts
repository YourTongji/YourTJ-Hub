import { computed, ref } from 'vue'
import type { StickerItem } from '@gooseforum/client'
import {
  ApiResponseError,
  deleteMySticker,
  getMyStickers,
  orderMyStickers,
  saveMyStickerEntry,
  uploadImageFile,
} from '@/runtime/api'

/**
 * The viewer's private sticker library ("My stickers"). Every picker and the
 * settings page share one module-level copy, like useStickerLibrary. Writes wait
 * for the server, so membership never looks changed before it is saved; a
 * rejected reorder reloads the list because another device may have changed it.
 */

/** Server limit (stickerservice.MaxLibraryItems). */
export const MAX_MY_STICKERS = 200

const items = ref<StickerItem[]>([])
const ready = ref(false)
const loading = ref(false)
const failed = ref(false)
const signedOut = ref(false)
const busy = ref(false)
let request: Promise<void> | null = null

/** Pending, rejected or withdrawn stickers stay listed but cannot be inserted. */
export function isStickerUsable(item: StickerItem) {
  return item.isEnabled !== false && (item.reviewStatus ?? 'approved') === 'approved'
}

function errorText(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function useMyStickers() {
  const usable = computed(() => items.value.filter(isStickerUsable))
  const names = computed(() => new Set(items.value.map((item) => item.name)))

  function load(refresh = false): Promise<void> {
    if (request) return request
    if (ready.value && !refresh) return Promise.resolve()
    loading.value = true
    failed.value = false
    request = getMyStickers()
      .then((next) => {
        items.value = next
        ready.value = true
        signedOut.value = false
      })
      .catch((error) => {
        if (error instanceof ApiResponseError && error.messageCode === 'auth.required') {
          signedOut.value = true
          items.value = []
        } else {
          failed.value = true
        }
      })
      .finally(() => {
        loading.value = false
        request = null
      })
    return request
  }

  async function write<T>(action: () => Promise<T>): Promise<T> {
    if (busy.value) throw new Error('busy')
    busy.value = true
    try {
      return await action()
    } finally {
      busy.value = false
    }
  }

  function upsert(item: StickerItem) {
    const index = items.value.findIndex((entry) => entry.name === item.name)
    items.value = index < 0
      ? [...items.value, item]
      : items.value.map((entry, position) => (position === index ? item : entry))
  }

  /** Save a sticker someone shared; saving one already collected is a no-op. */
  function collect(stickerName: string) {
    return write(async () => {
      const item = await saveMyStickerEntry({ stickerName })
      if (ready.value) upsert(item)
      return item
    })
  }

  /** Upload one image as a personal sticker; it waits for review before use. */
  function upload(file: File, displayName = '') {
    return write(async () => {
      const fileName = await uploadImageFile(file, file.name)
      const item = await saveMyStickerEntry({ fileName, ...(displayName.trim() ? { displayName: displayName.trim() } : {}) })
      upsert(item)
      return item
    })
  }

  /** The label is private to this account; an empty label shows the sticker's own name. */
  function rename(item: StickerItem, displayName: string) {
    return write(async () => {
      upsert(await saveMyStickerEntry({ stickerName: item.name, displayName: displayName.trim() }))
    })
  }

  function remove(item: StickerItem) {
    return write(async () => {
      await deleteMySticker(item.name)
      items.value = items.value.filter((entry) => entry.name !== item.name)
    })
  }

  /** Move one sticker to another position and save the complete order. */
  function move(from: number, to: number) {
    return write(async () => {
      if (from === to || to < 0 || to >= items.value.length) return
      const ordered = [...items.value]
      ordered.splice(to, 0, ...ordered.splice(from, 1))
      try {
        await orderMyStickers(ordered.map((entry) => entry.name))
        items.value = ordered
      } catch (error) {
        void load(true)
        throw error
      }
    })
  }

  return {
    items,
    usable,
    names,
    ready,
    loading,
    failed,
    signedOut,
    busy,
    load,
    collect,
    upload,
    rename,
    remove,
    move,
    errorText,
  }
}
