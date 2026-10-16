export interface SchedulerEnv {
  GITHUB_DISPATCH_TOKEN?: string
}

const kinds: Record<string, string> = {
  '2,17,32,47 * * * *': 'public',
  '7 * * * *': 'devices',
}

const REPO = 'YourTongji/YourTJ-Hub'
const WORKFLOW = 'collect-status.yml'
const STUCK_WAITING_MS = 20 * 60 * 1000
const DEFAULT_LIST_PER_PAGE = 100
/** Test-only override for list page size; production keeps DEFAULT_LIST_PER_PAGE. */
export const stuckRunListTest = { perPage: DEFAULT_LIST_PER_PAGE }

type WaitingRun = { id: number, head_branch?: string, status?: string, created_at?: string }
type PendingDeployment = { wait_timer?: number, reviewers?: unknown[] }

function githubHeaders(token: string): HeadersInit {
  return {
    Authorization: `Bearer ${token}`,
    Accept: 'application/vnd.github+json',
    'Content-Type': 'application/json',
    'User-Agent': 'yourtj-status-scheduler',
    'X-GitHub-Api-Version': '2022-11-28',
  }
}

function isAgedMainWaiting(run: WaitingRun, now: number): boolean {
  return run.head_branch === 'main'
    && run.status === 'waiting'
    && typeof run.created_at === 'string'
    && now - Date.parse(run.created_at) >= STUCK_WAITING_MS
}

/** True when every pending deployment has no reviewers and wait_timer is 0 (or the list is empty). */
function isStuckWithoutProtection(pending: PendingDeployment[]): boolean {
  return pending.every(item =>
    (!Array.isArray(item.reviewers) || item.reviewers.length === 0)
    && (item.wait_timer ?? 0) === 0,
  )
}

async function listAgedWaitingRuns(token: string, now: number): Promise<WaitingRun[]> {
  const cutoff = new Date(now - STUCK_WAITING_MS).toISOString()
  const aged: WaitingRun[] = []
  for (let page = 1; ; page += 1) {
    const list = await fetch(
      `https://api.github.com/repos/${REPO}/actions/workflows/${WORKFLOW}/runs`
        + `?status=waiting&branch=main&per_page=${stuckRunListTest.perPage}&page=${page}`
        + `&created=<${encodeURIComponent(cutoff)}`,
      {
        method: 'GET',
        redirect: 'manual',
        signal: AbortSignal.timeout(10_000),
        headers: githubHeaders(token),
      },
    )
    if (list.status !== 200) {
      await list.body?.cancel()
      console.warn(`Listing stuck status collection runs failed (${list.status})`)
      return aged
    }
    const body = await list.json() as { workflow_runs?: WaitingRun[] }
    const pageRuns = body.workflow_runs ?? []
    for (const run of pageRuns) {
      if (isAgedMainWaiting(run, now)) aged.push(run)
    }
    if (pageRuns.length < stuckRunListTest.perPage) break
  }
  return aged
}

async function shouldForceCancel(token: string, runId: number): Promise<boolean> {
  const pendingRes = await fetch(
    `https://api.github.com/repos/${REPO}/actions/runs/${runId}/pending_deployments`,
    {
      method: 'GET',
      redirect: 'manual',
      signal: AbortSignal.timeout(10_000),
      headers: githubHeaders(token),
    },
  )
  if (pendingRes.status !== 200) {
    await pendingRes.body?.cancel()
    console.warn(`Reading pending deployments for run ${runId} failed (${pendingRes.status})`)
    return false
  }
  const pending = await pendingRes.json() as PendingDeployment[]
  if (!Array.isArray(pending)) return false
  if (!isStuckWithoutProtection(pending)) {
    console.info(`Skipping force-cancel of run ${runId}: pending deployment has reviewers or wait_timer`)
    return false
  }
  return true
}

/** Soft-fail: clear collect-status runs stuck waiting on an unprotected environment so a new dispatch can proceed. */
async function forceCancelStuckWaitingRuns(token: string, now = Date.now()): Promise<void> {
  try {
    const stuck = await listAgedWaitingRuns(token, now)
    for (const run of stuck) {
      try {
        if (!(await shouldForceCancel(token, run.id))) continue
        const cancel = await fetch(
          `https://api.github.com/repos/${REPO}/actions/runs/${run.id}/force-cancel`,
          {
            method: 'POST',
            redirect: 'manual',
            signal: AbortSignal.timeout(10_000),
            headers: githubHeaders(token),
          },
        )
        await cancel.body?.cancel()
        if (cancel.status === 202) console.info(`Force-cancelled stuck status collection run: ${run.id}`)
        else console.warn(`Force-cancel of stuck status collection run ${run.id} failed (${cancel.status})`)
      } catch {
        // Soft-fail: a cancel error must not block the new dispatch.
      }
    }
  } catch {
    // Soft-fail: listing stuck runs must not block the new dispatch.
  }
}

export default {
  // A separate Worker isolates dispatch credentials from the public reader.
  // No route, request parameter or visitor can trigger a collection run.
  fetch() {
    return new Response('Not found', { status: 404, headers: { 'Cache-Control': 'no-store' } })
  },
  async scheduled(event: { cron: string }, env: SchedulerEnv) {
    const kind = Object.hasOwn(kinds, event.cron) ? kinds[event.cron] : undefined
    if (!kind) throw new Error('Unknown status collection schedule')
    if (!env.GITHUB_DISPATCH_TOKEN?.trim()) throw new Error('Status dispatch token is missing')

    await forceCancelStuckWaitingRuns(env.GITHUB_DISPATCH_TOKEN)

    let response: Response
    try {
      response = await fetch(`https://api.github.com/repos/${REPO}/actions/workflows/${WORKFLOW}/dispatches`, {
        method: 'POST',
        // Workers rejects redirect: 'error' before sending the request. Manual
        // mode plus the 204-only check below rejects redirects without leaking credentials.
        redirect: 'manual',
        signal: AbortSignal.timeout(10_000),
        headers: githubHeaders(env.GITHUB_DISPATCH_TOKEN),
        body: JSON.stringify({ ref: 'main', inputs: { kind } }),
      })
    } catch {
      // Do not log upstream error text or retry an ambiguously accepted POST.
      throw new Error('Status dispatch request failed')
    }
    await response.body?.cancel()
    if (response.status !== 204) throw new Error(`Status dispatch rejected (${response.status})`)
    console.info(`Status collection dispatched: ${kind}`)
  },
}
