<script setup lang="ts">
import { adminText } from '@/admin/runtime/i18n-text'

import { computed, onMounted, reactive, ref, watch } from 'vue'
import {
  BookOpen,
  Check,
  ChevronDown,
  ChevronUp,
  ExternalLink,
  Eye,
  FileText,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Trash2,
  UserPlus,
  X,
} from '@lucide/vue'
import AdminActionButton from '@/admin/components/AdminActionButton.vue'
import AdminConfirmDialog from '@/admin/components/AdminConfirmDialog.vue'
import AdminSection from '@/admin/components/AdminSection.vue'
import AdminToolbar from '@/admin/components/AdminToolbar.vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogScrollContent,
  DialogTitle,
} from '@/admin/components/ui/dialog'
import { Input } from '@/admin/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/admin/components/ui/select'
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
import { Textarea } from '@/admin/components/ui/textarea'
import {
  createWikiNamespace,
  createWikiPage,
  deleteWikiNamespace,
  getWikiEditors,
  getWikiNamespaces,
  getWikiRevisions,
  getWikiTree,
  getUserList,
  reviewWikiRevision,
  saveWikiEditors,
  saveWikiTree,
  updateWikiNamespace,
} from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type {
  AdminPayload,
  AdminUser,
  ManageHomeProps,
  WikiEditor,
  WikiNamespace,
  WikiNamespaceTree,
  WikiPageNode,
  WikiRevision,
} from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

const activeTab = ref('namespaces')

const namespaces = ref<WikiNamespace[]>([])
const nsLoading = ref(false)
const nsError = ref('')
const nsDialog = ref<{ mode: 'create' | 'edit', row: WikiNamespace | null } | null>(null)
const nsForm = reactive({ name: '', description: '' })
const nsSaving = ref(false)
const deletingNs = ref<WikiNamespace | null>(null)
const nsDeleting = ref(false)

const editorNamespace = ref('')
const editors = ref<WikiEditor[]>([])
const editorLoading = ref(false)
const editorSaving = ref(false)
const editorSearch = ref('')
const editorSearching = ref(false)
const editorCandidates = ref<AdminUser[]>([])
const selectedEditorUser = ref<AdminUser | null>(null)
let editorSearchTimer: ReturnType<typeof setTimeout> | undefined

const tree = ref<WikiNamespaceTree[]>([])
const treeLoading = ref(false)
const treeError = ref('')
const treeBusy = ref(false)
const renameRow = ref<{ group: WikiNamespaceTree, page: WikiPageNode } | null>(null)
const renameForm = reactive({ path: '', title: '' })
const renameSaving = ref(false)
const deletingPage = ref<{ group: WikiNamespaceTree, page: WikiPageNode } | null>(null)
const pageDeleting = ref(false)
const newPageDialog = ref(false)
const newPagePath = ref('')
const newPageTitle = ref('')
const newPageSaving = ref(false)

const revisions = ref<WikiRevision[]>([])
const reviewLoading = ref(false)
const reviewError = ref('')
const viewingRevision = ref<WikiRevision | null>(null)
const reviewActionRow = ref<{ item: WikiRevision, approve: boolean } | null>(null)
const reviewSaving = ref(false)

const globalLoading = computed(() => nsLoading.value || treeLoading.value || reviewLoading.value)

function formatTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function initial(name: string) {
  return (name || '?').slice(0, 1).toUpperCase()
}

// ---------- Namespaces ----------
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

function openCreateNs() {
  nsForm.name = ''
  nsForm.description = ''
  nsDialog.value = { mode: 'create', row: null }
}

function openEditNs(row: WikiNamespace) {
  nsForm.name = row.name
  nsForm.description = row.description
  nsDialog.value = { mode: 'edit', row }
}

async function submitNamespace() {
  if (!nsForm.name.trim()) {
    adminToast.warning(adminText('k00ng'))
    return
  }
  nsSaving.value = true
  try {
    if (nsDialog.value?.mode === 'create') {
      await createWikiNamespace({ name: nsForm.name.trim(), description: nsForm.description.trim() })
    } else if (nsDialog.value?.row) {
      await updateWikiNamespace(nsDialog.value.row.name, nsForm.description.trim())
    }
    nsDialog.value = null
    await loadNamespaces()
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k00n0'))
  } finally {
    nsSaving.value = false
  }
}

