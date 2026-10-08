<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { EyeOff, FileText, MessageCircle } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { AnonymousProfileProps, LayoutPayload } from '@gooseforum/client'
import { getIdentityState, type IdentityState } from '@/runtime/anonymous-identity'
import ProfileHeader from '@/site/components/ProfileHeader.vue'
import ProfileManageButton from '@/site/components/ProfileManageButton.vue'
import ProfileStats from '@/site/components/ProfileStats.vue'
import ProfileTabs from '@/site/components/ProfileTabs.vue'
import AnonymousIdentityDialog from '@/site/components/AnonymousIdentityDialog.vue'
import EmptyState from '@/site/components/EmptyState.vue'
import TopicList from '@/site/components/TopicList.vue'

const page = defineProps<{ props: AnonymousProfileProps; layout: LayoutPayload }>()
const { t } = useI18n()
const ownState = ref<IdentityState>()
const manageOpen = ref(false)
const returnFocus = ref<HTMLButtonElement>()
let generation = 0
let managedViewer = 0
let managedUid = ''
onBeforeUnmount(() => { generation++ })
const activeTab = ref('topics')
watch(() => page.props, () => {
  activeTab.value = new URLSearchParams(window.location.search).get('tab') === 'replies' ? 'replies' : 'topics'
}, { immediate: true })
watch(() => [page.layout.viewer.isAuthenticated ? page.layout.viewer.id : 0, page.props.persona.publicUid], async () => {
  const request = ++generation
  ownState.value = undefined
  manageOpen.value = false
  if (!page.layout.viewer.isAuthenticated) return
  try {
    const state = await getIdentityState()
    if (request === generation) ownState.value = state
  } catch { /* Public viewing remains available if the private state cannot load. */ }
}, { immediate: true })
const isOwnProfile = computed(() => ownState.value?.persona?.publicUid === page.props.persona.publicUid)
const showContent = computed(() => page.props.showContent && (!isOwnProfile.value || ownState.value?.showContent !== false))
const persona = computed(() => isOwnProfile.value ? ownState.value!.persona! : page.props.persona)
const tabs = computed(() => [
  { key: 'topics', label: t('user.stats.topics'), url: `${page.props.persona.profileUrl}?tab=topics`, active: activeTab.value === 'topics' },
  { key: 'replies', label: t('user.stats.replies'), url: `${page.props.persona.profileUrl}?tab=replies`, active: activeTab.value === 'replies' },
])
const stats = computed(() => [
  { label: t('user.stats.topics'), value: page.props.topicCount },
  { label: t('user.stats.replies'), value: page.props.replyCount },
])
const hasNext = computed(() => page.props.page * 20 < (activeTab.value === 'replies' ? page.props.replyCount : page.props.topicCount))
const pageUrl = (number: number) => `${page.props.persona.profileUrl}?tab=${activeTab.value}&page=${number}`
function manage(event: MouseEvent) {
  managedViewer = page.layout.viewer.id
  managedUid = page.props.persona.publicUid
  returnFocus.value = event.currentTarget as HTMLButtonElement
  manageOpen.value = true
}
function updateState(state: IdentityState) {
  if (page.layout.viewer.isAuthenticated && page.layout.viewer.id === managedViewer && page.props.persona.publicUid === managedUid) ownState.value = state
}
watch(manageOpen, (open) => {
  if (!open && isOwnProfile.value && ownState.value?.showContent !== page.props.showContent) window.location.reload()
})
</script>

<template>
  <article class="pb-12">
    <section class="gf-card overflow-hidden">
      <ProfileHeader :avatar-url="persona.avatarUrl" :avatar-alt="persona.name">
        <template #identity>
          <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 sm:gap-y-2">
            <h1 class="break-words text-xl font-bold leading-tight tracking-tight text-base-content sm:text-2xl">{{ persona.name }}</h1>
            <span class="gf-badge gf-badge-muted rounded text-[11px]">{{ t('anonymous.identity') }}</span>
          </div>
          <p class="gf-profile-bio mt-2">{{ t('anonymous.historyHint') }}</p>
        </template>
        <template #actions>
          <ProfileManageButton v-if="isOwnProfile" :label="t('anonymous.manageShort')" @click="manage" />
        </template>
      </ProfileHeader>
      <ProfileTabs v-if="showContent" :tabs="tabs" />
      <div v-if="showContent" class="p-4">
        <ProfileStats :items="stats" />
        <div class="pt-4">
          <template v-if="activeTab === 'topics'">
            <TopicList :topics="page.props.topics" />
            <EmptyState v-if="!page.props.topics.length" :icon="FileText" :title="t('user.emptyTopics')" />
          </template>
          <template v-else>
            <div class="divide-y divide-line">
              <a v-for="reply in page.props.replies" :key="reply.id" :href="reply.url"
                class="flex min-w-0 gap-3 py-4 first:pt-0 last:pb-0">
                <MessageCircle class="mt-0.5 h-4 w-4 shrink-0 text-icon-muted" />
                <p class="min-w-0 break-words text-sm leading-6 text-base-content/75">{{ reply.excerpt }}</p>
              </a>
            </div>
            <EmptyState v-if="!page.props.replies.length" :icon="MessageCircle" :title="t('user.emptyData')" />
          </template>
        </div>
      </div>
      <EmptyState v-else :icon="EyeOff" :title="t('anonymous.profileContentHidden')" class="p-8" />
      <nav v-if="showContent && (page.props.page > 1 || hasNext)" class="flex justify-between gap-3 border-t border-line p-4">
        <a v-if="page.props.page > 1" :href="pageUrl(page.props.page - 1)" class="gf-button gf-button-secondary">{{ t('common.previousPage') }}</a>
        <a v-if="hasNext" :href="pageUrl(page.props.page + 1)" class="gf-button gf-button-secondary ml-auto">{{ t('anonymous.next') }}</a>
      </nav>
    </section>
    <AnonymousIdentityDialog v-model:open="manageOpen" :return-focus="returnFocus" @updated="updateState" />
  </article>
</template>
