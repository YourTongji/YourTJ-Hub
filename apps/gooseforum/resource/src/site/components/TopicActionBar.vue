<script setup lang="ts">
import { computed, ref } from 'vue'
import { Ban, Bell, Bookmark, CornerDownLeft, Flag, Heart, History, MoreHorizontal, RotateCcw, Share2, Trash2 } from '@lucide/vue'
import { PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui'
import type { PostPayload } from '@gooseforum/client'
import type { PostStreamTopicActions } from './PostStream.vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  authenticated: boolean
  canPost: boolean
  isQuestionTopic: boolean
  isProminentReply: boolean
  topicActions: PostStreamTopicActions
  topicProcessStatus: number
  // 首楼在窗口内时为首楼楼层；首楼未加载（深链窗口）时为 null
  replyTarget: PostPayload | null
  isLiked: boolean
  actingLike: boolean
  isBookmarked: boolean
  actingBookmark: boolean
  isWatched: boolean
  actingWatch: boolean
  actingModeration: boolean
  actionMessage: string
  actionMessageSuccess: boolean
}>()

const emit = defineEmits<{
  reply: []
  like: []
  bookmark: []
  watch: []
  share: []
  report: []
  'delete-topic': []
  moderate: [action: 'ban' | 'unban']
  'edit-history': []
}>()

const { t } = useI18n()
const menuOpen = ref(false)

const topicRemoved = computed(() => Boolean(props.topicActions.authorDeleted || props.topicActions.moderatorRemoved))
const canReply = computed(() => (!props.authenticated || props.canPost) && !topicRemoved.value)
const showWatch = computed(() => props.authenticated && !topicRemoved.value)
const showReport = computed(() => props.authenticated && !props.topicActions.isOwnTopic && !topicRemoved.value)
const showOwnTopicActions = computed(() => (props.topicActions.isOwnTopic || props.topicActions.canModerateTopic) && !topicRemoved.value)
const showEditHistory = computed(() => (props.replyTarget?.revisionCount ?? 0) > 1)
// 移动端“更多”弹层无任何可见项时隐藏入口，避免空菜单（如游客视角）
const hasMenuItems = computed(() =>
  showWatch.value || showEditHistory.value || showReport.value
    || (props.topicActions.isOwnTopic && !topicRemoved.value)
    || (props.topicActions.canModerateTopic && !topicRemoved.value),
)
</script>

