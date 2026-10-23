// @vitest-environment happy-dom
// issue #1320: 个人主页区分「已关注」与「已互关」。主页按钮与关注列表条目
// 都只渲染服务端给出的 isMutualFollowing，取关后回落为「关注」。
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { UserConnectionPayload, UserProfileProps } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import { followUser } from '../src/runtime/api'
import UserPage from '../src/site/pages/UserPage.vue'
import { layout, memberProfile } from './fixtures/profile'

vi.mock('../src/runtime/api', async importOriginal => ({
  ...await importOriginal<typeof import('../src/runtime/api')>(),
  followUser: vi.fn(),
}))
vi.mock('../src/runtime/router', () => ({ fetchPage: vi.fn() }))
vi.mock('../src/runtime/content-updates', () => ({ useContentUpdates: vi.fn() }))

const t = (key: string) => i18n.global.t(key)

function connection(overrides: Partial<UserConnectionPayload> & Pick<UserConnectionPayload, 'id' | 'username'>): UserConnectionPayload {
  return {
    nickname: overrides.username,
    avatarUrl: '',
    bio: '',
    url: `/u/${overrides.id}`,
    isFollowing: true,
    isMutualFollowing: false,
    isSelf: false,
    ...overrides,
  }
}

function mutualProfile(): UserProfileProps {
  // 条目必须每次新建：连接行取关会原地改写 connection 对象，共享常量会被后续用例污染。
  const mutualEntry = connection({ id: 7, username: 'mutual_peer', isFollowing: true, isMutualFollowing: true })
  const oneWayEntry = connection({ id: 8, username: 'oneway_peer', isFollowing: true, isMutualFollowing: false })
  return {
    ...memberProfile,
    isOwnProfile: false,
    canFollow: true,
    section: 'following',
    user: { ...memberProfile.user, isSelf: false, isFollowing: true, isMutualFollowing: true },
    following: [mutualEntry, oneWayEntry],
    followers: [],
  }
}

let wrapper: ReturnType<typeof mount> | undefined

beforeEach(() => {
  vi.mocked(followUser).mockReset()
  history.replaceState({}, '', '/u/1024/following')
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

// header（含主页关注按钮）只在非独立关注列表分区渲染；列表条目在 section=following。
function mountPage(overrides: Partial<UserProfileProps> = {}) {
  wrapper = mount(UserPage, {
    props: { layout, props: { ...mutualProfile(), ...overrides } },
    global: { plugins: [i18n], stubs: { PrivateNoteEditor: true } },
  })
  return wrapper
}

// 主页关注按钮是 gf-button-md；列表条目按钮是 gf-button-sm。
function pageFollowButton() {
  return wrapper!.findAll('button').find((button) => button.classes().includes('gf-button-md'))!
}

function connectionRow(username: string) {
  return wrapper!.findAll('div.divide-y > div').find((row) => row.text().includes(`@${username}`))!
}

describe('UserPage 互关状态（issue #1320）', () => {
  test('互关用户的主页按钮显示已互关并换成 UserRoundCheck 图标', () => {
    mountPage({ section: 'summary' })

    const button = pageFollowButton()
    expect(button.text()).toBe(t('user.mutualFollowing'))
    expect(button.get('svg').attributes('class')).toContain('user-round-check')
  })

  test('关注列表条目按服务端互关位分别显示已互关与已关注', () => {
    mountPage()

    const mutualButton = connectionRow('mutual_peer').get('button')
    expect(mutualButton.text()).toBe(t('user.mutualFollowing'))
    expect(mutualButton.get('svg').attributes('class')).toContain('user-round-check')

    const oneWayButton = connectionRow('oneway_peer').get('button')
    expect(oneWayButton.text()).toBe(t('user.following'))
    expect(oneWayButton.get('svg').attributes('class')).toContain('user-plus')
  })

  test('取消关注后主页按钮回到关注', async () => {
    vi.mocked(followUser).mockResolvedValue({ isFollowing: false, isMutualFollowing: false })
    mountPage({ section: 'summary' })

    await pageFollowButton().trigger('click')
    await flushPromises()

    expect(followUser).toHaveBeenCalledWith(1024, true)
    expect(pageFollowButton().text()).toBe(t('user.follow'))
    expect(pageFollowButton().get('svg').attributes('class')).toContain('user-plus')
  })

  test('互关的连接行取关后回落为关注并撤销互关图标', async () => {
    vi.mocked(followUser).mockResolvedValue({ isFollowing: false, isMutualFollowing: false })
    mountPage()

    await connectionRow('mutual_peer').get('button').trigger('click')
    await flushPromises()

    expect(followUser).toHaveBeenCalledWith(7, true)
    const button = connectionRow('mutual_peer').get('button')
    expect(button.text()).toBe(t('user.follow'))
    expect(button.get('svg').attributes('class')).toContain('user-plus')
  })
})
