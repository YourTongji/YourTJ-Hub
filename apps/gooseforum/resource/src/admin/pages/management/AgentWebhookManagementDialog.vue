<script setup lang="ts">
import { Checkbox } from '@/admin/components/ui/checkbox'
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Copy, RotateCw, Send, ShieldCheck } from '@lucide/vue'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/admin/components/ui/dialog'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import { Switch } from '@/admin/components/ui/switch'
import { getAgentInteractionIntents, getAgentList, getAgentWebhookDeliveries, redeliverAgentWebhookDelivery, replayAgentInteractionIntent, rotateAgentWebhookSecret, saveAgentWebhookConfig, testAgentWebhook } from '@/admin/runtime/api'
import type { AdminAgent, AdminAgentInteractionIntent, AdminAgentWebhookDelivery } from '@/admin/types'

const props = defineProps<{ open: boolean, agent: AdminAgent | null }>()
const emit = defineEmits<{ 'update:open': [open: boolean], updated: [] }>()
const { t } = useI18n()

const pageSize = 10
const eventOptions = [
  { value: 'agent.mentioned', label: 'agentWebhook.mentioned' },
  { value: 'agent.post_replied', label: 'agentWebhook.postReplied' },
  { value: 'agent.topic_commented', label: 'agentWebhook.topicCommented' },
  { value: 'forum.topic_created', label: 'agentWebhook.topicCreated' },
  { value: 'forum.post_created', label: 'agentWebhook.postCreated' },
] as const

const currentAgent = ref<AdminAgent | null>(null)
const form = reactive({ eventsEnabled: false, eventTypes: [] as string[], webhookEnabled: false, webhookEndpoint: '' })
const configSaving = ref(false)
const testSending = ref(false)
const secretRotating = ref(false)
const emergencyRotation = ref(false)
const rotationConfirmOpen = ref(false)
const newSecret = ref('')
const newSecretVersion = ref(0)
const secretCopied = ref(false)
const conflict = ref(false)
const actionError = ref('')
const notice = ref('')

const section = ref<'deliveries' | 'intents'>('deliveries')
const deliveryPage = ref(1)
const deliveryTotal = ref(0)
const deliveries = ref<AdminAgentWebhookDelivery[]>([])
const deliveriesLoading = ref(false)
const deliveriesError = ref('')
const intentPage = ref(1)
const intentTotal = ref(0)
const intents = ref<AdminAgentInteractionIntent[]>([])
const intentsLoading = ref(false)
const intentsError = ref('')
const busyDeliveryId = ref<number | null>(null)
const busyIntentId = ref<string | number | null>(null)
let dialogSession = 0

// Closing or selecting another Agent invalidates every in-flight response,
// including one-time secrets. Each session owns its loading and action state.
watch(() => [props.open, props.agent?.agentId] as const, () => {
  dialogSession++
  newSecret.value = ''
  secretCopied.value = false
  rotationConfirmOpen.value = false
  emergencyRotation.value = false
  configSaving.value = false
  testSending.value = false
  secretRotating.value = false
  busyDeliveryId.value = null
  busyIntentId.value = null
  deliveries.value = []
  intents.value = []
}, { immediate: true })

const deliveryPages = computed(() => Math.max(1, Math.ceil(deliveryTotal.value / pageSize)))
const intentPages = computed(() => Math.max(1, Math.ceil(intentTotal.value / pageSize)))

function syncForm(agent: AdminAgent) {
  currentAgent.value = agent
  form.eventsEnabled = agent.eventsEnabled
  form.eventTypes = [...agent.eventTypes]
  form.webhookEnabled = agent.webhookEnabled
  form.webhookEndpoint = agent.webhookEndpoint || ''
  conflict.value = false
}

watch(() => [props.open, props.agent] as const, ([open, agent]) => {
  if (!open || !agent) return
  syncForm(agent)
  section.value = 'deliveries'
  deliveryPage.value = 1
  intentPage.value = 1
  actionError.value = ''
  notice.value = ''
  void loadDeliveries()
  void loadIntents()
}, { immediate: true })

function messageCode(error: unknown) {
  return (error as { messageCode?: string })?.messageCode || ''
}

function isConfigConflict(error: unknown) {
  return messageCode(error) === 'agent.webhook.configConflict'
    || messageCode(error).toLowerCase().includes('configconflict')
}

