// CourseSummaryPayload 以别名导入：本文件 1663 行另有一个同名但形状不同的
// CourseSummaryPayload（AI 总结：consensus/keywords/pros/cons），二者同名异物。
// 这里导入的是课程卡片（id/name/ratingAvg/...），故别名为 CourseCatalogItem 避免混淆。
import type { CourseSummaryPayload as CourseCatalogItem, ModerationDeletedContentView, ModerationLogListResponse, ModerationReportListResponse, NotificationFilter, NotificationListResponse, PostPayload, PostWindowPayload, UserCardPayload } from '@gooseforum/client'
import { i18n } from './i18n'
import { resolveApiMessage } from './api-message'

interface ApiResponse<T> {
  code?: number
  messageCode?: string
  params?: Record<string, unknown>
  result?: T
  data?: T
}

export class ApiResponseError extends Error {
  readonly messageCode?: string
  readonly retryAfterSeconds?: number

  constructor(message: string, messageCode?: string, retryAfterSeconds?: number) {
    super(message)
    this.name = 'ApiResponseError'
    this.messageCode = messageCode
    this.retryAfterSeconds = retryAfterSeconds
  }
}

function rateLimitMessage(data: ApiResponse<unknown>, fallback: string, retryAfterSeconds?: number) {
  return resolveApiMessage({
    ...data,
    params: { ...(data.params ?? {}), retryAfterSeconds },
  }, fallback)
}

function responseMessage(data: ApiResponse<unknown>, fallback: string) {
  return resolveApiMessage(data, fallback)
}

function t(key: string) {
  return i18n.global.t(key)
}

async function readApiResponse<T>(response: Response, fallback: string): Promise<T> {
  const data = await response.json().catch(() => undefined) as ApiResponse<T> | undefined
  if (response.status === 429) {
    const retryHeader = Number(response.headers.get('Retry-After'))
    const retryAfterSeconds = Number.isFinite(retryHeader) && retryHeader > 0
      ? retryHeader
      : Number(data?.params?.retryAfterSeconds) || undefined
    throw new ApiResponseError(
      data?.messageCode ? rateLimitMessage(data, fallback, retryAfterSeconds) : fallback,
      data?.messageCode,
      retryAfterSeconds,
    )
  }
  if (data?.code !== undefined && data.code !== 0) {
    throw new ApiResponseError(responseMessage(data, fallback), data.messageCode)
  }
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }
  if (!data) {
    throw new Error(fallback)
  }
  return (data.result ?? data.data) as T
}

async function readApiSuccessMessage(response: Response, successFallback: string, errorFallback: string): Promise<string> {
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }
  const data = (await response.json()) as ApiResponse<unknown>
  if (data.code !== undefined && data.code !== 0) {
    throw new ApiResponseError(responseMessage(data, errorFallback), data.messageCode)
  }
  return responseMessage(data, successFallback)
}

export interface CreatePostResult {
  id: number
  postNo?: number
  renderedContent: string
}

export interface UpdatePostResult {
  id: number
  postNo?: number
  content: string
  renderedContent: string
  updatedAt: string
  lastEditorId: number
  lastEditedAt: string
  revisionCount: number
}

export interface PostRevisionResult {
  postId: number
  versions: Array<{
    version: number
    editor: PostPayload['author']
    content: string
    renderedHTML: string
    processStatus: number
    createdAt: string
  }>
  hasMore: boolean
  beforeVersion: number
}

export async function getPostRevisions(postId: number, beforeVersion = 0, limit = 20): Promise<PostRevisionResult> {
  const params = new URLSearchParams({
    postId: String(postId),
    limit: String(limit),
  })
  if (beforeVersion > 0) params.set('beforeVersion', String(beforeVersion))
  const response = await fetch(`/api/forum/posts/revisions?${params.toString()}`, {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<PostRevisionResult>(response, t('api.revisionsLoadFailed'))
}


export async function updatePost(postId: number, content: string): Promise<UpdatePostResult> {
  const response = await fetch('/api/forum/posts/update', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      postId,
      content,
    }),
  })
  return readApiResponse<UpdatePostResult>(response, t('api.replyUpdateFailed'))
}

export interface DeletePostResult {
  hasChildren: boolean
}

export async function deletePost(postId: number): Promise<DeletePostResult> {
  const response = await fetch('/api/forum/posts/delete', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      postId,
    }),
  })
  return readApiResponse<DeletePostResult>(response, t('api.replyDeleteFailed'))
}

export async function deleteTopic(topicId: number): Promise<boolean> {
  const response = await fetch('/api/forum/topics/delete', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      topicId,
    }),
  })
  return readApiResponse<boolean>(response, t('api.topicDeleteFailed'))
}

export type DeletedContentType = 'topic' | 'post'

export interface DeletedContentItem {
  id: number
  contentType: DeletedContentType
  title?: string
  excerpt?: string
  topicId?: number
  postNo?: number
  visibility: string
  retention: string
  deletedAt: string
  canRestore: boolean
  canPermanent: boolean
  hasReplies?: boolean
}

export interface DeletedContentListResult {
  items: DeletedContentItem[]
  hasMore: boolean
  nextCursorId: number
}

export async function getDeletedContent(contentType: DeletedContentType, cursorId = 0, limit = 20): Promise<DeletedContentListResult> {
  const params = new URLSearchParams({
    contentType,
    limit: String(limit),
  })
  if (cursorId > 0) params.set('cursorId', String(cursorId))

  const response = await fetch(`/api/forum/user/deleted-content?${params.toString()}`, {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<DeletedContentListResult>(response, t('api.deletedContentLoadFailed'))
}

export async function restoreDeletedContent(contentType: DeletedContentType, contentId: number): Promise<boolean> {
  const response = await fetch('/api/forum/user/content-restore', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      contentType,
      contentId,
    }),
  })
  return readApiResponse<boolean>(response, t('api.contentRestoreFailed'))
}

export async function purgeDeletedContent(contentType: DeletedContentType, contentId: number): Promise<boolean> {
  const response = await fetch('/api/forum/user/content-purge', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      contentType,
      contentId,
      reason: 'user_purge',
    }),
  })
  return readApiResponse<boolean>(response, t('api.contentPurgeFailed'))
}

/** 隐私紧急删除（PRD R8）：跳过 30 天恢复窗口，全渠道立即彻底删除。 */
export async function privacyEraseContent(contentType: DeletedContentType, contentId: number): Promise<boolean> {
  const response = await fetch('/api/forum/user/content-privacy-erase', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      contentType,
      contentId,
    }),
  })
  return readApiResponse<boolean>(response, t('api.contentPurgeFailed'))
}

