export type TopicPreviewCloser = () => void

interface ActivePreview {
  owner: TopicPreviewCloser
  closeImmediately: TopicPreviewCloser
}

let activePreview: ActivePreview | undefined
const previewClosers = new Set<TopicPreviewCloser>()

function closeAllPreviews() {
  for (const close of previewClosers) close()
}

function closeHiddenPagePreviews() {
  if (document.hidden) closeAllPreviews()
}

export function registerTopicPreview(close: TopicPreviewCloser) {
  if (previewClosers.size === 0) {
    window.addEventListener('blur', closeAllPreviews)
    window.addEventListener('goose:page', closeAllPreviews)
    document.addEventListener('visibilitychange', closeHiddenPagePreviews)
  }
  previewClosers.add(close)
}

export function unregisterTopicPreview(close: TopicPreviewCloser) {
  previewClosers.delete(close)
  if (activePreview?.owner === close) activePreview = undefined
  if (previewClosers.size === 0) {
    window.removeEventListener('blur', closeAllPreviews)
    window.removeEventListener('goose:page', closeAllPreviews)
    document.removeEventListener('visibilitychange', closeHiddenPagePreviews)
  }
}

export function activateTopicPreview(owner: TopicPreviewCloser, closeImmediately: TopicPreviewCloser) {
  if (activePreview?.owner !== owner) activePreview?.closeImmediately()
  activePreview = { owner, closeImmediately }
}

export function deactivateTopicPreview(owner: TopicPreviewCloser) {
  if (activePreview?.owner === owner) activePreview = undefined
}
