// @vitest-environment happy-dom
// issue #1124: profile privacy on Web. Owner, other members and guests get
// different copy; hidden sections never render rows; settings only show a
// choice as saved after the server confirms it.
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { ProfileSectionVisibility, UserProfileProps } from '@gooseforum/client'
import { i18n, setLocale } from '../src/runtime/i18n'
import { followUser, getPrivacySettings, updatePrivacySettings, type UserPrivacySettings } from '../src/runtime/api'
import { fetchPage } from '../src/runtime/router'
import { LEGACY_PRIVACY_KEY, readLegacyPrivacy, widensAudience } from '../src/runtime/profile-privacy'
import ProfileSectionPrivacy from '../src/site/components/ProfileSectionPrivacy.vue'
import ProfilePrivacySettings from '../src/site/components/ProfilePrivacySettings.vue'
import SiteSelect from '../src/site/components/SiteSelect.vue'
import UserPage from '../src/site/pages/UserPage.vue'
import { layout, memberProfile } from './fixtures/profile'

vi.mock('../src/runtime/api', async importOriginal => ({
  ...await importOriginal<typeof import('../src/runtime/api')>(),
  getPrivacySettings: vi.fn(),
  updatePrivacySettings: vi.fn(),
  followUser: vi.fn(),
}))
vi.mock('../src/runtime/router', () => ({ fetchPage: vi.fn() }))
vi.mock('../src/runtime/content-updates', () => ({ useContentUpdates: vi.fn() }))

const defaults: UserPrivacySettings = {
  profileTopics: 'everyone', profileActivity: 'everyone', profileLikes: 'everyone', followingList: 'everyone', followersList: 'everyone',
}
let wrapper: ReturnType<typeof mount> | undefined

