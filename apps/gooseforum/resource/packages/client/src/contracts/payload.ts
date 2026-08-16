export interface PagePayload<TProps = unknown, TComponent extends string = string> {
  component: TComponent
  props: TProps
  meta: PageMeta
  layout: LayoutPayload
  url: string
  version: string
}

export interface PageMeta {
  title: string
  description?: string
  canonical?: string
  prevUrl?: string
  nextUrl?: string
  robots?: string
  openGraph?: {
    title?: string
    description?: string
    type?: string
    url?: string
    siteName?: string
    image?: string
    publishedTime?: string
    modifiedTime?: string
    author?: string
    section?: string
    tags?: string[]
  }
  twitter?: {
    card?: string
    title?: string
    description?: string
    image?: string
  }
  jsonLd?: unknown
}

export interface ErrorPageProps {
  code: string
  title: string
  messageCode?: string
  params?: Record<string, unknown>
}

export interface LoginPageProps {
  initialMode: 'login' | 'register' | 'forgot'
  redirectUrl: string
  githubUrl: string
  googleReady: boolean
}

export interface ResetPasswordPageProps {
  token: string
}

export interface LayoutPayload {
  site: SitePayload
  viewer: ViewerPayload
  header?: NavItemPayload[]
  sidebar: SidebarPayload
  footer: FooterPayload
  unread: UnreadStatusPayload
  theme: ThemePayload
}

export interface ThemePayload {
  enabled: boolean
  href?: string
  colors?: Record<string, string>
  current: 'gf-light' | 'gf-dark'
  themeColor: string
}

export const siteThemeTokenKeys = [
  'color-base-100',
  'color-base-200',
  'color-base-300',
  'color-base-content',
  'color-icon-muted',
  'color-line',
  'color-primary',
  'color-primary-content',
  'color-secondary',
  'color-secondary-content',
  'color-accent',
  'color-accent-content',
  'color-neutral',
  'color-neutral-content',
  'color-info',
  'color-info-content',
  'color-success',
  'color-success-content',
  'color-warning',
  'color-warning-content',
  'color-error',
  'color-error-content',
  'radius-selector',
  'radius-field',
  'radius-box',
  'size-selector',
  'size-field',
  'border',
  'depth',
] as const

export type SiteThemeTokenKey = typeof siteThemeTokenKeys[number]

export interface SiteThemeTokens {
  'color-base-100': string
  'color-base-200': string
  'color-base-300': string
  'color-base-content': string
  'color-icon-muted': string
  'color-line': string
  'color-primary': string
  'color-primary-content': string
  'color-secondary': string
  'color-secondary-content': string
  'color-accent': string
  'color-accent-content': string
  'color-neutral': string
  'color-neutral-content': string
  'color-info': string
  'color-info-content': string
  'color-success': string
  'color-success-content': string
  'color-warning': string
  'color-warning-content': string
  'color-error': string
  'color-error-content': string
  'radius-selector': string
  'radius-field': string
  'radius-box': string
  'size-selector': string
  'size-field': string
  border: string
  depth: string
}

export function createEmptySiteThemeTokens(): SiteThemeTokens {
  return {
    'color-base-100': '',
    'color-base-200': '',
    'color-base-300': '',
    'color-base-content': '',
    'color-icon-muted': '',
    'color-line': '',
    'color-primary': '',
    'color-primary-content': '',
    'color-secondary': '',
    'color-secondary-content': '',
    'color-accent': '',
    'color-accent-content': '',
    'color-neutral': '',
    'color-neutral-content': '',
    'color-info': '',
    'color-info-content': '',
    'color-success': '',
    'color-success-content': '',
    'color-warning': '',
    'color-warning-content': '',
    'color-error': '',
    'color-error-content': '',
    'radius-selector': '',
    'radius-field': '',
    'radius-box': '',
    'size-selector': '',
    'size-field': '',
    border: '',
    depth: '',
  }
}

export function cloneSiteThemeTokens(tokens: SiteThemeTokens): SiteThemeTokens {
  return { ...tokens }
}

