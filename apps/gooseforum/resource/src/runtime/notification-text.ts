import type { NotificationPayload } from '@gooseforum/client'

// System notification callers must pass only public presentation fields.
export function notificationText(item: Pick<NotificationPayload, 'eventType' | 'payload'> & Partial<NotificationPayload>, t: (key: string, params?: Record<string, string>) => string): string {
  const actor = item.actor?.username || item.payload.actorName || item.payload.metadata?.followerName || t('notifications.actorFallback')
  const topic = item.topic?.title || item.payload.topicTitle || ''
  const event = item.payload.hasDirectMention ? 'mention' : item.eventType
  const key: Record<string, string> = { mention: 'mention', post_reply: 'reply', comment: 'comment', topic_post: 'topicPost', like: 'like', follow: 'follow', wiki_updated: 'wiki', badge: 'badge', review_pending: 'reviewPending', review_approved: 'reviewApproved', review_rejected: 'reviewRejected' }
  if (!key[event]) {
    const title = (item.title || item.payload.title || '').trim()
    return title && !title.startsWith('notifications.') ? title : t('notifications.fallback')
  }
  if (event.startsWith('review_')) return t(`notifications.templates.${key[event]}`)
  if (event === 'badge') {
    const badge = item.payload.metadata?.badgeName || (item.payload.templateParams as {badgeName?: string} | undefined)?.badgeName
    return badge ? t('notifications.presentation.badge', { badge }) : t('notifications.badgeGeneric')
  }
  const withTopic = topic && ['mention', 'comment', 'topic_post', 'wiki_updated'].includes(event)
  return t(`notifications.presentation.${key[event]}${withTopic ? 'Topic' : ''}`, { actor, topic })
}
