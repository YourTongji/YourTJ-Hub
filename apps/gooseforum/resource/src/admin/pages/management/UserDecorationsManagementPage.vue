<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CheckCircle2, Loader2, Package, RefreshCw, Search, UserRound } from '@lucide/vue'
import type { DecorationState } from '@gooseforum/client'
import { getUserDecorations, changeUserDecoration } from '@/admin/runtime/api'
import { BasicPage } from '@/admin/components/global-layout'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import { Textarea } from '@/admin/components/ui/textarea'
import { Badge } from '@/admin/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/admin/components/ui/card'
import AdminSelect from '@/admin/components/AdminSelect.vue'
import AdminConfirmDialog from '@/admin/components/AdminConfirmDialog.vue'
import DecorationStylePreview from '@/admin/components/DecorationStylePreview.vue'
import ManagementTable from './ManagementTable.vue'

const { t, locale } = useI18n()
const userId = ref<string | number>('')
const loadedUserId = ref(0)
const state = ref<DecorationState>()
const code = ref('')
const reason = ref('')
const error = ref('')
const success = ref('')
const loading = ref(false)
const saving = ref(false)
const confirmingRevoke = ref(false)
const busy = computed(() => loading.value || saving.value)
const validUserId = computed(() => Number.isSafeInteger(Number(userId.value)) && Number(userId.value) > 0)
const currentAccount = computed(() => !!state.value && Number(userId.value) === loadedUserId.value)
const selected = computed(() => state.value?.catalog.find(item => item.code === code.value))
const owned = computed(() => !!state.value?.ownedCodes.includes(code.value))
const ready = computed(() => currentAccount.value && !!selected.value && !!reason.value.trim() && reason.value.trim().length <= 500 && !busy.value)
const canGrant = computed(() => ready.value && !owned.value && selected.value?.isEnabled)
const canRevoke = computed(() => ready.value && owned.value)
const options = computed(() => [{ value: '', label: t('decoration.grantsAdmin.selectItem') }, ...(state.value?.catalog ?? []).map(item => ({
  value: item.code,
  label: `${item.name} · ${item.code}${!item.isEnabled ? ` · ${t('decoration.disabled')}` : ''}${state.value?.ownedCodes.includes(item.code) ? ` · ${t('decoration.owned')}` : ''}`,
}))])
const auditColumns = computed(() => [t('decoration.time'), t('decoration.item'), t('decoration.action'), t('decoration.source'), t('decoration.actor'), t('decoration.reason')])

// Keep the operation key across ambiguous network failures so retry cannot duplicate a grant.
let pending: { action: 'grant' | 'revoke'; userId: number; code: string; reason: string; requestKey: string } | undefined
watch([userId, code, reason], () => { confirmingRevoke.value = false })

async function load() {
  if (!validUserId.value || busy.value) return
  const target = Number(userId.value)
  loading.value = true
  error.value = ''
  success.value = ''
  if (target !== loadedUserId.value) {
    state.value = undefined
    loadedUserId.value = 0
    code.value = ''
    reason.value = ''
  }
  try {
    state.value = await getUserDecorations(target)
    loadedUserId.value = target
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('decoration.loadFailed')
  } finally { loading.value = false }
}

async function change(action: 'grant' | 'revoke') {
  if (action === 'grant' ? !canGrant.value : !canRevoke.value) return
  const body = { userId: loadedUserId.value, code: code.value, reason: reason.value.trim(), requestKey: '' }
  const name = selected.value!.name
  if (!pending || pending.action !== action || pending.userId !== body.userId || pending.code !== body.code || pending.reason !== body.reason) {
    pending = { action, ...body, requestKey: crypto.randomUUID() }
  }
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    const previousAudits = state.value?.audits
    const updated = await changeUserDecoration(action, pending)
    state.value = { ...updated, audits: updated.audits ?? previousAudits }
    pending = undefined
    reason.value = ''
    confirmingRevoke.value = false
    success.value = action === 'grant'
      ? t('decoration.grantsAdmin.grantSuccess', { name, id: body.userId })
      : t('decoration.grantsAdmin.revokeSuccess', { name, id: body.userId })
    // The write already succeeded; a failed audit refresh must never repeat it.
    try { state.value = await getUserDecorations(body.userId) }
    catch (e) { error.value = e instanceof Error ? e.message : t('decoration.loadFailed') }
  } catch (e) {
    confirmingRevoke.value = false
    error.value = e instanceof Error ? e.message : t('decoration.saveFailed')
  } finally { saving.value = false }
}

