<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PasswordInput from './PasswordInput.vue'
import { ApiResponseError, completeTongjiRegistration, getTongjiRegistration, type TongjiRegistrationStatus } from '@/runtime/api'
import { safeUrl } from '@/runtime/safe-url'
const props = defineProps<{ terms?: boolean; privacy?: boolean }>()
const { t } = useI18n()
const status = ref<TongjiRegistrationStatus | null>(null)
const username = ref('')
const password = ref('')
const confirmation = ref('')
const agree = ref(false)
const loading = ref(true)
const saving = ref(false)
const error = ref('')
onMounted(async () => {
  try { status.value = await getTongjiRegistration() }
  catch { error.value = t('tongjiRegistration.expired') }
  finally { loading.value = false }
})
async function submit() {
  if (saving.value || !status.value) return
  error.value = ''
  if (!username.value.trim() || !password.value) { error.value = t('auth.validation.registerRequired'); return }
  if (password.value !== confirmation.value) { error.value = t('auth.validation.passwordMismatch'); return }
  if ((props.terms || props.privacy) && !agree.value) { error.value = t(props.terms && props.privacy ? 'auth.validation.termsRequired' : props.terms ? 'auth.validation.termsOnlyRequired' : 'auth.validation.privacyOnlyRequired'); return }
  saving.value = true
  try {
    const result = await completeTongjiRegistration(username.value.trim(), password.value, status.value.csrfToken)
    const target = safeUrl(result.redirect)
    window.location.assign(target.startsWith('/') && !target.startsWith('//') ? target : '/')
  } catch (err) {
    if (err instanceof ApiResponseError && err.messageCode === 'auth.required') {
      status.value = null
      error.value = t('tongjiRegistration.expired')
    } else { error.value = err instanceof Error ? err.message : t('auth.validation.registerFailed') }
  }
  finally { saving.value = false }
}
</script>
<template>
  <main class="min-h-screen bg-base-200 px-4 py-10 text-base-content">
    <section class="gf-card mx-auto max-w-md space-y-5 p-5 sm:p-8">
      <h1 class="text-2xl font-bold">{{ t('tongjiRegistration.title') }}</h1>
      <p class="text-sm text-base-content/70">{{ t('tongjiRegistration.intro') }}</p>
      <p v-if="error" role="alert" class="text-sm text-error">{{ error }}</p>
      <p v-if="loading" role="status">{{ t('common.loading') }}</p>
      <form v-else-if="status" class="space-y-4" @submit.prevent="submit">
        <p class="break-all text-sm">{{ t('tongjiRegistration.verifiedEmail') }}: {{ status.email }}</p>
        <label class="block space-y-2">
          <span class="text-sm">{{ t('auth.username') }}</span>
          <input v-model="username" class="gf-input w-full" required autocomplete="username" maxlength="32" />
        </label>
        <PasswordInput v-model="password" :label="t('auth.password')" :placeholder="t('auth.password')" autocomplete="new-password" />
        <PasswordInput v-model="confirmation" :label="t('auth.confirmPassword')" :placeholder="t('auth.confirmPassword')" autocomplete="new-password" />
        <label v-if="terms || privacy" class="flex items-start gap-2 text-sm">
          <input v-model="agree" type="checkbox" class="mt-1" />
          <span>{{ terms && privacy ? t('auth.agreeTerms') : terms ? t('auth.agreeTermsOnly') : t('auth.agreePrivacyOnly') }}
            <a v-if="terms" href="/terms" target="_blank" rel="noopener noreferrer" class="text-primary">{{ t('auth.termsLink') }}</a>
            <a v-if="privacy" href="/privacy" target="_blank" rel="noopener noreferrer" class="text-primary">{{ t('auth.privacyLink') }}</a>
          </span>
        </label>
        <button type="submit" class="gf-button gf-button-primary w-full" :disabled="saving">{{ t('auth.createAccount') }}</button>
      </form>
      <a v-else href="/login" class="gf-button gf-button-primary">{{ t('tongjiRegistration.restart') }}</a>
    </section>
  </main>
</template>