function requestDeleteNamespace(row: WikiNamespace) {
  if (row.pageCount > 0) {
    adminToast.warning(adminText('k00nk'))
    return
  }
  deletingNs.value = row
}

async function confirmDeleteNamespace() {
  if (!deletingNs.value) return
  const name = deletingNs.value.name
  nsDeleting.value = true
  try {
    await deleteWikiNamespace(name)
    deletingNs.value = null
    if (editorNamespace.value === name) editorNamespace.value = ''
    await Promise.all([loadNamespaces(), loadTree()])
    adminToast.success(adminText('k002u'))
  } catch (err) {
    const messageCode = (err as { messageCode?: string } | null)?.messageCode
    adminToast.error(err, messageCode === 'wiki.namespace.hasPages' ? adminText('k00nk') : adminText('k00n0'))
  } finally {
    nsDeleting.value = false
  }
}

// ---------- Editors ----------
function selectEditorNamespace(value: unknown) {
  editorNamespace.value = typeof value === 'string' ? value : ''
}

async function loadEditors(namespace: string) {
  if (!namespace) {
    editors.value = []
    return
  }
  editorLoading.value = true
  try {
    editors.value = await getWikiEditors(namespace)
  } catch (err) {
    adminToast.error(err, adminText('k00n1'))
  } finally {
    editorLoading.value = false
  }
}

watch(editorNamespace, (value) => {
  void loadEditors(value)
})

async function searchEditorUsers(keyword: string) {
  const value = keyword.trim()
  selectedEditorUser.value = null
  if (!value) {
    editorCandidates.value = []
    return
  }
  editorSearching.value = true
  try {
    const params = /^\d+$/.test(value)
      ? { userId: Number(value), page: 1, pageSize: 8 }
      : { username: value, page: 1, pageSize: 8 }
    const result = await getUserList(params)
    editorCandidates.value = result.list || []
  } catch (err) {
    editorCandidates.value = []
    adminToast.error(err, adminText('k00nq'))
  } finally {
    editorSearching.value = false
  }
}

watch(editorSearch, (value) => {
  if (editorSearchTimer) clearTimeout(editorSearchTimer)
  editorSearchTimer = setTimeout(() => {
    void searchEditorUsers(value)
  }, 220)
})

function isEditor(userId: number) {
  return editors.value.some(editor => editor.userId === userId)
}

function selectEditorCandidate(user: AdminUser) {
  if (isEditor(user.userId)) return
  selectedEditorUser.value = user
  editorSearch.value = user.username || String(user.userId)
  editorCandidates.value = []
}

function addEditorUser(user: AdminUser) {
  if (isEditor(user.userId)) return
  editors.value.push({ userId: user.userId, username: user.username, avatarUrl: user.avatarUrl || undefined })
  editorSearch.value = ''
  selectedEditorUser.value = null
  editorCandidates.value = []
}

function addEditorByInput() {
  const value = editorSearch.value.trim()
  if (!value) return
  if (selectedEditorUser.value) {
    addEditorUser(selectedEditorUser.value)
    return
  }
  if (/^\d+$/.test(value)) {
    const userId = Number(value)
    if (!isEditor(userId)) {
      editors.value.push({ userId, username: `#${userId}` })
    }
    editorSearch.value = ''
    return
  }
  adminToast.warning(adminText('k00ej'))
}

function removeEditor(userId: number) {
  editors.value = editors.value.filter(editor => editor.userId !== userId)
}

async function saveEditors() {
  if (!editorNamespace.value) {
    adminToast.warning(adminText('k00nl'))
    return
  }
  editorSaving.value = true
  try {
    await saveWikiEditors(editorNamespace.value, editors.value.map(editor => editor.userId))
    await loadEditors(editorNamespace.value)
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k00n1'))
  } finally {
    editorSaving.value = false
  }
}

