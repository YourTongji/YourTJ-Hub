<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserBadgePayload } from '@gooseforum/client'
import { displayBadges } from '@/runtime/api'
import { badgeClass, badgeIconURL } from '@/site/utils/badge-style'
const props = defineProps<{ badges: UserBadgePayload[]; selected: UserBadgePayload[] }>()
const { t } = useI18n()
const codes = ref(props.selected.map(b => b.code))
const saving = ref(false)
const error = ref('')
const saved = ref(false)
watch(() => props.selected, value => { codes.value = value.map(b => b.code) })
const chosen = computed(() => codes.value.map(code => props.badges.find(b => b.code === code)).filter((b): b is UserBadgePayload => !!b))
function toggle(code: string) {
  if (saving.value) return
  saved.value = false
  if (codes.value.includes(code)) codes.value = codes.value.filter(c => c !== code)
  else if (codes.value.length < 5) codes.value.push(code)
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
  try { await displayBadges([...codes.value]); saved.value = true }
  catch (err) { error.value = err instanceof Error ? err.message : t('api.badgeWearFailed') }
  finally { saving.value = false }
}
</script>
<template>
  <section class="mt-4 space-y-3 border-t border-line pt-3">
    <h3 class="text-sm font-semibold">{{ t('badgeDisplay.title') }}</h3>
    <p class="text-xs text-base-content/60">{{ t('badgeDisplay.hint') }}</p>
    <div class="flex flex-wrap gap-2">
      <label v-for="badge in badges" :key="badge.code" class="inline-flex min-h-11 items-center gap-2 rounded-lg border border-line px-3 text-sm">
        <input type="checkbox" :checked="codes.includes(badge.code)" :disabled="saving || (!codes.includes(badge.code) && codes.length >= 5)" @change="toggle(badge.code)" />
        <span class="inline-flex h-6 w-6 items-center justify-center rounded-full" :class="badgeClass(badge.color, badge.level)"><img :src="badgeIconURL(badge)" alt="" class="h-4 w-4" /></span>
        {{ badge.name }}
      </label>
    </div>
    <ol class="space-y-1">
      <li v-for="(badge, index) in chosen" :key="badge.code" class="flex items-center gap-2 text-sm">
        <span class="min-w-0 flex-1 break-words">{{ index + 1 }}. {{ badge.name }}</span>
        <button type="button" class="gf-button min-h-11" :disabled="saving || index === 0" :aria-label="`${t('badgeDisplay.up')} ${badge.name}`" @click="move(index, -1)">↑</button>
        <button type="button" class="gf-button min-h-11" :disabled="saving || index === chosen.length - 1" :aria-label="`${t('badgeDisplay.down')} ${badge.name}`" @click="move(index, 1)">↓</button>
      </li>
    </ol>
    <button type="button" class="gf-button gf-button-primary" :disabled="saving" @click="save">{{ t('badgeDisplay.save') }}</button>
    <p v-if="error" role="alert" class="text-sm text-error">{{ error }}</p>
    <p v-if="saved" role="status" class="text-sm text-success">{{ t('settings.status.badgeSaved') }}</p>
  </section>
</template>