/** 删除生命周期埋点（PRD R14）：前端点击/确认类事件上报。 */
export async function reportContentEvent(eventType: 'content_delete_clicked' | 'content_delete_confirmed', contentType: DeletedContentType, contentId: number): Promise<boolean> {
  const response = await fetch('/api/forum/user/content-event', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      eventType,
      contentType,
      contentId,
    }),
  })
  return readApiResponse<boolean>(response, t('api.operationFailed'))
}

export interface MyContentItem {
  id: number
  contentType: DeletedContentType
  title: string
  excerpt?: string
  topicId?: number
  postNo?: number
  createdAt: string
}

export interface MyContentListResult {
  items: MyContentItem[]
  hasMore: boolean
  nextCursorId: number
}

/** 我的内容列表（PRD R9）：本人仍公开的内容/回复，供批量删除。 */
export async function getMyContent(contentType: DeletedContentType, cursorId = 0, limit = 20): Promise<MyContentListResult> {
  const params = new URLSearchParams({ contentType, limit: String(limit) })
  if (cursorId > 0) params.set('cursorId', String(cursorId))
  const response = await fetch(`/api/forum/user/my-content?${params.toString()}`, {
    headers: { Accept: 'application/json' },
  })
  return readApiResponse<MyContentListResult>(response, t('api.deletedContentLoadFailed'))
}

export interface BatchDeleteResultItem {
  contentId: number
  success: boolean
  message?: string
}

export interface BatchDeleteContentResult {
  succeeded: number
  failed: number
  results: BatchDeleteResultItem[]
}

/** 批量删除本人内容（PRD R9）：超过频率阈值时后端要求二次确认。
 * force=true 时后端强制校验当前用户密码，防止账号被盗后无脑清空内容。 */
export async function batchDeleteContent(
  contentType: DeletedContentType,
  contentIds: number[],
  force = false,
  password = '',
): Promise<BatchDeleteContentResult> {
  const response = await fetch('/api/forum/user/content-batch-delete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ contentType, contentIds, force, password }),
  })
  return readApiResponse<BatchDeleteContentResult>(response, t('api.topicDeleteFailed'))
}

/** 注销账号（PRD R10）：mode=anonymize 保留内容匿名化；mode=delete 先删除全部内容再注销。
 * 注销不可逆，后端强制校验当前密码。 */
export async function closeAccount(mode: 'anonymize' | 'delete', password: string): Promise<boolean> {
  const response = await fetch('/api/forum/user/account-close', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ mode, password }),
  })
  return readApiResponse<boolean>(response, t('api.operationFailed'))
}

/** 退出登录并吊销当前会话。 */
export async function logout(): Promise<boolean> {
  const response = await fetch('/api/logout', { method: 'POST' })
  return readApiResponse<boolean>(response, t('api.operationFailed'))
}

export interface PostWindowInput {
  topicId: number
  anchorPostId?: number
  anchorPostNo?: number
  beforePostNo?: number
  afterPostNo?: number
  limit?: number
}

export async function getPostWindow(input: PostWindowInput): Promise<PostWindowPayload> {
  const params = new URLSearchParams({
    topicId: String(input.topicId),
  })
  if (input.anchorPostId) params.set('anchorPostId', String(input.anchorPostId))
  if (input.anchorPostNo) params.set('anchorPostNo', String(input.anchorPostNo))
  if (input.beforePostNo) params.set('beforePostNo', String(input.beforePostNo))
  if (input.afterPostNo) params.set('afterPostNo', String(input.afterPostNo))
  if (input.limit) params.set('limit', String(input.limit))

  const response = await fetch(`/api/forum/posts/window?${params.toString()}`, {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<PostWindowPayload>(response, t('api.repliesLoadFailed'))
}

export async function likeTopic(id: number, action: 1 | 2): Promise<boolean> {
  const response = await fetch('/api/forum/topics/like', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      topicId: id,
      action,
    }),
  })
  return readApiResponse<boolean>(response, t('api.likeFailed'))
}

export async function bookmarkTopic(id: number, action: 1 | 2): Promise<boolean> {
  const response = await fetch('/api/forum/topics/bookmark', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      topicId: id,
      action,
    }),
  })
  return readApiResponse<boolean>(response, t('api.bookmarkFailed'))
}

export async function bookmarkCourse(courseId: number, action: 1 | 2): Promise<boolean> {
  const response = await fetch('/api/forum/courses/bookmark', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      courseId,
      action,
    }),
  })
  return readApiResponse<boolean>(response, t('api.bookmarkFailed'))
}

export async function watchTopic(id: number, action: 1 | 2): Promise<boolean> {
  const response = await fetch('/api/forum/topics/watch', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      topicId: id,
      action,
    }),
  })
  return readApiResponse<boolean>(response, t('api.watchFailed'))
}

export async function likePost(postId: number, action: 1 | 2): Promise<boolean> {
  const response = await fetch('/api/forum/posts/like', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      postId,
      action,
    }),
  })
  return readApiResponse<boolean>(response, t('api.likeFailed'))
}

export async function bookmarkPost(postId: number, action: 1 | 2): Promise<boolean> {
  const response = await fetch('/api/forum/posts/bookmark', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      postId,
      action,
    }),
  })
  return readApiResponse<boolean>(response, t('api.bookmarkFailed'))
}

export async function updateTopicStatus(id: number, topicStatus: 0 | 1): Promise<boolean> {
  const response = await fetch('/api/forum/topics/status', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      topicId: id,
      topicStatus,
    }),
  })
  return readApiResponse<boolean>(response, t('api.topicStatusFailed'))
}

export async function updateModerationTopicStatus(id: number, action: 'ban' | 'unban'): Promise<boolean> {
  const response = await fetch('/api/forum/moderation/topic-status', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ topicId: id, action }),
  })
  return readApiResponse<boolean>(response, t('api.moderationActionFailed'))
}

export async function submitReport(targetType: 'topic' | 'post', targetId: number, reason: string, note: string): Promise<boolean> {
  const response = await fetch('/api/forum/report', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ targetType, targetId, reason, note }),
  })
  return readApiResponse<boolean>(response, t('api.reportFailed'))
}

export async function updateModerationPostStatus(id: number, action: 'ban' | 'unban'): Promise<boolean> {
  const response = await fetch('/api/forum/moderation/post-status', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ postId: id, action }),
  })
  return readApiResponse<boolean>(response, t('api.moderationActionFailed'))
}

