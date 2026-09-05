// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'layout.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$LayoutPayloadImpl _$$LayoutPayloadImplFromJson(Map<String, dynamic> json) =>
    _$LayoutPayloadImpl(
      site: SitePayload.fromJson(json['site'] as Map<String, dynamic>),
      viewer: ViewerPayload.fromJson(json['viewer'] as Map<String, dynamic>),
      header: (json['header'] as List<dynamic>?)
          ?.map((e) => NavItemPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      sidebar: SidebarPayload.fromJson(json['sidebar'] as Map<String, dynamic>),
      footer: FooterPayload.fromJson(json['footer'] as Map<String, dynamic>),
      unread: UnreadStatusPayload.fromJson(
        json['unread'] as Map<String, dynamic>,
      ),
      theme: ThemePayload.fromJson(json['theme'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$$LayoutPayloadImplToJson(_$LayoutPayloadImpl instance) =>
    <String, dynamic>{
      'site': instance.site,
      'viewer': instance.viewer,
      'header': instance.header,
      'sidebar': instance.sidebar,
      'footer': instance.footer,
      'unread': instance.unread,
      'theme': instance.theme,
    };

_$SitePayloadImpl _$$SitePayloadImplFromJson(Map<String, dynamic> json) =>
    _$SitePayloadImpl(
      name: json['name'] as String,
      description: json['description'] as String,
      logo: json['logo'] as String,
      favicon: json['favicon'] as String,
      externalLinks: json['externalLinks'] as String?,
      brandType: json['brandType'] as String,
      brandText: json['brandText'] as String,
      brandImage: json['brandImage'] as String,
    );

Map<String, dynamic> _$$SitePayloadImplToJson(_$SitePayloadImpl instance) =>
    <String, dynamic>{
      'name': instance.name,
      'description': instance.description,
      'logo': instance.logo,
      'favicon': instance.favicon,
      'externalLinks': instance.externalLinks,
      'brandType': instance.brandType,
      'brandText': instance.brandText,
      'brandImage': instance.brandImage,
    };

_$ViewerPayloadImpl _$$ViewerPayloadImplFromJson(Map<String, dynamic> json) =>
    _$ViewerPayloadImpl(
      id: (json['id'] as num).toInt(),
      username: json['username'] as String,
      email: json['email'] as String,
      avatarUrl: json['avatarUrl'] as String,
      isAuthenticated: json['isAuthenticated'] as bool,
      canAccessAdmin: json['canAccessAdmin'] as bool,
      isModerator: json['isModerator'] as bool,
      requiresEmailVerification: json['requiresEmailVerification'] as bool,
      adminPermissions: (json['adminPermissions'] as List<dynamic>?)
          ?.map((e) => (e as num).toInt())
          .toList(),
    );

Map<String, dynamic> _$$ViewerPayloadImplToJson(_$ViewerPayloadImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'username': instance.username,
      'email': instance.email,
      'avatarUrl': instance.avatarUrl,
      'isAuthenticated': instance.isAuthenticated,
      'canAccessAdmin': instance.canAccessAdmin,
      'isModerator': instance.isModerator,
      'requiresEmailVerification': instance.requiresEmailVerification,
      'adminPermissions': instance.adminPermissions,
    };

_$NavItemPayloadImpl _$$NavItemPayloadImplFromJson(Map<String, dynamic> json) =>
    _$NavItemPayloadImpl(
      key: json['key'] as String,
      label: json['label'] as String,
      i18nLabel: json['i18nLabel'] as String?,
      url: json['url'] as String,
    );

Map<String, dynamic> _$$NavItemPayloadImplToJson(
  _$NavItemPayloadImpl instance,
) => <String, dynamic>{
  'key': instance.key,
  'label': instance.label,
  'i18nLabel': instance.i18nLabel,
  'url': instance.url,
};

_$CategoryNavPayloadImpl _$$CategoryNavPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CategoryNavPayloadImpl(
  id: (json['id'] as num).toInt(),
  label: json['label'] as String,
  url: json['url'] as String,
  color: json['color'] as String,
);

Map<String, dynamic> _$$CategoryNavPayloadImplToJson(
  _$CategoryNavPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'label': instance.label,
  'url': instance.url,
  'color': instance.color,
};

_$SidebarGroupPayloadImpl _$$SidebarGroupPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$SidebarGroupPayloadImpl(
  key: json['key'] as String,
  title: json['title'] as String,
  i18nLabel: json['i18nLabel'] as String?,
  items: (json['items'] as List<dynamic>)
      .map((e) => NavItemPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$$SidebarGroupPayloadImplToJson(
  _$SidebarGroupPayloadImpl instance,
) => <String, dynamic>{
  'key': instance.key,
  'title': instance.title,
  'i18nLabel': instance.i18nLabel,
  'items': instance.items,
};

_$SidebarPayloadImpl _$$SidebarPayloadImplFromJson(Map<String, dynamic> json) =>
    _$SidebarPayloadImpl(
      main: (json['main'] as List<dynamic>?)
          ?.map((e) => NavItemPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      resources: (json['resources'] as List<dynamic>?)
          ?.map((e) => NavItemPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      groups: (json['groups'] as List<dynamic>?)
          ?.map((e) => SidebarGroupPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      categories: (json['categories'] as List<dynamic>)
          .map((e) => CategoryNavPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      activeKey: json['activeKey'] as String,
    );

Map<String, dynamic> _$$SidebarPayloadImplToJson(
  _$SidebarPayloadImpl instance,
) => <String, dynamic>{
  'main': instance.main,
  'resources': instance.resources,
  'groups': instance.groups,
  'categories': instance.categories,
  'activeKey': instance.activeKey,
};

_$FooterPayloadImpl _$$FooterPayloadImplFromJson(Map<String, dynamic> json) =>
    _$FooterPayloadImpl(
      links: (json['links'] as List<dynamic>)
          .map((e) => FooterLinkPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      primary: (json['primary'] as List<dynamic>)
          .map((e) => e as String)
          .toList(),
    );

Map<String, dynamic> _$$FooterPayloadImplToJson(_$FooterPayloadImpl instance) =>
    <String, dynamic>{'links': instance.links, 'primary': instance.primary};

_$FooterLinkPayloadImpl _$$FooterLinkPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$FooterLinkPayloadImpl(
  name: json['name'] as String,
  url: json['url'] as String,
);

Map<String, dynamic> _$$FooterLinkPayloadImplToJson(
  _$FooterLinkPayloadImpl instance,
) => <String, dynamic>{'name': instance.name, 'url': instance.url};

_$ThemePayloadImpl _$$ThemePayloadImplFromJson(Map<String, dynamic> json) =>
    _$ThemePayloadImpl(
      enabled: json['enabled'] as bool,
      href: json['href'] as String?,
      colors: (json['colors'] as Map<String, dynamic>?)?.map(
        (k, e) => MapEntry(k, e as String),
      ),
      current: json['current'] as String,
      themeColor: json['themeColor'] as String,
    );

Map<String, dynamic> _$$ThemePayloadImplToJson(_$ThemePayloadImpl instance) =>
    <String, dynamic>{
      'enabled': instance.enabled,
      'href': instance.href,
      'colors': instance.colors,
      'current': instance.current,
      'themeColor': instance.themeColor,
    };

_$UnreadStatusPayloadImpl _$$UnreadStatusPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UnreadStatusPayloadImpl(
  notifications: json['notifications'] as bool,
  messages: json['messages'] as bool,
  moderationReports: json['moderationReports'] as bool?,
  latestNotificationType: json['latestNotificationType'] as String?,
  latestUnreadId: (json['latestUnreadId'] as num?)?.toInt(),
);

Map<String, dynamic> _$$UnreadStatusPayloadImplToJson(
  _$UnreadStatusPayloadImpl instance,
) => <String, dynamic>{
  'notifications': instance.notifications,
  'messages': instance.messages,
  'moderationReports': instance.moderationReports,
  'latestNotificationType': instance.latestNotificationType,
  'latestUnreadId': instance.latestUnreadId,
};

_$ThemePreviewPropsImpl _$$ThemePreviewPropsImplFromJson(
  Map<String, dynamic> json,
) => _$ThemePreviewPropsImpl(
  theme: SiteThemeConfig.fromJson(json['theme'] as Map<String, dynamic>),
  defaults: SiteThemeConfig.fromJson(json['defaults'] as Map<String, dynamic>),
);

Map<String, dynamic> _$$ThemePreviewPropsImplToJson(
  _$ThemePreviewPropsImpl instance,
) => <String, dynamic>{'theme': instance.theme, 'defaults': instance.defaults};

_$SiteThemeConfigImpl _$$SiteThemeConfigImplFromJson(
  Map<String, dynamic> json,
) => _$SiteThemeConfigImpl(
  version: (json['version'] as num).toInt(),
  enabled: json['enabled'] as bool,
  themes: (json['themes'] as List<dynamic>)
      .map((e) => SiteThemeDefinition.fromJson(e as Map<String, dynamic>))
      .toList(),
  prepublish: json['prepublish'] == null
      ? null
      : SiteThemePrepublish.fromJson(
          json['prepublish'] as Map<String, dynamic>,
        ),
  publishedAt: json['publishedAt'] as String?,
);

Map<String, dynamic> _$$SiteThemeConfigImplToJson(
  _$SiteThemeConfigImpl instance,
) => <String, dynamic>{
  'version': instance.version,
  'enabled': instance.enabled,
  'themes': instance.themes,
  'prepublish': instance.prepublish,
  'publishedAt': instance.publishedAt,
};

_$SiteThemeDefinitionImpl _$$SiteThemeDefinitionImplFromJson(
  Map<String, dynamic> json,
) => _$SiteThemeDefinitionImpl(
  name: json['name'] as String,
  label: json['label'] as String,
  colorScheme: json['colorScheme'] as String,
  tokens: Map<String, String>.from(json['tokens'] as Map),
);

Map<String, dynamic> _$$SiteThemeDefinitionImplToJson(
  _$SiteThemeDefinitionImpl instance,
) => <String, dynamic>{
  'name': instance.name,
  'label': instance.label,
  'colorScheme': instance.colorScheme,
  'tokens': instance.tokens,
};

_$SiteThemePrepublishImpl _$$SiteThemePrepublishImplFromJson(
  Map<String, dynamic> json,
) => _$SiteThemePrepublishImpl(
  enabled: json['enabled'] as bool,
  themes: (json['themes'] as List<dynamic>)
      .map((e) => SiteThemeDefinition.fromJson(e as Map<String, dynamic>))
      .toList(),
  updatedAt: json['updatedAt'] as String?,
);

Map<String, dynamic> _$$SiteThemePrepublishImplToJson(
  _$SiteThemePrepublishImpl instance,
) => <String, dynamic>{
  'enabled': instance.enabled,
  'themes': instance.themes,
  'updatedAt': instance.updatedAt,
};
