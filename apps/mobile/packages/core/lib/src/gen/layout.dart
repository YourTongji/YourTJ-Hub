import 'package:freezed_annotation/freezed_annotation.dart';

part 'layout.freezed.dart';
part 'layout.g.dart';

@freezed
abstract class LayoutPayload with _$LayoutPayload {
  const factory LayoutPayload({
    required SitePayload site,
    required ViewerPayload viewer,
    List<NavItemPayload>? header,
    required SidebarPayload sidebar,
    required FooterPayload footer,
    required UnreadStatusPayload unread,
    required ThemePayload theme,
  }) = _LayoutPayload;

  factory LayoutPayload.fromJson(Map<String, dynamic> json) => _$LayoutPayloadFromJson(json);
}

@freezed
abstract class SitePayload with _$SitePayload {
  const factory SitePayload({
    required String name,
    required String description,
    required String logo,
    required String favicon,
    String? externalLinks,
    required String brandType,
    required String brandText,
    required String brandImage,
  }) = _SitePayload;

  factory SitePayload.fromJson(Map<String, dynamic> json) => _$SitePayloadFromJson(json);
}

@freezed
abstract class ViewerPayload with _$ViewerPayload {
  const factory ViewerPayload({
    required int id,
    required String username,
    required String email,
    required String avatarUrl,
    required bool isAuthenticated,
    required bool canAccessAdmin,
    required bool isModerator,
    required bool requiresEmailVerification,
    List<int>? adminPermissions,
  }) = _ViewerPayload;

  factory ViewerPayload.fromJson(Map<String, dynamic> json) => _$ViewerPayloadFromJson(json);
}

@freezed
abstract class NavItemPayload with _$NavItemPayload {
  const factory NavItemPayload({
    required String key,
    required String label,
    String? i18nLabel,
    required String url,
  }) = _NavItemPayload;

  factory NavItemPayload.fromJson(Map<String, dynamic> json) => _$NavItemPayloadFromJson(json);
}

@freezed
abstract class CategoryNavPayload with _$CategoryNavPayload {
  const factory CategoryNavPayload({
    required int id,
    required String label,
    required String url,
    required String color,
  }) = _CategoryNavPayload;

  factory CategoryNavPayload.fromJson(Map<String, dynamic> json) =>
      _$CategoryNavPayloadFromJson(json);
}

@freezed
abstract class SidebarGroupPayload with _$SidebarGroupPayload {
  const factory SidebarGroupPayload({
    required String key,
    required String title,
    String? i18nLabel,
    required List<NavItemPayload> items,
  }) = _SidebarGroupPayload;

  factory SidebarGroupPayload.fromJson(Map<String, dynamic> json) =>
      _$SidebarGroupPayloadFromJson(json);
}

@freezed
abstract class SidebarPayload with _$SidebarPayload {
  const factory SidebarPayload({
    List<NavItemPayload>? main,
    List<NavItemPayload>? resources,
    List<SidebarGroupPayload>? groups,
    required List<CategoryNavPayload> categories,
    required String activeKey,
  }) = _SidebarPayload;

  factory SidebarPayload.fromJson(Map<String, dynamic> json) => _$SidebarPayloadFromJson(json);
}

@freezed
abstract class FooterPayload with _$FooterPayload {
  const factory FooterPayload({
    required List<FooterLinkPayload> links,
    required List<String> primary,
  }) = _FooterPayload;

  factory FooterPayload.fromJson(Map<String, dynamic> json) => _$FooterPayloadFromJson(json);
}

@freezed
abstract class FooterLinkPayload with _$FooterLinkPayload {
  const factory FooterLinkPayload({
    required String name,
    required String url,
  }) = _FooterLinkPayload;

  factory FooterLinkPayload.fromJson(Map<String, dynamic> json) =>
      _$FooterLinkPayloadFromJson(json);
}

@freezed
abstract class ThemePayload with _$ThemePayload {
  const factory ThemePayload({
    required bool enabled,
    String? href,
    Map<String, String>? colors,
    required String current,
    required String themeColor,
  }) = _ThemePayload;

  factory ThemePayload.fromJson(Map<String, dynamic> json) => _$ThemePayloadFromJson(json);
}

@freezed
abstract class UnreadStatusPayload with _$UnreadStatusPayload {
  const factory UnreadStatusPayload({
    required bool notifications,
    required bool messages,
    bool? moderationReports,
    String? latestNotificationType,
  }) = _UnreadStatusPayload;

  factory UnreadStatusPayload.fromJson(Map<String, dynamic> json) =>
      _$UnreadStatusPayloadFromJson(json);
}

@freezed
abstract class ThemePreviewProps with _$ThemePreviewProps {
  const factory ThemePreviewProps({
    required SiteThemeConfig theme,
    required SiteThemeConfig defaults,
  }) = _ThemePreviewProps;

  factory ThemePreviewProps.fromJson(Map<String, dynamic> json) =>
      _$ThemePreviewPropsFromJson(json);
}

@freezed
abstract class SiteThemeConfig with _$SiteThemeConfig {
  const factory SiteThemeConfig({
    required int version,
    required bool enabled,
    required List<SiteThemeDefinition> themes,
    SiteThemePrepublish? prepublish,
    String? publishedAt,
  }) = _SiteThemeConfig;

  factory SiteThemeConfig.fromJson(Map<String, dynamic> json) =>
      _$SiteThemeConfigFromJson(json);
}

@freezed
abstract class SiteThemeDefinition with _$SiteThemeDefinition {
  const factory SiteThemeDefinition({
    required String name,
    required String label,
    required String colorScheme,
    required Map<String, String> tokens,
  }) = _SiteThemeDefinition;

  factory SiteThemeDefinition.fromJson(Map<String, dynamic> json) =>
      _$SiteThemeDefinitionFromJson(json);
}

@freezed
abstract class SiteThemePrepublish with _$SiteThemePrepublish {
  const factory SiteThemePrepublish({
    required bool enabled,
    required List<SiteThemeDefinition> themes,
    String? updatedAt,
  }) = _SiteThemePrepublish;

  factory SiteThemePrepublish.fromJson(Map<String, dynamic> json) =>
      _$SiteThemePrepublishFromJson(json);
}
