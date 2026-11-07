import { onScopeDispose, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { WikiPageDetailPayload } from '@gooseforum/client'
import { bookmarkTopic, likeTopic, watchTopic } from '@/runtime/api'
import { useFlashMessages } from '@/runtime/flash-message'

export interface WikiInteractionState {
  likeCount: number
  isLiked: boolean
  isBookmarked: boolean
  isWatched: boolean
}
export type WikiInteraction = 'like' | 'bookmark' | 'watch'
export type WikiInteractionPending = Record<WikiInteraction, boolean>

function initialState(page: WikiPageDetailPayload): WikiInteractionState {
  return { likeCount: page.likeCount, isLiked: page.liked, isBookmarked: page.bookmarked, isWatched: page.watched }
}

// Both responsive bars stay mounted. Own state and in-flight guards once per page,
// so resizing cannot start a second request or lose an optimistic update/rollback.
export function useWikiPageInteractions(getPage: () => WikiPageDetailPayload) {
  const { t } = useI18n()
  const { push: pushFlash } = useFlashMessages()
  const interactions = ref(initialState(getPage()))
  const pending = reactive<WikiInteractionPending>({ like: false, bookmark: false, watch: false })
  let generation = 0
  watch(getPage, (page) => {
    generation++
    interactions.value = initialState(page)
    pending.like = pending.bookmark = pending.watch = false
  }, { flush: 'sync' })
  onScopeDispose(() => { generation++ })

  const actions = {
    like: { field: 'isLiked', request: likeTopic, failed: 'api.likeFailed' },
    bookmark: { field: 'isBookmarked', request: bookmarkTopic, failed: 'api.bookmarkFailed' },
    watch: { field: 'isWatched', request: watchTopic, failed: 'api.watchFailed' },
  } as const

  async function toggle(action: WikiInteraction) {
    if (pending[action]) return
    const { field, request, failed } = actions[action]
    const requestGeneration = generation
    const topicId = getPage().topicId
    const previous = interactions.value[field]
    const previousCount = interactions.value.likeCount
    const next = !previous
    pending[action] = true
    interactions.value[field] = next
    if (action === 'like') {
      interactions.value.likeCount = Math.max(0, previousCount + (next ? 1 : -1))
    }
    try {
      await request(topicId, next ? 1 : 2)
      if (generation !== requestGeneration) return
      if (action === 'bookmark') pushFlash(next ? t('topic.bookmarkAdded') : t('topic.bookmarkRemoved'))
      if (action === 'watch') pushFlash(next ? t('topic.watchAdded') : t('topic.watchRemoved'))
    } catch (error) {
      if (generation !== requestGeneration) return
      // Roll back only this action; another action may have completed meanwhile.
      interactions.value[field] = previous
      if (action === 'like') interactions.value.likeCount = previousCount
      pushFlash(error instanceof Error ? error.message : t(failed), 'error')
    } finally {
      if (generation === requestGeneration) pending[action] = false
    }
  }

  return { interactions, pending, toggle }
}
