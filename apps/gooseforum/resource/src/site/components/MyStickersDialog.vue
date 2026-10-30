<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { X } from '@lucide/vue'
import { DialogClose, DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import MyStickersManager from '@/site/components/MyStickersManager.vue'
import { myStickersDialogOpen } from '@/site/composables/useMyStickersDialog'

const { t } = useI18n()
</script>

<template>
  <DialogRoot v-model:open="myStickersDialogOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40 duration-150 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
      <DialogContent
        data-test="my-stickers-dialog"
        class="fixed left-1/2 top-1/2 z-[2101] flex max-h-[min(40rem,calc(100dvh-2rem))] w-[calc(100%-2rem)] max-w-2xl -translate-x-1/2 -translate-y-1/2 flex-col rounded-box bg-base-100 text-base-content shadow-2xl outline-none duration-150 ease-out data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
      >
        <div class="flex items-start justify-between gap-3 px-5 pb-3 pt-5">
          <div class="min-w-0">
            <DialogTitle class="text-base font-semibold">{{ t('stickers.mine.title') }}</DialogTitle>
            <DialogDescription class="mt-1 text-pretty text-sm text-base-content/65">{{ t('stickers.mine.description') }}</DialogDescription>
          </div>
          <DialogClose class="gf-icon-button size-8 shrink-0" :aria-label="t('common.close')">
            <X class="size-4" />
          </DialogClose>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto px-5 pb-5">
          <MyStickersManager />
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
