/// Mirrors /api/campus. School callbacks always return a clean 303 redirect, including session rejection. No school credential or raw student ID is exposed. The private profile may contain the viewer’s name.
class CampusBinding {
  const CampusBinding({
    required this.maskedId,
    required this.boundAt,
    required this.revision,
    required this.needsAuthorization,
  });
  final String maskedId;

  /// Current identity binding time; preserved on same-identity reauthorization.
  final String boundAt;
  final String revision;
  final bool needsAuthorization;
  factory CampusBinding.fromJson(Map<String, dynamic> json) => CampusBinding(
    maskedId: json['maskedId'] as String,
    boundAt: json['boundAt'] as String,
    revision: json['revision'] as String,
    needsAuthorization: json['needsAuthorization'] as bool,
  );
}

class CampusCandidate {
  const CampusCandidate({
    required this.maskedId,
    required this.mode,
    required this.expiresAt,
  });
  final String maskedId;
  final String mode;
  final String expiresAt;
  factory CampusCandidate.fromJson(Map<String, dynamic> json) =>
      CampusCandidate(
        maskedId: json['maskedId'] as String,
        mode: json['mode'] as String,
        expiresAt: json['expiresAt'] as String,
      );
}

class CampusStatus {
  const CampusStatus({
    required this.enabled,
    required this.binding,
    required this.candidate,
  });
  final bool enabled;
  final CampusBinding? binding;
  final CampusCandidate? candidate;
  factory CampusStatus.fromJson(Map<String, dynamic> json) => CampusStatus(
    enabled: json['enabled'] as bool,
    binding: json['binding'] == null
        ? null
        : CampusBinding.fromJson(json['binding'] as Map<String, dynamic>),
    candidate: json['candidate'] == null
        ? null
        : CampusCandidate.fromJson(json['candidate'] as Map<String, dynamic>),
  );
}

class CampusMetric {
  const CampusMetric({
    required this.label,
    required this.value,
    required this.unit,
  });
  final String label;
  final String value;
  final String unit;
  factory CampusMetric.fromJson(Map<String, dynamic> json) => CampusMetric(
    label: json['label'] as String,
    value: json['value'] as String,
    unit: json['unit'] as String,
  );
}

class CampusPoint {
  const CampusPoint({required this.label, required this.value});
  final String label;
  final double value;
  factory CampusPoint.fromJson(Map<String, dynamic> json) => CampusPoint(
    label: json['label'] as String,
    value: (json['value'] as num).toDouble(),
  );
}

class CampusEvent {
  const CampusEvent({
    required this.name,
    required this.teacher,
    required this.room,
    required this.campus,
    required this.day,
    required this.start,
    required this.end,
    required this.weeks,
    required this.credits,
  });
  final String name;
  final String teacher;
  final String room;
  final String campus;
  final int day;
  final int start;
  final int end;
  final List<int> weeks;
  final String credits;
  factory CampusEvent.fromJson(Map<String, dynamic> json) => CampusEvent(
    name: json['name'] as String,
    teacher: json['teacher'] as String,
    room: json['room'] as String,
    campus: json['campus'] as String,
    day: json['day'] as int,
    start: json['start'] as int,
    end: json['end'] as int,
    weeks: (json['weeks'] as List).cast<int>(),
    credits: json['credits'] as String,
  );
}

class CampusDataset {
  const CampusDataset({
    required this.key,
    required this.status,
    required this.updatedAt,
    required this.metrics,
    required this.columns,
    required this.rows,
    required this.events,
    required this.series,
    this.messages = const [],
  });
  final String key;
  final String status;
  final String updatedAt;
  final List<CampusMetric> metrics;
  final List<String> columns;
  final List<List<String>> rows;
  final List<CampusEvent> events;
  final List<CampusPoint> series;
  final List<CampusMessageSummary> messages;
  factory CampusDataset.fromJson(Map<String, dynamic> json) => CampusDataset(
    key: json['key'] as String,
    status: json['status'] as String,
    updatedAt: json['updatedAt'] as String,
    metrics: (json['metrics'] as List)
        .map((v) => CampusMetric.fromJson(v as Map<String, dynamic>))
        .toList(),
    columns: (json['columns'] as List).cast<String>(),
    rows: (json['rows'] as List)
        .map((row) => (row as List).cast<String>())
        .toList(),
    events: (json['events'] as List)
        .map((v) => CampusEvent.fromJson(v as Map<String, dynamic>))
        .toList(),
    messages: (json['messages'] as List? ?? [])
        .map((v) => CampusMessageSummary.fromJson(v as Map<String, dynamic>))
        .toList(),
    series: (json['series'] as List)
        .map((v) => CampusPoint.fromJson(v as Map<String, dynamic>))
        .toList(),
  );
}

class CampusStartRequest {
  const CampusStartRequest({required this.mode});
  final String mode;
  factory CampusStartRequest.fromJson(Map<String, dynamic> json) =>
      CampusStartRequest(mode: json['mode'] as String);
}

class CampusUnbindRequest {
  const CampusUnbindRequest({required this.revision});
  final String revision;
  factory CampusUnbindRequest.fromJson(Map<String, dynamic> json) =>
      CampusUnbindRequest(revision: json['revision'] as String);
}

class CampusMessageSummary {
  const CampusMessageSummary({
    required this.id,
    required this.title,
    required this.publisher,
    required this.publishedAt,
  });
  final String id;
  final String title;
  final String publisher;
  final String publishedAt;
  factory CampusMessageSummary.fromJson(Map<String, dynamic> json) =>
      CampusMessageSummary(
        id: json['id'] as String,
        title: json['title'] as String,
        publisher: json['publisher'] as String,
        publishedAt: json['publishedAt'] as String,
      );
}

class CampusMessageLink {
  const CampusMessageLink({required this.label, required this.url});
  final String label;
  final String url;
  factory CampusMessageLink.fromJson(Map<String, dynamic> json) =>
      CampusMessageLink(
        label: json['label'] as String,
        url: json['url'] as String,
      );
}

class CampusMessageDetail extends CampusMessageSummary {
  const CampusMessageDetail({
    required super.id,
    required super.title,
    required super.publisher,
    required super.publishedAt,
    required this.content,
    required this.links,
  });
  final String content;
  final List<CampusMessageLink> links;
  factory CampusMessageDetail.fromJson(Map<String, dynamic> json) =>
      CampusMessageDetail(
        id: json['id'] as String,
        title: json['title'] as String,
        publisher: json['publisher'] as String,
        publishedAt: json['publishedAt'] as String,
        content: json['content'] as String,
        links: (json['links'] as List)
            .map((v) => CampusMessageLink.fromJson(v as Map<String, dynamic>))
            .toList(),
      );
}

/// Explicit private iCalendar snapshot; never place in the offline cache.
class CampusCalendarExport {
  const CampusCalendarExport({
    required this.filename,
    required this.content,
    required this.eventCount,
  });
  final String filename;
  final String content;
  final int eventCount;
  factory CampusCalendarExport.fromJson(Map<String, dynamic> json) =>
      CampusCalendarExport(
        filename: json['filename'] as String,
        content: json['content'] as String,
        eventCount: json['eventCount'] as int,
      );
}
