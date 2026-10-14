<script setup lang="ts">
import { computed, ref, watchPostEffect } from 'vue'
import type { DecorationStyle } from '@gooseforum/client'
import { decorationThemeRevision } from '@/site/utils/decoration-theme'
import { hasNicknameContrast, decorationVariables } from '@/site/utils/decoration-style'
const props = defineProps<{ styleDefinition?: DecorationStyle | null }>()
const element = ref<HTMLElement>()
const readable = ref(false)
watchPostEffect(() => {
  void decorationThemeRevision.value
  const colors = document.documentElement.dataset.theme === 'gf-dark' ? props.styleDefinition?.darkColors : props.styleDefinition?.lightColors
  readable.value = false
  if (!element.value || !colors) return
  const context = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
  if (!context) return
  for (let parent = element.value.parentElement; parent; parent = parent.parentElement) {
    const style = getComputedStyle(parent)
    if (style.backgroundImage !== 'none') return
    context.clearRect(0, 0, 1, 1); context.fillStyle = style.backgroundColor; context.fillRect(0, 0, 1, 1)
    const pixel = context.getImageData(0, 0, 1, 1).data
    if (!pixel[3]) continue
    if (pixel[3] < 255) return
    readable.value = hasNicknameContrast(colors, [pixel[0], pixel[1], pixel[2]])
    return
  }
})
const variables = computed(() => decorationVariables(readable.value && props.styleDefinition?.kind === 'nicknameStyle' ? props.styleDefinition : null))
</script>
<template><span ref="element" :class="{ 'gf-decorated-nickname': Object.keys(variables).length > 0 }" :style="variables"><slot /></span></template>
<style scoped>
.gf-decorated-nickname { background-image: var(--decoration-light); background-clip: text; -webkit-background-clip: text; color: transparent; }
:global([data-theme="gf-dark"] .gf-decorated-nickname) { background-image: var(--decoration-dark); }
@media (forced-colors: active) { .gf-decorated-nickname { background: none; color: CanvasText; } }
</style>
