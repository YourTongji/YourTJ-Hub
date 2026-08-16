<script setup lang="ts">
import { adminText } from '@/admin/runtime/i18n-text'

import { computed, onMounted, ref } from 'vue'
import {
  BookOpen,
  Clock,
  ExternalLink,
  FileText,
  Folder,
  History,
  KeyRound,
  RefreshCw,
} from '@lucide/vue'
import AdminSection from '@/admin/components/AdminSection.vue'
import AdminToolbar from '@/admin/components/AdminToolbar.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/admin/components/ui/table'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/admin/components/ui/tabs'
import {
  getWikiNamespaces,
  getWikiSyncRuns,
  getWikiSyncStatus,
  getWikiTree,
  getWikiWebhookSecret,
  saveWikiWebhookSecret,
  triggerWikiSync,
  type WikiSyncRunView,
  type WikiSyncStatus,
} from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type {
  AdminPayload,
  ManageHomeProps,
  WikiNamespace,
  WikiNamespaceTree,
  WikiPageNode,
} from '@/admin/types'
import { flattenAdminTree } from '@/admin/utils/wiki-tree'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

const activeTab = ref('namespaces')

const namespaces = ref<WikiNamespace[]>([])
const nsLoading = ref(false)
const nsError = ref('')

const tree = ref<WikiNamespaceTree[]>([])
const treeLoading = ref(false)
const treeError = ref('')

const syncStatus = ref<WikiSyncStatus | null>(null)
const syncLoading = ref(false)
const syncError = ref('')
const syncRuns = ref<WikiSyncRunView[]>([])
const runsLoading = ref(false)
const syncTriggering = ref(false)

// webhook 验签密钥（securestore 加密落库，仅回显是否已配置）
const webhookConfigured = ref(false)
const webhookSecretInput = ref('')
const webhookSaving = ref(false)
const webhookLoading = ref(false)

const globalLoading = computed(() => nsLoading.value || treeLoading.value || syncLoading.value || runsLoading.value || webhookLoading.value)

function formatTime(value: string | undefined | null) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function shortSha(sha: string) {
  return sha ? sha.slice(0, 7) : '-'
}

function triggerLabel(trigger: string) {
  if (trigger === 'manual') return adminText('k00qc')
  if (trigger === 'webhook') return adminText('k00qu')
  if (trigger === 'schedule') return adminText('k00qd')
  if (trigger === 'startup') return adminText('k00qy')
  return trigger
}

function statusLabel(status: string) {
  if (status === 'success') return adminText('k00qg')
  if (status === 'failed') return adminText('k00qh')
  if (status === 'running') return adminText('k00qi')
  return status
}

function statusTone(status: string) {
  // Badge 无 success variant：成功用 secondary（灰），失败用 destructive，运行中 default。
  if (status === 'success') return 'secondary'
  if (status === 'failed') return 'destructive'
  return 'default'
}

// 从同步状态配置推导 GitHub 编辑外链基址（与后端 GitConfig.RepoPath 同逻辑）。
function repoEditBase() {
  const repo = syncStatus.value?.repo || ''
  let path = repo.replace(/\.git$/, '').replace(/\/+$/, '')
  for (const prefix of ['https://github.com/', 'http://github.com/', 'git@github.com:']) {
    if (path.startsWith(prefix)) {
      path = path.slice(prefix.length)
      break
    }
  }
  return path.includes('/') ? path : ''
}

function editUrlFor(page: WikiPageNode) {
  const base = repoEditBase()
  if (!base) return ''
  const branch = syncStatus.value?.branch || 'main'
  // D7：GitHub 外链必须用仓库真实路径 sourcePath（path 首段已是 slug，
  // 与仓库目录名解耦）。逐段转义：目录名/文件名允许 #/% 等字符，但 URL
  // 拼接时 # 会开启 fragment、% 会被当转义前缀 → GitHub 404。
  const repoPath = (page.sourcePath || page.path)
    .split('/')
    .map((seg) => encodeURIComponent(seg))
    .join('/')
  return `https://github.com/${base}/edit/${branch}/${repoPath}.md`
}

