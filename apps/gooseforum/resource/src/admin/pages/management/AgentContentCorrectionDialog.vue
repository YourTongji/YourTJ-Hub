<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Checkbox } from '@/admin/components/ui/checkbox'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import { Textarea } from '@/admin/components/ui/textarea'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/admin/components/ui/dialog'
import { editBotContent, getBotContentSource } from '@/admin/runtime/api'
import { resolveApiMessage } from '@/runtime/api-message'
import type { AdminAgent, BotContentSource } from '@/admin/types'

const props = defineProps<{ agent: AdminAgent, categories?: Array<{ id: number, label: string }> }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const kind = ref<'topic' | 'post'>('topic')
const id = ref('')
const source = ref<BotContentSource | null>(null)
const loadedKind = ref<'topic' | 'post'>('topic')
const busy = ref(false)
const error = ref('')
const notice = ref('')
const form = reactive({ title: '', content: '', categoryId: [] as number[] })
const categoryOptions = computed(() => {
  const options = [...(props.categories ?? [])]
  for (const id of source.value?.categoryId ?? []) {
    if (!options.some(option => option.id === id)) options.push({ id, label: t('agentContent.categoryFallback', { id }) })
  }
  return options
})
function toggleCategory(id: number, checked: boolean | 'indeterminate') {
  if (checked === true && !form.categoryId.includes(id) && form.categoryId.length < 3) form.categoryId.push(id)
  if (checked === false) form.categoryId = form.categoryId.filter(value => value !== id)
}
const validId = computed(() => /^\d+$/.test(id.value) && Number.isSafeInteger(Number(id.value)) && Number(id.value) > 0)

async function load() {
  if (!validId.value || busy.value) return
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    const result = await getBotContentSource(kind.value === 'topic' ? { topicId: Number(id.value) } : { postId: Number(id.value) })
    if (result.botUserId !== props.agent.agentId) throw new Error(t('agentContent.wrongBot'))
    source.value = result
    loadedKind.value = kind.value
    Object.assign(form, { title: result.title, content: result.content, categoryId: [...result.categoryId] })
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('agentContent.failed')
  } finally {
    busy.value = false
  }
}

async function save() {
  if (!source.value || busy.value) return
  const categoryId = form.categoryId
  if (loadedKind.value === 'topic' && (categoryId.length < 1 || categoryId.length > 3 || categoryId.some(value => !Number.isSafeInteger(value) || value <= 0) || new Set(categoryId).size !== categoryId.length)) {
    error.value = t('agentContent.categoriesInvalid')
    return
  }
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    const result = await editBotContent(loadedKind.value, {
      ...(loadedKind.value === 'topic' ? { topicId: source.value.topicId, title: form.title, categoryId } : { postId: source.value.postId }),
      content: form.content,
      expectedRevisionNo: source.value.revisionNo,
    })
    notice.value = result.messageCode === 'content.moderation.checking'
      ? resolveApiMessage(result, t('agentContent.pending'))
      : t('agentContent.saved')
    source.value = null
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('agentContent.failed')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <Dialog :open="true" @update:open="(open) => !open && !busy && emit('close')">
    <DialogContent class="max-h-[90dvh] overflow-y-auto sm:max-w-2xl" :show-close-button="!busy">
      <DialogHeader>
        <DialogTitle>{{ t('agentContent.open') }} · {{ agent.nickname || agent.username }}</DialogTitle>
        <DialogDescription>{{ t('agentContent.description') }}</DialogDescription>
      </DialogHeader>
      <form class="grid gap-3" @submit.prevent="load">
        <div class="flex gap-2">
          <Button type="button" :variant="kind === 'topic' ? 'secondary' : 'outline'" :disabled="busy" @click="kind = 'topic'">{{ t('agentContent.topic') }}</Button>
          <Button type="button" :variant="kind === 'post' ? 'secondary' : 'outline'" :disabled="busy" @click="kind = 'post'">{{ t('agentContent.post') }}</Button>
        </div>
        <label class="grid gap-2 text-sm">{{ kind === 'topic' ? t('agentContent.topicId') : t('agentContent.postId') }}
          <Input v-model="id" inputmode="numeric" :disabled="busy" />
        </label>
        <p v-if="source" class="text-xs text-muted-foreground">{{ t('agentContent.reloadWarning') }}</p>
        <Button class="w-fit" type="submit" variant="outline" :disabled="busy || !validId">{{ t('agentContent.load') }}</Button>
      </form>
      <form v-if="source" class="grid gap-3 border-t pt-3" @submit.prevent="save">
        <p class="text-sm text-muted-foreground">{{ t('agentContent.loaded', { topic: source.topicId, post: source.postId, revision: source.revisionNo }) }}</p>
        <template v-if="loadedKind === 'topic'">
          <label class="grid gap-2 text-sm">{{ t('agentContent.title') }}<Input v-model="form.title" :disabled="busy" /></label>
          <fieldset class="grid gap-2">
            <legend class="mb-2 text-sm">{{ t('agentContent.categories') }}</legend>
            <label v-for="category in categoryOptions" :key="category.id" class="flex items-center gap-2 text-sm">
              <Checkbox :model-value="form.categoryId.includes(category.id)" :disabled="busy || (!form.categoryId.includes(category.id) && form.categoryId.length >= 3)" @update:model-value="toggleCategory(category.id, $event)" />
              {{ category.label }}
            </label>
          </fieldset>
        </template>
        <label class="grid gap-2 text-sm">{{ t('agentContent.content') }}<Textarea v-model="form.content" class="min-h-64 font-mono" :disabled="busy" required /></label>
        <Button class="w-fit" type="submit" :disabled="busy || !form.content.trim()">{{ t('agentContent.save') }}</Button>
      </form>
      <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
      <p v-if="notice" role="status" class="text-sm">{{ notice }}</p>
      <DialogFooter><Button type="button" variant="outline" :disabled="busy" @click="emit('close')">{{ t('common.close') }}</Button></DialogFooter>
    </DialogContent>
  </Dialog>
</template>