export async function fetchModerationReports(cursor = 0, pageSize = 20, status = 'open'): Promise<ModerationReportListResponse> {
  const response = await fetch('/api/forum/moderation/reports', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ cursor, pageSize, status }),
  })
  return readApiResponse<ModerationReportListResponse>(response, t('api.moderationReportsFailed'))
}

export async function updateModerationReportStatus(id: number, action: 'ban' | 'resolve' | 'reject'): Promise<boolean> {
  const response = await fetch('/api/forum/moderation/report-status', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ id, action }),
  })
  return readApiResponse<boolean>(response, t('api.moderationActionFailed'))
}

export async function fetchModerationLogs(cursor = 0, pageSize = 20): Promise<ModerationLogListResponse> {
  const response = await fetch('/api/forum/moderation/logs', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ cursor, pageSize }),
  })
  return readApiResponse<ModerationLogListResponse>(response, t('api.moderationLogsFailed'))
}

/** 版主查看已删除内容原文（PRD R7）：必须提供理由，每次查看都会记审计日志。 */
export async function viewDeletedContent(contentType: 'topic' | 'post', contentId: number, reason: string): Promise<ModerationDeletedContentView> {
  const response = await fetch('/api/forum/moderation/view-deleted-content', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ contentType, contentId, reason }),
  })
  return readApiResponse<ModerationDeletedContentView>(response, t('api.moderationActionFailed'))
}

export async function markAllNotificationsRead(): Promise<boolean> {
  const response = await fetch('/api/forum/notification/mark-all-read', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  })
  return readApiResponse<boolean>(response, t('api.markReadFailed'))
}

export async function markNotificationRead(notificationId: number): Promise<boolean> {
  const response = await fetch('/api/forum/notification/mark-read', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ notificationId }),
    keepalive: true,
  })
  return readApiResponse<boolean>(response, t('api.markReadFailed'))
}

export async function fetchNotifications(filter: NotificationFilter, cursor = 0, limit = 20): Promise<NotificationListResponse> {
  const params = new URLSearchParams({
    filter,
    cursor: String(cursor),
    limit: String(limit),
  })
  const response = await fetch(`/api/forum/notifications?${params.toString()}`, {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<NotificationListResponse>(response, t('api.notificationsLoadFailed'))
}

export async function getUserCard(userId: number): Promise<UserCardPayload> {
  const response = await fetch(`/api/user-card?userId=${encodeURIComponent(String(userId))}`, {
    headers: {
      Accept: 'application/json',
    },
  })
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }

  const data = (await response.json()) as ApiResponse<UserCardPayload>
  if (data.code !== undefined && data.code !== 0) {
    throw new Error(responseMessage(data, t('api.userLoadFailed')))
  }

  const result = data.result ?? data.data
  if (!result) {
    throw new Error(t('api.userEmpty'))
  }
  return result
}

export async function followUser(userId: number, isFollowing: boolean): Promise<boolean> {
  const response = await fetch('/api/forum/follow-user', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      id: userId,
      action: isFollowing ? 2 : 1,
    }),
  })
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }

  const data = (await response.json()) as ApiResponse<boolean>
  if (data.code !== undefined && data.code !== 0) {
    throw new Error(responseMessage(data, t('api.followFailed')))
  }
  return data.result ?? data.data ?? true
}

export interface SubmitTopicInput {
  topicId: number
  title: string
  content: string
  categoryId: number[]
  topicStatus: 0 | 1
  website?: string
  captchaId?: string
  captchaCode?: string
  contentType?: 0 | 1 | 2 | 3 // 0=regular, 1=question, 2=thought, 3=article
}

export async function submitTopic(topic: SubmitTopicInput): Promise<number> {
  const response = await fetch('/api/forum/topics/write', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(topic),
  })
  if (response.status === 429) {
    return readApiResponse<number>(response, t('api.topicSaveFailed'))
  }
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }

  const data = (await response.json()) as ApiResponse<number>
  if (data.code !== undefined && data.code !== 0) {
    throw new ApiResponseError(responseMessage(data, t('api.topicSaveFailed')), data.messageCode)
  }
  return data.result ?? data.data ?? topic.topicId
}

export async function createPost(topicId: number, content: string, replyToPostId = 0, extra?: { captchaId?: string, captchaCode?: string, website?: string }): Promise<CreatePostResult | number | boolean> {
  const response = await fetch('/api/forum/posts/create', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      topicId,
      content,
      replyToPostId,
      ...(extra ?? {}),
    }),
  })
  return readApiResponse<CreatePostResult | number | boolean>(response, t('api.replyFailed'))
}

export async function uploadImage(file: File): Promise<string> {
  const initResponse = await fetch('/file/img-upload/init', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ filename: file.name, contentType: file.type, size: file.size }),
  })
  const init = await readApiResponse<ImageUploadInitResult>(initResponse, t('api.imageUploadFailed'))
  if (init.mode === 'proxy') return uploadImageThroughServer(file)
  if (init.mode !== 'direct' || !init.name || !init.upload?.url || init.upload.method !== 'POST') {
    if (init.name) await abortDirectImageUpload(init.name)
    throw new Error(t('api.imageUploadEmpty'))
  }

  const formData = new FormData()
  for (const [key, value] of Object.entries(init.upload.fields || {})) formData.append(key, value)
  formData.append('file', file, file.name)
  let uploadResponse: Response
  try {
    uploadResponse = await fetch(init.upload.url, { method: 'POST', body: formData })
  } catch (uploadError) {
    try {
      return await completeDirectImageUpload(init.name)
    } catch {
      // 对象请求可能仍在途；服务端会安全过期未完成的直传对象。
      throw uploadError
    }
  }
  if (!uploadResponse.ok) {
    await abortDirectImageUpload(init.name)
    throw new Error(`HTTP ${uploadResponse.status}`)
  }
  try {
    return await completeDirectImageUpload(init.name)
  } catch (error) {
    if (!isTransientUploadError(error)) await abortDirectImageUpload(init.name)
    throw error
  }
}

interface ImageUploadInitResult {
  mode: 'proxy' | 'direct'
  name?: string
  upload?: {
    url: string
    method: string
    fields: Record<string, string>
    expiresAt: string
  }
}

// uploadImageThroughServer 走服务端代理 multipart 上传（本地存储提供方）。
async function uploadImageThroughServer(file: File): Promise<string> {
  const formData = new FormData()
  formData.append('file', file)
  const response = await fetch('/file/img-upload', {
    method: 'POST',
    body: formData,
  })
  const result = await readApiResponse<{ url: string }>(response, t('api.imageUploadFailed'))
  if (!result?.url) {
    throw new Error(t('api.imageUploadEmpty'))
  }
  return result.url
}

