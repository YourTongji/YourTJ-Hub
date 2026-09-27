<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { PenLine, RotateCw, X } from '@lucide/vue'
import { privateNotes } from '@/runtime/private-notes'

const props = defineProps<{ userId: number; username: string }>()
const { t } = useI18n()
const open = ref(false)
const busy = ref(false)
const error = ref('')
const value = ref('')
const trigger = ref<HTMLButtonElement | null>(null)
const panel = ref<HTMLDivElement | null>(null)
const input = ref<HTMLInputElement | null>(null)
const panelStyle = ref<Record<string, string>>({ top: '0px', left: '16px', width: 'min(22rem, calc(100vw - 2rem))' })
const ready = computed(() => privateNotes.state.loaded && !privateNotes.state.loading && !privateNotes.state.failed)
const existing = computed(() => privateNotes.state.notes.get(props.userId)?.note ?? '')

watch(() => [props.userId, privateNotes.state.ownerId], () => {
  open.value = false
  error.value = ''
})

function positionPanel() {
  if (!open.value) return
  const anchor = trigger.value?.getBoundingClientRect()
  if (!anchor || !panel.value) return

  const margin = 16
  const width = Math.min(352, Math.max(0, window.innerWidth - margin * 2))
  const rightLimit = Math.max(margin, window.innerWidth - width - margin)
  const left = Math.min(Math.max(anchor.left, margin), rightLimit)
  const height = panel.value.getBoundingClientRect().height
  let top = anchor.bottom + 8
  if (top + height > window.innerHeight - margin) top = anchor.top - height - 8
  const bottomLimit = Math.max(margin, window.innerHeight - height - margin)
  top = Math.min(Math.max(top, margin), bottomLimit)
  panelStyle.value = { top: `${top}px`, left: `${left}px`, width: `${width}px` }
}

async function edit() {
  if (!ready.value) return
  value.value = existing.value
  error.value = ''
  open.value = true
  await nextTick()
  positionPanel()
  input.value?.focus()
}

function close(restoreFocus = false) {
  if (busy.value) return
  open.value = false
  if (restoreFocus) void nextTick(() => trigger.value?.focus())
}

function toggle() {
  if (open.value) close(true)
  else void edit()
}

function onDocumentPointerDown(event: PointerEvent) {
  const target = event.target
  if (!(target instanceof Node)) return
  if (trigger.value?.contains(target) || panel.value?.contains(target)) return
  close()
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && open.value) {
    event.stopPropagation()
    close(true)
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocumentPointerDown)
  window.addEventListener('resize', positionPanel)
  window.addEventListener('scroll', positionPanel, true)
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  window.removeEventListener('resize', positionPanel)
  window.removeEventListener('scroll', positionPanel, true)
  window.removeEventListener('keydown', onKeydown)
})

async function save() {
  if (!ready.value || busy.value || Array.from(value.value.trim()).length > 64) return
  busy.value = true
  error.value = ''
  try {
    await privateNotes.update(props.userId, props.username, value.value)
    busy.value = false
    close(true)
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('api.operationFailed')
  } finally {
    busy.value = false
  }
}
</script>
<template>
  <div v-if="privateNotes.state.ownerId && privateNotes.state.ownerId !== userId" class="inline-flex shrink-0 items-center">
    <button
      ref="trigger"
      type="button"
      data-testid="private-note-edit"
      class="inline-flex h-8 w-8 items-center justify-center rounded-md text-base-content/55 transition hover:bg-base-200 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 disabled:cursor-not-allowed disabled:opacity-45"
      :class="existing ? 'text-primary' : ''"
      :aria-label="t('privateNote.edit')"
      :title="t('privateNote.edit')"
      :aria-expanded="open"
      aria-haspopup="dialog"
      :disabled="!ready"
      @click="toggle"
    >
      <PenLine class="h-4 w-4" aria-hidden="true" />
    </button>
    <button
      v-if="privateNotes.state.failed && !open"
      type="button"
      data-testid="private-note-retry"
      class="ml-1 inline-flex h-8 w-8 items-center justify-center rounded-md text-base-content/55 transition hover:bg-base-200 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
      :aria-label="t('common.retry')"
      :title="t('common.retry')"
      @click="privateNotes.refresh()"
    >
      <RotateCw class="h-4 w-4" aria-hidden="true" />
    </button>

    <Teleport to="body">
      <div
        v-if="open"
        ref="panel"
        role="dialog"
        :aria-label="t('privateNote.edit')"
        class="gf-menu-surface fixed z-[120] max-h-[calc(100vh-2rem)] overflow-y-auto rounded-xl border border-line bg-base-100 p-4 text-base-content shadow-xl"
        :style="panelStyle"
        @pointerdown.stop
        @keydown.esc.stop.prevent="close(true)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="text-sm font-semibold">{{ t('privateNote.edit') }}</p>
            <p class="mt-1 text-xs leading-relaxed text-base-content/60">{{ t('privateNote.hint') }}</p>
          </div>
          <button
            type="button"
            class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-base-content/55 hover:bg-base-200 hover:text-base-content focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
            :aria-label="t('common.cancel')"
            @click="close(true)"
          >
            <X class="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <form class="mt-3 space-y-3" @submit.prevent="save">
          <label class="block text-sm font-medium">
            {{ t('privateNote.label') }}
            <input
              ref="input"
              v-model="value"
              type="text"
              class="gf-input mt-1 w-full"
              :disabled="busy || !ready"
              :aria-label="t('privateNote.label')"
            />
          </label>
          <p class="text-xs text-base-content/60">{{ Array.from(value.trim()).length }}/64</p>
          <div class="flex gap-2">
            <button type="submit" class="gf-button gf-button-primary min-h-10 flex-1" :disabled="busy || !ready || Array.from(value.trim()).length > 64">
              {{ t('common.save') }}
            </button>
            <button type="button" class="gf-button min-h-10" :disabled="busy" @click="close(true)">
              {{ t('common.cancel') }}
            </button>
          </div>
          <p v-if="error" role="alert" class="text-sm text-error">{{ error }}</p>
        </form>
      </div>
    </Teleport>
  </div>
</template>
