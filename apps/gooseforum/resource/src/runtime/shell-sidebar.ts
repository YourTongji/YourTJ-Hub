import { ref } from 'vue'

const sidebarCollapsedStorageKey = 'goose:shell-sidebar-collapsed'

function readSidebarCollapsed(): boolean {
  if (typeof window === 'undefined') return false

  try {
    return window.localStorage.getItem(sidebarCollapsedStorageKey) === '1'
  } catch {
    // Storage may be unavailable in private or restricted browsing contexts.
    return false
  }
}

// 侧栏折叠是跨页面的外观偏好：SPA 导航复用同一个 AppShell 实例，
// 但登录/重置密码/校园地图这类独立页往返会重建组件，因此状态放在模块级别
// （与 home-feed-mode 同一模式）。
const sharedSidebarCollapsed = ref(readSidebarCollapsed())

function setSidebarCollapsed(collapsed: boolean) {
  sharedSidebarCollapsed.value = collapsed
  if (typeof window === 'undefined') return

  try {
    window.localStorage.setItem(sidebarCollapsedStorageKey, collapsed ? '1' : '0')
  } catch {
    // Storage may be unavailable in private or restricted browsing contexts.
  }
}

function toggleSidebar() {
  setSidebarCollapsed(!sharedSidebarCollapsed.value)
}

// 测试用：把共享状态重读到当前 localStorage（与 resetShellState 同一约定）。
export function resetShellSidebar() {
  sharedSidebarCollapsed.value = readSidebarCollapsed()
}

export function useShellSidebar() {
  return {
    sidebarCollapsed: sharedSidebarCollapsed,
    setSidebarCollapsed,
    toggleSidebar,
  }
}
