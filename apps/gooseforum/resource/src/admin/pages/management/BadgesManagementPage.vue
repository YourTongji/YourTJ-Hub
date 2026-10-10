<script setup lang="ts">import { adminText } from '@/admin/runtime/i18n-text'

import AdminSelect from '@/admin/components/AdminSelect.vue'
import { computed, onMounted, reactive, ref } from 'vue'
import { Edit3, Plus, RefreshCw, Trash2 } from '@lucide/vue'
import AdminActionButton from '@/admin/components/AdminActionButton.vue'
import AdminConfirmDialog from '@/admin/components/AdminConfirmDialog.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Button } from '@/admin/components/ui/button'
import { Badge } from '@/admin/components/ui/badge'
import { Input } from '@/admin/components/ui/input'
import { Textarea } from '@/admin/components/ui/textarea'
import { Switch } from '@/admin/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/admin/components/ui/dialog'
import { deleteBadge, getBadges, saveBadge } from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type { AdminBadge, AdminPayload, ManageHomeProps } from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

const emptyBadge: AdminBadge = {
  code: '',
  type: 'custom',
  grantMode: 'manual',
  name: '',
  description: '',
  iconType: 'asset',
  iconKey: '',
  iconUrl: '/static/badges/contributor.svg',
  color: 'blue',
  level: 'bronze',
  isEnabled: true,
  isWearable: true,
  sortOrder: 1000,
}

const colorOptions = [
  'blue',
  'emerald',
  'teal',
  'sky',
  'cyan',
  'rose',
  'violet',
  'purple',
  'fuchsia',
  'indigo',
  'amber',
  'orange',
  'yellow',
  'slate',
]

const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const error = ref('')
const badges = ref<AdminBadge[]>([])
const editing = ref<AdminBadge | null>(null)
const deletingBadge = ref<AdminBadge | null>(null)
const form = reactive<AdminBadge>({ ...emptyBadge })

const stats = computed(() => ({
  system: badges.value.filter(item => item.type === 'system').length,
  custom: badges.value.filter(item => item.type === 'custom').length,
}))

function toneClass(badge: AdminBadge) {
  const color = colorOptions.includes(badge.color) ? badge.color : 'blue'
  const classes: Record<string, string> = {
    blue: 'bg-blue-100 text-blue-700 ring-blue-200 dark:bg-blue-500/15 dark:text-blue-300 dark:ring-blue-500/30',
    emerald: 'bg-emerald-100 text-emerald-700 ring-emerald-200 dark:bg-emerald-500/15 dark:text-emerald-300 dark:ring-emerald-500/30',
    teal: 'bg-teal-100 text-teal-700 ring-teal-200 dark:bg-teal-500/15 dark:text-teal-300 dark:ring-teal-500/30',
    sky: 'bg-sky-100 text-sky-700 ring-sky-200 dark:bg-sky-500/15 dark:text-sky-300 dark:ring-sky-500/30',
    cyan: 'bg-cyan-100 text-cyan-700 ring-cyan-200 dark:bg-cyan-500/15 dark:text-cyan-300 dark:ring-cyan-500/30',
    rose: 'bg-rose-100 text-rose-700 ring-rose-200 dark:bg-rose-500/15 dark:text-rose-300 dark:ring-rose-500/30',
    violet: 'bg-violet-100 text-violet-700 ring-violet-200 dark:bg-violet-500/15 dark:text-violet-300 dark:ring-violet-500/30',
    purple: 'bg-purple-100 text-purple-700 ring-purple-200 dark:bg-purple-500/15 dark:text-purple-300 dark:ring-purple-500/30',
    fuchsia: 'bg-fuchsia-100 text-fuchsia-700 ring-fuchsia-200 dark:bg-fuchsia-500/15 dark:text-fuchsia-300 dark:ring-fuchsia-500/30',
    indigo: 'bg-indigo-100 text-indigo-700 ring-indigo-200 dark:bg-indigo-500/15 dark:text-indigo-300 dark:ring-indigo-500/30',
    amber: 'bg-amber-100 text-amber-700 ring-amber-200 dark:bg-amber-500/15 dark:text-amber-300 dark:ring-amber-500/30',
    orange: 'bg-orange-100 text-orange-700 ring-orange-200 dark:bg-orange-500/15 dark:text-orange-300 dark:ring-orange-500/30',
    yellow: 'bg-yellow-100 text-yellow-700 ring-yellow-200 dark:bg-yellow-500/15 dark:text-yellow-300 dark:ring-yellow-500/30',
    slate: 'bg-slate-100 text-slate-700 ring-slate-200 dark:bg-slate-500/15 dark:text-slate-300 dark:ring-slate-500/30',
  }
  return classes[color]
}

