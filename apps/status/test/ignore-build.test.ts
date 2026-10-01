import { execFileSync, spawnSync } from 'node:child_process'
import { cpSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterAll, beforeAll, expect, it } from 'vitest'

const repo = mkdtempSync(join(tmpdir(), 'status-build-test-'))
const base = join(repo, 'apps/status')
const config = readFileSync('netlify.toml', 'utf8')
const command = config.match(/^\s*ignore = "([^"]+)"$/m)?.[1]
let initial: string
let forumOnly: string
let statusChange: string

function git(...args: string[]) {
  return execFileSync('git', args, {
    cwd: repo,
    encoding: 'utf8',
    env: { ...process.env, GIT_AUTHOR_NAME: 'Status Test', GIT_AUTHOR_EMAIL: 'test@example.com',
      GIT_COMMITTER_NAME: 'Status Test', GIT_COMMITTER_EMAIL: 'test@example.com' },
  }).trim()
}

function commit() {
  git('add', '.')
  git('-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', 'commit', '-qm', 'fixture')
  return git('rev-parse', 'HEAD')
}

beforeAll(() => {
  mkdirSync(base, { recursive: true })
  git('init', '-q')
  writeFileSync(join(base, 'page.txt'), 'old status')
  writeFileSync(join(repo, 'forum.txt'), 'old forum')
  initial = commit()
  writeFileSync(join(repo, 'forum.txt'), 'new forum')
  forumOnly = commit()
  writeFileSync(join(base, 'page.txt'), 'new status')
  statusChange = commit()
  if (existsSync('scripts')) cpSync('scripts', join(base, 'scripts'), { recursive: true })
  writeFileSync(join(repo, 'mock-fetch.mjs'), 'globalThis.fetch = async () => Response.json(JSON.parse(process.env.STATUS_TEST_MANIFEST || "{}"))')
})

afterAll(() => rmSync(repo, { recursive: true, force: true }))

function decision(context: string, cached: string, current: string, manifest?: object, force = '') {
  expect(command, 'Netlify must use the explicit build policy').toBeTruthy()
  const result = spawnSync('bash', ['-c', command!], {
    cwd: base,
    encoding: 'utf8',
    env: { ...process.env, URL: 'https://status.example.com', STATUS_FORCE_BUILD: force, CONTEXT: context, CACHED_COMMIT_REF: cached, COMMIT_REF: current,
      NODE_OPTIONS: `--import=${join(repo, 'mock-fetch.mjs')}`, STATUS_TEST_MANIFEST: JSON.stringify(manifest ?? {}) },
  })
  expect(result.error).toBeUndefined()
  expect(result.stderr).toBe('')
  return result.status
}

it('builds production when the published version is unknown or the cache points to this commit', () => {
  expect(decision('production', initial, forumOnly)).toBe(1)
  expect(decision('production', statusChange, statusChange)).toBe(1)
  expect(decision('production', '', statusChange)).toBe(1)
})

it('skips production only against content already published, independently of a preview cache', () => {
  const manifest = { schema: 1, context: 'production', tree: git('rev-parse', `${initial}:apps/status`) }
  expect(decision('production', initial, forumOnly, manifest)).toBe(0)
  expect(decision('production', forumOnly, statusChange, manifest)).toBe(1)
  const preview = { ...manifest, context: 'deploy-preview', tree: git('rev-parse', `${statusChange}:apps/status`) }
  expect(decision('production', initial, statusChange, preview)).toBe(1)
  expect(decision('production', initial, forumOnly, manifest, 'true')).toBe(1)
  expect(decision('production', forumOnly, forumOnly, manifest)).toBe(1)
  expect(decision('production', '', forumOnly, manifest)).toBe(1)
})

it.each(['deploy-preview', 'branch-deploy'])('only skips unrelated changes for %s', context => {
  expect(decision(context, initial, forumOnly)).toBe(0)
  expect(decision(context, forumOnly, statusChange)).toBe(1)
})

it('builds previews when cache metadata is missing, identical or unavailable in the checkout', () => {
  expect(decision('deploy-preview', '', statusChange)).toBe(1)
  expect(decision('deploy-preview', initial, '')).toBe(1)
  expect(decision('deploy-preview', statusChange, statusChange)).toBe(1)
  expect(decision('deploy-preview', '0'.repeat(40), statusChange)).toBe(1)
})

it('builds when the deployment context is unknown', () => {
  expect(decision('', initial, forumOnly)).toBe(1)
})
