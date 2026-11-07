// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, test } from 'vitest'
import type { LayoutPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import AppShell from '../src/site/components/AppShell.vue'
import { useUnreadStatus } from '../src/runtime/unread-status'

function layout(viewer: Partial<LayoutPayload['viewer']> = {}): LayoutPayload {
  return {
    site: { name: 'yourtj', description: '', logo: '', favicon: '', brandType: 'text', brandText: 'yourtj', brandImage: '' },
    viewer: {
      id: 42,
      username: 'menu-tester',
      email: '',
      avatarUrl: '',
      isAuthenticated: true,
      canAccessAdmin: false,
      isModerator: false,
      requiresEmailVerification: false,
      adminPermissions: [],
      ...viewer,
    },
    sidebar: { main: [], resources: [], groups: [], categories: [], activeKey: 'topics' },
    footer: { links: [], primary: [] },
    unread: { notifications: false, messages: false, moderationReports: false },
    theme: { enabled: false, current: 'gf-light', themeColor: '#fbfdff' },
    umamiEnabled: false,
  }
}

async function openUserMenu(viewer: Partial<LayoutPayload['viewer']> = {}) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
  const wrapper = mount(AppShell, {
    props: { layout: layout(viewer) },
    global: { plugins: [i18n, router], stubs: { AnonymousProfileLink: true } },
  })
  await wrapper.get(`button[aria-label="${i18n.global.t('shell.userMenu')}"]`).trigger('focusin')
  await flushPromises()
  return wrapper
}

describe('AppShell user menu groups', () => {
  test('keeps drafts with personal links and separates the three menu groups', async () => {
    const wrapper = await openUserMenu()
    const menu = wrapper.get('[data-user-menu]')
    const groups = menu.findAll('[data-user-menu-group]')

    expect(groups.map(group => group.attributes('data-user-menu-group'))).toEqual([
      'personal',
      'preferences',
      'settings',
    ])

    const personalLinks = groups[0]!.findAll('a').map(link => link.attributes('href'))
    expect(personalLinks).toContain('/u/42')
    expect(personalLinks).toContain('/notifications')
    expect(personalLinks).toContain('/messages')
    expect(personalLinks).toContain('/drafts')
    expect(personalLinks.indexOf('/drafts')).toBeGreaterThan(personalLinks.indexOf('/messages'))

    expect(groups[0]!.classes()).not.toContain('border-t')
    expect(groups[1]!.classes()).toContain('border-t')
    expect(groups[1]!.classes()).toContain('sm:hidden')
    expect(groups[2]!.classes()).toContain('border-t')
    expect(groups[1]!.findAll('button').length).toBeGreaterThan(0)
    expect(groups[2]!.find('a[href="/settings"]').exists()).toBe(true)

    wrapper.unmount()
  })

  test('retains agent and admin visibility conditions', async () => {
    const restricted = await openUserMenu()
    const restrictedMenu = restricted.get('[data-user-menu]')
    expect(restrictedMenu.find('a[href="/admin/agents"]').exists()).toBe(false)
    expect(restrictedMenu.find('a[href="/theme-preview"]').exists()).toBe(false)
    expect(restrictedMenu.find('a[href="/admin"]').exists()).toBe(false)
    restricted.unmount()

    const privileged = await openUserMenu({ canManageAgents: true, canAccessAdmin: true })
    const privilegedMenu = privileged.get('[data-user-menu]')
    expect(privilegedMenu.find('a[href="/admin/agents"]').exists()).toBe(true)
    expect(privilegedMenu.find('a[href="/theme-preview"]').exists()).toBe(true)
    expect(privilegedMenu.find('a[href="/admin"]').exists()).toBe(true)
    privileged.unmount()
  })
})


test('a fresh zero-count navigation payload updates the shell badge without waiting for a poll', async () => {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }] })
  const initial = layout()
  initial.unread = { notifications: true, messages: false, notificationCount: 3 }
  const wrapper = mount(AppShell, { props: { layout: initial }, global: { plugins: [i18n, router], stubs: { AnonymousProfileLink: true } } })
  const status = useUnreadStatus()
  try {
    expect(status.notificationCount.value).toBe(3)
    await wrapper.setProps({ layout: { ...initial, unread: { notifications: false, messages: false, notificationCount: 0 } } })
    expect(status.notificationCount.value).toBe(0)
    expect(status.notifications.value).toBe(false)
    expect(JSON.parse(localStorage.getItem('goose:unread-status:42')!).data.notificationCount).toBe(0)
  } finally {
    wrapper.unmount()
    status.stopPolling(true)
  }
})
