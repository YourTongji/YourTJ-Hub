import type { LayoutPayload } from '@gooseforum/client'

export interface AdminPayload<TProps = unknown> {
  component: string
  props: TProps
  meta: {
    title: string
    robots?: string
  }
  layout: LayoutPayload
  url: string
  version: string
}

export type ManageHomeProps = Record<string, never>

export interface ApiEnvelope<T> {
  code?: number
  messageCode?: string
  params?: Record<string, unknown>
  result?: T
  data?: T
}

export interface PageResult<T> {
  list: T[]
  total: number
  hasNext?: boolean
  page: number
  pageSize?: number
  size?: number
}

export interface AdminCategory {
  id: number
  category: string
  desc?: string
  icon?: string
  color?: string
  slug?: string
  sort?: number
  moderators?: AdminCategoryModerator[]
}

export interface AdminCategoryModerator {
  id: number
  userId: number
  username: string
  /** 当前昵称；无昵称时缺省（备注名显示 note(display name) 用）。 */
  nickname?: string
  avatarUrl?: string
  status: number
}

export interface AdminUser {
  userId: number
  username: string
  /** 当前昵称；无昵称时缺省（备注名显示 note(display name) 用）。 */
  nickname?: string
  avatarUrl?: string | null
  email: string
  status: number
  validate: number
  actorType: number
  prestige: number
  roleId?: number | null
  roleList?: { name: string, value: number }[] | null
  createTime: string
  lastActiveTime?: string | null
  badges?: UserBadge[]
}

export interface UserBadge extends AdminBadge {
  source?: string
  reason?: string
  grantedAt?: string
}

export interface UserBadgeOptions {
  options: AdminBadge[]
  active: UserBadge[]
}

export interface AdminRole {
  roleId: number
  roleName: string
  effective: number
  permissions: { id: number, name: string }[]
  createTime: string
}

export interface AdminPermissionOption {
  name: string
  label?: string
  value: number
}

export interface AdminTopic {
  id: number
  title: string
  description?: string | null
  categoryId: number[]
  userId: number
  username: string
  /** 作者当前昵称；无昵称时缺省（备注名显示 note(display name) 用）。 */
  nickname?: string
  userAvatarUrl?: string | null
  topicStatus: number
  processStatus: number
  viewCount: number
  replyCount: number
  likeCount: number
  pinWeight: number
  /** 该主题是否禁止 Agent 评论（Agent 评论策略面板）。 */
  agentCommentDisabled: boolean
  createdAt: string
  updatedAt?: string
}

export interface AdminAgentCommentPolicy {
  allowAgentComments: boolean
}

export interface AdminAgentCommentTopicPolicy {
  topicId: number
  agentCommentDisabled: boolean
}

export interface AdminOptRecord {
  id: number
  optUserId: number
  optType: number
  targetType: number
  targetId: string
  optInfo: string
  optInfoPayload?: {
    messageCode?: string
    params?: Record<string, unknown>
  }
  createdAt: string
}

export interface AdminFileResource {
  id: number
  name: string
  type: string
  size: number
  userId: number
  uploaderUsername?: string
  createdAt: string
  url: string
}

export interface TopicSource extends AdminTopic {
  content: string
}

export interface AdminBadge {
  code: string
  type: 'system' | 'custom' | string
  grantMode: 'auto' | 'manual' | string
  name: string
  description: string
  iconType?: string
  iconKey?: string
  iconUrl: string
  color: string
  level: string
  isEnabled: boolean
  isWearable: boolean
  sortOrder: number
  isSystem?: boolean
  canDelete?: boolean
}

export interface AdminSticker {
  id: number
  name: string
  fileName: string
  url: string
  sortOrder: number
  isEnabled: boolean
  createdBy: number
}

export interface StickerImportIssue {
  name: string
  reason: string
}

export interface StickerImportResult {
  imported: number
  skipped: number
  failed: StickerImportIssue[]
}

export interface FriendLink {
  name: string
  url: string
  desc?: string
  logoUrl?: string
  status?: number
}

