import { createApp as createVueApp, type Component, type Data } from 'vue'
import { twemojiVuePlugin } from '@/runtime/twemoji'

export function createBrowserApp(rootComponent: Component, rootProps?: Data | null) {
  return createVueApp(rootComponent, rootProps).use(twemojiVuePlugin)
}
