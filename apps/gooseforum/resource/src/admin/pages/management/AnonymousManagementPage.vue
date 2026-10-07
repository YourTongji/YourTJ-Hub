<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ChevronLeft,
  ChevronRight,
  RefreshCw,
  Search,
  Shield,
  ShieldOff,
} from '@lucide/vue'
import { BasicPage } from '@/admin/components/global-layout'
import AdminSection from '@/admin/components/AdminSection.vue'
import AdminToolbar from '@/admin/components/AdminToolbar.vue'
import { Button } from '@/admin/components/ui/button'
import { Badge } from '@/admin/components/ui/badge'
import { Input } from '@/admin/components/ui/input'
import { Textarea } from '@/admin/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/admin/components/ui/dialog'
import {
  listAnonymousIdentities,
  governAnonymousIdentity,
  type AdminAnonymousIdentity,
} from '@/admin/runtime/api'
import type { AdminPayload } from '@/admin/types'

const props = defineProps<{ payload?: AdminPayload }>()

const { t, locale } = useI18n()
const reason = ref('')
const activeReason = ref('')
const search = ref('')
const appliedSearch = ref('')
const status = ref<'all' | 'active' | 'disabled' | 'banned'>('all')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const rows = ref<AdminAnonymousIdentity[]>([])
const loading = ref(false)
const error = ref('')
const selected = ref<AdminAnonymousIdentity>()
const actionReason = ref('')
const saving = ref(false)
const actionError = ref('')
let request = 0
let session = 0
const pages = computed(() =>
  Math.max(1, Math.ceil(total.value / pageSize.value)),
)
const date = (value: string) => new Date(value).toLocaleDateString(locale.value)
const state = (row: AdminAnonymousIdentity) =>
  row.governanceDisabled ? 'banned' : row.disabled ? 'disabled' : 'active'

async function load() {
  if (!activeReason.value) return
  const id = ++request
  loading.value = true
  error.value = ''
  // Never retain a previously revealed page after a denied or failed request.
  rows.value = []
  total.value = 0
  try {
    const result = await listAnonymousIdentities({
      page: page.value,
      pageSize: pageSize.value,
      search: appliedSearch.value,
      status: status.value,
      reason: activeReason.value,
    })
    if (id !== request) return
    rows.value = result.items
    total.value = result.total
  } catch (e) {
    if (id === request)
      error.value = e instanceof Error ? e.message : t('common.loadFailed')
  } finally {
    if (id === request) loading.value = false
  }
}
function reveal() {
  if (!reason.value.trim()) return
  activeReason.value = reason.value.trim()
  page.value = 1
  void load()
}
function filter() {
  appliedSearch.value = search.value.trim()
  page.value = 1
  void load()
}
function changePage(next: number) {
  page.value = next
  void load()
}
function lock() {
  ++request
  ++session
  rows.value = []
  total.value = 0
  activeReason.value = ''
  reason.value = ''
  loading.value = false
  saving.value = false
  selected.value = undefined
  actionReason.value = ''
  actionError.value = ''
  error.value = ''
  search.value = ''
  appliedSearch.value = ''
  status.value = 'all'
  page.value = 1
}
function choose(row: AdminAnonymousIdentity) {
  selected.value = row
  actionReason.value = ''
  actionError.value = ''
}
async function govern() {
  if (!selected.value || !actionReason.value.trim() || saving.value) return
  saving.value = true
  actionError.value = ''
  const epoch = session
  try {
    await governAnonymousIdentity({
      publicUid: selected.value.publicUid,
      disabled: !selected.value.governanceDisabled,
      reason: actionReason.value.trim(),
    })
    if (epoch !== session) return
    selected.value = undefined
    await load()
  } catch (e) {
    if (epoch !== session) return
    actionError.value =
      e instanceof Error ? e.message : t('anonymousAdmin.saveFailed')
  } finally {
    if (epoch === session) saving.value = false
  }
}
watch(
  () => [
    props.payload?.layout.viewer.id,
    props.payload?.layout.viewer.isAuthenticated,
    props.payload?.layout.viewer.adminPermissions?.join(','),
  ],
  lock,
)
onMounted(() => window.addEventListener('goose:session-cleared', lock))
onBeforeUnmount(() => {
  window.removeEventListener('goose:session-cleared', lock)
  lock()
})
</script>

