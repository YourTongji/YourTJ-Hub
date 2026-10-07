<script setup lang="ts">
import { Settings } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
defineProps<{ href?: string; label?: string }>()
const emit = defineEmits<{ click: [event: MouseEvent] }>()
const { t } = useI18n()
</script>
<template>
  <component :is="href ? 'a' : 'button'" :href="href" :type="href ? undefined : 'button'"
    class="gf-button gf-button-md gf-button-secondary" @click="emit('click', $event)">
    <Settings class="h-4 w-4 shrink-0" />{{ label || t('user.editProfile') }}
  </component>
</template>