// completeDirectImageUpload 在浏览器直传对象后发布图片；瞬时错误重试一次。
async function completeDirectImageUpload(name: string): Promise<string> {
  const complete = async () => {
    const response = await fetch('/file/img-upload/complete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    const result = await readApiResponse<{ url: string }>(response, t('api.imageUploadFailed'))
    if (!result?.url) throw new Error(t('api.imageUploadEmpty'))
    return result.url
  }
  try {
    return await complete()
  } catch (error) {
    if (!isTransientUploadError(error)) throw error
    return complete()
  }
}

function isTransientUploadError(error: unknown) {
  // 网络错误（TypeError）与 HTTP 5xx（readApiResponse 抛出的裸 Error）视为瞬时错误；
  // 业务失败（ApiResponseError，HTTP 200 + code 1）不重试。
  return error instanceof TypeError || (error instanceof Error && /^HTTP 5\d\d/.test(error.message))
}

// abortDirectImageUpload 取消未完成的直传对象；失败可忽略（服务端清理任务兜底）。
async function abortDirectImageUpload(name: string) {
  try {
    await fetch('/file/img-upload/abort', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
  } catch {
    // 过期的待发布对象也会被服务端清理任务移除。
  }
}

export interface ChatMessagePayload {
  id: number
  senderId: number
  content: string
  msgType: number
  isRead: number
  createdAt: string
  isSelf: boolean
}

export interface ChatMessagesResponse {
  list: ChatMessagePayload[]
  hasMoreBefore: boolean
  hasMoreAfter: boolean
  nextBeforeId: number
  latestId: number
}

export interface ChatMessagesInput {
  convId: number
  beforeId?: number
  afterId?: number
  limit?: number
}

export async function getChatMessages(input: ChatMessagesInput): Promise<ChatMessagesResponse> {
  const response = await fetch('/api/forum/chat/messages', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      convId: input.convId,
      beforeId: input.beforeId || 0,
      afterId: input.afterId || 0,
      limit: input.limit || 30,
    }),
  })
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }

  const data = (await response.json()) as ApiResponse<ChatMessagesResponse>
  if (data.code !== undefined && data.code !== 0) {
    throw new Error(responseMessage(data, t('api.messagesLoadFailed')))
  }
  const result = data.result ?? data.data
  return {
    list: result?.list ?? [],
    hasMoreBefore: Boolean(result?.hasMoreBefore),
    hasMoreAfter: Boolean(result?.hasMoreAfter),
    nextBeforeId: result?.nextBeforeId ?? 0,
    latestId: result?.latestId ?? 0,
  }
}

export async function sendChatMessage(peerId: number, content: string): Promise<number> {
  const response = await fetch('/api/forum/chat/send', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ peerId, content, msgType: 1 }),
  })
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }

  const data = (await response.json()) as ApiResponse<{ convId: number }>
  if (data.code !== undefined && data.code !== 0) {
    throw new Error(responseMessage(data, t('api.sendFailed')))
  }
  return data.result?.convId ?? data.data?.convId ?? 0
}

export async function markChatRead(convId: number): Promise<boolean> {
  const response = await fetch('/api/forum/chat/mark-read', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ convId }),
  })
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }

  const data = (await response.json()) as ApiResponse<boolean>
  if (data.code !== undefined && data.code !== 0) {
    throw new Error(responseMessage(data, t('api.markReadFailed')))
  }
  return data.result ?? data.data ?? true
}

export interface SaveUserInfoInput {
  nickname: string
  locale?: string
  bio: string
  signature: string
  website: string
  websiteName: string
  externalInformation: Record<string, { link?: string }>
}

export async function saveUserInfo(input: SaveUserInfoInput): Promise<boolean> {
  const response = await fetch('/api/set-user-info', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  })
  await readApiResponse<unknown>(response, t('api.profileSaveFailed'))
  return true
}

export async function saveUserProfileCover(profileCoverUrl: string): Promise<boolean> {
  const response = await fetch('/api/set-user-profile-cover', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ profileCoverUrl }),
  })
  await readApiResponse<unknown>(response, t('api.coverSaveFailed'))
  return true
}

export async function savePresetAvatar(avatarUrl: string): Promise<string> {
  const response = await fetch('/api/set-preset-avatar', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ avatarUrl }),
  })
  const result = await readApiResponse<{ avatarUrl?: string }>(response, t('api.avatarPresetFailed'))
  if (!result.avatarUrl) throw new Error(t('api.avatarPresetEmpty'))
  return result.avatarUrl
}

export async function wearBadge(badgeCode: string): Promise<boolean> {
  const response = await fetch('/api/wear-badge', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ badgeCode }),
  })
  await readApiResponse<unknown>(response, t('api.badgeWearFailed'))
  return true
}

export async function saveUserEmail(email: string, password: string): Promise<boolean> {
  const response = await fetch('/api/set-user-email', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password }),
  })
  await readApiResponse<unknown>(response, t('api.emailSaveFailed'))
  return true
}

export async function resendActivationEmail(): Promise<string> {
  const response = await fetch('/api/resend-activation-email', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  })
  return readApiSuccessMessage(response, t('settings.status.activationEmailSent'), t('api.activationEmailSendFailed'))
}

export async function saveUserName(username: string): Promise<boolean> {
  const response = await fetch('/api/set-user-name', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ username }),
  })
  await readApiResponse<unknown>(response, t('api.usernameSaveFailed'))
  return true
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<boolean> {
  const response = await fetch('/api/change-password', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ oldPassword, newPassword }),
  })
  await readApiResponse<unknown>(response, t('api.passwordChangeFailed'))
  return true
}

export async function uploadAvatar(avatar: Blob | Blob[]): Promise<string> {
  const formData = new FormData()
  const avatars = Array.isArray(avatar) ? avatar : [avatar]
  const fields = ['avatar', 'avatarMedium']
  const filenames = ['avatar.webp', 'avatar_medium.webp']
  avatars.slice(0, 2).forEach((item, index) => {
    formData.append(fields[index], item, item instanceof File ? item.name : filenames[index])
  })
  const response = await fetch('/api/upload-avatar', {
    method: 'POST',
    body: formData,
  })
  const result = await readApiResponse<string | { avatarUrl?: string; url?: string }>(response, t('api.avatarUploadFailed'))
  if (typeof result === 'string') return result
  const url = result.avatarUrl || result.url
  if (!url) throw new Error(t('api.avatarUploadEmpty'))
  return url
}

