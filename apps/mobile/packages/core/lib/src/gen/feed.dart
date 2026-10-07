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
  const FeedEventsRequest(this.patches);
  final List<FeedEventPatch> patches;
  Map<String, dynamic> toJson() => {
    'patches': patches.map((p) => p.toJson()).toList(),
  };
}

class FeedEventsResponse {
  const FeedEventsResponse(this.code, this.accepted);
  factory FeedEventsResponse.fromJson(Map<String, dynamic> json) =>
      FeedEventsResponse(json['code'] as int, json['result'] as bool);
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
