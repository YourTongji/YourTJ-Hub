<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CircleCheck, CircleX, Info, TriangleAlert, X } from '@lucide/vue'
import { dismiss, useFlashMessages, type FlashMessageType } from '@/runtime/flash-message'

const { messages } = useFlashMessages()
const { t } = useI18n()

const visibleMessages = computed(() => messages.value)

// 图标底色：类型色 12% 透明度圆底，弱化色块、突出可读性
function tintClass(type: FlashMessageType) {
  switch (type) {
    case 'success':
      return 'bg-success/12'
    case 'warning':
      return 'bg-warning/12'
    case 'error':
      return 'bg-error/12'
    default:
      return 'bg-primary/12'
  }
}

function iconFor(type: FlashMessageType) {
  switch (type) {
    case 'success':
      return CircleCheck
    case 'warning':
      return TriangleAlert
    case 'error':
      return CircleX
    default:
      return Info
  }
}

function iconClass(type: FlashMessageType) {
  switch (type) {
    case 'success':
      return 'text-success'
    case 'warning':
      return 'text-warning'
    case 'error':
      return 'text-error'
    default:
      return 'text-primary'
  }
}
</script>

<template>
  <div
    class="pointer-events-none fixed inset-x-0 bottom-4 z-[120] px-4 sm:inset-x-auto sm:bottom-auto sm:right-5 sm:top-20 sm:px-0 lg:right-8"
    aria-live="polite"
    aria-atomic="true"
  >
    <TransitionGroup
      name="gf-flash"
      tag="div"
      class="mx-auto flex w-full max-w-[400px] flex-col items-stretch gap-2 sm:mx-0 sm:items-end"
    >
      <div
        v-for="item in visibleMessages"
        :key="item.id"
        class="gf-alert pointer-events-auto relative flex w-full items-center gap-3 overflow-hidden py-3 pl-3 pr-2 text-sm sm:max-w-[380px]"
        role="status"
      >
        <span
          class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full"
          :class="tintClass(item.type)"
          aria-hidden="true"
        >
          <component :is="iconFor(item.type)" class="h-5 w-5" :class="iconClass(item.type)" />
        </span>
        <p class="min-w-0 flex-1 leading-5 text-base-content">{{ item.message }}</p>
        <button
          type="button"
          class="-mr-1 inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-icon-muted transition hover:bg-base-300 hover:text-base-content"
          :aria-label="t('flash.close')"
          @click="dismiss(item.id)"
        >
          <X class="h-4 w-4" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>