export interface SiteThemeConfig {
  version: number
  enabled: boolean
  themes: SiteThemeDefinition[]
  prepublish?: SiteThemePrepublish
  publishedAt?: string
}

export interface SiteThemeDefinition {
  name: 'gf-light' | 'gf-dark'
  label: string
  colorScheme: 'light' | 'dark'
  tokens: SiteThemeTokens
}

export interface SiteThemePrepublish {
  enabled: boolean
  themes: SiteThemeDefinition[]
  updatedAt?: string
}

export interface ThemePreviewProps {
  theme: SiteThemeConfig
  defaults: SiteThemeConfig
}

export interface UnreadStatusPayload {
  notifications: boolean
  messages: boolean
  moderationReports?: boolean
  latestNotificationType?: string
}

export interface SitePayload {
  name: string
  description: string
  logo: string
  favicon: string
  externalLinks?: string
  brandType: string
  brandText: string
  brandImage: string
}

export interface ViewerPayload {
  id: number
  username: string
  email: string
  avatarUrl: string
  isAuthenticated: boolean
  canAccessAdmin: boolean
  isModerator: boolean
  requiresEmailVerification: boolean
  adminPermissions: number[]
}

export interface CategoryNavPayload {
  id: number
  label: string
  url: string
  color: string
}

export interface NavItemPayload {
  key: string
  label: string
  i18nLabel?: string
  url: string
}

export interface SidebarPayload {
  main?: NavItemPayload[]
  resources?: NavItemPayload[]
  groups?: Array<{
    key: string
    title: string
    i18nLabel?: string
    items: NavItemPayload[]
  }>
  categories: CategoryNavPayload[]
  activeKey: string
  /** 侧栏模式：wiki 模式下替换左栏为 wiki 导航树。 */
  mode?: 'forum' | 'wiki'
  /** wiki 模式下的左栏导航树（wiki 模式才填充）。 */
  wikiTree?: WikiTreeNamespace[]
}

export interface WikiTreePage {
  pageId: number
  path: string
  title: string
  active: boolean
}

export interface WikiTreeNamespace {
  name: string
  label: string
  /** 有效 URL key（slug，未分配时降级=显示名）；拼 /wiki/{slug}/{page.path} 用。 */
  slug: string
  pages: WikiTreePage[]
}

export interface FooterPayload {
  links: Array<{ name: string; url: string }>
  primary: string[]
}

export interface PaginationPayload {
  page: number
  nextPage: number
  hasNext: boolean
  nextUrl: string
}

export interface HomeProps {
  sort: string
  tabs: Array<{ key: string; label?: string; url: string; active: boolean }>
  topics: TopicPayload[]
  pagination: PaginationPayload
  announcement: {
    enabled: boolean
    html: string
    publishedAt?: string
    items?: Array<{ id: string; title: string; html: string }>
  }
}

export interface TopicDetailProps {
  topic: TopicDetailPayload
  postStream: PostWindowPayload
  hotTopics: TopicPayload[]
  permissions: {
    isOwnTopic: boolean
    canPost: boolean
    canModerateTopic: boolean
  }
}

export interface TopicDetailPayload {
  id: number
  title: string
  description: string
  url: string
  topicStatus: number
  processStatus: number
  authorDeleted: boolean
  moderatorRemoved: boolean
  author: {
    id: number
    username: string
    nickname?: string
    avatarUrl: string
    wornBadge?: UserBadgePayload | null
  }
  participants: Array<{ id: number; username: string; avatarUrl: string; wornBadge?: UserBadgePayload | null }>
  categories: Array<{ id: number; name: string; url: string; color: string }>
  replyCount: number
  maxPostNo: number
  viewCount: number
  likeCount: number
  isLiked: boolean
  isBookmarked: boolean
  isWatched: boolean
  createdAt: string
  updatedAt: string
}

