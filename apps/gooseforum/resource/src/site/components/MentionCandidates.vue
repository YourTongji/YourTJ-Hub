<!--
  @mention 候选面板（issue #564 实现，issue #590 自 PostComposer 抽取共享）。
  纯展示 listbox：五态（loading/候选/空/错误/继续输入）、aria、滚动跟随与进场动效；
  会话状态与键盘/搜索编排见 useMentionAutocomplete.ts。三个编辑器（PostComposer /
  PublishPage / QuickPublishModal）统一消费。
  design-taste：浮动工具语言，VARIANCE 4 / MOTION 2。
-->
<script setup lang="ts">
import { computed, ref, watch, type CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MentionUser } from '@/runtime/mention'
import { MENTION_LISTBOX_ID, MENTION_OPTION_ID_PREFIX } from '@/site/composables/useMentionAutocomplete'

const props = defineProps<{
  open: boolean
  query: string
  candidates: MentionUser[]
  activeIndex: number
  loading: boolean
  failed: boolean
  docked: boolean
  panelStyle: CSSProperties
}>()

const emit = defineEmits<{
  select: [user: MentionUser]
}>()

const { t } = useI18n()
const panelRef = ref<HTMLElement | null>(null)

const statusText = computed(() => {
  if (props.loading) return t('mention.loading')
  if (props.failed) return t('mention.searchFailed')
  if (!props.candidates.length) {
    return props.query.trim() ? t('mention.noResults') : t('mention.keepTyping')
  }
  return t('mention.resultCount', { count: props.candidates.length })
})

function optionId(user: MentionUser) {
  return `${MENTION_OPTION_ID_PREFIX}${user.id}`
}

function tagLabel(tag: MentionUser['tag']) {
  if (tag === 'reply-target') return t('mention.replyTarget')
  if (tag === 'topic-author') return t('mention.topicAuthor')
  if (tag === 'participant') return t('mention.participant')
  return ''
}

// active 项滚入视野（键盘导航时列表可能超出可视高度）
watch(
  () => props.activeIndex,
  () => {
    const list = panelRef.value
    const item = list?.querySelector<HTMLElement>('[data-active="true"]')
    if (!list || !item) return
    const itemTop = item.offsetTop
    const itemBottom = itemTop + item.offsetHeight
    if (itemTop < list.scrollTop) list.scrollTop = itemTop
    else if (itemBottom > list.scrollTop + list.clientHeight) list.scrollTop = itemBottom - list.clientHeight
  },
)

function onOptionPointerDown(event: PointerEvent) {
  // 阻止失焦：editor 保持唯一主要 focus（issue #564 a11y）
  event.preventDefault()
}
</script>

<template>
  <!-- 桌面贴近 caret 浮层（空间不足向上翻转/不越界），移动端（≤640px / 200% zoom 窄空间）停靠编辑区内 -->
  <Transition name="mention">
    <div
      v-if="open"
      :id="MENTION_LISTBOX_ID"
      ref="panelRef"
      role="listbox"
      :aria-label="t('mention.listboxLabel')"
      class="gf-mention-panel"
      :class="{ 'is-docked': docked }"
      :style="panelStyle"
    >
      <div v-if="loading && !candidates.length" class="gf-mention-status">{{ t('mention.loading') }}</div>
      <template v-else-if="candidates.length">
        <div
          v-for="(user, index) in candidates"
          :id="optionId(user)"
          :key="user.id"
          role="option"
          :aria-selected="index === activeIndex"
          :aria-label="`${user.nickname || user.username} @${user.username}`"
          class="gf-mention-option"
          :class="{ 'is-active': index === activeIndex }"
          :data-active="index === activeIndex || undefined"
          @pointerdown="onOptionPointerDown"
          @click="emit('select', user)"
        >
          <img :src="user.avatarUrl" alt="" class="gf-mention-avatar" loading="lazy" />
          <span class="min-w-0 flex-1">
            <span class="gf-mention-nickname">{{ user.nickname || user.username }}</span>
            <span class="gf-mention-username">@{{ user.username }}</span>
          </span>
          <span v-if="user.tag" class="gf-mention-tag">{{ tagLabel(user.tag) }}</span>
        </div>
      </template>
      <div v-else class="gf-mention-status" :class="{ 'is-error': failed }">
        {{ failed ? t('mention.searchFailed') : (query.trim() ? t('mention.noResults') : t('mention.keepTyping')) }}
      </div>
    </div>
  </Transition>
  <!-- mention 会话状态（loading/empty/error/result count）polite live region，面板外常驻以稳定播报 -->
  <div aria-live="polite" class="sr-only">{{ open ? statusText : '' }}</div>
</template>

<style>
/*
 * @mention 候选面板样式（issue #564 实现，issue #590 随组件迁移）。
 * 桌面：absolute 贴近 caret 的浮层（352px）；移动/窄空间：is-docked 停靠编辑区底部。
 * 行高 48px 起（touch target ≥44px）；自身滚动不带动页面（overscroll-behavior: contain）。
 */
.gf-mention-panel {
  position: absolute;
  z-index: 30;
  width: 352px;
  max-width: calc(100vw - 2rem);
  max-height: min(416px, 50vh);
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 4px;
  border: 1px solid color-mix(in oklch, var(--gf-color-base-content) 14%, transparent);
  border-radius: 12px;
  background: var(--color-base-100, #fff);
  box-shadow: 0 8px 24px 0 rgba(0, 0, 0, 0.12);
  scrollbar-width: thin;
}

.gf-mention-panel.is-docked {
  position: static;
  width: 100%;
  margin-top: 8px;
  max-height: 234px;
}

.gf-mention-option {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 4px 10px 4px 12px;
  border-radius: 8px;
  cursor: pointer;
  position: relative;
  -webkit-user-select: none;
  user-select: none;
}

/* active 不只靠颜色：左侧 3px 强调条 + aria-selected + 背景 */
.gf-mention-option.is-active {
  background: color-mix(in oklch, var(--gf-color-primary) 12%, transparent);
}

.gf-mention-option.is-active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: 999px;
  background: var(--gf-color-primary);
}

.gf-mention-avatar {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: 999px;
  object-fit: cover;
  background: color-mix(in oklch, var(--gf-color-base-content) 10%, transparent);
}

.gf-mention-nickname,
.gf-mention-username {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gf-mention-nickname {
  font-size: 14px;
  font-weight: 500;
  color: var(--gf-color-base-content);
}

.gf-mention-username {
  font-size: 12px;
  color: color-mix(in oklch, var(--gf-color-base-content) 55%, transparent);
}

.gf-mention-tag {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  line-height: 1.4;
  color: color-mix(in oklch, var(--gf-color-base-content) 60%, transparent);
  background: color-mix(in oklch, var(--gf-color-base-content) 8%, transparent);
}

.gf-mention-status {
  padding: 10px 12px;
  font-size: 13px;
  color: color-mix(in oklch, var(--gf-color-base-content) 55%, transparent);
}

.gf-mention-status.is-error {
  color: var(--gf-color-error, #e5484d);
}

/* 进场：轻微上浮淡入；prefers-reduced-motion 下无动画 */
.mention-enter-active {
  transition:
    opacity 0.15s cubic-bezier(0.2, 0, 0, 1),
    transform 0.15s cubic-bezier(0.2, 0, 0, 1);
}

.mention-enter-from {
  opacity: 0;
  transform: translateY(-4px);
}

@media (prefers-reduced-motion: reduce) {
  .mention-enter-active {
    transition: none;
  }

  .mention-enter-from {
    opacity: 1;
    transform: none;
  }
}
</style>
