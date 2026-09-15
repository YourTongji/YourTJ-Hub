// Allow a bounded device/server clock skew, never an unlimited future timestamp.
export function isRecentStatusTime(value: string | undefined, now: number, maxAge: number) {
  const age = now - Date.parse(value ?? '')
  return Number.isFinite(age) && age >= -60_000 && age <= maxAge
}
