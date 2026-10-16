import { config } from '@vue/test-utils'
import { twemojiVuePlugin } from '../src/runtime/twemoji'

config.global.plugins = [twemojiVuePlugin]
