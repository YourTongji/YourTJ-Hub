<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Check, Info, Loader2, Lock } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import SiteSelect from '@/site/components/SiteSelect.vue'
import {
  getPrivacySettings,
  updatePrivacySettings,
  type ProfileAudience,
  type ProfilePrivacySection,
  type UserPrivacySettings,
} from '@/runtime/api'
import {
  clearLegacyPrivacy,
  PROFILE_PRIVACY_SECTIONS,
  readLegacyPrivacy,
  widensAudience,
} from '@/runtime/profile-privacy'

const props = defineProps<{ profileUrl: string }>()
const { t } = useI18n()

const settings = ref<UserPrivacySettings | null>(null)
const loading = ref(true)
const loadFailed = ref(false)
const saving = ref<ProfilePrivacySection | 'legacy' | null>(null)
const failedSection = ref<ProfilePrivacySection | 'legacy' | null>(null)
const pendingWiden = ref<{ key: ProfilePrivacySection, audience: ProfileAudience } | null>(null)
const legacyUpdate = ref<Partial<UserPrivacySettings> | null>(null)
const widenCancel = ref<HTMLButtonElement | null>(null)
const root = ref<HTMLElement | null>(null)
const widenReturnKey = ref<ProfilePrivacySection | null>(null)
// Saves confirm on the row itself; a page-level banner would shift the
// layout under the select the user is still working with.
const savedSection = ref<ProfilePrivacySection | 'legacy' | null>(null)
const announcement = ref('')
let savedTimer: number | undefined

function markSaved(target: ProfilePrivacySection | 'legacy') {
  window.clearTimeout(savedTimer)
  savedSection.value = target
  announcement.value = t('settings.status.profileVisibilitySaved')
  savedTimer = window.setTimeout(() => { savedSection.value = null }, 2000)
}

onBeforeUnmount(() => window.clearTimeout(savedTimer))

// Start the confirmation on the safe choice so Enter never widens by accident.
function focusWidenCancel(event: Event) {
  event.preventDefault()
  widenCancel.value?.focus()
}

// Return focus to the row's select. The default target is the option that was
// focused when the dialog opened, which belongs to the select's closed menu.
function returnWidenFocus(event: Event) {
  event.preventDefault()
  root.value?.querySelector<HTMLElement>(`[data-test="profile-privacy-row-${widenReturnKey.value}"] [role="combobox"]`)?.focus()
}

const rows = computed(() => PROFILE_PRIVACY_SECTIONS.map(section => ({
  key: section.key,
  label: t(`settings.privacy.sections.${section.key}`),
  description: t(`settings.privacy.sections.${section.key}Description`),
  options: section.audiences.map(value => ({ value, label: t(`settings.privacy.audiences.${value}`) })),
})))

async function load() {
  loading.value = true
  loadFailed.value = false
  try {
    settings.value = await getPrivacySettings()
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  legacyUpdate.value = readLegacyPrivacy()
  // A legacy record with every switch on hid nothing; drop it silently.
  if (!legacyUpdate.value) clearLegacyPrivacy()
  void load()
})

function choose(key: ProfilePrivacySection, value: string) {
  const current = settings.value?.[key]
  const audience = value as ProfileAudience
  if (!current || current === audience || saving.value) return
  // Restoring a wider audience re-exposes history, so confirm it first.
  if (widensAudience(current, audience)) {
    pendingWiden.value = { key, audience }
    widenReturnKey.value = key
    return
  }
  void save(key, { [key]: audience })
}

function confirmWiden() {
  const pending = pendingWiden.value
  pendingWiden.value = null
  if (pending) void save(pending.key, { [pending.key]: pending.audience })
}

async function save(target: ProfilePrivacySection | 'legacy', update: Partial<UserPrivacySettings>) {
  saving.value = target
  failedSection.value = null
  savedSection.value = null
  announcement.value = ''
  try {
    // The select stays on the confirmed server value until this resolves, so a
    // failure needs no rollback and never shows an unsaved choice as saved.
    settings.value = await updatePrivacySettings(update)
    markSaved(target)
    return true
  } catch {
    failedSection.value = target
    return false
  } finally {
    saving.value = null
  }
}

async function applyLegacy() {
  if (!legacyUpdate.value) return
  if (await save('legacy', legacyUpdate.value)) dismissLegacy()
}

function dismissLegacy() {
  clearLegacyPrivacy()
  legacyUpdate.value = null
}

const widenDescription = computed(() => pendingWiden.value
  ? t('settings.privacy.widenDescription', {
    section: t(`settings.privacy.sections.${pendingWiden.value.key}`),
    audience: t(`settings.privacy.audiences.${pendingWiden.value.audience}`),
  })
  : '')
</script>

