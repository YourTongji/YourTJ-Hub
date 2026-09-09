// @vitest-environment happy-dom
// issue #530：设置页 canSetPassword 分支 UI 生效的组件级验收。
// canSetPassword=true（无邮箱 OAuth 绑定账号，服务端门禁）→ 不再渲染
// 「当前密码」输入框，改为 set-password 提示文案 + 「设置密码」按钮；
// 提交走 setPassword 并跳转登录页（会话已全端吊销）。
import { describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import type { LayoutPayload, SettingsPageProps } from '@gooseforum/client'

// 设置页 onMounted 会拉取绑定/会话/TOTP 状态并初始化 Web Push / 浏览器通知；
// 统一 mock 为安静值，聚焦 canSetPassword 表单分支本身。
vi.mock('../src/runtime/api', () => ({
  ApiResponseError: class ApiResponseError extends Error {},
  batchDeleteContent: vi.fn(async () => true),
  changePassword: vi.fn(async () => true),
  closeAccount: vi.fn(async () => true),
  disableTotp: vi.fn(async () => true),
  enableTotp: vi.fn(async () => ({ recoveryCodes: [] })),
  getDeletedContent: vi.fn(async () => ({ items: [], hasMore: false, nextCursorId: 0 })),
  getMyContent: vi.fn(async () => ({ items: [], hasMore: false, nextCursorId: 0 })),
  getOAuthBindings: vi.fn(async () => ({})),
  getTotpSetup: vi.fn(async () => ({ otpauthUrl: '' })),
  getTotpStatus: vi.fn(async () => ({ enabled: false })),
  listSessions: vi.fn(async () => []),
  logout: vi.fn(async () => true),
  purgeDeletedContent: vi.fn(async () => true),
  resendActivationEmail: vi.fn(async () => ''),
  restoreDeletedContent: vi.fn(async () => true),
  revokeAllSessions: vi.fn(async () => true),
  revokeSession: vi.fn(async () => true),
  savePresetAvatar: vi.fn(async () => true),
  saveUserEmail: vi.fn(async () => ''),
  saveUserInfo: vi.fn(async () => ''),
  saveUserName: vi.fn(async () => ''),
  saveUserProfileCover: vi.fn(async () => ''),
  sensitiveWordsFromError: vi.fn(async () => []),
  setPassword: vi.fn(async () => true),
  unbindOAuth: vi.fn(async () => true),
  uploadAvatar: vi.fn(async () => ''),
  uploadImageFile: vi.fn(async () => ''),
  wearBadge: vi.fn(async () => true),
}))
vi.mock('../src/runtime/web-push', () => ({
  PushError: class PushError extends Error {},
  currentPushSubscription: vi.fn(async () => null),
  disableWebPush: vi.fn(async () => true),
  enableWebPush: vi.fn(async () => true),
  prepareWebPush: vi.fn(() => null),
  rebindPushSubscription: vi.fn(async () => true),
}))
vi.mock('../src/runtime/browser-notification', () => ({
  disableBrowserNotifications: vi.fn(),
  enableBrowserNotifications: vi.fn(async () => true),
  isBrowserNotificationEnabled: vi.fn(() => false),
  isBrowserNotificationSupported: vi.fn(() => false),
}))
vi.mock('../src/runtime/site-theme', async () => {
  const { ref } = await import('vue')
  return {
    useSiteTheme: () => ({ preference: ref('auto'), setPreference: vi.fn() }),
  }
})
vi.mock('../src/runtime/flash-message', () => ({
  useFlashMessages: () => ({ push: vi.fn() }),
}))

import { changePassword, setPassword } from '../src/runtime/api'
import SettingsPage from '../src/site/pages/SettingsPage.vue'

const layout: LayoutPayload = {
  site: {
    name: 'yourtj',
    description: '',
    logo: '',
    favicon: '',
    brandType: 'text',
    brandText: 'yourtj',
    brandImage: '',
  },
  viewer: {
    id: 7,
    username: 'oauth-user',
    email: '',
    avatarUrl: '/static/pic/1.webp',
    isAuthenticated: true,
    canAccessAdmin: false,
    isModerator: false,
    requiresEmailVerification: false,
    adminPermissions: [],
  },
  sidebar: { categories: [], activeKey: 'settings' },
  footer: { links: [], primary: [] },
  unread: { notifications: false, messages: false },
  posting: { maxTitleLength: 100 },
  theme: { enabled: false, current: 'gf-light', themeColor: '' },
  insightFlareEnabled: false,
}

function buildProps(canSetPassword: boolean): SettingsPageProps {
  return {
    user: {
      id: 7,
      username: 'oauth-user',
      email: '',
      nickname: '',
      locale: 'zh',
      avatarUrl: '/static/pic/1.webp',
      profileCoverUrl: '',
      bio: '',
      signature: '',
      websiteName: '',
      website: '',
      prestige: 0,
      createdAt: '2026-01-01T00:00:00Z',
      externalInformation: {},
      wornBadgeCode: '',
      badges: [],
      wearableBadges: [],
      wornBadge: null,
    },
    googleOAuthReady: false,
    canSetPassword,
    stats: {
      topicCount: 0,
      replyCount: 0,
      followerCount: 0,
      followingCount: 0,
      likeReceivedCount: 0,
      likeGivenCount: 0,
      collectionCount: 0,
      createdAt: '2026-01-01T00:00:00Z',
    },
    tabs: [
      { key: 'profile', url: '/settings', active: true },
      { key: 'account', url: '/settings?tab=account', active: false },
      { key: 'privacy', url: '/settings?tab=privacy', active: false },
      { key: 'binding', url: '/settings?tab=binding', active: false },
      { key: 'security', url: '/settings?tab=security', active: false },
      { key: 'content', url: '/settings?tab=content', active: false },
      { key: 'deleted', url: '/settings?tab=deleted', active: false },
      { key: 'general', url: '/settings?tab=general', active: false },
    ],
  }
}

function mountPage(canSetPassword: boolean) {
  return mount(SettingsPage, {
    props: { layout, props: buildProps(canSetPassword) },
    global: {
      plugins: [i18n],
      stubs: {
        UserAvatar: true,
        AvatarImageEditor: true,
        CoverImageEditor: true,
        SectionHeader: true,
        SiteSelect: true,
        teleport: true,
      },
    },
    attachTo: document.body,
  })
}

// 定位「设置/修改密码」表单：含 auth.newPassword 标签的那个 form
// （账号页同一分区还有 email 表单，security 分区是 TOTP 表单）。
function findPasswordForm(wrapper: ReturnType<typeof mountPage>) {
  const newPasswordLabel = i18n.global.t('auth.newPassword')
  return wrapper.findAll('form').find((item) => item.text().includes(newPasswordLabel))
}

describe('SettingsPage canSetPassword 分支（issue #530）', () => {
  test('canSetPassword=true：隐藏当前密码框，展示 set-password 提示与「设置密码」按钮', () => {
    const wrapper = mountPage(true)
    const form = findPasswordForm(wrapper)
    expect(form).toBeTruthy()

    const hint = i18n.global.t('settings.account.setPasswordHint')
    const currentLabel = i18n.global.t('settings.account.currentPassword')
    expect(form!.text()).toContain(hint)
    expect(form!.text()).not.toContain(currentLabel)

    // 只有 新密码 + 确认密码 两个输入框（没有当前密码）
    const inputs = form!.findAll('input[type="password"]')
    expect(inputs).toHaveLength(2)

    const submit = form!.find('button[type="submit"]')
    expect(submit.text()).toContain(i18n.global.t('settings.account.setPassword'))
    expect(submit.text()).not.toContain(i18n.global.t('settings.account.changePassword'))
    wrapper.unmount()
  })

  test('canSetPassword=false：保留当前密码框与「修改密码」按钮（普通改密路径不回归）', () => {
    const wrapper = mountPage(false)
    const form = findPasswordForm(wrapper)
    expect(form).toBeTruthy()

    expect(form!.text()).not.toContain(i18n.global.t('settings.account.setPasswordHint'))
    expect(form!.text()).toContain(i18n.global.t('settings.account.currentPassword'))
    expect(form!.findAll('input[type="password"]')).toHaveLength(3)

    const submit = form!.find('button[type="submit"]')
    expect(submit.text()).toContain(i18n.global.t('settings.account.changePassword'))
    wrapper.unmount()
  })

  test('canSetPassword=true 提交时调用 setPassword（免旧密码），不调用 changePassword', async () => {
    const wrapper = mountPage(true)
    const form = findPasswordForm(wrapper)
    expect(form).toBeTruthy()

    const inputs = form!.findAll('input[type="password"]')
    await inputs[0].setValue('brand-new-pw1')
    await inputs[1].setValue('brand-new-pw1')
    // 成功即 TokenVersion++ 全端吊销 → 组件跳转 /login；断言聚焦在
    // 「走了 set-password 分支」本身（window.location 赋值在 happy-dom 下安全）。
    await form!.find('button[type="submit"]').trigger('submit')
    await flushPromises()

    expect(vi.mocked(setPassword)).toHaveBeenCalledWith('brand-new-pw1')
    expect(vi.mocked(changePassword)).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
