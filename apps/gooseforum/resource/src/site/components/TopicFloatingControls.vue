<script setup lang="ts">
import { MessageSquare, X } from '@lucide/vue'
import { formatNumber } from '@/runtime/format'
import PostPositionRail from '@/site/components/PostPositionRail.vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  authenticated: boolean
  canPost: boolean
  currentLabel: string
  currentNo: number
  endLabel: string
  hasRail: boolean
  maxNo: number
  mobileRailOpen: boolean
  open: boolean
  progressCurrent?: number
  progressEnd?: number
  progressStart?: number
  railBusy: boolean
  startLabel: string
}>()

const emit = defineEmits<{
  earliest: []
  latest: []
  openReply: []
  selectRail: [postNo: number]
  'update:mobileRailOpen': [value: boolean]
}>()

const { t } = useI18n()
function toggleMobileRail() {
  emit('update:mobileRailOpen', !props.mobileRailOpen)
}

function closeMobileRail() {
  emit('update:mobileRailOpen', false)
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="mobileRailOpen"
      class="pointer-events-auto fixed inset-0 z-[89] xl:hidden"
      @click="closeMobileRail"
    />
    <div class="pointer-events-none fixed inset-x-0 bottom-4 z-[90] px-3 sm:px-6">
      <div class="relative mx-auto flex w-full max-w-full justify-center">
        <Transition name="floating-reply" mode="out-in">
          <div
            v-if="mobileRailOpen"
            class="gf-floating-surface pointer-events-auto relative w-[min(18rem,calc(100vw-2rem))] p-2 xl:hidden"
            @click.stop
          >
            <div class="mb-1 flex items-center justify-between gap-3 px-1">
              <div class="text-xs font-semibold text-base-content/55">{{ t('topic.replyPosition') }}</div>
              <button
                type="button"
                class="inline-flex h-7 w-7 items-center justify-center rounded-md text-icon-muted transition hover:bg-base-300 hover:text-base-content"
                :aria-label="t('common.close')"
                @click="closeMobileRail"
              >
                <X class="h-4 w-4" />
              </button>
            </div>
            <PostPositionRail
              :current="currentNo"
              :max="maxNo"
              :start-label="startLabel"
              :end-label="endLabel"
              :current-label="currentLabel"
              :busy="railBusy"
              :progress-current="progressCurrent"
              :progress-end="progressEnd"
              :progress-start="progressStart"
              @earliest="emit('earliest')"
              @latest="emit('latest')"
              @select="emit('selectRail', $event)"
            />
          </div>
          <div v-else-if="!open" class="pointer-events-auto flex max-w-full flex-col items-center gap-2">
            <div class="gf-floating-surface flex w-fit max-w-full items-center gap-1 rounded-full p-1">
              <button
                v-if="hasRail"
                type="button"
                class="inline-flex h-9 items-center rounded-full px-2.5 text-sm font-black tabular-nums text-primary transition hover:bg-info/10 hover:text-primary xl:hidden"
                :aria-expanded="mobileRailOpen"
                :aria-label="t('topic.replyPosition')"
                @click="toggleMobileRail"
              >
                {{ `${currentNo} / ${formatNumber(maxNo)}` }}
              </button>
              <button
                v-if="!authenticated || canPost"
                type="button"
                data-test="mobile-topic-reply"
                class="inline-flex h-9 items-center gap-1.5 rounded-full px-3 text-sm font-semibold text-base-content/75 transition hover:bg-info/10 hover:text-primary sm:hidden"
                :title="t('topic.joinDiscussion')"
                @click="emit('openReply')"
              >
                <MessageSquare class="h-4 w-4" />
                <span>{{ t('topic.joinDiscussion') }}</span>
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </div>
  </Teleport>
</template>
