<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle, DialogTrigger } from 'reka-ui'
import type { UserCardPayload } from '@gooseforum/client'
import { updateAgent } from '@/admin/runtime/api'
import { useAvatarCropUpload } from '@/site/composables/useAvatarCropUpload'
import { useCoverCropUpload } from '@/site/composables/useCoverCropUpload'
import AvatarImageEditor from './AvatarImageEditor.vue'
import CoverImageEditor from './CoverImageEditor.vue'
import ProfileManageButton from './ProfileManageButton.vue'
import UserAvatar from './UserAvatar.vue'

const props = defineProps<{ user: UserCardPayload }>()
const emit = defineEmits<{ saved: [] }>()
const { t } = useI18n()
const open = ref(false)
const saving = ref(false)
const error = ref('')
const avatarEditor = ref<InstanceType<typeof AvatarImageEditor> | null>(null)
const coverEditor = ref<InstanceType<typeof CoverImageEditor> | null>(null)
const draft = reactive({ nickname: '', avatarUrl: '', profileCoverUrl: '', bio: '', signature: '', website: '', websiteName: '' })
const fields = ['nickname', 'bio', 'signature', 'websiteName', 'website'] as const
const presets = Array.from({ length: 12 }, (_, index) => `/static/pic/${index + 1}.webp`)
const onError = (message: string) => { error.value = message }
const { avatarInput, uploadingAvatar, avatarCropOpen, avatarImageUrl, cropError, chooseAvatar, handleAvatarChange, closeAvatarCrop, saveAvatarFromCanvas } = useAvatarCropUpload({ agentId: props.user.userId, initialAvatarUrl: props.user.avatarUrl, onStatus: () => {}, onError })
const { coverInput, uploadingCover, coverCropOpen, coverImageUrl, coverCropError, chooseCover, handleCoverChange, closeCoverCrop, saveCoverFromCanvas } = useCoverCropUpload({ onStatus: () => {}, onError })
const busy = computed(() => saving.value || uploadingAvatar.value || uploadingCover.value)
let uploadedAvatar = false