export interface FriendLinkGroup {
  name: string
  emoji?: string
  color?: string
  links: FriendLink[]
}

export interface SponsorItem {
  name: string
  avatarUrl: string
  message: string
  link: string
}

export interface SponsorsConfig {
  sponsors: Record<'level0' | 'level1' | 'level2' | 'level3', SponsorItem[]>
  content: { title: string, description: string }
  contact: { title: string, description: string, buttonText: string, buttonLink: string }
  rules: { content: string }[]
}

export interface SiteSettings {
  siteName: string
  siteUrl: string
  siteLogo: string
  siteEmail: string
  siteDescription: string
  siteKeywords: string
  externalLinks?: string
}

export interface SiteChromeItem {
  id: string
  enabled: boolean
  type: 'link' | 'text' | string
  label: string
  i18nLabel: string
  url: string
}

export interface SiteChromeGroup {
  id: string
  title: string
  i18nLabel: string
  items: SiteChromeItem[]
}

export interface SiteChromeConfig {
  header: SiteChromeItem[]
  mainMenu: SiteChromeItem[]
  resources: SiteChromeItem[]
  sidebarGroups: SiteChromeGroup[]
  footerInfo?: {
    primary: { content: string }[]
    list: { name: string, url: string }[]
  }
  brandType?: string
  brandText?: string
  brandImage?: string
}

export interface MailSettings {
  enableMail: boolean
  smtpHost: string
  smtpPort: number
  useSSL: boolean
  smtpUsername: string
  smtpPassword: string
  fromName: string
  fromEmail: string
  /** GET 回显（issue #324 S2）：密码是否已配置（服务端加密存储，不回显密码）。 */
  smtpPasswordConfigured?: boolean
}

export interface SecuritySettings {
  enableSignup: boolean
  enableEmailVerification: boolean
  maxDailySignups: number
  allowedDomains: string[]
  reservedUsernames: string[]
  bannedUsernames: string[]
  sensitiveWords: string[]
  sensitiveAction: 'block' | 'review'
  captchaRequired: boolean
}

export interface RateLimitRule {
  action: string
  windowSeconds: number
  limitPerIp: number
  limitPerUser: number
}

export interface RateLimitSettings {
  enabled: boolean
  skipAdmin: boolean
  actions: RateLimitRule[]
  newUserCaptchaAfterPosts: number
  newUserCaptchaDays: number
  minSubmitSeconds: number
}

export interface MCPSettings {
  enabled: boolean
  writes: boolean
}

export interface AiSummarySettings {
  enabled: boolean
  globalPerMinute: number
  /** OpenAI-compatible 端点，如 https://api.openai.com/v1 */
  baseUrl: string
  /** 模型 ID，如 gpt-4o */
  model: string
  /** 保存请求携带的明文 apiKey（留空 = 保留已存密钥）；GET 回显恒为空 */
  apiKey: string
  /** GET 回显（issue #324 安全模式）：apiKey 是否已配置（服务端加密存储，不回显密钥） */
  apiKeyConfigured?: boolean
  temperature?: number
  maxTokens?: number
}

/** /models 端点返回的模型条目（OpenAI compatible）。 */
export interface AiSummaryModelItem {
  id: string
  owned_by: string
}

export interface OnesystemSettings {
  cookieConfigured: boolean
  cookieConfiguredUndergraduate: boolean
  cookieConfiguredGraduate: boolean
  xTokenConfiguredGraduate: boolean
}

/** 单个学期的排课数据同步状态（issue #248 管理端同步入口）。 */
export interface PkSyncStatusItem {
  audience: 'undergraduate' | 'graduate'
  calendarId: number
  calendarName: string
  status: string
  rowsWritten: number
  totalPages: number
  lastCommittedPage: number
  errorMsg: string
  startedAt?: string | null
  finishedAt?: string | null
}

