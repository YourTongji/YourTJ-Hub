/// @mention 服务端候选搜索桥接（issue #565）。
///
/// 复用聚合搜索 `/api/forum/search?scope=users`（与 Web `searchForumUsers`
/// 同契约）；以 Riverpod provider 注入 mention 会话，测试可 override。
/// 排序/去重/上限由会话层（mention_session.dart）统一处理，这里只透传。
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../asset_url.dart';
import '../../providers.dart';
import 'mention_session.dart';

/// 查询用户候选：query → 候选列表（服务端返回顺序）。
typedef MentionUserSearch = Future<List<MentionUser>> Function(String query);

final mentionUserSearchProvider = Provider<MentionUserSearch>((ref) {
  return (query) async {
    final page = await ref
        .read(topicRepositoryProvider)
        .search(query: query, scope: 'users');
    // 搜索服务不可用或 users scope 失败时按失败抛出：会话层保留本地候选并
    // 提示"搜索失败可继续输入"，避免索引不可用被伪装成"没有匹配的用户"。
    if (page.searchUnavailable == true ||
        (page.failedScopes?.contains('users') ?? false)) {
      throw Exception('search users unavailable');
    }
    return <MentionUser>[
      for (final user in page.users)
        MentionUser(
          id: user.id,
          username: user.username,
          nickname: user.nickname,
          avatarUrl: resolveApiAssetUrl(user.avatarUrl),
        ),
    ];
  };
});
