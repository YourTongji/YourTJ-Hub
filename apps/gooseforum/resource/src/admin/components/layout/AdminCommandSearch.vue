<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { CornerDownLeft, Search } from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { adminNavGroups } from '@/admin/runtime/navigation'
import { commandEntries, searchCommands, type CommandEntry } from '@/admin/runtime/command-search'

const { t, locale } = useI18n()
const router = useRouter()
const open = ref(false)
const query = ref('')
const active = ref(0)
const input = ref<HTMLInputElement>()
const list = ref<HTMLElement>()
const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent)
const shortcut = isMac ? '⌘ K' : 'Ctrl K'

const entries = computed(() => {
  void locale.value
  return commandEntries(adminNavGroups())
})
const results = computed(() => searchCommands(entries.value, query.value))
// Without a query, keep the sidebar grouping so the list doubles as a page index.
const sections = computed(() => {
  if (query.value.trim()) return [{ title: '', rows: results.value.map((entry, index) => ({ entry, index })) }]
  const groups: { title: string; rows: { entry: CommandEntry; index: number }[] }[] = []
  results.value.forEach((entry, index) => {
    const last = groups.at(-1)
    if (last?.title === entry.group) last.rows.push({ entry, index })
    else groups.push({ title: entry.group, rows: [{ entry, index }] })
  })
  return groups
})

watch(query, () => { active.value = 0 })
watch(open, (value) => {
  if (!value) return
  query.value = ''
  active.value = 0
  void nextTick(() => input.value?.focus())
})
watch(active, () => {
  void nextTick(() => list.value?.querySelector<HTMLElement>('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' }))
})

function move(step: number) {
  const count = results.value.length
  if (count) active.value = (active.value + step + count) % count
}
function choose(entry?: CommandEntry) {
  if (!entry) return
  open.value = false
  void router.push(entry.item.url)
}
function onGlobalKeydown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && !event.altKey && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    open.value = !open.value
  }
}
onMounted(() => window.addEventListener('keydown', onGlobalKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onGlobalKeydown))
</script>

<template>
  <button
    type="button"
    class="hidden h-9 w-64 items-center gap-2 rounded-lg bg-muted/70 px-3 text-sm text-muted-foreground transition-colors duration-150 hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 md:inline-flex lg:w-72"
    :aria-label="t('adminShell.searchLabel')"
    aria-haspopup="dialog"
    @click="open = true"
  >
    <Search class="size-4 shrink-0" />
    <span class="flex-1 truncate text-start">{{ t('adminShell.searchPlaceholder') }}</span>
    <kbd class="pointer-events-none inline-flex h-5 select-none items-center rounded bg-background px-1.5 font-mono text-[10px] font-medium text-muted-foreground shadow-[0_0_0_1px_var(--border)]">{{ shortcut }}</kbd>
  </button>
  <button
    type="button"
    class="inline-flex size-9 items-center justify-center rounded-lg text-muted-foreground transition-colors duration-150 hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 md:hidden"
    :aria-label="t('adminShell.searchLabel')"
    aria-haspopup="dialog"
    @click="open = true"
  >
    <Search class="size-4" />
  </button>

  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[60] bg-black/40 data-[state=open]:animate-in data-[state=open]:fade-in-0" />
      <DialogContent
        class="fixed left-1/2 top-[12vh] z-[61] flex max-h-[min(36rem,76vh)] w-[calc(100%-1.5rem)] max-w-xl -translate-x-1/2 flex-col overflow-hidden rounded-xl bg-popover text-popover-foreground shadow-[0_24px_64px_-12px_oklch(0_0_0/0.35),0_0_0_1px_var(--border)] outline-none data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-[0.98]"
      >
        <DialogTitle class="sr-only">{{ t('adminShell.searchLabel') }}</DialogTitle>
        <DialogDescription class="sr-only">{{ t('adminShell.searchKeyboardHint') }}</DialogDescription>
        <div class="flex items-center gap-3 border-b px-4">
          <Search class="size-4 shrink-0 text-muted-foreground" />
          <input
            ref="input"
            v-model="query"
            type="text"
            role="combobox"
            aria-autocomplete="list"
            aria-expanded="true"
            aria-controls="admin-command-results"
            :aria-activedescendant="results.length ? `admin-command-${active}` : undefined"
            :placeholder="t('adminShell.searchPlaceholder')"
            class="h-12 min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            autocomplete="off"
            spellcheck="false"
            @keydown.down.prevent="move(1)"
            @keydown.up.prevent="move(-1)"
            @keydown.enter.prevent="choose(results[active])"
          />
        </div>
        <div id="admin-command-results" ref="list" role="listbox" :aria-label="t('adminShell.searchLabel')" class="min-h-0 flex-1 overflow-y-auto overscroll-contain p-1.5">
          <template v-for="section in sections" :key="section.title">
            <p v-if="section.title" class="px-2.5 pb-1 pt-2.5 text-xs font-medium text-muted-foreground" aria-hidden="true">{{ section.title }}</p>
            <div
              v-for="row in section.rows"
              :id="`admin-command-${row.index}`"
              :key="row.entry.item.url"
              role="option"
              :aria-selected="row.index === active"
              class="flex cursor-pointer items-center gap-3 rounded-lg px-2.5 py-2 text-sm"
              :class="row.index === active ? 'bg-accent text-accent-foreground' : 'text-foreground'"
              @mousemove="active = row.index"
              @click="choose(row.entry)"
            >
              <component :is="row.entry.item.icon" class="size-4 shrink-0 text-muted-foreground" />
              <span class="min-w-0 flex-1 truncate">{{ row.entry.item.title }}</span>
              <span v-if="!section.title" class="shrink-0 truncate text-xs text-muted-foreground">{{ row.entry.group }}</span>
              <CornerDownLeft v-if="row.index === active" class="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
            </div>
          </template>
          <div v-if="!results.length" class="px-4 py-10 text-center" role="status">
            <p class="text-sm font-medium">{{ t('adminShell.searchEmpty', { query: query.trim() }) }}</p>
            <p class="mt-1 text-xs text-muted-foreground">{{ t('adminShell.searchEmptyHint') }}</p>
          </div>
        </div>
        <div class="hidden items-center gap-4 border-t px-4 py-2 text-xs text-muted-foreground sm:flex" aria-hidden="true">
          <span class="inline-flex items-center gap-1.5"><kbd class="rounded bg-muted px-1 font-mono">↑↓</kbd>{{ t('adminShell.searchNavigate') }}</span>
          <span class="inline-flex items-center gap-1.5"><kbd class="rounded bg-muted px-1 font-mono">Enter</kbd>{{ t('adminShell.searchOpen') }}</span>
          <span class="inline-flex items-center gap-1.5"><kbd class="rounded bg-muted px-1 font-mono">Esc</kbd>{{ t('adminShell.searchClose') }}</span>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