export interface PostPayload {
  id: number
  topicId: number
  postNo: number
  content: string
  renderedContent: string
  processStatus: number
  isHidden: boolean
  isAuthorDeleted: boolean
  isModeratorRemoved: boolean
  canModerate: boolean
  author: {
    id: number
    username: string
    nickname?: string
    avatarUrl: string
    wornBadge?: UserBadgePayload | null
  }
  createdAt: string
  replyToPostId?: number
  replyToUserId?: number
  replyToUsername?: string
  isOwnPost: boolean
  updatedAt?: string
  lastEditor?: {
    id: number
    username: string
    nickname?: string
    avatarUrl: string
    wornBadge?: UserBadgePayload | null
  }
  lastEditedAt?: string
  revisionCount: number
  likeCount: number
  isLiked: boolean
  isBookmarked: boolean
}

export interface ReplyTargetPayload {
  id: number
  postNo?: number
  author: {
    id: number
    username: string
    nickname?: string
    avatarUrl: string
    wornBadge?: UserBadgePayload | null
  }
  renderedContent?: string
  isAuthorDeleted?: boolean
  isModeratorRemoved?: boolean
  unavailable?: boolean
}

export interface PostWindowPayload {
  posts: PostPayload[]
  replyTargets: ReplyTargetPayload[]
  anchorPostId?: number
  beforePostNo?: number
  afterPostNo?: number
  hasBefore: boolean
  hasAfter: boolean
  total: number
  maxPostNo: number
}

export interface TopicPayload {
  id: number
  title: string
  description: string
  firstImageUrl?: string
  images?: string[]
  url: string
  author: {
    id: number
    username: string
    nickname?: string
    avatarUrl: string
    wornBadge?: UserBadgePayload | null
  }
  participants: Array<{ id: number; username: string; avatarUrl: string; wornBadge?: UserBadgePayload | null }>
  categories: Array<{ id: number; name: string; url: string; color: string }>
  replyCount: number
  viewCount: number
  pinWeight: number
  processStatus: number
  activityText: string
  lastUpdateTime: string
  unseen?: boolean
}

export interface ModerationPageProps {
  categoryTabs: Array<{ key: string; label?: string; url: string; active: boolean }>
  topics: TopicPayload[]
  pagination: {
    page: number
    nextPage: number
    hasNext: boolean
    nextUrl: string
  }
}

export interface ModerationLogSubject {
  type: 'topic' | 'post' | 'category' | 'user' | 'system' | string
  id: number
  title: string
  url?: string
  excerpt?: string
}

export interface ModerationLogItem {
  id: number
  action: string
  actor: {
    id: number
    username: string
    avatarUrl: string
  }
  subject: ModerationLogSubject
  categories: Array<{ id: number; name: string; url: string; color: string }>
  messageCode: string
  params: Record<string, unknown>
  createdAt: string
}

export interface ModerationLogListResponse {
  items: ModerationLogItem[]
  nextCursor: number
  hasNext: boolean
}

export interface ModerationReportItem {
  id: number
  targetType: 'topic' | 'post'
  targetId: number
  targetUrl: string
  title: string
  excerpt: string
  reason: string
  note: string
  status: string
  resolution: string
  reporter: {
    id: number
    username: string
    avatarUrl: string
  }
  handler: {
    id: number
    username: string
    avatarUrl: string
  }
  categories: Array<{ id: number; name: string; url: string; color: string }>
  createdAt: string
  handledAt?: string
  targetDeleted?: boolean
}

export interface ModerationReportListResponse {
  items: ModerationReportItem[]
  nextCursor: number
  hasNext: boolean
}

export interface ModerationDeletedContentView {
  contentType: 'topic' | 'post'
  contentId: number
  topicId?: number
  title: string
  content: string
  authorId: number
  authorName: string
  categories: Array<{ id: number; name: string; url: string; color: string }>
  deletedBy: number
  deletedByWho: string
  deletedAt: string
  deleteReason: string
  targetUrl: string
}

export interface UserCardPayload {
  userId: number
  username: string
  nickname: string
  avatarUrl: string
  profileCoverUrl: string
  bio: string
  signature: string
  websiteName: string
  website: string
  prestige: number
  externalInformation: Record<string, { link?: string }>
  isAdmin: boolean
  topicCount: number
  replyCount: number
  likeReceivedCount: number
  likeGivenCount: number
  followerCount: number
  followingCount: number
  collectionCount: number
  isOnline: boolean
  isFollowing: boolean
  isSelf: boolean
  badges: UserBadgePayload[]
  wornBadge?: UserBadgePayload | null
  lastActiveTime: string
  createdAt: string
  isAccountClosed: boolean
}

