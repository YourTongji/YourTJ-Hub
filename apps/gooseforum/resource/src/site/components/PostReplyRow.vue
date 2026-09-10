<script setup lang="ts">
import { computed } from 'vue'
import { Ban, Bookmark, ChevronDown, ChevronUp, CornerDownLeft, Flag, Heart, PencilLine, RotateCcw, Share2, Trash2 } from '@lucide/vue'
import type { PostPayload } from '@gooseforum/client'
import { formatDateTime, formatNumber } from '@/runtime/format'
import { showUserCard } from '@/runtime/user-card-events'
import { useI18n } from 'vue-i18n'
import UserAvatar from '@/site/components/UserAvatar.vue'
import { buildBeamAvatarDataUri } from '@/site/utils/course-review-share'

/** 楼层互动状态（由父组件 postActionState 提供，保持同一对象引用以继承响应式更新）。 */
export interface PostReplyActionState {
  likeCount: number
  isLiked: boolean
  isBookmarked: boolean
  actingLike: boolean
  actingBookmark: boolean
}

const props = withDefaults(defineProps<{
  post: PostPayload
  /** 深链高亮态。 */
  highlighted?: boolean
  authenticated: boolean
  canPost: boolean
  /** 编辑保存中 / 删除中 / 版主操作进行中（按钮禁用态，源自父组件全局状态）。 */
  savingEdit?: boolean
  deleting?: boolean
  moderationBusy?: boolean
  actionState: PostReplyActionState
  /** 树状视图缩进层级（0 = 不缩进；0/undefined 时与扁平堆叠像素等价）。上限在调用侧收口。 */
  indentLevel?: number
  /** 树状视图：是否有子回复（显示折叠按钮）。 */
  collapsible?: boolean
  /** 树状视图：当前是否处于折叠态。 */
  collapsed?: boolean
}>(), {
  highlighted: false,
  savingEdit: false,
  deleting: false,
  moderationBusy: false,
  indentLevel: 0,
  collapsible: false,
  collapsed: false,
})

// 业务动作全部由父组件承接（replyTo/startEditPost/togglePostLike 等），
// 本组件只负责渲染与透传，不复制任何请求/状态逻辑。
const emit = defineEmits<{
  reply: []
  edit: []
  delete: []
  like: []
  bookmark: []
  share: []
  report: []
  moderate: [action: 'ban' | 'unban']
  toggleCollapse: []
}>()

const { t } = useI18n()

// 展示型判定，与主流楼层同一规则；回复行不可能是首楼。
const isRemoved = computed(() => props.post.isAuthorDeleted || props.post.isModeratorRemoved)
const canEdit = computed(() => props.post.isOwnPost && !props.post.isHidden && !isRemoved.value)
const canDelete = computed(() => props.post.isOwnPost && !props.post.isHidden && !isRemoved.value)
const canReply = computed(() => (!props.authenticated || props.canPost) && !props.post.isHidden && !isRemoved.value)
const canLike = computed(() => props.authenticated && !props.post.isHidden && !isRemoved.value)
const canReport = computed(() => !props.post.isOwnPost && !props.post.isHidden && !isRemoved.value)

function authorDisplayName(author: { username: string; nickname?: string }) {
  return author.nickname || author.username
}

// 匿名楼层占位头像：复用课程评价的 boring-avatars beam 占位（seed 用楼层 id，跨语言稳定）
function anonymousAvatarSrc(post: PostPayload, size = 24): string {
  return buildBeamAvatarDataUri(`anonymous-${post.id}`, size)
}

function lastEditedLabel(post: PostPayload) {
  if (!post.lastEditedAt || !post.lastEditor) return ''
  return t('topic.lastEditedBy', {
    time: formatDateTime(post.lastEditedAt),
    user: authorDisplayName(post.lastEditor),
  })
}
</script>

