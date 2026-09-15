import { createI18n } from 'vue-i18n'
import zh from '../locales/zh'
import en from '../locales/en'
import ja from '../locales/ja'
import de from '../locales/de'
export type Locale = 'zh' | 'en' | 'ja' | 'de'
export const i18n = createI18n({ legacy: false, locale: 'zh', fallbackLocale: 'en', messages: { zh, en, ja, de } })
export function setLocale(locale: Locale) { i18n.global.locale.value = locale; document.documentElement.lang = locale }