// ---------- Page tree ----------
function sortedPages(group: WikiNamespaceTree) {
  return [...(group.pages || [])].sort((a, b) => (a.sortOrder ?? 0) - (b.sortOrder ?? 0))
}

function pagePosition(group: WikiNamespaceTree, page: WikiPageNode) {
  return sortedPages(group).findIndex(item => item.pageId === page.pageId)
}

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

function openRename(group: WikiNamespaceTree, page: WikiPageNode) {
  // 管理树 path 为 namespace 相对路径；预填完整路径避免歧义（review B2）。
  renameForm.path = page.path.startsWith(`${group.name}/`) ? page.path : `${group.name}/${page.path}`
  renameForm.title = page.title
  renameRow.value = { group, page }
}

async function submitRename() {
  if (!renameRow.value) return
  if (!renameForm.path.trim()) {
    adminToast.warning(adminText('k00o4'))
    return
  }
  renameSaving.value = true
  try {
    await saveWikiTree([{
      op: 'rename',
      pageId: renameRow.value.page.pageId,
      newPath: renameForm.path.trim(),
      ...(renameForm.title.trim() ? { newTitle: renameForm.title.trim() } : {}),
    }])
    renameRow.value = null
    await loadTree()
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k00n2'))
  } finally {
    renameSaving.value = false
  }
}

async function movePage(group: WikiNamespaceTree, page: WikiPageNode, direction: 'up' | 'down') {
  const pages = sortedPages(group)
  const index = pages.findIndex(item => item.pageId === page.pageId)
  const targetIndex = direction === 'up' ? index - 1 : index + 1
  if (index < 0 || targetIndex < 0 || targetIndex >= pages.length) return
  const target = pages[targetIndex]
  treeBusy.value = true
  try {
    await saveWikiTree([
      { op: 'sort', pageId: page.pageId, sortOrder: target.sortOrder },
      { op: 'sort', pageId: target.pageId, sortOrder: page.sortOrder },
    ])
    await loadTree()
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k00n2'))
  } finally {
    treeBusy.value = false
  }
}

async function confirmDeletePage() {
  if (!deletingPage.value) return
  pageDeleting.value = true
  try {
    await saveWikiTree([{ op: 'delete', pageId: deletingPage.value.page.pageId }])
    deletingPage.value = null
    await loadTree()
    adminToast.success(adminText('k002u'))
  } catch (err) {
    const messageCode = (err as { messageCode?: string } | null)?.messageCode
    adminToast.error(err, messageCode === 'wiki.page.hasChildren' ? adminText('k00nx') : adminText('k00n2'))
  } finally {
    pageDeleting.value = false
  }
}

function openNewPage(group: WikiNamespaceTree) {
  newPagePath.value = `${group.name}/`
  newPageTitle.value = ''
  newPageDialog.value = true
}

async function confirmNewPage() {
  const path = newPagePath.value.trim().replace(/^\/+/, '')
  const title = newPageTitle.value.trim()
  if (!path) {
    adminToast.warning(adminText('k00o4'))
    return
  }
  if (!title) {
    adminToast.warning(adminText('k00i5'))
    return
  }
  newPageSaving.value = true
  try {
    // 管理端新建页面直接调用创建 API（review P2：此前仅 window.open 跳转，
    // 不落库，页面实际无法创建）。namespace 取路径首段（openNewPage 已预填 group.name/）。
    const namespace = path.split('/')[0]
    await createWikiPage({ namespace, path, title, content: '' })
    newPageDialog.value = false
    await loadTree()
    adminToast.success(adminText('k000e'))
  } catch (err) {
    adminToast.error(err, adminText('k00n2'))
  } finally {
    newPageSaving.value = false
  }
}

// ---------- Review ----------
async function loadRevisions() {
  reviewLoading.value = true
  reviewError.value = ''
  try {
    revisions.value = await getWikiRevisions('pending')
  } catch (err) {
    reviewError.value = err instanceof Error ? err.message : adminText('k00n3')
  } finally {
    reviewLoading.value = false
  }
}

