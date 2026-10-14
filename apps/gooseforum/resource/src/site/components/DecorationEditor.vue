<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DecorationState, UserDecoration } from '@gooseforum/client'
import { getDecorations, equipDecorations } from '@/runtime/api'
import UserAvatar from './UserAvatar.vue'
import UserNickname from './UserNickname.vue'
const props = defineProps<{ avatarUrl: string; name: string }>()
const emit = defineEmits<{ changed: [decoration: UserDecoration] }>()
const { t } = useI18n()
const state = ref<DecorationState>()
const selected = reactive({ avatarFrame: '', nicknameStyle: '' })
const loading = ref(false), saving = ref(false), error = ref(''), saved = ref(false)
const kinds = ['avatarFrame', 'nicknameStyle'] as const
async function load() {
  loading.value = true; error.value = ''
  try { state.value = await getDecorations(); Object.assign(selected, state.value.equipped) }
  catch (e) { error.value = e instanceof Error ? e.message : t('decoration.loadFailed') }
  finally { loading.value = false }
}
async function save() {
  saving.value = true; error.value = ''; saved.value = false
  try {
    state.value = await equipDecorations({ ...selected }); Object.assign(selected, state.value.equipped)
    emit('changed', { avatarFrame: state.value.catalog.find(item => item.code === selected.avatarFrame && item.isEnabled), nicknameStyle: state.value.catalog.find(item => item.code === selected.nicknameStyle && item.isEnabled) }); saved.value = true
  } catch (e) { error.value = e instanceof Error ? e.message : t('decoration.saveFailed') }
  finally { saving.value = false }
}
onMounted(load)
</script>
<template>
<section class="mt-4 space-y-3 border-t border-line pt-3">
  <h3 class="text-sm font-semibold">{{ t('decoration.title') }}</h3>
  <p class="text-xs text-base-content/60">{{ t('decoration.grantHint') }}</p>
  <p v-if="loading" role="status">{{ t('decoration.loading') }}</p>
  <template v-if="state">
    <fieldset v-for="kind in kinds" :key="kind" :disabled="saving" class="space-y-2">
      <legend class="mb-2 text-sm font-medium">{{ kind === 'avatarFrame' ? t('decoration.avatarFrame') : t('decoration.nicknameStyle') }}</legend>
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <label class="flex cursor-pointer items-center gap-2 rounded-lg border border-line p-3">
          <input v-model="selected[kind]" type="radio" :name="kind" value="" @change="saved = false" />{{ t('decoration.none') }}
        </label>
        <label v-for="item in state.catalog.filter(item => item.kind === kind)" :key="item.code" class="flex min-w-0 items-center gap-2 rounded-lg border border-line p-3" :class="{ 'opacity-60': !item.isEnabled || !state.ownedCodes.includes(item.code) }">
          <input v-model="selected[kind]" type="radio" :name="kind" :value="item.code" :disabled="!item.isEnabled || !state.ownedCodes.includes(item.code)" @change="saved = false" />
          <div class="min-w-0 space-y-2">
            <UserAvatar v-if="kind === 'avatarFrame'" :src="props.avatarUrl" :alt="props.name" :frame="item" class="h-10 w-10 rounded-full" img-class="rounded-full" />
            <UserNickname v-else :style-definition="item" class="text-sm font-semibold">{{ props.name }}</UserNickname>
            <p class="break-words text-xs">{{ item.name }}</p>
            <p v-if="!item.isEnabled" class="text-xs">{{ t('decoration.disabled') }}</p>
            <p v-else-if="!state.ownedCodes.includes(item.code)" class="text-xs">{{ t('decoration.locked') }}</p>
          </div>
        </label>
      </div>
    </fieldset>
    <button type="button" class="gf-button gf-button-primary" :disabled="saving" @click="save">{{ t('decoration.save') }}</button>
  </template>
  <button v-else-if="error" type="button" class="gf-button" @click="load">{{ t('decoration.retry') }}</button>
  <p v-if="error" role="alert" class="text-sm text-error">{{ error }}</p><p v-if="saved" role="status" class="text-sm text-success">{{ t('decoration.saved') }}</p>
</section>
</template>
