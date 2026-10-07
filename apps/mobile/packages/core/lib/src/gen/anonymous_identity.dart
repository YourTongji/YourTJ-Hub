/// Persistent forum persona DTOs. Private owner/seed fields are never mirrored.
/// New draws use phrase6-v1: two-character action + two-character scene/object +
/// 的 + one-character animal. Existing names/batches retain their exact text.
class AnonymousPersona {
  const AnonymousPersona({
    required this.publicUid,
    required this.name,
    required this.avatarUrl,
    required this.profileUrl,
  });
  final String publicUid, name, avatarUrl, profileUrl;
  String get kind => 'persona';
  factory AnonymousPersona.fromJson(Map<String, dynamic> j) => AnonymousPersona(
    publicUid: j['publicUid'] as String,
    name: j['name'] as String,
    avatarUrl: j['avatarUrl'] as String,
    profileUrl: j['profileUrl'] as String,
  );
}

class AnonymousNameBatch {
  const AnonymousNameBatch({
    required this.id,
    required this.words,
    required this.expiresAt,
    required this.day,
    required this.createdAt,
  });
  final String id, day;
  final DateTime createdAt;
  final List<String> words;
  final DateTime expiresAt;
  factory AnonymousNameBatch.fromJson(Map<String, dynamic> j) =>
      AnonymousNameBatch(
        id: j['id'] as String,
        words: List<String>.from(j['words'] as List),
        expiresAt: DateTime.parse(j['expiresAt'] as String),
        day: j['day'] as String,
        createdAt: DateTime.parse(j['createdAt'] as String),
      );
}

class AnonymousIdentityState {
  const AnonymousIdentityState({
    required this.persona,
    required this.day,
    required this.remaining,
    required this.resetsAt,
    required this.availableAt,
    required this.nameSelectedAt,
    required this.lexiconVersion,
    required this.disabled,
    required this.governanceDisabled,
    required this.batches,
  });
  final AnonymousPersona? persona;
  final String day, lexiconVersion;
  final int remaining;
  final DateTime resetsAt;
  final DateTime? availableAt, nameSelectedAt;
  final bool disabled, governanceDisabled;
  final List<AnonymousNameBatch> batches;
  bool get locked =>
      availableAt != null && DateTime.now().isBefore(availableAt!);
  bool get usable => persona != null && !disabled && !governanceDisabled;
  factory AnonymousIdentityState.fromJson(Map<String, dynamic> j) =>
      AnonymousIdentityState(
        persona: j['persona'] == null
            ? null
            : AnonymousPersona.fromJson(j['persona'] as Map<String, dynamic>),
        day: j['day'] as String,
        lexiconVersion: j['lexiconVersion'] as String,
        nameSelectedAt: j['nameSelectedAt'] == null
            ? null
            : DateTime.parse(j['nameSelectedAt'] as String),
        remaining: (j['remaining'] as num).toInt(),
        resetsAt: DateTime.parse(j['resetsAt'] as String),
        availableAt: j['nameChangeAvailableAt'] == null
            ? null
            : DateTime.parse(j['nameChangeAvailableAt'] as String),
        disabled: j['disabled'] as bool,
        governanceDisabled: j['governanceDisabled'] as bool,
        batches: (j['batches'] as List)
            .map((b) => AnonymousNameBatch.fromJson(b as Map<String, dynamic>))
            .toList(),
      );
}

/// Returned only by an explicitly authorized, audited reveal operation.
/// This result must never enter public author models or persisted caches.
class AnonymousReveal {
  const AnonymousReveal({
    required this.publicUid,
    required this.userId,
    required this.username,
  });
  final String publicUid, username;
  final int userId;
  factory AnonymousReveal.fromJson(Map<String, dynamic> j) => AnonymousReveal(
    publicUid: j['publicUid'] as String,
    userId: (j['userId'] as num).toInt(),
    username: j['username'] as String,
  );
}