function itemName(value: string) { return state.value?.catalog.find(item => item.code === value)?.name || value || t('decoration.none') }
function isEquipped(value: string) { return state.value?.equipped.avatarFrame === value || state.value?.equipped.nicknameStyle === value }
function isEnabled(value: string) { return !!state.value?.catalog.find(item => item.code === value)?.isEnabled }
function actionText(value: string) {
  if (value === 'grant') return t('decoration.grant')
  if (value === 'revoke') return t('decoration.revoke')
  if (value === 'equip') return t('decoration.grantsAdmin.equipAction')
  if (value === 'definition') return t('decoration.grantsAdmin.definitionAction')
  return value
}
function sourceText(value: string) {
  if (value === 'manual') return t('decoration.grantsAdmin.manualSource')
  if (value === 'self') return t('decoration.grantsAdmin.selfSource')
  return value
}
function formatTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'medium' }).format(date)
}
</script>

<template>
  <BasicPage :title="t('decoration.userGrants')" :description="t('decoration.grantsAdmin.description')" sticky>
    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2 text-base"><UserRound class="size-4 text-muted-foreground" />{{ t('decoration.grantsAdmin.accountTitle') }}</CardTitle>
        <CardDescription>{{ t('decoration.grantsAdmin.accountHint') }}</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="flex flex-col gap-3 sm:flex-row sm:items-end" @submit.prevent="load">
          <div class="grid w-full gap-2 sm:max-w-xs">
            <label for="decoration-user-id" class="text-sm font-medium">{{ t('decoration.userId') }}</label>
            <Input id="decoration-user-id" v-model="userId" type="number" min="1" step="1" required :disabled="busy" :placeholder="t('decoration.grantsAdmin.userIdPlaceholder')" />
          </div>
          <Button type="submit" :disabled="busy || !validUserId"><Loader2 v-if="loading" class="size-4 animate-spin" /><Search v-else class="size-4" />{{ loading ? t('decoration.loading') : t('decoration.load') }}</Button>
          <Button v-if="state && currentAccount" variant="outline" type="button" :disabled="busy" @click="load"><RefreshCw class="size-4" />{{ t('decoration.grantsAdmin.refresh') }}</Button>
        </form>
      </CardContent>
    </Card>

    <div v-if="success" role="status" class="mt-4 flex items-center gap-2 rounded-lg border border-primary/25 bg-primary/5 p-3 text-sm"><CheckCircle2 class="size-4 shrink-0 text-primary" />{{ success }}</div>
    <div v-if="error" role="alert" class="mt-4 flex flex-wrap items-center gap-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
      <span>{{ error }}</span>
      <span v-if="success" class="text-xs">{{ t('decoration.grantsAdmin.refreshFailed') }}</span>
      <span v-else-if="pending" class="text-xs">{{ t('decoration.grantsAdmin.retryHint') }}</span>
      <Button v-if="!state || success" variant="outline" size="sm" type="button" :disabled="busy || !validUserId" @click="load">{{ t('decoration.retry') }}</Button>
    </div>
    <div v-if="!state" class="mt-6 grid place-items-center gap-3 rounded-lg border border-dashed px-6 py-12 text-center text-sm text-muted-foreground">
      <Loader2 v-if="loading" class="size-7 animate-spin" /><UserRound v-else class="size-7" />
      <p>{{ loading ? t('decoration.loading') : t('decoration.grantsAdmin.noAccount') }}</p>
    </div>

    <section v-if="state" class="mt-6 space-y-6">
      <div class="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-muted/20 px-4 py-3">
        <div class="flex flex-wrap items-center gap-2"><UserRound class="size-4 text-muted-foreground" /><h2 class="text-sm font-semibold">{{ t('decoration.grantsAdmin.loadedAccount', { id: loadedUserId }) }}</h2><Badge variant="secondary">{{ t('decoration.grantsAdmin.inventoryCount', { count: state.ownedCodes.length }) }}</Badge></div>
        <p v-if="!currentAccount" role="status" class="text-sm text-destructive">{{ t('decoration.grantsAdmin.accountChanged') }}</p>
      </div>

      <div class="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(340px,0.85fr)]">
        <Card>
          <CardHeader><CardTitle class="flex items-center gap-2 text-base"><Package class="size-4 text-muted-foreground" />{{ t('decoration.grantsAdmin.inventoryTitle') }}</CardTitle></CardHeader>
          <CardContent class="space-y-5">
            <dl class="grid gap-3 sm:grid-cols-2">
              <div v-for="kind in (['avatarFrame', 'nicknameStyle'] as const)" :key="kind" class="rounded-lg border bg-muted/20 p-3">
                <dt class="text-xs text-muted-foreground">{{ kind === 'avatarFrame' ? t('decoration.avatarFrame') : t('decoration.nicknameStyle') }}</dt>
                <dd class="mt-1 flex flex-wrap items-center gap-2 text-sm font-medium">{{ itemName(state.equipped[kind]) }}<Badge v-if="state.equipped[kind]" variant="outline">{{ t('decoration.grantsAdmin.equipped') }}</Badge></dd>
                <p v-if="state.equipped[kind] && !isEnabled(state.equipped[kind])" class="mt-1 text-xs text-muted-foreground">{{ t('decoration.disabled') }}</p>
              </div>
            </dl>
            <p v-if="state.ownedCodes.length === 0" class="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">{{ t('decoration.grantsAdmin.inventoryEmpty') }}</p>
            <div v-else class="grid gap-2 sm:grid-cols-2">
              <button v-for="ownedCode in state.ownedCodes" :key="ownedCode" type="button" :disabled="busy || !currentAccount" :aria-pressed="code === ownedCode" class="grid gap-2 rounded-lg border p-3 text-left transition-colors hover:bg-muted/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-60" :class="code === ownedCode ? 'border-primary bg-primary/5' : 'border-border'" @click="code = ownedCode">
                <span class="text-sm font-medium">{{ itemName(ownedCode) }}</span><span class="break-all font-mono text-xs text-muted-foreground">{{ ownedCode }}</span>
                <span class="flex flex-wrap gap-1"><Badge variant="secondary">{{ t('decoration.owned') }}</Badge><Badge v-if="isEquipped(ownedCode)" variant="outline">{{ t('decoration.grantsAdmin.equipped') }}</Badge><Badge v-if="!isEnabled(ownedCode)" variant="outline">{{ t('decoration.disabled') }}</Badge></span>
              </button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle class="text-base">{{ t('decoration.grantsAdmin.operationTitle') }}</CardTitle><CardDescription>{{ t('decoration.grantsAdmin.operationHint') }}</CardDescription></CardHeader>
          <CardContent class="space-y-4">
            <div class="grid gap-2"><label for="decoration-item" class="text-sm font-medium">{{ t('decoration.item') }}</label><AdminSelect id="decoration-item" v-model="code" :options="options" :disabled="busy || !currentAccount || state.catalog.length === 0" class="w-full" /></div>
            <p v-if="state.catalog.length === 0" class="text-sm text-muted-foreground">{{ t('decoration.grantsAdmin.catalogEmpty') }}</p>
            <div v-if="selected" class="space-y-3">
              <div class="flex flex-wrap items-center gap-2"><Badge variant="outline">{{ selected.kind === 'avatarFrame' ? t('decoration.avatarFrame') : t('decoration.nicknameStyle') }}</Badge><Badge variant="secondary">{{ owned ? t('decoration.owned') : t('decoration.grantsAdmin.notOwned') }}</Badge><Badge v-if="!selected.isEnabled" variant="outline">{{ t('decoration.disabled') }}</Badge><Badge v-if="isEquipped(selected.code)" variant="outline">{{ t('decoration.grantsAdmin.equipped') }}</Badge></div>
              <DecorationStylePreview :style="selected" compact />
              <p v-if="!selected.isEnabled" class="text-xs text-muted-foreground">{{ t('decoration.grantsAdmin.disabledHint') }}</p>
            </div>
            <div class="grid gap-2"><label for="decoration-reason" class="text-sm font-medium">{{ t('decoration.reason') }}</label><Textarea id="decoration-reason" v-model="reason" maxlength="500" rows="3" :disabled="busy || !currentAccount" :placeholder="t('decoration.grantsAdmin.reasonPlaceholder')" /><p class="text-right text-xs tabular-nums text-muted-foreground">{{ reason.length }} / 500</p></div>
            <div class="flex flex-wrap gap-2 border-t pt-4"><Button data-testid="grant" type="button" :disabled="!canGrant" @click="change('grant')"><Loader2 v-if="saving" class="size-4 animate-spin" />{{ t('decoration.grant') }}</Button><Button data-testid="revoke" variant="destructive" type="button" :disabled="!canRevoke" @click="confirmingRevoke = true">{{ t('decoration.revoke') }}</Button></div>
          </CardContent>
        </Card>
      </div>

      <div class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-2"><h2 class="text-base font-semibold">{{ t('decoration.audit') }}</h2><p class="text-xs text-muted-foreground">{{ t('decoration.grantsAdmin.auditHint') }}</p></div>
        <ManagementTable :columns="auditColumns" :show-pagination="false" :loading="loading">
          <tr v-if="!state.audits?.length"><td :colspan="auditColumns.length" class="h-24 px-4 text-center text-muted-foreground">{{ t('decoration.grantsAdmin.auditEmpty') }}</td></tr>
          <tr v-for="audit in state.audits" :key="audit.id" class="hover:bg-muted/30">
            <td class="whitespace-nowrap px-4 py-3 text-xs text-muted-foreground"><time :datetime="audit.createdAt">{{ formatTime(audit.createdAt) }}</time></td>
            <td class="px-4 py-3"><span class="block font-medium">{{ itemName(audit.code) }}</span><span v-if="audit.code" class="font-mono text-xs text-muted-foreground">{{ audit.code }}</span></td>
            <td class="px-4 py-3"><Badge :variant="audit.action === 'revoke' ? 'destructive' : 'secondary'">{{ actionText(audit.action) }}</Badge></td>
            <td class="px-4 py-3">{{ sourceText(audit.source) }}</td><td class="px-4 py-3 font-mono text-xs">#{{ audit.actorId }}</td><td class="max-w-sm break-words px-4 py-3 text-muted-foreground">{{ audit.reason }}</td>
          </tr>
        </ManagementTable>
      </div>
    </section>

    <AdminConfirmDialog :open="confirmingRevoke" :title="t('decoration.grantsAdmin.revokeTitle')" :description="t('decoration.grantsAdmin.revokeDescription', { name: selected?.name || '', id: loadedUserId, reason: reason.trim() })" :confirm-text="t('decoration.revoke')" :loading="saving" @update:open="(open) => !saving && (confirmingRevoke = open)" @confirm="change('revoke')" />
  </BasicPage>
</template>