// ---------- Namespaces（只读：GitHub SSOT，命名空间由仓库顶层目录驱动） ----------
async function loadNamespaces() {
  nsLoading.value = true
  nsError.value = ''
  try {
    namespaces.value = await getWikiNamespaces()
  } catch (err) {
    nsError.value = err instanceof Error ? err.message : adminText('k00n0')
  } finally {
    nsLoading.value = false
  }
}

// ---------- Page tree（只读：GitHub SSOT，结构由仓库决定） ----------
// 嵌套树扁平化见 @/admin/utils/wiki-tree（层级以缩进呈现，issue #289）。

async function loadTree() {
  treeLoading.value = true
  treeError.value = ''
  try {
    tree.value = await getWikiTree()
  } catch (err) {
    treeError.value = err instanceof Error ? err.message : adminText('k00n2')
  } finally {
    treeLoading.value = false
  }
}


// ---------- GitHub 同步面板 ----------
async function loadSyncStatus() {
  syncLoading.value = true
  syncError.value = ''
  try {
    syncStatus.value = await getWikiSyncStatus()
  } catch (err) {
    syncError.value = err instanceof Error ? err.message : adminText('k00n0')
  } finally {
    syncLoading.value = false
  }
}

async function loadSyncRuns() {
  runsLoading.value = true
  try {
    syncRuns.value = await getWikiSyncRuns()
  } catch (err) {
    adminToast.error(err, adminText('k00n0'))
  } finally {
    runsLoading.value = false
  }
}

async function runSync() {
  // 未启用/加载中/运行中时按钮已禁用；此处兜底防直接调用。
  if (syncTriggering.value || syncLoading.value || !syncStatus.value?.enabled) return
  syncTriggering.value = true
  try {
    await triggerWikiSync()
    adminToast.success(adminText('k00qw'))
  } catch (err) {
    const messageCode = (err as { messageCode?: string } | null)?.messageCode
    if (messageCode === 'wiki.sync.running') {
      adminToast.warning(adminText('k00qk'))
    } else {
      adminToast.error(err, adminText('k00ql'))
    }
  } finally {
    syncTriggering.value = false
    await Promise.all([loadSyncStatus(), loadSyncRuns(), loadTree()])
  }
}

// ---------- webhook 验签密钥（securestore 加密落库，仅回显是否已配置） ----------
async function loadWebhookSecret() {
  webhookLoading.value = true
  try {
    const status = await getWikiWebhookSecret()
    webhookConfigured.value = status.configured
  } catch (err) {
    adminToast.error(err, adminText('k00n0'))
  } finally {
    webhookLoading.value = false
  }
}

async function saveWebhookSecret() {
  if (webhookSaving.value) return
  webhookSaving.value = true
  try {
    await saveWikiWebhookSecret(webhookSecretInput.value.trim())
    webhookSecretInput.value = ''
    await loadWebhookSecret()
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k00n0'))
  } finally {
    webhookSaving.value = false
  }
}

async function loadAll() {
  await Promise.all([loadNamespaces(), loadTree(), loadSyncStatus(), loadSyncRuns(), loadWebhookSecret()])
}

onMounted(() => {
  void loadAll()
})
</script>