// uploadImageFile 上传一张通用图片（个人资料封面等），
// 复用 /file/img-upload 的权限、类型与大小校验。
export async function uploadImageFile(file: Blob, filename: string): Promise<string> {
  const formData = new FormData()
  formData.append('file', file, filename)
  const response = await fetch('/file/img-upload', {
    method: 'POST',
    body: formData,
  })
  const result = await readApiResponse<string | { url?: string }>(response, t('api.imageUploadFailed'))
  if (typeof result === 'string') return result
  const url = result.url
  if (!url) throw new Error(t('api.imageUploadEmpty'))
  return url
}

export interface OAuthBindingPayload {
  bound: boolean
  provider?: string
  createdAt?: string
  updatedAt?: string
}

export type OAuthBindingsPayload = Record<string, OAuthBindingPayload>

export async function getOAuthBindings(): Promise<OAuthBindingsPayload> {
  const response = await fetch('/api/oauth/bindings', {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<OAuthBindingsPayload>(response, t('api.bindingsLoadFailed'))
}

export async function unbindOAuth(provider: string): Promise<boolean> {
  const response = await fetch(`/api/auth/${encodeURIComponent(provider)}/unbind`, {
    method: 'POST',
  })
  await readApiResponse<unknown>(response, t('api.unbindFailed'))
  return true
}

export interface UserSessionPayload {
  id: number
  ipMasked: string
  userAgent: string
  createdAt: number
  expiresAt: number
  isCurrent: boolean
}

export async function listSessions(): Promise<UserSessionPayload[]> {
  const response = await fetch('/api/user/sessions', {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<UserSessionPayload[]>(response, t('api.sessionsLoadFailed'))
}

export async function revokeSession(id: number): Promise<boolean> {
  const response = await fetch('/api/user/sessions/revoke', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ id }),
  })
  await readApiResponse<unknown>(response, t('api.sessionRevokeFailed'))
  return true
}

export async function revokeAllSessions(): Promise<boolean> {
  const response = await fetch('/api/user/sessions/revoke-all', {
    method: 'POST',
  })
  await readApiResponse<unknown>(response, t('api.sessionRevokeAllFailed'))
  return true
}

export interface TotpSetupPayload {
  secret: string
  otpauthUrl: string
}

export interface TotpEnablePayload {
  recoveryCodes: string[]
}

export async function getTotpSetup(password: string): Promise<TotpSetupPayload> {
  const response = await fetch('/api/user/totp/setup', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ password }),
  })
  return readApiResponse<TotpSetupPayload>(response, t('api.totpSetupFailed'))
}

export async function enableTotp(code: string): Promise<TotpEnablePayload> {
  const response = await fetch('/api/user/totp/enable', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ code }),
  })
  return readApiResponse<TotpEnablePayload>(response, t('api.totpEnableFailed'))
}

export async function disableTotp(code: string): Promise<boolean> {
  const response = await fetch('/api/user/totp/disable', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ code }),
  })
  await readApiResponse<unknown>(response, t('api.totpDisableFailed'))
  return true
}

export async function verifyTotp(code: string): Promise<boolean> {
  const response = await fetch('/api/auth/totp/verify', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ code }),
  })
  await readApiResponse<unknown>(response, t('api.totpVerifyFailed'))
  return true
}

export interface TotpStatusPayload {
  enabled: boolean
}

export async function getTotpStatus(): Promise<TotpStatusPayload> {
  const response = await fetch('/api/user/totp/status', {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<TotpStatusPayload>(response, t('api.totpSetupFailed'))
}

interface CaptchaPayload {
  captchaId: string
  captchaImg: string
}

interface LoginPublicKeyPayload {
  publicKey: string
  serverTs: number
}

const loginInvalidRequestCode = 'auth.login.invalidRequest'

let publicKeyPromise: Promise<LoginPublicKeyPayload> | undefined

export async function getCaptcha(): Promise<CaptchaPayload> {
  const response = await fetch('/api/get-captcha', {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<CaptchaPayload>(response, t('api.captchaLoadFailed'))
}

export interface LoginResult {
  twoFactorRequired: boolean
}

export async function login(username: string, password: string, captchaId: string, captchaCode: string): Promise<LoginResult> {
  let result: LoginResult
  try {
    result = await submitLogin(username, password, captchaId, captchaCode, true)
  } catch (error) {
    if (!(error instanceof ApiResponseError) || error.messageCode !== loginInvalidRequestCode) {
      throw error
    }
    clearLoginPublicKey()
    result = await submitLogin(username, password, captchaId, captchaCode, true)
  }
  return result
}

async function submitLogin(username: string, password: string, captchaId: string, captchaCode: string, refreshKey = false): Promise<LoginResult> {
  const encryptedPassword = await encryptLoginPassword(password, refreshKey)
  const response = await fetch('/api/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      username,
      encryptedPassword,
      captchaId,
      captchaCode,
    }),
  })
  const result = await readApiResponse<{ twoFactorRequired?: boolean }>(response, t('api.loginFailed'))
  return { twoFactorRequired: Boolean(result?.twoFactorRequired) }
}


export async function register(
  username: string,
  email: string,
  password: string,
  captchaId: string,
  captchaCode: string,
  locale?: string,
  website = '',
): Promise<string> {
  const response = await fetch('/api/register', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      userName: username,
      email,
      passWord: password,
      locale,
      captchaId,
      captchaCode,
      website,
    }),
  })
  return readApiSuccessMessage(response, t('auth.validation.registerSuccess'), t('api.registerFailed'))
}

export async function forgotPassword(email: string, captchaId: string, captchaCode: string, website = ''): Promise<string> {
  const response = await fetch('/api/forgot-password', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      email,
      captchaId,
      captchaCode,
      website,
    }),
  })
  return readApiSuccessMessage(response, t('server.auth.passwordReset.mailQueued'), t('api.resetEmailFailed'))
}


export async function resetPassword(token: string, newPassword: string): Promise<string> {
  const response = await fetch('/api/reset-password', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      token,
      newPassword,
    }),
  })
  return readApiSuccessMessage(response, t('server.auth.passwordReset.success'), t('api.passwordResetFailed'))
}

async function encryptLoginPassword(password: string, refreshKey = false): Promise<string> {
  const key = await getLoginPublicKey(refreshKey)
  const payload = JSON.stringify({
    password,
    ts: key.serverTs,
  })
  if (!window.crypto?.subtle) {
    return encryptLoginPasswordWithForge(key.publicKey, payload)
  }
  try {
    return await encryptLoginPasswordWithWebCrypto(key.publicKey, payload)
  } catch {
    return encryptLoginPasswordWithForge(key.publicKey, payload)
  }
}