async function loadLatestConfig() {
  const agent = currentAgent.value
  if (!agent) return
  const session = dialogSession
  try {
    const latest = (await getAgentList()).find(item => item.agentId === agent.agentId)
    if (session !== dialogSession) return
    if (!latest) throw new Error(t('agentWebhook.loadFailed'))
    syncForm(latest)
    emit('updated')
  } catch (error) {
    if (session !== dialogSession) return
    actionError.value = error instanceof Error ? error.message : t('agentWebhook.loadFailed')
  }
}

async function saveConfig() {
  const agent = currentAgent.value
  if (!agent || configSaving.value) return
  const session = dialogSession
  configSaving.value = true
  actionError.value = ''
  notice.value = ''
  try {
    const updated = await saveAgentWebhookConfig({
      agentId: agent.agentId,
      configVersion: agent.configVersion,
      eventsEnabled: form.eventsEnabled,
      eventTypes: [...form.eventTypes],
      webhookEnabled: form.webhookEnabled,
      webhookEndpoint: form.webhookEndpoint.trim(),
    })
    if (session !== dialogSession) return
    syncForm(updated)
    notice.value = t('agentWebhook.configSaved')
    emit('updated')
  } catch (error) {
    if (session !== dialogSession) return
    if (isConfigConflict(error)) {
      conflict.value = true
      actionError.value = t('agentWebhook.configConflict')
    } else {
      actionError.value = error instanceof Error ? error.message : t('agentWebhook.loadFailed')
    }
  } finally {
    if (session === dialogSession) configSaving.value = false
  }
}

async function sendTest() {
  const agent = currentAgent.value
  if (!agent || testSending.value) return
  const session = dialogSession
  testSending.value = true
  actionError.value = ''
  notice.value = ''
  try {
    const delivery = await testAgentWebhook(agent.agentId)
    if (session !== dialogSession) return
    notice.value = delivery.status === 'accepted'
      ? t('agentWebhook.testAccepted')
      : t('agentWebhook.testPending')
    deliveryPage.value = 1
    await loadDeliveries()
  } catch (error) {
    if (session !== dialogSession) return
    actionError.value = error instanceof Error ? error.message : t('agentWebhook.testFailed')
  } finally {
    if (session === dialogSession) testSending.value = false
  }
}

async function rotateSecret() {
  const agent = currentAgent.value
  if (!agent || secretRotating.value) return
  const session = dialogSession
  secretRotating.value = true
  actionError.value = ''
  try {
    const result = await rotateAgentWebhookSecret({
      agentId: agent.agentId,
      configVersion: agent.configVersion,
      emergency: emergencyRotation.value,
    })
    if (session !== dialogSession) return
    newSecret.value = result.secret
    newSecretVersion.value = result.secretVersion
    secretCopied.value = false
    rotationConfirmOpen.value = false
    currentAgent.value = { ...agent, configVersion: result.configVersion, secretConfigured: true, secretVersion: result.secretVersion }
    emergencyRotation.value = false
    emit('updated')
  } catch (error) {
    if (session !== dialogSession) return
    if (isConfigConflict(error)) {
      conflict.value = true
      actionError.value = t('agentWebhook.configConflict')
    } else {
      actionError.value = error instanceof Error ? error.message : t('agentWebhook.loadFailed')
    }
  } finally {
    if (session === dialogSession) secretRotating.value = false
  }
}

async function copySecret() {
  if (!newSecret.value) return
  const session = dialogSession
  try {
    await navigator.clipboard.writeText(newSecret.value)
    if (session !== dialogSession) return
    secretCopied.value = true
  } catch {
    if (session !== dialogSession) return
    actionError.value = t('agentWebhook.copyFailed')
  }
}

function closeSecret() {
  newSecret.value = ''
  secretCopied.value = false
}

function formatTime(value?: string | number | null) {
  if (!value) return '—'
  const date = typeof value === 'number'
    ? new Date(value < 100_000_000_000 ? value * 1000 : value)
    : new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString()
}

function statusText(status: string) {
  const key: Record<string, string> = {
    pending: 'pending', running: 'running', accepted: 'accepted', retry_wait: 'retryWait',
    dead: 'dead', failed: 'failed', cancelled: 'cancelled', expired: 'expired',
  }
  const label = key[status]
  return label ? t(`agentWebhook.${label}`) : status
}

