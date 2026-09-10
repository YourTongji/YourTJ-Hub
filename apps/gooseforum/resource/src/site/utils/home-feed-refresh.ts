import type { TopicPayload } from '@gooseforum/client'

export function countNewTopics(current: TopicPayload[], incoming: TopicPayload[]): number {
  if (current.length === 0) return incoming.length
  const newestKnownID = Math.max(...current.map((topic) => topic.id))
  return incoming.reduce((count, topic) => count + (topic.id > newestKnownID ? 1 : 0), 0)
}

export function prependTopics(current: TopicPayload[], incoming: TopicPayload[]): TopicPayload[] {
  const seen = new Set<number>()
  const merged: TopicPayload[] = []
  for (const topic of [...incoming, ...current]) {
    if (seen.has(topic.id)) continue
    seen.add(topic.id)
    merged.push(topic)
  }
  return merged
}

export function firstPageUrl(pageUrl: string, origin: string): URL {
  const url = new URL(pageUrl, origin)
  url.searchParams.delete('page')
  return url
}