async function confirmReviewAction() {
  if (!reviewActionRow.value) return
  const { item, approve } = reviewActionRow.value
  reviewSaving.value = true
  try {
    await reviewWikiRevision(item.revisionId, approve ? 'approve' : 'reject')
    reviewActionRow.value = null
    if (viewingRevision.value?.revisionId === item.revisionId) viewingRevision.value = null
    await loadRevisions()
    adminToast.success(approve ? adminText('k00gj') : adminText('k00gk'))
  } catch (err) {
    adminToast.error(err, adminText('k00n3'))
  } finally {
    reviewSaving.value = false
  }
}

async function loadAll() {
  await Promise.all([loadNamespaces(), loadTree(), loadRevisions()])
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
        <TabsTrigger value="editors">{{ adminText('k00n7') }}</TabsTrigger>
        <TabsTrigger value="pages">{{ adminText('k00n8') }}</TabsTrigger>
        <TabsTrigger value="review">{{ adminText('k00n9') }}</TabsTrigger>
      </TabsList>

      <TabsContent value="namespaces">
        <AdminSection>
          <template #header>
            <AdminToolbar class="border-b-0">
              <div class="flex items-center justify-between gap-3">
                <Badge variant="secondary" class="h-9 rounded-md px-3">
                  {{ namespaces.length }} {{ adminText('k00n6') }}
                </Badge>
                <Button type="button" @click="openCreateNs">
                  <Plus class="size-4" />
                  {{ adminText('k00nc') }}
                </Button>
              </div>
            </AdminToolbar>
          </template>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ adminText('k00af') }}</TableHead>
                <TableHead>{{ adminText('k00ag') }}</TableHead>
                <TableHead class="w-24">{{ adminText('k00na') }}</TableHead>
                <TableHead class="w-40">{{ adminText('k00nb') }}</TableHead>
                <TableHead class="w-44 text-right">{{ adminText('k007m') }}</TableHead>
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
                  <TableCell class="max-w-md truncate text-muted-foreground">{{ item.description || '-' }}</TableCell>
                  <TableCell>{{ item.pageCount ?? 0 }}</TableCell>
                  <TableCell class="text-xs text-muted-foreground">{{ formatTime(item.updatedAt) }}</TableCell>
                  <TableCell>
                    <div class="flex justify-end gap-2">
                      <AdminActionButton @click="openEditNs(item)">
                        <Pencil class="size-3.5" />
                        {{ adminText('k00nd') }}
                      </AdminActionButton>
                      <AdminActionButton tone="danger" @click="requestDeleteNamespace(item)">
                        <Trash2 class="size-3.5" />
                        {{ adminText('k00ne') }}
                      </AdminActionButton>
                    </div>
                  </TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
        </AdminSection>
      </TabsContent>

      <TabsContent value="editors">
        <AdminSection>
          <template #header>
            <AdminToolbar class="border-b-0">
              <div class="flex flex-wrap items-center gap-3">
                <Select :model-value="editorNamespace" :disabled="namespaces.length === 0" @update:model-value="selectEditorNamespace">
                  <SelectTrigger class="h-9 w-64">
                    <SelectValue :placeholder="adminText('k00nm')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="ns in namespaces" :key="ns.name" :value="ns.name">
                      {{ ns.name }}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <span v-if="!editorNamespace" class="text-sm text-muted-foreground">{{ adminText('k00nl') }}</span>
              </div>
            </AdminToolbar>
          </template>
          <div v-if="!editorNamespace" class="px-4 py-10 text-center text-sm text-muted-foreground">
            {{ adminText('k00nl') }}
          </div>
          <div v-else class="grid gap-4 p-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
            <div class="space-y-3">
              <div class="text-sm font-medium">{{ adminText('k0094') }}</div>
              <form class="flex gap-2" @submit.prevent="addEditorByInput">
                <div class="relative min-w-0 flex-1">
                  <Input v-model="editorSearch" :placeholder="adminText('k00f1')" autocomplete="off" />
                  <div
                    v-if="editorSearch.trim() && (editorCandidates.length || editorSearching)"
                    class="absolute left-0 right-0 top-[calc(100%+4px)] z-50 overflow-hidden rounded-md border bg-popover shadow-md"
                  >
                    <div v-if="editorSearching" class="px-3 py-2 text-sm text-muted-foreground">{{ adminText('k00ev') }}</div>
                    <button
                      v-for="user in editorCandidates"
                      v-else
                      :key="user.userId"
                      class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors"
                      :class="isEditor(user.userId) ? 'cursor-default opacity-55' : 'hover:bg-muted'"
                      type="button"
                      :disabled="isEditor(user.userId)"
                      @click="selectEditorCandidate(user)"
                    >
                      <img v-if="user.avatarUrl" :src="user.avatarUrl" class="size-7 rounded-full object-cover ring-1 ring-border" alt="" />
                      <span v-else class="flex size-7 items-center justify-center rounded-full bg-muted text-xs font-semibold">{{ initial(user.username) }}</span>
                      <span class="min-w-0 flex-1 truncate">{{ user.username }}</span>
                      <span v-if="isEditor(user.userId)" class="shrink-0 rounded-full bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground">{{ adminText('k00ew') }}</span>
                      <span class="shrink-0 font-mono text-xs text-muted-foreground">#{{ user.userId }}</span>
                    </button>
                  </div>
                </div>
                <Button type="submit" variant="outline" size="sm" :disabled="editorSaving">
                  <UserPlus class="size-3.5" />
                  {{ adminText('k0094') }}
                </Button>
              </form>
              <p class="text-xs text-muted-foreground">{{ adminText('k00n5') }}</p>
            </div>

            <div class="overflow-hidden rounded-lg border">
              <div class="flex items-center justify-between border-b bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                <span>{{ adminText('k00nn') }}</span>
                <span>{{ editors.length }}</span>
              </div>
              <div v-if="editorLoading" class="px-4 py-8 text-center text-sm text-muted-foreground">{{ adminText('k0046') }}</div>
              <div v-else-if="editors.length === 0" class="px-4 py-8 text-center text-sm text-muted-foreground">{{ adminText('k00no') }}</div>
              <div v-else class="max-h-72 divide-y overflow-y-auto">
                <div v-for="editor in editors" :key="editor.userId" class="flex items-center justify-between gap-3 px-3 py-2">
                  <div class="flex min-w-0 items-center gap-2">
                    <img v-if="editor.avatarUrl" :src="editor.avatarUrl" class="size-8 rounded-full object-cover ring-1 ring-border" alt="" />
                    <span v-else class="grid size-8 place-items-center rounded-full bg-muted text-xs font-semibold">{{ initial(editor.username) }}</span>
                    <div class="min-w-0">
                      <div class="truncate text-sm font-medium">{{ editor.username || `#${editor.userId}` }}</div>
                      <div class="text-xs text-muted-foreground">ID {{ editor.userId }}</div>
                    </div>
                  </div>
                  <Button variant="ghost" size="icon-sm" type="button" :title="adminText('k00np')" :disabled="editorSaving" @click="removeEditor(editor.userId)">
                    <X class="size-4" />
                  </Button>
                </div>
              </div>
              <div class="flex justify-end gap-2 border-t bg-muted/20 px-3 py-2">
                <Button type="button" size="sm" :disabled="editorSaving || editorLoading" @click="saveEditors">
                  <Save class="size-3.5" />
                  {{ editorSaving ? adminText('k005f') : adminText('k005g') }}
                </Button>
              </div>
            </div>
          </div>
        </AdminSection>
      </TabsContent>

      <TabsContent value="pages">
        <AdminSection>
          <template #header>
            <AdminToolbar class="border-b-0">
              <Badge variant="secondary" class="h-9 rounded-md px-3">
                {{ tree.length }} {{ adminText('k00n6') }}
              </Badge>
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
                  <Button type="button" size="sm" variant="outline" class="h-8 text-xs" @click="openNewPage(group)">
                    <Plus class="size-3.5" />
                    {{ adminText('k00nu') }}
                  </Button>
                </div>
                <div v-if="!group.pages.length" class="px-4 py-6 text-center text-sm text-muted-foreground">{{ adminText('k00ny') }}</div>
                <div v-else class="divide-y">
                  <div v-for="page in sortedPages(group)" :key="page.pageId" class="flex items-center gap-2 px-3 py-2 pl-6">
                    <FileText class="size-4 shrink-0 text-muted-foreground" />
                    <div class="min-w-0 flex-1">
                      <div class="truncate text-sm font-medium">{{ page.title || page.path }}</div>
                      <div class="truncate font-mono text-xs text-muted-foreground">{{ page.path }}</div>
                    </div>
                    <div class="flex shrink-0 items-center gap-1">
                      <AdminActionButton
                        compact
                        :title="adminText('k00ns')"
                        :disabled="treeBusy || pagePosition(group, page) === 0"
                        @click="movePage(group, page, 'up')"
                      >
                        <ChevronUp class="size-3.5" />
                      </AdminActionButton>
                      <AdminActionButton
                        compact
                        :title="adminText('k00nt')"
                        :disabled="treeBusy || pagePosition(group, page) === sortedPages(group).length - 1"
                        @click="movePage(group, page, 'down')"
                      >
                        <ChevronDown class="size-3.5" />
                      </AdminActionButton>
                      <AdminActionButton compact :title="adminText('k00nr')" @click="openRename(group, page)">
                        <Pencil class="size-3.5" />
                      </AdminActionButton>
                      <AdminActionButton compact tone="danger" :title="adminText('k005i')" @click="deletingPage = { group, page }">
                        <Trash2 class="size-3.5" />
                      </AdminActionButton>
                      <a
                        :href="`/wiki/${page.path}`"
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

      <TabsContent value="review">
        <AdminSection>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ adminText('k00g1') }}</TableHead>
                <TableHead>{{ adminText('k00i5') }}</TableHead>
                <TableHead>{{ adminText('k00nz') }}</TableHead>
                <TableHead>{{ adminText('k003b') }}</TableHead>
                <TableHead class="w-56 text-right">{{ adminText('k007m') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="reviewLoading && revisions.length === 0">
                <TableCell colspan="5" class="h-28 text-center text-muted-foreground">{{ adminText('k0046') }}</TableCell>
              </TableRow>
              <TableRow v-else-if="reviewError">
                <TableCell colspan="5" class="h-28 text-center text-destructive">{{ reviewError }}</TableCell>
              </TableRow>
              <TableRow v-else-if="revisions.length === 0">
                <TableCell colspan="5" class="h-28 text-center text-muted-foreground">{{ adminText('k00o1') }}</TableCell>
              </TableRow>
              <template v-else>
                <TableRow v-for="item in revisions" :key="item.revisionId">
                  <TableCell class="font-mono text-xs text-muted-foreground">{{ item.path }}</TableCell>
                  <TableCell class="font-medium">{{ item.title || '-' }}</TableCell>
                  <TableCell>{{ item.editorName || '-' }}</TableCell>
                  <TableCell class="text-xs text-muted-foreground">{{ formatTime(item.updatedAt) }}</TableCell>
                  <TableCell>
                    <div class="flex justify-end gap-1.5">
                      <Button type="button" size="sm" variant="outline" class="h-8 text-xs" @click="viewingRevision = item">
                        <Eye class="size-3.5" />
                        {{ adminText('k00o0') }}
                      </Button>
                      <Button type="button" size="sm" variant="outline" class="h-8 text-xs" @click="reviewActionRow = { item, approve: true }">
                        <Check class="size-3.5" />
                        {{ adminText('k00gj') }}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        class="h-8 text-xs text-destructive hover:text-destructive"
                        @click="reviewActionRow = { item, approve: false }"
                      >
                        <X class="size-3.5" />
                        {{ adminText('k00gk') }}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
        </AdminSection>
      </TabsContent>
    </Tabs>

    <Dialog :open="nsDialog !== null" @update:open="(open) => !open && (nsDialog = null)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ nsDialog?.mode === 'edit' ? adminText('k00nd') : adminText('k00nc') }}</DialogTitle>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="submitNamespace">
          <label v-if="nsDialog?.mode === 'create'" class="grid gap-2 text-sm font-medium">
            {{ adminText('k00af') }}
            <Input v-model="nsForm.name" :placeholder="adminText('k00nh')" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00ag') }}
            <Textarea v-model="nsForm.description" class="min-h-24 resize-y" :placeholder="adminText('k00ni')" />
          </label>
          <DialogFooter>
            <Button variant="outline" type="button" @click="nsDialog = null">{{ adminText('k009q') }}</Button>
            <Button type="submit" :disabled="nsSaving">{{ nsSaving ? adminText('k005f') : adminText('k005g') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <AdminConfirmDialog
      :open="deletingNs !== null"
      :title="adminText('k00ne')"
      :description="adminText('k00nf', { name: deletingNs?.name })"
      :loading="nsDeleting"
      @update:open="(open) => !open && (deletingNs = null)"
      @confirm="confirmDeleteNamespace"
    />

    <Dialog :open="renameRow !== null" @update:open="(open) => !open && (renameRow = null)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ adminText('k00nr') }}</DialogTitle>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="submitRename">
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00g1') }}
            <Input v-model="renameForm.path" class="font-mono" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00i5') }}
            <Input v-model="renameForm.title" />
          </label>
          <DialogFooter>
            <Button variant="outline" type="button" @click="renameRow = null">{{ adminText('k009q') }}</Button>
            <Button type="submit" :disabled="renameSaving">{{ renameSaving ? adminText('k005f') : adminText('k005g') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <AdminConfirmDialog
      :open="deletingPage !== null"
      :title="adminText('k00nw', { title: deletingPage?.page.title || deletingPage?.page.path })"
      :loading="pageDeleting"
      @update:open="(open) => !open && (deletingPage = null)"
      @confirm="confirmDeletePage"
    />

    <Dialog :open="newPageDialog" @update:open="(open) => !open && (newPageDialog = false)">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ adminText('k00nu') }}</DialogTitle>
        </DialogHeader>
        <form class="grid gap-4" @submit.prevent="confirmNewPage">
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00g1') }}
            <Input v-model="newPagePath" class="font-mono" placeholder="guide/hello" />
          </label>
          <label class="grid gap-2 text-sm font-medium">
            {{ adminText('k00i5') }}
            <Input v-model="newPageTitle" :placeholder="adminText('k00i5')" />
          </label>
          <DialogFooter>
            <Button variant="outline" type="button" @click="newPageDialog = false">{{ adminText('k009q') }}</Button>
            <Button type="submit" :disabled="newPageSaving">
              <Save class="size-4" />
              {{ newPageSaving ? adminText('k005f') : adminText('k005g') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog :open="viewingRevision !== null" @update:open="(open) => !open && (viewingRevision = null)">
      <DialogScrollContent class="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{{ viewingRevision?.title || '-' }}</DialogTitle>
          <DialogDescription>
            {{ viewingRevision?.path }} · {{ viewingRevision?.editorName }} · {{ formatTime(viewingRevision?.updatedAt || '') }}
          </DialogDescription>
        </DialogHeader>
        <pre class="max-h-[60vh] overflow-auto whitespace-pre-wrap break-words rounded-md border bg-muted/20 p-3 font-mono text-xs leading-5">{{ viewingRevision?.content || '' }}</pre>
        <DialogFooter>
          <Button variant="outline" type="button" @click="viewingRevision = null">{{ adminText('k009q') }}</Button>
        </DialogFooter>
      </DialogScrollContent>
    </Dialog>

    <AdminConfirmDialog
      :open="reviewActionRow !== null"
      :title="reviewActionRow?.approve ? adminText('k00gj') : adminText('k00gk')"
      :description="reviewActionRow?.approve ? adminText('k00o2') : adminText('k00o3')"
      :confirm-text="reviewActionRow?.approve ? adminText('k00gj') : adminText('k00gk')"
      :loading="reviewSaving"
      @update:open="(open) => !open && (reviewActionRow = null)"
      @confirm="confirmReviewAction"
    />
  </BasicPage>
</template>
