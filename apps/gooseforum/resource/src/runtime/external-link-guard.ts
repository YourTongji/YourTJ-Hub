import { reactive } from 'vue'
import { getDomain } from 'tldts'
import { safeUrl } from '@/runtime/safe-url'

const trustedDomainPrefix = 'goose:trusted-external-domain:'
let internalOrigins = new Set<string>()

interface PendingExternalNavigation {
  href: string
  hostname: string
  registrableDomain: string
  risk: ExternalLinkRisk
  newTab: boolean
  returnFocus: HTMLElement | null
}

export type ExternalLinkRisk = 'normal_external' | 'suspicious' | 'blocked'

export function setExternalLinkGuardInternalOrigins(origins: readonly string[]) {
  internalOrigins = new Set(origins.flatMap((value) => {
    try {
      const parsed = new URL(value)
      return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? [parsed.origin] : []
    } catch {
      return []
    }
  }))
}

export const externalLinkGuardState = reactive<{
  open: boolean
  pending: PendingExternalNavigation | null
}>({
  open: false,
  pending: null,
})

export function externalDomain(rawHref: string, base = window.location.href): string | null {
  const safeHref = safeUrl(rawHref, 'external')
  if (!safeHref) return null
  const parsed = new URL(safeHref, base)
  const current = new URL(base)
  if (parsed.origin === current.origin || internalOrigins.has(parsed.origin)) return null
  return parsed.hostname.toLowerCase()
}

export function registrableDomain(hostname: string): string {
  return getDomain(hostname, { allowPrivateDomains: true })?.toLowerCase() || hostname.toLowerCase()
}

function isTrusted(domain: string): boolean {
  try {
    return sessionStorage.getItem(`${trustedDomainPrefix}${domain}`) === '1'
  } catch {
    return false
  }
}

export function trustExternalDomain(domain: string) {
  try {
    sessionStorage.setItem(`${trustedDomainPrefix}${domain}`, '1')
  } catch {
    // Private browsing can disable sessionStorage; navigation still works.
  }
}

// Resolver denials apply only to the exact URL observed on this anchor.
const resolvedRisks = new WeakMap<HTMLAnchorElement, string>()
export function setExternalLinkPreviewBlocked(anchor: HTMLAnchorElement, blocked: boolean) {
  if (blocked) resolvedRisks.set(anchor, anchor.href)
  else resolvedRisks.delete(anchor)
}

function linkRisk(anchor: HTMLAnchorElement): ExternalLinkRisk {
  const risk = anchor.dataset.externalRisk
  if (risk === 'blocked' || resolvedRisks.get(anchor) === anchor.href) return 'blocked'
  if (risk === 'suspicious') return risk
  const labelUrl = safeUrl(anchor.textContent?.trim(), 'external')
  if (labelUrl && registrableDomain(new URL(labelUrl).hostname) !== registrableDomain(new URL(anchor.href).hostname)) return 'suspicious'
  return 'normal_external'
}

export function openExternalLinkGuard(anchor: HTMLAnchorElement, event: MouseEvent): boolean {
  const href = safeUrl(anchor.href, 'external')
  if (!href) return false
  const hostname = externalDomain(href)
  if (!hostname) return false
  const domain = registrableDomain(hostname)
  const risk = linkRisk(anchor)
  if (risk === 'normal_external' && isTrusted(domain)) return false

  event.preventDefault()
  externalLinkGuardState.pending = {
    href,
    hostname,
    registrableDomain: domain,
    risk,
    newTab: event.button === 1 || event.metaKey || event.ctrlKey || event.shiftKey || anchor.target === '_blank',
    returnFocus: anchor,
  }
  externalLinkGuardState.open = true
  return true
}

export function cancelExternalLinkGuard() {
  const returnFocus = externalLinkGuardState.pending?.returnFocus
  externalLinkGuardState.open = false
  externalLinkGuardState.pending = null
  requestAnimationFrame(() => returnFocus?.focus())
}

export function continueExternalNavigation(remember: boolean) {
  const pending = externalLinkGuardState.pending
  if (!pending || pending.risk === 'blocked') return
  if (remember && pending.risk === 'normal_external') trustExternalDomain(pending.registrableDomain)
  externalLinkGuardState.open = false
  externalLinkGuardState.pending = null
  if (pending.newTab) {
    window.open(pending.href, '_blank', 'noopener,noreferrer')
  } else {
    window.location.assign(pending.href)
  }
}

export function installExternalLinkGuard(root: HTMLElement): () => void {
  const handle = (event: MouseEvent) => {
    if (event.defaultPrevented || (event.type === 'click' && event.button !== 0) || (event.type === 'auxclick' && event.button !== 1)) return
    const target = event.target
    if (!(target instanceof Element)) return
    const anchor = target.closest<HTMLAnchorElement>('a[href]')
    if (!anchor || !root.contains(anchor) || anchor.hasAttribute('data-no-external-guard')) return
    openExternalLinkGuard(anchor, event)
  }
  root.addEventListener('click', handle)
  root.addEventListener('auxclick', handle)
  return () => {
    root.removeEventListener('click', handle)
    root.removeEventListener('auxclick', handle)
  }
}
