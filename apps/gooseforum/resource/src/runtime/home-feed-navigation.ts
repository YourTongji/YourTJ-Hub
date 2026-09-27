import { readonly, ref } from 'vue'

const pendingUrl = ref<string | null>(null)
const failedUrl = ref<string | null>(null)
let sequence = 0

export const homeFeedNavigation = {
  pendingUrl: readonly(pendingUrl),
  failedUrl: readonly(failedUrl),
  begin(url: string) {
    pendingUrl.value = url
    failedUrl.value = null
    return ++sequence
  },
  complete(request: number) {
    if (request !== sequence) return false
    pendingUrl.value = null
    failedUrl.value = null
    return true
  },
  fail(request: number, url: string) {
    if (request !== sequence) return false
    pendingUrl.value = null
    failedUrl.value = url
    return true
  },
  cancel() {
    sequence++
    pendingUrl.value = null
    failedUrl.value = null
  },
}
