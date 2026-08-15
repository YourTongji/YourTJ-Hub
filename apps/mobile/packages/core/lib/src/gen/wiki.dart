/// Wiki 域契约镜像（对应 packages/api-contract 的 wiki 域）。
///
/// 手写维护，与后端 JSON 形状保持一致；`GfResponse<T>.fromJson` 配合
/// 各类型的 `fromJson` 工厂解析 `{code, messageCode, params, result}` 信封。
/// 时间字段统一为 RFC3339 字符串（后端 `time.Time` JSON 序列化格式）。
///
/// GitHub SSOT：wiki 内容以 GitHub 仓库为唯一真实源，论坛侧只读投影 +
/// 同步器。站内编辑/审核/回滚/贡献者管理 API 已移除（走 GitHub PR）。
library;

/// 命名空间摘要（公开 `GET /api/wiki/namespaces` 与首页复用）。
class WikiNamespace {
  const WikiNamespace({
    required this.name,
    required this.slug,
    required this.description,
    required this.sortOrder,
    required this.pageCount,
    required this.updatedAt,
    required this.firstPagePath,
  });

  final String name;

  /// URL 友好标识（^[a-z0-9]+(-[a-z0-9]+)*$ ≤64），与显示名 name 分离；
  /// 未分配时为空串。
  final String slug;
  final String description;
  final int sortOrder;
  final int pageCount;
  final String updatedAt;
  final String firstPagePath;

  factory WikiNamespace.fromJson(Map<String, dynamic> json) {
    return WikiNamespace(
      name: json['name'] as String? ?? '',
      slug: json['slug'] as String? ?? '',
      description: json['description'] as String? ?? '',
      sortOrder: (json['sortOrder'] as num?)?.toInt() ?? 0,
      pageCount: (json['pageCount'] as num?)?.toInt() ?? 0,
      updatedAt: json['updatedAt'] as String? ?? '',
      firstPagePath: json['firstPagePath'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'name': name,
      'slug': slug,
      'description': description,
      'sortOrder': sortOrder,
      'pageCount': pageCount,
      'updatedAt': updatedAt,
      'firstPagePath': firstPagePath,
    };
  }
}

/// 导航树中的一页（公开 `GET /api/wiki/tree`）。
class WikiTreePage {
  const WikiTreePage({
    required this.pageId,
    required this.path,
    required this.title,
    required this.active,
  });

  final int pageId;
  final String path;
  final String title;
  final bool active;

  factory WikiTreePage.fromJson(Map<String, dynamic> json) {
    return WikiTreePage(
      pageId: (json['pageId'] as num?)?.toInt() ?? 0,
      path: json['path'] as String? ?? '',
      title: json['title'] as String? ?? '',
      active: json['active'] as bool? ?? false,
    );
  }

  Map<String, dynamic> toJson() {
    return {'pageId': pageId, 'path': path, 'title': title, 'active': active};
  }
}

/// 导航树中的一个 namespace 分组。
class WikiTreeNamespace {
  const WikiTreeNamespace({
    required this.name,
    required this.label,
    required this.pages,
  });

  final String name;
  final String label;
  final List<WikiTreePage> pages;

  factory WikiTreeNamespace.fromJson(Map<String, dynamic> json) {
    return WikiTreeNamespace(
      name: json['name'] as String? ?? '',
      label: json['label'] as String? ?? '',
      pages: (json['pages'] as List<dynamic>? ?? const [])
          .map((item) => WikiTreePage.fromJson(item as Map<String, dynamic>))
          .toList(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'name': name,
      'label': label,
      'pages': pages.map((page) => page.toJson()).toList(),
    };
  }
}

/// 首页最近更新条目（`GET /api/wiki/home`）。
class WikiRecentPage {
  const WikiRecentPage({
    required this.pageId,
    required this.path,
    required this.title,
    required this.updatedAt,
    required this.editorId,
    required this.editorName,
  });

  final int pageId;
  final String path;
  final String title;
  final String updatedAt;
  final int editorId;
  final String editorName;

