import { defineConfig, mergeConfig } from 'vitest/config'
import { fileURLToPath, URL } from 'node:url'
import viteConfig from './vite.config'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      setupFiles: [fileURLToPath(new URL('./test/setup.ts', import.meta.url))],
    },
  }),
)
