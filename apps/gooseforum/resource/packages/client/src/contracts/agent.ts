import type { PostWindowPayload, SearchPageProps } from './payload.js'
import type { components } from '../gen/openapi.js'

export type AgentEvent = components['schemas']['AgentEvent']
export type AgentEventData = components['schemas']['AgentEventData']
export type AgentEventPage = NonNullable<components['schemas']['AgentEventsSuccess']['result']>
export type AgentAckRequest = components['schemas']['AgentAckRequest']

export interface MentionTarget {
  userId: number
  username: string
  nickname: string
  avatarUrl: string
  actorType: 'human' | 'bot'
}

export interface AgentMeResult {
  agentId: number
  username: string
  nickname: string
  avatarUrl: string
  tokenPrefix: string
  enabled: 0 | 1
  createdAt: number
  updatedAt: number
}

export interface AgentTopicItem {
  agentRepliesDisabled?: boolean
  id: number
  title: string
  excerpt: string
  categoryIds: number[]
  userId: number
  status: 0 | 1
  processStatus: 0 | 1 | 2
  replyCount: number
  viewCount: number
  postCount: number
  lastPostedAt?: number
  createdAt: number
  updatedAt: number
}

export interface AgentTopicListResult {
  list: AgentTopicItem[]
  page: number
  pageSize: number
  hasNext: boolean
}

export interface AgentWriteTopicRequest {
  title: string
  content: string
  categoryId: number[]
  sourceEventId?: string
}

export type AgentPostListResult = PostWindowPayload

export interface AgentCreatePostRequest {
  content: string
  replyToPostId?: number
  sourceEventId?: string
}

export interface AgentCreatePostResult {
  id: number
  postNo: number
  renderedContent: string
  isAnswer?: boolean
}

export type AgentSearchResult = SearchPageProps
