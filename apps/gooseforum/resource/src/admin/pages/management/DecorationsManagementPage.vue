<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Edit3, Loader2, Palette, Plus, RefreshCw, Search, X } from '@lucide/vue'
import type { DecorationStyle } from '@gooseforum/client'
import { getDecorations, saveDecoration } from '@/admin/runtime/api'
import { BasicPage } from '@/admin/components/global-layout'
import AdminSelect from '@/admin/components/AdminSelect.vue'
import DecorationStylePreview from '@/admin/components/DecorationStylePreview.vue'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import { Badge } from '@/admin/components/ui/badge'
import { Switch } from '@/admin/components/ui/switch'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/admin/components/ui/dialog'
import ManagementTable from './ManagementTable.vue'
import { decorationAssetScale, hasNicknameContrast, isSafeDecorationAssetUrl } from '@/site/utils/decoration-style'

const { t } = useI18n()
const items = ref<DecorationStyle[]>([])
const loading = ref(false)
const loaded = ref(false)
const loadError = ref('')
const saveError = ref('')
const saving = ref(false)
const saved = ref(false)
const open = ref(false)
const editing = ref(false)
const submitted = ref(false)
const search = ref('')
const kindFilter = ref('all')
const statusFilter = ref('all')
const fresh = (): DecorationStyle => ({ code: '', name: '', kind: 'avatarFrame', lightColors: ['#166534', '#0D9488'], darkColors: ['#86EFAC', '#2DD4BF'], assetUrl: '', assetScale: 1.25, isEnabled: true, sortOrder: 100 })
const form = reactive(fresh())
const scalePercent = computed({
  get: () => Number(form.assetScale ?? 1.25) * 100,
  set: (percent: number) => { form.assetScale = percent / 100 },
})
const palettes = ['lightColors', 'darkColors'] as const
const kindOptions = computed(() => [
  { value: 'avatarFrame', label: t('decoration.avatarFrame') },
  { value: 'nicknameStyle', label: t('decoration.nicknameStyle') },
])
const filtered = computed(() => items.value.filter(item => {
  const query = search.value.trim().toLowerCase()
  return (!query || `${item.name} ${item.code}`.toLowerCase().includes(query))
    && (kindFilter.value === 'all' || item.kind === kindFilter.value)
    && (statusFilter.value === 'all' || item.isEnabled === (statusFilter.value === 'enabled'))
}).sort((a, b) => a.sortOrder - b.sortOrder || a.code.localeCompare(b.code)))
const errors = computed(() => {
  const result: Record<string, string> = {}
  if (!/^[a-z][a-z0-9_-]{0,63}$/.test(form.code)) result.code = t('decoration.configAdmin.codeHint')
  if (!form.name.trim() || Array.from(form.name.trim()).length > 80) result.name = t('decoration.configAdmin.nameInvalid')
  for (const palette of palettes) {
    const colors = form[palette]
    if (colors.length < 1 || colors.length > 2 || colors.some(color => !/^#[0-9a-f]{6}$/i.test(color))) result[palette] = t('decoration.configAdmin.colorInvalid')
    else if (form.kind === 'nicknameStyle') {
      const backgrounds = palette === 'lightColors' ? [[255, 255, 255], [243, 244, 246]] : [[17, 24, 39], [31, 41, 55]]
      if (backgrounds.some(background => !hasNicknameContrast(colors, background))) result[palette] = t('decoration.configAdmin.contrastInvalid')
    }
  }
  if (!Number.isFinite(form.assetScale) || Number(form.assetScale) < 0.5 || Number(form.assetScale) > 2) result.assetScale = t('decoration.configAdmin.scaleHint')
  if (form.assetUrl && !isSafeDecorationAssetUrl(form.assetUrl)) result.assetUrl = t('decoration.configAdmin.assetInvalid')
  if (!Number.isInteger(Number(form.sortOrder)) || Number(form.sortOrder) < 0 || Number(form.sortOrder) > 100000) result.sortOrder = t('decoration.configAdmin.orderHint')
  return result
})
watch(() => form.kind, kind => { if (kind === 'nicknameStyle') { form.assetUrl = ''; form.assetScale = 1.25 } })
function edit(item?: DecorationStyle) {
  if (saving.value) return
  const source = item ?? fresh()
  Object.assign(form, source, { lightColors: [...source.lightColors], darkColors: [...source.darkColors], assetUrl: source.assetUrl ?? '', assetScale: decorationAssetScale(source) })
  editing.value = !!item
  submitted.value = false
  saved.value = false
  saveError.value = ''
  open.value = true
}
function close(value: boolean) { if (!saving.value) open.value = value }
async function load() {
  if (loading.value) return
  loading.value = true
  loadError.value = ''
  try { items.value = await getDecorations(); loaded.value = true }
  catch (error) { loadError.value = error instanceof Error ? error.message : t('decoration.loadFailed') }
  finally { loading.value = false }
}
async function save() {
  if (saving.value) return
  submitted.value = true
  saveError.value = ''
  if (Object.keys(errors.value).length) return
  saving.value = true
  try {
    const updated = await saveDecoration({ ...form, name: form.name.trim(), sortOrder: Number(form.sortOrder), lightColors: [...form.lightColors], darkColors: [...form.darkColors] })
    // Use the confirmed write response; a second list read cannot invalidate a successful save.
    const index = items.value.findIndex(item => item.code === updated.code)
    if (index === -1) items.value.push(updated)
    else items.value[index] = updated
    loaded.value = true
    open.value = false
    saved.value = true
  } catch (error) { saveError.value = error instanceof Error ? error.message : t('decoration.saveFailed') }
  finally { saving.value = false }
}
onMounted(load)
</script>

<template>
  <BasicPage :title="t('decoration.title')" :description="t('decoration.configAdmin.description')" sticky>
    <template #actions>
      <Button type="button" variant="outline" :disabled="loading || saving" @click="load"><RefreshCw class="size-4" :class="{ 'animate-spin': loading }" />{{ t('decoration.configAdmin.refresh') }}</Button>
      <Button type="button" :disabled="!loaded || loading || saving" @click="edit()"><Plus class="size-4" />{{ t('decoration.new') }}</Button>
    </template>
    <div class="space-y-4">
      <div v-if="saved" role="status" class="flex items-center gap-2 rounded-lg border border-emerald-600/20 bg-emerald-500/5 p-3 text-sm text-emerald-700 dark:text-emerald-400"><Check class="size-4" />{{ t('decoration.saved') }}</div>
      <div class="rounded-lg border bg-card p-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-2"><Palette class="size-4 text-muted-foreground" /><h2 class="font-semibold">{{ t('decoration.configAdmin.catalog') }}</h2><Badge variant="secondary">{{ items.length }}</Badge></div>
          <p class="text-xs text-muted-foreground">{{ t('decoration.configAdmin.catalogHint') }}</p>
        </div>
        <div class="mt-4 grid gap-3 sm:grid-cols-[1fr_160px_160px]">
          <div class="relative"><Search class="pointer-events-none absolute left-3 top-2.5 size-4 text-muted-foreground" /><Input v-model="search" :aria-label="t('decoration.configAdmin.search')" :placeholder="t('decoration.configAdmin.search')" class="pl-9" /></div>
          <AdminSelect v-model="kindFilter" :aria-label="t('decoration.kind')" :options="[{ value: 'all', label: t('decoration.configAdmin.allKinds') }, ...kindOptions]" />
          <AdminSelect v-model="statusFilter" :aria-label="t('decoration.configAdmin.status')" :options="[{ value: 'all', label: t('decoration.configAdmin.allStatuses') }, { value: 'enabled', label: t('decoration.enabled') }, { value: 'disabled', label: t('decoration.disabled') }]" />
        </div>
      </div>
      <p v-if="loadError" role="alert" class="sr-only">{{ loadError }}</p>
      <ManagementTable :columns="[t('decoration.configAdmin.appearance'), t('decoration.name'), t('decoration.kind'), t('decoration.configAdmin.status'), t('decoration.sortOrder'), t('decoration.action')]" :loading="loading" :error="loadError" :show-pagination="false" @retry="load">
        <tr v-for="item in filtered" :key="item.code">
          <td class="w-64 p-4"><DecorationStylePreview :style="item" compact /></td>
          <td class="p-4"><p class="font-medium">{{ item.name }}</p><p class="mt-1 font-mono text-xs text-muted-foreground">{{ item.code }}</p><p v-if="item.assetUrl" class="mt-2 max-w-56 break-all text-xs text-muted-foreground">{{ item.assetUrl }}</p></td>
          <td class="p-4 text-muted-foreground">{{ item.kind === 'avatarFrame' ? t('decoration.avatarFrame') : t('decoration.nicknameStyle') }}</td>
          <td class="p-4"><Badge :variant="item.isEnabled ? 'secondary' : 'outline'">{{ item.isEnabled ? t('decoration.enabled') : t('decoration.disabled') }}</Badge></td>
          <td class="p-4 tabular-nums text-muted-foreground">{{ item.sortOrder }}</td>
          <td class="p-4"><Button type="button" variant="outline" size="sm" :disabled="saving" @click="edit(item)"><Edit3 class="size-4" />{{ t('decoration.configAdmin.edit') }}</Button></td>
        </tr>
        <tr v-if="!filtered.length"><td colspan="6" class="px-4 py-14 text-center"><Palette class="mx-auto mb-3 size-8 text-muted-foreground/50" /><p class="font-medium">{{ t(items.length ? 'decoration.configAdmin.noMatches' : 'decoration.configAdmin.empty') }}</p><p class="mt-2 text-sm text-muted-foreground">{{ t(items.length ? 'decoration.configAdmin.filterHint' : 'decoration.configAdmin.emptyHint') }}</p></td></tr>
      </ManagementTable>
    </div>

    <Dialog :open="open" @update:open="close">
      <DialogContent class="flex max-h-[90dvh] flex-col overflow-hidden sm:max-w-4xl" :show-close-button="!saving" @escape-key-down="saving && $event.preventDefault()" @interact-outside="saving && $event.preventDefault()">
        <DialogHeader>
          <DialogTitle>{{ editing ? t('decoration.configAdmin.edit') : t('decoration.new') }}</DialogTitle>
          <DialogDescription>{{ t('decoration.configAdmin.editorHint') }}</DialogDescription>
        </DialogHeader>
        <form class="flex min-h-0 flex-col gap-4" @submit.prevent="save">
          <div class="grid min-h-0 gap-6 overflow-y-auto pr-1 lg:grid-cols-[1fr_280px]">
            <div class="space-y-5">
              <section class="space-y-3">
                <h3 class="text-sm font-semibold">{{ t('decoration.configAdmin.identity') }}</h3>
                <div class="grid gap-4 sm:grid-cols-2">
                  <div class="space-y-2"><label for="decoration-code" class="text-sm font-medium">{{ t('decoration.code') }}</label><Input id="decoration-code" v-model="form.code" required maxlength="64" :disabled="editing || saving" :aria-invalid="submitted && !!errors.code" aria-describedby="decoration-code-hint" placeholder="tea-garden-v1" /><p id="decoration-code-hint" class="text-xs text-muted-foreground">{{ t('decoration.configAdmin.codeHint') }}</p></div>
                  <div class="space-y-2"><label for="decoration-name" class="text-sm font-medium">{{ t('decoration.name') }}</label><Input id="decoration-name" v-model="form.name" required maxlength="80" :disabled="saving" :aria-invalid="submitted && !!errors.name" /></div>
                </div>
                <div class="space-y-2"><label for="decoration-kind" class="text-sm font-medium">{{ t('decoration.kind') }}</label><AdminSelect id="decoration-kind" v-model="form.kind" :options="kindOptions" :disabled="editing || saving" class="w-full" /></div>
              </section>
              <section class="space-y-3 border-t pt-4">
                <h3 class="text-sm font-semibold">{{ t('decoration.configAdmin.palette') }}</h3>
                <p class="text-xs text-muted-foreground">{{ t('decoration.configAdmin.paletteHint') }}</p>
                <fieldset v-for="palette in palettes" :key="palette" :disabled="saving" class="space-y-2">
                  <legend class="mb-2 text-sm font-medium">{{ t(`decoration.${palette}`) }}</legend>
                  <div v-for="(_, index) in form[palette]" :key="index" class="flex items-center gap-2">
                    <input :value="/^#[0-9a-f]{6}$/i.test(form[palette][index]) ? form[palette][index] : '#000000'" type="color" :aria-label="`${t(`decoration.${palette}`)} ${index + 1}`" class="size-9 shrink-0 cursor-pointer rounded-md border border-input bg-background p-1" @input="form[palette][index] = ($event.target as HTMLInputElement).value" />
                    <Input :id="`decoration-${palette}-${index}`" v-model="form[palette][index]" maxlength="7" :aria-label="`${t(`decoration.${palette}`)} ${index + 1} HEX`" :aria-invalid="submitted && !!errors[palette]" :aria-describedby="errors[palette] ? `error-${palette}` : undefined" class="font-mono" placeholder="#RRGGBB" />
                    <Button v-if="index > 0" type="button" variant="ghost" size="icon" :aria-label="t('decoration.configAdmin.removeColor')" @click="form[palette].splice(index, 1)"><X class="size-4" /></Button>
                  </div>
                  <Button v-if="form[palette].length < 2" type="button" variant="outline" size="sm" @click="form[palette].push(form[palette][0])"><Plus class="size-3" />{{ t('decoration.configAdmin.addColor') }}</Button>
                  <p v-if="submitted && errors[palette]" :id="`error-${palette}`" class="text-xs text-destructive">{{ errors[palette] }}</p>
                </fieldset>
              </section>
              <section v-if="form.kind === 'avatarFrame'" class="space-y-2 border-t pt-4"><label for="decoration-asset" class="text-sm font-medium">{{ t('decoration.assetUrl') }}</label><Input id="decoration-asset" v-model="form.assetUrl" :disabled="saving" maxlength="160" placeholder="/static/decorations/tea-garden-v1.png" :aria-invalid="submitted && !!errors.assetUrl" aria-describedby="decoration-asset-hint" /><p id="decoration-asset-hint" class="text-xs leading-relaxed text-muted-foreground">{{ t('decoration.assetHint') }}</p><p v-if="submitted && errors.assetUrl" class="text-xs text-destructive">{{ errors.assetUrl }}</p>
                <div class="space-y-3 rounded-lg border bg-muted/20 p-3">
                  <div class="flex flex-wrap items-center justify-between gap-2"><label for="decoration-scale" class="text-sm font-medium">{{ t('decoration.configAdmin.scale') }}</label><Button type="button" variant="ghost" size="sm" :disabled="saving" @click="form.assetScale = 1.25">{{ t('decoration.configAdmin.resetScale') }}</Button></div>
                  <div class="flex items-center gap-3"><input id="decoration-scale" v-model.number="scalePercent" type="range" min="50" max="200" step="1" :disabled="saving" class="min-w-0 flex-1 accent-primary" aria-describedby="decoration-scale-hint" /><div class="flex w-24 items-center gap-2"><Input v-model.number="scalePercent" type="number" min="50" max="200" step="1" :disabled="saving" :aria-label="t('decoration.configAdmin.scale')" :aria-invalid="submitted && !!errors.assetScale" aria-describedby="decoration-scale-hint" /><span class="text-xs text-muted-foreground">%</span></div></div>
                  <p id="decoration-scale-hint" class="text-xs leading-relaxed text-muted-foreground">{{ t('decoration.configAdmin.scaleHint') }}</p>
                </div>
              </section>
              <section class="grid gap-4 border-t pt-4 sm:grid-cols-2">
                <div class="space-y-2"><label for="decoration-order" class="text-sm font-medium">{{ t('decoration.sortOrder') }}</label><Input id="decoration-order" v-model="form.sortOrder" type="number" min="0" max="100000" step="1" :disabled="saving" :aria-invalid="submitted && !!errors.sortOrder" aria-describedby="decoration-order-hint" /><p id="decoration-order-hint" class="text-xs text-muted-foreground">{{ t('decoration.configAdmin.orderHint') }}</p></div>
                <div class="space-y-2"><label for="decoration-enabled" class="text-sm font-medium">{{ t('decoration.enabled') }}</label><div class="flex h-9 items-center justify-between rounded-md border px-3"><span class="text-sm">{{ t(form.isEnabled ? 'decoration.enabled' : 'decoration.disabled') }}</span><Switch id="decoration-enabled" v-model="form.isEnabled" :disabled="saving" /></div></div>
              </section>
            </div>
            <aside class="space-y-3 rounded-lg border bg-muted/20 p-4 lg:sticky lg:top-0 lg:self-start">
              <h3 class="text-sm font-semibold">{{ t('decoration.configAdmin.preview') }}</h3>
              <DecorationStylePreview :style="form" />
              <p class="text-xs leading-relaxed text-muted-foreground">{{ t('decoration.configAdmin.previewHint') }}</p>
              <p v-if="!form.isEnabled" class="rounded-md border p-3 text-xs text-muted-foreground">{{ t('decoration.configAdmin.disabledHint') }}</p>
            </aside>
          </div>
          <div v-if="saveError || (submitted && Object.keys(errors).length)" role="alert" class="rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{{ saveError || t('decoration.configAdmin.fixErrors') }}</div>
          <DialogFooter class="border-t pt-4"><Button type="button" variant="outline" :disabled="saving" @click="close(false)">{{ t('decoration.configAdmin.cancel') }}</Button><Button type="submit" :disabled="saving"><Loader2 v-if="saving" class="size-4 animate-spin" />{{ t(saving ? 'decoration.configAdmin.saving' : 'decoration.save') }}</Button></DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </BasicPage>
</template>