async function loadDeliveries() {
  const agent = currentAgent.value
  if (!agent) return
  const session = dialogSession
  deliveriesLoading.value = true
  deliveriesError.value = ''
  try {
    const result = await getAgentWebhookDeliveries(agent.agentId, deliveryPage.value, pageSize)
    if (session !== dialogSession) return
    deliveries.value = result.list
    deliveryTotal.value = result.total
  } catch (error) {
    if (session !== dialogSession) return
    deliveriesError.value = error instanceof Error ? error.message : t('agentWebhook.loadFailed')
  } finally {
    if (session === dialogSession) deliveriesLoading.value = false
  }
}

async function loadIntents() {
  const agent = currentAgent.value
  if (!agent) return
  const session = dialogSession
  intentsLoading.value = true
  intentsError.value = ''
  try {
    const result = await getAgentInteractionIntents(agent.agentId, intentPage.value, pageSize)
    if (session !== dialogSession) return
    intents.value = result.list
    intentTotal.value = result.total
  } catch (error) {
    if (session !== dialogSession) return
    intentsError.value = error instanceof Error ? error.message : t('agentWebhook.loadFailed')
  } finally {
    if (session === dialogSession) intentsLoading.value = false
  }
}

async function redeliver(delivery: AdminAgentWebhookDelivery) {
  const agent = currentAgent.value
  if (!agent || busyDeliveryId.value !== null) return
  const session = dialogSession
  busyDeliveryId.value = delivery.id
  actionError.value = ''
  try {
    await redeliverAgentWebhookDelivery(agent.agentId, delivery.id)
    if (session !== dialogSession) return
    await loadDeliveries()
  } catch (error) {
    if (session !== dialogSession) return
    actionError.value = error instanceof Error ? error.message : t('agentWebhook.redeliverFailed')
  } finally {
    if (session === dialogSession) busyDeliveryId.value = null
  }
}

async function replay(intent: AdminAgentInteractionIntent) {
  const agent = currentAgent.value
  if (!agent || busyIntentId.value !== null) return
  const session = dialogSession
  busyIntentId.value = intent.intentId
  actionError.value = ''
  try {
    await replayAgentInteractionIntent(agent.agentId, intent.intentId)
    if (session !== dialogSession) return
    await loadIntents()
  } catch (error) {
    if (session !== dialogSession) return
    actionError.value = error instanceof Error ? error.message : t('agentWebhook.replayFailed')
  } finally {
    if (session === dialogSession) busyIntentId.value = null
  }
}

