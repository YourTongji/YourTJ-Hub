import { expect, it } from 'vitest'
import { readFileSync, mkdtempSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { execFileSync } from 'node:child_process'
import { runInNewContext } from 'node:vm'
import { parse } from 'yaml'

const workflow = parse(readFileSync('../../.github/workflows/collect-status.yml', 'utf8'))
const allowed = (job: { if: string }, event: string, ref: string, enabled = 'true', repo = 'YourTongji/YourTJ-Hub', scheduler = '') =>
  runInNewContext(job.if, { github: { event_name: event, ref, repository: repo }, vars: { STATUS_COLLECTION_ENABLED: enabled, STATUS_SCHEDULER_ENABLED: scheduler } })

// Evaluate the expressions in the real YAML; this checks the queue policy, not
// GitHub's hosted-runner scheduler. Run IDs/SHAs must not split a kind's queue.
function group(job: { concurrency: { group: string } }, kind: string, runId = 1) {
  return job.concurrency.group.replace(/\$\{\{(.*?)\}\}/g, (_, expression: string) => String(runInNewContext(expression, {
    inputs: { kind },
    github: { workflow: workflow.name, ref: 'refs/heads/main', run_id: runId, sha: `commit-${runId}`, event: { schedule: kind === 'devices' ? '7 * * * *' : '2,17,32,47 * * * *' } },
  })))
}

it('does not reserve a shared workflow queue before skipped jobs are filtered', () => {
  // A queued runner used to hold this group indefinitely, blocking both kinds
  // and even the disabled default-branch timer despite the job execution timeout.
  expect(workflow.concurrency).toBeUndefined()
})

it('supersedes a stuck collector only with a newer collection of the same kind', () => {
  const job = workflow.jobs.collect
  expect(job.concurrency?.['cancel-in-progress']).toBe(true)
  for (const kind of ['public', 'devices']) {
    expect(group(job, kind, 1)).toBe(group(job, kind, 2))
  }
  expect(group(job, 'public')).not.toBe(group(job, 'devices'))
})

it('isolates fallback timers from their main collectors and from the other kind', () => {
  const job = workflow.jobs.dispatch
  expect(job.concurrency?.['cancel-in-progress']).toBe(true)
  for (const kind of ['public', 'devices']) {
    expect(group(job, kind, 1)).toBe(group(job, kind, 2))
    for (const collectedKind of ['public', 'devices']) {
      expect(group(job, kind)).not.toBe(group(workflow.jobs.collect, collectedKind))
    }
  }
  expect(group(job, 'public')).not.toBe(group(job, 'devices'))
})

it('disables the delayed GitHub timer when Cloudflare owns scheduling, while keeping main dispatch usable', () => {
  expect(allowed(workflow.jobs.dispatch, 'schedule', 'refs/heads/dev', 'true', 'YourTongji/YourTJ-Hub', 'true')).toBe(false)
  expect(allowed(workflow.jobs.collect, 'workflow_dispatch', 'refs/heads/main', 'true', 'YourTongji/YourTJ-Hub', 'true')).toBe(true)
})

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
