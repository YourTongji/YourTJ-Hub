import 'package:freezed_annotation/freezed_annotation.dart';

import 'auth.dart';
import 'chat.dart';
import 'common.dart';
import 'content_pages.dart';
import 'layout.dart';
import 'moderation.dart';
import 'notification.dart';
import 'publish.dart';
import 'search.dart';
import 'topic.dart';
import 'user.dart';

part 'page.freezed.dart';
part 'page.g.dart';

/// 页面级数据通道(X-Goose-Page: true)返回的完整页面负载。
@freezed
abstract class PagePayload with _$PagePayload {
  const factory PagePayload({
    required String component,
    required Map<String, dynamic> props,
    required PageMeta meta,
    required LayoutPayload layout,
    required String url,
    required String version,
  }) = _PagePayload;

  factory PagePayload.fromJson(Map<String, dynamic> json) =>
      _$PagePayloadFromJson(json);
}

/// 页面组件注册表:component 字符串 → props 类型。
abstract final class PageComponent {
  static const home = 'home.index';
  static const topicDetail = 'topic.detail';
  static const userProfile = 'user.profile';
  static const category = 'category.index';
  static const links = 'links.index';
  static const sponsors = 'sponsors.index';
  static const notifications = 'notifications.index';
  static const drafts = 'drafts.index';
  static const terms = 'terms.index';
  static const messages = 'messages.index';
  static const moderation = 'moderation.index';
  static const settings = 'settings.index';
  static const themePreview = 'theme.preview';
  static const publish = 'publish.index';
  static const search = 'search.index';
  static const course = 'course.index';
  static const courseDetail = 'course.detail';
  static const login = 'auth.login';
  static const resetPassword = 'auth.resetPassword';
  static const error = 'error.index';
  static const admin = 'admin.shell';
}

/// 解析页面 props 为强类型;component 不在注册表或 props 缺失时返回 null。
T? parsePageProps<T>(PagePayload page) {
  final props = page.props;
  if (props.isEmpty) return null;
  try {
    return switch (page.component) {
      PageComponent.home => HomeProps.fromJson(props) as T,
      PageComponent.topicDetail => TopicDetailProps.fromJson(props) as T,
      PageComponent.userProfile => UserProfileProps.fromJson(props) as T,
      PageComponent.category => CategoryPageProps.fromJson(props) as T,
      PageComponent.links => LinksPageProps.fromJson(props) as T,
      PageComponent.sponsors => SponsorsPageProps.fromJson(props) as T,
      PageComponent.notifications =>
        NotificationsPageProps.fromJson(props) as T,
      PageComponent.drafts => DraftsPageProps.fromJson(props) as T,
      PageComponent.terms => TermsPageProps.fromJson(props) as T,
      PageComponent.messages => MessagesPageProps.fromJson(props) as T,
      PageComponent.moderation => ModerationPageProps.fromJson(props) as T,
      PageComponent.settings => SettingsPageProps.fromJson(props) as T,
      PageComponent.themePreview => ThemePreviewProps.fromJson(props) as T,
      PageComponent.publish => PublishPageProps.fromJson(props) as T,
      PageComponent.search => SearchPageProps.fromJson(props) as T,
      PageComponent.course => CourseCatalogPageProps.fromJson(props) as T,
      PageComponent.courseDetail => CourseDetailPageProps.fromJson(props) as T,
      PageComponent.login => LoginPageProps.fromJson(props) as T,
      PageComponent.resetPassword =>
        ResetPasswordPageProps.fromJson(props) as T,
      PageComponent.error => ErrorPageProps.fromJson(props) as T,
      _ => null,
    };
  } catch (_) {
    return null;
  }
}
