<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshCw } from '@lucide/vue'
import { BasicPage } from '@/admin/components/global-layout'
import { Badge } from '@/admin/components/ui/badge'
import { Button } from '@/admin/components/ui/button'
import { Input } from '@/admin/components/ui/input'
import { Switch } from '@/admin/components/ui/switch'
import {
  getAgentCommentPolicy,
  getTopicsList,
  saveAgentCommentPolicy,
  setAgentCommentTopicPolicy,
} from '@/admin/runtime/api'
import { adminToast } from '@/admin/runtime/toast'
import type { AdminPayload, AdminTopic, ManageHomeProps } from '@/admin/types'

defineProps<{
  payload: AdminPayload<ManageHomeProps>
}>()

const { t } = useI18n()
const pageSize = 10

const allowAgentComments = ref(true)
const policyLoading = ref(false)
const policyLoaded = ref(false)
const policySaving = ref(false)
const policyDirty = ref(false)
const policyError = ref('')

const topics = ref<AdminTopic[]>([])
const topicsLoading = ref(false)
const topicsError = ref('')
const page = ref(1)
const hasNext = ref(false)
const search = ref('')
const onlyDisabled = ref(false)
const toggling = ref<number | null>(null)
let topicsRequest = 0

async function loadPolicy() {
  if (policyLoading.value || policySaving.value) return
  policyLoaded.value = false
  policyLoading.value = true
  policyError.value = ''
  try {
    const policy = await getAgentCommentPolicy()
    allowAgentComments.value = policy.allowAgentComments
    policyLoaded.value = true
    policyDirty.value = false
  } catch (err) {
    policyError.value = err instanceof Error ? err.message : t('agentPolicy.loadFailed')
  } finally {
    policyLoading.value = false
  }
}

function onAllowChange(value: boolean | 'indeterminate') {
  allowAgentComments.value = value === true
  policyDirty.value = true
}

async function savePolicy() {
  if (!policyLoaded.value || policyLoading.value || policySaving.value || !policyDirty.value) return
  policySaving.value = true
  policyError.value = ''
  try {
    await saveAgentCommentPolicy(allowAgentComments.value)
    policyDirty.value = false
    adminToast.success(t('agentPolicy.saved'))
  } catch (err) {
    policyError.value = err instanceof Error ? err.message : t('agentPolicy.saveFailed')
  } finally {
    policySaving.value = false
  }
}

async function loadTopics() {
  if (toggling.value !== null) return
  const request = ++topicsRequest
  topicsLoading.value = true
  topicsError.value = ''
  try {
    const result = await getTopicsList({
      page: page.value,
      pageSize,
      search: search.value,
      agentCommentDisabled: onlyDisabled.value ? true : undefined,
    })
    if (request !== topicsRequest) return
    topics.value = result.list
    hasNext.value = Boolean(result.hasNext)
  } catch (err) {
    if (request !== topicsRequest) return
    topicsError.value = err instanceof Error ? err.message : t('agentPolicy.loadFailed')
  } finally {
    if (request === topicsRequest) topicsLoading.value = false
  }
}

async function toggleTopic(topic: AdminTopic, disabled: boolean) {
  if (toggling.value !== null || topicsLoading.value) return
  toggling.value = topic.id
  try {
    await setAgentCommentTopicPolicy(topic.id, disabled)
    topics.value = topics.value.map(item => (item.id === topic.id ? { ...item, agentCommentDisabled: disabled } : item))
    if (onlyDisabled.value && !disabled) {
      topics.value = topics.value.filter(item => item.id !== topic.id)
    }
    adminToast.success(t(disabled ? 'agentPolicy.topicDisabled' : 'agentPolicy.topicAllowed', { title: topic.title }))
  } catch (err) {
    adminToast.error(err, t('agentPolicy.toggleFailed'))
  } finally {
    toggling.value = null
  }
}

function applyFilters() {
  page.value = 1
  loadTopics()
}

function toggleOnlyDisabled() {
  onlyDisabled.value = !onlyDisabled.value
  applyFilters()
}

function changePage(delta: number) {
  page.value = Math.max(1, page.value + delta)
  loadTopics()
}

onMounted(() => {
  loadPolicy()
  loadTopics()
})
</script>

