import 'anonymous_identity.dart';

/// Restricted admin-only DTOs. Never store in public models or offline caches.
class AdminAnonymousOwner {
  const AdminAnonymousOwner({
    required this.userId,
    required this.username,
    required this.closed,
    required this.frozen,
  });
  final int userId;
  final String username;
  final bool closed, frozen;
  factory AdminAnonymousOwner.fromJson(Map<String, dynamic> json) =>
      AdminAnonymousOwner(
        userId: (json['userId'] as num).toInt(),
        username: json['username'] as String,
        closed: json['closed'] as bool,
        frozen: json['frozen'] as bool,
      );
}

class AdminAnonymousIdentity {
  const AdminAnonymousIdentity({
    required this.persona,
    required this.owner,
    required this.disabled,
    required this.governanceDisabled,
    required this.selectedAt,
  });
  final AnonymousPersona persona;
  final AdminAnonymousOwner owner;
  final bool disabled, governanceDisabled;
  final DateTime selectedAt;
  factory AdminAnonymousIdentity.fromJson(Map<String, dynamic> json) =>
      AdminAnonymousIdentity(
        persona: AnonymousPersona.fromJson(json),
        owner: AdminAnonymousOwner.fromJson(
          json['owner'] as Map<String, dynamic>,
        ),
        disabled: json['disabled'] as bool,
        governanceDisabled: json['governanceDisabled'] as bool,
        selectedAt: DateTime.parse(json['selectedAt'] as String),
      );
}

class AdminAnonymousList {
  const AdminAnonymousList({
    required this.items,
    required this.total,
    required this.page,
    required this.pageSize,
  });
  final List<AdminAnonymousIdentity> items;
  final int total, page, pageSize;
  factory AdminAnonymousList.fromJson(Map<String, dynamic> json) =>
      AdminAnonymousList(
        items: (json['items'] as List)
            .map(
              (item) =>
                  AdminAnonymousIdentity.fromJson(item as Map<String, dynamic>),
            )
            .toList(),
        total: (json['total'] as num).toInt(),
        page: (json['page'] as num).toInt(),
        pageSize: (json['pageSize'] as num).toInt(),
      );
}

class AdminAnonymousListRequest {
  const AdminAnonymousListRequest({
    required this.reason,
    this.page = 1,
    this.pageSize = 10,
    this.search = '',
    this.status = 'all',
  });
  final String reason, search, status;
  final int page, pageSize;
  Map<String, dynamic> toJson() => {
    'page': page,
    'pageSize': pageSize,
    'search': search,
    'status': status,
    'reason': reason,
  };
}

class AdminAnonymousGovernRequest {
  const AdminAnonymousGovernRequest({
    required this.publicUid,
    required this.disabled,
    required this.reason,
  });
  final String publicUid, reason;
  final bool disabled;
  Map<String, dynamic> toJson() => {
    'publicUid': publicUid,
    'disabled': disabled,
    'reason': reason,
  };
}
