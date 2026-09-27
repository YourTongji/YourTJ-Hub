import { createApp, type App } from 'vue'
import { resolveLinkPreviews } from '@/runtime/api'
import { installExternalLinkGuard, setExternalLinkPreviewBlocked } from '@/runtime/external-link-guard'
import { i18n } from '@/runtime/i18n'
import {
  findRenderedLinkCandidates,
  LINK_PREVIEW_REQUEST_LIMIT,
  usableLinkPreview,
  type RenderedLinkCandidate,
} from '@/runtime/link-preview'
import type { ContentEnhancer } from '@/runtime/content-enhancements'

interface MountedPreview {
  app: App
  host: HTMLElement
  paragraph: HTMLParagraphElement
}

interface EnhancementState {
  abort: AbortController
  observer?: IntersectionObserver
  /** 本次增强实际接管的候选段落（含解析失败、未挂卡的），用于判断能否复用。 */
  candidates: RenderedLinkCandidate[]
  mounted: MountedPreview[]
  removeGuard: () => void
}

const states = new WeakMap<HTMLElement, EnhancementState>()
const previewPromises = new Map<string, Promise<Parameters<typeof usableLinkPreview>[0]>>()
const previewQueue = new Map<string, {
  resolve: (preview: Parameters<typeof usableLinkPreview>[0]) => void
  reject: (error: unknown) => void
}>()
let previewFlushScheduled = false

function queuePreview(url: string) {
  const existing = previewPromises.get(url)
  if (existing) return existing
  const promise = new Promise<Parameters<typeof usableLinkPreview>[0]>((resolve, reject) => {
    previewQueue.set(url, { resolve, reject })
  })
  previewPromises.set(url, promise)
  promise.catch(() => previewPromises.delete(url))
  if (!previewFlushScheduled) {
    previewFlushScheduled = true
    queueMicrotask(() => void flushPreviewQueue())
  }
  return promise
}

async function flushPreviewQueue() {
  const entries = Array.from(previewQueue.entries()).slice(0, LINK_PREVIEW_REQUEST_LIMIT)
  for (const [url] of entries) previewQueue.delete(url)
  previewFlushScheduled = previewQueue.size > 0
  if (previewFlushScheduled) queueMicrotask(() => void flushPreviewQueue())
  if (entries.length === 0) return
  try {
    const previews = await resolveLinkPreviews(entries.map(([url]) => url))
    entries.forEach(([url, pending], index) => {
      pending.resolve(previews.find(preview => preview.requestedUrl === url) ?? previews[index])
    })
  } catch (error) {
    entries.forEach(([, pending]) => pending.reject(error))
  }
}

async function mountPreview(candidate: RenderedLinkCandidate, preview: Parameters<typeof usableLinkPreview>[0], state: EnhancementState) {
  if (!usableLinkPreview(preview) || !candidate.paragraph.isConnected || state.abort.signal.aborted) return
  const { default: LinkPreviewCard } = await import('@/site/components/LinkPreviewCard.vue')
  if (!candidate.paragraph.isConnected || state.abort.signal.aborted) return
  const host = document.createElement('div')
  host.dataset.gfLinkPreviewHost = ''
  candidate.paragraph.insertAdjacentElement('afterend', host)
  const app = createApp(LinkPreviewCard, { preview })
  // 卡片自带文案（校园网卡片的兜底标题/描述由客户端出，服务端不返回中文），
  // 而这里是独立挂载的小应用，必须自己装上共享 i18n 实例。
  app.use(i18n)
  try {
    app.mount(host)
  } catch (error) {
    host.remove()
    throw error
  }
  candidate.paragraph.hidden = true
  state.mounted.push({ app, host, paragraph: candidate.paragraph })
}

/**
 * 候选集是否就是上次接管的那一批。用节点身份而不是内容比对：`v-html` 内容变化
 * 时 Vue 会把子树整体换成新节点，所以「同一批节点」等价于「内容没变」。
 */
