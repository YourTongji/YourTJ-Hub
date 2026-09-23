import type { LinkPreview } from '@gooseforum/client'
import { resolveLinkPreviews } from '@/runtime/api'
import { scanMarkdownLinkCandidates, usableLinkPreview } from '@/runtime/link-preview'

export interface LinkPreviewHint {
  domains: string[]
}

export interface LinkPreviewHintController {
  schedule(markdown: string): void
  compositionStart(): void
  compositionEnd(markdown: string): void
  dispose(): void
  /**
   * True once nothing is in flight: no debounced request pending and no
   * resolve round-trip running. Browser-test fixtures wait on this state
   * instead of sleeping past the debounce, so their waits track real state
   * rather than a constant in this file.
   */
  isSettled(): boolean
}

/**
 * 把域名列表拼成人类可读的一串。
 *
 * 分隔符由 i18n 提供而不是 Intl.ListFormat：实测 unit 模式在中文下分隔符为空
 * （拼出 "a.exampleb.example"），日文给的是空格，德文还会插连词 und——它是
 * 为单位列表设计的，不适用于域名枚举。
 */
export function formatLinkPreviewHintDomains(domains: readonly string[], separator: string): string {
  return domains.filter(Boolean).join(separator)
}

interface LinkPreviewHintOptions {
  delay?: number
  resolve?: (urls: readonly string[], signal: AbortSignal) => Promise<LinkPreview[]>
  onChange: (hint: LinkPreviewHint | null) => void
}

export function createLinkPreviewHintController(options: LinkPreviewHintOptions): LinkPreviewHintController {
  const delay = options.delay ?? 400
  const resolve = options.resolve ?? resolveLinkPreviews
  let timer: ReturnType<typeof setTimeout> | undefined
  let request: AbortController | undefined
  let sequence = 0
  let composing = false
  let candidateKey: string | null = null

  const cancelPending = () => {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
    request?.abort()
    request = undefined
  }

  // `request` must drop back to undefined once a resolve finishes, otherwise
  // isSettled would never turn true again after the first request.
  const isSettled = () => timer === undefined && request === undefined

  const schedule = (markdown: string) => {
    markdown = markdown ?? ''
    if (composing) return
    const urls = scanMarkdownLinkCandidates(markdown)
    const key = JSON.stringify(urls)
    // Typing elsewhere must not hide a valid hint or indefinitely delay its request.
    if (candidateKey === key) return
    candidateKey = key
    sequence += 1
    const currentSequence = sequence
    cancelPending()
    options.onChange(null)
    if (urls.length === 0) {
      options.onChange(null)
      return
    }
    timer = setTimeout(async () => {
      timer = undefined
      const activeRequest = new AbortController()
      request = activeRequest
      try {
        const previews = await resolve(urls, activeRequest.signal)
        if (currentSequence !== sequence || activeRequest.signal.aborted) return
        // 提示要如实反映「这次会出几张卡」：只报第一个的话，作者看到 3 个链接
        // 却只被告知 1 个域名，会误以为其余两个不会成卡（issue #729 实测）。
        // 只统计真正可用的预览，所以提示与实际渲染始终一致。
        const domains = previews
          .filter(usableLinkPreview)
          .map(preview => preview.displayHost || new URL(preview.url!).hostname)
        options.onChange(domains.length > 0 ? { domains } : null)
      } catch {
        if (currentSequence === sequence && !activeRequest.signal.aborted) {
          candidateKey = null
          options.onChange(null)
        }
      } finally {
        // A newer schedule may already own `request`; only retire our own handle.
        if (request === activeRequest) request = undefined
      }
    }, delay)
  }

  return {
    schedule,
    isSettled,
    compositionStart() {
      composing = true
      candidateKey = null
      sequence += 1
      cancelPending()
      options.onChange(null)
    },
    compositionEnd(markdown: string) {
      composing = false
      schedule(markdown)
    },
    dispose() {
      candidateKey = null
      sequence += 1
      cancelPending()
      options.onChange(null)
    },
  }
}
