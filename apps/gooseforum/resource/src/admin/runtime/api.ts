import { adminText } from '@/admin/runtime/i18n-text'
import { resolveApiMessage } from '@/runtime/api-message'
import type {
  ApiEnvelope,
  AdminAgent,
  AdminAgentCreateResult,
  AdminAgentRotateResult,
  AdminTaskRow,
  AdminTopic,
  AdminBadge,
  AdminCategory,
  AdminCategoryModerator,
  AdminFileResource,
  AdminOptRecord,
  AdminPermissionOption,
  AdminRole,
  AdminUser,
  AnnouncementConfig,
  ImportReport,
  TopicSource,
  DailyTraffic,
  FriendLinkGroup,
  GithubRelease,
  HttpNotifySettings,
  MailSettings,
  PageResult,
  PostingSettings,
  MCPSettings,
  RateLimitSettings,
  ReviewQueueItem,
  SecuritySettings,
  ServerVersion,
  SiteChromeConfig,
  SiteSettings,
  SiteStatistics,
  SponsorsConfig,
  StorageSettings,
  TermsOfServiceConfig,
  UserBadgeOptions,
} from '@/admin/types'

function responseMessage(data: ApiEnvelope<unknown>, fallback: string) {
  return resolveApiMessage(data, fallback)
}

async function readApiResponse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }
  const data = (await response.json()) as ApiEnvelope<T>
  if (data.code !== undefined && data.code !== 0) {
    throw new Error(responseMessage(data, fallback))
  }
  const result = data.result ?? data.data
  return result as T
}

async function getJson<T>(url: string, fallback: string): Promise<T> {
  const response = await fetch(url, { headers: { Accept: 'application/json' } })
  return readApiResponse<T>(response, fallback)
}

async function postJson<T>(url: string, body?: unknown, fallback = adminText('k000l')): Promise<T> {
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body ?? {}),
  })
  return readApiResponse<T>(response, fallback)
}

async function postForm<T>(url: string, body: FormData, fallback = adminText('k000l')): Promise<T> {
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
    },
    body,
  })
  return readApiResponse<T>(response, fallback)
}

async function postEnvelope<T>(url: string, body?: unknown, fallback = adminText('k000l')): Promise<ApiEnvelope<T>> {
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body ?? {}),
  })
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }
  const data = (await response.json()) as ApiEnvelope<T>
  if (data.code !== undefined && data.code !== 0) {
    throw new Error(responseMessage(data, fallback))
  }
  return data
}

export async function getSiteStatistics(): Promise<SiteStatistics> {
  const response = await fetch('/api/forum/get-site-statistics', {
    headers: { Accept: 'application/json' },
  })
  return readApiResponse<SiteStatistics>(response, adminText('k000m'))
}

export interface AdminImageUploadResult {
  url?: string
  filename?: string
  size?: number
}

export function uploadAdminImage(file: File) {
  const body = new FormData()
  body.append('file', file)
  return postForm<AdminImageUploadResult>('/api/admin/img-upload', body, adminText('k000c'))
}

export async function getTrafficOverview(startDate?: string, endDate?: string): Promise<DailyTraffic[]> {
  return postJson<DailyTraffic[]>('/api/admin/traffic-overview', { startDate, endDate }, adminText('k000n'))
}

export async function getServerVersion(): Promise<ServerVersion> {
  const response = await fetch('/api/admin/server-version', {
    headers: { Accept: 'application/json' },
  })
  return readApiResponse<ServerVersion>(response, adminText('k000o'))
}

export async function getGithubReleases(): Promise<GithubRelease[]> {
  const response = await fetch('https://api.github.com/repos/YourTongji/YourTJ-Hub/releases', {
    headers: { Accept: 'application/vnd.github+json' },
  })
  if (!response.ok) {
    throw new Error(adminText('k000p'))
  }
  return (await response.json()) as GithubRelease[]
}

export function getUserList(params: { page?: number, pageSize?: number, username?: string, userId?: number, email?: string }) {
  return postJson<PageResult<AdminUser>>('/api/admin/user-list', params, adminText('k000q'))
}

export function editUser(data: { userId: number, status: number, validate: number, roleId: number }) {
  return postJson<unknown>('/api/admin/user-edit', data, adminText('k000r'))
}

export function getAllRoleItem() {
  return getJson<{ name: string, value: number }[]>('/api/admin/get-all-role-item', adminText('k000s'))
}

export function getUserBadgeOptions(userId: number) {
  return postJson<UserBadgeOptions>('/api/admin/user-badge-options', { userId }, adminText('k000t'))
}

export function saveUserBadges(userId: number, badgeCodes: string[]) {
  return postJson<unknown>('/api/admin/save-user-badges', { userId, badgeCodes }, adminText('k000u'))
}

export function getRoleList() {
  return postJson<PageResult<AdminRole>>('/api/admin/role-list', {}, adminText('k000v'))
}

export function getPermissionList() {
  return postJson<AdminPermissionOption[]>('/api/admin/get-permission-list', {}, adminText('k000w'))
}

