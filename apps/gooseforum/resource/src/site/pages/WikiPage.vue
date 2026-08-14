<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChevronDown, ChevronRight, Clock, Eye, History, Loader2, MessageSquare, X } from '@lucide/vue'
import { getWikiRevisions, updateWikiPage } from '@/runtime/api'
import { formatDateTime, formatNumber } from '@/runtime/format'
import { useFlashMessages } from '@/runtime/flash-message'
import { showUserCard } from '@/runtime/user-card-events'
import MarkdownImageViewer from '@/site/components/MarkdownImageViewer.vue'
import PostStream from '@/site/components/PostStream.vue'
import UserAvatar from '@/site/components/UserAvatar.vue'
import VditorEditor from '@/site/components/VditorEditor.vue'
import WikiPageActions from '@/site/components/WikiPageActions.vue'
import WikiToc from '@/site/components/WikiToc.vue'
import { htmlToMarkdown } from '@/runtime/rich-paste'
import type { LayoutPayload, WikiDetailProps, WikiPageDetailPayload } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'

const page = defineProps<{
  layout: LayoutPayload
  props: WikiDetailProps
}>()

const { t } = useI18n()
const { push: pushFlash } = useFlashMessages()
const markdownImageViewer = ref<InstanceType<typeof MarkdownImageViewer> | null>(null)

// 兼容后端 props 顶层 canEdit/canReview/pending（并行分支实现）与契约 page 内字段两种形状。
type WikiDetailPropsWithTopLevel = WikiDetailProps & {
  canEdit?: boolean
  canReview?: boolean
  pending?: WikiPageDetailPayload['pending']
}
const detailProps = page.props as WikiDetailPropsWithTopLevel

const canEdit = computed(() => detailProps.page.canEdit ?? detailProps.canEdit ?? false)
const canReview = computed(() => detailProps.page.canReview ?? detailProps.canReview ?? false)
const pending = computed(() => detailProps.page.pending ?? detailProps.pending ?? null)

const editing = ref(false)
const mobileTocOpen = ref(false)
const editTitle = ref('')
const editContent = ref('')
const saving = ref(false)
const editError = ref('')

const interactions = ref({
  likeCount: detailProps.page.likeCount,
  isLiked: detailProps.page.liked,
  isBookmarked: detailProps.page.bookmarked,
  isWatched: detailProps.page.watched,
})

const emptyPostStream = {
  posts: [],
  replyTargets: [],
  hasBefore: false,
  hasAfter: false,
  total: 0,
  maxPostNo: 0,
}

async function startEdit() {
  editTitle.value = detailProps.page.title
  editContent.value = ''
  editError.value = ''
  editing.value = true
  // review P1：编辑应加载原始 Markdown（rendered HTML 反解有损）。
  // 公开修订历史接口返回最新 approved 修订的原始 markdown。
  try {
    const revisions = await getWikiRevisions(detailProps.page.id)
    if (revisions.length > 0) {
      editContent.value = revisions[0].content
      return
    }
  } catch {
    // 兜底：渲染 HTML 反解（有损，但至少可编辑）。
  }
  editContent.value = htmlToMarkdown(detailProps.page.content)
}

function cancelEdit() {
  if (saving.value) return
  editing.value = false
  editError.value = ''
}

async function saveEdit() {
  if (saving.value) return

  const title = editTitle.value.trim()
  const content = editContent.value.trim()
  if (!title) {
    editError.value = t('wiki.titleRequired')
    return
  }
  if (!content) {
    editError.value = t('wiki.contentRequired')
    return
  }

  saving.value = true
  editError.value = ''
  try {
    await updateWikiPage(detailProps.page.id, title, content)
    editing.value = false
    pushFlash(t('wiki.editSubmitted'), 'success')
    await refreshCurrentPage()
  } catch (error) {
    editError.value = error instanceof Error ? error.message : t('wiki.editFailed')
  } finally {
    saving.value = false
  }
}

async function refreshCurrentPage() {
  const { fetchPage } = await import('@/runtime/router')
  const payload = await fetchPage(new URL(window.location.href))
  window.dispatchEvent(new CustomEvent('goose:page', { detail: payload }))
}

function handleInteractionChange(state: { likeCount: number; isLiked: boolean; isBookmarked: boolean; isWatched: boolean }) {
  interactions.value = state
}

