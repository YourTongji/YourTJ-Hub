const RUNTIME_SCRIPT_TIMEOUT_MS = 15_000
const runtimeScriptLoads = new Map<string, Promise<void>>()

export function loadRuntimeScript(url: string, id: string) {
  const existing = document.getElementById(id) as HTMLScriptElement | null
  if (existing?.dataset.loaded === 'true') return Promise.resolve()

  const inFlight = runtimeScriptLoads.get(id)
  if (inFlight) return inFlight

  // An unmarked node may have already fired its event before this caller saw it.
  // Rebuild it so the new load has a reliable event source.
  existing?.remove()

  const script = document.createElement('script')
  script.id = id
  script.src = url
  script.async = true

  let resolveLoad!: () => void
  let rejectLoad!: (reason?: unknown) => void
  let settled = false
  const loadPromise = new Promise<void>((resolve, reject) => {
    resolveLoad = resolve
    rejectLoad = reject
  })
  runtimeScriptLoads.set(id, loadPromise)

  let timeout: number | undefined
  const cleanup = () => {
    script.removeEventListener('load', handleLoad)
    script.removeEventListener('error', handleError)
    if (timeout !== undefined) window.clearTimeout(timeout)
  }
  const settle = (callback: () => void) => {
    if (settled) return
    settled = true
    cleanup()
    if (runtimeScriptLoads.get(id) === loadPromise) runtimeScriptLoads.delete(id)
    callback()
  }
  const handleLoad = () => {
    script.dataset.loaded = 'true'
    settle(resolveLoad)
  }
  const handleError = () => {
    script.remove()
    settle(() => rejectLoad(new Error(`Failed to load runtime asset: ${url}`)))
  }

  script.addEventListener('load', handleLoad)
  script.addEventListener('error', handleError)
  timeout = window.setTimeout(() => {
    script.remove()
    settle(() => rejectLoad(new Error(`Timed out loading runtime asset: ${url}`)))
  }, RUNTIME_SCRIPT_TIMEOUT_MS)

  try {
    document.head.appendChild(script)
  } catch (error) {
    script.remove()
    settle(() => rejectLoad(error instanceof Error ? error : new Error(String(error))))
  }

  return loadPromise
}