export function saveRole(data: { id: number, roleName: string, permissions: number[] }) {
  return postJson<unknown>('/api/admin/role-save', data, adminText('k000x'))
}

export function deleteRole(id: number) {
  return postJson<unknown>('/api/admin/role-delete', { id }, adminText('k000y'))
}

export function getCategoryList() {
  return postJson<AdminCategory[]>('/api/admin/category-list', {}, adminText('k000z'))
}

export function saveCategory(data: AdminCategory & { id: number }) {
  return postJson<unknown>('/api/admin/category-save', data, adminText('k0010'))
}

export function deleteCategory(id: number) {
  return postJson<unknown>('/api/admin/category-delete', { id }, adminText('k0011'))
}

export function getGlobalModeratorList() {
  return postJson<AdminCategoryModerator[]>('/api/admin/global-moderator-list', {}, adminText('k00f4'))
}

export function addGlobalModerator(data: { userId?: number, username?: string }) {
  return postJson<unknown>('/api/admin/global-moderator-add', data, adminText('k00el'))
}

export function deleteGlobalModerator(id: number) {
  return postJson<unknown>('/api/admin/global-moderator-delete', { id }, adminText('k00ep'))
}

export function addCategoryModerator(data: { categoryId: number, userId?: number, username?: string }) {
  return postJson<unknown>('/api/admin/category-moderator-add', data, adminText('k00en'))
}

export function deleteCategoryModerator(id: number) {
  return postJson<unknown>('/api/admin/category-moderator-delete', { id }, adminText('k00er'))
}

export function getTopicsList(params: { page?: number, pageSize?: number, search?: string }) {
  return postJson<PageResult<AdminTopic>>('/api/admin/topics/list', params, adminText('k0012'))
}

export function getOptRecordList(params: { page?: number, pageSize?: number, optUserId?: number, optType?: number, targetType?: number, targetId?: number }) {
  return postJson<PageResult<AdminOptRecord>>('/api/admin/opt-record-page', params, adminText('k0013'))
}

export function getFileResourceList(params: { page?: number, pageSize?: number }) {
  return postJson<PageResult<AdminFileResource>>('/api/admin/file-resources', params, adminText('k00fb'))
}

export function getTopicSource(id: number) {
  return postJson<TopicSource>('/api/admin/topics/source', { topicId: id }, adminText('k0014'))
}

export function editTopic(data: { topicId: number, processStatus: number }) {
  return postJson<unknown>('/api/admin/topics/edit', data, adminText('k0015'))
}

export function deleteTopic(id: number, reason: string) {
  return postJson<unknown>('/api/admin/topics/delete', { topicId: id, reason }, adminText('k00cd'))
}

export function updateTopicPin(data: { topicId: number, pinWeight: number }) {
  return postJson<unknown>('/api/admin/topics/pin-edit', data, adminText('k0016'))
}

export function updateTopicCategories(data: { topicId: number, categoryId: number[] }) {
  return postJson<unknown>('/api/admin/topics/categories-edit', data, adminText('k0017'))
}

export function getFriendLinks() {
  return getJson<FriendLinkGroup[]>('/api/admin/friend-links', adminText('k0018'))
}

export function saveFriendLinks(linksInfo: FriendLinkGroup[]) {
  return postJson<unknown>('/api/admin/save-friend-links', { linksInfo }, adminText('k0019'))
}

export function getSponsors() {
  return getJson<SponsorsConfig>('/api/admin/sponsors', adminText('k001a'))
}

export function saveSponsors(sponsorsInfo: SponsorsConfig) {
  return postJson<unknown>('/api/admin/save-sponsors', { sponsorsInfo }, adminText('k001b'))
}

export function getBadges() {
  return getJson<AdminBadge[]>('/api/admin/badges', adminText('k001c'))
}

export function saveBadge(data: AdminBadge) {
  return postJson<unknown>('/api/admin/badge-save', data, adminText('k001d'))
}

export function deleteBadge(code: string) {
  return postJson<unknown>('/api/admin/badge-delete', { code }, adminText('k001e'))
}

export function getSiteSettings() {
  return getJson<SiteSettings>('/api/admin/site-settings', adminText('k001f'))
}

export function getSiteChrome() {
  return getJson<SiteChromeConfig>('/api/admin/site-chrome', '加载布局内容失败')
}

export function getMailSettings() {
  return getJson<MailSettings>('/api/admin/mail-settings', adminText('k001g'))
}

export function getSecuritySettings() {
  return getJson<SecuritySettings>('/api/admin/security-settings', adminText('k001h'))
}

export function getPostingSettings() {
  return getJson<PostingSettings>('/api/admin/posting-settings', adminText('k001i'))
}

export function getHttpNotifySettings() {
  return getJson<HttpNotifySettings>('/api/admin/http-notify-settings', adminText('k00ch'))
}

export function getAnnouncement() {
  return getJson<AnnouncementConfig>('/api/admin/announcement', adminText('k001j'))
}

export function saveSiteSettings(settings: SiteSettings) {
  return postJson<unknown>('/api/admin/save-site-settings', { settings }, adminText('k001k'))
}

