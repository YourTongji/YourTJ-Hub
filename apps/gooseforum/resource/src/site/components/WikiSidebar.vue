<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChevronDown, ChevronRight } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { WikiTreeNamespace } from '@gooseforum/client'
import WikiSidebarNode from './WikiSidebarNode.vue'

const props = defineProps<{
  tree: WikiTreeNamespace[]
}>()

const { t } = useI18n()
const collapsed = ref<Set<string>>(new Set())
const isHome = computed(() => {
  if (typeof window === 'undefined') return false
  return window.location.pathname === '/wiki' || window.location.pathname === '/wiki/'
})

const groups = computed(() => props.tree || [])

function isCollapsed(name: string) {
  return collapsed.value.has(name)
}

function toggleCollapse(name: string) {
  const next = new Set(collapsed.value)
  if (next.has(name)) next.delete(name)
  else next.add(name)
  collapsed.value = next
}

</script>

<template>
  <nav class="py-3" aria-label="Wiki sidebar">
    <div class="pb-2">
      <a
        href="/wiki"
        class="flex h-8 items-center gap-2 rounded-md px-2 text-[13px] font-semibold transition-colors duration-150"
        :class="isHome ? 'bg-info/10 text-primary' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
      >
        {{ t('wiki.home') }}
      </a>
    </div>

    <div v-if="!groups.length" class="px-2 py-3 text-xs text-base-content/55">
      {{ t('wiki.sidebarEmpty') }}
    </div>

    <div
      v-for="group in groups"
      :key="group.name"
      class="mt-2"
    >
      <button
        type="button"
        class="flex h-7 w-full items-center gap-1 rounded-md px-2 text-left text-[11px] font-bold uppercase tracking-wide text-base-content/55 transition-colors hover:bg-base-300 hover:text-base-content"
        :aria-expanded="!isCollapsed(group.name)"
        @click="toggleCollapse(group.name)"
      >
        <ChevronDown v-if="!isCollapsed(group.name)" class="h-3 w-3 shrink-0" />
        <ChevronRight v-else class="h-3 w-3 shrink-0" />
        <span class="truncate">{{ group.label }}</span>
      </button>
      <div v-if="!isCollapsed(group.name)" class="space-y-px">
        <WikiSidebarNode
          v-for="node in group.nodes"
          :key="`${node.kind}:${node.path}`"
          :node="node"
          :depth="0"
        />
      </div>
    </div>
  </nav>
</template>