  factory WikiRecentPage.fromJson(Map<String, dynamic> json) {
    return WikiRecentPage(
      pageId: (json['pageId'] as num?)?.toInt() ?? 0,
      path: json['path'] as String? ?? '',
      title: json['title'] as String? ?? '',
      updatedAt: json['updatedAt'] as String? ?? '',
      editorId: (json['editorId'] as num?)?.toInt() ?? 0,
      editorName: json['editorName'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'pageId': pageId,
      'path': path,
      'title': title,
      'updatedAt': updatedAt,
      'editorId': editorId,
      'editorName': editorName,
    };
  }
}

/// 首页数据（`GET /api/wiki/home` 的 result）。
class WikiHomeData {
  const WikiHomeData({required this.namespaces, required this.recent});

  final List<WikiNamespace> namespaces;
  final List<WikiRecentPage> recent;

  factory WikiHomeData.fromJson(Map<String, dynamic> json) {
    return WikiHomeData(
      namespaces: (json['namespaces'] as List<dynamic>? ?? const [])
          .map((item) => WikiNamespace.fromJson(item as Map<String, dynamic>))
          .toList(),
      recent: (json['recent'] as List<dynamic>? ?? const [])
          .map((item) => WikiRecentPage.fromJson(item as Map<String, dynamic>))
          .toList(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'namespaces': namespaces.map((ns) => ns.toJson()).toList(),
      'recent': recent.map((page) => page.toJson()).toList(),
    };
  }
}

/// 页面详情（wiki 页面渲染负载中的 `page`）。
class WikiPageDetail {
  const WikiPageDetail({
    required this.id,
    required this.topicId,
    required this.namespace,
    required this.path,
    required this.title,
    required this.content,
    required this.toc,
    required this.updatedAt,
    required this.editorId,
    required this.editorName,
    required this.likeCount,
    required this.viewCount,
    required this.postCount,
    required this.liked,
    required this.bookmarked,
    required this.watched,
    this.canEdit = false,
    this.publishedRevisionNo = 0,
    this.editUrl = '',
    this.historyUrl = '',
  });

  final int id;
  final int topicId;
  final String namespace;
  final String path;
  final String title;
  final String content;
  final List<WikiTocItem> toc;
  final String updatedAt;
  final int editorId;
  final String editorName;
  final int likeCount;
  final int viewCount;
  final int postCount;
  final bool liked;
  final bool bookmarked;
  final bool watched;

  /// GitHub SSOT：配置了 [wiki.git] 仓库时为 true（展示「编辑此页」外链）。
  final bool canEdit;
  final int publishedRevisionNo;

  /// GitHub 仓库编辑外链（{repo}/edit/{branch}/{path}.md；未配置时为空）。
  final String editUrl;

  /// GitHub 仓库历史外链（{repo}/commits/{branch}/{path}.md；未配置时为空）。
  final String historyUrl;

  factory WikiPageDetail.fromJson(Map<String, dynamic> json) {
    return WikiPageDetail(
      id: (json['id'] as num?)?.toInt() ?? 0,
      topicId: (json['topicId'] as num?)?.toInt() ?? 0,
      namespace: json['namespace'] as String? ?? '',
      path: json['path'] as String? ?? '',
      title: json['title'] as String? ?? '',
      content: json['content'] as String? ?? '',
      toc: (json['toc'] as List<dynamic>? ?? const [])
          .map((item) => WikiTocItem.fromJson(item as Map<String, dynamic>))
          .toList(),
      updatedAt: json['updatedAt'] as String? ?? '',
      editorId: (json['editorId'] as num?)?.toInt() ?? 0,
      editorName: json['editorName'] as String? ?? '',
      likeCount: (json['likeCount'] as num?)?.toInt() ?? 0,
      viewCount: (json['viewCount'] as num?)?.toInt() ?? 0,
      postCount: (json['postCount'] as num?)?.toInt() ?? 0,
      liked: json['liked'] as bool? ?? false,
      bookmarked: json['bookmarked'] as bool? ?? false,
      watched: json['watched'] as bool? ?? false,
      canEdit: json['canEdit'] as bool? ?? false,
      publishedRevisionNo: (json['publishedRevisionNo'] as num?)?.toInt() ?? 0,
      editUrl: json['editUrl'] as String? ?? '',
      historyUrl: json['historyUrl'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'topicId': topicId,
      'namespace': namespace,
      'path': path,
      'title': title,
      'content': content,
      'toc': toc.map((item) => item.toJson()).toList(),
      'updatedAt': updatedAt,
      'editorId': editorId,
      'editorName': editorName,
      'likeCount': likeCount,
      'viewCount': viewCount,
      'postCount': postCount,
      'liked': liked,
      'bookmarked': bookmarked,
      'watched': watched,
      'canEdit': canEdit,
      'publishedRevisionNo': publishedRevisionNo,
      if (editUrl.isNotEmpty) 'editUrl': editUrl,
      if (historyUrl.isNotEmpty) 'historyUrl': historyUrl,
    };
  }
}

/// 目录条目（`WikiPageDetail.toc`）。
class WikiTocItem {
  const WikiTocItem({
    required this.level,
    required this.id,
    required this.text,
  });