beforeEach(async () => {
  vi.resetAllMocks()
  localStorage.clear()
  document.body.innerHTML = ''
  await setLocale('zh')
  history.replaceState({}, '', '/u/1024/activity?tab=likes')
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

function section(state: ProfileSectionVisibility, opts: { own?: boolean, authenticated?: boolean } = {}) {
  wrapper = mount(ProfileSectionPrivacy, {
    props: { state, isOwnProfile: opts.own ?? false, authenticated: opts.authenticated ?? true, name: '同济用户', settingsUrl: '/settings' },
    global: { plugins: [i18n] },
  })
  return wrapper
}

describe('ProfileSectionPrivacy', () => {
  test('other members see who the owner shares with, named, without a sign-in link', () => {
    const w = section({ visible: false, reason: 'followers' })
    expect(w.text()).toContain('这部分内容未公开')
    expect(w.text()).toContain('同济用户 只向关注者展示这部分内容，关注后即可查看。')
    expect(w.find('a[href^="/login"]').exists()).toBe(false)
  })

  test('guests get a sign-in link only when signing in can reveal the section', () => {
    let w = section({ visible: false, reason: 'signin' }, { authenticated: false })
    expect(w.text()).toContain('同济用户 只向登录用户展示这部分内容。')
    expect(w.get('a[href^="/login"]').attributes('href')).toBe(`/login?redirect=${encodeURIComponent('/u/1024/activity?tab=likes')}`)
    w.unmount()
    w = section({ visible: false, reason: 'followers' }, { authenticated: false })
    expect(w.text()).toContain('登录并关注后即可查看')
    expect(w.find('a[href^="/login"]').exists()).toBe(true)
    w.unmount()
    for (const reason of ['mutual', 'private'] as const) {
      w = section({ visible: false, reason }, { authenticated: false })
      expect(w.find('a[href^="/login"]').exists(), reason).toBe(false)
      w.unmount()
    }
    wrapper = undefined
  })

  test('the owner sees who can see a restricted section and a link to change it', () => {
    const w = section({ visible: true, audience: 'mutual' }, { own: true })
    expect(w.get('[data-test="profile-section-owner-note"]').text()).toContain('只有和你互相关注的人能看到这部分内容。')
    expect(w.get('a').attributes('href')).toBe('/settings?tab=privacy')
    w.unmount()
    wrapper = section({ visible: true, audience: 'everyone' }, { own: true })
    expect(wrapper.html()).not.toContain('profile-section-owner-note')
  })

  test('copy never uses third-person pronouns', () => {
    for (const reason of ['signin', 'followers', 'mutual', 'private'] as const) {
      const text = section({ visible: false, reason }).text()
      expect(text).not.toMatch(/TA|他|她/)
      wrapper!.unmount()
    }
    wrapper = undefined
  })
})

describe('UserPage visibility', () => {
  function profile(overrides: Partial<UserProfileProps>): UserProfileProps {
    return { ...memberProfile, user: { ...memberProfile.user, isSelf: false }, isOwnProfile: false, ...overrides }
  }
  const global = { plugins: [i18n], stubs: { PrivateNoteEditor: true } }

  test('a hidden activity tab renders the placeholder and no rows, even with stale props', () => {
    const props = profile({
      section: 'activity', activityTab: 'topics',
      visibility: { ...memberProfile.visibility, profileTopics: { visible: false, reason: 'mutual' } },
    })
    wrapper = mount(UserPage, { props: { layout, props }, global })
    expect(wrapper.get('[data-test="profile-section-hidden"]').attributes('data-reason')).toBe('mutual')
    expect(wrapper.text()).not.toContain(memberProfile.topics[0].title)
  })

  test('locked tabs are marked before they are opened', () => {
    const activityTabs = ['timeline', 'topics', 'likes'].map(key => ({ key, label: key, url: key === 'timeline' ? '/u/1024/activity' : `/u/1024/activity/${key}`, active: key === 'timeline' }))
    const props = profile({
      section: 'activity', activityTab: 'timeline', activityTabs,
      visibility: { ...memberProfile.visibility, profileLikes: { visible: false, reason: 'followers' } },
    })
    wrapper = mount(UserPage, { props: { layout, props }, global })
    const locked = wrapper.findAll('a').filter(a => a.find('[data-test="profile-tab-lock"]').exists())
    expect(locked.map(a => a.attributes('href'))).toEqual(['/u/1024/activity/likes'])
    expect(locked[0].text()).toContain('未公开')
    wrapper.unmount()

    // The owner sees which sections have a narrower audience instead, plus
    // bookmarks, which nobody else can ever see.
    const own: UserProfileProps = {
      ...memberProfile, section: 'activity', activityTab: 'timeline', activityTabs,
      visibility: { ...memberProfile.visibility, profileTopics: { visible: true, audience: 'followers' } },
    }
    wrapper = mount(UserPage, { props: { layout, props: own }, global })
    const restricted = wrapper.findAll('a').filter(a => a.find('[data-test="profile-tab-lock"]').exists())
    expect(restricted.map(a => a.attributes('href'))).toEqual(['/u/1024/bookmarks', '/u/1024/activity/topics'])
    expect(restricted.every(a => a.text().includes('已限制可见范围'))).toBe(true)
  })

  test('following reveals a followers-only section without a reload', async () => {
    const like = { id: 7, topicId: 7, title: '只给关注者看的点赞', url: '/p/post/7', likedAt: '2026-10-01T00:00:00Z' }
    const hidden = profile({
      section: 'activity', activityTab: 'likes', canFollow: true, likes: [],
      user: { ...memberProfile.user, isSelf: false, isFollowing: false },
      visibility: { ...memberProfile.visibility, profileLikes: { visible: false, reason: 'followers' } },
    })
    const shown = { ...hidden, likes: [like], user: { ...hidden.user, isFollowing: true }, visibility: { ...hidden.visibility, profileLikes: { visible: true } } }
    vi.mocked(followUser).mockResolvedValue(undefined as never)
    vi.mocked(fetchPage).mockResolvedValueOnce({ props: shown } as never)
    wrapper = mount(UserPage, { props: { layout, props: hidden }, global })
    expect(wrapper.find('[data-test="profile-section-hidden"]').exists()).toBe(true)

    const followButton = () => wrapper!.findAll('button').find(b => /关注/.test(b.text()))!
    await followButton().trigger('click')
    await flushPromises()
    expect(fetchPage).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-test="profile-section-hidden"]').exists()).toBe(false)
    expect(wrapper.text()).toContain(like.title)

    // Unfollowing keeps what was already seen until the next server read.
    await followButton().trigger('click')
    await flushPromises()
    expect(fetchPage).toHaveBeenCalledTimes(1)
  })

  test('following does not re-read the page when nothing is hidden for followers', async () => {
    const open = profile({ section: 'activity', activityTab: 'likes', canFollow: true, user: { ...memberProfile.user, isSelf: false, isFollowing: false } })
    vi.mocked(followUser).mockResolvedValue(undefined as never)
    wrapper = mount(UserPage, { props: { layout, props: open }, global })
    await wrapper.findAll('button').find(b => /关注/.test(b.text()))!.trigger('click')
    await flushPromises()
    expect(followUser).toHaveBeenCalledTimes(1)
    expect(fetchPage).not.toHaveBeenCalled()
  })

  test('the summary hides recent topics behind an inline note', () => {
    const props = profile({ visibility: { ...memberProfile.visibility, profileTopics: { visible: false, reason: 'private' } } })
    wrapper = mount(UserPage, { props: { layout, props }, global })
    expect(wrapper.text()).toContain('同济用户 选择不在个人主页展示这部分内容。')
    expect(wrapper.text()).not.toContain(memberProfile.topics[0].title)
    expect(wrapper.text()).not.toContain('还没有发布内容')
  })

  test('the owner keeps the rows and gets the audience note', () => {
    const props: UserProfileProps = {
      ...memberProfile, section: 'activity', activityTab: 'topics',
      visibility: { ...memberProfile.visibility, profileTopics: { visible: true, audience: 'self' } },
    }
    wrapper = mount(UserPage, { props: { layout, props }, global })
    expect(wrapper.get('[data-test="profile-section-owner-note"]').text()).toContain('只有你能看到这部分内容。')
    expect(wrapper.text()).toContain(memberProfile.topics[0].title)
  })
})

describe('ProfilePrivacySettings', () => {
  async function mountSettings() {
    wrapper = mount(ProfilePrivacySettings, { props: { profileUrl: '/u/1024' }, global: { plugins: [i18n] }, attachTo: document.body })
    await flushPromises()
    return wrapper
  }
  const select = (w: ReturnType<typeof mount>, key: string) => w.get(`[data-test="profile-privacy-row-${key}"]`).findComponent(SiteSelect)

  test('loads server values and offers only the audiences each section supports', async () => {
    let resolve!: (value: UserPrivacySettings) => void
    vi.mocked(getPrivacySettings).mockReturnValue(new Promise(r => { resolve = r }))
    wrapper = mount(ProfilePrivacySettings, { props: { profileUrl: '/u/1024' }, global: { plugins: [i18n] } })
    expect(wrapper.find('[data-test="profile-privacy-loading"]').exists()).toBe(true)
    resolve({ ...defaults, followersList: 'mutual' })
    await flushPromises()
    expect(select(wrapper, 'followersList').props('modelValue')).toBe('mutual')
    expect(select(wrapper, 'profileLikes').props('options').map((o: { value: string }) => o.value)).toEqual(['everyone', 'followers', 'self'])
    expect(select(wrapper, 'followingList').props('options').map((o: { value: string }) => o.value)).toEqual(['everyone', 'followers', 'mutual', 'self'])
    expect(wrapper.get('a[href="/u/1024"]').text()).toBe('查看我的主页')
  })

  test('bookmarks are listed as only-you with nothing to choose', async () => {
    vi.mocked(getPrivacySettings).mockResolvedValue(defaults)
    const w = await mountSettings()
    const row = w.get('[data-test="profile-privacy-row-bookmarks"]')
    expect(row.text()).toContain('收藏')
    expect(row.text()).toContain('始终只有你能看到')
    expect(row.text()).toContain('仅自己')
    expect(row.findComponent(SiteSelect).exists()).toBe(false)
    expect(row.find('button, select, [role="combobox"]').exists()).toBe(false)
  })

  test('a load failure offers retry', async () => {
    vi.mocked(getPrivacySettings).mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(defaults)
    const w = await mountSettings()
    expect(w.get('[role="alert"]').text()).toContain('无法加载个人主页可见范围。')
    await w.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="profile-privacy-row-profileTopics"]').exists()).toBe(true)
  })

  test('narrowing saves immediately and confirms on the row', async () => {
    vi.useFakeTimers()
    vi.mocked(getPrivacySettings).mockResolvedValue(defaults)
    vi.mocked(updatePrivacySettings).mockResolvedValue({ ...defaults, profileLikes: 'self' })
    const w = await mountSettings()
    select(w, 'profileLikes').vm.$emit('update:modelValue', 'self')
    await flushPromises()
    expect(updatePrivacySettings).toHaveBeenCalledWith({ profileLikes: 'self' })
    expect(select(w, 'profileLikes').props('modelValue')).toBe('self')
    // Confirmation stays next to the edited row; no page banner shifts the layout.
    expect(w.get('[data-test="profile-privacy-row-profileLikes"] [data-test="profile-privacy-saved"]').text()).toBe('已保存')
    expect(w.findAll('[data-test="profile-privacy-saved"]')).toHaveLength(1)
    expect(w.get('[data-test="profile-privacy-announcement"]').text()).toBe('可见范围已保存到你的账号')
    vi.advanceTimersByTime(2000)
    await flushPromises()
    expect(w.find('[data-test="profile-privacy-saved"]').exists()).toBe(false)
    vi.useRealTimers()
  })

  test('a failed save keeps the confirmed value and explains next to the row', async () => {
    vi.mocked(getPrivacySettings).mockResolvedValue(defaults)
    vi.mocked(updatePrivacySettings).mockRejectedValue(new Error('offline'))
    const w = await mountSettings()
    select(w, 'profileTopics').vm.$emit('update:modelValue', 'followers')
    await flushPromises()
    expect(select(w, 'profileTopics').props('modelValue')).toBe('everyone')
    expect(w.get('[data-test="profile-privacy-row-profileTopics"] [role="alert"]').text()).toContain('未能保存')
    expect(w.find('[data-test="profile-privacy-saved"]').exists()).toBe(false)
    expect(w.get('[data-test="profile-privacy-announcement"]').text()).toBe('')
  })

  test('widening asks first; cancel sends nothing, confirm saves', async () => {
    vi.mocked(getPrivacySettings).mockResolvedValue({ ...defaults, followingList: 'self' })
    vi.mocked(updatePrivacySettings).mockResolvedValue(defaults)
    const w = await mountSettings()
    select(w, 'followingList').vm.$emit('update:modelValue', 'everyone')
    await flushPromises()
    const dialog = document.querySelector('[data-test="profile-privacy-widen"]')!
    expect(dialog.textContent).toContain('「关注列表」将改为「所有人」可见')
    expect(document.activeElement?.textContent?.trim()).toBe('取消')
    ;(document.activeElement as HTMLButtonElement).click()
    await flushPromises()
    expect(updatePrivacySettings).not.toHaveBeenCalled()
    expect(document.querySelector('[data-test="profile-privacy-widen"]')).toBeNull()

    select(w, 'followingList').vm.$emit('update:modelValue', 'everyone')
    await flushPromises()
    ;[...document.querySelectorAll<HTMLButtonElement>('[data-test="profile-privacy-widen"] button')].find(b => b.textContent?.includes('扩大可见范围'))!.click()
    await flushPromises()
    expect(updatePrivacySettings).toHaveBeenCalledWith({ followingList: 'everyone' })
    expect(select(w, 'followingList').props('modelValue')).toBe('everyone')
  })

  test('legacy browser-only switches migrate only on request', async () => {
    localStorage.setItem(LEGACY_PRIVACY_KEY, JSON.stringify({ showTopics: false, showFollowing: true, emailNotifications: true }))
    vi.mocked(getPrivacySettings).mockResolvedValue(defaults)
    vi.mocked(updatePrivacySettings).mockResolvedValue({ ...defaults, profileTopics: 'self', profileActivity: 'self', profileLikes: 'self' })
    const w = await mountSettings()
    expect(updatePrivacySettings).not.toHaveBeenCalled()
    const legacy = w.get('[data-test="profile-privacy-legacy"]')
    await legacy.findAll('button').find(b => b.text() === '应用到我的账号')!.trigger('click')
    await flushPromises()
    expect(updatePrivacySettings).toHaveBeenCalledWith({ profileTopics: 'self', profileActivity: 'self', profileLikes: 'self' })
    expect(localStorage.getItem(LEGACY_PRIVACY_KEY)).toBeNull()
    expect(w.find('[data-test="profile-privacy-legacy"]').exists()).toBe(false)
  })

  test('dismissing the legacy notice clears it without saving; an all-on record is dropped silently', async () => {
    localStorage.setItem(LEGACY_PRIVACY_KEY, JSON.stringify({ showFollowing: false }))
    vi.mocked(getPrivacySettings).mockResolvedValue(defaults)
    let w = await mountSettings()
    await w.get('[data-test="profile-privacy-legacy"]').findAll('button').find(b => b.text() === '忽略')!.trigger('click')
    expect(localStorage.getItem(LEGACY_PRIVACY_KEY)).toBeNull()
    expect(updatePrivacySettings).not.toHaveBeenCalled()
    w.unmount()
    localStorage.setItem(LEGACY_PRIVACY_KEY, JSON.stringify({ showTopics: true, showFollowing: true }))
    w = await mountSettings()
    expect(w.find('[data-test="profile-privacy-legacy"]').exists()).toBe(false)
    expect(localStorage.getItem(LEGACY_PRIVACY_KEY)).toBeNull()
  })
})

test('profile-privacy helpers rank audiences and map legacy switches', () => {
  expect(widensAudience('self', 'mutual')).toBe(true)
  expect(widensAudience('members', 'followers')).toBe(false)
  expect(readLegacyPrivacy({ getItem: () => '{"showFollowing":false}' })).toEqual({ followingList: 'self', followersList: 'self' })
  expect(readLegacyPrivacy({ getItem: () => 'not json' })).toBeNull()
})