async function encryptLoginPasswordWithWebCrypto(publicKey: string, payload: string): Promise<string> {
  const key = await window.crypto.subtle.importKey(
    'spki',
    pemToArrayBuffer(publicKey),
    {
      name: 'RSA-OAEP',
      hash: 'SHA-256',
    },
    false,
    ['encrypt'],
  )

  const encrypted = await window.crypto.subtle.encrypt(
    { name: 'RSA-OAEP' },
    key,
    new TextEncoder().encode(payload),
  )

  return arrayBufferToBase64(encrypted)
}

async function encryptLoginPasswordWithForge(publicKey: string, payload: string): Promise<string> {
  const { default: forge } = await import('node-forge')
  const key = forge.pki.publicKeyFromPem(publicKey)
  const encrypted = key.encrypt(forge.util.encodeUtf8(payload), 'RSA-OAEP', {
    md: forge.md.sha256.create(),
    mgf1: {
      md: forge.md.sha256.create(),
    },
  })
  return forge.util.encode64(encrypted)
}

async function getLoginPublicKey(refresh = false): Promise<LoginPublicKeyPayload> {
  if (refresh) {
    clearLoginPublicKey()
  }
  if (!publicKeyPromise) {
    publicKeyPromise = fetch('/api/login-public-key', {
      headers: {
        Accept: 'application/json',
      },
    })
      .then((response) => readApiResponse<LoginPublicKeyPayload>(response, t('api.loginKeyLoadFailed')))
      .catch((error) => {
        publicKeyPromise = undefined
        throw error
      })
  }
  return publicKeyPromise
}

function clearLoginPublicKey() {
  publicKeyPromise = undefined
}

function pemToArrayBuffer(pem: string): ArrayBuffer {
  const base64 = pem
    .replace(/-----BEGIN PUBLIC KEY-----/g, '')
    .replace(/-----END PUBLIC KEY-----/g, '')
    .replace(/\s/g, '')
  const binary = window.atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes.buffer
}

function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i += 1) {
    binary += String.fromCharCode(bytes[i])
  }
  return window.btoa(binary)
}

// ---- 课评（course review）----

export interface ReviewAuthorPayload {
  kind: 'anonymous' | 'member' | 'legacy'
  label: string
  avatarUrl?: string
}

export interface ReviewViewerPayload {
  canEdit: boolean
  canDelete: boolean
  isHelpful: boolean
  isDisliked: boolean
}

export interface ReviewPayload {
  id: number
  offeringId: number
  rating: number | null
  content: string
  contentHtml: string
  author: ReviewAuthorPayload
  viewer: ReviewViewerPayload
  helpfulCount: number
  dislikeCount: number
  createdAt: string
  updatedAt: string
}

export interface CreateCourseReviewInput {
  offeringId: number
  rating: number
  content: string
  isAnonymous: boolean
}

export interface UpdateCourseReviewInput {
  rating?: number | null
  content?: string
  isAnonymous?: boolean
}

export interface ModerationCourseReviewReportItem {
  id: number
  reviewId: number
  reason: string
  note: string
  status: string
  resolution: string
  excerpt: string
  reporter: { id: number; username: string; avatarUrl: string }
  handler: { id: number; username: string; avatarUrl: string }
  createdAt: string
  handledAt?: string
  reportCount: number
}

export interface ModerationCourseReviewReportListResponse {
  items: ModerationCourseReviewReportItem[]
  nextCursor: number
  hasNext: boolean
}

export interface CourseReviewAuthorRevealPayload {
  reviewId: number
  authorUserId?: number
  username?: string
  nickname?: string
  isAnonymous: boolean
  source: string
}

export interface RelatedCourseItem {
  id: number
  primaryCode: string
  name: string
  department: string
  teacherName?: string
  instructors?: string[]
  ratingAvg: number
  ratingCount: number
  reviewCount: number
}

// 课程沿革条目（GET /courses/:id/related 的 lineage）：已确认关系（approved/merged），
// direction=to（本卡为当前卡，from 为历史旧卡）/ from（本卡为历史卡，to 为新卡）。
export interface CourseLineageItem {
  relationId: number
  fromCourseId: number
  fromName: string
  toCourseId: number
  toName: string
  relationType: 'EQUIVALENT' | 'RENAMED_FROM' | 'SPLIT_FROM' | 'MERGED_FROM' | 'RELATED'
  status: 'approved' | 'merged'
  direction: 'to' | 'from'
}
export interface CourseRelatedResult {
  teacherOtherCourses: RelatedCourseItem[]
  sameCourseOtherTeachers: RelatedCourseItem[]
  lineage?: CourseLineageItem[]
}

