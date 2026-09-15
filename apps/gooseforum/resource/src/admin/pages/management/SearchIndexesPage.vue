<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { CheckCheck, Loader2, RefreshCw, Search } from '@lucide/vue'
import { BasicPage } from '@/admin/components/global-layout'
import AdminSection from '@/admin/components/AdminSection.vue'
import AdminConfirmDialog from '@/admin/components/AdminConfirmDialog.vue'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import { createSearchMaintenance, getSearchMaintenance } from '@/admin/runtime/api'
import { maintenanceActive, maintenancePoller } from '@/admin/runtime/search-maintenance'
import type { SearchMaintenanceRequest, SearchMaintenanceStatus } from '@/admin/runtime/search-maintenance'
import { adminToast } from '@/admin/runtime/toast'

const { t, locale } = useI18n()
const text = (key: string, values: Record<string, unknown> = {}) => t(`searchAdmin.${key}`, values)
const status = ref<SearchMaintenanceStatus | null>(null)
const loading = ref(false)
const submitting = ref(false)
const loadError = ref(false)
const confirmIndex = ref<SearchMaintenanceRequest['index'] | null>(null)
const active = computed(() => status.value?.jobs.some(maintenanceActive) ?? false)
const disabled = computed(() => !status.value?.available || !status.value?.maintenanceEnabled || active.value || submitting.value || loading.value || loadError.value)
const jobStatuses = ['pending', 'running', 'success', 'failed', 'retrying']
let disposed = false

async function load() {
  loading.value = true
  loadError.value = false
  try {
    const next = await getSearchMaintenance()
    if (!disposed) status.value = next
    return next.jobs.some(maintenanceActive)
  } catch {
    if (!disposed) loadError.value = true
    return false
  } finally {
    if (!disposed) loading.value = false
  }
}
const poller = maintenancePoller(load, () => !document.hidden)
const refresh = () => { void poller.refresh() }

async function submit(index: SearchMaintenanceRequest['index'], action: SearchMaintenanceRequest['action']) {
  if (disabled.value) return
  confirmIndex.value = null
  submitting.value = true
  try {
    const response = await createSearchMaintenance({ index, action })
    if (disposed) return
    adminToast.success(text(response.created ? 'submitted' : 'alreadyRunning'))
    await poller.refresh()
  } catch (err) {
    if (!disposed) adminToast.error(err, text('submitFailed'))
  } finally {
    if (!disposed) submitting.value = false
  }
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'medium' }).format(new Date(value))
}
function versions(values: Record<string, number>) {
  return Object.entries(values).map(([version, count]) => `${version === '0' ? text('unmarked') : `v${version}`} × ${count.toLocaleString()}`).join(' · ') || '—'
}
onMounted(() => { refresh(); document.addEventListener('visibilitychange', refresh) })
onUnmounted(() => { disposed = true; poller.stop(); document.removeEventListener('visibilitychange', refresh) })
</script>

