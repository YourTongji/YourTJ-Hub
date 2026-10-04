export interface SchedulerEnv {
  GITHUB_DISPATCH_TOKEN?: string
}

const kinds: Record<string, string> = {
  '2,17,32,47 * * * *': 'public',
  '7 * * * *': 'devices',
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

    let response: Response
    try {
      response = await fetch('https://api.github.com/repos/YourTongji/YourTJ-Hub/actions/workflows/collect-status.yml/dispatches', {
        method: 'POST',
        redirect: 'error',
        signal: AbortSignal.timeout(10_000),
        headers: {
          Authorization: `Bearer ${env.GITHUB_DISPATCH_TOKEN}`,
          Accept: 'application/vnd.github+json',
          'Content-Type': 'application/json',
          'User-Agent': 'yourtj-status-scheduler',
          'X-GitHub-Api-Version': '2022-11-28',
        },
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
