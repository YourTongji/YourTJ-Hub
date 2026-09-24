<script setup lang="ts">
import { reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { scheduleSync } from '@/site/composables/useScheduleSync'
const { t } = useI18n()
const choices = reactive<Record<string, Record<string, 'local' | 'remote'>>>({})
function describe(value: unknown): string {
  if (value == null) return t('planSync.deleted')
  if (typeof value !== 'object') return String(value)
  if (Array.isArray(value)) return value.map(describe).join(', ')
  const item = value as Record<string, unknown>
  if (item.course)
    return `${(item.course as { courseName: string }).courseName}: ${describe(item.selected)}`
  if (item.courses)
    return [
      item.name,
      ...Object.values(item.courses as object).map(describe),
      ...Object.values(item.events as object).map(describe),
    ].join('\n')
  return [item.label, item.day, item.sections, item.weeks]
    .filter((v) => v != null)
    .map(describe)
    .join(' · ')
}
function label(path: string[]) {
  if (!path.length) return t('planSync.plan')
  if (path[0] === 'courses') return `${t('planSync.course')} ${path[1]}`
  const key = path.at(-1) ?? 'plan'
  return t(
    `planSync.${['name', 'createdAt', 'label', 'day', 'sections', 'weeks'].includes(key) ? key : 'event'}`,
  )
}
function choiceKey(id: string, revision?: number) {
  return `${id}:${revision ?? 'deleted'}`
}
</script>
<template>
  <section
    v-if="
      scheduleSync.mergeBlocked.value ||
      scheduleSync.conflicts.value.length ||
      Object.keys(scheduleSync.drafts.value).length ||
      scheduleSync.needsOwnerConfirmation()
    "
    class="mb-4 space-y-3"
    aria-live="polite"
  >
    <p v-if="scheduleSync.mergeBlocked.value" class="text-sm text-warning">
      {{
        t(
          scheduleSync.mergeBlockedReason.value === 'rejected'
            ? 'planSync.rejected'
            : 'planSync.capacity',
        )
      }}
    </p>
    <div v-if="scheduleSync.needsOwnerConfirmation()" class="rounded-lg border border-line p-4">
      <p class="mb-2 text-sm">{{ t('planSync.adoptHint') }}</p>
      <button class="gf-button gf-button-primary min-h-11" @click="scheduleSync.saveNow(true)">
        {{ t('planSync.adopt') }}
      </button>
    </div>
    <div
      v-for="conflict in scheduleSync.conflicts.value"
      :key="choiceKey(conflict.id, conflict.remote?.revision)"
      class="space-y-3 rounded-lg border border-warning p-4"
    >
      <h2 class="font-semibold">
        {{ t('planSync.title') }} ·
        {{ conflict.local?.name || conflict.remote?.plan.name || conflict.base?.name }}
      </h2>
      <p class="text-sm">{{ t('planSync.body') }}</p>
      <fieldset
        v-for="field in conflict.fields"
        :key="JSON.stringify(field.path)"
        class="space-y-2"
      >
        <legend class="text-sm font-medium">{{ label(field.path) }}</legend>
        <label
          v-for="side in ['local', 'remote'] as const"
          :key="side"
          class="flex min-h-11 gap-2 rounded border border-line p-2"
        >
          <input
            type="radio"
            :name="encodeURIComponent(`${conflict.id}:${JSON.stringify(field.path)}`)"
            :value="side"
            :checked="
              choices[choiceKey(conflict.id, conflict.remote?.revision)]?.[
                JSON.stringify(field.path)
              ] === side
            "
            @change="
              (choices[choiceKey(conflict.id, conflict.remote?.revision)] ??= {})[
                JSON.stringify(field.path)
              ] = side
            "
          />
          <span class="min-w-0 whitespace-pre-wrap break-words text-sm"
            >{{ t(`planSync.${side}`) }}: {{ describe(field[side]) }}</span
          >
        </label>
      </fieldset>
      <button
        class="gf-button gf-button-primary min-h-11"
        :disabled="
          conflict.fields.some(
            (f) =>
              !choices[choiceKey(conflict.id, conflict.remote?.revision)]?.[JSON.stringify(f.path)],
          )
        "
        @click="
          scheduleSync.resolveConflict(
            conflict.id,
            choices[choiceKey(conflict.id, conflict.remote?.revision)] || {},
          )
        "
      >
        {{ t('planSync.apply') }}
      </button>
    </div>
    <div
      v-if="Object.keys(scheduleSync.drafts.value).length"
      class="rounded-lg border border-line p-4"
    >
      <h2 class="font-semibold">{{ t('planSync.drafts') }}</h2>
      <p class="text-sm">{{ t('planSync.draftHint') }}</p>
      <div
        v-for="(draft, id) in scheduleSync.drafts.value"
        :key="id"
        class="mt-2 flex flex-wrap items-center justify-between gap-2"
      >
        <span>{{ draft.name }}</span
        ><button class="gf-button min-h-11" @click="scheduleSync.restoreDraft(String(id))">
          {{ t('planSync.restore') }}
        </button>
      </div>
    </div>
  </section>
</template>
