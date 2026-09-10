/// 排课器节次作息契约镜像（`GET /api/pk/section-times` 的 data）。
///
/// 手写维护；与后端响应形状一致。后端返回现行 11 节编号的作息表
/// （未配置时为内置默认 11 节表，旧 12 节存量经归一重映射，与
/// defaultconfig 同源）；历史 12 节制学期由客户端内置历史表渲染，
/// 不消费本响应。
library;

/// 单个节次的起止时间（HH:MM）。
class SectionTimeSetting {
  const SectionTimeSetting({
    required this.section,
    required this.start,
    required this.end,
  });

  final int section;
  final String start;
  final String end;

  factory SectionTimeSetting.fromJson(Map<String, dynamic> json) =>
      SectionTimeSetting(
        section: (json['section'] as num?)?.toInt() ?? 0,
        start: json['start'] as String? ?? '',
        end: json['end'] as String? ?? '',
      );
}

/// GET /api/pk/section-times 响应 data。
class SectionTimesPayload {
  const SectionTimesPayload({
    required this.sectionTimes,
    required this.maxRowsDefault,
  });

  final List<SectionTimeSetting> sectionTimes;

  /// 默认行数（恒 11：现行 11 节制；历史 12 节制学期不消费本响应）。
  final int maxRowsDefault;

  factory SectionTimesPayload.fromJson(Map<String, dynamic> json) =>
      SectionTimesPayload(
        sectionTimes: (json['sectionTimes'] as List<dynamic>? ?? const [])
            .map(
              (e) => SectionTimeSetting.fromJson(
                Map<String, dynamic>.from(e as Map),
              ),
            )
            .toList(),
        maxRowsDefault: (json['maxRowsDefault'] as num?)?.toInt() ?? 11,
      );
}
