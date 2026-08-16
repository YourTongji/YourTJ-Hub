<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ChevronDown, ChevronRight, House, Library } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { WikiTreeNamespace } from '@gooseforum/client'
import WikiSearchBar from './WikiSearchBar.vue'
import WikiSidebarNode from './WikiSidebarNode.vue'

const props = defineProps<{
  tree: WikiTreeNamespace[]
}>()

// 移动端抽屉复用本组件时，点击导航项后由宿主关闭抽屉。
const emit = defineEmits<{
  navigate: []
}>()

const { t } = useI18n()
const route = useRoute()
const collapsed = ref<Set<string>>(new Set())
// 用响应式 route.path 而非 window.location.pathname：
// SPA 导航（vue-router pushState）下 AppShell/WikiSidebar 不会重新挂载，
// 读 window.location 的 computed 会缓存旧值导致高亮态不随页面切换更新。
const isSiteHome = computed(() => route.path === '/' || route.path === '')
const isWikiHome = computed(() => route.path === '/wiki' || route.path === '/wiki/')

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
  <nav class="pb-3" aria-label="Wiki sidebar">
    <!-- sticky 头部：胶囊 + 搜索栏。外层 aside 自身已是 sticky top-16 挂在 header 下沿，
         因此这里相对侧栏滚动容器用 top-0 贴顶即可，不能再用 top-16（会留下 64px 空缺）；
         也不要用负 margin（会让 sticky 元素天然位置越过容器顶，页面一加载就被钉住偏下）。 -->
    <div class="sticky top-0 z-20 bg-base-200/95 px-2 pb-2 pt-3 backdrop-blur-sm">
      <div class="flex gap-1 rounded-full border border-line bg-base-100 p-1 shadow-sm">
        <a
          href="/"
          class="flex h-7 flex-1 items-center justify-center gap-1.5 rounded-full px-2 text-[12px] font-semibold transition-colors duration-150"
          :class="isSiteHome ? 'bg-primary/10 text-primary' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
          @click="emit('navigate')"
        >
          <House class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
          <span class="truncate">{{ t('common.home') }}</span>
        </a>
        <a
          href="/wiki"
          class="flex h-7 flex-1 items-center justify-center gap-1.5 rounded-full px-2 text-[12px] font-semibold transition-colors duration-150"
          :class="isWikiHome ? 'bg-primary/10 text-primary' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
          @click="emit('navigate')"
        >
          <Library class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
          <span class="truncate">{{ t('wiki.home') }}</span>
        </a>
      </div>
      <div class="mt-2">
        <WikiSearchBar />
      </div>
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
          @navigate="emit('navigate')"
        />
      </div>
    </div>
  </nav>
</template>