async function loadBadges() {
  loading.value = true
  error.value = ''
  try {
    badges.value = await getBadges()
  } catch (err) {
    error.value = err instanceof Error ? err.message : adminText('k0034')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { ...emptyBadge })
  editing.value = { ...emptyBadge }
}

function openEdit(badge: AdminBadge) {
  Object.assign(form, { ...emptyBadge, ...badge })
  editing.value = badge
}

async function submitBadge() {
  if (!form.name.trim()) {
    adminToast.warning(adminText('k0035'))
    return
  }
  saving.value = true
  try {
    await saveBadge({
      ...form,
      name: form.name.trim(),
      code: form.code.trim(),
      type: form.type || 'custom',
      grantMode: form.grantMode || 'manual',
      iconType: form.iconType || 'asset',
      color: colorOptions.includes(form.color) ? form.color : 'blue',
      sortOrder: Number(form.sortOrder || 1000),
    })
    editing.value = null
    await loadBadges()
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k001d'))
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deletingBadge.value) return
  deleting.value = true
  try {
    await deleteBadge(deletingBadge.value.code)
    deletingBadge.value = null
    await loadBadges()
    adminToast.success(adminText('k002u'))
  } catch (err) {
    adminToast.error(err, adminText('k001e'))
  } finally {
    deleting.value = false
  }
}

onMounted(() => {
  void loadBadges()
})
</script>