export function saveSiteChrome(settings: SiteChromeConfig) {
  return postJson<unknown>('/api/admin/save-site-chrome', { settings }, '保存布局内容失败')
}

export function saveMailSettings(settings: MailSettings) {
  return postJson<unknown>('/api/admin/save-mail-settings', { settings }, adminText('k001l'))
}

export function testMailConnection(settings: MailSettings, testEmail: string) {
  return postEnvelope<{ success?: boolean, messageCode?: string, params?: Record<string, unknown> }>('/api/admin/test-mail-connection', { settings, testEmail }, adminText('k000i'))
}

export function saveSecuritySettings(settings: SecuritySettings) {
  return postJson<unknown>('/api/admin/save-security-settings', { settings }, adminText('k001m'))
}

export function savePostingSettings(settings: PostingSettings) {
  return postJson<unknown>('/api/admin/save-posting-settings', { settings }, adminText('k001n'))
}

export function getRateLimitSettings() {
  return getJson<RateLimitSettings>('/api/admin/rate-limit-settings', adminText('k00ih'))
}

export function saveRateLimitSettings(settings: RateLimitSettings) {
  return postJson<unknown>('/api/admin/save-rate-limit-settings', { settings }, adminText('k00ii'))
}

export function getMCPSettings() {
  return getJson<MCPSettings>('/api/admin/mcp-settings', adminText('k00mh'))
}

export function saveMCPSettings(settings: MCPSettings) {
  return postJson<unknown>('/api/admin/save-mcp-settings', { settings }, adminText('k00mi'))
}

export function saveHttpNotifySettings(settings: HttpNotifySettings) {
  return postJson<unknown>('/api/admin/save-http-notify-settings', { settings }, adminText('k00ci'))
}

export function saveAnnouncement(settings: AnnouncementConfig) {
  return postJson<unknown>('/api/admin/save-announcement', { settings }, adminText('k001o'))
}

export function getStorageSettings() {
  return getJson<StorageSettings>('/api/admin/storage-settings', adminText('k00fk'))
}

export function saveStorageSettings(settings: StorageSettings) {
  return postJson<unknown>('/api/admin/save-storage-settings', { settings }, adminText('k00fk'))
}

export function testStorageConnection(settings: StorageSettings) {
  return postJson<{ success: boolean, messageCode: string, params?: { error?: string } }>(
    '/api/admin/test-storage-connection',
    { settings },
    adminText('k00fl'),
  )
}

export function createStorageMigrateTask(clearAfterMigrate: boolean) {
  return postJson<{ taskId: number }>('/api/admin/storage-migrate-task', { clearAfterMigrate }, adminText('k00fm'))
}

export function getStorageMigrateTasks() {
  return getJson<AdminTaskRow[]>('/api/admin/storage-migrate-tasks', adminText('k00fm'))
}

export function getTermsOfService() {
  return getJson<TermsOfServiceConfig>('/api/admin/terms-of-service', adminText('k00go'))
}

export function saveTermsOfService(settings: TermsOfServiceConfig) {
  return postJson<unknown>('/api/admin/save-terms-of-service', { settings }, adminText('k00go'))
}

export function getReviewQueue(kind: 'topic' | 'post', page: number, pageSize: number) {
  return postJson<{ items: ReviewQueueItem[], total: number, page: number, pageSize: number }>(
    '/api/admin/review-queue',
    { kind, page, pageSize },
    adminText('k00gd'),
  )
}

export function reviewAction(kind: 'topic' | 'post', id: number, approve: boolean) {
  return postJson<unknown>('/api/admin/review-action', { kind, id, approve }, adminText('k00gd'))
}

export function createExportTask(tables: string[], format: 'json' | 'csv') {
  return postJson<{ taskId: number }>('/api/admin/data/export', { tables, format }, adminText('k00h5'))
}

export function getExportTasks() {
  return getJson<AdminTaskRow[]>('/api/admin/data/export/tasks', adminText('k00h5'))
}

export function downloadExportTask(taskId: number) {
  window.open(`/api/admin/data/export/download/${taskId}`, '_blank')
}

export function importData(file: File) {
  const body = new FormData()
  body.append('file', file)
  return postForm<ImportReport>('/api/admin/data/import', body, adminText('k00hk'))
}

export function getAgentList() {
  return postJson<AdminAgent[]>('/api/admin/agent-list', {}, adminText('k00k2'))
}

export function createAgent(data: { username: string, nickname?: string, webhookEndpoint?: string }) {
  return postJson<AdminAgentCreateResult>('/api/admin/agent-create', data, adminText('k00k3'))
}

export function updateAgent(data: { agentId: number, nickname?: string, webhookEndpoint?: string, enabled?: number }) {
  return postJson<AdminAgent>('/api/admin/agent-update', data, adminText('k00k4'))
}

export function rotateAgentToken(agentId: number) {
  return postJson<AdminAgentRotateResult>('/api/admin/agent-rotate-token', { agentId }, adminText('k00k5'))
}

export function disableAgent(agentId: number) {
  return postJson<unknown>('/api/admin/agent-disable', { agentId }, adminText('k00k6'))
}
