// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import { getIdentityState, type IdentityState } from '../src/runtime/anonymous-identity'
import AnonymousProfilePage from '../src/site/pages/AnonymousProfilePage.vue'
import UserPage from '../src/site/pages/UserPage.vue'
import ProfileHeader from '../src/site/components/ProfileHeader.vue'
import ProfileManageButton from '../src/site/components/ProfileManageButton.vue'
import AnonymousProfileLink from '../src/site/components/AnonymousProfileLink.vue'
import { anonymousProfile, identityState, layout, memberProfile } from './fixtures/profile'
vi.mock('../src/runtime/anonymous-identity', async (importOriginal) => ({ ...await importOriginal<typeof import('../src/runtime/anonymous-identity')>(), getIdentityState: vi.fn() }))
vi.mock('../src/runtime/content-updates', () => ({ useContentUpdates: vi.fn() }))
let wrapper: ReturnType<typeof mount>
beforeEach(async () => {
  vi.resetAllMocks(); await setLocale('zh'); history.replaceState({}, '', '/a/' + 'a'.repeat(32))
  vi.mocked(getIdentityState).mockResolvedValue(structuredClone(identityState))
})
afterEach(() => wrapper?.unmount())
const global = { plugins: [i18n], stubs: { AnonymousIdentityDialog: true, PrivateNoteEditor: true } }
test('member and persona reuse the header and owner management control', async () => {
  wrapper = mount(UserPage, { props: { layout, props: memberProfile }, global })
  expect(wrapper.findComponent(ProfileHeader).exists()).toBe(true)
  expect(wrapper.findComponent(ProfileManageButton).props('href')).toBe('/settings')
  wrapper.unmount()
  wrapper = mount(AnonymousProfilePage, { props: { layout, props: anonymousProfile }, global })
  await flushPromises()
  expect(wrapper.findComponent(ProfileHeader).exists()).toBe(true)
  await wrapper.findComponent(ProfileManageButton).trigger('click')
  expect(wrapper.findComponent({ name: 'AnonymousIdentityDialog' }).attributes('open')).toBe('true')
  expect(wrapper.text()).not.toContain(memberProfile.user.signature)
  expect(wrapper.find('a[href^="/u/"]').exists()).toBe(false)
})
test('guests do not fetch private state or get management controls', async () => {
  wrapper = mount(AnonymousProfilePage, { props: { layout: { ...layout, viewer: { ...layout.viewer, isAuthenticated: false } }, props: anonymousProfile }, global })
  await flushPromises()
  expect(getIdentityState).not.toHaveBeenCalled()
  expect(wrapper.findComponent(ProfileManageButton).exists()).toBe(false)
})
test('another signed-in persona does not get management controls', async () => {
  vi.mocked(getIdentityState).mockResolvedValue({ ...identityState, persona: { ...identityState.persona, publicUid: 'b'.repeat(32) } })
  wrapper = mount(AnonymousProfilePage, { props: { layout, props: anonymousProfile }, global })
  await flushPromises()
  expect(wrapper.findComponent(ProfileManageButton).exists()).toBe(false)
})
test('logout discards a pending private ownership read', async () => {
  let resolve!: (state: IdentityState) => void
  vi.mocked(getIdentityState).mockReturnValue(new Promise(done => { resolve = done }))
  wrapper = mount(AnonymousProfilePage, { props: { layout, props: anonymousProfile }, global })
  await wrapper.setProps({ layout: { ...layout, viewer: { ...layout.viewer, isAuthenticated: false } } })
  resolve(structuredClone(identityState)); await flushPromises()
  expect(wrapper.findComponent(ProfileManageButton).exists()).toBe(false)
})
test('reply tab uses independent counts and retains its paging target', async () => {
  history.replaceState({}, '', '/a/' + 'a'.repeat(32) + '?tab=replies&page=2')
  wrapper = mount(AnonymousProfilePage, { props: { layout, props: { ...anonymousProfile, page: 2, topicCount: 0, replyCount: 41 } }, global })
  await flushPromises()
  expect(wrapper.text()).toContain(anonymousProfile.replies[0].excerpt)
  expect(wrapper.find(`a[href="${anonymousProfile.persona.profileUrl}?tab=replies&page=3"]`).exists()).toBe(true)
})
test('ordinary user menu enters the persona profile or setup', async () => {
  wrapper = mount(AnonymousProfileLink, { props: { viewerId: 1024 }, global })
  await flushPromises()
  expect(wrapper.get('a').attributes('href')).toBe(identityState.persona.profileUrl)
  vi.mocked(getIdentityState).mockResolvedValue({ ...identityState, persona: null })
  await wrapper.setProps({ viewerId: 2048 }); await flushPromises()
  expect(wrapper.get('a').attributes('href')).toBe('/settings?tab=privacy')
})
