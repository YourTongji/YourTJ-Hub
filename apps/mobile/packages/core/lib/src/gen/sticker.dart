/// 表情包域契约镜像（对应 packages/api-contract 的 paths/admin-stickers.yaml 与
/// paths/forum-read.yaml#forumStickerList，MADR 0030）。
///
/// 手写维护（user_content.dart / wiki_search.dart 同风格）：Dart 代码生成仍为
/// Planned，未跑 build_runner，因此不使用 freezed part 文件。字段名与
/// components/schemas.yaml 的 Sticker* / AdminSticker* 块逐一对应；统一
/// {result, code, messageCode, params} 信封由客户端解包后交本文件模型解析，
/// 解析侧做容错默认值。
library;

/// 公开表情包条目（GET /api/forum/stickers，启用列表，name 全局唯一）。
class StickerItemPayload {
  const StickerItemPayload({required this.name, required this.url});
  final String name;
  final String url;
  factory StickerItemPayload.fromJson(Map<String, dynamic> json) =>
      StickerItemPayload(
        name: json['name'] as String? ?? '',
        url: json['url'] as String? ?? '',
      );
}

/// 管理端表情包行（GET /api/admin/stickers，含禁用行）。
class StickerAdminItemPayload {
  const StickerAdminItemPayload({
    required this.id,
    required this.name,
    required this.fileName,
    required this.url,
    required this.sortOrder,
    required this.isEnabled,
    required this.createdBy,
  });
  final int id;
  final String name;
  final String fileName;
  final String url;
  final int sortOrder;
  final bool isEnabled;
  final int createdBy;
  factory StickerAdminItemPayload.fromJson(Map<String, dynamic> json) =>
      StickerAdminItemPayload(
        id: (json['id'] as num?)?.toInt() ?? 0,
        name: json['name'] as String? ?? '',
        fileName: json['fileName'] as String? ?? '',
        url: json['url'] as String? ?? '',
        sortOrder: (json['sortOrder'] as num?)?.toInt() ?? 0,
        isEnabled: json['isEnabled'] == true,
        createdBy: (json['createdBy'] as num?)?.toInt() ?? 0,
      );
}

/// 创建/更新表情包请求（POST /api/admin/sticker-save；id=0 创建）。
class StickerSaveRequest {
  const StickerSaveRequest({
    this.id = 0,
    required this.name,
    this.fileName,
    this.sortOrder = 0,
    this.isEnabled = true,
  });
  final int id;
  final String name;
  final String? fileName;
  final int sortOrder;
  final bool isEnabled;
  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'fileName': ?fileName,
    'sortOrder': sortOrder,
    'isEnabled': isEnabled,
  };
}

/// 压缩包导入的单条失败原因（zip 包内某条目未被导入的原因）。
class StickerImportIssuePayload {
  const StickerImportIssuePayload({required this.name, required this.reason});
  final String name;
  final String reason;
  factory StickerImportIssuePayload.fromJson(Map<String, dynamic> json) =>
      StickerImportIssuePayload(
        name: json['name'] as String? ?? '',
        reason: json['reason'] as String? ?? '',
      );
}

/// 压缩包导入结果（POST /api/admin/sticker-import；failed 条目非致命）。
class StickerImportResultPayload {
  const StickerImportResultPayload({
    required this.imported,
    required this.skipped,
    required this.failed,
  });
  final int imported;
  final int skipped;
  final List<StickerImportIssuePayload> failed;
  factory StickerImportResultPayload.fromJson(Map<String, dynamic> json) =>
      StickerImportResultPayload(
        imported: (json['imported'] as num?)?.toInt() ?? 0,
        skipped: (json['skipped'] as num?)?.toInt() ?? 0,
        failed: (json['failed'] as List? ?? [])
            .map(
              (e) => StickerImportIssuePayload.fromJson(
                Map<String, dynamic>.from(e as Map),
              ),
            )
            .toList(),
      );
}
