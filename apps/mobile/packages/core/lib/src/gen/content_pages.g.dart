// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'content_pages.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$LinksPagePropsImpl _$$LinksPagePropsImplFromJson(Map<String, dynamic> json) =>
    _$LinksPagePropsImpl(
      groups: (json['groups'] as List<dynamic>)
          .map((e) => LinkGroupPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      totalCount: (json['totalCount'] as num).toInt(),
    );

Map<String, dynamic> _$$LinksPagePropsImplToJson(
  _$LinksPagePropsImpl instance,
) => <String, dynamic>{
  'groups': instance.groups,
  'totalCount': instance.totalCount,
};

_$LinkGroupPayloadImpl _$$LinkGroupPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$LinkGroupPayloadImpl(
  name: json['name'] as String,
  emoji: json['emoji'] as String,
  color: json['color'] as String,
  links: (json['links'] as List<dynamic>)
      .map((e) => FriendLinkPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$$LinkGroupPayloadImplToJson(
  _$LinkGroupPayloadImpl instance,
) => <String, dynamic>{
  'name': instance.name,
  'emoji': instance.emoji,
  'color': instance.color,
  'links': instance.links,
};

_$FriendLinkPayloadImpl _$$FriendLinkPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$FriendLinkPayloadImpl(
  name: json['name'] as String,
  desc: json['desc'] as String,
  url: json['url'] as String,
  logoUrl: json['logoUrl'] as String,
);

Map<String, dynamic> _$$FriendLinkPayloadImplToJson(
  _$FriendLinkPayloadImpl instance,
) => <String, dynamic>{
  'name': instance.name,
  'desc': instance.desc,
  'url': instance.url,
  'logoUrl': instance.logoUrl,
};

_$SponsorsPagePropsImpl _$$SponsorsPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$SponsorsPagePropsImpl(
  sections: (json['sections'] as List<dynamic>)
      .map((e) => SponsorSectionPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  totalCount: (json['totalCount'] as num).toInt(),
  content: SponsorsPageIntroPayload.fromJson(
    json['content'] as Map<String, dynamic>,
  ),
  contact: SponsorsContactPayload.fromJson(
    json['contact'] as Map<String, dynamic>,
  ),
  rules: (json['rules'] as List<dynamic>)
      .map((e) => SponsorsRulePayload.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$$SponsorsPagePropsImplToJson(
  _$SponsorsPagePropsImpl instance,
) => <String, dynamic>{
  'sections': instance.sections,
  'totalCount': instance.totalCount,
  'content': instance.content,
  'contact': instance.contact,
  'rules': instance.rules,
};

_$SponsorSectionPayloadImpl _$$SponsorSectionPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$SponsorSectionPayloadImpl(
  key: json['key'] as String,
  label: json['label'] as String,
  tone: json['tone'] as String,
  sponsors: (json['sponsors'] as List<dynamic>)
      .map((e) => SponsorPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$$SponsorSectionPayloadImplToJson(
  _$SponsorSectionPayloadImpl instance,
) => <String, dynamic>{
  'key': instance.key,
  'label': instance.label,
  'tone': instance.tone,
  'sponsors': instance.sponsors,
};

_$SponsorPayloadImpl _$$SponsorPayloadImplFromJson(Map<String, dynamic> json) =>
    _$SponsorPayloadImpl(
      name: json['name'] as String,
      message: json['message'] as String,
      link: json['link'] as String,
      avatarUrl: json['avatarUrl'] as String,
    );

Map<String, dynamic> _$$SponsorPayloadImplToJson(
  _$SponsorPayloadImpl instance,
) => <String, dynamic>{
  'name': instance.name,
  'message': instance.message,
  'link': instance.link,
  'avatarUrl': instance.avatarUrl,
};

_$SponsorsPageIntroPayloadImpl _$$SponsorsPageIntroPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$SponsorsPageIntroPayloadImpl(
  title: json['title'] as String,
  description: json['description'] as String,
);

Map<String, dynamic> _$$SponsorsPageIntroPayloadImplToJson(
  _$SponsorsPageIntroPayloadImpl instance,
) => <String, dynamic>{
  'title': instance.title,
  'description': instance.description,
};

_$SponsorsContactPayloadImpl _$$SponsorsContactPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$SponsorsContactPayloadImpl(
  title: json['title'] as String,
  description: json['description'] as String,
  buttonText: json['buttonText'] as String,
  buttonLink: json['buttonLink'] as String,
);

Map<String, dynamic> _$$SponsorsContactPayloadImplToJson(
  _$SponsorsContactPayloadImpl instance,
) => <String, dynamic>{
  'title': instance.title,
  'description': instance.description,
  'buttonText': instance.buttonText,
  'buttonLink': instance.buttonLink,
};

_$SponsorsRulePayloadImpl _$$SponsorsRulePayloadImplFromJson(
  Map<String, dynamic> json,
) => _$SponsorsRulePayloadImpl(content: json['content'] as String);

Map<String, dynamic> _$$SponsorsRulePayloadImplToJson(
  _$SponsorsRulePayloadImpl instance,
) => <String, dynamic>{'content': instance.content};

_$TermsPagePropsImpl _$$TermsPagePropsImplFromJson(Map<String, dynamic> json) =>
    _$TermsPagePropsImpl(
      enabled: json['enabled'] as bool,
      contentHtml: json['contentHtml'] as String,
    );

Map<String, dynamic> _$$TermsPagePropsImplToJson(
  _$TermsPagePropsImpl instance,
) => <String, dynamic>{
  'enabled': instance.enabled,
  'contentHtml': instance.contentHtml,
};
