<script setup lang="ts">
import { computed } from 'vue'
import { Lock } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { ProfileSectionVisibility } from '@gooseforum/client'
import EmptyState from '@/site/components/EmptyState.vue'

// One profile section's privacy state (issue #1124). Viewers who cannot see
// the section get a "not public" state instead of an empty list; the owner
// gets a reminder of who else can see it.
const props = defineProps<{
  state?: ProfileSectionVisibility
  isOwnProfile: boolean
  authenticated: boolean
  name: string
  settingsUrl: string
  compact?: boolean
}>()
const { t } = useI18n()

const hidden = computed(() => props.state ? !props.state.visible : false)
const ownerNote = computed(() => {
  if (!props.isOwnProfile) return ''
  switch (props.state?.audience) {
    case 'members': return t('user.privacy.ownerMembers')
    case 'followers': return t('user.privacy.ownerFollowers')
    case 'mutual': return t('user.privacy.ownerMutual')
    case 'self': return t('user.privacy.ownerSelf')
    default: return ''
  }
})
const reason = computed(() => {
  const name = props.name
  switch (props.state?.reason) {
    case 'signin': return t('user.privacy.reasonSignin', { name })
    case 'followers': return t(props.authenticated ? 'user.privacy.reasonFollowers' : 'user.privacy.reasonFollowersGuest', { name })
    case 'mutual': return t('user.privacy.reasonMutual', { name })
    default: return t('user.privacy.reasonPrivate', { name })
  }
})
// Only offer sign-in when signing in could actually reveal the section.
const loginHref = computed(() => {
  if (props.authenticated || (props.state?.reason !== 'signin' && props.state?.reason !== 'followers')) return ''
  const current = typeof window === 'undefined' ? '' : `${window.location.pathname}${window.location.search}`
  return current ? `/login?redirect=${encodeURIComponent(current)}` : '/login'
})
const manageHref = computed(() => `${props.settingsUrl}?tab=privacy`)
</script>

<template>
  <div v-if="hidden" data-test="profile-section-hidden" :data-reason="state?.reason">
    <p v-if="compact" class="flex items-start gap-2 py-3 text-sm text-base-content/60">
      <Lock class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <span>{{ reason }}</span>
    </p>
    <EmptyState v-else :icon="Lock" :title="t('user.privacy.hiddenTitle')" :description="reason">
      <a v-if="loginHref" :href="loginHref" class="gf-button gf-button-md gf-button-secondary">{{ t('user.privacy.signIn') }}</a>
    </EmptyState>
  </div>
  <p
    v-else-if="ownerNote"
    class="flex flex-wrap items-center gap-x-2 gap-y-1 rounded-md bg-base-200/70 px-3 py-2 text-xs text-base-content/70"
    data-test="profile-section-owner-note"
  >
    <Lock class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
    <span class="min-w-0 flex-1">{{ ownerNote }}</span>
    <a :href="manageHref" class="font-medium text-primary hover:underline hover:underline-offset-2">{{ t('user.privacy.manage') }}</a>
  </p>
</template>