export interface UserProfileProps {
  user: UserCardPayload
  section: 'summary' | 'activity' | 'badges' | 'bookmarks'
  activityTab: 'timeline' | 'topics' | 'likes' | 'bookmarks' | 'following' | 'followers'
  tabs: Array<{ key: string; label?: string; url: string; active: boolean }>
  activityTabs: Array<{ key: string; label?: string; url: string; active: boolean }>
  pagination: PaginationPayload
  badges: UserBadgePayload[]
  topics: TopicPayload[]
  activities: UserActivityPayload[]
  likes: UserLikePayload[]
  bookmarks: UserBookmarkPayload[]
  following: UserConnectionPayload[]
  followers: UserConnectionPayload[]
  isOwnProfile: boolean
  canMessage: boolean
  canFollow: boolean
  messageUrl: string
  settingsUrl: string
}

export interface BadgePayload {
  code: string
  type: string
  grantMode: string
  name: string
  description: string
  iconType: string
  iconKey: string
  iconUrl: string
  color: string
  level: string
  isEnabled: boolean
  isWearable: boolean
  sortOrder: number
}

export interface UserBadgePayload extends BadgePayload {
  source: string
  reason: string
  grantedAt: string
}

export interface UserActivityPayload {
  id: number
  action: number
  subjectType: string
  subjectId: number
  contentPreview: string
  url: string
  label: string
  createdAt: string
}

export interface UserLikePayload {
  id: number
  topicId: number
  title: string
  url: string
  likedAt: string
}

export interface UserBookmarkPayload {
  id: number
  type: 'topic' | 'post'
  topicId: number
  postId?: number
  postNo?: number
  title: string
  excerpt?: string
  url: string
  bookmarkedAt: string
}

export interface UserConnectionPayload {
  id: number
  username: string
  nickname: string
  avatarUrl: string
  bio: string
  url: string
}

export interface CategoryPageProps {
  category: {
    id: number
    name: string
    description: string
    icon: string
    color: string
    url: string
  }
  sort: string
  tabs: Array<{ key: string; label?: string; url: string; active: boolean }>
  topics: TopicPayload[]
  pagination: {
    page: number
    nextPage: number
    hasNext: boolean
    nextUrl: string
  }
}

export interface LinksPageProps {
  groups: LinkGroupPayload[]
  totalCount: number
}

export interface LinkGroupPayload {
  name: string
  emoji: string
  color: string
  links: FriendLinkPayload[]
}

export interface FriendLinkPayload {
  name: string
  desc: string
  url: string
  logoUrl: string
}

export interface SponsorsPageProps {
  sections: SponsorSectionPayload[]
  totalCount: number
  content: SponsorsPageIntroPayload
  contact: SponsorsContactPayload
  rules: SponsorsRulePayload[]
}

export interface TermsPageProps {
  enabled: boolean
  contentHtml: string
}

export interface PrivacyPageProps {
  enabled: boolean
  contentHtml: string
}

export interface SponsorSectionPayload {
  key: string
  label: string
  tone: string
  sponsors: SponsorPayload[]
}

export interface SponsorPayload {
  name: string
  message: string
  link: string
  avatarUrl: string
}

export interface SponsorsPageIntroPayload {
  title: string
  description: string
}

export interface SponsorsContactPayload {
  title: string
  description: string
  buttonText: string
  buttonLink: string
}

export interface SponsorsRulePayload {
  content: string
}

export interface NotificationsPageProps {
  total: number
  unreadCount: number
  notifications: NotificationPayload[]
  pagination: {
    page: number
    nextPage: number
    hasNext: boolean
    nextUrl: string
  }
}

export type NotificationFilter = 'all' | 'unread'

export interface NotificationListResponse {
  items: NotificationPayload[]
  nextCursor: number
  hasNext: boolean
  unreadCount: number
}

