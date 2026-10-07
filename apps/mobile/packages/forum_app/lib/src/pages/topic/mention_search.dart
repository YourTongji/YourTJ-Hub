/// @mention 服务端候选搜索桥接（issue #565）。
///
/// 使用专用 mention-targets API，纳入启用的 Agent 但不扩展普通用户搜索；
/// 以 Riverpod provider 注入 mention 会话，测试可 override。
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
    final targets = await ref
        .read(topicRepositoryProvider)
        .mentionTargets(query: query, limit: 20);
    return <MentionUser>[
      for (final target in targets)
        MentionUser(
          id: target.userId,
          username: target.username,
          nickname: target.nickname,
          avatarUrl: resolveApiAssetUrl(target.avatarUrl),
          actorType: target.actorType,
        ),
    ];
  };
});
