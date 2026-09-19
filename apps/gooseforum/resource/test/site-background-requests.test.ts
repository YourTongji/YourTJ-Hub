// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  fetchPage: vi.fn(async () => ({ layout: { umamiEnabled: false } })),
  navigate: vi.fn(),
  payload: { component: 'home.index', layout: { umamiEnabled: false, theme: {} } },
}))

vi.mock('vue', async (original) => ({
  ...await original<typeof import('vue')>(),
  createApp: () => ({ use: vi.fn(), directive: vi.fn(), mount: vi.fn() }),
}))
vi.mock('@/site/App.vue', () => ({ default: {} }))
vi.mock('@/site/components/PayloadRouteView.vue', () => ({ default: {} }))
vi.mock('@/runtime/payload', () => ({ readInitialPayload: () => mocks.payload, updateDocumentMeta: vi.fn() }))
vi.mock('@/runtime/router', () => ({
  fetchPage: mocks.fetchPage,
  preparePayload: async (payload: unknown) => ({ payload, component: {} }),
  installNavigation: (_initial: unknown, _view: unknown, commit: (page: unknown) => void) => { mocks.navigate.mockImplementation(commit); return { isReady: async () => {}, push: vi.fn() } },
}))
vi.mock('@/runtime/i18n', () => ({ currentLocale: () => 'zh', i18n: {} }))
vi.mock('@/runtime/flash-message', () => ({ hydrateFlashMessages: vi.fn() }))
vi.mock('@/runtime/site-theme', () => ({ applySiteThemePayload: vi.fn(), applyStoredTheme: vi.fn(), initSystemThemeListener: vi.fn() }))
vi.mock('@/runtime/appearance-settings', () => ({ applyStoredAppearanceSettings: vi.fn() }))
vi.mock('@/runtime/ba-touch-effect', () => ({ installBaTouchEffect: vi.fn() }))
vi.mock('@/runtime/app-navigation', () => ({ registerAppNavigator: vi.fn() }))
vi.mock('@/runtime/code-highlight-directive', () => ({ codeHighlightDirective: {} }))
vi.mock('@/runtime/math-render-directive', () => ({ mathRenderDirective: {} }))
vi.mock('@/runtime/code-copy-directive', () => ({ codeCopyDirective: {} }))
vi.mock('@/runtime/content-enhancements', () => ({ contentEnhancementsDirective: {} }))

afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks() })

test('an idle or background tab does not fetch a full page to poll analytics configuration', async () => {
  vi.useFakeTimers()
  vi.spyOn(document.head, 'appendChild').mockImplementation((node) => node)
  await import('../src/site/main')
  await vi.advanceTimersByTimeAsync(180_000)
  document.dispatchEvent(new Event('visibilitychange'))
  window.dispatchEvent(new Event('pageshow'))
  await Promise.resolve()
  expect(mocks.fetchPage).not.toHaveBeenCalled()
})

test('entering private campus unloads existing document scripts even when analytics is already disabled', async () => {
  vi.resetModules()
  vi.spyOn(document.head, 'appendChild').mockImplementation((node) => node)
  const reload = vi.spyOn(window.location, 'reload').mockImplementation(() => {})
  await import('../src/site/main')
  mocks.navigate({ payload: { ...mocks.payload, component: 'campus.home' }, component: {} })
  expect(reload).toHaveBeenCalledOnce()
})