export type NotificationTemplateKey =
  | 'notifications.templates.comment'
  | 'notifications.templates.postReply'
  | 'notifications.templates.topicPost'
  | 'notifications.templates.follow'
  | 'notifications.templates.badge'
  | 'notifications.templates.wikiUpdated'

export interface DraftsPageProps {
  total: number
  drafts: DraftPayload[]
  pagination: {
    page: number
    nextPage: number
    hasNext: boolean
    nextUrl: string
  }
}

export interface DraftPayload {
  id: number
  title: string
  description: string
  editUrl: string
  replyCount: number
  viewCount: number
  processStatus: number
  updatedAt: string
  createdAt: string
  categories: Array<{ id: number; name: string; url: string; color: string }>
}

export interface NotificationPayload {
  id: number
  eventType: string
  isRead: boolean
  createdAt: string
  title: string
  content: string
  actor: {
    id: number
    username: string
    avatarUrl?: string
  }
  topic?: {
    id: number
    title: string
    url: string
  }
  payload: {
    title?: string
    content?: string
    templateKey?: NotificationTemplateKey
    templateParams?: NotificationTemplateParams
    actorId: number
    actorName?: string
    topicId?: number
    postId?: number
    topicTitle?: string
    metadata?: {
      followerName?: string
      badgeCode?: string
      badgeName?: string
      badgeIconUrl?: string
      profileUrl?: string
    }
  }
}

export interface NotificationTemplateParams {
  preview?: string
  followerName?: string
  badgeCode?: string
  badgeName?: string
}

export interface MessagesPageProps {
  conversations: ChatItemPayload[]
  suggestedUsers: UserConnectionPayload[]
}

export interface ChatItemPayload {
  id: number
  peerId: number
  peerUsername: string
  peerAvatar: string
  lastMsg: string
  lastMsgTime: string
  unreadCount: number
  convId: number
  peerUrl: string
}

export interface SettingsPageProps {
  user: SettingsUserPayload
  stats: {
    topicCount: number
    replyCount: number
    followerCount: number
    followingCount: number
    likeReceivedCount: number
    likeGivenCount: number
    collectionCount: number
    createdAt: string
  }
  tabs: Array<{ key: string; label?: string; url: string; active: boolean }>
}

export interface SettingsUserPayload {
  id: number
  username: string
  email: string
  nickname: string
  locale: string
  avatarUrl: string
  profileCoverUrl: string
  bio: string
  signature: string
  websiteName: string
  website: string
  prestige: number
  createdAt: string
  externalInformation: Record<string, { link?: string }>
  wornBadgeCode: string
  badges: UserBadgePayload[]
  wearableBadges: UserBadgePayload[]
  wornBadge?: UserBadgePayload | null
}

export interface PublishPageProps {
  topicId: number
  isEditing: boolean
  categories: PublishCategoryPayload[]
  topic: {
    title: string
    content: string
    categoryIds: number[]
    topicStatus: number
  }
}

export interface PublishCategoryPayload {
  id: number
  name: string
  color: string
}

export interface UserSearchPayload {
  id: number
  username: string
  nickname: string
  avatarUrl: string
  bio: string
}

export interface CategorySearchPayload {
  id: number
  name: string
  slug: string
  icon: string
  color: string
  desc: string
}

export interface SearchPageProps {
  query: string
  scope: string
  topics: TopicPayload[]
  users: UserSearchPayload[]
  categories: CategorySearchPayload[]
  courses: CourseSearchPayload[]
  total: number
  usersTotal: number
  categoriesTotal: number
  coursesTotal: number
  totalPages: number
  pagination: {
    page: number
    nextPage: number
    hasNext: boolean
    nextUrl: string
  }
  failedScopes?: string[]
  searchUnavailable?: boolean
}

export interface CourseSearchPayload {
  id: number
  primaryCode: string
  name: string
  department: string
  creditX10: number
  aliases?: string[]
  instructors?: string[]
  terms?: string[]
  campus?: string[]
  // B1 统计投影（PRD §5.1）：非 NULL 评分均分 / 可见评价数；无评分时省略。
  ratingAvg?: number
  reviewCount?: number
}

