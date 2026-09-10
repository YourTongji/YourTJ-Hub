<script setup lang="ts">
import { adminText } from '@/admin/runtime/i18n-text'

import { computed, onMounted, reactive, ref } from 'vue'
import { Bot, Copy, KeyRound, Pencil, Plus, RefreshCw, ShieldOff } from '@lucide/vue'
import AdminActionButton from '@/admin/components/AdminActionButton.vue'
import AdminConfirmDialog from '@/admin/components/AdminConfirmDialog.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/admin/components/ui/dialog'
import { Input } from '@/admin/components/ui/input'
import { Switch } from '@/admin/components/ui/switch'
import { createAgent, disableAgent, getAgentList, rotateAgentToken, updateAgent } from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type { AdminAgent, AdminPayload, ManageHomeProps } from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const agents = ref<AdminAgent[]>([])
const editing = ref<AdminAgent | null>(null)
const creating = ref(false)
const rotating = ref<AdminAgent | null>(null)
const rotatingToken = ref('')
const disabling = ref<AdminAgent | null>(null)
const copied = ref(false)
const form = reactive({ username: '', nickname: '', webhookEndpoint: '', enabled: true })

const enabledCount = computed(() => agents.value.filter(item => item.enabled === 1).length)
const disabledCount = computed(() => agents.value.filter(item => item.enabled !== 1).length)

async function loadAgents() {
  loading.value = true
  error.value = ''
  try {
    agents.value = await getAgentList()
  } catch (err) {
    error.value = err instanceof Error ? err.message : adminText('k00k2')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { username: '', nickname: '', webhookEndpoint: '', enabled: true })
  copied.value = false
  creating.value = true
}

function openEdit(agent: AdminAgent) {
  Object.assign(form, {
    username: agent.username,
    nickname: agent.nickname,
    webhookEndpoint: agent.webhookEndpoint,
    enabled: agent.enabled === 1,
  })
  editing.value = agent
}

async function submitCreate() {
  if (!form.username.trim()) {
    adminToast.warning(adminText('k00k9'))
    return
  }
  saving.value = true
  try {
    const result = await createAgent({
      username: form.username.trim(),
      nickname: form.nickname.trim(),
      webhookEndpoint: form.webhookEndpoint.trim(),
    })
    creating.value = false
    await loadAgents()
    copied.value = false
    rotatingToken.value = result.token
    rotating.value = result.agent
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k00k3'))
  } finally {
    saving.value = false
  }
}

async function submitUpdate() {
  if (!editing.value) return
  saving.value = true
  try {
    await updateAgent({
      agentId: editing.value.agentId,
      nickname: form.nickname.trim(),
      webhookEndpoint: form.webhookEndpoint.trim(),
      enabled: form.enabled ? 1 : 0,
    })
    editing.value = null
    await loadAgents()
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k00k4'))
  } finally {
    saving.value = false
  }
}

async function confirmRotate() {
  if (!rotating.value || saving.value) return
  saving.value = true
  copied.value = false
  try {
    const result = await rotateAgentToken(rotating.value.agentId)
    rotatingToken.value = result.token
    await loadAgents()
    adminToast.success(adminText('k00l1'))
  } catch (err) {
    adminToast.error(err, adminText('k00k5'))
  } finally {
    saving.value = false
  }
}

async function copyToken() {
  if (!rotatingToken.value) return
  try {
    await navigator.clipboard.writeText(rotatingToken.value)
    copied.value = true
  } catch {
    adminToast.warning(adminText('k00l2'))
  }
}

async function confirmDisable() {
  if (!disabling.value) return
  saving.value = true
  try {
    await disableAgent(disabling.value.agentId)
    disabling.value = null
    await loadAgents()
    adminToast.success(adminText('k00l3'))
  } catch (err) {
    adminToast.error(err, adminText('k00k6'))
  } finally {
    saving.value = false
  }
}

function formatTime(millis?: number | null) {
  if (!millis) return '—'
  return new Date(millis).toLocaleString()
}

