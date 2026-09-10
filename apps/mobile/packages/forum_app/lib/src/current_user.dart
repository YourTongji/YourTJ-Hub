import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'providers.dart';

/// 当前登录用户状态。
class CurrentUser {
  const CurrentUser({required this.id, required this.username});

  final int id;
  final String username;
}

/// 从 JWT payload 解析当前登录用户 id(HS256,仅解码不验签;
/// 服务端校验在每次请求时进行)。
///
/// JWT 结构:header.payload.signature;payload 为 base64url JSON。
/// 后端 CustomClaims 无 JSON tag → 字段名为 `UserId`。
int? _parseUserIdFromToken(String token) {
  try {
    final parts = token.split('.');
    if (parts.length != 3) return null;
    final payload = jsonDecode(
      utf8.decode(base64Url.decode(base64Url.normalize(parts[1]))),
    ) as Map<String, dynamic>;
    final raw = payload['UserId'];
    if (raw is num) return raw.toInt();
    if (raw is String) return int.tryParse(raw);
    return null;
  } catch (_) {
    return null;
  }
}

/// 当前登录用户;未登录时为 null。
///
/// 从 TokenStorage 读取会话令牌并解析 UserId;用户名暂取 token 中的
/// username 字段(后端 CustomClaims 无该字段时为 null,页面回退到
/// "我的" 默认展示)。
final currentUserProvider = FutureProvider<CurrentUser?>((ref) async {
  final storage = ref.watch(tokenStorageProvider);
  final token = await storage.read();
  if (token == null || token.isEmpty) return null;
  final int? id = _parseUserIdFromToken(token);
  if (id == null) return null;
  return CurrentUser(id: id, username: '');
});
