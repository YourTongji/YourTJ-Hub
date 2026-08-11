<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { LoaderCircle, Languages, LockKeyhole, Mail, Moon, ShieldCheck, Sun, UserRound } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { forgotPassword, getCaptcha, login, register, verifyTotp } from '@/runtime/api'
import { queueFlashMessage } from '@/runtime/flash-message'
import { setLocale, supportedLocales, type Locale } from '@/runtime/i18n'
import { useSiteTheme } from '@/runtime/site-theme'
import type { LayoutPayload, LoginPageProps } from '@gooseforum/client'

const page = defineProps<{
  layout: LayoutPayload
  props: LoginPageProps
}>()

type Mode = 'login' | 'register' | 'forgot'

const { t, locale } = useI18n()
const { isDark, toggleTheme } = useSiteTheme()
const mode = ref<Mode>(page.props.initialMode || 'login')
const langMenuOpen = ref(false)
let langCloseTimer: number | undefined
const twoFactorPending = ref(false)
const totpCode = ref('')
const captchaImg = ref('')
const captchaId = ref('')
const captchaLoading = ref(false)
const notice = ref('')
const error = ref('')
const loading = reactive({
  login: false,
  register: false,
  forgot: false,
  totp: false,
})

const loginForm = reactive({
  username: '',
  password: '',
  captcha: '',
})

const registerForm = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  captcha: '',
  agree: false,
  website: '',
})

const forgotForm = reactive({
  email: '',
  captcha: '',
  website: '',
})

const title = computed(() => {
  if (mode.value === 'register') return t('auth.registerTitle')
  if (mode.value === 'forgot') return t('auth.forgotTitle')
  return t('auth.loginTitle')
})

const subtitle = computed(() => {
  if (mode.value === 'register') return t('auth.registerSubtitle')
  if (mode.value === 'forgot') return t('auth.forgotSubtitle')
  return t('auth.loginSubtitle')
})

const showSocial = computed(() => mode.value !== 'forgot')
// 仅允许站内相对路径跳转，拒绝 javascript:、//host 及任何含反斜杠的值
// （浏览器会将 \ 归一化为 /，/\evil.com 会被解析为跨域地址，服务端已同步校验）
const homeUrl = computed(() => {
  const target = page.props.redirectUrl || '/'
  if (!target.startsWith('/') || target.startsWith('//') || target.includes('\\')) return '/'
  if (target.length > 1 && target[1] === '/') return '/'
  return target
})
const brandImage = computed(() => page.layout.site.brandImage || '/static/pic/brand-default.webp')

onMounted(() => {
  refreshCaptcha()
})

function switchMode(next: Mode) {
  mode.value = next
  error.value = ''
  notice.value = ''
}

async function refreshCaptcha() {
  captchaLoading.value = true
  try {
    const captcha = await getCaptcha()
    captchaId.value = captcha.captchaId
    captchaImg.value = captcha.captchaImg
  } catch (err) {
    error.value = errorMessage(err, t('auth.validation.captchaLoadFailed'))
  } finally {
    captchaLoading.value = false
  }
}

async function handleLogin() {
  if (twoFactorPending.value) {
    await handleTotpVerify()
    return
  }
  if (!loginForm.username || !loginForm.password || !loginForm.captcha) {
    error.value = t('auth.validation.loginRequired')
    return
  }
  loading.login = true
  error.value = ''
  try {
    const result = await login(loginForm.username, loginForm.password, captchaId.value, loginForm.captcha)
    if (result.twoFactorRequired) {
      twoFactorPending.value = true
      notice.value = t('auth.validation.twoFactorRequired')
      return
    }
    window.location.href = homeUrl.value
  } catch (err) {
    error.value = errorMessage(err, t('auth.validation.loginFailed'))
    loginForm.captcha = ''
    refreshCaptcha()
  } finally {
    loading.login = false
  }
}

async function handleTotpVerify() {
  if (!totpCode.value) {
    error.value = t('auth.validation.twoFactorRequired')
    return
  }
  loading.totp = true
  error.value = ''
  try {
    await verifyTotp(totpCode.value)
    window.location.href = homeUrl.value
  } catch (err) {
    error.value = errorMessage(err, t('api.totpVerifyFailed'))
    totpCode.value = ''
  } finally {
    loading.totp = false
  }
}

function backToPasswordLogin() {
  twoFactorPending.value = false
  totpCode.value = ''
  notice.value = ''
  error.value = ''
}

