<script lang="ts" setup>
import { adminText } from '@/admin/runtime/i18n-text'
import { ExternalLink, GalleryVerticalEnd } from '@lucide/vue'
import { computed } from 'vue'
import { Avatar, AvatarFallback, AvatarImage } from '@/admin/components/ui/avatar'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from '@/admin/components/ui/sidebar'
import { RouterLink, useRoute } from 'vue-router'
import type { LayoutPayload } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'
import { adminNavGroups, type NavGroup, type NavItem } from '@/admin/runtime/navigation'

defineProps<{
  layout: LayoutPayload
}>()

const route = useRoute()
const { locale } = useI18n()
const currentPath = computed(() => route.path.replace(/\/+$/, '') || '/admin')

const navGroups = computed<NavGroup[]>(() => {
  void locale.value
  return adminNavGroups()
})

function isActive(item: NavItem) {
  return currentPath.value === item.url
}
</script>

<template>
  <Sidebar collapsible="icon" class="z-50">
    <SidebarHeader>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            as-child
            size="lg"
            class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
          >
            <a href="/" target="_blank" rel="noopener noreferrer" :title="adminText('k007s')">
              <div class="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <GalleryVerticalEnd class="size-4" />
              </div>
              <div class="grid flex-1 text-left text-sm leading-tight">
                <span class="truncate font-semibold">{{ layout.site.brandText || layout.site.name || 'YourTJHub' }}</span>
                <span class="truncate text-xs">Admin</span>
              </div>
            </a>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarHeader>

    <SidebarContent class="gap-0">
      <SidebarGroup v-for="group in navGroups" :key="group.title" class="py-1">
        <SidebarGroupLabel class="h-7">{{ group.title }}</SidebarGroupLabel>
        <SidebarMenu>
          <SidebarMenuItem v-for="item in group.items" :key="item.title">
            <SidebarMenuButton as-child :is-active="isActive(item)" :tooltip="item.title">
              <a
                v-if="item.external"
                :href="item.url"
                target="_blank"
                rel="noopener noreferrer"
              >
                <component :is="item.icon" />
                <span>{{ item.title }}</span>
                <ExternalLink class="ml-auto size-4" />
              </a>
              <RouterLink v-else :to="item.url">
                <component :is="item.icon" />
                <span>{{ item.title }}</span>
              </RouterLink>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarGroup>
    </SidebarContent>

    <SidebarFooter>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton size="lg">
            <Avatar class="size-8 rounded-lg">
              <AvatarImage :src="layout.viewer.avatarUrl" :alt="layout.viewer.username || 'admin'" />
              <AvatarFallback class="rounded-lg">CN</AvatarFallback>
            </Avatar>
            <div class="grid flex-1 text-left text-sm leading-tight">
              <span class="truncate font-semibold">{{ layout.viewer.username || 'shadcn' }}</span>
              <span class="truncate text-xs">{{ layout.viewer.email || adminText('k007x') }}</span>
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarFooter>

    <SidebarRail />
  </Sidebar>
</template>
