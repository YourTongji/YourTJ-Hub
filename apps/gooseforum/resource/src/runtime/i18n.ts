import { createI18n } from 'vue-i18n'
import zh from '@/locales/zh'

export const supportedLocales = ['zh', 'en', 'ja', 'de'] as const
export type Locale = (typeof supportedLocales)[number]

export const fallbackLocale: Locale = 'zh'

// 懒加载契约（首屏瘦身）：zh 静态打包为 fallback（zh 用户首帧零回退）；
// en/ja/de 按需动态 import，加载完成前 vue-i18n 自动回退 zh —— 非 zh 用户
// 首帧会短暂显示中文，这是已接受的取舍。加载器必须用显式语言 specifier：
// import.meta.glob 会把 locales/admin-raw.*.generated.ts 一并扫成 chunk。
// 四份语言包键结构一致（scripts/check-i18n-keys.mjs 门禁），消息形状以
// Record<string, unknown> 描述即可。
type LocaleMessages = Record<string, unknown>

const localeLoaders: Record<Exclude<Locale, 'zh'>, () => Promise<{ default: LocaleMessages }>> = {
  en: () => import('@/locales/en'),
  ja: () => import('@/locales/ja'),
  de: () => import('@/locales/de'),
}

// 已加载语言集合：zh 启动即就绪，其余语言首次加载后记录，重复切换不二次加载。
const loadedLocales = new Set<Locale>(['zh'])

export function normalizeLocale(value?: string | null): Locale | undefined {
  const normalized = (value || '').trim().toLowerCase()
  const short = normalized.split(/[-_,;]/)[0] as Locale
  return supportedLocales.includes(short) ? short : undefined
}

function readCookie(name: string) {
  if (typeof document === 'undefined') return ''
  return document.cookie
    .split('; ')
    .find((item) => item.startsWith(`${name}=`))
    ?.split('=')
    .slice(1)
    .join('=') || ''
}

export function detectLocale(): Locale {
  const queryLocale = typeof window !== 'undefined'
    ? normalizeLocale(new URL(window.location.href).searchParams.get('lang'))
    : undefined
  if (queryLocale) return queryLocale

  const cookieLocale = normalizeLocale(decodeURIComponent(readCookie('lang')))
  if (cookieLocale) return cookieLocale

  if (typeof navigator !== 'undefined') {
    for (const language of navigator.languages || [navigator.language]) {
      const locale = normalizeLocale(language)
      if (locale) return locale
    }
  }

  return fallbackLocale
}

export function setLocaleCookie(locale: Locale) {
  const secure = window.location.protocol === 'https:' ? '; Secure' : ''
  document.cookie = `lang=${locale}; path=/; max-age=31536000; samesite=lax${secure}`
}

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: detectLocale(),
  fallbackLocale,
  messages: { zh },
  // 开发环境开启缺失键告警（issue #225/#230）：键名泄漏在 dev 控制台即时可见，
  // 生产保持静默降级（fallback zh）。
  missingWarn: import.meta.env.DEV,
  fallbackWarn: import.meta.env.DEV,
})

// i18n.global.locale 的静态类型只识别出 messages 中的 zh；懒加载语言在运行期
// 注入，读写统一经此句柄放宽到完整 Locale 联合（vue-i18n 运行期无此限制）。
const activeLocale = i18n.global.locale as unknown as { value: Locale }

// vue-i18n 的静态类型只从 messages 识别出 zh；懒加载在运行期注入其余语言包，
// 写入侧显式放宽签名（各语言键结构一致，由 check-i18n-keys 门禁保证）。
function installLocaleMessages(locale: string, messages: LocaleMessages) {
  ;(i18n.global as unknown as {
    setLocaleMessage(locale: string, message: LocaleMessages): void
  }).setLocaleMessage(locale, messages)
}

let localeRequestSeq = 0

/**
 * 切换语言：目标语言未加载时先异步加载语言包，再应用 locale / cookie / <html lang>。
 * 调用方无需 await（沿用同步调用形态）；加载失败保持现状（zh fallback 兜底）。
 * 快速连切时以单调递增序号防乱序：只有最新一次请求会落地。
 */
export function setLocale(locale: Locale): Promise<void> {
  const requestSeq = ++localeRequestSeq
  // 记录发起时刻的当前语言：加载期间若被直接改掉（如测试固定 zh），旧请求不再覆盖。
  const startLocale = activeLocale.value

  const apply = () => {
    if (requestSeq !== localeRequestSeq) return
    if (activeLocale.value !== startLocale) return
    activeLocale.value = locale
    setLocaleCookie(locale)
    document.documentElement.lang = locale
  }

  if (loadedLocales.has(locale)) {
    apply()
    return Promise.resolve()
  }
  return localeLoaders[locale as Exclude<Locale, 'zh'>]()
    .then((mod) => {
      installLocaleMessages(locale, mod.default)
      loadedLocales.add(locale)
      apply()
    })
    .catch(() => {
      // 语言包加载失败：保持当前状态（zh fallback 兜底），后续切换会重试。
    })
}

export function currentLocale() {
  return activeLocale.value
}

// 启动时探测到的语言非 zh（cookie / ?lang= / 浏览器语言）→ 异步补载并应用。
// 首帧仍先渲染 zh fallback，语言 chunk 落地后无缝切换，不阻塞关键路径。
if (activeLocale.value !== 'zh') {
  void setLocale(activeLocale.value)
}
