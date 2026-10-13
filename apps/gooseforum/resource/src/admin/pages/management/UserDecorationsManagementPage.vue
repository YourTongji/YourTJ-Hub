<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DecorationState } from '@gooseforum/client'
import { getUserDecorations, changeUserDecoration } from '@/admin/runtime/api'
import { BasicPage } from '@/admin/components/global-layout'
const { t } = useI18n()
const userId = ref<number>(), state = ref<DecorationState>(), code = ref(''), reason = ref(''), error = ref(''), busy = ref(false)
let loadedUserId = 0
// Keep the operation key across ambiguous network failures so retry cannot duplicate a grant.
let pending: { action: 'grant' | 'revoke'; userId: number; code: string; reason: string; requestKey: string } | undefined
async function load() { if (!userId.value || busy.value) return; busy.value = true; error.value = ''; state.value = undefined; try { state.value = await getUserDecorations(userId.value); loadedUserId = userId.value; code.value = ''; pending = undefined } catch (e) { error.value = e instanceof Error ? e.message : t('decoration.loadFailed') } finally { busy.value = false } }
async function change(action: 'grant' | 'revoke') {
  if (!loadedUserId || !code.value || !reason.value.trim() || busy.value) return
  busy.value = true; error.value = ''
  const body = { userId: loadedUserId, code: code.value, reason: reason.value.trim(), requestKey: '' }
  if (!pending || pending.action !== action || pending.userId !== body.userId || pending.code !== body.code || pending.reason !== body.reason) pending = { action, ...body, requestKey: crypto.randomUUID() }
  try {
    const previousAudits = state.value?.audits
    const updated = await changeUserDecoration(action, pending)
    state.value = { ...updated, audits: updated.audits ?? previousAudits }
    pending = undefined; reason.value = ''
    try { state.value = await getUserDecorations(loadedUserId) }
    catch (e) { error.value = e instanceof Error ? e.message : t('decoration.loadFailed') }
  }
  catch (e) { error.value = e instanceof Error ? e.message : t('decoration.saveFailed') } finally { busy.value = false }
}
</script>
<template><BasicPage :title="t('decoration.userGrants')" :description="t('decoration.grantHint')">
<form class="flex gap-3" @submit.prevent="load"><label>{{ t('decoration.userId') }}<input v-model.number="userId" type="number" min="1" required :disabled="busy" class="gf-input" /></label><button type="submit" class="gf-button" :disabled="busy">{{ t('decoration.load') }}</button></form>
<section v-if="state" class="mt-5 space-y-3"><p>{{ t('decoration.userId') }}: {{ loadedUserId }}</p><label class="block">{{ t('decoration.item') }}<select v-model="code" :disabled="busy" class="gf-input"><option value="">—</option><option v-for="item in state.catalog" :key="item.code" :value="item.code">{{ item.name }} · {{ item.code }} {{ state.ownedCodes.includes(item.code) ? t('decoration.owned') : '' }} {{ !item.isEnabled ? t('decoration.disabled') : '' }}</option></select></label>
<label class="block">{{ t('decoration.reason') }}<textarea v-model="reason" required maxlength="500" :disabled="busy" class="gf-input" /></label>
<div class="flex gap-3"><button type="button" class="gf-button gf-button-primary" :disabled="busy || !code || !reason.trim() || state.ownedCodes.includes(code)" @click="change('grant')">{{ t('decoration.grant') }}</button><button type="button" class="gf-button" :disabled="busy || !code || !reason.trim() || !state.ownedCodes.includes(code)" @click="change('revoke')">{{ t('decoration.revoke') }}</button></div>
<div v-if="state.audits?.length" class="overflow-x-auto"><h2 class="font-semibold">{{ t('decoration.audit') }}</h2><table class="mt-2 w-full text-left text-sm"><thead><tr><th>{{ t('decoration.time') }}</th><th>{{ t('decoration.item') }}</th><th>{{ t('decoration.action') }}</th><th>{{ t('decoration.source') }}</th><th>{{ t('decoration.actor') }}</th><th>{{ t('decoration.reason') }}</th></tr></thead><tbody><tr v-for="audit in state.audits" :key="audit.id" class="border-t"><td class="p-2">{{ audit.createdAt }}</td><td class="p-2">{{ audit.code }}</td><td class="p-2">{{ audit.action }}</td><td class="p-2">{{ audit.source }}</td><td class="p-2">{{ audit.actorId }}</td><td class="max-w-md break-words p-2">{{ audit.reason }}</td></tr></tbody></table></div></section><p v-if="error" role="alert" class="text-sm text-error">{{ error }}</p>
</BasicPage></template>
