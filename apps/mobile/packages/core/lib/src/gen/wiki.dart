/// Wiki 域契约镜像（对应 packages/api-contract 的 wiki 域）。
///
/// 手写维护，与后端 JSON 形状保持一致；`GfResponse<T>.fromJson` 配合
/// 各类型的 `fromJson` 工厂解析 `{code, messageCode, params, result}` 信封。
/// 时间字段统一为 RFC3339 字符串（后端 `time.Time` JSON 序列化格式）。

/// 命名空间摘要（公开 `GET /api/wiki/namespaces` 与首页复用）。
class WikiNamespace {
  const WikiNamespace({
    required this.name,
    required this.description,
    required this.sortOrder,
    required this.pageCount,
    required this.updatedAt,
  });

  final String name;
  final String description;
  final int sortOrder;
  final int pageCount;
  final String updatedAt;

  factory WikiNamespace.fromJson(Map<String, dynamic> json) {
    return WikiNamespace(
      name: json['name'] as String? ?? '',
      description: json['description'] as String? ?? '',
      sortOrder: (json['sortOrder'] as num?)?.toInt() ?? 0,
      pageCount: (json['pageCount'] as num?)?.toInt() ?? 0,
      updatedAt: json['updatedAt'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'name': name,
      'description': description,
      'sortOrder': sortOrder,
      'pageCount': pageCount,
      'updatedAt': updatedAt,
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

/// 修订条目（公开 `GET /api/wiki/revisions?pageId=`）。
class WikiRevision {
  const WikiRevision({
    required this.revisionId,
    required this.pageId,
    required this.revisionNo,
    required this.title,
    required this.content,
    required this.status,
    required this.editorId,
    required this.editorName,
    required this.updatedAt,
  });

  final int revisionId;
  final int pageId;
  final int revisionNo;
  final String title;
  final String content;

  /// approved | pending | rejected | superseded
  final String status;
  final int editorId;
  final String editorName;
  final String updatedAt;

  factory WikiRevision.fromJson(Map<String, dynamic> json) {
    return WikiRevision(
      revisionId: (json['revisionId'] as num?)?.toInt() ?? 0,
      pageId: (json['pageId'] as num?)?.toInt() ?? 0,
      revisionNo: (json['revisionNo'] as num?)?.toInt() ?? 0,
      title: json['title'] as String? ?? '',
      content: json['content'] as String? ?? '',
      status: json['status'] as String? ?? '',
      editorId: (json['editorId'] as num?)?.toInt() ?? 0,
      editorName: json['editorName'] as String? ?? '',
      updatedAt: json['updatedAt'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'revisionId': revisionId,
      'pageId': pageId,
      'revisionNo': revisionNo,
      'title': title,
      'content': content,
      'status': status,
      'editorId': editorId,
      'editorName': editorName,
      'updatedAt': updatedAt,
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

/// 创建页面请求体（`POST /api/wiki/pages`）。
class WikiCreatePageRequest {
  const WikiCreatePageRequest({
    required this.namespace,
    required this.path,
    required this.title,
    required this.content,
  });

  final String namespace;
  final String path;
  final String title;
  final String content;

  Map<String, dynamic> toJson() {
    return {
      'namespace': namespace,
      'path': path,
      'title': title,
      'content': content,
    };
  }
}

/// 创建页面响应 result（`{pageId, path}`）。
class WikiCreatePageResponse {
  const WikiCreatePageResponse({required this.pageId, required this.path});

  final int pageId;
  final String path;

  factory WikiCreatePageResponse.fromJson(Map<String, dynamic> json) {
    return WikiCreatePageResponse(
      pageId: (json['pageId'] as num?)?.toInt() ?? 0,
      path: json['path'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {'pageId': pageId, 'path': path};
  }
}

/// 更新页面请求体（`PUT /api/wiki/pages/{pageId}`）。
class WikiUpdatePageRequest {
  const WikiUpdatePageRequest({required this.title, required this.content});

  final String title;
  final String content;

  Map<String, dynamic> toJson() {
    return {'title': title, 'content': content};
  }
}

/// 更新页面响应 result（`{revisionId, status: pending}`）。
class WikiUpdatePageResponse {
  const WikiUpdatePageResponse({
    required this.revisionId,
    required this.status,
  });

  final int revisionId;
  final String status;

  factory WikiUpdatePageResponse.fromJson(Map<String, dynamic> json) {
    return WikiUpdatePageResponse(
      revisionId: (json['revisionId'] as num?)?.toInt() ?? 0,
      status: json['status'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {'revisionId': revisionId, 'status': status};
  }
}

/// 审阅修订请求体（`POST /api/wiki/revisions/{revisionId}/review`）。
class WikiReviewRevisionRequest {
  const WikiReviewRevisionRequest({required this.action});

  /// approve | reject
  final String action;

  Map<String, dynamic> toJson() {
    return {'action': action};
  }
}

/// 审阅修订响应 result（`{revisionId, status: approved|rejected}`）。
class WikiReviewRevisionResponse {
  const WikiReviewRevisionResponse({
    required this.revisionId,
    required this.status,
  });

  final int revisionId;
  final String status;

  factory WikiReviewRevisionResponse.fromJson(Map<String, dynamic> json) {
    return WikiReviewRevisionResponse(
      revisionId: (json['revisionId'] as num?)?.toInt() ?? 0,
      status: json['status'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {'revisionId': revisionId, 'status': status};
  }
}

/// 创建命名空间请求体（`POST /api/admin/wiki/namespaces`）。
class WikiCreateNamespaceRequest {
  const WikiCreateNamespaceRequest({
    required this.name,
    required this.description,
  });

  final String name;
  final String description;

  Map<String, dynamic> toJson() {
    return {'name': name, 'description': description};
  }
}

/// 更新命名空间请求体（`PUT /api/admin/wiki/namespaces/{name}`）。
class WikiUpdateNamespaceRequest {
  const WikiUpdateNamespaceRequest({required this.description});

  final String description;

  Map<String, dynamic> toJson() {
    return {'description': description};
  }
}

/// 命名空间管理操作的统一 result（`{ok: true}`）。
class WikiNamespaceActionResponse {
  const WikiNamespaceActionResponse({required this.ok});

  final bool ok;

  factory WikiNamespaceActionResponse.fromJson(Map<String, dynamic> json) {
    return WikiNamespaceActionResponse(ok: json['ok'] as bool? ?? false);
  }

  Map<String, dynamic> toJson() {
    return {'ok': ok};
  }
}

/// 命名空间编辑者条目（`GET /api/admin/wiki/namespaces/{name}/editors`）。
class WikiEditorSummary {
  const WikiEditorSummary({
    required this.userId,
    required this.username,
    required this.avatarUrl,
  });

  final int userId;
  final String username;
  final String avatarUrl;

  factory WikiEditorSummary.fromJson(Map<String, dynamic> json) {
    return WikiEditorSummary(
      userId: (json['userId'] as num?)?.toInt() ?? 0,
      username: json['username'] as String? ?? '',
      avatarUrl: json['avatarUrl'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {'userId': userId, 'username': username, 'avatarUrl': avatarUrl};
  }
}

/// 替换命名空间编辑者请求体（`PUT /api/admin/wiki/namespaces/{name}/editors`）。
class WikiUpdateEditorsRequest {
  const WikiUpdateEditorsRequest({required this.userIds});

  final List<int> userIds;

  Map<String, dynamic> toJson() {
    return {'userIds': userIds};
  }
}

/// 管理端树中的一页（`GET /api/admin/wiki/tree`）。
class WikiAdminTreePage {
  const WikiAdminTreePage({
    required this.pageId,
    required this.path,
    required this.title,
    required this.sortOrder,
  });

  final int pageId;
  final String path;
  final String title;
  final int sortOrder;

  factory WikiAdminTreePage.fromJson(Map<String, dynamic> json) {
    return WikiAdminTreePage(
      pageId: (json['pageId'] as num?)?.toInt() ?? 0,
      path: json['path'] as String? ?? '',
      title: json['title'] as String? ?? '',
      sortOrder: (json['sortOrder'] as num?)?.toInt() ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'pageId': pageId,
      'path': path,
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

/// 树操作（`PUT /api/admin/wiki/tree` 的 ops 元素）。
class WikiTreeOp {
  const WikiTreeOp({
    required this.op,
    required this.pageId,
    this.parentPath,
    this.newPath,
    this.sortOrder,
  });

  /// move | rename | sort | delete
  final String op;
  final int pageId;
  final String? parentPath;
  final String? newPath;
  final int? sortOrder;

  Map<String, dynamic> toJson() {
    return {
      'op': op,
      'pageId': pageId,
      if (parentPath != null) 'parentPath': parentPath,
      if (newPath != null) 'newPath': newPath,
      if (sortOrder != null) 'sortOrder': sortOrder,
    };
  }
}

/// 树操作请求体（`PUT /api/admin/wiki/tree`）。
class WikiTreeOpsRequest {
  const WikiTreeOpsRequest({required this.ops});

  final List<WikiTreeOp> ops;

  Map<String, dynamic> toJson() {
    return {'ops': ops.map((op) => op.toJson()).toList()};
  }
}

/// 树操作统一 result（`{ok: true}`）。
class WikiTreeOpsResponse {
  const WikiTreeOpsResponse({required this.ok});

  final bool ok;

  factory WikiTreeOpsResponse.fromJson(Map<String, dynamic> json) {
    return WikiTreeOpsResponse(ok: json['ok'] as bool? ?? false);
  }

  Map<String, dynamic> toJson() {
    return {'ok': ok};
  }
}

/// 管理端修订条目（`GET /api/admin/wiki/revisions?status=`）。
class WikiAdminRevision {
  const WikiAdminRevision({
    required this.revisionId,
    required this.pageId,
    required this.path,
    required this.title,
    required this.content,
    required this.editorId,
    required this.editorName,
    required this.updatedAt,
  });

  final int revisionId;
  final int pageId;
  final String path;
  final String title;
  final String content;
  final int editorId;
  final String editorName;
  final String updatedAt;

  factory WikiAdminRevision.fromJson(Map<String, dynamic> json) {
    return WikiAdminRevision(
      revisionId: (json['revisionId'] as num?)?.toInt() ?? 0,
      pageId: (json['pageId'] as num?)?.toInt() ?? 0,
      path: json['path'] as String? ?? '',
      title: json['title'] as String? ?? '',
      content: json['content'] as String? ?? '',
      editorId: (json['editorId'] as num?)?.toInt() ?? 0,
      editorName: json['editorName'] as String? ?? '',
      updatedAt: json['updatedAt'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'revisionId': revisionId,
      'pageId': pageId,
      'path': path,
      'title': title,
      'content': content,
      'editorId': editorId,
      'editorName': editorName,
      'updatedAt': updatedAt,
    };
  }
}