export async function getCourseRelated(courseId: number): Promise<CourseRelatedResult> {
  const response = await fetch(`/api/forum/courses/${courseId}/related`, {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<CourseRelatedResult>(response, t('api.courseRelatedLoadFailed'))
}

// CourseCatalogPageResult 课程目录 JSON API 分页结果（GET /api/forum/courses）。
// 命名区别于 SSR props 的 CourseCatalogPageProps：后者额外携带 departments/terms/
// campuses/收藏集合等静态面板数据，翻页并不需要。
export interface CourseCatalogPageResult {
  list: CourseCatalogItem[]
  page: number
  size: number
  total: number
  hasNext: boolean
}

export interface ListCoursesInput {
  keyword?: string
  department?: string[]
  term?: string[]
  campus?: string[]
  instructor?: string[]
  onlyWithReviews?: boolean
  sortBy?: string
  page?: number
  size?: number
}

// 课程目录翻页走 JSON API 而非 SSR 页面路由：SSR 首屏才需要 departments/terms/
// campuses 等静态筛选项与收藏集合，翻页重复请求 SSR 会把这些查询按滚动页数
// 线性放大（每页重跑 3 个 DISTINCT 扫描 + 收藏集合查询，而前端只取课程列表）。
export async function listCourses(input: ListCoursesInput = {}): Promise<CourseCatalogPageResult> {
  const params = new URLSearchParams()
  if (input.keyword) params.set('keyword', input.keyword)
  for (const value of input.department ?? []) params.append('department', value)
  for (const value of input.term ?? []) params.append('term', value)
  for (const value of input.campus ?? []) params.append('campus', value)
  for (const value of input.instructor ?? []) params.append('instructor', value)
  if (input.onlyWithReviews) params.set('onlyWithReviews', '1')
  if (input.sortBy) params.set('sortBy', input.sortBy)
  params.set('page', String(input.page ?? 1))
  params.set('size', String(input.size ?? 20))
  const response = await fetch(`/api/forum/courses?${params.toString()}`, {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<CourseCatalogPageResult>(response, t('common.loadFailed'))
}

export interface ReviewPage {
  list: ReviewPayload[]
  nextCursor?: string
  total: number
}

// 默认 pageSize=20（issue #174 验收约定默认 20、上限 50）。
export async function listCourseReviews(courseId: number, offeringId = 0, cursor = '', pageSize = 20): Promise<ReviewPage> {
  const params = new URLSearchParams()
  if (offeringId > 0) params.set('offeringId', String(offeringId))
  if (cursor) params.set('cursor', cursor)
  params.set('pageSize', String(pageSize))
  const query = params.toString()
  const response = await fetch(`/api/forum/courses/${courseId}/reviews${query ? `?${query}` : ''}`, {
    headers: {
      Accept: 'application/json',
    },
  })
  return readApiResponse<ReviewPage>(response, t('api.reviewsLoadFailed'))
}

export async function createCourseReview(input: CreateCourseReviewInput): Promise<ReviewPayload> {
  const response = await fetch('/api/forum/course-reviews', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  })
  return readApiResponse<ReviewPayload>(response, t('api.reviewCreateFailed'))
}

export async function updateCourseReview(reviewId: number, input: UpdateCourseReviewInput): Promise<ReviewPayload> {
  const response = await fetch(`/api/forum/course-reviews/${reviewId}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  })
  return readApiResponse<ReviewPayload>(response, t('api.reviewUpdateFailed'))
}

export async function deleteCourseReview(reviewId: number): Promise<boolean> {
  const response = await fetch(`/api/forum/course-reviews/${reviewId}`, {
    method: 'DELETE',
  })
  return readApiResponse<boolean>(response, t('api.reviewDeleteFailed'))
}

export async function setReviewHelpful(reviewId: number, helpful: boolean): Promise<boolean> {
  const response = await fetch(`/api/forum/course-reviews/${reviewId}/helpful`, {
    method: helpful ? 'PUT' : 'DELETE',
  })
  return readApiResponse<boolean>(response, t('api.reviewHelpfulFailed'))
}

export async function setReviewDislike(reviewId: number, dislike: boolean): Promise<boolean> {
  const response = await fetch(`/api/forum/course-reviews/${reviewId}/dislike`, {
    method: dislike ? 'PUT' : 'DELETE',
  })
  return readApiResponse<boolean>(response, t('api.reviewDislikeFailed'))
}

export async function reportCourseReview(reviewId: number, reason: string, note: string): Promise<boolean> {
  const response = await fetch(`/api/forum/course-reviews/${reviewId}/reports`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ reason, note }),
  })
  return readApiResponse<boolean>(response, t('api.reviewReportFailed'))
}

export async function moderationCourseReviewStatus(reviewId: number, action: 'hide' | 'show'): Promise<boolean> {
  const response = await fetch('/api/forum/moderation/course-review-status', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ reviewId, action }),
  })
  return readApiResponse<boolean>(response, t('api.moderationActionFailed'))
}

export async function fetchModerationCourseReviewReports(
  status: 'open' | 'resolved' | 'rejected',
  cursor = 0,
  pageSize = 20,
): Promise<ModerationCourseReviewReportListResponse> {
  const response = await fetch('/api/forum/moderation/course-review-reports', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ status, cursor, pageSize }),
  })
  return readApiResponse<ModerationCourseReviewReportListResponse>(response, t('api.moderationCourseReviewReportsFailed'))
}

export async function revealCourseReviewAuthor(reviewId: number, reason: string): Promise<CourseReviewAuthorRevealPayload> {
  const response = await fetch('/api/forum/moderation/course-review-reveal', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ reviewId, reason }),
  })
  return readApiResponse<CourseReviewAuthorRevealPayload>(response, t('api.moderationCourseReviewRevealFailed'))
}

// ---- 课评管理（课程/评价 CRUD + 统计重建，CourseManager） ----

export interface AdminCourseItem {
  id: number
  primaryCode: string
  name: string
  department: string
  creditX10: number
  status: number
  aliases: string[]
  instructors: string[]
  reviewCount: number
  ratingAvg?: number
  createdAt: string
}

export interface AdminCourseListResult {
  list: AdminCourseItem[]
  page: number
  size: number
  total: number
  hasNext: boolean
}

export interface AdminCourseCreateInput {
  primaryCode: string
  name: string
  department?: string
  creditX10?: number
  aliases?: string[]
  instructors?: string[]
}

export interface AdminCourseUpdateInput {
  primaryCode?: string
  name?: string
  department?: string
  creditX10?: number
  reviewScope?: string
  teamKey?: string
  aliases?: string[]
  instructors?: string[]
}

export interface AdminReviewItem {
  id: number
  offeringId: number
  courseId: number
  courseCode: string
  courseName: string
  rating: number | null
  content: string
  status: number
  author: ReviewAuthorPayload
  createdAt: string
  updatedAt: string
}

export interface AdminReviewListResult {
  items: AdminReviewItem[]
  nextCursor: number
  hasNext: boolean
}

export interface AdminReviewUpdateInput {
  rating?: number | null
  content?: string
}

export async function fetchAdminCourses(keyword = '', department = '', page = 1, pageSize = 20): Promise<AdminCourseListResult> {
  const response = await fetch('/api/forum/moderation/course-list', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ keyword, department, page, pageSize }),
  })
  return readApiResponse<AdminCourseListResult>(response, t('api.adminCourseListFailed'))
}

export async function createAdminCourse(input: AdminCourseCreateInput): Promise<AdminCourseItem> {
  const response = await fetch('/api/forum/moderation/course-create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return readApiResponse<AdminCourseItem>(response, t('api.adminCourseCreateFailed'))
}

export async function updateAdminCourse(courseId: number, input: AdminCourseUpdateInput): Promise<AdminCourseItem> {
  const response = await fetch('/api/forum/moderation/course-update', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ courseId, ...input }),
  })
  return readApiResponse<AdminCourseItem>(response, t('api.adminCourseUpdateFailed'))
}

export async function deleteAdminCourse(courseId: number): Promise<boolean> {
  const response = await fetch('/api/forum/moderation/course-delete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ courseId }),
  })
  return readApiResponse<boolean>(response, t('api.adminCourseDeleteFailed'))
}

export async function fetchAdminReviews(keyword = '', status = -1, cursor = 0, pageSize = 20): Promise<AdminReviewListResult> {
  const response = await fetch('/api/forum/moderation/course-review-list', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ keyword, status, cursor, pageSize }),
  })
  return readApiResponse<AdminReviewListResult>(response, t('api.adminReviewListFailed'))
}

export async function updateAdminReview(reviewId: number, input: AdminReviewUpdateInput): Promise<ReviewPayload> {
  const response = await fetch('/api/forum/moderation/course-review-edit', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ reviewId, ...input }),
  })
  return readApiResponse<ReviewPayload>(response, t('api.adminReviewUpdateFailed'))
}

export async function deleteAdminReview(reviewId: number): Promise<boolean> {
  const response = await fetch('/api/forum/moderation/course-review-delete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ reviewId }),
  })
  return readApiResponse<boolean>(response, t('api.adminReviewDeleteFailed'))
}

export async function rebuildCourseStats(): Promise<boolean> {
  const response = await fetch('/api/forum/moderation/course-stats-rebuild', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  })
  return readApiResponse<boolean>(response, t('api.adminCourseStatsRebuildFailed'))
}

// ---- 课程沿革审核（CourseManager） ----

export interface CourseRelationItem {
  id: number
  fromCourseId: number
  toCourseId: number
  relationType: 'EQUIVALENT' | 'RENAMED_FROM' | 'SPLIT_FROM' | 'MERGED_FROM' | 'RELATED'
  source: 'rule' | 'manual'
  confidence: number
  evidenceJson: string
  manual: boolean
  status: 'pending' | 'approved' | 'ignored' | 'merged'
  createdAt: string
  updatedAt: string
}

export interface CourseRelationListResult {
  list: CourseRelationItem[]
  page: number
  size: number
  total: number
  hasNext: boolean
}

export interface CourseRelationCreateInput {
  fromCourseId: number
  toCourseId: number
  relationType: string
  evidence?: string
  confidence?: number
}

export interface CourseMergeResult {
  relationId: number
  fromCourseId: number
  toCourseId: number
  fromName: string
  toName: string
  movedOfferings: number
  migratedAliases: number
  skippedAliases: number
}

export interface AdminCourseDetailItem {
  id: number
  primaryCode: string
  name: string
  department: string
  creditX10: number
  reviewScope?: string
  teamKey?: string
}

export async function fetchCourseRelations(status = '', page = 1, pageSize = 20): Promise<CourseRelationListResult> {
  const response = await fetch('/api/forum/moderation/course-relation-list', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status, page, pageSize }),
  })
  return readApiResponse<CourseRelationListResult>(response, t('api.adminCourseRelationListFailed'))
}

export async function approveCourseRelation(relationId: number): Promise<CourseRelationItem> {
  const response = await fetch('/api/forum/moderation/course-relation-approve', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ relationId }),
  })
  return readApiResponse<CourseRelationItem>(response, t('api.adminCourseRelationOpFailed'))
}

export async function ignoreCourseRelation(relationId: number): Promise<CourseRelationItem> {
  const response = await fetch('/api/forum/moderation/course-relation-ignore', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ relationId }),
  })
  return readApiResponse<CourseRelationItem>(response, t('api.adminCourseRelationOpFailed'))
}

export async function createCourseRelation(input: CourseRelationCreateInput): Promise<CourseRelationItem> {
  const response = await fetch('/api/forum/moderation/course-relation-create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return readApiResponse<CourseRelationItem>(response, t('api.adminCourseRelationCreateFailed'))
}

export async function mergeCourseRelation(relationId: number): Promise<CourseMergeResult> {
  const response = await fetch('/api/forum/moderation/course-merge', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ relationId }),
  })
  return readApiResponse<CourseMergeResult>(response, t('api.courseMergeFailed'))
}

export async function undoMergeCourseRelation(relationId: number): Promise<CourseMergeResult> {
  const response = await fetch('/api/forum/moderation/course-merge-undo', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ relationId }),
  })
  return readApiResponse<CourseMergeResult>(response, t('api.courseMergeUndoFailed'))
}

// 管理端编辑弹窗预填 reviewScope/teamKey 用；隐藏课程详情不可读时由调用方降级。
export async function getCourseDetail(courseId: number): Promise<AdminCourseDetailItem> {
  const response = await fetch(`/api/forum/courses/${courseId}`, {
    headers: { Accept: 'application/json' },
  })
  return readApiResponse<AdminCourseDetailItem>(response, t('api.courseDetailLoadFailed'))
}

// ---- B7: AI 课程总结（issue #181） ----

// CourseSummaryStatus 与后端契约一致；error/rateLimited 为前端本地状态。
// none 仅在 check 预检（?check=true）返回：无缓存行、从未生成过。
export type CourseSummaryStatus = 'cached' | 'generated' | 'insufficient_data' | 'none' | 'disabled' | 'error' | 'rateLimited'

export type CourseSummarySentiment = 'positive' | 'neutral' | 'negative'

export interface CourseSummaryRepresentativeReview {
  excerpt: string
  sentiment: CourseSummarySentiment
}

export interface CourseSummaryPayload {
  consensus: string
  keywords: string[]
  pros: string[]
  cons: string[]
  representativeReviews: CourseSummaryRepresentativeReview[]
}

export interface CourseSummaryResult {
  status: CourseSummaryStatus
  summary?: CourseSummaryPayload
  generatedAt?: string
  model?: string
  retryAfterSeconds?: number
}

// getCourseSummary 获取课程 AI 总结。
// check=true 走只读预检：不生成、不消耗限流，返回 cached/insufficient_data/none/disabled，
// 供页面挂载时决定是否自动展开。
export async function getCourseSummary(courseId: number, refresh = false, check = false): Promise<CourseSummaryResult> {
  const params = new URLSearchParams()
  if (refresh) params.set('refresh', 'true')
  if (check) params.set('check', 'true')
  const query = params.toString()
  const response = await fetch(`/api/forum/courses/${courseId}/summary${query ? `?${query}` : ''}`, {
    headers: { Accept: 'application/json' },
  })
  if (response.status === 429) {
    const retryHeader = Number(response.headers.get('Retry-After'))
    const retryAfterSeconds = Number.isFinite(retryHeader) && retryHeader > 0 ? retryHeader : undefined
    return { status: 'rateLimited', retryAfterSeconds }
  }
  if (!response.ok) {
    return { status: 'error' }
  }
  const data = (await response.json().catch(() => undefined)) as
    | { code?: number; result?: CourseSummaryResult; data?: CourseSummaryResult }
    | undefined
  if (!data) return { status: 'error' }
  const result = (data.result ?? data.data) as CourseSummaryResult | undefined
  if (!result) return { status: 'error' }
  return result
}