<template>
  <div data-test="topic-actions">
    <!-- 桌面端操作栏：完整平铺展开，不必收纳入更多菜单（sm 及以上屏幕显示） -->
    <div data-test="desktop-topic-actions" class="hidden sm:flex sm:items-center sm:justify-between sm:gap-2">
      <div class="flex flex-wrap items-center gap-2">
        <!-- 回复 / 回答按钮（“问题”、“文章”、“瞬间”主操作醒目化，去除计数） -->
        <button
          v-if="canReply"
          type="button"
          class="gf-button gf-button-sm rounded-full active:scale-95 transition-all duration-150 flex items-center gap-1.5"
          :class="isProminentReply
            ? 'bg-primary text-primary-content font-medium px-3.5 shadow-sm hover:shadow hover:bg-primary/90'
            : 'px-3 text-base-content/70 hover:bg-base-200 hover:text-base-content'"
          @click="emit('reply')"
        >
          <CornerDownLeft class="h-4 w-4 shrink-0" />
          <span>{{ isQuestionTopic ? t('topic.writeAnswer') : t('topic.reply') }}</span>
        </button>

        <!-- 点赞按钮 -->
        <button
          type="button"
          class="gf-button gf-button-sm rounded-full px-3 active:scale-95 transition-all"
          :class="isLiked ? 'bg-error/10 text-error font-medium hover:bg-error/15' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
          :disabled="actingLike || topicRemoved"
          @click="emit('like')"
        >
          <Heart class="h-4 w-4 shrink-0 transition-transform" :class="{ 'scale-110': isLiked }" :fill="isLiked ? 'currentColor' : 'none'" />
          <span>{{ t('topic.like') }}</span>
        </button>

        <!-- 收藏按钮 -->
        <button
          type="button"
          class="gf-button gf-button-sm rounded-full px-3 active:scale-95 transition-all"
          :class="isBookmarked ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 font-medium hover:bg-amber-500/15' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
          :disabled="actingBookmark || topicRemoved"
          @click="emit('bookmark')"
        >
          <Bookmark class="h-4 w-4 shrink-0 transition-transform" :class="{ 'scale-110': isBookmarked }" :fill="isBookmarked ? 'currentColor' : 'none'" />
          <span>{{ isBookmarked ? t('topic.bookmarked') : t('topic.bookmark') }}</span>
        </button>

        <!-- 关注话题 -->
        <button
          v-if="showWatch"
          type="button"
          class="gf-button gf-button-sm rounded-full px-3 text-base-content/70 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
          :class="{ 'text-success font-medium hover:text-success': isWatched }"
          :disabled="actingWatch"
          @click="emit('watch')"
        >
          <Bell class="h-4 w-4 shrink-0" :fill="isWatched ? 'currentColor' : 'none'" />
          <span>{{ isWatched ? t('topic.watched') : t('topic.watch') }}</span>
        </button>

        <!-- 查看编辑历史 -->
        <button
          v-if="showEditHistory"
          type="button"
          class="gf-button gf-button-sm rounded-full px-3 text-base-content/70 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
          @click="emit('edit-history')"
        >
          <History class="h-4 w-4 shrink-0" />
          <span>{{ t('topic.editHistory') }}</span>
        </button>
      </div>

      <div class="flex items-center gap-1.5">
        <!-- 分享按钮 -->
        <button
          type="button"
          class="gf-icon-button h-8 w-8 rounded-full text-base-content/60 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
          :title="t('topic.share')"
          @click="emit('share')"
        >
          <Share2 class="h-4 w-4" />
          <span class="sr-only">{{ t('topic.share') }}</span>
        </button>

        <!-- 举报话题 -->
        <button
          v-if="showReport"
          type="button"
          class="gf-button gf-button-sm rounded-full px-2.5 text-base-content/70 hover:bg-warning/10 hover:text-warning active:scale-95 transition-all"
          @click="emit('report')"
        >
          <Flag class="h-3.5 w-3.5 shrink-0" />
          <span>{{ t('topic.report') }}</span>
        </button>

        <!-- 删除话题（作者） -->
        <button
          v-if="topicActions.isOwnTopic && !topicRemoved"
          type="button"
          class="gf-button gf-button-sm rounded-full px-2.5 text-error hover:bg-error/10 active:scale-95 transition-all"
          @click="emit('delete-topic')"
        >
          <Trash2 class="h-3.5 w-3.5 shrink-0" />
          <span>{{ t('topic.deleteTopic') }}</span>
        </button>

        <!-- 封禁/解封（版主） -->
        <button
          v-if="topicActions.canModerateTopic && topicProcessStatus === 0"
          type="button"
          class="gf-button gf-button-sm rounded-full px-2.5 text-warning hover:bg-warning/10 active:scale-95 transition-all"
          :disabled="actingModeration"
          @click="emit('moderate', 'ban')"
        >
          <Ban class="h-3.5 w-3.5 shrink-0" />
          <span>{{ t('topic.moderationBan') }}</span>
        </button>
        <button
          v-else-if="topicActions.canModerateTopic && topicProcessStatus === 1"
          type="button"
          class="gf-button gf-button-sm rounded-full px-2.5 text-primary hover:bg-info/10 active:scale-95 transition-all"
          :disabled="actingModeration"
          @click="emit('moderate', 'unban')"
        >
          <RotateCcw class="h-3.5 w-3.5 shrink-0" />
          <span>{{ t('topic.moderationUnban') }}</span>
        </button>
      </div>
    </div>

    <!-- 移动端操作栏：单行高频互动 + 优雅的 Popover 收纳菜单（<sm 屏幕显示） -->
    <div data-test="mobile-topic-actions" class="flex sm:hidden items-center justify-between gap-1">
      <!-- 移动端主题互动；回复由悬浮入口提供 -->
      <div class="flex items-center gap-1.5">
        <!-- 点赞按钮 -->
        <button
          type="button"
          class="gf-button gf-button-sm rounded-full px-2.5 active:scale-95 transition-all"
          :class="isLiked ? 'bg-error/10 text-error font-medium hover:bg-error/15' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
          :disabled="actingLike || topicRemoved"
          @click="emit('like')"
        >
          <Heart class="h-3.5 w-3.5 shrink-0 transition-transform" :class="{ 'scale-110': isLiked }" :fill="isLiked ? 'currentColor' : 'none'" />
          <span>{{ t('topic.like') }}</span>
        </button>

        <!-- 收藏按钮 -->
        <button
          type="button"
          class="gf-button gf-button-sm rounded-full px-2.5 active:scale-95 transition-all"
          :class="isBookmarked ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 font-medium hover:bg-amber-500/15' : 'text-base-content/70 hover:bg-base-200 hover:text-base-content'"
          :disabled="actingBookmark || topicRemoved"
          @click="emit('bookmark')"
        >
          <Bookmark class="h-3.5 w-3.5 shrink-0 transition-transform" :class="{ 'scale-110': isBookmarked }" :fill="isBookmarked ? 'currentColor' : 'none'" />
        </button>
      </div>

      <!-- 右侧分享与更多收纳 -->
      <div class="flex items-center gap-1">
        <!-- 分享按钮 -->
        <button
          type="button"
          class="gf-icon-button h-8 w-8 rounded-full text-base-content/60 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
          :title="t('topic.share')"
          @click="emit('share')"
        >
          <Share2 class="h-4 w-4" />
          <span class="sr-only">{{ t('topic.share') }}</span>
        </button>

        <!-- 更多操作 Popover（关注、编辑历史、举报、删除、封禁） -->
        <PopoverRoot v-if="hasMenuItems" v-model:open="menuOpen">
          <PopoverTrigger as-child>
            <button
              type="button"
              class="gf-icon-button h-8 w-8 rounded-full text-base-content/60 hover:bg-base-200 hover:text-base-content active:scale-95 transition-all"
              :title="t('topic.more')"
            >
              <MoreHorizontal class="h-4 w-4" />
              <span class="sr-only">{{ t('topic.more') }}</span>
            </button>
          </PopoverTrigger>
          <PopoverPortal>
            <PopoverContent
              side="top"
              align="end"
              :side-offset="8"
              :collision-padding="16"
              class="z-[70] min-w-[160px] max-w-[200px] rounded-2xl border border-line/60 bg-base-100/98 p-1.5 shadow-[0_12px_36px_-6px_rgba(0,0,0,0.14),0_4px_12px_-2px_rgba(0,0,0,0.06)] backdrop-blur-md outline-none animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 duration-150"
            >
              <div class="flex flex-col gap-0.5" role="menu">
                <!-- 关注话题 -->
                <button
                  v-if="showWatch"
                  type="button"
                  role="menuitem"
                  class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-base-content/85 transition-colors hover:bg-base-200/80 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 cursor-pointer"
                  :class="{ 'text-success hover:text-success': isWatched }"
                  :disabled="actingWatch"
                  @click="emit('watch'); menuOpen = false"
                >
                  <Bell class="h-4 w-4 shrink-0" :fill="isWatched ? 'currentColor' : 'none'" />
                  <span>{{ isWatched ? t('topic.watched') : t('topic.watch') }}</span>
                </button>

                <!-- 查看编辑历史 -->
                <button
                  v-if="showEditHistory"
                  type="button"
                  role="menuitem"
                  class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-base-content/85 transition-colors hover:bg-base-200/80 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 cursor-pointer"
                  @click="emit('edit-history'); menuOpen = false"
                >
                  <History class="h-4 w-4 shrink-0" />
                  <span>{{ t('topic.editHistory') }}</span>
                </button>

                <!-- 举报话题 -->
                <button
                  v-if="showReport"
                  type="button"
                  role="menuitem"
                  class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-base-content/85 transition-colors hover:bg-warning/10 hover:text-warning active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warning/60 cursor-pointer"
                  @click="emit('report'); menuOpen = false"
                >
                  <Flag class="h-4 w-4 shrink-0" />
                  <span>{{ t('topic.report') }}</span>
                </button>

                <div v-if="showOwnTopicActions" class="my-1 border-t border-line/60" />

                <!-- 删除话题（作者） -->
                <button
                  v-if="topicActions.isOwnTopic && !topicRemoved"
                  type="button"
                  role="menuitem"
                  class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-error transition-colors hover:bg-error/10 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error/60 cursor-pointer"
                  @click="emit('delete-topic'); menuOpen = false"
                >
                  <Trash2 class="h-4 w-4 shrink-0" />
                  <span>{{ t('topic.deleteTopic') }}</span>
                </button>

                <!-- 封禁/解封（版主） -->
                <button
                  v-if="topicActions.canModerateTopic && topicProcessStatus === 0"
                  type="button"
                  role="menuitem"
                  class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-warning transition-colors hover:bg-warning/10 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warning/60 cursor-pointer"
                  :disabled="actingModeration"
                  @click="emit('moderate', 'ban'); menuOpen = false"
                >
                  <Ban class="h-4 w-4 shrink-0" />
                  <span>{{ t('topic.moderationBan') }}</span>
                </button>
                <button
                  v-else-if="topicActions.canModerateTopic && topicProcessStatus === 1"
                  type="button"
                  role="menuitem"
                  class="flex w-full items-center gap-2.5 rounded-xl px-2.5 py-2 text-left text-xs font-medium text-primary transition-colors hover:bg-info/10 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 cursor-pointer"
                  :disabled="actingModeration"
                  @click="emit('moderate', 'unban'); menuOpen = false"
                >
                  <RotateCcw class="h-4 w-4 shrink-0" />
                  <span>{{ t('topic.moderationUnban') }}</span>
                </button>
              </div>
            </PopoverContent>
          </PopoverPortal>
        </PopoverRoot>
      </div>
    </div>
    <span v-if="actionMessage" class="mt-2 block text-xs" :class="actionMessageSuccess ? 'text-base-content/75' : 'text-error'">{{ actionMessage }}</span>
  </div>
</template>
