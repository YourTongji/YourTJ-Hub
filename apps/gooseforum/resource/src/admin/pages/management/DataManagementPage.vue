<script setup lang="ts">
import { adminText } from '@/admin/runtime/i18n-text'

import { onMounted, onUnmounted, ref } from 'vue'
import { CheckCircle2, Database, Download, FileJson, Loader2, RefreshCw, Upload, XCircle } from '@lucide/vue'
import AdminConfirmDialog from '@/admin/components/AdminConfirmDialog.vue'
import AdminSection from '@/admin/components/AdminSection.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/admin/components/ui/table'
import { createExportTask, downloadExportTask, getExportTasks, importData } from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type { AdminPayload, AdminTaskRow, ImportReport, ManageHomeProps } from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

const exportTables = ref<string[]>(['users', 'topics', 'posts', 'postRevisions'])
const exportFormat = ref<'json' | 'csv'>('json')
const exportTasks = ref<AdminTaskRow[]>([])
const tasksLoading = ref(false)
const tasksError = ref('')
const creating = ref(false)
const createConfirm = ref(false)

const importFile = ref<File | null>(null)
const importing = ref(false)
const importReport = ref<ImportReport | null>(null)

interface ExportTaskPayload {
  tables?: string[]
  format?: string
  fileName?: string
  progress?: number
  errorCount?: number
}

let pollTimer: ReturnType<typeof setInterval> | null = null

function parseTaskJson(task: AdminTaskRow): ExportTaskPayload {
  try {
    const parsed = JSON.parse(task.taskJson || '{}') as ExportTaskPayload
    return parsed || {}
  } catch {
    return {}
  }
}

function taskProgress(task: AdminTaskRow) {
  const payload = parseTaskJson(task)
  if (typeof payload.progress === 'number') return Math.min(100, Math.max(0, payload.progress))
  return 0
}

function taskStatusText(status: number) {
  const statusKey = ['k00i0', 'k00i1', 'k00i2', 'k00i3', 'k00i4'][status] || 'k00i3'
  return adminText(statusKey)
}

function hasActiveTasks() {
  return exportTasks.value.some(task => task.status !== 2 && task.status !== 3)
}

async function loadExportTasks() {
  tasksLoading.value = true
  tasksError.value = ''
  try {
    exportTasks.value = await getExportTasks()
  } catch (err) {
    tasksError.value = err instanceof Error ? err.message : adminText('k00h5')
  } finally {
    tasksLoading.value = false
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => {
    if (!hasActiveTasks()) return
    void loadExportTasks()
  }, 5000)
}

function stopPolling() {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function toggleTable(table: string) {
  if (exportTables.value.includes(table)) {
    exportTables.value = exportTables.value.filter(item => item !== table)
  } else {
    exportTables.value = [...exportTables.value, table]
  }
}

async function confirmCreateExport() {
  createConfirm.value = false
  if (exportTables.value.length === 0) {
    adminToast.warning(adminText('k00i9'))
    return
  }
  creating.value = true
  try {
    await createExportTask(exportTables.value, exportFormat.value)
    adminToast.success(adminText('k00ic'))
    await loadExportTasks()
    startPolling()
  } catch (err) {
    adminToast.error(err, adminText('k00h5'))
  } finally {
    creating.value = false
  }
}

function onImportFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  importFile.value = input.files?.[0] || null
  importReport.value = null
}

async function submitImport() {
  if (!importFile.value) {
    adminToast.warning(adminText('k00ia'))
    return
  }
  importing.value = true
  try {
    importReport.value = await importData(importFile.value)
    adminToast.success(adminText('k00id'))
  } catch (err) {
    importReport.value = null
    adminToast.error(err, adminText('k00hk'))
  } finally {
    importing.value = false
  }
}

function formatTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

onMounted(() => {
  void loadExportTasks()
  startPolling()
})

onUnmounted(stopPolling)
</script>