function changeOpen(value: boolean) {
  if (busy.value) return
  if (value) {
    for (const key of Object.keys(draft) as Array<keyof typeof draft>) draft[key] = props.user[key] || ''
    error.value = ''
    uploadedAvatar = false
  } else {
    closeAvatarCrop()
    closeCoverCrop()
    if (uploadedAvatar) emit('saved')
  }
  open.value = value
}
async function saveAvatar(canvas: HTMLCanvasElement) {
  const url = await saveAvatarFromCanvas(canvas)
  if (!url) return
  draft.avatarUrl = url
  uploadedAvatar = true
  closeAvatarCrop()
}
async function saveCover(canvas: HTMLCanvasElement) {
  const url = await saveCoverFromCanvas(canvas)
  if (!url) return
  draft.profileCoverUrl = url
  closeCoverCrop()
}
async function save() {
  if (busy.value) return
  saving.value = true
  error.value = ''
  try {
    await updateAgent({ agentId: props.user.userId, ...draft })
    open.value = false
    emit('saved')
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('api.operationFailed')
  } finally {
    saving.value = false
  }
}
</script>
<template>
  <DialogRoot :open="open" @update:open="changeOpen">
    <DialogTrigger as-child><ProfileManageButton :label="t('agentProfile.title')" /></DialogTrigger>
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[2200] bg-black/40" />
      <DialogContent class="fixed left-1/2 top-1/2 z-[2201] max-h-[90dvh] w-[calc(100vw_-_2rem)] max-w-xl -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-base-100 p-5 shadow-xl" @escape-key-down="busy && $event.preventDefault()" @pointer-down-outside="busy && $event.preventDefault()">
        <DialogTitle class="text-lg font-semibold">{{ t('agentProfile.title') }}</DialogTitle>
        <DialogDescription class="mt-1 text-sm text-base-content/65">{{ t('agentProfile.help') }}</DialogDescription>
        <p v-if="error" role="alert" class="mt-3 text-sm text-error">{{ error }}</p>
        <form class="mt-4 space-y-4" @submit.prevent="save">
          <fieldset :disabled="busy" class="min-w-0 space-y-4">
            <div class="flex items-center gap-3">
              <UserAvatar :src="draft.avatarUrl" :alt="draft.nickname || user.username" class="h-16 w-16 rounded-full" />
              <button type="button" class="gf-button gf-button-sm gf-button-secondary" @click="chooseAvatar">{{ t('settings.avatar.uploadCustom') }}</button>
            </div>
            <div class="flex gap-2 overflow-x-auto pb-1" :aria-label="t('settings.avatar.presetsTitle')">
              <button v-for="url in presets" :key="url" type="button" class="h-10 w-10 shrink-0 rounded border border-line p-0.5" :aria-pressed="draft.avatarUrl === url" :aria-label="t('settings.avatar.selectPreset')" @click="draft.avatarUrl = url"><UserAvatar :src="url" :alt="''" class="h-full w-full rounded" /></button>
            </div>
            <div v-if="avatarCropOpen" class="space-y-2">
              <AvatarImageEditor ref="avatarEditor" :image-url="avatarImageUrl" :saving="uploadingAvatar" @save="saveAvatar" @cancel="closeAvatarCrop" />
              <p v-if="cropError" role="alert" class="text-sm text-error">{{ cropError }}</p>
              <div class="flex gap-2">
                <button type="button" class="gf-button gf-button-sm gf-button-primary" @click="avatarEditor?.save()">{{ t('common.save') }}</button>
                <button type="button" class="gf-button gf-button-sm gf-button-muted" @click="closeAvatarCrop">{{ t('common.cancel') }}</button>
              </div>
            </div>
            <div class="space-y-2">
              <img v-if="draft.profileCoverUrl && !coverCropOpen" :src="draft.profileCoverUrl" :alt="t('user.editCover')" class="aspect-[5/1] w-full rounded-lg object-cover" />
              <label class="block text-sm">{{ t('user.editCover') }}
                <input v-model="draft.profileCoverUrl" name="profileCoverUrl" type="text" inputmode="url" maxlength="512" class="gf-input mt-1 w-full" />
              </label>
              <div class="flex flex-wrap gap-2">
                <button type="button" class="gf-button gf-button-sm gf-button-secondary" @click="chooseCover">{{ t('user.editCover') }}</button>
                <button v-if="draft.profileCoverUrl" type="button" class="gf-button gf-button-sm gf-button-muted" @click="draft.profileCoverUrl = ''">{{ t('settings.profile.confirmClearAction') }}</button>
              </div>
              <div v-if="coverCropOpen" class="space-y-2">
                <CoverImageEditor ref="coverEditor" :image-url="coverImageUrl" :saving="uploadingCover" @save="saveCover" @cancel="closeCoverCrop" />
                <div class="flex gap-2">
                  <button type="button" class="gf-button gf-button-sm gf-button-primary" @click="coverEditor?.save()">{{ t('common.save') }}</button>
                  <button type="button" class="gf-button gf-button-sm gf-button-muted" @click="closeCoverCrop">{{ t('common.cancel') }}</button>
                </div>
              </div>
              <p v-if="coverCropError" role="alert" class="text-sm text-error">{{ coverCropError }}</p>
            </div>
            <label v-for="field in fields" :key="field" class="block text-sm">
              {{ t(`settings.profile.${field === 'nickname' ? 'displayName' : field}`) }}
              <textarea v-if="field === 'bio' || field === 'signature'" v-model="draft[field]" :name="field" :maxlength="field === 'bio' ? 500 : 255" rows="2" class="gf-textarea mt-1 w-full" />
              <input v-else v-model="draft[field]" :name="field" :type="field === 'website' ? 'url' : 'text'" :maxlength="field === 'nickname' || field === 'websiteName' ? 64 : 255" class="gf-input mt-1 w-full" />
            </label>
          </fieldset>
          <div class="flex justify-end gap-2">
            <button type="button" class="gf-button gf-button-sm gf-button-muted" :disabled="busy" @click="changeOpen(false)">{{ t('common.cancel') }}</button>
            <button type="submit" class="gf-button gf-button-sm gf-button-primary" :disabled="busy || avatarCropOpen || coverCropOpen">{{ saving ? t('settings.savingShort') : t('settings.profile.save') }}</button>
          </div>
        </form>
        <input ref="avatarInput" type="file" accept="image/*" class="hidden" @change="handleAvatarChange" />
        <input ref="coverInput" type="file" accept="image/*" class="hidden" @change="handleCoverChange" />
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