/** 排课数据定时同步配置（issue #569 管理端「定时同步」）。 */
export interface PkSyncScheduleSettings {
  /** 定时同步总开关；关闭时不注册 cron，也不执行。 */
  enabled: boolean
  /** 5 段标准 cron 表达式（分 时 日 月 周），如 "30 2 * * *"。 */
  schedule: string
  /** 目标学期（数字 calendarId / 学期名）；空 = 最近已同步学期。 */
  term: string
  /** 以目标学期为终点向前同步的连续学期数（1..8）。 */
  depth: number
  /** 数据来源：undergraduate | graduate。 */
  audience: 'undergraduate' | 'graduate'
}

/** 一系统凭证校验结果（issue #856 管理端保存前探测）。 */
export interface PkValidateCredentialResult {
  valid: boolean
  message: string
}

/** 排课器节次作息：单节开始/结束时间（HH:MM）。 */
export interface ScheduleSectionTime {
  section: number
  start: string
  end: string
}

/** 排课器节次作息设置（控制 /schedule 课表左侧的节次时间展示）。 */
export interface ScheduleSettings {
  sectionTimes: ScheduleSectionTime[]
}

export interface StorageSettings {
  provider: 'local' | 's3'
  endpoint: string
  internalEndpoint: string
  bucket: string
  region: string
  bucketLookup: 'auto' | 'dns' | 'path'
  secure: boolean
  accessKey: string
  secretKey: string
  publicUrlPrefix: string
  /** GET 回显（issue #324 S3）：凭据是否已配置（服务端加密存储，不回显凭据）。 */
  accessKeyConfigured?: boolean
  secretKeyConfigured?: boolean
}

export interface TermsOfServiceConfig {
  enabled: boolean
  content: string
}

export interface PrivacyPolicyConfig {
  enabled: boolean
  content: string
}

export interface AdminTaskRow {
  id: number
  type: string
  status: number
  taskJson: string
  retryCount: number
  lastError: string
  processedAt: string
  createdAt: string
}

export interface ReviewQueueItem {
  revisionId?: number
  content?: string
  reviewReason?: string
  id: number
  title: string
  excerpt: string
  userId: number
  username: string
  /** 作者当前昵称；无昵称时缺省（备注名显示 note(display name) 用）。 */
  nickname?: string
  processStatus: number
  createdAt: string
  topicId?: number
  postNo?: number
  /** 待审内容引用的图片（≤9）；待审图片经 /file/img 授权预览读取。 */
  images?: string[]
  /** 仅当本条因 AI 图文审查转入待审时返回（issue #975）。 */
  aiReview?: AiModerationDecision
  /** 发布后检查模式下正在后台自动检查（通常很快自动公开或拒绝）。 */
  aiChecking?: boolean
}

export type AiModerationPolicyKey = 'adult' | 'political_sensitive' | 'violence' | 'illegal_or_dangerous' | 'other'

export interface AiModerationPolicyRule {
  key: AiModerationPolicyKey
  label: string
  /** 站点规则原文（管理员维护，写入 Jev state；模型只执行，不自创规则）。 */
  definition: string
  enabled: boolean
  /** 该规则可触发的最高动作：review 规则永不自动拦截。 */
  action: 'review' | 'block'
  reviewThreshold?: number
  blockThreshold?: number
}

export interface AiModerationOptions {
  enabled: boolean
  mode: 'shadow' | 'enforce' | 'deferred'
  textModeration: boolean
  jevEndpoint: string
  jevModel: string
  jevTimeoutMs: number
  jevRetries: number
  visionBaseUrl: string
  visionModel: string
  visionTimeoutMs: number
  policyRevision: string
  policies: AiModerationPolicyRule[]
  defaultReviewThreshold: number
  defaultBlockThreshold: number
  reviewNeededThreshold: number
  severityBlockThreshold: number
  externalImageAction: 'review' | 'block'
  maxImagesPerDecision: number
  globalRequestsPerMinute: number
  perUserRequestsPerMinute: number
}

/** GET 回显：密钥只回显是否已配置（issue #324 安全模式）。 */
export interface AiModerationSettingsView extends AiModerationOptions {
  jevApiKeyConfigured: boolean
  visionApiKeyConfigured: boolean
}