  final int level;
  final String id;
  final String text;

  factory WikiTocItem.fromJson(Map<String, dynamic> json) {
    return WikiTocItem(
      level: (json['level'] as num?)?.toInt() ?? 0,
      id: json['id'] as String? ?? '',
      text: json['text'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {'level': level, 'id': id, 'text': text};
  }
}

/// 管理端树中的一页（`GET /api/admin/wiki/tree`）。
class WikiAdminTreePage {
  const WikiAdminTreePage({
    required this.pageId,
    required this.path,
    required this.sourcePath,
    required this.title,
    required this.sortOrder,
  });

  final int pageId;

  /// URL 友好路径（首段 = slug，降级 = 显示名）。
  final String path;

  /// 仓库真实相对路径（GitHub 编辑/历史外链用）。
  final String sourcePath;
  final String title;
  final int sortOrder;

  factory WikiAdminTreePage.fromJson(Map<String, dynamic> json) {
    return WikiAdminTreePage(
      pageId: (json['pageId'] as num?)?.toInt() ?? 0,
      path: json['path'] as String? ?? '',
      sourcePath: json['sourcePath'] as String? ?? '',
      title: json['title'] as String? ?? '',
      sortOrder: (json['sortOrder'] as num?)?.toInt() ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'pageId': pageId,
      'path': path,
      'sourcePath': sourcePath,
      'title': title,
      'sortOrder': sortOrder,
    };
  }
}

/// 管理端树中的一个 namespace 分组。
class WikiAdminTreeNamespace {
  const WikiAdminTreeNamespace({
    required this.name,
    required this.label,
    required this.pages,
  });

  final String name;
  final String label;
  final List<WikiAdminTreePage> pages;

  factory WikiAdminTreeNamespace.fromJson(Map<String, dynamic> json) {
    return WikiAdminTreeNamespace(
      name: json['name'] as String? ?? '',
      label: json['label'] as String? ?? '',
      pages: (json['pages'] as List<dynamic>? ?? const [])
          .map(
            (item) => WikiAdminTreePage.fromJson(item as Map<String, dynamic>),
          )
          .toList(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'name': name,
      'label': label,
      'pages': pages.map((page) => page.toJson()).toList(),
    };
  }
}

/// 同步页面计数（`GET /api/admin/wiki/sync/status`）。
class WikiSyncPageCounts {
  const WikiSyncPageCounts({required this.total, required this.namespaces});

  final int total;
  final int namespaces;

  factory WikiSyncPageCounts.fromJson(Map<String, dynamic> json) {
    return WikiSyncPageCounts(
      total: (json['total'] as num?)?.toInt() ?? 0,
      namespaces: (json['namespaces'] as num?)?.toInt() ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {'total': total, 'namespaces': namespaces};
  }
}

/// 一次同步运行视图（`GET /api/admin/wiki/sync/status` 与 `/sync/runs`）。
class WikiSyncRunView {
  const WikiSyncRunView({
    required this.id,
    required this.headSha,
    required this.trigger,
    required this.status,
    required this.pagesAdded,
    required this.pagesUpdated,
    required this.pagesDeleted,
    this.error = '',
    required this.startedAt,
    this.finishedAt,
  });

  final int id;
  final String headSha;

  /// manual | schedule | webhook | startup
  final String trigger;

  /// running | success | failed
  final String status;
  final int pagesAdded;
  final int pagesUpdated;
  final int pagesDeleted;
  final String error;
  final String startedAt;
  final String? finishedAt;

  factory WikiSyncRunView.fromJson(Map<String, dynamic> json) {
    return WikiSyncRunView(
      id: (json['id'] as num?)?.toInt() ?? 0,
      headSha: json['headSha'] as String? ?? '',
      trigger: json['trigger'] as String? ?? '',
      status: json['status'] as String? ?? '',
      pagesAdded: (json['pagesAdded'] as num?)?.toInt() ?? 0,
      pagesUpdated: (json['pagesUpdated'] as num?)?.toInt() ?? 0,
      pagesDeleted: (json['pagesDeleted'] as num?)?.toInt() ?? 0,
      error: json['error'] as String? ?? '',
      startedAt: json['startedAt'] as String? ?? '',
      finishedAt: json['finishedAt'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'headSha': headSha,
      'trigger': trigger,
      'status': status,
      'pagesAdded': pagesAdded,
      'pagesUpdated': pagesUpdated,
      'pagesDeleted': pagesDeleted,
      if (error.isNotEmpty) 'error': error,
      'startedAt': startedAt,
      if (finishedAt != null) 'finishedAt': finishedAt,
    };
  }
}

/// 同步面板状态（`GET /api/admin/wiki/sync/status` 的 result）。
class WikiSyncStatus {
  const WikiSyncStatus({
    required this.enabled,
    required this.repo,
    required this.branch,
    required this.headSha,
    this.lastRun,
    this.recentRuns = const [],
    required this.pages,
  });

  final bool enabled;
  final String repo;
  final String branch;
  final String headSha;
  final WikiSyncRunView? lastRun;
  final List<WikiSyncRunView> recentRuns;
  final WikiSyncPageCounts pages;

  factory WikiSyncStatus.fromJson(Map<String, dynamic> json) {
    return WikiSyncStatus(
      enabled: json['enabled'] as bool? ?? false,
      repo: json['repo'] as String? ?? '',
      branch: json['branch'] as String? ?? '',
      headSha: json['headSha'] as String? ?? '',
      lastRun: json['lastRun'] == null
          ? null
          : WikiSyncRunView.fromJson(json['lastRun'] as Map<String, dynamic>),
      recentRuns: (json['recentRuns'] as List<dynamic>? ?? const [])
          .map((item) => WikiSyncRunView.fromJson(item as Map<String, dynamic>))
          .toList(),
      pages: WikiSyncPageCounts.fromJson(
        json['pages'] as Map<String, dynamic>? ?? const {},
      ),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'enabled': enabled,
      'repo': repo,
      'branch': branch,
      'headSha': headSha,
      if (lastRun != null) 'lastRun': lastRun!.toJson(),
      'recentRuns': recentRuns.map((run) => run.toJson()).toList(),
      'pages': pages.toJson(),
    };
  }
}

/// 手动同步已接受（`POST /api/admin/wiki/sync`）。同步异步执行，
/// 进度通过 `sync/status` / `sync/runs` 轮询。
class WikiSyncAccepted {
  const WikiSyncAccepted({required this.accepted});

  final bool accepted;

  factory WikiSyncAccepted.fromJson(Map<String, dynamic> json) {
    return WikiSyncAccepted(accepted: json['accepted'] as bool? ?? false);
  }

  Map<String, dynamic> toJson() {
    return {'accepted': accepted};
  }
}

/// webhook 验签密钥配置状态（`GET /api/admin/wiki/sync/webhook-secret` 的 result）。
class WikiWebhookSecretStatus {
  const WikiWebhookSecretStatus({required this.configured});

  final bool configured;

  factory WikiWebhookSecretStatus.fromJson(Map<String, dynamic> json) {
    return WikiWebhookSecretStatus(configured: json['configured'] as bool? ?? false);
  }

  Map<String, dynamic> toJson() {
    return {'configured': configured};
  }
}

/// 保存/清除 webhook 验签密钥请求体（`POST /api/admin/wiki/sync/webhook-secret`）。
class WikiWebhookSecretSaveRequest {
  const WikiWebhookSecretSaveRequest({required this.secret});

  /// 明文密钥（仅保存瞬间存在）；空串表示清除已存密钥。
  final String secret;

  Map<String, dynamic> toJson() {
    return {'secret': secret};
  }
}

/// 保存 webhook 验签密钥的 result（`{ok: true}`）。
class WikiWebhookSecretSaveResult {
  const WikiWebhookSecretSaveResult({required this.ok});

  final bool ok;

  factory WikiWebhookSecretSaveResult.fromJson(Map<String, dynamic> json) {
    return WikiWebhookSecretSaveResult(ok: json['ok'] as bool? ?? false);
  }

  Map<String, dynamic> toJson() {
    return {'ok': ok};
  }
}