function canReplay(status: string) {
  return ['dead', 'failed'].includes(status)
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="grid-cols-1 max-h-[92vh] overflow-y-auto sm:max-w-5xl">
      <DialogHeader>
        <DialogTitle>{{ t('agentWebhook.title') }} · {{ agent?.username }}</DialogTitle>
        <DialogDescription>{{ t('agentWebhook.description') }}</DialogDescription>
      </DialogHeader>

      <template v-if="currentAgent">
        <section class="grid min-w-0 gap-4 rounded-lg border p-4">
          <div class="grid gap-3 md:grid-cols-2">
            <div class="flex items-center justify-between gap-4 rounded-md bg-muted/35 px-3 py-2">
              <span class="text-sm font-medium">{{ t('agentWebhook.eventsEnabled') }}</span>
              <Switch v-model="form.eventsEnabled" />
            </div>
            <div class="flex items-center justify-between gap-4 rounded-md bg-muted/35 px-3 py-2">
              <span class="text-sm font-medium">{{ t('agentWebhook.webhookEnabled') }}</span>
              <Switch v-model="form.webhookEnabled" />
            </div>
          </div>

          <label class="grid gap-2 text-sm font-medium">
            {{ t('agentWebhook.endpoint') }}
            <Input v-model="form.webhookEndpoint" type="url" placeholder="https://example.com/webhook" autocomplete="url" />
          </label>

          <fieldset class="grid min-w-0 gap-2">
            <legend class="text-sm font-medium">{{ t('agentWebhook.eventTypes') }}</legend>
            <label v-for="event in eventOptions" :key="event.value" class="flex items-center gap-2 text-sm">
              <Checkbox
                :model-value="form.eventTypes.includes(event.value)"
                @update:model-value="form.eventTypes = $event === true ? [...form.eventTypes, event.value] : form.eventTypes.filter((type) => type !== event.value)"
              />
              {{ t(event.label) }}
            </label>
          </fieldset>

          <div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <span>{{ t('agentWebhook.configVersion', { version: currentAgent.configVersion }) }}</span>
            <span>·</span>
            <span>{{ t('agentWebhook.endpointGeneration', { generation: currentAgent.endpointGeneration }) }}</span>
            <span>·</span>
            <span>{{ t('agentWebhook.subscriptionGeneration', { generation: currentAgent.subscriptionGeneration }) }}</span>
          </div>

          <div class="flex flex-wrap items-center gap-2 rounded-md border bg-muted/20 p-3 text-sm">
            <ShieldCheck class="size-4 text-muted-foreground" />
            <span class="font-medium">{{ t('agentWebhook.secret') }}</span>
            <span class="text-muted-foreground">
              {{ currentAgent.secretConfigured ? t('agentWebhook.secretConfigured', { version: currentAgent.secretVersion }) : t('agentWebhook.secretMissing') }}
            </span>
            <Button class="ml-auto" variant="outline" size="sm" type="button" @click="rotationConfirmOpen = true">
              <RotateCw class="size-4" /> {{ t('agentWebhook.rotateSecret') }}
            </Button>
          </div>

          <p class="text-xs text-muted-foreground">{{ t('agentWebhook.emergencyHelp') }}</p>
          <div v-if="currentAgent.summaryUnavailable" class="rounded-md border border-amber-500/30 bg-amber-500/5 p-3 text-sm">
            {{ t('agentWebhook.summaryUnavailable') }}
          </div>
          <dl v-else class="grid gap-2 rounded-md bg-muted/25 p-3 text-sm sm:grid-cols-3">
            <div><dt class="text-muted-foreground">{{ t('agentWebhook.lastAccepted') }}</dt><dd>{{ formatTime(currentAgent.latestAcceptedAt) }}</dd></div>
            <div><dt class="text-muted-foreground">{{ t('agentWebhook.pendingCount') }}</dt><dd>{{ currentAgent.pendingCount }}</dd></div>
            <div v-if="currentAgent.pauseReason"><dt class="text-muted-foreground">{{ t('agentWebhook.pauseReason') }}</dt><dd class="break-words">{{ currentAgent.pauseReason }}</dd></div>
          </dl>

          <div class="flex flex-wrap items-center gap-2">
            <Button type="button" :disabled="configSaving" @click="saveConfig">
              {{ configSaving ? t('agentWebhook.saving') : t('agentWebhook.save') }}
            </Button>
            <Button type="button" variant="outline" :disabled="testSending" @click="sendTest">
              <Send class="size-4" /> {{ testSending ? t('agentWebhook.saving') : t('agentWebhook.test') }}
            </Button>
            <span class="text-xs text-muted-foreground">{{ t('agentWebhook.testNotReply') }}</span>
          </div>
        </section>

        <div v-if="notice" role="status" class="rounded-md border border-emerald-500/30 bg-emerald-500/5 p-3 text-sm">{{ notice }}</div>
        <div v-if="actionError" role="alert" class="grid gap-2 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
          <span>{{ actionError }}</span>
          <Button v-if="conflict" class="w-fit" size="sm" variant="outline" type="button" @click="loadLatestConfig">{{ t('agentWebhook.reloadLatest') }}</Button>
        </div>

        <section class="grid min-w-0 gap-3 rounded-lg border p-4">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex flex-wrap items-center gap-1" role="tablist" :aria-label="t('agentWebhook.deliveries')">
              <Button role="tab" :aria-selected="section === 'deliveries'" :variant="section === 'deliveries' ? 'secondary' : 'ghost'" size="sm" type="button" @click="section = 'deliveries'; void loadDeliveries()">{{ t('agentWebhook.deliveries') }}</Button>
              <Button role="tab" :aria-selected="section === 'intents'" :variant="section === 'intents' ? 'secondary' : 'ghost'" size="sm" type="button" @click="section = 'intents'; void loadIntents()">{{ t('agentWebhook.intents') }}</Button>
            </div>
            <Button variant="outline" size="sm" type="button" @click="section === 'deliveries' ? loadDeliveries() : loadIntents()">{{ t('agentWebhook.refresh') }}</Button>
          </div>

          <template v-if="section === 'deliveries'">
            <p class="text-xs text-muted-foreground">{{ t('agentWebhook.acceptedMeaning') }}</p>
            <div v-if="deliveriesError" role="alert" class="text-sm text-destructive">{{ deliveriesError }}</div>
            <p v-else-if="deliveriesLoading" class="py-5 text-center text-sm text-muted-foreground">{{ t('agentWebhook.loading') }}</p>
            <p v-else-if="!deliveries.length" class="py-5 text-center text-sm text-muted-foreground">{{ t('agentWebhook.noDeliveries') }}</p>
            <div v-else class="grid gap-3">
              <article v-for="delivery in deliveries" :key="delivery.id" class="grid gap-2 rounded-md border p-3 text-sm" :data-testid="`delivery-${delivery.id}`">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
                    <span class="font-medium">{{ t('agentWebhook.deliveryId') }} {{ delivery.id }}</span>
                    <span class="break-all text-muted-foreground">{{ t('agentWebhook.eventId') }} <code>{{ delivery.eventId }}</code></span>
                    <span class="rounded-full bg-muted px-2 py-0.5 text-xs">{{ statusText(delivery.status) }}</span>
                  </div>
                  <Button v-if="['dead', 'failed'].includes(delivery.status)" size="sm" variant="outline" type="button" :disabled="busyDeliveryId !== null" @click="redeliver(delivery)">
                    {{ busyDeliveryId === delivery.id ? t('agentWebhook.saving') : t('agentWebhook.redeliver') }}
                  </Button>
                </div>
                <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                  <span>{{ t('agentWebhook.endpointGeneration', { generation: delivery.endpointGeneration }) }}</span>
                  <span>{{ t('agentWebhook.round') }} {{ delivery.round }}</span>
                  <span>{{ t('agentWebhook.attempts') }} {{ delivery.attemptCount }}</span>
                  <span>{{ t('agentWebhook.totalAttempts') }} {{ delivery.totalAttempts }}</span>
                  <span v-if="delivery.nextRunAt">{{ t('agentWebhook.nextRun') }} {{ formatTime(delivery.nextRunAt) }}</span>
                  <span v-if="delivery.expiresAt">{{ t('agentWebhook.expires') }} {{ formatTime(delivery.expiresAt) }}</span>
                </div>
                <p v-if="delivery.reason" class="break-words text-xs text-muted-foreground">{{ t('agentWebhook.deliveryReason') }}: {{ delivery.reason }}</p>
                <div v-if="delivery.attempts.length" class="grid gap-1 border-t pt-2">
                  <div v-for="attempt in delivery.attempts" :key="attempt.id" class="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                    <span>#{{ attempt.number }} · {{ t('agentWebhook.round') }} {{ attempt.round }}</span>
                    <span v-if="attempt.httpStatus">{{ t('agentWebhook.httpStatus') }} {{ attempt.httpStatus }}</span>
                    <span v-if="attempt.errorClass">{{ t('agentWebhook.errorClass') }} {{ attempt.errorClass }}</span>
                    <span v-if="attempt.durationMs !== null && attempt.durationMs !== undefined">{{ t('agentWebhook.duration', { duration: attempt.durationMs }) }}</span>
                    <span v-if="attempt.authorizedAt">{{ t('agentWebhook.authorizedAt') }} {{ formatTime(attempt.authorizedAt) }}</span>
                    <span v-if="attempt.completedAt">{{ t('agentWebhook.completedAt') }} {{ formatTime(attempt.completedAt) }}</span>
                  </div>
                </div>
              </article>
            </div>
            <div class="flex items-center justify-between gap-3 text-sm">
              <span>{{ t('agentWebhook.pageOf', { page: deliveryPage }) }} · {{ deliveryTotal }}</span>
              <div class="flex gap-2">
                <Button variant="outline" size="sm" type="button" :disabled="deliveryPage <= 1 || deliveriesLoading" @click="deliveryPage--; void loadDeliveries()">{{ t('agentWebhook.previous') }}</Button>
                <Button variant="outline" size="sm" type="button" :disabled="deliveryPage >= deliveryPages || deliveriesLoading" @click="deliveryPage++; void loadDeliveries()">{{ t('agentWebhook.next') }}</Button>
              </div>
            </div>
          </template>

          <template v-else>
            <div v-if="intentsError" role="alert" class="text-sm text-destructive">{{ intentsError }}</div>
            <p v-else-if="intentsLoading" class="py-5 text-center text-sm text-muted-foreground">{{ t('agentWebhook.loading') }}</p>
            <p v-else-if="!intents.length" class="py-5 text-center text-sm text-muted-foreground">{{ t('agentWebhook.noIntents') }}</p>
            <div v-else class="grid gap-3">
              <article v-for="intent in intents" :key="intent.intentId" class="grid gap-2 rounded-md border p-3 text-sm" :data-testid="`intent-${intent.intentId}`">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
                    <span class="break-all font-medium">{{ t('agentWebhook.intentId') }} {{ intent.intentId }}</span>
                    <span class="rounded-full bg-muted px-2 py-0.5 text-xs">{{ statusText(intent.status) }}</span>
                  </div>
                  <Button v-if="canReplay(intent.status)" size="sm" variant="outline" type="button" :disabled="busyIntentId !== null" @click="replay(intent)">
                    {{ busyIntentId === intent.intentId ? t('agentWebhook.saving') : t('agentWebhook.replay') }}
                  </Button>
                </div>
                <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                  <span v-if="intent.sourceOccurrenceId" class="break-all">{{ t('agentWebhook.sourceOccurrence') }} <code>{{ intent.sourceOccurrenceId }}</code></span>
                  <span v-if="intent.postId">{{ t('agentWebhook.post') }} {{ intent.postId }}</span>
                  <span v-if="intent.revision">{{ t('agentWebhook.revision') }} {{ intent.revision }}</span>
                  <span>{{ t('agentWebhook.retryCount') }} {{ intent.retryCount ?? 0 }}</span>
                  <span>{{ t('agentWebhook.expires') }} {{ formatTime(intent.expiresAt) }}</span>
                </div>
                <p v-if="intent.errorCode" class="break-words text-xs text-destructive">{{ t('agentWebhook.lastError') }}: {{ intent.errorCode }}</p>
              </article>
            </div>
            <div class="flex items-center justify-between gap-3 text-sm">
              <span>{{ t('agentWebhook.pageOf', { page: intentPage }) }} · {{ intentTotal }}</span>
              <div class="flex gap-2">
                <Button variant="outline" size="sm" type="button" :disabled="intentPage <= 1 || intentsLoading" @click="intentPage--; void loadIntents()">{{ t('agentWebhook.previous') }}</Button>
                <Button variant="outline" size="sm" type="button" :disabled="intentPage >= intentPages || intentsLoading" @click="intentPage++; void loadIntents()">{{ t('agentWebhook.next') }}</Button>
              </div>
            </div>
          </template>
        </section>
      </template>

      <DialogFooter>
        <Button variant="outline" type="button" @click="emit('update:open', false)">{{ t('common.close') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <Dialog :open="rotationConfirmOpen" @update:open="rotationConfirmOpen = $event">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t('agentWebhook.rotateSecret') }}</DialogTitle>
        <DialogDescription>{{ t('agentWebhook.secretShownOnce') }}</DialogDescription>
      </DialogHeader>
      <label class="flex items-start gap-3 text-sm">
        <Switch v-model="emergencyRotation" />
        <span class="grid gap-1"><span>{{ t('agentWebhook.emergencyRotation') }}</span><span class="text-xs text-muted-foreground">{{ t('agentWebhook.emergencyHelp') }}</span></span>
      </label>
      <DialogFooter>
        <Button variant="outline" type="button" :disabled="secretRotating" @click="rotationConfirmOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" type="button" :disabled="secretRotating" @click="rotateSecret">{{ secretRotating ? t('agentWebhook.saving') : t('agentWebhook.rotateSecret') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <Dialog :open="Boolean(newSecret)" @update:open="(open) => !open && closeSecret()">
    <DialogContent class="sm:max-w-lg" :show-close-button="false">
      <DialogHeader>
        <DialogTitle>{{ t('agentWebhook.secret') }} · {{ t('agentWebhook.secretConfigured', { version: newSecretVersion }) }}</DialogTitle>
        <DialogDescription>{{ t('agentWebhook.secretShownOnce') }}</DialogDescription>
      </DialogHeader>
      <code class="block break-all rounded-md border bg-muted/35 p-3 font-mono text-sm">{{ newSecret }}</code>
      <DialogFooter>
        <Button variant="outline" type="button" @click="closeSecret">{{ t('common.close') }}</Button>
        <Button type="button" :disabled="secretCopied" @click="copySecret"><Copy class="size-4" />{{ secretCopied ? t('agentWebhook.copied') : t('agentWebhook.copySecret') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