/** 保存负载：密钥明文仅在请求瞬间存在；空串保留已存密钥，clear* 显式清除。 */
export interface AiModerationSettingsInput extends AiModerationOptions {
  jevApiKey?: string
  visionApiKey?: string
  clearJevApiKey?: boolean
  clearVisionApiKey?: boolean
}

export interface AiModerationImageRecord {
  fileName?: string
  url?: string
  sha256?: string
  status: string
  evidence?: string
}

export interface AiModerationReason {
  code: 'block_threshold' | 'severity_escalation' | 'between_thresholds' | 'review_only_rule' | 'review_needed' | 'evidence_incomplete' | 'external_image_blocked'
  policy?: string
  score?: number
  threshold?: number
  limit?: number
  severity?: number
  detail?: string
}

export interface AiModerationDecision {
  id: number
  subjectType: 'topic' | 'post'
  subjectId: number
  authorId: number
  mode: 'shadow' | 'enforce' | 'deferred'
  policyRevision: string
  visionModel: string
  jevModel: string
  images: AiModerationImageRecord[]
  signals: { ruleProbabilities?: Record<string, number>, severity?: number, reviewNeeded?: number }
  triggeredPolicies: string[]
  /** 结论原因（旧记录可能为空）。 */
  reasons?: AiModerationReason[]
  evidenceStatus: string
  finalAction: 'allow' | 'review' | 'block'
  appliedAction: string
  errorKind: string
  humanAction: '' | 'approved' | 'rejected'
  latencyMs: number
  cost: number
  createdAt: string
}

/** 测试连接结果：只含分类/状态码/耗时，不回带 provider 响应原文。 */
export interface AiModerationConnectionCheck {
  ok: boolean
  kind: string
  httpStatus?: number
  model?: string
  latencyMs: number
}

export interface AiModerationReplayReport {
  samples: number
  matrix: Record<'approved' | 'rejected', Record<'allow' | 'review' | 'block', number>>
  falseBlock: number
  missedViolation: number
  reviewRate: number
  changed: number
}

export interface ImportReport {
  taskId: number
  status: 'pending' | 'running' | 'retrying' | 'success' | 'failed'
  errors: Array<{ line: number; table: string; reason: string }>
  importedTables: string[]
}

export interface PostingSettings {
  textControl: {
    minPostLength: number
    maxPostLength: number
    minTitleLength: number
    maxTitleLength: number
    newUserPostCooldownMinutes: number
    /** 每用户每日新主题上限（0 = 不限额，负值由服务端归一为 0；仅约束新建主题）。 */
    maxDailyTopicsPerUser: number
  }
  uploadControl: {
    allowAttachments: boolean
    authorizedExtensions: string[]
    maxAttachmentSizeKb: number
    maxDailyUploadsPerUser: number
    newUserUploadCooldownMinutes: number
  }
  llms: {
    enabled: boolean
    fullText: boolean
    files: boolean
  }
}

/** 通道类型（issue #1049）：generic 原样 JSON；feishu 飞书自定义机器人审批卡片。 */
export type HttpNotifyChannelType = 'generic' | 'feishu' | 'astrbot'

export interface HttpNotifyEndpoint {
  id: string
  name: string
  channelType: HttpNotifyChannelType
  enabled: boolean
  /** 飞书 webhook 地址按凭据加密存储，GET 恒为空；留空保存保留已配置地址。 */
  url: string
  /** 通道内的接收目标：AstrBot 为会话 umo（/sid 显示的 SID），其他通道为空。 */
  target?: string
  secret: string
  events: string[]
  timeoutSeconds: number
  failureCount: number
  lastError: string
  abnormalTerminated: boolean
  /** GET 回显（issue #324 S1）：端点密钥是否已配置（服务端加密存储，不回显密钥）。 */
  secretConfigured?: boolean
  /** GET 回显（issue #1049）：webhook 地址是否已配置（飞书地址不回显明文）。 */
  urlConfigured?: boolean
}

