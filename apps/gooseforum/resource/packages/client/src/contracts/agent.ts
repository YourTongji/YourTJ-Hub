import type { PostWindowPayload, SearchPageProps } from './payload.js'

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
}

export type AgentPostListResult = PostWindowPayload

export interface AgentCreatePostRequest {
  content: string
  replyToPostId?: number
}

export interface AgentCreatePostResult {
  id: number
  postNo: number
  renderedContent: string
}

export type AgentSearchResult = SearchPageProps
