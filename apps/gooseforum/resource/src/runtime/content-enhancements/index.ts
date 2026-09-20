import type { ObjectDirective } from 'vue'
import { mermaidContentEnhancer } from '@/runtime/content-enhancements/mermaid'
import { linkPreviewContentEnhancer } from '@/runtime/content-enhancements/link-preview'

/**
 * Content enhancement pipeline for already-rendered content (v-html).
 *
 * Enhancers decorate rendered HTML in place — they never change the stored
 * Markdown or the server-rendered result. Each enhancer loads its own heavy
 * chunk lazily (detected from the rendered DOM), so pages that do not use the
 * enhanced feature keep the base bundle unchanged. A failed enhancer leaves
 * the original rendered HTML untouched.
 *
 * `enhance` MUST be idempotent and cheap when nothing changed: Vue re-renders
 * the host component for reasons unrelated to the content (lazy images
 * finishing, the sticker library resolving, like/scroll state), and `v-html`
 * only rewrites the DOM when the bound string actually changes — so a repeat
 * call usually finds the previous decorations still in place. An enhancer that
 * tears its own work down on every call makes the decoration visibly flash and
 * repeats whatever remote work it triggered. Each enhancer therefore owns its
 * state and decides for itself whether the DOM it decorated is still current;
 * `dispose` is a final teardown (`unmounted`), not a per-update reset.
 *
 * The pipeline assumes `v-html`-style wholesale replacement: a content change
 * swaps the subtree for fresh nodes, so enhancers detect it by node identity
 * rather than by diffing attributes.
 */

export interface ContentEnhancer {
  name: string
  enhance: (root: HTMLElement) => Promise<void> | void
  dispose?: (root: HTMLElement) => void
}

const contentEnhancers: ContentEnhancer[] = [
  mermaidContentEnhancer,
  linkPreviewContentEnhancer,
]

export async function enhanceRenderedContent(root: HTMLElement) {
  await Promise.all(contentEnhancers.map(async (enhancer) => {
    try {
      await enhancer.enhance(root)
    } catch (error) {
      console.warn(`Unable to apply the ${enhancer.name} content enhancer.`, error)
    }
  }))
}

export const contentEnhancementsDirective: ObjectDirective<HTMLElement> = {
  mounted(element) {
    void enhanceRenderedContent(element)
  },
  updated(element) {
    void enhanceRenderedContent(element)
  },
  unmounted(element) {
    contentEnhancers.forEach(enhancer => enhancer.dispose?.(element))
  },
}
