// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'

// i18n.ts 持有模块级单例（i18n 实例 / localeLoaders / loadedLocales / 竞序 token），
// 每个用例通过 vi.resetModules() + 动态 import 拿到全新模块实例，互不污染。
vi.mock('@/locales/en', () => ({ default: { title: 'English' } }))
vi.mock('@/locales/ja', () => ({ default: { title: '日本語' } }))
vi.mock('@/locales/de', () => ({ default: { title: 'Deutsch' } }))

async function loadI18nModule() {
  vi.resetModules()
  return import('../src/runtime/i18n')
}

describe('i18n 语言包懒加载', () => {
  beforeEach(() => {
    // 固定探测结果为 zh：模块启动时不会自动补载 en，才能观察「加载前」空态。
    document.cookie = 'lang=zh; path=/; max-age=31536000'
  })

  test('zh 同步可用；en 语言包在加载前为空', async () => {
    const { i18n } = await loadI18nModule()
    expect(Object.keys(i18n.global.getLocaleMessage('zh')).length).toBeGreaterThan(0)
    expect(i18n.global.getLocaleMessage('en')).toEqual({})
    expect(i18n.global.locale.value).toBe('zh')
  })

  test('await setLocale(en) 后 en 语言包就绪并生效', async () => {
    const { i18n, setLocale } = await loadI18nModule()
    await setLocale('en')
    expect(i18n.global.getLocaleMessage('en')).not.toEqual({})
    expect(i18n.global.locale.value).toBe('en')
    expect(i18n.global.t('title')).toBe('English')
  })

  test('快速连切 setLocale(en) → setLocale(ja)：最新请求生效，无串扰', async () => {
    const { i18n, setLocale } = await loadI18nModule()
    const enRequest = setLocale('en')
    const jaRequest = setLocale('ja')
    await Promise.all([enRequest, jaRequest])
    expect(i18n.global.locale.value).toBe('ja')
    expect(i18n.global.t('title')).toBe('日本語')
  })

  test('目标语言已加载后重复 setLocale 直接应用，不重复加载', async () => {
    const { i18n, setLocale } = await loadI18nModule()
    await setLocale('ja')
    await setLocale('ja')
    expect(i18n.global.locale.value).toBe('ja')
    expect(i18n.global.t('title')).toBe('日本語')
  })
})
