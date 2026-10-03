<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, ShieldAlert } from '@lucide/vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { closeModerationBlocked, moderationBlockedState as state } from '@/runtime/moderation-blocked'

// 内容被 AI 图文审查拦截时的友好提示（issue #975）：说明未发布原因、
// 内容已保留、下一步怎么改；不展示模型类别、概率或证据。
const { t } = useI18n()
</script>

<template>
  <DialogRoot :open="state.open" @update:open="value => { if (!value) closeModerationBlocked() }">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2100] bg-black/40 backdrop-blur-[2px]" />
      <DialogContent
        data-test="moderation-blocked-dialog"
        class="fixed left-1/2 top-1/2 z-[2101] w-[calc(100%_-_2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-box border border-line bg-base-100 p-5 text-base-content shadow-2xl outline-none"
      >
        <div class="flex items-start gap-3">
          <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-warning/15 text-warning">
            <ShieldAlert class="size-5" />
          </span>
          <div class="min-w-0 space-y-2">
            <DialogTitle class="text-base font-semibold leading-7">{{ t('moderationBlocked.title') }}</DialogTitle>
            <DialogDescription class="text-sm leading-6 text-base-content/80">{{ state.message || t('moderationBlocked.fallback') }}</DialogDescription>
            <p class="text-sm leading-6 text-base-content/70">
              {{ state.kind === 'externalImage' ? t('moderationBlocked.externalHint') : t('moderationBlocked.policyHint') }}
            </p>
            <p class="rounded-field bg-base-200/70 px-3 py-2 text-xs leading-5 text-base-content/65">{{ t('moderationBlocked.draftKept') }}</p>
          </div>
        </div>
        <div class="mt-5 flex flex-wrap items-center justify-end gap-2">
          <a v-if="state.kind === 'policy'" href="/terms" target="_blank" rel="noopener" class="gf-button gf-button-sm gf-button-ghost">
            {{ t('moderationBlocked.rules') }}<ArrowUpRight class="size-4" />
          </a>
          <button type="button" class="gf-button gf-button-sm gf-button-primary" data-test="moderation-blocked-back" @click="closeModerationBlocked">
            {{ t('moderationBlocked.back') }}
          </button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
