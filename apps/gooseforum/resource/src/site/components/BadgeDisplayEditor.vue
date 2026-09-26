<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Draggable from 'vuedraggable'
import { GripVertical } from '@lucide/vue'
import type { UserBadgePayload } from '@gooseforum/client'
import { displayBadges } from '@/runtime/api'
import { badgeClass, badgeIconURL } from '@/site/utils/badge-style'

const props = defineProps<{ badges: UserBadgePayload[]; selected: UserBadgePayload[] }>()
const { t } = useI18n()
const codes = ref(props.selected.map(badge => badge.code))
const saving = ref(false)
const error = ref('')
const saved = ref(false)

watch(() => props.selected, value => { codes.value = value.map(badge => badge.code) })

const selectedBadges = computed<UserBadgePayload[]>({
  get: () => codes.value
    .map(code => props.badges.find(badge => badge.code === code))
    .filter((badge): badge is UserBadgePayload => !!badge),
  set: badges => {
    codes.value = badges.map(badge => badge.code)
    saved.value = false
  },
})

const availableBadges = computed(() => props.badges.filter(badge => !codes.value.includes(badge.code)))

function toggle(code: string) {
  if (saving.value) return
  saved.value = false
  if (codes.value.includes(code)) {
    codes.value = codes.value.filter(selectedCode => selectedCode !== code)
  } else if (codes.value.length < 5) {
    codes.value = [...codes.value, code]
  }
  void nextTick(() => document.getElementById(`profile-badge-${code}`)?.focus())
}

function move(index: number, delta: number) {
  const next = index + delta
  if (saving.value || next < 0 || next >= codes.value.length) return
  const copy = [...codes.value]
  ;[copy[index], copy[next]] = [copy[next], copy[index]]
  codes.value = copy
  saved.value = false
}

async function save() {
  if (saving.value) return
  saving.value = true
  saved.value = false
  error.value = ''
  try {
    await displayBadges([...codes.value])
    saved.value = true
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('api.badgeWearFailed')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="mt-4 space-y-3 border-t border-line pt-3">
    <h3 class="text-sm font-semibold">{{ t('badgeDisplay.title') }}</h3>
    <p class="text-xs text-base-content/60">{{ t('badgeDisplay.hint') }}</p>

    <div class="space-y-2">
      <h4 id="badge-display-selected-label" class="text-xs font-medium text-base-content/70">
        {{ t('badgeDisplay.selected', { count: selectedBadges.length }) }}
      </h4>
      <Draggable
        v-if="selectedBadges.length"
        v-model="selectedBadges"
        item-key="code"
        handle=".badge-display-editor__drag-handle"
        :animation="180"
        :disabled="saving || selectedBadges.length < 2"
        :delay="180"
        :delay-on-touch-only="true"
        :touch-start-threshold="10"
        ghost-class="opacity-35"
        class="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3"
        role="list"
        aria-labelledby="badge-display-selected-label"
      >
        <template #item="{ element: badge, index }">
          <div
            class="group inline-flex min-h-12 min-w-0 items-center gap-2 rounded-lg border border-primary/45 bg-primary/5 px-3 text-sm transition-colors hover:border-primary/70"
            role="listitem"
          >
            <input
              :id="`profile-badge-${badge.code}`"
              type="checkbox"
              class="accent-primary"
              :checked="true"
              :disabled="saving"
              @change="toggle(badge.code)"
            />
            <label :for="`profile-badge-${badge.code}`" class="inline-flex min-w-0 flex-1 cursor-pointer items-center gap-2 py-2">
              <span class="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded bg-primary/10 text-[10px] font-semibold text-primary" aria-hidden="true">
                {{ index + 1 }}
              </span>
              <span
                class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full"
                :class="badgeClass(badge.color, badge.level)"
              >
                <img :src="badgeIconURL(badge)" alt="" class="h-4 w-4" />
              </span>
              <span class="min-w-0 break-words">{{ badge.name }}</span>
            </label>
            <button
              type="button"
              class="badge-display-editor__drag-handle inline-flex h-8 w-8 shrink-0 cursor-grab touch-none items-center justify-center rounded-md text-base-content/50 transition-colors hover:bg-base-content/5 hover:text-base-content active:cursor-grabbing focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
              :disabled="saving || selectedBadges.length < 2"
              :aria-label="t('badgeDisplay.reorder', { name: badge.name })"
              aria-keyshortcuts="Alt+ArrowUp Alt+ArrowDown"
              @click.stop.prevent
              @keydown.alt.up.stop.prevent="move(index, -1)"
              @keydown.alt.down.stop.prevent="move(index, 1)"
            >
              <GripVertical aria-hidden="true" class="h-4 w-4" />
            </button>
          </div>
        </template>
      </Draggable>
      <p v-else class="text-xs text-base-content/55">{{ t('badgeDisplay.empty') }}</p>
    </div>

    <div v-if="availableBadges.length" class="space-y-2">
      <h4 id="badge-display-available-label" class="text-xs font-medium text-base-content/70">{{ t('badgeDisplay.available') }}</h4>
      <div
        class="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3"
        role="list"
        aria-labelledby="badge-display-available-label"
      >
        <div
          v-for="badge in availableBadges"
          :key="badge.code"
          class="inline-flex min-h-12 min-w-0 items-center gap-2 rounded-lg border border-line px-3 text-sm transition-colors hover:border-base-content/30 hover:bg-base-content/[0.03]"
          role="listitem"
        >
          <input
            :id="`profile-badge-${badge.code}`"
            type="checkbox"
            class="accent-primary"
            :checked="false"
            :disabled="saving || selectedBadges.length >= 5"
            @change="toggle(badge.code)"
          />
          <label :for="`profile-badge-${badge.code}`" class="inline-flex min-w-0 flex-1 cursor-pointer items-center gap-2 py-2">
            <span
              class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full"
              :class="badgeClass(badge.color, badge.level)"
            >
              <img :src="badgeIconURL(badge)" alt="" class="h-4 w-4" />
            </span>
            <span class="min-w-0 break-words">{{ badge.name }}</span>
          </label>
        </div>
      </div>
    </div>

    <button type="button" class="gf-button gf-button-primary" :disabled="saving" @click="save">
      {{ t('badgeDisplay.save') }}
    </button>
    <p v-if="error" role="alert" class="text-sm text-error">{{ error }}</p>
    <p v-if="saved" role="status" class="text-sm text-success">{{ t('settings.status.badgeSaved') }}</p>
  </section>
</template>