async function handleRegister() {
  if (!registerForm.username || !registerForm.email || !registerForm.password || !registerForm.captcha) {
    error.value = t('auth.validation.registerRequired')
    return
  }
  if (registerForm.password !== registerForm.confirmPassword) {
    error.value = t('auth.validation.passwordMismatch')
    return
  }
  if (!registerForm.agree) {
    error.value = t('auth.validation.termsRequired')
    return
  }
  loading.register = true
  error.value = ''
  try {
    const message = await register(registerForm.username, registerForm.email, registerForm.password, captchaId.value, registerForm.captcha, String(locale.value), registerForm.website)
    queueFlashMessage(message || t('auth.validation.registerSuccess'), 'success')
    window.location.href = homeUrl.value
  } catch (err) {
    error.value = errorMessage(err, t('auth.validation.registerFailed'))
    registerForm.captcha = ''
    refreshCaptcha()
  } finally {
    loading.register = false
  }
}

async function handleForgot() {
  if (!forgotForm.email || !forgotForm.captcha) {
    error.value = t('auth.validation.forgotRequired')
    return
  }
  loading.forgot = true
  error.value = ''
  try {
    notice.value = await forgotPassword(forgotForm.email, captchaId.value, forgotForm.captcha, forgotForm.website)
    forgotForm.captcha = ''
    refreshCaptcha()
  } catch (err) {
    error.value = errorMessage(err, t('auth.validation.resetEmailFailed'))
    forgotForm.captcha = ''
    refreshCaptcha()
  } finally {
    loading.forgot = false
  }
}

function setLangMenu(open: boolean) {
  window.clearTimeout(langCloseTimer)
  langCloseTimer = undefined
  langMenuOpen.value = open
}

function closeLangMenuSoon() {
  window.clearTimeout(langCloseTimer)
  langCloseTimer = window.setTimeout(() => {
    langMenuOpen.value = false
  }, 120)
}

function setLang(lang: Locale) {
  setLocale(lang)
  langMenuOpen.value = false
}

function errorMessage(err: unknown, fallback: string) {
  return err instanceof Error && err.message ? err.message : fallback
}
</script>