export interface CourseCatalogPageProps {
  query: {
    keyword?: string
    department?: string
    term?: string
    campus?: string
    instructor?: string
    onlyWithReviews?: boolean
    sortBy?: string
    page: number
    size: number
  }
  courses: CourseSummaryPayload[]
  pagination: {
    page: number
    nextPage: number
    hasNext: boolean
    nextUrl: string
  }
  departments: string[]
  /** 可筛选学期（value=code，label 优先学期名），按 starts_on 倒序。 */
  terms: Array<{ value: string; label: string }>
  /** 可筛选校区（course_offering.campus 原始值），按字典序。 */
  campuses: string[]
}

export interface CourseSummaryPayload {
  id: number
  primaryCode: string
  name: string
  department: string
  creditX10: number
  aliases?: string[]
  instructors?: string[]
  recentTerms?: string[]
  // B1 统计投影（PRD §5.1）：非 NULL 评分均分 / 可见评价数；无评分时省略。
  ratingAvg?: number
  reviewCount?: number
}

export interface CourseDetailPageProps {
  course: {
    id: number
    primaryCode: string
    name: string
    department: string
    creditX10: number
    aliases?: string[]
    // B1 统计投影（PRD §5.1）：均分 / 评论数 / 1-5 星各档计数（index 0 = 1 星）。
    // 无评分/无评价时省略（omitempty），前端按 undefined 降级展示。
    ratingAvg?: number
    reviewCount?: number
    ratingDistribution?: number[]
    offerings?: Array<{
      id: number
      termCode: string
      termName?: string
      campus?: string
      faculty?: string
      instructors?: string[]
      ratingAvg?: number
      reviewCount?: number
    }>
  }
}

export interface CourseReviewModerationPageProps {
  // 课评审核页数据全部走 JSON API 异步加载（见 runtime/api.ts），SSR 仅提供空壳。
}

export interface CourseManagementPageProps {
  // 课程/评价管理页数据全部走 JSON API 异步加载（见 runtime/api.ts），SSR 仅提供空壳。
}

export interface SchedulePageProps {
  // 排课器数据全部走 PK JSON API（/api/pk/*）异步加载（见 runtime/pk-api.ts），SSR 仅提供空壳。
}

// ---- wiki 分站 ----

export interface WikiNamespacePayload {
  name: string
  description: string
  pageCount: number
  updatedAt: string
  /** 有效 URL key（slug，未分配时降级=显示名）。 */
  slug: string
  /** 首个 approved 页面的完整路径（namespace/slug），供首页 namespace 卡跳转。 */
  firstPagePath?: string
}

export interface WikiRecentPagePayload {
  pageId: number
  path: string
  title: string
  updatedAt: string
  editorId: number
  editorName: string
}

export interface WikiHomeProps {
  namespaces: WikiNamespacePayload[]
  recent: WikiRecentPagePayload[]
  /** PageManager/Admin 可见「前往管理端」。 */
  canManage: boolean
}

export interface WikiTocItem {
  level: number
  id: string
  text: string
}

export interface WikiContributorPayload {
  userId: number
  username: string
  avatarUrl: string
  count: number
  lastEditedAt: string
}

export interface WikiPageDetailPayload {
  id: number
  topicId: number
  namespace: string
  path: string
  title: string
  /** 服务端 goldmark 输出的渲染 HTML。 */
  content: string
  toc: WikiTocItem[]
  updatedAt: string
  editorId: number
  editorName: string
  likeCount: number
  viewCount: number
  postCount: number
  liked: boolean
  bookmarked: boolean
  watched: boolean
  canEdit: boolean
  publishedRevisionNo: number
  /** GitHub SSOT：仓库编辑外链（{repo}/edit/{branch}/{path}.md；未配置时为空）。 */
  editUrl?: string
  /** GitHub SSOT：仓库历史外链（{repo}/commits/{branch}/{path}.md；未配置时为空）。 */
  historyUrl?: string
}

export interface WikiDetailProps {
  page: WikiPageDetailPayload
  contributors: WikiContributorPayload[]
  /** 复用现有 TopicPayload 类型（TopicPage 的 hotTopics 同型）。 */
  hotTopics: TopicPayload[]
}