onMounted(() => {
  void loadAgents()
})
</script>

<template>
  <BasicPage :title="adminText('k00k0')" :description="adminText('k00k1')" sticky>
    <template #actions>
      <div class="flex items-center gap-2">
        <Button variant="outline" type="button" @click="loadAgents">
          <RefreshCw class="size-4" />
          {{ adminText('k004q') }}
        </Button>
        <Button type="button" @click="openCreate">
          <Plus class="size-4" />
          {{ adminText('k00k8') }}
        </Button>
      </div>
    </template>

    <div class="mb-3 flex flex-wrap gap-2 text-sm text-muted-foreground">
      <Badge variant="secondary">
        <Bot class="size-3.5" />
        {{ adminText('k00l4') }} {{ enabledCount }}
      </Badge>
      <Badge variant="outline">{{ adminText('k00l5') }} {{ disabledCount }}</Badge>
      <span v-if="loading">{{ adminText('k0046') }}</span>
    </div>

    <div v-if="error" class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">{{ error }}</div>
    <div v-else class="overflow-hidden rounded-lg border bg-card">
      <div class="overflow-x-auto">
        <table class="w-full min-w-[760px] text-sm">
          <thead class="border-b bg-muted/45 text-xs font-medium text-muted-foreground">
            <tr>
              <th class="h-11 px-4 text-left align-middle">{{ adminText('k00l6') }}</th>
              <th class="h-11 px-4 text-left align-middle">{{ adminText('k00i6') }}</th>
              <th class="h-11 px-4 text-left align-middle">{{ adminText('k00l7') }}</th>
              <th class="h-11 px-4 text-left align-middle">{{ adminText('k00kf') }}</th>
              <th class="h-11 px-4 text-left align-middle">{{ adminText('k00kc') }}</th>
              <th class="h-11 px-4 text-left align-middle">{{ adminText('k00kg') }}</th>
              <th class="h-11 px-4 text-left align-middle">{{ adminText('k00i7') }}</th>
              <th class="h-11 px-4 text-right align-middle">{{ adminText('k00ki') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-if="loading">
              <td colspan="8" class="h-28 px-4 text-center text-muted-foreground">{{ adminText('k0046') }}</td>
            </tr>
            <tr v-else-if="agents.length === 0">
              <td colspan="8" class="h-28 px-4 text-center text-muted-foreground">{{ adminText('k00l0') }}</td>
            </tr>
            <tr v-for="agent in agents" v-else :key="agent.agentId" class="hover:bg-muted/40">
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <Bot class="size-4 shrink-0 text-muted-foreground" />
                  <Badge variant="secondary">{{ adminText('k00l6') }}</Badge>
                </div>
              </td>
              <td class="px-4 py-3 font-medium">{{ agent.username }}</td>
              <td class="px-4 py-3 text-muted-foreground">{{ agent.nickname || '—' }}</td>
              <td class="px-4 py-3 font-mono text-xs text-muted-foreground">{{ agent.tokenPrefix }}…</td>
              <td class="px-4 py-3">
                <Badge :variant="agent.enabled === 1 ? 'secondary' : 'outline'" :class="agent.enabled === 1 ? '' : 'text-muted-foreground'">
                  {{ agent.enabled === 1 ? adminText('k00kd') : adminText('k00ke') }}
                </Badge>
              </td>
              <td class="px-4 py-3 text-xs text-muted-foreground">{{ formatTime(agent.lastUsedAt) }}</td>
              <td class="px-4 py-3 text-xs text-muted-foreground">{{ formatTime(agent.createdAt) }}</td>
              <td class="px-4 py-3">
                <div class="flex items-center justify-end gap-1">
                  <AdminActionButton compact :title="adminText('k00kj')" @click="openEdit(agent)">
                    <Pencil class="size-3.5" />
                  </AdminActionButton>
                  <AdminActionButton compact tone="primary" :title="adminText('k00kk')" @click="rotating = agent; rotatingToken = ''; copied = false">
                    <KeyRound class="size-3.5" />
                  </AdminActionButton>
                  <AdminActionButton
                    v-if="agent.enabled === 1"
                    compact
                    tone="danger"
                    :title="adminText('k00kl')"
                    @click="disabling = agent"
                  >
                    <ShieldOff class="size-3.5" />
                  </AdminActionButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <Dialog :open="creating" @update:open="(open) => !open && (creating = false)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ adminText('k00kr') }}</DialogTitle>
          <DialogDescription>{{ adminText('k00ks') }}</DialogDescription>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="submitCreate">
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00i6') }}
            <Input v-model="form.username" :placeholder="adminText('k00l8')" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00ka') }}
            <Input v-model="form.nickname" maxlength="64" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00kb') }}
            <Input v-model="form.webhookEndpoint" placeholder="https://example.com/hook" />
          </label>
          <DialogFooter>
            <Button variant="outline" type="button" @click="creating = false">{{ adminText('k009q') }}</Button>
            <Button type="submit" :disabled="saving">{{ saving ? adminText('k00kp') : adminText('k00ko') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog :open="editing !== null" @update:open="(open) => !open && (editing = null)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ adminText('k00kq') }} · {{ form.username }}</DialogTitle>
          <DialogDescription>{{ adminText('k00l9') }}</DialogDescription>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="submitUpdate">
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00ka') }}
            <Input v-model="form.nickname" maxlength="64" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00kb') }}
            <Input v-model="form.webhookEndpoint" placeholder="https://example.com/hook" />
          </label>
          <div class="grid gap-2 text-sm font-medium">
            {{ adminText('k00kc') }}
            <div class="flex h-9 items-center justify-between rounded-md border bg-background px-3">
              <span class="text-sm text-muted-foreground">{{ form.enabled ? adminText('k00kd') : adminText('k00ke') }}</span>
              <Switch v-model="form.enabled" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" type="button" @click="editing = null">{{ adminText('k009q') }}</Button>
            <Button type="submit" :disabled="saving">{{ saving ? adminText('k00kp') : adminText('k00ko') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog :open="rotating !== null" @update:open="(open) => !open && !saving && (rotating = null)">
      <DialogContent class="sm:max-w-lg" :show-close-button="!saving">
        <DialogHeader>
          <DialogTitle>{{ adminText('k00kk') }}</DialogTitle>
          <DialogDescription>{{ adminText('k00kt') }}</DialogDescription>
        </DialogHeader>
        <div v-if="rotatingToken" class="space-y-3">
          <div class="rounded-md border bg-muted/30 p-3">
            <code class="block break-all font-mono text-sm">{{ rotatingToken }}</code>
          </div>
          <div class="flex items-start gap-2 rounded-md border border-amber-300/40 bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-950/30 dark:text-amber-200">
            <Bot class="mt-0.5 size-4 shrink-0" />
            <span>{{ adminText('k00ku') }}</span>
          </div>
          <DialogFooter>
            <Button variant="outline" type="button" :disabled="saving" @click="rotating = null">{{ adminText('k009q') }}</Button>
            <Button type="button" :disabled="copied" @click="copyToken">
              <Copy class="size-4" />
              {{ copied ? adminText('k00kv') : adminText('k00kw') }}
            </Button>
          </DialogFooter>
        </div>
        <DialogFooter v-else>
          <Button variant="outline" type="button" :disabled="saving" @click="rotating = null">{{ adminText('k009q') }}</Button>
          <Button type="button" :disabled="saving" @click="confirmRotate">
            <KeyRound class="size-4" />
            {{ saving ? adminText('k00kp') : adminText('k00kx') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <AdminConfirmDialog
      :open="disabling !== null"
      :title="adminText('k00kl')"
      :description="`${adminText('k00ky')}${disabling?.username || ''}${adminText('k00kz')}`"
      :loading="saving"
      @update:open="(open) => !open && (disabling = null)"
      @confirm="confirmDisable"
    />
  </BasicPage>
</template>
