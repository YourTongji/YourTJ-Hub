import { ref, watch, type Ref } from 'vue'

/** Keep recovery attached to the current displayed failure, never a previous one. */
export function useWriteRecovery(message: Ref<string>) {
  const code = ref<string>()
  watch(message, () => { code.value = undefined }, { flush: 'sync' })
  function capture(error: unknown) {
    code.value = typeof error === 'object' && error !== null && 'messageCode' in error && typeof error.messageCode === 'string' ? error.messageCode : undefined
  }
  return { code, capture }
}
