<script setup lang="ts">
import { ref } from 'vue'
import { Eye, EyeOff, LockKeyhole } from '@lucide/vue'
import { useI18n } from 'vue-i18n'

/**
 * 带左侧锁图标与右侧明文/密文切换按钮的密码输入框（issue #375）。
 * 兼容现有内联写法：外层 relative block 包裹、左侧图标绝对定位，
 * 右侧切换按钮不干扰左侧定位与表单提交。v-model 绑定到输入值。
 */
const model = defineModel<string>({ default: '' })

defineProps<{
  placeholder?: string
  autocomplete?: string
  /** 可访问名称（sr-only label 的文本），同时作为切换按钮的 aria-label 来源 */
  label?: string
}>()

const { t } = useI18n()
const show = ref(false)
</script>

<template>
  <span class="relative block">
    <LockKeyhole class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/55" />
    <input
      v-model="model"
      :type="show ? 'text' : 'password'"
      class="gf-input pl-10 pr-11"
      :placeholder="placeholder"
      :autocomplete="autocomplete"
      :aria-label="label"
    />
    <button
      type="button"
      class="absolute right-2 top-1/2 inline-flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-md text-base-content/55 transition-colors duration-150 hover:bg-base-300/60 hover:text-base-content focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-primary"
      :aria-label="show ? t('auth.hidePassword') : t('auth.showPassword')"
      :title="show ? t('auth.hidePassword') : t('auth.showPassword')"
      :aria-pressed="show"
      @click="show = !show"
    >
      <EyeOff v-if="show" class="h-4 w-4" />
      <Eye v-else class="h-4 w-4" />
    </button>
  </span>
</template>
