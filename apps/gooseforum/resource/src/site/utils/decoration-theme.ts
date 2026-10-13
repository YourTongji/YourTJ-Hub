import { ref } from 'vue'
export const decorationThemeRevision = ref(0)
// One observer serves all decorated names, including custom theme token edits.
if (typeof document !== 'undefined') {
  new MutationObserver(() => decorationThemeRevision.value++).observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme', 'style', 'class'] })
}
