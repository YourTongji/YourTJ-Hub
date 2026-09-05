import type { ObjectDirective } from 'vue'
import { mermaidContentEnhancer } from '@/runtime/content-enhancements/mermaid'

/**
 * Content enhancement pipeline for already-rendered content (v-html).
 *
 * Enhancers decorate rendered HTML in place — they never change the stored
 * Markdown or the server-rendered result. Each enhancer loads its own heavy
 * chunk lazily (detected from the rendered DOM), so pages that do not use the
 * enhanced feature keep the base bundle unchanged. A failed enhancer leaves
 * the original rendered HTML untouched.
 */

export interface ContentEnhancer {
  name: string
  enhance: (root: HTMLElement) => Promise<void> | void
}

const contentEnhancers: ContentEnhancer[] = [
  mermaidContentEnhancer,
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
}