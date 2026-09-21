<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { forgotPassword, getCaptcha, saveUserName } from '@/runtime/api'
import { safeUrl } from '@/runtime/safe-url'

const props = defineProps<{ username: string; email: string; returnTo: string }>()
const { t } = useI18n()
const username = ref(props.username)
const saving = ref(false)
const sending = ref(false)
const captchaLoading = ref(false)
const captchaId = ref('')
const captchaImg = ref('')
const captchaCode = ref('')
const passwordOpen = ref(false)
const notice = ref('')
const error = ref('')
// Query parameters never become an external navigation capability.
const destination = computed(() => {
  const target = safeUrl(props.returnTo)
  return target.startsWith('/') && !target.startsWith('//') ? target : '/'
})
async function saveUsername() {
  if (saving.value) return
  saving.value = true
  error.value = ''
  notice.value = ''
  try {
    await saveUserName(username.value.trim())
    notice.value = t('onboarding.usernameSaved')
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('api.usernameSaveFailed')
  } finally { saving.value = false }
}
async function refreshCaptcha() {
  if (captchaLoading.value) return
  captchaLoading.value = true
  error.value = ''
  try {
    const captcha = await getCaptcha()
    captchaId.value = captcha.captchaId
    captchaImg.value = captcha.captchaImg
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('auth.validation.captchaLoadFailed')
  } finally { captchaLoading.value = false }
}
async function openPassword() {
  passwordOpen.value = true
  await refreshCaptcha()
}
async function sendPasswordEmail() {
  if (sending.value || !captchaId.value || !captchaCode.value.trim()) return
  sending.value = true
  error.value = ''
  notice.value = ''
  try {
    notice.value = await forgotPassword(props.email, captchaId.value, captchaCode.value.trim())
    passwordOpen.value = false
  } catch (err) {
    const failure = err instanceof Error ? err.message : t('api.resetEmailFailed')
    await refreshCaptcha()
    error.value = failure
  } finally {
    captchaCode.value = ''
    sending.value = false
  }
}
</script>

<template>
  <main class="gf-card mx-auto min-w-0 max-w-xl space-y-6 p-5 sm:p-8">
    <header class="space-y-2">
      <h1 class="text-2xl font-bold">{{ t('onboarding.title') }}</h1>
      <p class="text-sm text-base-content/70">{{ t('onboarding.intro') }}</p>
    </header>
    <p v-if="notice" role="status" class="break-words text-sm text-success">{{ notice }}</p>
    <p v-if="error" role="alert" class="break-words text-sm text-error">{{ error }}</p>
    <form class="space-y-3" @submit.prevent="saveUsername">
      <label class="block space-y-2">
        <span class="text-sm font-semibold">{{ t('onboarding.username') }}</span>
        <input v-model="username" required maxlength="32" autocomplete="username" class="gf-input w-full" />
      </label>
      <button class="gf-button gf-button-primary" type="submit" :disabled="saving">{{ t('onboarding.saveUsername') }}</button>
    </form>
    <section class="space-y-3 border-t border-base-300 pt-5">
      <h2 class="font-semibold">{{ t('onboarding.passwordTitle') }}</h2>
      <p class="text-sm text-base-content/70">{{ t('onboarding.passwordHint') }}</p>
      <p class="break-all text-sm">{{ t('onboarding.email') }}: {{ email }}</p>
      <button v-if="!passwordOpen" class="gf-button gf-button-secondary" type="button" @click="openPassword">{{ t('onboarding.passwordTitle') }}</button>
      <form v-else class="space-y-3" @submit.prevent="sendPasswordEmail">
        <label class="block space-y-2">
          <span class="text-sm">{{ t('onboarding.captcha') }}</span>
          <input v-model="captchaCode" required autocomplete="off" class="gf-input w-full" />
        </label>
        <button type="button" :disabled="captchaLoading" :aria-label="t('auth.captcha')" @click="refreshCaptcha">
          <img v-if="captchaImg" :src="captchaImg" :alt="t('auth.captcha')" class="max-w-full rounded" />
          <span v-else>{{ t('common.retry') }}</span>
        </button>
        <button class="gf-button gf-button-primary w-full whitespace-normal" type="submit" :disabled="sending || captchaLoading || !captchaId">{{ t('onboarding.sendEmail') }}</button>
      </form>
    </section>
    <footer class="flex flex-wrap items-center gap-3 border-t border-base-300 pt-5">
      <a :href="destination" class="gf-button gf-button-primary">{{ t('onboarding.continue') }}</a>
      <a :href="destination" class="text-sm text-base-content/60 underline">{{ t('onboarding.skip') }}</a>
    </footer>
  </main>
</template>
