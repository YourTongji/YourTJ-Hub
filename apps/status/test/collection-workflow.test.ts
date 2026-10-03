import { expect, it } from 'vitest'
import { readFileSync, mkdtempSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { execFileSync } from 'node:child_process'
import { runInNewContext } from 'node:vm'
import { parse } from 'yaml'

const workflow = parse(readFileSync('../../.github/workflows/collect-status.yml', 'utf8'))
const allowed = (job: { if: string }, event: string, ref: string, enabled = 'true', repo = 'YourTongji/YourTJ-Hub') =>
  runInNewContext(job.if, { github: { event_name: event, ref, repository: repo }, vars: { STATUS_COLLECTION_ENABLED: enabled } })

it('only a main workflow_dispatch can enter the secret-bearing collector', () => {
  const job = workflow.jobs.collect
  for (const event of ['schedule', 'workflow_dispatch', 'push']) {
    for (const ref of ['refs/heads/dev', 'refs/heads/main', 'refs/heads/feature']) {
      expect(allowed(job, event, ref), `${event} ${ref}`).toBe(event === 'workflow_dispatch' && ref === 'refs/heads/main')
    }
  }
  expect(allowed(job, 'workflow_dispatch', 'refs/heads/main', 'false')).toBe(false)
  expect(allowed(job, 'workflow_dispatch', 'refs/heads/main', 'true', 'fork/repo')).toBe(false)
  expect(job.environment).toBe('status-collector')
  expect(job.steps.find((step: { uses?: string }) => step.uses?.startsWith('actions/checkout@')).with.ref).toBe('${{ github.sha }}')
})

it('the default-branch schedule only dispatches the workflow definition on main without production secrets', () => {
  const job = workflow.jobs.dispatch
  expect(job).toBeDefined()
  expect(job.environment).toBeUndefined()
  expect(JSON.stringify(job)).not.toMatch(/secrets\.|R2_|UMAMI_|CLOUDFLARE_|checkout@/)
  expect(job.permissions).toEqual({ actions: 'write' })
  expect(allowed(job, 'schedule', 'refs/heads/dev')).toBe(true)
  expect(allowed(job, 'workflow_dispatch', 'refs/heads/main')).toBe(false)
  expect(allowed(job, 'schedule', 'refs/heads/dev', 'false')).toBe(false)
  expect(allowed(job, 'schedule', 'refs/heads/dev', 'true', 'fork/repo')).toBe(false)
  const dir = mkdtempSync(join(tmpdir(), 'status-dispatch-'))
  try {
    writeFileSync(join(dir, 'gh'), '#!/bin/sh\nprintf "%s\\n" "$@"\n', { mode: 0o700 })
    for (const kind of ['public', 'devices']) {
      const args = execFileSync('sh', ['-eu', '-c', job.steps[0].run], {
        env: { PATH: `${dir}:${process.env.PATH}`, GH_REPO: 'YourTongji/YourTJ-Hub', COLLECTION_KIND: kind },
        encoding: 'utf8',
      }).trim().split('\n')
      expect(args).toEqual(['workflow', 'run', 'collect-status.yml', '--repo', 'YourTongji/YourTJ-Hub', '--ref', 'main', '-f', `kind=${kind}`])
    }
  } finally { rmSync(dir, { recursive: true, force: true }) }
})
