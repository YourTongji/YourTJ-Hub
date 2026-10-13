<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DecorationStyle } from '@gooseforum/client'
import { getDecorations, saveDecoration } from '@/admin/runtime/api'
import { BasicPage } from '@/admin/components/global-layout'
import UserAvatar from '@/site/components/UserAvatar.vue'
import UserNickname from '@/site/components/UserNickname.vue'
const { t } = useI18n()
const items = ref<DecorationStyle[]>([]), error = ref(''), saving = ref(false), loaded = ref(false), saved = ref(false)
const fresh = (): DecorationStyle => ({ code: '', name: '', kind: 'avatarFrame', lightColors: [], darkColors: [], isEnabled: true, sortOrder: 100 })
const form = reactive(fresh()), light = ref(''), dark = ref('')
const preview = computed(() => ({ ...form, lightColors: light.value.split(',').map(color => color.trim()), darkColors: dark.value.split(',').map(color => color.trim()) }))
const editing = ref(false)
function edit(item?: DecorationStyle) { Object.assign(form, item ?? fresh()); form.assetUrl = item?.assetUrl ?? ''; editing.value = !!item; light.value = form.lightColors.join(', '); dark.value = form.darkColors.join(', '); saved.value = false }
async function load() { try { items.value = await getDecorations(); loaded.value = true } catch (e) { error.value = e instanceof Error ? e.message : t('decoration.loadFailed') } }
async function save() {
  error.value = ''; saving.value = true; saved.value = false
  try { const colors = (s: string) => s.split(',').map(s => s.trim()).filter(Boolean); form.lightColors = colors(light.value); form.darkColors = colors(dark.value); await saveDecoration({ ...form }); await load(); saved.value = true }
  catch (e) { error.value = e instanceof Error ? e.message : t('decoration.saveFailed') } finally { saving.value = false }
}
onMounted(load)
</script>
<template><BasicPage :title="t('decoration.title')" :description="t('decoration.adminHint')">
<div class="grid gap-6 lg:grid-cols-2">
<section class="space-y-3"><button type="button" class="gf-button" @click="edit()">{{ t('decoration.new') }}</button>
<button v-for="item in items" :key="item.code" type="button" class="flex w-full items-center gap-4 rounded-lg border p-4 text-left" @click="edit(item)">
<UserAvatar v-if="item.kind === 'avatarFrame'" src="/static/pic/default-avatar.webp" alt="" :frame="item" class="h-10 w-10 rounded-full" img-class="rounded-full" /><UserNickname v-else :style-definition="item">{{ item.name }}</UserNickname>
<span>{{ item.name }} · {{ item.code }} <span v-if="!item.isEnabled">({{ t('decoration.disabled') }})</span></span></button>
<button v-if="!loaded" type="button" class="gf-button" @click="load">{{ t('decoration.retry') }}</button></section>
<form class="space-y-3" @submit.prevent="save">
<label class="block">{{ t('decoration.code') }}<input v-model="form.code" required pattern="[a-z][a-z0-9_-]{0,63}" maxlength="64" :disabled="editing || saving" class="gf-input" /></label>
<label class="block">{{ t('decoration.name') }}<input v-model="form.name" required maxlength="80" :disabled="saving" class="gf-input" /></label>
<label class="block">{{ t('decoration.kind') }}<select v-model="form.kind" :disabled="editing || saving" class="gf-input"><option value="avatarFrame">{{ t('decoration.avatarFrame') }}</option><option value="nicknameStyle">{{ t('decoration.nicknameStyle') }}</option></select></label>
<label class="block">{{ t('decoration.lightColors') }}<input v-model="light" required :disabled="saving" class="gf-input" /></label>
<label class="block">{{ t('decoration.darkColors') }}<input v-model="dark" required :disabled="saving" class="gf-input" /></label>
<p class="text-sm">{{ t('decoration.colorsHint') }}</p>
<label v-if="form.kind === 'avatarFrame'" class="block">{{ t('decoration.assetUrl') }}<input v-model="form.assetUrl" maxlength="160" :disabled="saving" class="gf-input" /><span class="text-xs text-base-content/60">{{ t('decoration.assetHint') }}</span></label>
<div class="flex min-h-14 items-center gap-3 rounded-lg border bg-base-100 p-3"><UserAvatar v-if="form.kind === 'avatarFrame'" src="/static/pic/default-avatar.webp" alt="" :frame="preview" class="h-10 w-10 rounded-full" img-class="rounded-full" /><UserNickname v-else :style-definition="preview" class="font-semibold">{{ form.name || form.code }}</UserNickname></div>
<label class="block">{{ t('decoration.sortOrder') }}<input v-model.number="form.sortOrder" type="number" :disabled="saving" class="gf-input" /></label>
<label class="flex gap-2"><input v-model="form.isEnabled" type="checkbox" :disabled="saving" />{{ t('decoration.enabled') }}</label>
<button type="submit" :disabled="saving" class="gf-button gf-button-primary">{{ t('decoration.save') }}</button></form>
</div><p v-if="error" role="alert" class="text-sm text-error">{{ error }}</p><p v-if="saved" role="status" class="text-sm text-success">{{ t('decoration.saved') }}</p>
</BasicPage></template>