function sameCandidates(previous: readonly RenderedLinkCandidate[], next: readonly RenderedLinkCandidate[]): boolean {
  if (previous.length !== next.length) return false
  return previous.every((candidate, index) => {
    const current = next[index]
    return candidate.paragraph === current.paragraph &&
      candidate.url === current.url &&
      candidate.paragraph.isConnected
  })
}

export async function enhanceLinkPreviews(root: HTMLElement) {
  // 幂等：宿主组件会因为与正文无关的状态（懒加载图片完成、贴纸库就绪、点赞/滚动）
  // 重渲染，而 `v-html` 只在内容字符串真的变化时才重写 DOM。这批情况下上一轮注入
  // 的宿主卡片还在原位，整段拆掉重建只会让卡片闪一下，并让失败的候选再打一次
  // preview 请求（服务端有负缓存，重试基本是纯浪费）。候选节点与 URL 都没变就直接
  // 收工，把上一轮的 AbortController / IntersectionObserver / guard 继续用下去。
  const previous = states.get(root)
  const candidates = findRenderedLinkCandidates(root, { includeEnhanced: Boolean(previous) })
  if (previous && sameCandidates(previous.candidates, candidates)) return

  disposeLinkPreviews(root)
  const state: EnhancementState = {
    abort: new AbortController(),
    candidates,
    mounted: [],
    removeGuard: installExternalLinkGuard(root),
  }
  states.set(root, state)

  if (candidates.length === 0) return
  for (const candidate of candidates) candidate.paragraph.dataset.gfLinkPreview = 'pending'

  const pending = new Set<RenderedLinkCandidate>()
  let scheduled = false
  const flush = async () => {
    scheduled = false
    const batch = Array.from(pending)
    pending.clear()
    if (batch.length === 0 || state.abort.signal.aborted) return
    try {
      const previews = await Promise.all(batch.map(candidate => queuePreview(candidate.url)))
      if (state.abort.signal.aborted) return
      await Promise.all(batch.map(async (candidate, index) => {
        setExternalLinkPreviewBlocked(candidate.anchor, previews[index]?.status === 'blocked')
        candidate.paragraph.dataset.gfLinkPreview = usableLinkPreview(previews[index]) ? 'ready' : 'failed'
        await mountPreview(candidate, previews[index], state)
      }))
    } catch (error) {
      if (state.abort.signal.aborted) return
      for (const candidate of batch) candidate.paragraph.dataset.gfLinkPreview = 'failed'
      console.warn('Unable to resolve link previews.', error)
    }
  }
  const queue = (candidate: RenderedLinkCandidate) => {
    pending.add(candidate)
    if (scheduled) return
    scheduled = true
    queueMicrotask(() => void flush())
  }

  if (typeof IntersectionObserver === 'undefined') {
    candidates.forEach(queue)
    return
  }
  state.observer = new IntersectionObserver((entries) => {
    for (const entry of entries) {
      if (!entry.isIntersecting) continue
      const candidate = candidates.find(item => item.paragraph === entry.target)
      if (!candidate) continue
      state.observer?.unobserve(candidate.paragraph)
      queue(candidate)
    }
  }, { rootMargin: '160px 0px' })
  for (const candidate of candidates) state.observer.observe(candidate.paragraph)
}

export function disposeLinkPreviews(root: HTMLElement) {
  const state = states.get(root)
  if (!state) return
  state.abort.abort()
  state.observer?.disconnect()
  state.removeGuard()
  for (const mounted of state.mounted) {
    mounted.app.unmount()
    mounted.host.remove()
    if (mounted.paragraph.isConnected) mounted.paragraph.hidden = false
  }
  root.querySelectorAll<HTMLElement>('[data-gf-link-preview]').forEach((element) => {
    delete element.dataset.gfLinkPreview
  })
  states.delete(root)
}

export const linkPreviewContentEnhancer: ContentEnhancer = {
  name: 'link preview',
  enhance: enhanceLinkPreviews,
  dispose: disposeLinkPreviews,
}
