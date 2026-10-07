<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { EyeOff } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { getIdentityState } from '@/runtime/anonymous-identity'
const props = defineProps<{ viewerId: number }>()
const { t } = useI18n()
const href = ref('/settings?tab=privacy')
let generation = 0
onBeforeUnmount(() => { generation++ })
watch(() => props.viewerId, async (id) => {
  const request = ++generation
  href.value = '/settings?tab=privacy'
  if (!id) return
  try {
    const state = await getIdentityState()
    if (request === generation) href.value = state.persona?.profileUrl || '/settings?tab=privacy'
  } catch { /* The settings entry also offers retry when identity loading fails. */ }
}, { immediate: true })
</script>
<template>
  <a :href="href" class="gf-menu-item"><EyeOff class="h-4 w-4 text-icon-muted" />{{ t('anonymous.identity') }}</a>
</template>
