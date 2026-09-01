<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChevronDown, ChevronRight, FileText, Folder } from '@lucide/vue'
import type { WikiTreeNode } from '@gooseforum/client'
import { wikiHref } from '@/runtime/wiki-path'

defineOptions({ name: 'WikiSidebarNode' })

const props = defineProps<{
  node: WikiTreeNode
  depth: number
}>()

// 移动端抽屉复用侧栏时，点击页面链接后由宿主关闭抽屉（与 WikiSidebar 一致）。
const emit = defineEmits<{
  navigate: []
}>()

const collapsed = ref(false)
const isDirectory = computed(() => props.node.kind === 'directory')
</script>

<template>
  <div>
    <button
      v-if="isDirectory"
      type="button"
      class="flex h-7 w-full items-center gap-1 rounded-md pr-2 text-left text-[13px] font-medium text-base-content/70 transition-colors duration-150 hover:bg-base-300 hover:text-base-content"
      :style="{ paddingLeft: `${8 + depth * 12}px` }"
      :aria-expanded="!collapsed"
      @click="collapsed = !collapsed"
    >
      <ChevronDown v-if="!collapsed" class="h-3.5 w-3.5 shrink-0" />
      <ChevronRight v-else class="h-3.5 w-3.5 shrink-0" />
      <Folder class="h-3.5 w-3.5 shrink-0" />
      <span class="truncate">{{ node.title }}</span>
    </button>
    <a
      v-else
      :href="wikiHref(node.path)"
      class="flex h-7 items-center gap-2 rounded-md pr-2 text-[13px] font-medium transition-colors duration-150"
      :style="{ paddingLeft: `${20 + depth * 12}px` }"
      :class="node.active ? 'bg-info/10 text-primary' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
      @click="emit('navigate')"
    >
      <FileText class="h-3.5 w-3.5 shrink-0 opacity-70" />
      <span class="truncate">{{ node.title }}</span>
    </a>
    <div v-if="isDirectory && !collapsed" class="space-y-px">
      <WikiSidebarNode
        v-for="child in node.children"
        :key="`${child.kind}:${child.path}`"
        :node="child"
        :depth="depth + 1"
        @navigate="emit('navigate')"
      />
    </div>
  </div>
</template>