<template>
  <BasicPage :title="adminText('k00n4')" :description="adminText('k00n5')" sticky>
    <template #actions>
      <Button variant="outline" size="sm" type="button" :disabled="globalLoading" @click="loadAll">
        <RefreshCw class="size-4" :class="globalLoading ? 'animate-spin' : ''" />
        {{ adminText('k004q') }}
      </Button>
    </template>

    <Tabs v-model="activeTab" class="gap-5">
      <TabsList class="w-fit">
        <TabsTrigger value="namespaces">{{ adminText('k00n6') }}</TabsTrigger>
        <TabsTrigger value="pages">{{ adminText('k00n8') }}</TabsTrigger>
        <TabsTrigger value="sync">{{ adminText('k00q0') }}</TabsTrigger>
      </TabsList>

      <TabsContent value="namespaces">
        <AdminSection>
          <template #header>
            <AdminToolbar class="border-b-0">
              <div class="flex items-center justify-between gap-3">
                <Badge variant="secondary" class="h-9 rounded-md px-3">
                  {{ namespaces.length }} {{ adminText('k00n6') }}
                </Badge>
                <span class="text-sm text-muted-foreground">{{ adminText('k00qz') }}</span>
              </div>
            </AdminToolbar>
          </template>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ adminText('k00af') }}</TableHead>
                <TableHead>{{ adminText('k00r7') }}</TableHead>
                <TableHead>{{ adminText('k00ag') }}</TableHead>
                <TableHead class="w-24">{{ adminText('k00na') }}</TableHead>
                <TableHead class="w-40">{{ adminText('k00nb') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="nsLoading && namespaces.length === 0">
                <TableCell colspan="5" class="h-28 text-center text-muted-foreground">{{ adminText('k0046') }}</TableCell>
              </TableRow>
              <TableRow v-else-if="nsError">
                <TableCell colspan="5" class="h-28 text-center text-destructive">{{ nsError }}</TableCell>
              </TableRow>
              <TableRow v-else-if="namespaces.length === 0">
                <TableCell colspan="5" class="h-28 text-center text-muted-foreground">{{ adminText('k00nj') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-for="item in namespaces" :key="item.name">
                  <TableCell class="font-medium">{{ item.name }}</TableCell>
                  <TableCell class="font-mono text-xs text-muted-foreground">{{ item.slug || '-' }}</TableCell>
                  <TableCell class="max-w-md truncate text-muted-foreground">{{ item.description || '-' }}</TableCell>
                  <TableCell>{{ item.pageCount ?? 0 }}</TableCell>
                  <TableCell class="text-xs text-muted-foreground">{{ formatTime(item.updatedAt) }}</TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
        </AdminSection>
      </TabsContent>

      <TabsContent value="pages">
        <AdminSection>
          <template #header>
            <AdminToolbar class="border-b-0">
              <Badge variant="secondary" class="h-9 rounded-md px-3">
                {{ tree.length }} {{ adminText('k00n6') }}
              </Badge>
              <span class="text-sm text-muted-foreground">{{ adminText('k00q2') }}</span>
            </AdminToolbar>
          </template>
          <div class="grid gap-3 p-4">
            <div v-if="treeLoading && tree.length === 0" class="py-10 text-center text-sm text-muted-foreground">{{ adminText('k0046') }}</div>
            <div v-else-if="treeError" class="py-10 text-center text-sm text-destructive">{{ treeError }}</div>
            <div v-else-if="tree.length === 0" class="py-10 text-center text-sm text-muted-foreground">{{ adminText('k00ny') }}</div>
            <template v-else>
              <div v-for="group in tree" :key="group.name" class="overflow-hidden rounded-lg border">
                <div class="flex items-center justify-between gap-3 border-b bg-muted/20 px-3 py-2">
                  <div class="flex min-w-0 items-center gap-2">
                    <BookOpen class="size-4 shrink-0 text-muted-foreground" />
                    <span class="truncate text-sm font-medium">{{ group.label || group.name }}</span>
                    <span class="shrink-0 text-xs text-muted-foreground">{{ group.pages.length }}</span>
                  </div>
                </div>
                <div v-if="!group.pages.length" class="px-4 py-6 text-center text-sm text-muted-foreground">{{ adminText('k00ny') }}</div>
                <div v-else class="divide-y">
                  <div
                    v-for="page in flattenAdminTree(group.pages)"
                    :key="page.path"
                    class="flex items-center gap-2 px-3 py-2"
                    :style="{ paddingLeft: `${16 + page.depth * 18}px` }"
                  >
                    <Folder v-if="page.pageId === 0" class="size-4 shrink-0 text-muted-foreground" />
                    <FileText v-else class="size-4 shrink-0 text-muted-foreground" />
                    <div class="min-w-0 flex-1">
                      <div class="truncate text-sm font-medium">{{ page.title || page.path }}</div>
                      <div class="truncate font-mono text-xs text-muted-foreground">{{ page.path }}</div>
                    </div>
                    <div class="flex shrink-0 items-center gap-1">
                      <a
                        v-if="editUrlFor(page) && page.pageId !== 0"
                        :href="editUrlFor(page)"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="grid size-8 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                        :title="adminText('k00q3')"
                      >
                        <ExternalLink class="size-3.5" />
                      </a>
                      <a
                        v-if="page.pageId !== 0"
                        :href="`/wiki/${page.path.split('/').map((seg) => encodeURIComponent(seg)).join('/')}`"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="grid size-8 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                        :title="adminText('k00nv')"
                      >
                        <ExternalLink class="size-3.5" />
                      </a>
                    </div>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </AdminSection>
      </TabsContent>

      <TabsContent value="sync">
        <div class="grid gap-4">
          <AdminSection>
            <template #header>
              <AdminToolbar class="border-b-0">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div class="flex items-center gap-2">
                    <History class="size-4 shrink-0 text-muted-foreground" />
                    <span class="text-sm font-medium">{{ adminText('k00q0') }}</span>
                  </div>
                  <Button type="button" size="sm" :disabled="syncTriggering || syncLoading || !syncStatus?.enabled || syncStatus?.lastRun?.status === 'running'" @click="runSync">
                    <RefreshCw class="size-3.5" :class="syncTriggering ? 'animate-spin' : ''" />
                    {{ syncTriggering ? adminText('k00q8') : adminText('k00q7') }}
                  </Button>
                </div>
              </AdminToolbar>
            </template>
            <div v-if="syncLoading && !syncStatus" class="px-4 py-10 text-center text-sm text-muted-foreground">{{ adminText('k0046') }}</div>
            <div v-else-if="syncError" class="px-4 py-10 text-center text-sm text-destructive">{{ syncError }}</div>
            <template v-else-if="syncStatus">
              <div v-if="!syncStatus.enabled" class="px-4 py-8 text-center text-sm text-muted-foreground">{{ adminText('k00q6') }}</div>
              <div v-else class="grid gap-4 p-4 md:grid-cols-2">
                <div class="space-y-3">
                  <div class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
                    <span class="text-muted-foreground">{{ adminText('k00q9') }}</span>
                    <span class="truncate font-mono">{{ syncStatus.repo }}</span>
                    <span class="text-muted-foreground">{{ adminText('k00qa') }}</span>
                    <span class="font-mono">{{ syncStatus.branch }}</span>
                    <span class="text-muted-foreground">{{ adminText('k00qb') }}</span>
                    <span class="font-mono">{{ shortSha(syncStatus.headSha) }}</span>
                    <span class="text-muted-foreground">{{ adminText('k00qe') }}</span>
                    <span>{{ syncStatus.pages?.total ?? 0 }} / {{ syncStatus.pages?.namespaces ?? 0 }}</span>
                  </div>
                </div>
                <div class="space-y-2 text-sm">
                  <div class="text-muted-foreground">{{ adminText('k00qf') }}</div>
                  <div v-if="!syncStatus.lastRun" class="text-muted-foreground">{{ adminText('k00qm') }}</div>
                  <template v-else>
                    <div class="flex flex-wrap items-center gap-2">
                      <Badge :variant="statusTone(syncStatus.lastRun.status)">{{ statusLabel(syncStatus.lastRun.status) }}</Badge>
                      <span class="text-muted-foreground">{{ triggerLabel(syncStatus.lastRun.trigger) }}</span>
                      <span class="text-muted-foreground">{{ formatTime(syncStatus.lastRun.startedAt) }}</span>
                    </div>
                    <div class="text-xs text-muted-foreground">
                      +{{ syncStatus.lastRun.pagesAdded }} / ~{{ syncStatus.lastRun.pagesUpdated }} / -{{ syncStatus.lastRun.pagesDeleted }}
                    </div>
                    <div v-if="syncStatus.lastRun.error" class="text-xs text-destructive">{{ syncStatus.lastRun.error }}</div>
                  </template>
                </div>
              </div>
            </template>
          </AdminSection>

          <AdminSection>
            <template #header>
              <AdminToolbar class="border-b-0">
                <div class="flex items-center gap-2">
                  <KeyRound class="size-4 shrink-0 text-muted-foreground" />
                  <span class="text-sm font-medium">{{ adminText('k00r0') }}</span>
                </div>
              </AdminToolbar>
            </template>
            <div class="grid gap-4 p-4 md:grid-cols-2">
              <div class="space-y-2 text-sm">
                <div class="flex items-center gap-2">
                  <span class="text-muted-foreground">{{ adminText('k00r1') }}</span>
                  <Badge :variant="webhookConfigured ? 'secondary' : 'destructive'">
                    {{ webhookConfigured ? adminText('k00r2') : adminText('k00r3') }}
                  </Badge>
                </div>
                <p class="text-xs text-muted-foreground">{{ adminText('k00r4') }}</p>
              </div>
              <form class="flex items-end gap-2" @submit.prevent="saveWebhookSecret">
                <label class="grid flex-1 gap-2 text-sm font-medium">
                  {{ adminText('k00r5') }}
                  <Input
                    v-model="webhookSecretInput"
                    type="password"
                    autocomplete="new-password"
                    :placeholder="adminText('k00r6')"
                  />
                </label>
                <Button type="submit" size="sm" :disabled="webhookSaving || webhookLoading">
                  {{ webhookSaving ? adminText('k005f') : adminText('k005g') }}
                </Button>
              </form>
            </div>
          </AdminSection>

          <AdminSection>
            <template #header>
              <AdminToolbar class="border-b-0">
                <div class="flex items-center gap-2">
                  <Clock class="size-4 shrink-0 text-muted-foreground" />
                  <span class="text-sm font-medium">{{ adminText('k00qn') }}</span>
                </div>
              </AdminToolbar>
            </template>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead class="w-20">{{ adminText('k00qn') }}</TableHead>
                  <TableHead class="w-24">{{ adminText('k00qo') }}</TableHead>
                  <TableHead class="w-36">{{ adminText('k00qt') }}</TableHead>
                  <TableHead class="w-36">{{ adminText('k00qp') }}</TableHead>
                  <TableHead class="w-44">{{ adminText('k00qq') }}</TableHead>
                  <TableHead class="w-44">{{ adminText('k00qr') }}</TableHead>
                  <TableHead>{{ adminText('k00qs') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="runsLoading && syncRuns.length === 0">
                  <TableCell colspan="7" class="h-24 text-center text-muted-foreground">{{ adminText('k0046') }}</TableCell>
                </TableRow>
                <TableRow v-else-if="syncRuns.length === 0">
                  <TableCell colspan="7" class="h-24 text-center text-muted-foreground">{{ adminText('k00qm') }}</TableCell>
                </TableRow>
                <template v-else>
                  <TableRow v-for="run in syncRuns" :key="run.id">
                    <TableCell class="font-mono text-xs text-muted-foreground">#{{ run.id }}</TableCell>
                    <TableCell class="font-mono text-xs">{{ shortSha(run.headSha) }}</TableCell>
                    <TableCell>
                      <Badge :variant="statusTone(run.status)">{{ statusLabel(run.status) }}</Badge>
                    </TableCell>
                    <TableCell class="text-xs text-muted-foreground">{{ triggerLabel(run.trigger) }}</TableCell>
                    <TableCell class="text-xs text-muted-foreground">+{{ run.pagesAdded }} / ~{{ run.pagesUpdated }} / -{{ run.pagesDeleted }}</TableCell>
                    <TableCell class="text-xs text-muted-foreground">{{ formatTime(run.startedAt) }}</TableCell>
                    <TableCell class="max-w-xs truncate text-xs text-muted-foreground">{{ run.error || '-' }}</TableCell>
                  </TableRow>
                </template>
              </TableBody>
            </Table>
          </AdminSection>
        </div>
      </TabsContent>
    </Tabs>
  </BasicPage>
</template>