export interface HttpNotifySettings {
  enabled: boolean
  endpoints: HttpNotifyEndpoint[]
}

export interface AnnouncementConfig {
  enabled: boolean
  content: string
  publishedAt?: string
  items?: AnnouncementItemConfig[]
}

export interface AnnouncementItemConfig {
  id: string
  title: string
  content: string
  enabled: boolean
}

export interface SiteStatistics {
  userCount: number
  userMonthCount: number
  topicMaxId: number
  topicMonthCount: number
  postMaxId: number
  linksCount: number
}

export interface DailyTraffic {
  date: string
  regCount: number
  topicCount: number
  replyCount: number
  courseReviewCount: number
}

export interface ServerVersion {
  version: string
  commit: string
  buildDate: string
  mode: 'development' | 'snapshot' | 'release' | 'custom' | string
}

export interface GithubRelease {
  id: number
  tag_name: string
  published_at: string
  body: string
  html_url: string
  prerelease: boolean
  draft: boolean
}

export interface AdminAgent {
  agentId: number
  username: string
  nickname: string
  avatarUrl: string
  email: string
  tokenPrefix: string
  webhookEndpoint: string
  configVersion: number
  eventsEnabled: boolean
  eventTypes: string[]
  webhookEnabled: boolean
  endpointGeneration: number
  subscriptionGeneration: number
  secretConfigured: boolean
  secretVersion: number
  latestAcceptedAt: string | null
  pendingCount: number
  pauseReason: string
  summaryUnavailable: boolean
  enabled: number
  createdBy: number
  lastUsedAt?: number | null
  createdAt: number
  updatedAt: number
}

export interface AdminAgentWebhookSecretResult {
  secret: string
  secretVersion: number
  configVersion: number
}

export interface AdminAgentWebhookAttempt {
  id: string
  deliveryId: number
  round: number
  number: number
  httpStatus?: number | null
  errorClass?: string | null
  durationMs?: number | null
  authorizedAt?: string | null
  completedAt?: string | null
}

export interface AdminAgentWebhookDelivery {
  id: number
  instanceId: string
  eventId: string
  agentId: number
  endpointGeneration: number
  schemaVersion: number
  status: string
  reason?: string | null
  taskId?: number | null
  round: number
  attemptCount: number
  totalAttempts: number
  deadline?: string | null
  expiresAt?: string | null
  nextRunAt?: string | null
  createdBy?: number | null
  lastRedeliveredBy?: number | null
  acceptedAt?: string | null
  createdAt: string
  updatedAt: string
  attempts: AdminAgentWebhookAttempt[]
}

export interface AdminAgentWebhookDeliveryPage {
  list: AdminAgentWebhookDelivery[]
  total: number
  page: number
  pageSize: number
}

export interface AdminAgentInteractionIntent {
  id: string | number
  agentId?: number
  sourceOccurrenceId?: string
  postId?: number
  revision?: number
  status: string
  createdAt: string
  expiresAt?: string | null
  retryCount?: number
  lastError?: string | null
  taskId?: number | null
}

export interface AdminAgentInteractionIntentPage {
  list: AdminAgentInteractionIntent[]
  total: number
  page: number
  pageSize: number
}

export interface AdminAgentCreateResult {
  agent: AdminAgent
  token: string
}

export interface AdminAgentRotateResult {
  agentId: number
  token: string
}

export interface WikiNamespace {
  name: string
  description: string
  sortOrder: number
  pageCount: number
  updatedAt: string
}

export interface WikiTreeNode {
  kind: 'page' | 'directory'
  pageId: number
  path: string
  sourcePath: string
  title: string
  sortOrder: number
  children: WikiTreeNode[]
}

export interface WikiNamespaceTree {
  name: string
  label: string
  nodes: WikiTreeNode[]
}

/** Completed local course materialization (the API returns only after commit). */
export type PkMaterializeResult = import('@gooseforum/client/openapi').components['schemas']['PkMaterializeResult']
