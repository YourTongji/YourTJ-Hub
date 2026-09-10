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

_$CourseSummaryPayloadImpl _$$CourseSummaryPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CourseSummaryPayloadImpl(
  id: (json['id'] as num).toInt(),
  primaryCode: json['primaryCode'] as String,
  name: json['name'] as String,
  department: json['department'] as String,
  creditX10: (json['creditX10'] as num).toInt(),
  teacherId: (json['teacherId'] as num?)?.toInt(),
  teacherName: json['teacherName'] as String?,
  aliases: (json['aliases'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  instructors: (json['instructors'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  recentTerms: (json['recentTerms'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  ratingAvg: (json['ratingAvg'] as num?)?.toDouble(),
  reviewCount: (json['reviewCount'] as num?)?.toInt(),
);

Map<String, dynamic> _$$CourseSummaryPayloadImplToJson(
  _$CourseSummaryPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'primaryCode': instance.primaryCode,
  'name': instance.name,
  'department': instance.department,
  'creditX10': instance.creditX10,
  'teacherId': instance.teacherId,
  'teacherName': instance.teacherName,
  'aliases': instance.aliases,
  'instructors': instance.instructors,
  'recentTerms': instance.recentTerms,
  'ratingAvg': instance.ratingAvg,
  'reviewCount': instance.reviewCount,
};

_$CourseCatalogPagePropsImpl _$$CourseCatalogPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$CourseCatalogPagePropsImpl(
  query: CourseCatalogQueryPayload.fromJson(
    json['query'] as Map<String, dynamic>,
  ),
  courses: (json['courses'] as List<dynamic>)
      .map((e) => CourseSummaryPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  pagination: PaginationPayload.fromJson(
    json['pagination'] as Map<String, dynamic>,
  ),
  departments: (json['departments'] as List<dynamic>)
      .map((e) => e as String)
      .toList(),
);

Map<String, dynamic> _$$CourseCatalogPagePropsImplToJson(
  _$CourseCatalogPagePropsImpl instance,
) => <String, dynamic>{
  'query': instance.query,
  'courses': instance.courses,
  'pagination': instance.pagination,
  'departments': instance.departments,
};

_$CourseCatalogQueryPayloadImpl _$$CourseCatalogQueryPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CourseCatalogQueryPayloadImpl(
  keyword: json['keyword'] as String?,
  department: (json['department'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  term: (json['term'] as List<dynamic>?)?.map((e) => e as String).toList(),
  campus: (json['campus'] as List<dynamic>?)?.map((e) => e as String).toList(),
  instructor: (json['instructor'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  onlyWithReviews: json['onlyWithReviews'] as bool?,
  sortBy: json['sortBy'] as String?,
  page: (json['page'] as num).toInt(),
  size: (json['size'] as num).toInt(),
);

Map<String, dynamic> _$$CourseCatalogQueryPayloadImplToJson(
  _$CourseCatalogQueryPayloadImpl instance,
) => <String, dynamic>{
  'keyword': instance.keyword,
  'department': instance.department,
  'term': instance.term,
  'campus': instance.campus,
  'instructor': instance.instructor,
  'onlyWithReviews': instance.onlyWithReviews,
  'sortBy': instance.sortBy,
  'page': instance.page,
  'size': instance.size,
};

_$CourseOfferingPayloadImpl _$$CourseOfferingPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CourseOfferingPayloadImpl(
  id: (json['id'] as num).toInt(),
  termCode: json['termCode'] as String,
  termName: json['termName'] as String?,
  campus: json['campus'] as String?,
  faculty: json['faculty'] as String?,
  classCode: json['classCode'] as String?,
  className: json['className'] as String?,
  instructors: (json['instructors'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  ratingAvg: (json['ratingAvg'] as num?)?.toDouble(),
  reviewCount: (json['reviewCount'] as num?)?.toInt(),
);

Map<String, dynamic> _$$CourseOfferingPayloadImplToJson(
  _$CourseOfferingPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'termCode': instance.termCode,
  'termName': instance.termName,
  'campus': instance.campus,
  'faculty': instance.faculty,
  'classCode': instance.classCode,
  'className': instance.className,
  'instructors': instance.instructors,
  'ratingAvg': instance.ratingAvg,
  'reviewCount': instance.reviewCount,
};

_$CourseDetailPagePropsImpl _$$CourseDetailPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$CourseDetailPagePropsImpl(
  course: CourseDetailPayload.fromJson(json['course'] as Map<String, dynamic>),
);

Map<String, dynamic> _$$CourseDetailPagePropsImplToJson(
  _$CourseDetailPagePropsImpl instance,
) => <String, dynamic>{'course': instance.course};

_$CourseDetailPayloadImpl _$$CourseDetailPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CourseDetailPayloadImpl(
  id: (json['id'] as num).toInt(),
  primaryCode: json['primaryCode'] as String,
  name: json['name'] as String,
  department: json['department'] as String,
  creditX10: (json['creditX10'] as num).toInt(),
  teacherId: (json['teacherId'] as num?)?.toInt(),
  teacherName: json['teacherName'] as String?,
  aliases: (json['aliases'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  offerings: (json['offerings'] as List<dynamic>?)
      ?.map((e) => CourseOfferingPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  ratingAvg: (json['ratingAvg'] as num?)?.toDouble(),
  reviewCount: (json['reviewCount'] as num?)?.toInt(),
  ratingDistribution: (json['ratingDistribution'] as List<dynamic>?)
      ?.map((e) => (e as num).toInt())
      .toList(),
  reviewScope: json['reviewScope'] as String?,
  teamKey: json['teamKey'] as String?,
  teamInstructors: (json['teamInstructors'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  legacyNames: (json['legacyNames'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
);

Map<String, dynamic> _$$CourseDetailPayloadImplToJson(
  _$CourseDetailPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'primaryCode': instance.primaryCode,
  'name': instance.name,
  'department': instance.department,
  'creditX10': instance.creditX10,
  'teacherId': instance.teacherId,
  'teacherName': instance.teacherName,
  'aliases': instance.aliases,
  'offerings': instance.offerings,
  'ratingAvg': instance.ratingAvg,
  'reviewCount': instance.reviewCount,
  'ratingDistribution': instance.ratingDistribution,
  'reviewScope': instance.reviewScope,
  'teamKey': instance.teamKey,
  'teamInstructors': instance.teamInstructors,
  'legacyNames': instance.legacyNames,
};
