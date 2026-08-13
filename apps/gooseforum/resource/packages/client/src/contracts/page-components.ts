import type {
  CategoryPageProps,
  CourseCatalogPageProps,
  CourseDetailPageProps,
  CourseReviewModerationPageProps,
  DraftsPageProps,
  ErrorPageProps,
  HomeProps,
  LinksPageProps,
  LoginPageProps,
  MessagesPageProps,
  ModerationPageProps,
  NotificationsPageProps,
  PublishPageProps,
  ResetPasswordPageProps,
  SchedulePageProps,
  SearchPageProps,
  SettingsPageProps,
  SponsorsPageProps,
  TermsPageProps,
  PrivacyPageProps,
  ThemePreviewProps,
  TopicDetailProps,
  UserProfileProps,
  WikiDetailProps,
  WikiHomeProps,
  PagePayload,
} from './payload.js'

export const pageComponents = [
  'home.index',
  'topic.detail',
  'user.profile',
  'category.index',
  'links.index',
  'sponsors.index',
  'notifications.index',
  'terms.index',
  'privacy.index',
  'messages.index',
  'drafts.index',
  'moderation.index',
  'settings.index',
  'theme.preview',
  'publish.index',
  'search.index',
  'course.index',
  'course.detail',
  'course.reviewModeration',
  'course.schedule',
  'wiki.home',
  'wiki.detail',
  'auth.login',
  'auth.resetPassword',
  'error.index',
] as const

export type PageComponent = typeof pageComponents[number]

export interface PagePayloadMap {
  'home.index': HomeProps
  'topic.detail': TopicDetailProps
  'user.profile': UserProfileProps
  'category.index': CategoryPageProps
  'links.index': LinksPageProps
  'sponsors.index': SponsorsPageProps
  'notifications.index': NotificationsPageProps
  'terms.index': TermsPageProps
  'privacy.index': PrivacyPageProps
  'messages.index': MessagesPageProps
  'drafts.index': DraftsPageProps
  'moderation.index': ModerationPageProps
  'settings.index': SettingsPageProps
  'theme.preview': ThemePreviewProps
  'publish.index': PublishPageProps
  'search.index': SearchPageProps
  'course.index': CourseCatalogPageProps
  'course.detail': CourseDetailPageProps
  'course.reviewModeration': CourseReviewModerationPageProps
  'course.schedule': SchedulePageProps
  'wiki.home': WikiHomeProps
  'wiki.detail': WikiDetailProps
  'auth.login': LoginPageProps
  'auth.resetPassword': ResetPasswordPageProps
  'error.index': ErrorPageProps
}

export type PageProps<TComponent extends PageComponent> = PagePayloadMap[TComponent]

export type AnyPagePayload = {
  [TComponent in PageComponent]: PagePayload<PagePayloadMap[TComponent], TComponent>
}[PageComponent]