<template>
  <BasicPage :title="adminText('k0058')" :description="adminText('k0059')" sticky>
    <template #actions>
      <div class="flex items-center gap-2">
        <Button variant="outline" type="button" @click="loadBadges">
          <RefreshCw class="size-4" />
          {{ adminText('k004q') }}
        </Button>
        <Button type="button" @click="openCreate">
          <Plus class="size-4" />
          {{ adminText('k005a') }}
        </Button>
      </div>
    </template>

      <div class="mb-3 flex flex-wrap gap-2 text-sm text-muted-foreground">
        <Badge variant="secondary">{{ adminText('k00ba') }} {{ stats.system }}</Badge>
        <Badge variant="outline">{{ adminText('k002n') }} {{ stats.custom }}</Badge>
        <span v-if="loading">{{ adminText('k0046') }}</span>
      </div>

      <div v-if="error" class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">{{ error }}</div>
      <div v-else class="grid grid-cols-[repeat(auto-fill,minmax(5.5rem,1fr))] gap-3">
        <div v-for="badge in badges" :key="badge.code" class="group relative flex min-w-0 flex-col items-center rounded-md px-1 py-1.5 transition-colors hover:bg-muted/60">
          <button type="button" class="flex min-w-0 flex-col items-center" :title="`${badge.name} · ${badge.code}`" @click="openEdit(badge)">
            <div class="flex size-12 shrink-0 items-center justify-center ring-1 ring-inset transition-transform group-hover:scale-105" :class="toneClass(badge)" style="clip-path: polygon(25% 5%, 75% 5%, 100% 50%, 75% 95%, 25% 95%, 0 50%)">
              <img :src="badge.iconUrl || '/static/badges/contributor.svg'" :alt="badge.name" class="size-6 object-contain" />
            </div>
            <div class="mt-1 flex max-w-full items-center gap-1">
              <span class="truncate text-xs font-semibold leading-5" :class="badge.type === 'custom' ? 'text-blue-600 dark:text-blue-400' : 'text-foreground'">
                {{ badge.name }}
              </span>
            </div>
            <div class="max-w-full truncate text-[10px] leading-4 text-muted-foreground">
              {{ badge.grantMode === 'auto' ? adminText('k005b') : adminText('k005c') }} · {{ badge.isEnabled ? badge.level || 'bronze' : adminText('k005d') }}
            </div>
          </button>
          <div class="absolute right-0.5 top-0.5 flex gap-0.5 rounded-md bg-background/90 p-0.5 opacity-0 shadow-sm ring-1 ring-border transition-opacity group-hover:opacity-100">
            <AdminActionButton compact :title="adminText('k005j')" @click="openEdit(badge)">
              <Edit3 class="size-3.5" />
            </AdminActionButton>
            <AdminActionButton v-if="badge.type !== 'system'" compact tone="danger" :title="adminText('k005i')" @click="deletingBadge = badge">
              <Trash2 class="size-3.5" />
            </AdminActionButton>
          </div>
        </div>
      </div>

      <Dialog :open="editing !== null" @update:open="(open) => !open && (editing = null)">
        <DialogContent class="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{{ form.code ? adminText('k005e') : adminText('k005a') }}</DialogTitle>
            <DialogDescription>{{ adminText('k00bb') }}</DialogDescription>
          </DialogHeader>
          <form class="grid max-h-[68vh] gap-4 overflow-y-auto pr-2 sm:grid-cols-2" @submit.prevent="submitBadge">
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('k00bc') }}
              <Input v-model="form.code" :disabled="Boolean(editing?.code)" :placeholder="adminText('k005k')" />
            </label>
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('k00af') }}
              <Input v-model="form.name" />
            </label>
            <label class="grid gap-2 text-sm font-medium sm:col-span-2">
              {{ adminText('k00ag') }}
              <Textarea v-model="form.description" />
            </label>
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('k00bd') }}
              <Input v-model="form.iconUrl" />
            </label>
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('k00ad') }}
              <AdminSelect v-model="form.color" :options="colorOptions.map((color) => ({ value: color, label: color }))" class="w-full" />
            </label>
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('k00be') }}
              <Input v-model="form.level" />
            </label>
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('k00bf') }}
              <Input v-model.number="form.sortOrder" type="number" />
            </label>
            <label class="grid gap-2 text-sm font-medium">
              {{ adminText('k00bg') }}
              <AdminSelect
                v-model="form.grantMode"
                :disabled="form.type === 'system'"
                :options="[{ value: 'auto', label: adminText('k005b') }, { value: 'manual', label: adminText('k005c') }]"
                class="w-full"
              />
            </label>
            <div class="grid gap-2 text-sm font-medium">
              {{ adminText('k00bh') }}
              <div class="flex h-9 items-center justify-between rounded-md border bg-background px-3">
                <span class="text-sm text-muted-foreground">{{ adminText('k00bi') }}</span>
                <Switch v-model="form.isEnabled" />
              </div>
            </div>
            <div class="grid gap-2 text-sm font-medium">
              {{ adminText('k00d7') }}
              <div class="flex h-9 items-center justify-between rounded-md border bg-background px-3">
                <span class="text-sm text-muted-foreground">{{ adminText('k00d8') }}</span>
                <Switch v-model="form.isWearable" />
              </div>
            </div>
            <DialogFooter class="sm:col-span-2">
              <Button variant="outline" type="button" @click="editing = null">{{ adminText('k009q') }}</Button>
              <Button type="submit" :disabled="saving">{{ saving ? adminText('k005f') : adminText('k005g') }}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AdminConfirmDialog
        :open="deletingBadge !== null"
        :title="adminText('k00bj')"
        :description="`${adminText('k00bk')}${deletingBadge?.name || ''}${adminText('k00bl')}`"
        :loading="deleting"
        @update:open="(open) => !open && (deletingBadge = null)"
        @confirm="confirmDelete"
      />
    </BasicPage>
</template>