function handleMarkdownImageClick(event: MouseEvent) {
  const target = event.target
  if (!(target instanceof HTMLElement)) return

  const image = target.closest('.gf-prose-post img')
  if (!(image instanceof HTMLImageElement)) return

  const imageSrc = image.currentSrc || image.src
  if (!imageSrc) return

  const anchor = image.closest('a')
  if (anchor && !sameUrl(anchor.href, imageSrc)) return

  event.preventDefault()
  event.stopPropagation()

  const markdownImages = Array.from(document.querySelectorAll<HTMLImageElement>('.gf-prose-post img'))
    .map((item) => ({
      src: item.currentSrc || item.src,
      alt: item.alt || '',
    }))
    .filter((item) => item.src)
  const index = markdownImages.findIndex((item) => sameUrl(item.src, imageSrc))

  markdownImageViewer.value?.open(markdownImages, index >= 0 ? index : 0)
}

function sameUrl(left: string, right: string) {
  try {
    return new URL(left, window.location.href).href === new URL(right, window.location.href).href
  } catch {
    return left === right
  }
}
</script>

<template>
  <div class="min-w-0">
    <div class="min-w-0" @click="handleMarkdownImageClick">
      <section class="gf-card xl:w-[calc(100%+292px)]">
        <div class="min-w-0 xl:grid xl:grid-cols-[minmax(0,1fr)_256px]">
          <!-- 左列：标题 + 正文 / 编辑态 -->
          <div class="min-w-0">
            <div class="px-4 py-4 sm:px-5 sm:pt-5">
              <template v-if="!editing">
                <div class="flex flex-wrap items-center gap-2">
                  <a href="/wiki" class="text-[13px] font-medium text-base-content/55 hover:text-primary">{{ t('wiki.home') }}</a>
                  <span class="text-base-content/35">/</span>
                  <span class="text-[13px] font-medium text-base-content/75">{{ page.props.page.namespace }}</span>
                </div>
                <h1 class="mt-2 break-words text-2xl font-bold leading-tight text-base-content [overflow-wrap:anywhere] sm:text-3xl">
                  {{ page.props.page.title }}
                </h1>
                <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-[13px] text-base-content/55">
                  <span class="inline-flex items-center gap-1.5">
                    <Clock class="h-3.5 w-3.5" />
                    {{ formatDateTime(page.props.page.updatedAt) }}
                  </span>
                  <a
                    v-if="page.props.page.editorId"
                    :href="`/u/${page.props.page.editorId}`"
                    class="inline-flex items-center gap-1.5 font-medium text-base-content/75 hover:text-primary"
                    @click="showUserCard({ id: page.props.page.editorId, username: page.props.page.editorName, avatarUrl: '' }, $event)"
                  >
                    {{ page.props.page.editorName }}
                  </a>
                  <span class="inline-flex items-center gap-1.5">
                    <Eye class="h-3.5 w-3.5" />
                    {{ formatNumber(page.props.page.viewCount) }}
                  </span>
                  <span class="inline-flex items-center gap-1.5">
                    <MessageSquare class="h-3.5 w-3.5" />
                    {{ formatNumber(page.props.page.postCount) }}
                  </span>
                </div>
              </template>

              <template v-else>
                <div class="flex items-center justify-between gap-3">
                  <h1 class="text-xl font-bold text-base-content sm:text-2xl">{{ t('wiki.editTitle') }}</h1>
                  <button
                    type="button"
                    class="gf-icon-button h-8 w-8 shrink-0 text-base-content/45 transition-colors hover:bg-base-300 hover:text-base-content disabled:cursor-not-allowed disabled:opacity-50"
                    :disabled="saving"
                    :aria-label="t('common.close')"
                    @click="cancelEdit"
                  >
                    <X class="h-4 w-4" />
                  </button>
                </div>
                <div class="mt-4">
                  <input
                    v-model="editTitle"
                    class="gf-input w-full"
                    maxlength="512"
                    :placeholder="t('wiki.titlePlaceholder')"
                  />
                </div>
              </template>
            </div>

            <!-- 移动端（xl 以下）：右栏 aside 隐藏，操作按钮 + 目录折叠展示在此保持可达 -->
            <div class="xl:hidden">
              <div class="border-t border-line/70 px-4 py-4 sm:px-5">
                <WikiPageActions
                  :page="page.props.page"
                  :can-edit="canEdit"
                  @edit="startEdit"
                  @interaction-change="handleInteractionChange"
                />
              </div>
              <div v-if="(page.props.page.toc || []).length" class="border-t border-line/70">
                <button
                  type="button"
                  class="gf-button gf-button-sm gf-button-muted m-4 sm:m-5"
                  :aria-expanded="mobileTocOpen"
                  @click="mobileTocOpen = !mobileTocOpen"
                >
                  <ChevronRight v-if="!mobileTocOpen" class="h-4 w-4" />
                  <ChevronDown v-else class="h-4 w-4" />
                  {{ t('wiki.tocTitle') }}
                </button>
                <div v-if="mobileTocOpen" class="pb-2">
                  <WikiToc :items="page.props.page.toc || []" />
                </div>
              </div>
            </div>

            <!-- pending 横幅：有未审核编辑且当前用户可编辑/可审核时提示 -->
            <div
              v-if="pending && (canEdit || canReview)"
              class="mx-4 mb-4 flex items-start gap-2.5 rounded-[var(--gf-radius-field)] border border-warning/25 bg-warning/10 px-3 py-2.5 sm:mx-5"
              role="status"
            >
              <History class="mt-0.5 h-4 w-4 shrink-0 text-warning" aria-hidden="true" />
              <div class="min-w-0 text-[13px] leading-5 text-base-content/75">
                <span class="font-semibold text-base-content">{{ t('wiki.pendingBanner') }}</span>
                <span class="ml-1">
                  {{ t('wiki.pendingEditor', { editor: pending.editorName || `#${pending.editorId}` }) }}
                </span>
              </div>
            </div>

            <div v-if="!editing" class="border-t border-line/70 px-4 py-4 sm:px-5 sm:py-5">
              <div
                v-if="page.props.page.content"
                v-code-copy
                v-code-highlight
                v-math-render
                class="gf-prose gf-prose-post"
                v-html="page.props.page.content"
              />
              <div v-else class="rounded border border-dashed border-line bg-base-200/60 px-4 py-8 text-center text-sm text-base-content/55">
                {{ t('wiki.contentEmpty') }}
              </div>
            </div>

            <!-- 编辑态：标题 + VditorEditor + 保存/取消 -->
            <div v-else class="border-t border-line/70 px-4 py-4 sm:px-5 sm:py-5">
              <VditorEditor
                v-model="editContent"
                :placeholder="t('wiki.contentPlaceholder')"
                :min-height="420"
                outline
              />
              <p v-if="editError" class="mt-3 text-sm text-error" role="alert">{{ editError }}</p>
              <div class="mt-4 flex items-center justify-end gap-2">
                <button
                  type="button"
                  class="gf-button gf-button-md gf-button-muted"
                  :disabled="saving"
                  @click="cancelEdit"
                >
                  {{ t('common.cancel') }}
                </button>
                <button
                  type="button"
                  class="gf-button gf-button-md gf-button-primary"
                  :disabled="saving"
                  :aria-busy="saving"
                  @click="saveEdit"
                >
                  <Loader2 v-if="saving" class="h-4 w-4 animate-spin" aria-hidden="true" />
                  {{ saving ? t('common.saving') : t('common.save') }}
                </button>
              </div>
            </div>
          </div>

          <!-- 右栏：目录 + 贡献者 + 操作 -->
          <aside class="hidden min-w-0 border-l border-line/70 xl:block">
            <div class="sticky top-19">
              <WikiToc :items="page.props.page.toc || []" />

              <div class="border-t border-line px-4 py-4">
                <WikiPageActions
                  :page="page.props.page"
                  :can-edit="canEdit"
                  @edit="startEdit"
                  @interaction-change="handleInteractionChange"
                />
              </div>

              <div v-if="page.props.contributors.length" class="border-t border-line px-4 py-4">
                <h3 class="text-sm font-semibold text-base-content/55">{{ t('wiki.contributors') }}</h3>
                <ul class="mt-3 space-y-2.5">
                  <li v-for="contributor in page.props.contributors" :key="contributor.userId">
                    <a
                      :href="`/u/${contributor.userId}`"
                      class="flex min-w-0 items-center gap-2.5 rounded-md p-1 transition-colors hover:bg-base-200"
                      @click="showUserCard({ id: contributor.userId, username: contributor.username, avatarUrl: contributor.avatarUrl }, $event)"
                    >
                      <UserAvatar
                        :src="contributor.avatarUrl"
                        :alt="contributor.username"
                        class="h-7 w-7 shrink-0 rounded-full object-cover ring-1 ring-line"
                      />
                      <span class="min-w-0 flex-1 truncate text-[13px] font-medium text-base-content/80">{{ contributor.username }}</span>
                      <span class="shrink-0 text-xs tabular-nums text-base-content/45">{{ contributor.count }}</span>
                    </a>
                  </li>
                </ul>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <!-- 评论流：复用话题评论，按 topicId -->
      <PostStream
        class="mt-4"
        :topic-id="page.props.page.topicId"
        :topic-title="page.props.page.title"
        :initial-post-stream="emptyPostStream"
        :viewer="page.layout.viewer"
        :can-post="page.layout.viewer.isAuthenticated"
        :interactions="interactions"
        :hot-topics="page.props.hotTopics"
        hide-first-post
        :sync-url="false"
        auto-load-first-window
        wide
        @interaction-state="handleInteractionChange"
      />
    </div>

    <MarkdownImageViewer ref="markdownImageViewer" />
  </div>
</template>
