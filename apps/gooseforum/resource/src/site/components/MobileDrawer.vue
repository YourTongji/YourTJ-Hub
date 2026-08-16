<script setup lang="ts">
import { computed } from 'vue'
import { X } from '@lucide/vue'
import { DialogContent, DialogOverlay, DialogRoot, DialogTitle } from 'reka-ui'
import type { FooterPayload, WikiTreeNamespace } from '@gooseforum/client'
import WikiSidebar from './WikiSidebar.vue'

interface SidebarNavItem {
  key: string
  label: string
  i18nLabel?: string
  url: string
  active: boolean
}

interface SidebarCategoryItem extends SidebarNavItem {
  id: number
  color: string
}

interface SidebarGroupItem {
  key: string
  title: string
  i18nLabel?: string
  items: SidebarNavItem[]
}

const props = defineProps<{
  open: boolean
  primaryItems: SidebarNavItem[]
  resourceItems: SidebarNavItem[]
  sidebarGroups: SidebarGroupItem[]
  categoryItems: SidebarCategoryItem[]
  footer: FooterPayload
  /** wiki 模式：抽屉顶部展示完整 wiki 导航树（首页/命名空间/页面），与桌面侧栏一致。 */
  wikiMode?: boolean
  wikiTree?: WikiTreeNamespace[]
  hasUnreadMessages?: boolean
  hasUnreadNotifications?: boolean
  hasModerationReports?: boolean
  closeLabel: string
  menuLabel: string
  resourcesLabel: string
  categoriesLabel: string
  sidebarIcon: (item: SidebarNavItem) => unknown
}>()

const emit = defineEmits<{
  close: []
}>()

const hasFooter = computed(() => props.footer.links.length > 0 || props.footer.primary.length > 0)

function close() {
  emit('close')
}
</script>

<template>
  <DialogRoot :open="props.open" @update:open="(open) => { if (!open) close() }">
    <DialogOverlay
      class="fixed inset-0 z-[60] bg-neutral/40 duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 lg:hidden"
    />
    <DialogContent
      class="gf-drawer-surface fixed inset-y-0 left-0 z-[60] h-full w-80 max-w-[85vw] overflow-y-auto p-3 outline-none duration-300 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left lg:hidden"
      :aria-describedby="undefined"
    >
        <div class="mb-3 flex h-10 items-center justify-between">
          <DialogTitle class="text-base font-bold text-base-content">{{ menuLabel }}</DialogTitle>
          <button class="inline-flex h-8 w-8 items-center justify-center rounded-md text-icon-muted hover:bg-base-300 hover:text-base-content" type="button" :aria-label="closeLabel" @click="close">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div v-if="wikiMode" class="pb-2">
          <WikiSidebar :tree="wikiTree || []" @navigate="close" />
        </div>
        <template v-else>
          <div class="space-y-0.5">
            <a
              v-for="item in primaryItems"
              :key="item.key"
              :href="item.url"
              class="flex h-9 items-center gap-2 rounded-md px-2 text-sm font-medium"
              :class="item.active ? 'bg-info/10 text-primary' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
            >
              <component
                :is="sidebarIcon(item)"
                v-if="sidebarIcon(item)"
                class="h-4 w-4 shrink-0"
                aria-hidden="true"
              />
              <span class="min-w-0 flex-1 truncate">{{ item.label }}</span>
              <span
                v-if="(item.key === 'messages' && hasUnreadMessages) || (item.key === 'notifications' && hasUnreadNotifications) || (item.key === 'moderation' && hasModerationReports)"
                class="h-2 w-2 shrink-0 rounded-full bg-error/100"
                aria-hidden="true"
              />
            </a>
          </div>
          <div v-if="resourceItems.length" class="mt-4 space-y-0.5">
            <div class="px-2 text-[10px] font-bold uppercase tracking-wide text-base-content/55">{{ resourcesLabel }}</div>
            <a
              v-for="item in resourceItems"
              :key="item.key"
              :href="item.url"
              class="flex h-9 items-center gap-2 rounded-md px-2 text-sm font-medium"
              :class="item.active ? 'bg-info/10 text-primary' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
            >
              <component
                :is="sidebarIcon(item)"
                v-if="sidebarIcon(item)"
                class="h-4 w-4 shrink-0"
                aria-hidden="true"
              />
              <span class="min-w-0 flex-1 truncate">{{ item.label }}</span>
            </a>
          </div>
          <div
            v-for="group in sidebarGroups"
            :key="group.key"
            class="mt-4 space-y-0.5"
          >
            <div class="px-2 text-[10px] font-bold uppercase tracking-wide text-base-content/55">{{ group.title }}</div>
            <a
              v-for="item in group.items"
              :key="item.key"
              :href="item.url"
              class="flex h-9 items-center gap-2 rounded-md px-2 text-sm font-medium"
              :class="item.active ? 'bg-info/10 text-primary' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
            >
              <component
                :is="sidebarIcon(item)"
                v-if="sidebarIcon(item)"
                class="h-4 w-4 shrink-0"
                aria-hidden="true"
              />
              <span class="min-w-0 flex-1 truncate">{{ item.label }}</span>
            </a>
          </div>
          <div v-if="categoryItems.length" class="mt-4 space-y-0.5">
            <div class="px-2 text-[10px] font-bold uppercase tracking-wide text-base-content/55">{{ categoriesLabel }}</div>
            <a
              v-for="category in categoryItems"
              :key="category.key"
              :href="category.url"
              class="flex h-9 items-center gap-2 rounded-md px-2 text-sm font-medium"
              :class="category.active ? 'bg-base-300 text-base-content' : 'text-base-content/75 hover:bg-base-300 hover:text-base-content'"
            >
              <span class="h-2 w-2 rounded-[3px]" :style="{ backgroundColor: category.color }" />
              <span class="min-w-0 flex-1 truncate">{{ category.label }}</span>
            </a>
          </div>
          <footer v-if="hasFooter" class="mt-2 border-t border-line px-2 pt-2 text-xs leading-5 text-base-content/75">
            <div v-if="footer.links.length" class="flex flex-wrap items-center gap-x-3 gap-y-0.5">
              <a
                v-for="link in footer.links"
                :key="`${link.name}-${link.url}`"
                :href="link.url"
                class="inline-flex min-h-6 items-center rounded hover:text-primary"
              >
                {{ link.name }}
              </a>
            </div>
            <div v-if="footer.primary.length" class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-base-content/75">
              <span
                v-for="item in footer.primary"
                :key="item"
                class="inline-flex min-h-6 items-center rounded"
              >
                {{ item }}
              </span>
            </div>
          </footer>
        </template>
      </DialogContent>
  </DialogRoot>
</template>