<template>
  <section ref="root" class="pb-6" aria-labelledby="profile-privacy-title" data-test="profile-privacy-settings">
    <div class="flex flex-wrap items-start justify-between gap-x-4 gap-y-1">
      <div class="min-w-0">
        <h3 id="profile-privacy-title" class="text-sm font-semibold text-base-content">{{ t('settings.privacy.profileTitle') }}</h3>
        <p class="mt-0.5 text-sm text-base-content/60">{{ t('settings.privacy.profileDescription') }}</p>
      </div>
      <a :href="props.profileUrl" class="inline-flex min-h-8 items-center text-xs font-medium text-primary hover:underline hover:underline-offset-2">
        {{ t('settings.privacy.viewProfile') }}
      </a>
    </div>

    <div
      v-if="legacyUpdate && settings"
      class="mt-4 rounded-lg bg-warning/10 p-3"
      data-test="profile-privacy-legacy"
    >
      <p class="text-sm font-semibold text-base-content">{{ t('settings.privacy.legacyTitle') }}</p>
      <p class="mt-1 text-sm leading-relaxed text-base-content/70">{{ t('settings.privacy.legacyDescription') }}</p>
      <p v-if="failedSection === 'legacy'" role="alert" class="mt-2 text-sm text-error">{{ t('settings.privacy.saveFailed') }}</p>
      <div class="mt-3 flex flex-wrap gap-2">
        <button type="button" class="gf-button gf-button-sm gf-button-primary" :disabled="saving !== null" @click="applyLegacy">
          <Loader2 v-if="saving === 'legacy'" class="h-4 w-4 animate-spin" aria-hidden="true" />
          {{ t('settings.privacy.legacyApply') }}
        </button>
        <button type="button" class="gf-button gf-button-sm gf-button-muted" :disabled="saving !== null" @click="dismissLegacy">
          {{ t('settings.privacy.legacyDismiss') }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="mt-3 divide-y divide-line" aria-busy="true" data-test="profile-privacy-loading">
      <div v-for="row in rows" :key="row.key" class="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="space-y-1.5">
          <div class="h-4 w-24 animate-pulse rounded bg-base-300/70" />
          <div class="h-3.5 w-48 max-w-full animate-pulse rounded bg-base-300/50" />
        </div>
        <div class="h-9 w-full animate-pulse rounded-md bg-base-300/50 sm:w-44" />
      </div>
    </div>

    <div v-else-if="loadFailed" class="mt-3 flex flex-wrap items-center gap-3 rounded-lg bg-error/10 px-3 py-2.5" role="alert">
      <p class="text-sm text-error">{{ t('settings.privacy.loadFailed') }}</p>
      <button type="button" class="gf-button gf-button-sm gf-button-muted" @click="load">{{ t('settings.privacy.retry') }}</button>
    </div>

    <div v-else-if="settings" class="mt-3 divide-y divide-line">
      <div
        v-for="row in rows"
        :key="row.key"
        class="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-6"
        :data-test="`profile-privacy-row-${row.key}`"
      >
        <div class="min-w-0">
          <p class="flex items-center gap-1.5 text-sm font-semibold text-base-content">
            {{ row.label }}
            <Loader2 v-if="saving === row.key" class="h-3.5 w-3.5 shrink-0 animate-spin text-base-content/55" aria-hidden="true" />
            <span v-else-if="savedSection === row.key" class="inline-flex items-center gap-1 text-xs font-medium text-success" data-test="profile-privacy-saved">
              <Check class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              {{ t('settings.privacy.saved') }}
            </span>
          </p>
          <p class="mt-0.5 text-sm text-base-content/55">{{ row.description }}</p>
          <p v-if="failedSection === row.key" role="alert" class="mt-1 text-xs text-error">{{ t('settings.privacy.saveFailed') }}</p>
        </div>
        <div class="sm:shrink-0">
          <SiteSelect
            class="w-full sm:w-44"
            :model-value="settings[row.key]"
            :options="row.options"
            :label="t('settings.privacy.audienceLabel', { section: row.label })"
            :disabled="saving !== null"
            @update:model-value="choose(row.key, $event)"
          />
        </div>
      </div>
      <!-- Bookmarks have no audience to choose; listing them keeps that from reading as public. -->
      <div class="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-6" data-test="profile-privacy-row-bookmarks">
        <div class="min-w-0">
          <p class="text-sm font-semibold text-base-content">{{ t('user.tabs.bookmarks') }}</p>
          <p class="mt-0.5 text-sm text-base-content/55">{{ t('settings.privacy.bookmarksDescription') }}</p>
        </div>
        <p class="flex items-center gap-1.5 text-sm text-base-content/60 sm:w-44 sm:shrink-0 sm:justify-between sm:px-3" data-test="profile-privacy-bookmarks-audience">
          <span>{{ t('settings.privacy.audiences.self') }}</span>
          <Lock class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        </p>
      </div>
    </div>

    <p class="sr-only" aria-live="polite" data-test="profile-privacy-announcement">{{ announcement }}</p>

    <p class="mt-3 flex gap-2 text-xs leading-relaxed text-base-content/55">
      <Info class="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      <span>{{ t('settings.privacy.profileScope') }}</span>
    </p>

    <DialogRoot :open="pendingWiden !== null" @update:open="value => { if (!value) pendingWiden = null }">
      <DialogPortal>
        <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40 backdrop-blur-[2px]" />
        <DialogContent
          data-test="profile-privacy-widen"
          class="fixed left-1/2 top-1/2 z-[2101] w-[calc(100%_-_2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-box border border-line bg-base-100 p-5 text-base-content shadow-2xl outline-none"
          @open-auto-focus="focusWidenCancel"
          @close-auto-focus="returnWidenFocus"
        >
          <DialogTitle class="text-base font-semibold leading-7">{{ t('settings.privacy.widenTitle') }}</DialogTitle>
          <DialogDescription class="mt-2 text-sm leading-6 text-base-content/70">
            {{ widenDescription }}
          </DialogDescription>
          <div class="mt-5 flex flex-wrap justify-end gap-2">
            <button ref="widenCancel" type="button" class="gf-button gf-button-sm gf-button-muted" @click="pendingWiden = null">
              {{ t('common.cancel') }}
            </button>
            <button type="button" class="gf-button gf-button-sm gf-button-primary" @click="confirmWiden">
              {{ t('settings.privacy.widenConfirm') }}
            </button>
          </div>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </section>
</template>
