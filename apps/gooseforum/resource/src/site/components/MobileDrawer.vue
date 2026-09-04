<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { X } from '@lucide/vue'
import { DialogContent, DialogOverlay, DialogRoot, DialogTitle } from 'reka-ui'
import type { FooterPayload, WikiTreeNamespace } from '@gooseforum/client'
import WikiSidebar from './WikiSidebar.vue'
import { safeUrl } from '@/runtime/safe-url'

interface SidebarNavItem {
  key: string
  label: string
  i18nLabel?: string
  url: string
  active: boolean
}

interface SidebarGroupItem {
  key: string
  title: string
  i18nLabel?: string
  items: SidebarNavItem[]
}
interface SidebarCategoryItem extends SidebarNavItem {
  id: number
  color: string
}


interface SidebarSection {
  key: string
  title?: string
  items: SidebarNavItem[]
}

const props = defineProps<{
  open: boolean
  sections: SidebarSection[]
  resourceItems: SidebarNavItem[]
  sidebarGroups: SidebarGroupItem[]
  categoryItems: SidebarCategoryItem[]
  footer: FooterPayload
  /** wiki 模式：抽屉顶部展示完整 wiki 导航树（首页/命名空间/页面），与桌面侧栏一致。 */
  wikiMode?: boolean
  wikiTree?: WikiTreeNamespace[]
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

const footerLinks = computed(() => props.footer.links.map((link) => ({ ...link, url: safeUrl(link.url, 'site-link') })))
const footerPrimary = computed(() => props.footer.primary)
const hasFooter = computed(() => footerLinks.value.length > 0 || footerPrimary.value.length > 0)

function close() {
  emit('close')
}

// 抽屉仅在小屏（<lg）展示。resize 到 lg 后遮罩/抽屉被 lg:hidden 隐藏，但
// DialogRoot 仍 mounted/open——reka-ui 会继续 trap focus 并锁定 body 滚动，
// 页面会变得不可点。监听断点变化，进入 lg 时自动关闭 drawer。
let desktopQuery: MediaQueryList | null = null
let desktopChangeHandler: ((event: MediaQueryListEvent) => void) | null = null

onMounted(() => {
  if (typeof window.matchMedia !== 'function') return
  desktopQuery = window.matchMedia('(min-width: 1024px)')
  desktopChangeHandler = (event) => {
    if (event.matches && props.open) close()
  }
  // addEventListener 优于 addListener（happy-dom 两者都支持，真实浏览器以 addEventListener 为准）
  desktopQuery.addEventListener('change', desktopChangeHandler)
  // 挂载时可能已处于桌面断点（异步挂载/打开后立即旋转或调整窗口）：change 事件不会触发，
  // 但 Dialog 仍 open 且 lg:hidden 只隐藏内容，模态锁残留。立即处理初始 matches。
  if (desktopQuery.matches && props.open) close()
})

onBeforeUnmount(() => {
  if (desktopQuery && desktopChangeHandler) {
    desktopQuery.removeEventListener('change', desktopChangeHandler)
  }
})
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
          <button class="inline-flex h-10 w-10 items-center justify-center rounded-full text-icon-muted transition-colors duration-150 hover:bg-base-200 hover:text-base-content active:scale-[0.96] motion-reduce:active:scale-100" type="button" :aria-label="closeLabel" @click="close">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div v-if="wikiMode" class="pb-2">
          <WikiSidebar :tree="wikiTree || []" @navigate="close" />
        </div>
        <template v-else>
          <nav :aria-label="menuLabel">
            <template v-for="(section, sectionIndex) in sections" :key="section.key">
              <div
                v-if="section.title"
                class="mb-1 px-2 text-[10px] font-bold uppercase tracking-wide text-base-content/55"
                :class="sectionIndex === 0 ? 'pt-0' : 'mt-4'"
              >
                {{ section.title }}
              </div>
              <div class="space-y-0.5">
                <a
                  v-for="item in section.items"
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
                    v-if="section.key === 'admin' && item.key === 'moderation' && hasModerationReports"
                    class="h-2 w-2 shrink-0 rounded-full bg-error"
                    aria-hidden="true"
                  />
                </a>
              </div>
            </template>
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
                <span class="h-2 w-2 shrink-0 rounded-[3px]" :style="{ backgroundColor: category.color }" />
                <span class="min-w-0 flex-1 truncate">{{ category.label }}</span>
              </a>
            </div>
          </nav>
          <footer v-if="hasFooter" class="mt-2 border-t border-line px-2 pt-2 text-xs leading-5 text-base-content/75">
            <div v-if="footerLinks.length" class="flex flex-wrap items-center gap-x-3 gap-y-0.5">
              <template v-for="link in footerLinks" :key="`${link.name}-${link.url}`">
                <a
                  v-if="link.url"
                  :href="link.url"
                  class="inline-flex min-h-6 items-center rounded hover:text-primary"
                >
                  {{ link.name }}
                </a>
                <span v-else class="inline-flex min-h-6 items-center rounded">{{ link.name }}</span>
              </template>
            </div>
            <div v-if="footerPrimary.length" class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-base-content/75">
              <span
                v-for="item in footerPrimary"
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
