<script setup lang="ts">
import { ClipboardList, Clock, FileText, PenLine, UserRound, X, Zap } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { COURSE_REVIEW_TEMPLATES } from '@/site/utils/course-review-templates'

defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
  select: [payload: { id: string; content: string }]
}>()

const { t } = useI18n()

const templateIcons: Record<string, typeof FileText> = {
  comprehensive: FileText,
  quick: Zap,
  'teacher-focused': UserRound,
  'exam-focused': ClipboardList,
  workload: Clock,
  blank: PenLine,
}

function iconFor(id: string) {
  return templateIcons[id] ?? FileText
}

function onSelect(id: string, content: string) {
  emit('select', { id, content })
}
</script>

<template>
  <Teleport to="body">
    <Transition name="gf-modal">
    <div
      v-if="open"
      class="fixed inset-0 z-[80] flex items-center justify-center bg-black/40 p-4"
      role="dialog"
      aria-modal="true"
      :aria-label="t('courseDetailPage.templateSelectorTitle')"
      @click.self="emit('close')"
    >
      <div class="gf-panel w-full max-w-2xl max-h-[80vh] overflow-y-auto p-5 shadow-lg">
        <div class="flex items-start justify-between gap-3">
          <div>
            <h2 class="text-sm font-semibold text-base-content">
              {{ t('courseDetailPage.templateSelectorTitle') }}
            </h2>
            <p class="mt-0.5 text-[13px] text-base-content/55">
              {{ t('courseDetailPage.templateSelectorHint') }}
            </p>
          </div>
          <button
            type="button"
            class="rounded-md p-1 text-base-content/55 transition hover:bg-base-300 hover:text-base-content/75"
            :aria-label="t('common.close')"
            @click="emit('close')"
          >
            <X class="h-4 w-4" />
          </button>
        </div>

        <div class="mt-4 grid gap-3 sm:grid-cols-2">
          <button
            v-for="template in COURSE_REVIEW_TEMPLATES"
            :key="template.id"
            type="button"
            class="group flex items-start gap-3 rounded-[var(--gf-radius-box)] border border-line/70 bg-base-200/45 p-4 text-left transition hover:border-primary/40 hover:bg-info/10"
            @click="onSelect(template.id, template.content)"
          >
            <span
              class="mt-0.5 inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-base-300/70 text-base-content/55 transition group-hover:bg-primary/15 group-hover:text-primary"
            >
              <component :is="iconFor(template.id)" class="h-5 w-5" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="block text-sm font-semibold text-base-content">{{ t(template.nameKey) }}</span>
              <span class="mt-0.5 block text-[12px] leading-5 text-base-content/55">{{ t(template.descriptionKey) }}</span>
            </span>
          </button>
        </div>
      </div>
    </div>
    </Transition>
  </Teleport>
</template>
