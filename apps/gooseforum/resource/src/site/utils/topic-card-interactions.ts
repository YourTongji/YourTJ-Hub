import { reactive } from 'vue'
import type { TopicPayload } from '@gooseforum/client'
import { bookmarkTopic, likeTopic } from '@/runtime/api'

// Owned by the list, so recycling a card never releases its pending action.
// Each field follows fresh payloads independently; a viewer change invalidates
// the owner before any response can write into the next viewer's list.
export function createTopicCardInteraction(initial: TopicPayload, t: (key: string) => string) {
  let source = initial
  let active = true
  let generation = 0
  const state = reactive({
    liked: initial.liked,
    bookmarked: initial.bookmarked,
    likeCount: initial.likeCount,
    actingLike: false,
    actingBookmark: false,
    error: '',
  })

  function sync(next: TopicPayload) {
    if (next.id !== source.id || (next.liked === undefined && source.liked !== undefined)) {
      generation++
      state.actingLike = false
      state.actingBookmark = false
      state.error = ''
    }
    source = next
    if (!state.actingLike) {
      state.liked = next.liked
      state.likeCount = next.likeCount
    }
    if (!state.actingBookmark) state.bookmarked = next.bookmarked
  }

  async function toggle(bookmark: boolean) {
    const busy = bookmark ? 'actingBookmark' : 'actingLike'
    const field = bookmark ? 'bookmarked' : 'liked'
    if (!active || state[busy]) return
    if (state[field] === undefined) {
      const current = `${window.location.pathname}${window.location.search}${window.location.hash}`
      window.location.href = `/login?redirect=${encodeURIComponent(current)}`
      return
    }
    const requestGeneration = generation
    const target = !state[field]
    state[busy] = true
    state[field] = target
    state.error = ''
    if (!bookmark) state.likeCount = Math.max(0, state.likeCount + (target ? 1 : -1))
    try {
      const success = await (bookmark ? bookmarkTopic : likeTopic)(source.id, target ? 1 : 2)
      if (!active || requestGeneration !== generation) return
      if (!success) throw new Error(t(bookmark ? 'api.bookmarkFailed' : 'api.likeFailed'))
      source[field] = target
      if (!bookmark) source.likeCount = state.likeCount
    } catch (error) {
      if (!active || requestGeneration !== generation) return
      // source is the latest authoritative payload, including a refresh that
      // completed while this action was pending. Never roll back the other field.
      state[field] = source[field]
      if (!bookmark) state.likeCount = source.likeCount
      state.error = error instanceof Error ? error.message : t(bookmark ? 'api.bookmarkFailed' : 'api.likeFailed')
    } finally {
      if (active && requestGeneration === generation) state[busy] = false
    }
  }

  return { state, sync, toggle, dispose: () => { active = false } }
}

export type TopicCardInteraction = ReturnType<typeof createTopicCardInteraction>
