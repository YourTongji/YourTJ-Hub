import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'

const source = (path: string) => readFileSync(resolve(__dirname, path), 'utf8')

describe('UUID compatibility call sites', () => {
  test.each([
    ['SiteChrome setup, normalization, and item creation', '../src/admin/pages/management/SiteChromeManagementPage.vue', 4],
    ['HTTP notification endpoint normalization and creation', '../src/admin/pages/AdminSettingsPage.vue', 2],
    ['schedule draft restore', '../src/site/composables/useScheduleSync.ts', 1],
    ['anonymous draw idempotency key', '../src/site/components/AnonymousIdentityDialog.vue', 1],
  ])('%s uses the shared UUID helper', (_name, path, expectedCalls) => {
    const page = source(path)
    expect(page.match(/createUuidV4\(\)/g)).toHaveLength(expectedCalls)
    expect(page).not.toContain('crypto.randomUUID()')
  })
})
