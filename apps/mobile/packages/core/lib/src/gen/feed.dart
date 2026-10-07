import 'topic.dart';
import 'common.dart';

/// Mirrors the bounded feed observation request in OpenAPI. Traces remain in
/// memory and must not be serialized into the persistent offline page cache.
class FeedEventPatch {
  const FeedEventPatch({
    required this.trace,
    required this.visibleMask,
    this.dwell = const {},
  });
  final String trace;
  final int visibleMask;
  final Map<String, int> dwell;
  Map<String, dynamic> toJson() => {
    'trace': trace,
    'visibleMask': visibleMask,
    if (dwell.isNotEmpty) 'dwell': dwell,
  };
}

class FeedEventsRequest {
  const FeedEventsRequest(this.patches, {this.seenPatches = const []});
  final List<FeedSeenPatch> seenPatches;
  final List<FeedEventPatch> patches;
  Map<String, dynamic> toJson() => {
    'patches': patches.map((p) => p.toJson()).toList(),
    if (seenPatches.isNotEmpty)
      'seenPatches': seenPatches.map((p) => p.toJson()).toList(),
  };
}

class FeedEventsResponse {
  const FeedEventsResponse(this.code, this.accepted);
  factory FeedEventsResponse.fromJson(Map<String, dynamic> json) =>
      FeedEventsResponse(
        json['code'] as int,
        json['result'] == true ||
            (json['result'] is Map && json['result']['seenConfirmed'] == true),
      );
  final int code;
  final bool accepted;
}

/// Admin-only anonymous projection; the mobile app has no raw telemetry reader.
class FeedSummary {
  FeedSummary.fromJson(Map<String, dynamic> json)
    : enabled = json['enabled'] as bool,
      rankingReady = json['rankingReady'] as bool,
      metricsEnabled = json['metricsEnabled'] as bool,
      rolloutPercent = json['rolloutPercent'] as int,
      rawRetentionDays = json['rawRetentionDays'] as int,
      paramsHash = json['paramsHash'] as String,
      truncated = json['truncated'] as bool,
      rows = (json['rows'] as List).cast<Map<String, dynamic>>(),
      periods = (json['periods'] as List).cast<Map<String, dynamic>>(),
      health = json['health'] as Map<String, dynamic>;
  final bool enabled, rankingReady, metricsEnabled, truncated;
  final int rolloutPercent, rawRetentionDays;
  final String paramsHash;
  final List<Map<String, dynamic>> rows, periods;
  final Map<String, dynamic> health;
}

/// Functional card exposure protocol. The Flutter observer is not wired to it yet.
class FeedSeenProof {
  const FeedSeenProof({
    required this.token,
    required this.topicIds,
    required this.issuedAt,
    required this.expiresAt,
  });
  factory FeedSeenProof.fromJson(Map<String, dynamic> json) => FeedSeenProof(
    token: json['token'] as String,
    topicIds: (json['topicIds'] as List).cast<int>(),
    issuedAt: json['issuedAt'] as int,
    expiresAt: json['expiresAt'] as int,
  );
  final String token;
  final List<int> topicIds;
  final int issuedAt, expiresAt;
  Map<String, dynamic> toJson() => {
    'token': token,
    'topicIds': topicIds,
    'issuedAt': issuedAt,
    'expiresAt': expiresAt,
  };
}

class FeedSeenPatch {
  const FeedSeenPatch({required this.proof, required this.seen});
  final String proof;
  final Map<String, int> seen;
  Map<String, dynamic> toJson() => {'proof': proof, 'seen': seen};
}

class FeedRefreshRequest {
  const FeedRefreshRequest({
    this.seenPatches = const [],
    this.replaceSnapshotId = '',
  });
  final List<FeedSeenPatch> seenPatches;
  final String replaceSnapshotId;
  Map<String, dynamic> toJson() => {
    'seenPatches': seenPatches.map((p) => p.toJson()).toList(),
    if (replaceSnapshotId.isNotEmpty) 'replaceSnapshotId': replaceSnapshotId,
  };
}

class FeedReconcileRequest {
  const FeedReconcileRequest(this.topicIds);
  final List<int> topicIds;
  Map<String, dynamic> toJson() => {'topicIds': topicIds};
}

class FeedSessionResult {
  FeedSessionResult.fromJson(Map<String, dynamic> json)
    : viewerId = json['viewerId'] as int,
      topics = (json['topics'] as List)
          .map((t) => TopicPayload.fromJson(t as Map<String, dynamic>))
          .toList(),
      removedIds = (json['removedIds'] as List).cast<int>(),
      seenProofs = (json['seenProofs'] as List)
          .map((p) => FeedSeenProof.fromJson(p as Map<String, dynamic>))
          .toList(),
      snapshotId = json['snapshotId'] as String,
      pagination = json['pagination'] == null
          ? null
          : PaginationPayload.fromJson(
              json['pagination'] as Map<String, dynamic>,
            ),
      available = json['available'] as bool,
      seenConfirmed = json['seenConfirmed'] as bool;
  final int viewerId;
  final List<TopicPayload> topics;
  final List<int> removedIds;
  final List<FeedSeenProof> seenProofs;
  final String snapshotId;
  final PaginationPayload? pagination;
  final bool available, seenConfirmed;
}