<template>
  <main class="login-main relative min-h-screen overflow-hidden bg-base-100 text-base-content sm:bg-base-200 sm:px-6 sm:py-8 lg:px-8">
    <!-- 波点动效背景：纯 CSS 点阵 + 慢速漂移（transform 合成层，不影响首屏性能） -->
    <div class="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
      <div class="gf-dot-grid absolute inset-0" />
    </div>

    <div class="absolute right-3 top-3 z-30 flex items-center gap-1 sm:right-4 sm:top-4">
      <div
        class="relative"
        @mouseenter="setLangMenu(true)"
        @mouseleave="closeLangMenuSoon()"
        @focusin="setLangMenu(true)"
        @focusout="closeLangMenuSoon()"
      >
        <button
          type="button"
          class="inline-flex h-9 w-9 items-center justify-center rounded-full text-icon-muted transition-colors duration-150 hover:bg-base-300 hover:text-base-content"
          :aria-label="t('shell.switchLanguage')"
          :title="t('shell.switchLanguage')"
          :aria-expanded="langMenuOpen"
          @click="langMenuOpen = !langMenuOpen"
        >
          <Languages class="h-5 w-5" />
        </button>
        <Transition name="gf-menu">
          <div v-if="langMenuOpen" class="absolute right-0 top-full z-[70] w-36 pt-2">
            <div class="gf-menu-surface overflow-hidden py-1">
              <button
                v-for="item in supportedLocales"
                :key="item"
                class="block w-full px-3 py-1.5 text-left text-sm transition-colors duration-150 hover:bg-base-200"
                :class="locale === item ? 'font-semibold text-primary' : 'text-base-content/75'"
                type="button"
                @click="setLang(item)"
              >
                {{ t(`locale.${item}`) }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
      <button
        type="button"
        class="inline-flex h-9 w-9 items-center justify-center rounded-full text-icon-muted transition-colors duration-150 hover:bg-base-300 hover:text-base-content"
        :aria-label="t(isDark ? 'auth.switchToLight' : 'auth.switchToDark')"
        :title="t(isDark ? 'auth.switchToLight' : 'auth.switchToDark')"
        @click="toggleTheme"
      >
        <Sun v-if="isDark" class="h-5 w-5" />
        <Moon v-else class="h-5 w-5" />
      </button>
    </div>

    <section class="relative z-10 mx-auto flex min-h-screen w-full max-w-[880px] flex-col items-center justify-center sm:min-h-[calc(100vh-4rem)]">
      <div class="login-card gf-card grid w-full overflow-hidden border-0 shadow-none sm:border sm:shadow-[0_8px_40px_-24px_rgb(0_0_0/calc(var(--gf-depth)*0.35))] md:grid-cols-2">
        <div class="login-column flex flex-col justify-center px-4 py-8 sm:min-h-[470px] sm:px-8 sm:py-6">
          <!-- 移动端品牌区：桌面双栏的右栏品牌内容以顶部品牌头呈现（白龙茶/乌龙茶缩放后置于标语两侧） -->
          <div v-if="showSocial" class="mb-8 md:hidden">
            <div class="flex items-center justify-center gap-2">
              <figure class="mt-8 shrink-0">
                <img src="/static/pic/tea-white.webp" :alt="t('auth.teaWhite')" class="mx-auto h-20 w-auto -scale-x-100" />
                <figcaption class="mt-1.5 text-center text-xs font-semibold text-base-content/55">{{ t('auth.teaWhite') }}</figcaption>
              </figure>
              <div class="flex min-w-0 flex-col items-center text-center">
                <img :src="brandImage" :alt="page.layout.site.name" class="h-10 w-auto object-contain" />
                <p class="mt-3 text-xl font-bold leading-snug tracking-tight text-base-content">{{ t('auth.panelTagline') }}</p>
                <p class="mt-1 text-xs leading-5 text-base-content/45">{{ t('auth.panelTaglineSource') }}</p>
              </div>
              <figure class="mt-8 shrink-0">
                <img src="/static/pic/tea-dark.webp" :alt="t('auth.teaDark')" class="mx-auto h-20 w-auto" />
                <figcaption class="mt-1.5 text-center text-xs font-semibold text-base-content/55">{{ t('auth.teaDark') }}</figcaption>
              </figure>
            </div>
          </div>

          <div class="mb-4">
            <h1 class="text-[27px] font-bold leading-tight tracking-tight text-base-content">{{ title }}</h1>
            <p class="mt-1.5 text-sm leading-6 text-base-content/55">{{ subtitle }}</p>
          </div>

          <div v-if="mode !== 'forgot'" class="gf-segmented mb-6 grid-cols-2">
            <button type="button" class="gf-segmented-item" :class="mode === 'login' ? 'gf-segmented-item-active' : 'gf-segmented-item-idle'" @click="switchMode('login')">{{ t('shell.login') }}</button>
            <button type="button" class="gf-segmented-item" :class="mode === 'register' ? 'gf-segmented-item-active' : 'gf-segmented-item-idle'" @click="switchMode('register')">{{ t('shell.register') }}</button>
          </div>

          <p v-if="error" class="gf-status-message gf-status-message-error mb-4">{{ error }}</p>
          <p v-if="notice" class="gf-status-message gf-status-message-success mb-4">{{ notice }}</p>

          <form v-if="mode === 'login' && twoFactorPending" class="space-y-3" @submit.prevent="handleTotpVerify">
            <div class="flex items-center gap-2 text-sm font-semibold text-base-content">
              <ShieldCheck class="h-4 w-4 text-primary" />
              {{ t('auth.twoFactorTitle') }}
            </div>
            <p class="text-sm text-base-content/55">{{ t('auth.twoFactorDescription') }}</p>
            <label class="block">
              <span class="sr-only">{{ t('auth.twoFactorCode') }}</span>
              <input
                v-model="totpCode"
                class="gf-input text-center text-lg tracking-[0.4em]"
                :placeholder="t('auth.twoFactorCode')"
                inputmode="text"
                autocomplete="one-time-code"
                autocapitalize="characters"
              />
            </label>
            <button type="submit" class="gf-button gf-button-xl gf-button-primary w-full" :disabled="loading.totp">
              <LoaderCircle v-if="loading.totp" class="h-4 w-4 animate-spin" />
              {{ t('auth.twoFactorVerify') }}
            </button>
            <button type="button" class="w-full text-sm font-medium text-base-content/55 hover:text-base-content" @click="backToPasswordLogin">
              {{ t('auth.twoFactorBack') }}
            </button>
          </form>

          <form v-else-if="mode === 'login'" class="space-y-3" @submit.prevent="handleLogin">
            <label class="block">
              <span class="sr-only">{{ t('auth.usernameOrEmail') }}</span>
              <span class="relative block">
                <UserRound class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/55" />
                <input v-model.trim="loginForm.username" class="gf-input pl-10" :placeholder="t('auth.usernameOrEmail')" autocomplete="username" />
              </span>
            </label>
            <label class="block">
              <span class="sr-only">{{ t('auth.password') }}</span>
              <span class="relative block">
                <LockKeyhole class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/55" />
                <input v-model="loginForm.password" type="password" class="gf-input pl-10" :placeholder="t('auth.password')" autocomplete="current-password" />
              </span>
            </label>
            <div class="flex gap-3">
              <input v-model.trim="loginForm.captcha" class="gf-input min-w-0 flex-1" :placeholder="t('auth.captcha')" />
              <button type="button" class="relative h-10 w-28 overflow-hidden gf-panel" @click="refreshCaptcha">
                <LoaderCircle v-if="captchaLoading || !captchaImg" class="mx-auto h-5 w-5 animate-spin text-base-content/55" />
                <img v-else :src="captchaImg" :alt="t('auth.captchaAlt')" class="gf-captcha-image h-full w-full object-cover" />
              </button>
            </div>
            <div class="flex justify-end">
              <button type="button" class="text-sm font-medium text-primary hover:text-primary" @click="switchMode('forgot')">{{ t('auth.forgotPassword') }}</button>
            </div>
            <button type="submit" class="gf-button gf-button-xl gf-button-primary w-full" :disabled="loading.login">
              <LoaderCircle v-if="loading.login" class="h-4 w-4 animate-spin" />
              {{ t('shell.login') }}
            </button>
          </form>

          <form v-else-if="mode === 'register'" class="space-y-3" @submit.prevent="handleRegister">
            <label class="block">
              <span class="sr-only">{{ t('auth.username') }}</span>
              <span class="relative block">
                <UserRound class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/55" />
                <input v-model.trim="registerForm.username" class="gf-input pl-10" :placeholder="t('auth.username')" autocomplete="username" />
              </span>
            </label>
            <label class="block">
              <span class="sr-only">{{ t('auth.email') }}</span>
              <span class="relative block">
                <Mail class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/55" />
                <input v-model.trim="registerForm.email" type="email" class="gf-input pl-10" :placeholder="t('auth.email')" autocomplete="email" />
              </span>
            </label>
            <label class="block">
              <span class="sr-only">{{ t('auth.password') }}</span>
              <span class="relative block">
                <LockKeyhole class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/55" />
                <input v-model="registerForm.password" type="password" class="gf-input pl-10" :placeholder="t('auth.password')" autocomplete="new-password" />
              </span>
            </label>
            <label class="block">
              <span class="sr-only">{{ t('auth.confirmPassword') }}</span>
              <span class="relative block">
                <LockKeyhole class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/55" />
                <input v-model="registerForm.confirmPassword" type="password" class="gf-input pl-10" :placeholder="t('auth.confirmPassword')" autocomplete="new-password" />
              </span>
            </label>
            <div class="flex gap-3">
              <span class="relative min-w-0 flex-1">
                <input v-model.trim="registerForm.captcha" class="gf-input" :placeholder="t('auth.captcha')" />
              </span>
              <button type="button" class="relative h-10 w-28 overflow-hidden gf-panel" @click="refreshCaptcha">
                <LoaderCircle v-if="captchaLoading || !captchaImg" class="mx-auto h-5 w-5 animate-spin text-base-content/55" />
                <img v-else :src="captchaImg" :alt="t('auth.captchaAlt')" class="gf-captcha-image h-full w-full object-cover" />
              </button>
            </div>
            <label class="flex items-start gap-2 text-sm leading-5 text-base-content/55">
              <input v-model="registerForm.agree" type="checkbox" class="mt-1 h-4 w-4 rounded border-line text-primary focus:ring-primary" />
              <span>
                {{ t('auth.agreeTerms') }}
                <a href="/terms" target="_blank" rel="noopener noreferrer" class="font-medium text-primary hover:text-primary">{{ t('auth.termsLink') }}</a>
              </span>
            </label>
            <input v-model="registerForm.website" type="text" class="hidden" tabindex="-1" autocomplete="off" aria-hidden="true" />
            <button type="submit" class="gf-button gf-button-xl gf-button-neutral w-full" :disabled="loading.register">
              <LoaderCircle v-if="loading.register" class="h-4 w-4 animate-spin" />
              {{ t('auth.createAccount') }}
            </button>
          </form>

          <form v-else class="space-y-3.5" @submit.prevent="handleForgot">
            <label class="block">
              <span class="relative block">
                <Mail class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/55" />
                <input v-model.trim="forgotForm.email" type="email" class="gf-input pl-10" :placeholder="t('auth.registeredEmail')" autocomplete="email" />
              </span>
            </label>
            <div class="flex gap-3">
              <input v-model.trim="forgotForm.captcha" class="gf-input min-w-0 flex-1" :placeholder="t('auth.captcha')" />
              <button type="button" class="relative h-10 w-28 overflow-hidden gf-panel" @click="refreshCaptcha">
                <LoaderCircle v-if="captchaLoading || !captchaImg" class="mx-auto h-5 w-5 animate-spin text-base-content/55" />
                <img v-else :src="captchaImg" :alt="t('auth.captchaAlt')" class="gf-captcha-image h-full w-full object-cover" />
              </button>
            </div>
            <input v-model="forgotForm.website" type="text" class="hidden" tabindex="-1" autocomplete="off" aria-hidden="true" />
            <button type="submit" class="gf-button gf-button-xl gf-button-primary w-full" :disabled="loading.forgot">
              <LoaderCircle v-if="loading.forgot" class="h-4 w-4 animate-spin" />
              {{ t('auth.sendResetEmail') }}
            </button>
            <button type="button" class="w-full text-sm font-medium text-primary hover:text-primary" @click="switchMode('login')">{{ t('auth.backToLogin') }}</button>
          </form>
        </div>

        <aside class="relative hidden overflow-hidden border-t border-line md:block md:border-l md:border-t-0">
          <div class="absolute inset-0 bg-gradient-to-br from-primary/20 via-info/5 to-base-100" />
          <div class="pointer-events-none absolute -right-24 -top-24 h-72 w-72 rounded-full bg-primary/15 blur-3xl" />
          <div class="pointer-events-none absolute -bottom-32 -left-20 h-80 w-80 rounded-full bg-info/15 blur-3xl" />

          <div class="relative flex h-full min-h-[470px] flex-col justify-between gap-8 p-8">
            <div>
              <img :src="brandImage" :alt="page.layout.site.name" class="h-10 w-auto object-contain" />
              <p class="mt-6 text-2xl font-bold leading-snug tracking-tight text-base-content">{{ t('auth.panelTagline') }}</p>
              <p class="mt-1.5 text-xs leading-5 text-base-content/45">{{ t('auth.panelTaglineSource') }}</p>
            </div>

            <div class="flex items-start justify-center gap-6">
              <figure class="min-w-0">
                <img src="/static/pic/tea-white.webp" :alt="t('auth.teaWhite')" class="mx-auto h-36 w-auto sm:h-44" />
                <figcaption class="mt-2 text-center text-xs font-semibold text-base-content/55">{{ t('auth.teaWhite') }}</figcaption>
              </figure>
              <figure class="min-w-0">
                <img src="/static/pic/tea-dark.webp" :alt="t('auth.teaDark')" class="mx-auto h-36 w-auto sm:h-44" />
                <figcaption class="mt-2 text-center text-xs font-semibold text-base-content/55">{{ t('auth.teaDark') }}</figcaption>
              </figure>
            </div>

            <div v-if="showSocial">
              <h2 class="text-xs font-bold uppercase tracking-wide text-base-content/45">{{ t('auth.continueWith') }}</h2>
              <div class="mt-3 space-y-2.5">
                <a :href="page.props.githubUrl" class="gf-button gf-button-lg gf-button-secondary w-full">
                  <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <path d="M12 0C5.37 0 0 5.37 0 12c0 5.3 3.44 9.8 8.21 11.39.6.11.79-.26.79-.58v-2.03c-3.34.73-4.04-1.42-4.04-1.42-.55-1.39-1.34-1.76-1.34-1.76-1.09-.75.08-.73.08-.73 1.21.08 1.85 1.24 1.85 1.24 1.07 1.83 2.81 1.3 3.49 1 .11-.78.42-1.3.76-1.6-2.67-.31-5.47-1.34-5.47-5.93 0-1.31.47-2.38 1.24-3.22-.12-.3-.54-1.52.12-3.18 0 0 1.01-.32 3.3 1.23A11.5 11.5 0 0 1 12 6c1.02 0 2.05.14 3.01.4 2.29-1.55 3.3-1.23 3.3-1.23.65 1.66.24 2.88.12 3.18.77.84 1.24 1.91 1.24 3.22 0 4.61-2.81 5.62-5.48 5.92.43.37.81 1.1.81 2.22v3.29c0 .32.19.69.8.58A12.01 12.01 0 0 0 24 12c0-6.63-5.37-12-12-12Z" />
                  </svg>
                  GitHub
                </a>

                <button type="button" class="gf-button gf-button-lg gf-button-secondary w-full cursor-not-allowed opacity-70">
                  {{ t('auth.googleUnavailable') }}
                </button>
              </div>
            </div>
          </div>
        </aside>
      </div>

      <div v-if="showSocial" class="mb-6 w-full px-4 md:hidden">
        <h2 class="mb-2 text-center text-xs font-bold uppercase tracking-wide text-base-content/45">{{ t('auth.continueWith') }}</h2>
        <div class="grid grid-cols-2 gap-2">
          <a :href="page.props.githubUrl" class="gf-button gf-button-md gf-button-secondary w-full">
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.3 3.44 9.8 8.21 11.39.6.11.79-.26.79-.58v-2.03c-3.34.73-4.04-1.42-4.04-1.42-.55-1.39-1.34-1.76-1.34-1.76-1.09-.75.08-.73.08-.73 1.21.08 1.85 1.24 1.85 1.24 1.07 1.83 2.81 1.3 3.49 1 .11-.78.42-1.3.76-1.6-2.67-.31-5.47-1.34-5.47-5.93 0-1.31.47-2.38 1.24-3.22-.12-.3-.54-1.52.12-3.18 0 0 1.01-.32 3.3 1.23A11.5 11.5 0 0 1 12 6c1.02 0 2.05.14 3.01.4 2.29-1.55 3.3-1.23 3.3-1.23.65 1.66.24 2.88.12 3.18.77.84 1.24 1.91 1.24 3.22 0 4.61-2.81 5.62-5.48 5.92.43.37.81 1.1.81 2.22v3.29c0 .32.19.69.8.58A12.01 12.01 0 0 0 24 12c0-6.63-5.37-12-12-12Z" />
            </svg>
            GitHub
          </a>

          <button type="button" class="gf-button gf-button-md gf-button-secondary w-full cursor-not-allowed opacity-70">
            {{ t('auth.googleUnavailable') }}
          </button>
        </div>
      </div>
    </section>
  </main>
</template>

<style scoped>
/* 波点动效背景：
   - 纯 CSS radial-gradient 点阵（无图片请求）
   - 漂移动画仅用 transform（GPU 合成层，不触发重排/重绘）
   - 80s 慢速循环 + 径向遮罩边缘淡出，视觉柔和、CPU 占用可忽略 */
.gf-dot-grid {
  background-image: radial-gradient(
    circle,
    color-mix(in oklch, var(--gf-color-primary) 35%, transparent) 2px,
    transparent 2px
  );
  background-size: 24px 24px;
  mask-image: radial-gradient(ellipse at center, black 20%, transparent 78%);
  -webkit-mask-image: radial-gradient(ellipse at center, black 20%, transparent 78%);
  animation: gf-dot-drift 80s linear infinite;
  will-change: transform;
}

@keyframes gf-dot-drift {
  from {
    transform: translate3d(0, 0, 0);
  }
  to {
    transform: translate3d(-24px, -24px, 0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .gf-dot-grid {
    animation: none;
  }
}

/* 移动端（<640px）：表单区以圆角卡片呈现，与波点背景区分；
   卡片上下留白露出波点，白底保证浅色文字清晰可读（Tailwind max-* 变体不可用，用原生媒体查询） */
@media (max-width: 639.98px) {
  .login-main {
    background: var(--gf-color-base-200);
  }

  /* 卡片高度贴合内容，由内容决定；上下留白由 flex 居中自然产生，波点大面积可见 */
  .login-card {
    background: var(--gf-color-base-100);
    border: var(--gf-border) solid var(--gf-color-line);
    border-radius: var(--gf-radius-box);
    box-shadow: 0 8px 32px -24px rgb(15 23 42 / calc(var(--gf-depth) * 0.4));
    margin-block: 1.5rem;
  }
}
</style>
