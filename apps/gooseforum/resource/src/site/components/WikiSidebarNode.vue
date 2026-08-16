<script setup lang="ts">
import { computed } from 'vue'
import { ChevronDown, ChevronRight } from '@lucide/vue'
import type { WikiTreePage } from '@gooseforum/client'

const props = defineProps<{
  node: WikiTreePage
  depth: number
}>()

const collapsed = defineModel<Set<string>>('collapsed', { required: true })

const hasChildren = computed(() => (props.node.children?.length ?? 0) > 0)
const isDirectory = computed(() => props.node.pageId === 0)

function isCollapsed(key: string) {
  return collapsed.value.has(key)
}

function toggle(key: string) {
  const next = new Set(collapsed.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsed.value = next
}

// GitHub SSOT：路径保留中文等 Unicode（不再小写归一），URL 按段编码。
function wikiHref(path: string): string {
  return '/wiki/' + path.split('/').map((seg) => encodeURIComponent(seg)).join('/')
}
</script>

<template>
  <div>
    <!-- 纯目录节点：分组头（可折叠，不可点击）。 -->
    <button
      v-if="isDirectory"
      type="button"
      class="flex h-7 w-full items-center gap-1 rounded-md pr-2 text-left text-xs font-semibold text-base-content/60 transition-colors hover:bg-base-300 hover:text-base-content"
      :style="{ paddingLeft: `${8 + depth * 14}px` }"
      :aria-expanded="!isCollapsed(node.path)"
      @click="toggle(node.path)"
    >
      <ChevronDown v-if="!isCollapsed(node.path)" class="h-3 w-3 shrink-0" />
      <ChevronRight v-else class="h-3 w-3 shrink-0" />
      <span class="truncate">{{ node.title }}</span>
    </button>

    <!-- 页面节点：链接；带子级时折叠箭头独立可点。 -->
    <div
      v-else
      class="flex w-full items-center gap-1 rounded-md pr-2 transition-colors duration-150"
      :class="node.active ? 'bg-info/10 text-primary' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
      :style="{ paddingLeft: `${8 + depth * 14}px` }"
    >
      <button
        v-if="hasChildren"
        type="button"
        class="grid h-4 w-4 shrink-0 place-items-center rounded hover:bg-base-300"
        :aria-expanded="!isCollapsed(node.path)"
        @click="toggle(node.path)"
      >
        <ChevronDown v-if="!isCollapsed(node.path)" class="h-3 w-3" />
        <ChevronRight v-else class="h-3 w-3" />
      </button>
      <span v-else class="h-4 w-4 shrink-0" />
      <a
        :href="wikiHref(node.path)"
        class="flex min-h-7 min-w-0 flex-1 items-center text-[13px] font-medium transition-colors duration-150"
        :class="node.active ? 'text-primary' : 'hover:text-base-content'"
      >
        <span class="truncate">{{ node.title }}</span>
      </a>
    </div>
    </div>

    <div v-if="hasChildren && !isCollapsed(node.path)" class="space-y-px">
      <WikiSidebarNode
        v-for="child in node.children || []"
        :key="child.path"
        v-model:collapsed="collapsed"
        :node="child"
        :depth="depth + 1"
      />
    </div>
  </div>
</template>