<template>
  <BasicPage
    :title="t('anonymousAdmin.title')"
    :description="t('anonymousAdmin.description')"
  >
    <template #actions>
      <Button
        v-if="activeReason"
        variant="outline"
        :disabled="loading"
        @click="load"
        ><RefreshCw class="size-4" />{{ t('feed.refresh') }}</Button
      >
    </template>
    <div class="space-y-4">
      <AdminSection body-class="p-4 sm:p-5">
        <form v-if="!activeReason" class="space-y-3" @submit.prevent="reveal">
          <div class="flex items-start gap-3">
            <Shield class="mt-0.5 size-5 shrink-0 text-muted-foreground" />
            <div>
              <h2 class="font-medium">
                {{ t('anonymousAdmin.privateTitle') }}
              </h2>
              <p class="mt-1 text-sm text-muted-foreground">
                {{ t('anonymousAdmin.privateDescription') }}
              </p>
            </div>
          </div>
          <label
            for="anonymous-view-reason"
            class="block text-sm font-medium"
            >{{ t('anonymousAdmin.reason') }}</label
          >
          <div class="flex flex-col gap-2 sm:flex-row">
            <Input
              id="anonymous-view-reason"
              v-model="reason"
              :placeholder="t('anonymousAdmin.reasonPlaceholder')"
              maxlength="512"
            /><Button type="submit" :disabled="!reason.trim()">{{
              t('anonymousAdmin.view')
            }}</Button>
          </div>
        </form>
        <div
          v-else
          class="flex flex-wrap items-center justify-between gap-2 text-sm"
        >
          <p class="min-w-0 break-words text-muted-foreground">
            <Shield class="mr-1 inline size-4" />{{
              t('anonymousAdmin.reason')
            }}
            · {{ activeReason }}
          </p>
          <Button variant="ghost" size="sm" @click="lock">{{
            t('anonymousAdmin.hide')
          }}</Button>
        </div>
      </AdminSection>

      <AdminSection v-if="activeReason">
        <template #header>
          <AdminToolbar>
            <form class="flex w-full flex-wrap gap-2" @submit.prevent="filter">
              <div
                class="relative min-w-0 basis-full sm:min-w-40 sm:basis-auto sm:flex-1"
              >
                <Search
                  class="absolute left-3 top-2.5 size-4 text-muted-foreground"
                /><Input
                  v-model="search"
                  class="pl-9"
                  :placeholder="t('anonymousAdmin.searchPlaceholder')"
                  :aria-label="t('anonymousAdmin.searchPlaceholder')"
                  maxlength="128"
                />
              </div>
              <select
                v-model="status"
                class="h-9 flex-1 rounded-md border bg-background px-3 text-sm sm:flex-none"
                :aria-label="t('anonymousAdmin.status')"
                :disabled="loading"
                @change="filter"
              >
                <option
                  v-for="value in ['all', 'active', 'disabled', 'banned']"
                  :key="value"
                  :value="value"
                >
                  {{ t(`anonymousAdmin.${value}`) }}
                </option>
              </select>
              <Button type="submit" variant="outline" :disabled="loading">{{
                t('common.search')
              }}</Button>
            </form>
          </AdminToolbar>
        </template>
        <div
          v-if="loading"
          class="p-12 text-center text-sm text-muted-foreground"
          role="status"
        >
          {{ t('common.loading') }}
        </div>
        <div v-else-if="error" class="space-y-3 p-8 text-center">
          <p role="alert" class="text-sm text-destructive">{{ error }}</p>
          <Button variant="outline" @click="load">{{
            t('common.retry')
          }}</Button>
        </div>
        <div
          v-else-if="!rows.length"
          class="p-12 text-center text-sm text-muted-foreground"
        >
          {{ t('anonymousAdmin.empty') }}
        </div>
        <template v-else>
          <div class="hidden overflow-x-auto md:block">
            <table class="w-full text-sm">
              <thead
                class="border-b bg-muted/45 text-left text-xs text-muted-foreground"
              >
                <tr>
                  <th
                    v-for="key in [
                      'identity',
                      'owner',
                      'status',
                      'selectedAt',
                      'actions',
                    ]"
                    :key="key"
                    class="px-5 py-3"
                  >
                    {{ t(`anonymousAdmin.${key}`) }}
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y">
                <tr v-for="row in rows" :key="row.publicUid">
                  <td class="px-5 py-4">
                    <div class="flex items-center gap-3">
                      <img
                        :src="row.avatarUrl"
                        :alt="row.name"
                        class="size-10 rounded-full border"
                      />
                      <div>
                        <a
                          :href="row.profileUrl"
                          target="_blank"
                          rel="noopener noreferrer"
                          class="font-medium hover:underline"
                          >{{ row.name }}</a
                        >
                        <p class="mt-1 font-mono text-xs text-muted-foreground">
                          {{ row.publicUid }}
                        </p>
                      </div>
                    </div>
                  </td>
                  <td class="px-5 py-4">
                    <a
                      v-if="!row.owner.closed"
                      :href="`/u/${row.owner.userId}`"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="font-medium hover:underline"
                      >{{ row.owner.username }}</a
                    ><span v-else>{{ row.owner.username }}</span>
                    <p class="mt-1 text-xs text-muted-foreground">
                      ID {{ row.owner.userId
                      }}<span v-if="row.owner.closed">
                        · {{ t('anonymousAdmin.closed') }}</span
                      ><span v-else-if="row.owner.frozen">
                        · {{ t('anonymousAdmin.ownerFrozen') }}</span
                      >
                    </p>
                  </td>
                  <td class="px-5 py-4">
                    <Badge
                      :variant="
                        row.governanceDisabled ? 'destructive' : 'secondary'
                      "
                      >{{ t(`anonymousAdmin.${state(row)}`) }}</Badge
                    >
                  </td>
                  <td class="px-5 py-4 text-muted-foreground">
                    {{ date(row.selectedAt) }}
                  </td>
                  <td class="px-5 py-4">
                    <Button
                      :variant="row.governanceDisabled ? 'outline' : 'ghost'"
                      size="sm"
                      :class="{ 'text-destructive': !row.governanceDisabled }"
                      @click="choose(row)"
                      ><ShieldOff class="size-4" />{{
                        t(
                          row.governanceDisabled
                            ? 'anonymousAdmin.restore'
                            : 'anonymousAdmin.ban',
                        )
                      }}</Button
                    >
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="divide-y md:hidden">
            <article
              v-for="row in rows"
              :key="row.publicUid"
              class="space-y-3 p-4"
            >
              <div class="flex items-center gap-3">
                <img
                  :src="row.avatarUrl"
                  :alt="row.name"
                  class="size-10 rounded-full border"
                /><a
                  :href="row.profileUrl"
                  class="min-w-0 flex-1 font-medium"
                  >{{ row.name }}</a
                ><Badge
                  :variant="
                    row.governanceDisabled ? 'destructive' : 'secondary'
                  "
                  >{{ t(`anonymousAdmin.${state(row)}`) }}</Badge
                >
              </div>
              <p class="break-all font-mono text-xs text-muted-foreground">
                {{ row.publicUid }}
              </p>
              <div class="flex items-center justify-between gap-2">
                <div class="min-w-0 text-sm">
                  <p class="break-words">
                    {{ t('anonymousAdmin.owner') }}：{{ row.owner.username }}
                  </p>
                  <p class="text-xs text-muted-foreground">
                    ID {{ row.owner.userId }} · {{ date(row.selectedAt)
                    }}<span v-if="row.owner.closed">
                      · {{ t('anonymousAdmin.closed') }}</span
                    ><span v-else-if="row.owner.frozen">
                      · {{ t('anonymousAdmin.ownerFrozen') }}</span
                    >
                  </p>
                </div>
                <Button variant="outline" size="sm" @click="choose(row)">{{
                  t(
                    row.governanceDisabled
                      ? 'anonymousAdmin.restore'
                      : 'anonymousAdmin.ban',
                  )
                }}</Button>
              </div>
            </article>
          </div>
        </template>
        <div
          class="flex flex-wrap items-center justify-between gap-3 border-t bg-muted/10 px-4 py-3 text-sm text-muted-foreground"
        >
          <span>{{ t('anonymousAdmin.total', { count: total }) }}</span>
          <div class="flex items-center gap-2">
            <select
              v-model="pageSize"
              :disabled="loading"
              :aria-label="t('anonymousAdmin.pageSize')"
              class="h-9 rounded-md border bg-background px-2"
              @change="changePage(1)"
            >
              <option v-for="size in [10, 20, 50]" :key="size" :value="size">
                {{ size }} / {{ t('anonymousAdmin.page') }}
              </option></select
            ><Button
              variant="outline"
              size="icon"
              :aria-label="t('common.previousPage')"
              :disabled="loading || page <= 1"
              @click="changePage(page - 1)"
              ><ChevronLeft class="size-4" /></Button
            ><span>{{ page }} / {{ pages }}</span
            ><Button
              variant="outline"
              size="icon"
              :aria-label="t('common.nextPage')"
              :disabled="loading || page >= pages"
              @click="changePage(page + 1)"
              ><ChevronRight class="size-4"
            /></Button>
          </div>
        </div>
      </AdminSection>
    </div>
  </BasicPage>
  <Dialog
    :open="!!selected"
    @update:open="
      (value) => {
        if (!value && !saving) selected = undefined
      }
    "
  >
    <DialogContent :show-close-button="!saving">
      <DialogHeader
        ><DialogTitle
          >{{
            t(
              selected?.governanceDisabled
                ? 'anonymousAdmin.restore'
                : 'anonymousAdmin.ban',
            )
          }}
          · {{ selected?.name }}</DialogTitle
        ><DialogDescription>{{
          t('anonymousAdmin.governanceHint')
        }}</DialogDescription></DialogHeader
      >
      <form class="space-y-4" @submit.prevent="govern">
        <label
          for="anonymous-action-reason"
          class="block text-sm font-medium"
          >{{ t('anonymousAdmin.actionReason') }}</label
        ><Textarea
          id="anonymous-action-reason"
          v-model="actionReason"
          maxlength="512"
          :disabled="saving"
          :placeholder="t('anonymousAdmin.actionReasonPlaceholder')"
        />
        <p v-if="actionError" role="alert" class="text-sm text-destructive">
          {{ actionError }}
        </p>
        <div class="flex justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            :disabled="saving"
            @click="selected = undefined"
            >{{ t('common.cancel') }}</Button
          ><Button
            type="submit"
            :variant="selected?.governanceDisabled ? 'default' : 'destructive'"
            :disabled="saving || !actionReason.trim()"
            >{{
              t(saving ? 'common.loading' : 'anonymousAdmin.confirm')
            }}</Button
          >
        </div>
      </form>
    </DialogContent>
  </Dialog>
</template>