<template>
  <BasicPage :title="adminText('k00h0')" :description="adminText('k00h1')" sticky>
    <template #actions>
      <Button variant="outline" size="sm" type="button" :disabled="tasksLoading" @click="loadExportTasks">
        <RefreshCw class="size-4" :class="tasksLoading ? 'animate-spin' : ''" />
        {{ adminText('k004q') }}
      </Button>
    </template>

    <div class="space-y-6">
      <AdminSection>
        <template #header>
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div class="flex items-center gap-2 text-sm font-semibold"><Database class="size-4 text-muted-foreground" />{{ adminText('k00h2') }}</div>
              <p class="mt-0.5 text-xs text-muted-foreground">{{ adminText('k00h3') }}</p>
            </div>
          </div>
        </template>

        <div class="space-y-4 p-4">
          <div class="flex flex-wrap items-end gap-4">
            <div class="space-y-2">
              <div class="text-sm font-medium">{{ adminText('k00h4') }}</div>
              <div class="flex flex-wrap gap-2">
                <label
                  v-for="table in ['users', 'topics', 'posts', 'postRevisions']"
                  :key="table"
                  class="inline-flex cursor-pointer items-center gap-2 rounded-md border px-3 py-1.5 text-sm transition-colors"
                  :class="exportTables.includes(table) ? 'border-primary bg-primary/10 text-primary' : 'border-border text-muted-foreground hover:bg-muted/50'"
                >
                  <input
                    type="checkbox"
                    class="size-4 rounded border"
                    :checked="exportTables.includes(table)"
                    @change="toggleTable(table)"
                  />
                  {{ table }}
                </label>
              </div>
            </div>
            <div class="space-y-2">
              <div class="text-sm font-medium">{{ adminText('k00h6') }}</div>
              <select
                v-model="exportFormat"
                class="h-9 rounded-md border bg-background px-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <option value="json">JSON</option>
                <option value="csv">CSV</option>
              </select>
            </div>
            <Button type="button" :disabled="creating" @click="createConfirm = true">
              <Loader2 v-if="creating" class="size-4 animate-spin" />
              <Download v-else class="size-4" />
              {{ adminText('k00h7') }}
            </Button>
          </div>

          <div class="space-y-2 border-t pt-4">
            <div class="flex items-center gap-2 text-sm font-semibold"><Database class="size-4 text-muted-foreground" />{{ adminText('k00h8') }}</div>
            <div v-if="tasksLoading && exportTasks.length === 0" class="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">{{ adminText('k0046') }}</div>
            <div v-else-if="tasksError" class="rounded-lg border border-dashed p-6 text-center text-sm text-destructive">{{ tasksError }}</div>
            <div v-else-if="exportTasks.length === 0" class="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">{{ adminText('k00hq') }}</div>
            <div v-else class="space-y-2">
              <div v-for="task in exportTasks" :key="task.id" class="space-y-2 rounded-lg border bg-background p-3 text-sm">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div class="flex min-w-0 flex-wrap items-center gap-2">
                    <span class="font-mono text-xs text-muted-foreground">#{{ task.id }}</span>
                    <Badge :variant="task.status === 2 ? 'default' : task.status === 3 ? 'destructive' : 'secondary'" class="px-2 py-0 text-xs">{{ taskStatusText(task.status) }}</Badge>
                    <span class="truncate text-xs text-muted-foreground">{{ parseTaskJson(task).fileName || '-' }}</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <span class="text-xs text-muted-foreground">{{ formatTime(task.createdAt) }}</span>
                    <Button v-if="task.status === 2" type="button" size="sm" variant="outline" class="h-7 text-xs" @click="downloadExportTask(task.id)">
                      <Download class="size-3.5" />
                      {{ adminText('k00ha') }}
                    </Button>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <div class="h-2 flex-1 overflow-hidden rounded-full bg-muted">
                    <div class="h-full rounded-full bg-primary transition-all" :style="{ width: `${taskProgress(task)}%` }" />
                  </div>
                  <span class="w-10 text-right text-xs text-muted-foreground">{{ adminText('k00h9') }} {{ taskProgress(task) }}%</span>
                </div>
                <div v-if="task.lastError" class="truncate text-xs text-destructive">{{ task.lastError }}</div>
              </div>
            </div>
          </div>
        </div>
      </AdminSection>

      <AdminSection>
        <template #header>
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div class="flex items-center gap-2 text-sm font-semibold"><Upload class="size-4 text-muted-foreground" />{{ adminText('k00hb') }}</div>
              <p class="mt-0.5 text-xs text-muted-foreground">{{ adminText('k00hc') }}</p>
            </div>
          </div>
        </template>

        <div class="space-y-4 p-4">
          <div class="flex flex-wrap items-center gap-3">
            <label class="inline-flex h-9 w-fit cursor-pointer items-center gap-2 rounded-md border bg-background px-3 text-sm font-medium shadow-xs hover:bg-accent">
              <FileJson class="size-4" />
              {{ importFile ? importFile.name : adminText('k00hd') }}
              <input class="hidden" type="file" accept=".json,application/json" @change="onImportFileChange" />
            </label>
            <Button type="button" :disabled="importing" @click="submitImport">
              <Loader2 v-if="importing" class="size-4 animate-spin" />
              <Upload v-else class="size-4" />
              {{ adminText('k00he') }}
            </Button>
          </div>

          <div v-if="importReport" class="space-y-3 rounded-lg border bg-muted/10 p-4">
            <div class="flex flex-wrap items-center gap-2 text-sm font-semibold"><CheckCircle2 class="size-4 text-emerald-600" />{{ adminText('k00hf') }}</div>
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <div class="rounded-md border bg-background p-3">
                <div class="text-xs text-muted-foreground">{{ adminText('k00hg') }}</div>
                <div class="mt-1 text-lg font-semibold">{{ importReport.total }}</div>
              </div>
              <div class="rounded-md border bg-background p-3">
                <div class="text-xs text-muted-foreground">{{ adminText('k00hh') }}</div>
                <div class="mt-1 text-lg font-semibold text-emerald-600">{{ importReport.success }}</div>
              </div>
              <div class="rounded-md border bg-background p-3">
                <div class="text-xs text-muted-foreground">{{ adminText('k00hi') }}</div>
                <div class="mt-1 text-lg font-semibold text-muted-foreground">{{ importReport.skipped }}</div>
              </div>
              <div class="rounded-md border bg-background p-3">
                <div class="text-xs text-muted-foreground">{{ adminText('k00hj') }}</div>
                <div class="mt-1 text-lg font-semibold text-destructive">{{ importReport.failed }}</div>
              </div>
            </div>
            <div v-if="importReport.importedTables.length" class="flex flex-wrap items-center gap-2 text-sm">
              <span class="text-muted-foreground">{{ adminText('k00hp') }}:</span>
              <Badge v-for="table in importReport.importedTables" :key="table" variant="secondary" class="font-mono">{{ table }}</Badge>
            </div>
            <div v-if="importReport.errors.length" class="space-y-2">
              <div class="flex items-center gap-2 text-sm font-medium"><XCircle class="size-4 text-destructive" />{{ adminText('k00hl') }}</div>
              <div class="overflow-x-auto rounded-md border bg-background">
                <table class="w-full min-w-[480px] text-sm">
                  <thead class="border-b bg-muted/45 text-xs font-medium text-muted-foreground">
                    <tr>
                      <th class="h-10 px-3 text-left">{{ adminText('k00hm') }}</th>
                      <th class="h-10 px-3 text-left">{{ adminText('k00h4') }}</th>
                      <th class="h-10 px-3 text-left">{{ adminText('k00ho') }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y">
                    <tr v-for="(entry, index) in importReport.errors" :key="index">
                      <td class="px-3 py-2 font-mono text-xs text-muted-foreground">{{ entry.line }}</td>
                      <td class="px-3 py-2 font-mono text-xs">{{ entry.table }}</td>
                      <td class="px-3 py-2 text-xs text-muted-foreground">{{ entry.reason }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </AdminSection>
    </div>

    <AdminConfirmDialog
      :open="createConfirm"
      :title="adminText('k00ib')"
      :description="adminText('k00ib')"
      :confirm-text="adminText('k00h7')"
      :loading="creating"
      @update:open="(open) => !open && (createConfirm = false)"
      @confirm="confirmCreateExport"
    />
  </BasicPage>
</template>