<template>
  <BasicPage :title="t('agentPolicy.title')" :description="t('agentPolicy.description')" sticky>
    <template #actions>
      <Button variant="outline" type="button" :disabled="policyLoading || policySaving || topicsLoading || toggling !== null" @click="loadPolicy(); loadTopics()">
        <RefreshCw class="size-4" />
        {{ t('agentPolicy.refresh') }}
      </Button>
    </template>

    <section class="rounded-lg border bg-card p-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-medium">{{ t('agentPolicy.globalTitle') }}</h2>
          <p class="mt-1 max-w-2xl text-xs text-muted-foreground">{{ t('agentPolicy.globalHint') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <span class="text-sm">{{ t('agentPolicy.allowAgentComments') }}</span>
          <Switch :model-value="allowAgentComments" :disabled="!policyLoaded || policyLoading || policySaving" :aria-label="t('agentPolicy.allowAgentComments')" @update:model-value="onAllowChange" />
          <Button type="button" :disabled="!policyLoaded || policyLoading || policySaving || !policyDirty" @click="savePolicy">
            {{ policySaving ? t('agentPolicy.saving') : t('agentPolicy.save') }}
          </Button>
        </div>
      </div>
      <p v-if="policyError" class="mt-3 text-sm text-destructive" role="alert">{{ policyError }}</p>
    </section>

    <section class="mt-4 overflow-hidden rounded-lg border bg-card">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b p-4">
        <div>
          <h2 class="text-sm font-medium">{{ t('agentPolicy.topicsTitle') }}</h2>
          <p class="mt-1 max-w-2xl text-xs text-muted-foreground">{{ t('agentPolicy.topicsHint') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Input v-model="search" :disabled="toggling !== null" class="w-56 max-w-full" :placeholder="t('agentPolicy.searchPlaceholder')" @keyup.enter="applyFilters" />
          <Button variant="outline" type="button" :disabled="toggling !== null" @click="applyFilters">{{ t('agentPolicy.search') }}</Button>
          <Button :variant="onlyDisabled ? 'default' : 'outline'" type="button" :disabled="toggling !== null" @click="toggleOnlyDisabled">
            {{ onlyDisabled ? t('agentPolicy.filterDisabled') : t('agentPolicy.filterAll') }}
          </Button>
        </div>
      </div>

      <p v-if="topicsError" class="p-4 text-sm text-destructive" role="alert">{{ topicsError }}</p>
      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[640px] text-sm">
          <thead class="border-b bg-muted/45 text-xs font-medium text-muted-foreground">
            <tr>
              <th class="h-11 px-4 text-left align-middle">{{ t('agentPolicy.columnTopic') }}</th>
              <th class="h-11 px-4 text-left align-middle">{{ t('agentPolicy.columnAuthor') }}</th>
              <th class="h-11 px-4 text-left align-middle">{{ t('agentPolicy.columnPolicy') }}</th>
              <th class="h-11 px-4 text-right align-middle">{{ t('agentPolicy.columnAction') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-if="topicsLoading">
              <td colspan="4" class="h-24 px-4 text-center text-muted-foreground">{{ t('agentPolicy.loading') }}</td>
            </tr>
            <tr v-else-if="topics.length === 0">
              <td colspan="4" class="h-24 px-4 text-center text-muted-foreground">{{ t('agentPolicy.empty') }}</td>
            </tr>
            <tr v-for="topic in topics" v-else :key="topic.id" class="hover:bg-muted/40">
              <td class="px-4 py-3">
                <div class="font-medium">{{ topic.title }}</div>
                <div class="text-xs text-muted-foreground">#{{ topic.id }}</div>
              </td>
              <td class="px-4 py-3 text-muted-foreground">{{ topic.nickname || topic.username || '—' }}</td>
              <td class="px-4 py-3">
                <Badge :variant="topic.agentCommentDisabled ? 'outline' : 'secondary'">
                  {{ topic.agentCommentDisabled ? t('agentPolicy.policyDisabled') : t('agentPolicy.policyAllowed') }}
                </Badge>
              </td>
              <td class="px-4 py-3 text-right">
                <div class="inline-flex items-center gap-2">
                  <span class="text-xs text-muted-foreground">{{ t('agentPolicy.banLabel') }}</span>
                  <Switch
                    :model-value="topic.agentCommentDisabled"
                    :disabled="toggling !== null"
                    :aria-label="`${t('agentPolicy.banLabel')}: ${topic.title}`"
                    @update:model-value="(value: boolean | 'indeterminate') => toggleTopic(topic, value === true)"
                  />
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="flex items-center justify-end gap-2 border-t p-3 text-xs text-muted-foreground">
        <Button variant="outline" size="sm" type="button" :disabled="page <= 1 || topicsLoading || toggling !== null" @click="changePage(-1)">
          {{ t('agentPolicy.previous') }}
        </Button>
        <span>{{ page }}</span>
        <Button variant="outline" size="sm" type="button" :disabled="!hasNext || topicsLoading || toggling !== null" @click="changePage(1)">
          {{ t('agentPolicy.next') }}
        </Button>
      </div>
    </section>
  </BasicPage>
</template>
