// Shared by collectors, the read API and the browser: lowering collection frequency
// must not leave otherwise valid snapshots stale between scheduled runs.
export const CURRENT_POLICY = { freshFor: 150_000, retainFor: 900_000 } as const
export const HISTORY_POLICY = { freshFor: 20 * 60_000, retainFor: 60 * 60_000 } as const
export const DEVICE_POLICY = { freshFor: 70 * 60_000, retainFor: 180 * 60_000 } as const
export const POLL_SECONDS = 60
export const sourcePolicies = {
  server: CURRENT_POLICY, uptime: CURRENT_POLICY, traffic: HISTORY_POLICY, devices: DEVICE_POLICY,
} as const
