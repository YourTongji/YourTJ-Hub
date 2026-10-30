<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DecorationStyle } from '@gooseforum/client'
import UserAvatar from '@/site/components/UserAvatar.vue'
import { decorationVariables } from '@/runtime/decoration-style'

const props = defineProps<{ style: DecorationStyle; compact?: boolean }>()
const { t } = useI18n()
const themes = computed(() => [
  { key: 'light', colors: props.style.lightColors },
  { key: 'dark', colors: props.style.darkColors },
].map(theme => {
  // Preview both palettes independently of the administrator's current theme or enabled setting.
  const definition = { ...props.style, isEnabled: true, lightColors: theme.colors, darkColors: theme.colors }
  const gradient = decorationVariables(definition)['--decoration-light']
  // The save gate owns the contrast floor; the preview always shows the gradient the administrator typed.
  return { ...theme, definition, nicknameStyle: gradient ? { backgroundImage: gradient, backgroundClip: 'text', color: 'transparent' } : undefined }
}))
</script>

<template>
  <div class="space-y-2">
    <div class="grid grid-cols-2 gap-3">
      <div v-for="theme in themes" :key="theme.key" class="min-w-0 rounded-lg border p-3" :class="theme.key === 'light' ? 'border-slate-200 bg-white text-slate-900' : 'border-slate-700 bg-slate-900 text-slate-100'">
        <p class="mb-4 text-xs" :class="theme.key === 'light' ? 'text-slate-500' : 'text-slate-400'">{{ t(`decoration.configAdmin.${theme.key}Preview`) }}</p>
        <div class="flex min-w-0 items-center gap-3" :class="compact ? 'flex-col text-center' : 'flex-col py-5 text-center'">
          <UserAvatar src="/static/pic/default-avatar.webp" :alt="t('decoration.configAdmin.previewMember')" :frame="style.kind === 'avatarFrame' ? theme.definition : null" :class="compact ? 'size-9 rounded-full' : 'size-12 rounded-full'" img-class="rounded-full" />
          <span class="max-w-full break-words text-sm font-semibold" :style="style.kind === 'nicknameStyle' ? theme.nicknameStyle : undefined">{{ style.name || t('decoration.configAdmin.previewMember') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
