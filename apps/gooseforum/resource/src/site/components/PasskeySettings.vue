<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { KeyRound, Loader2, Pencil, Plus, Trash2 } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import {
  listPasskeyCredentials,
  getPasskeyStatus,
  registerPasskey,
  renamePasskey,
  revokePasskey,
  type PasskeyCredentialSummary,
} from '@/runtime/api'

const props = defineProps<{ totpEnabled: boolean }>()
const { t } = useI18n()
const credentials = ref<PasskeyCredentialSummary[]>([])
const enabled = ref(false)
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const password = ref('')
const totpCode = ref('')
const name = ref('')
const editing = ref('')
const editName = ref('')

async function refresh() {
  loading.value = true
  try {
    const status = await getPasskeyStatus()
    enabled.value = status.enabled
    if (!enabled.value) {
      credentials.value = []
      return
    }
    credentials.value = await listPasskeyCredentials()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.security.passkeyFailed')
  } finally {
    loading.value = false
  }
}

onMounted(() => { void refresh() })

function clearProofInputs() {
  password.value = ''
  totpCode.value = ''
}

async function add() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    const created = await registerPasskey(password.value, totpCode.value, name.value)
    credentials.value = [...credentials.value, created]
    name.value = ''
    clearProofInputs()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.security.passkeyFailed')
  } finally {
    busy.value = false
  }
}

async function saveName(id: string) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await renamePasskey(password.value, totpCode.value, id, editName.value)
    credentials.value = credentials.value.map((item) => item.id === id ? { ...item, name: editName.value.trim() } : item)
    editing.value = ''
    clearProofInputs()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.security.passkeyFailed')
  } finally {
    busy.value = false
  }
}

async function remove(id: string) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await revokePasskey(password.value, totpCode.value, id)
    credentials.value = credentials.value.filter((item) => item.id !== id)
    editing.value = ''
    clearProofInputs()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : t('settings.security.passkeyFailed')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div v-if="enabled || loading" class="space-y-3 border-b border-line p-4">
    <div class="rounded-lg border border-line bg-base-100 p-4">
      <div class="flex items-start justify-between gap-4">
        <div class="flex min-w-0 gap-3">
          <KeyRound class="mt-0.5 h-5 w-5 shrink-0 text-primary" />
          <div>
            <h3 class="font-semibold text-base-content">{{ t('settings.security.passkeyTitle') }}</h3>
            <p class="mt-1 text-sm text-base-content/55">{{ t('settings.security.passkeyDescription') }}</p>
          </div>
        </div>
        <button type="button" class="gf-button gf-button-md gf-button-primary shrink-0" :disabled="busy" @click="refresh">
          {{ t('settings.security.refresh') }}
        </button>
      </div>
      <p v-if="error" role="alert" class="mt-3 text-sm text-error">{{ error }}</p>
      <div v-if="loading" class="mt-4 flex items-center gap-2 text-sm text-base-content/55">
        <Loader2 class="h-4 w-4 animate-spin" /> {{ t('settings.security.loading') }}
      </div>
      <div v-else class="mt-4 space-y-3">
        <p v-if="credentials.length === 0" class="text-sm text-base-content/55">{{ t('settings.security.passkeyEmpty') }}</p>
        <div v-for="credential in credentials" :key="credential.id" class="rounded-md border border-line p-3">
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <p class="truncate font-medium text-base-content">{{ credential.name }}</p>
              <p class="mt-1 text-xs text-base-content/45">
                {{ t('settings.security.passkeyCreated') }} · {{ new Date(credential.createdAt).toLocaleDateString() }}
                · {{ t('settings.security.passkeyLastUsed') }} {{ credential.lastUsedAt ? new Date(credential.lastUsedAt).toLocaleDateString() : t('settings.security.passkeyNeverUsed') }}
                · {{ credential.backupEligible ? t('settings.security.passkeySyncCapable') : t('settings.security.passkeyDeviceBound') }}<template v-if="credential.backupEligible && credential.backupState"> · {{ t('settings.security.passkeyBackedUp') }}</template>
              </p>
            </div>
            <button type="button" class="gf-button gf-button-sm gf-button-muted" @click="editing = editing === credential.id ? '' : credential.id; editName = credential.name">
              <Pencil class="h-3.5 w-3.5" /> {{ t('settings.security.passkeyManage') }}
            </button>
          </div>
          <div v-if="editing === credential.id" class="mt-3 space-y-2 border-t border-line pt-3">
            <input v-model="editName" class="gf-input" maxlength="80" :placeholder="t('settings.security.passkeyName')" />
            <input v-model="password" type="password" class="gf-input" autocomplete="current-password" :placeholder="t('settings.security.passkeyPassword')" />
            <input v-if="props.totpEnabled" v-model="totpCode" class="gf-input" inputmode="numeric" autocomplete="one-time-code" :placeholder="t('settings.security.totpCodePlaceholder')" />
            <div class="flex flex-wrap justify-end gap-2">
              <button type="button" class="gf-button gf-button-sm gf-button-muted" :disabled="busy" @click="saveName(credential.id)">{{ t('common.save') }}</button>
              <button type="button" class="gf-button gf-button-sm gf-button-error" :disabled="busy" @click="remove(credential.id)"><Trash2 class="h-3.5 w-3.5" /> {{ t('settings.security.revoke') }}</button>
            </div>
          </div>
        </div>
        <div class="border-t border-line pt-3">
          <p class="mb-3 text-xs text-base-content/45">{{ t('settings.security.passkeyRevokedHint') }}</p>
          <h4 class="text-sm font-semibold text-base-content">{{ t('settings.security.passkeyAdd') }}</h4>
          <div class="mt-2 space-y-2">
            <input v-model="name" class="gf-input" maxlength="80" :placeholder="t('settings.security.passkeyName')" />
            <input v-model="password" type="password" class="gf-input" autocomplete="current-password" :placeholder="t('settings.security.passkeyPassword')" />
            <input v-if="props.totpEnabled" v-model="totpCode" class="gf-input" inputmode="numeric" autocomplete="one-time-code" :placeholder="t('settings.security.totpCodePlaceholder')" />
            <div class="flex justify-end">
              <button type="button" class="gf-button gf-button-md gf-button-primary" :disabled="busy || !password || (props.totpEnabled && !totpCode)" @click="add">
                <Loader2 v-if="busy" class="h-4 w-4 animate-spin" /><Plus v-else class="h-4 w-4" />
                {{ t('settings.security.passkeyAdd') }}
              </button>
            </div>
          </div>
          <p class="mt-2 text-xs text-base-content/45">{{ t('settings.security.passkeyReauthHint') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
