<script setup lang="ts">
import { ExternalLink, FileText, Folder } from '@lucide/vue'
import type { WikiTreeNode } from '@/admin/types'

defineOptions({ name: 'WikiTreeNode' })

const props = defineProps<{
  node: WikiTreeNode
  depth: number
  editUrlFor: (node: WikiTreeNode) => string
  editTitle: string
  viewTitle: string
}>()

function wikiHref(path: string) {
  return `/wiki/${path.split('/').map((segment) => encodeURIComponent(segment)).join('/')}`
}
</script>

<template>
  <div>
    <div class="flex items-center gap-2 px-3 py-2" :style="{ paddingLeft: `${24 + depth * 18}px` }">
      <Folder v-if="node.kind === 'directory'" class="size-4 shrink-0 text-muted-foreground" />
      <FileText v-else class="size-4 shrink-0 text-muted-foreground" />
      <div class="min-w-0 flex-1">
        <div class="truncate text-sm font-medium">{{ node.title || node.path }}</div>
        <div class="truncate font-mono text-xs text-muted-foreground">{{ node.path }}</div>
      </div>
      <div v-if="node.kind === 'page'" class="flex shrink-0 items-center gap-1">
        <a
          v-if="editUrlFor(node)"
          :href="editUrlFor(node)"
          target="_blank"
          rel="noopener noreferrer"
          class="grid size-8 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          :title="editTitle"
        >
          <ExternalLink class="size-3.5" />
        </a>
        <a
          :href="wikiHref(node.path)"
          target="_blank"
          rel="noopener noreferrer"
          class="grid size-8 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          :title="viewTitle"
        >
          <ExternalLink class="size-3.5" />
        </a>
      </div>
    </div>
    <div v-if="node.children.length" class="divide-y">
      <WikiTreeNode
        v-for="child in node.children"
        :key="`${child.kind}:${child.path}`"
        :node="child"
        :depth="depth + 1"
        :edit-url-for="editUrlFor"
        :edit-title="editTitle"
        :view-title="viewTitle"
      />
    </div>
  </div>
</template>
