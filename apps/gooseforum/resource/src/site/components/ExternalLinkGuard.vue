<script setup lang="ts">
import { Ban, Check, Copy, ExternalLink, ShieldAlert } from '@lucide/vue'
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  cancelExternalLinkGuard,
  continueExternalNavigation,
  externalLinkGuardState,
} from '@/runtime/external-link-guard'

const { t } = useI18n()
const dialog = ref<HTMLDialogElement | null>(null)
const remember = ref(false)
const copied = ref(false)

/**
 * 风险状态需要视觉线索，不能只靠文案区分（suspicious/blocked 的文案不同但外观
 * 曾完全一致）。色板复用项目既有的 warning/error token，中性态刻意保持无彩，
 * 使唯一的强调色留给主按钮。
 */
const riskTone = computed(() => {
  switch (externalLinkGuardState.pending?.risk) {
    case 'blocked':
      return { icon: Ban, tile: 'bg-error/10 text-error' }
    case 'suspicious':
      return { icon: ShieldAlert, tile: 'bg-warning/10 text-warning' }
    default:
      return { icon: ExternalLink, tile: 'bg-base-300 text-base-content/70' }
  }
})

watch(() => externalLinkGuardState.open, async (open) => {
  if (open) {
    remember.value = false
    copied.value = false
    await nextTick()
    if (!dialog.value?.open) dialog.value?.showModal()
    return
  }
  if (dialog.value?.open) dialog.value.close()
})

function cancel() {
  dialog.value?.close()
  cancelExternalLinkGuard()
}

function proceed() {
  dialog.value?.close()
  continueExternalNavigation(remember.value)
}

async function copyURL() {
  const href = externalLinkGuardState.pending?.href
  if (!href) return
  try {
    await navigator.clipboard.writeText(href)
    copied.value = true
  } catch {
    // The complete URL remains selectable when clipboard access is denied.
  }
}
</script>

<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      role="dialog"
      aria-modal="true"
      class="gf-external-link-dialog gf-floating-surface m-auto max-h-[calc(100dvh-2rem)] w-[min(28rem,calc(100vw-2rem))] overflow-y-auto p-0 text-base-content"
      aria-labelledby="external-link-title"
      aria-describedby="external-link-description"
      @cancel.prevent="cancel"
      @close="externalLinkGuardState.open && cancelExternalLinkGuard()"
    >
      <div class="p-5 sm:p-6">
        <!--
          头部与 QuickPublishModal 的离开确认弹窗同构（40px 图标块 + 标题/说明），
          让本弹窗与项目里的确认对话框读起来是一套东西；图标随 risk 换色后，
          可疑/拦截态在视觉上就能区分，而不是只换一段文案。
        -->
        <div class="flex items-start gap-3">
          <span
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg"
            :class="riskTone.tile"
            aria-hidden="true"
          >
            <component :is="riskTone.icon" class="h-5 w-5" />
          </span>
          <div class="min-w-0 flex-1">
            <h2 id="external-link-title" class="text-base font-bold text-base-content">
              {{ t('linkPreview.guard.title') }}
            </h2>
            <p id="external-link-description" class="mt-1 text-sm leading-6 text-base-content/70">
              {{ t(`linkPreview.guard.${externalLinkGuardState.pending?.risk ?? 'normal_external'}`) }}
            </p>
          </div>
        </div>
        <!--
          目的地按「标签 + 值」呈现：host 是用户真正要核对的字符串（#733），完整
          URL 作为可复制的值。URL 用等宽字体便于逐字符比对，并抬到 13px/75% ——
          它是本弹窗里最该被读清的一行，不应是全弹窗最弱的一档文字。
        -->
        <strong class="mt-4 block break-all text-sm font-semibold text-base-content">
          {{ externalLinkGuardState.pending?.hostname }}
        </strong>
        <div class="mt-2 flex min-w-0 items-center gap-2 rounded-[var(--gf-radius-field)] border border-line bg-base-200/60 p-2">
          <code class="min-w-0 flex-1 select-all break-all font-mono text-[13px] leading-5 text-base-content/75">
            {{ externalLinkGuardState.pending?.href }}
          </code>
          <button
            type="button"
            class="gf-icon-button h-7 w-7 shrink-0 hover:bg-base-300 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
            :class="copied ? 'text-success' : ''"
            :aria-label="t(copied ? 'linkPreview.guard.copied' : 'linkPreview.guard.copy')"
            @click="copyURL"
          >
            <Check v-if="copied" class="h-3.5 w-3.5" aria-hidden="true" />
            <Copy v-else class="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        </div>
        <label
          v-if="externalLinkGuardState.pending?.risk === 'normal_external'"
          class="mt-4 flex min-h-11 cursor-pointer items-center gap-2.5 text-sm text-base-content/75"
        >
          <input v-model="remember" type="checkbox" class="checkbox">
          <span>{{ t('linkPreview.guard.remember') }}</span>
        </label>
        <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button type="button" class="gf-button gf-button-lg gf-button-secondary" @click="cancel">
            {{ t('common.cancel') }}
          </button>
          <button
            v-if="externalLinkGuardState.pending?.risk !== 'blocked'"
            type="button"
            class="gf-button gf-button-lg gf-button-primary"
            @click="proceed"
          >
            {{ t('linkPreview.guard.continue') }}
          </button>
        </div>
      </div>
    </dialog>
  </Teleport>
</template>

<style scoped>
.gf-external-link-dialog::backdrop {
  background: rgb(0 0 0 / 0.42);
}

@media (prefers-reduced-motion: no-preference) {
  .gf-external-link-dialog[open] {
    animation: gf-external-link-enter 120ms ease-out;
  }
}

@keyframes gf-external-link-enter {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
}
</style>
