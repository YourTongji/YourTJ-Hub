import type { components } from '@gooseforum/client/openapi'

export type SearchMaintenanceStatus = components['schemas']['SearchMaintenanceStatus']
export type SearchMaintenanceRequest = components['schemas']['SearchMaintenanceRequest']
export type SearchMaintenanceSubmission = components['schemas']['SearchMaintenanceSubmission']
export type SearchMaintenanceJob = components['schemas']['SearchMaintenanceJob']

export function maintenanceActive(job: SearchMaintenanceJob): boolean {
  return job.status === 0 || job.status === 1 || job.status === 4
}

// A self-scheduling poller never overlaps requests and only runs while the
// visible page has active maintenance. It stops on failure until explicit retry.
export function maintenancePoller(load: () => Promise<boolean>, visible: () => boolean) {
  let disposed = false
  let running = false
  let timer: ReturnType<typeof setTimeout> | undefined
  async function refresh() {
    if (disposed || running || !visible()) return
    clearTimeout(timer)
    running = true
    try {
      const active = await load()
      if (!disposed && active && visible()) timer = setTimeout(() => { void refresh() }, 5000)
    } finally {
      running = false
    }
  }
  return {
    refresh,
    stop() { disposed = true; clearTimeout(timer) },
  }
}