<template>
  <div class="scroll-mt-20 reply-row-indent" :class="{ 'reply-row-indented': indentLevel > 0 }" :style="{ '--reply-row-indent': indentLevel }">
    <div
      class="rounded-lg bg-base-200/45 p-3 transition-[background-color]"
      :class="{ 'bg-info/10': highlighted }"
    >
      <div class="mb-2 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
        <a v-if="!post.isAnonymous" :href="`/u/${post.author.id}`" class="shrink-0" @click="showUserCard(post.author, $event)">
          <UserAvatar :src="post.author.avatarUrl" :alt="post.author.username" :badge="post.author.wornBadge" class="h-6 w-6 rounded-full ring-1 ring-line" img-class="rounded-full" />
        </a>
        <span v-else class="shrink-0">
          <UserAvatar :src="anonymousAvatarSrc(post)" :alt="t('topic.authorAnonymous')" class="h-6 w-6 rounded-full ring-1 ring-line" img-class="rounded-full" />
        </span>
        <a v-if="!post.isAnonymous" :href="`/u/${post.author.id}`" class="min-w-0 truncate text-sm font-semibold text-base-content hover:text-primary" @click="showUserCard(post.author, $event)">{{ authorDisplayName(post.author) }}</a>
        <span v-else class="min-w-0 truncate text-sm font-semibold text-base-content/55">{{ t('topic.authorAnonymous') }}</span>
        <span class="shrink-0 text-xs font-semibold tabular-nums text-base-content/55">#{{ formatNumber(post.postNo) }}</span>
        <time class="hidden shrink-0 text-xs text-base-content/55 sm:inline">{{ formatDateTime(post.createdAt) }}</time>
          <button v-if="collapsible" type="button" class="inline-flex h-5 shrink-0 items-center gap-0.5 rounded-full bg-base-100 px-1.5 text-[11px] font-medium text-base-content/55 transition hover:text-base-content" :title="collapsed ? t('topic.expandReply') : t('topic.collapseReply')" @click="emit('toggleCollapse')">
            <ChevronDown v-if="collapsed" class="h-3 w-3" />
            <ChevronUp v-else class="h-3 w-3" />
            <span class="sr-only">{{ collapsed ? t('topic.expandReply') : t('topic.collapseReply') }}</span>
          </button>
        <span class="ml-auto flex shrink-0 items-center gap-1">
          <button v-if="canEdit" type="button" class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-icon-muted transition hover:bg-info/10 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50" :disabled="savingEdit || deleting" :title="t('common.edit')" @click="emit('edit')">
            <PencilLine class="h-3.5 w-3.5" />
            <span class="sr-only">{{ t('common.edit') }}</span>
          </button>
          <button v-if="canDelete" type="button" class="gf-icon-button h-7 w-7 shrink-0 hover:bg-error/10 hover:text-error disabled:cursor-not-allowed disabled:opacity-50" :disabled="deleting" :title="deleting ? t('topic.deleting') : t('topic.delete')" @click="emit('delete')">
            <Trash2 class="h-3.5 w-3.5" />
            <span class="sr-only">{{ deleting ? t('topic.deleting') : t('topic.delete') }}</span>
          </button>
          <button v-if="canReply" type="button" class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-icon-muted transition hover:bg-info/10 hover:text-primary" :title="t('topic.reply')" @click="emit('reply')">
            <CornerDownLeft class="h-3.5 w-3.5" />
            <span class="sr-only">{{ t('topic.reply') }}</span>
          </button>
          <button v-if="canLike" type="button" class="inline-flex h-7 shrink-0 items-center gap-1 rounded-md px-1 text-icon-muted transition hover:bg-error/10 hover:text-error disabled:cursor-not-allowed disabled:opacity-50" :class="{ 'text-error hover:text-error': actionState.isLiked }" :title="t('topic.like')" :disabled="actionState.actingLike" @click="emit('like')">
            <Heart class="h-3.5 w-3.5" :fill="actionState.isLiked ? 'currentColor' : 'none'" />
            <span v-if="actionState.likeCount" class="hidden text-xs font-semibold tabular-nums sm:inline">{{ formatNumber(actionState.likeCount) }}</span>
            <span class="sr-only">{{ t('topic.like') }}</span>
          </button>
          <button v-if="canLike" type="button" class="gf-icon-button h-7 w-7 shrink-0 hover:bg-info/10 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50" :class="{ 'text-primary hover:text-primary': actionState.isBookmarked }" :title="actionState.isBookmarked ? t('topic.bookmarked') : t('topic.bookmark')" :disabled="actionState.actingBookmark" @click="emit('bookmark')">
            <Bookmark class="h-3.5 w-3.5" :fill="actionState.isBookmarked ? 'currentColor' : 'none'" />
            <span class="sr-only">{{ t('topic.bookmark') }}</span>
          </button>
          <button type="button" class="gf-icon-button h-7 w-7 shrink-0 hover:bg-base-200 hover:text-base-content" :title="t('topic.share')" @click="emit('share')">
            <Share2 class="h-3.5 w-3.5" />
            <span class="sr-only">{{ t('topic.share') }}</span>
          </button>
          <button v-if="canReport" type="button" class="gf-icon-button h-7 w-7 shrink-0 hover:bg-warning/10 hover:text-warning" :title="t('topic.report')" @click="emit('report')">
            <Flag class="h-3.5 w-3.5" />
            <span class="sr-only">{{ t('topic.report') }}</span>
          </button>
          <button v-if="post.canModerate && post.processStatus === 0" type="button" class="gf-icon-button h-7 w-7 shrink-0 hover:bg-error/10 hover:text-error disabled:opacity-50" :disabled="moderationBusy" :title="t('topic.moderationBan')" @click="emit('moderate', 'ban')">
            <Ban class="h-3.5 w-3.5" />
            <span class="sr-only">{{ t('topic.moderationBan') }}</span>
          </button>
          <button v-else-if="post.canModerate && post.processStatus === 1" type="button" class="gf-icon-button h-7 w-7 shrink-0 hover:bg-info/10 hover:text-primary disabled:opacity-50" :disabled="moderationBusy" :title="t('topic.moderationUnban')" @click="emit('moderate', 'unban')">
            <RotateCcw class="h-3.5 w-3.5" />
            <span class="sr-only">{{ t('topic.moderationUnban') }}</span>
          </button>
        </span>
      </div>
      <div v-if="post.isAuthorDeleted" class="mt-2 rounded border border-dashed border-line bg-base-100/60 px-3 py-2 text-sm text-base-content/55">
        {{ t('topic.authorDeletedPlaceholder') }}
      </div>
      <div v-else-if="post.isModeratorRemoved" class="mt-2 rounded border border-dashed border-line bg-base-100/60 px-3 py-2 text-sm text-base-content/55">
        {{ t('topic.moderatorRemovedPlaceholder') }}
      </div>
      <div v-else-if="post.isHidden && !post.canModerate" class="mt-2 rounded border border-line bg-base-100/60 px-3 py-2 text-sm text-base-content/45">
        {{ t('topic.hiddenReplyPlaceholder') }}
      </div>
      <div v-else v-code-copy v-code-highlight v-math-render v-content-enhancements class="gf-prose gf-prose-post mt-1" v-html="post.renderedContent" />
      <div v-if="!post.lastEditedAt && post.updatedAt && post.updatedAt !== post.createdAt" class="mt-2 text-xs font-medium text-base-content/55">
        {{ t('topic.editedAt', { time: formatDateTime(post.updatedAt) }) }}
      </div>
      <div v-if="post.lastEditedAt && post.lastEditor" class="mt-2 text-xs font-medium text-base-content/55">
        {{ lastEditedLabel(post) }}
      </div>
    </div>
  </div>
</template>

<style scoped>
/*
 * 树状视图缩进（#520 增强）：每级 16px（移动）/ 24px（桌面），0 级时不产生任何偏移。
 * 相比旧值（12/20px）逐级放大，多级嵌套的层级差异一眼可辨。
 */
.reply-row-indent {
  margin-left: calc(var(--reply-row-indent, 0) * 16px);
}

@media (min-width: 640px) {
  .reply-row-indent {
    margin-left: calc(var(--reply-row-indent, 0) * 24px);
  }
}

/*
 * 逐级引导线（#520）：缩进行在左侧留白内画一条贯穿线，与上一级内容对齐成「树杈」，
 * 深层嵌套不再靠纯缩进脑补。缩进封顶后（indentLevel 停增）引导线仍随层级存在。
 */
.reply-row-indented {
  position: relative;
}

.reply-row-indented::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: -10px;
  width: 2px;
  border-radius: 999px;
  background: color-mix(in oklch, var(--gf-color-line) 85%, transparent);
}

@media (min-width: 640px) {
  .reply-row-indented::before {
    left: -14px;
  }
}
</style>