<template>
  <BasicPage :title="text('title')" :description="text('description')" sticky>
    <template #actions>
      <div class="flex flex-wrap gap-2">
        <Button variant="outline" size="sm" :disabled="loading" @click="refresh"><RefreshCw class="size-4" :class="{ 'animate-spin': loading }" />{{ text('refresh') }}</Button>
        <Button variant="outline" size="sm" :disabled="disabled" @click="submit('all', 'check')"><CheckCheck class="size-4" />{{ text('checkAll') }}</Button>
        <Button size="sm" :disabled="disabled" @click="confirmIndex = 'all'">{{ text('rebuildAll') }}</Button>
      </div>
    </template>
    <div class="space-y-5">
      <div v-if="loadError" role="alert" class="rounded-lg border border-destructive/30 p-4 text-sm text-destructive">{{ text('loadFailed') }}</div>
      <p v-if="!status && loading" role="status" class="text-sm text-muted-foreground">{{ text('loading') }}</p>
      <template v-if="status">
        <div class="rounded-lg border bg-muted/20 p-4 text-sm">
          <p v-if="!status.configured" role="status">{{ text('notConfigured') }}</p>
          <p v-else-if="!status.available" role="alert">{{ text('offline') }}</p>
          <p v-else class="font-medium">{{ text('engine') }} <span class="font-mono">{{ status.engineVersion }}</span></p>
          <p v-if="!status.maintenanceEnabled" class="mt-2 font-medium">{{ text('readOnly') }}</p>
          <p class="mt-2 text-xs leading-relaxed text-muted-foreground">{{ text('checkNote') }} {{ text('versionsNote') }}</p>
        </div>
        <div class="grid gap-4 xl:grid-cols-2">
          <AdminSection v-for="item in status.indexes" :key="item.index">
            <template #header>
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="flex items-center gap-2 font-semibold"><Search class="size-4 text-muted-foreground" />{{ text(item.index) }} <span class="font-mono text-xs font-normal text-muted-foreground">{{ item.index }}</span></div>
                <Badge v-if="status.available && !item.exists" variant="destructive">{{ text('missingIndex') }}</Badge>
                <Badge v-else-if="item.indexing" variant="secondary">{{ text('indexing') }}</Badge>
                <Badge v-else-if="item.lastCheck" :variant="item.lastCheck.complete && item.lastCheck.expectedVersion === item.expectedVersion ? 'default' : 'secondary'">{{ text(item.lastCheck.expectedVersion !== item.expectedVersion ? 'versionMismatch' : !item.lastCheck.stable ? 'unstable' : item.lastCheck.complete ? 'complete' : 'drift') }}</Badge>
                <Badge v-else variant="outline">{{ text('unchecked') }}</Badge>
              </div>
            </template>
            <div class="space-y-4 p-4">
              <dl class="grid grid-cols-2 gap-4 text-sm">
                <div><dt class="text-xs text-muted-foreground">{{ text('documents') }}</dt><dd class="mt-1 text-2xl font-semibold tabular-nums">{{ status.available ? item.documents.toLocaleString() : '—' }}</dd></div>
                <div><dt class="text-xs text-muted-foreground">{{ text('targetVersion') }}</dt><dd class="mt-1 text-2xl font-semibold">v{{ item.expectedVersion }}</dd></div>
                <div class="col-span-2"><dt class="text-xs text-muted-foreground">{{ text('observedVersion') }}</dt><dd class="mt-1">{{ item.lastCheck ? versions(item.lastCheck.observedVersions) : text('unchecked') }}</dd></div>
              </dl>
              <div v-if="item.lastCheck" class="space-y-3 rounded-md bg-muted/35 p-3 text-xs">
                <dl class="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <div v-for="key in (['expected', 'missing', 'extra', 'outdated'] as const)" :key="key"><dt class="text-muted-foreground">{{ text(key) }}</dt><dd class="mt-1 font-semibold tabular-nums">{{ item.lastCheck[key].toLocaleString() }}</dd></div>
                </dl>
                <p>{{ text('settings') }}: {{ text(item.lastCheck.settingsOK ? 'settingsOK' : 'settingsDrift') }}</p>
                <p class="text-muted-foreground">{{ text('lastCheck') }}: {{ formatTime(item.lastCheck.checkedAt) }}</p>
              </div>
              <div class="flex flex-wrap gap-2">
                <Button size="sm" variant="outline" :disabled="disabled" @click="submit(item.index, 'check')">{{ text('check') }}</Button>
                <Button size="sm" variant="outline" :disabled="disabled" @click="confirmIndex = item.index">{{ text('rebuild') }}</Button>
              </div>
            </div>
          </AdminSection>
        </div>
        <AdminSection>
          <template #header><span class="text-sm font-semibold">{{ text('jobs') }}</span></template>
          <div class="divide-y">
            <p v-if="!status.jobs.length" class="p-4 text-sm text-muted-foreground">{{ text('emptyJobs') }}</p>
            <div v-for="job in status.jobs" :key="job.id" class="space-y-2 p-4 text-sm">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div class="flex flex-wrap items-center gap-2"><Loader2 v-if="maintenanceActive(job)" class="size-4 animate-spin" /><span class="font-mono text-xs text-muted-foreground">#{{ job.id }}</span><span>{{ text(job.index) }} · {{ text(job.action) }}</span><Badge variant="secondary">{{ text(job.phase === 'skipped' ? 'skipped' : jobStatuses[job.status] || 'failed') }}</Badge></div>
                <Button v-if="job.status === 3" size="sm" variant="outline" :disabled="disabled" @click="job.action === 'rebuild' ? confirmIndex = job.index : submit(job.index, job.action)">{{ text('retry') }}</Button>
              </div>
              <p v-if="maintenanceActive(job)" role="status" class="text-xs text-muted-foreground">{{ job.currentIndex ? text(job.currentIndex) + ' · ' : '' }}{{ text(job.phase) }} · {{ text('processed', { count: job.processed }) }}</p>
              <p v-if="job.errorCode" class="text-xs text-destructive">{{ text('failure') }}</p>
              <p class="text-xs text-muted-foreground">{{ formatTime(job.createdAt) }}</p>
            </div>
          </div>
        </AdminSection>
      </template>
    </div>
    <AdminConfirmDialog :open="confirmIndex !== null" :title="text('confirmTitle', { index: confirmIndex ? text(confirmIndex) : '' })" :description="text('confirmDescription')" :confirm-text="text('rebuild')" :cancel-text="text('cancel')" :loading="submitting" @update:open="(open) => { if (!open) confirmIndex = null }" @confirm="confirmIndex && submit(confirmIndex, 'rebuild')" />
  </BasicPage>
</template>
