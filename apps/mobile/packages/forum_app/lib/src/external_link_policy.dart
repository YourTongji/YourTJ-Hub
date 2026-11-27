import 'dart:convert';

import 'package:core/core.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app_config.dart';
import 'link_navigation.dart';
import 'providers.dart';

/// 外链免确认白名单同步（issue #1200）。
///
/// - 启动时先从本地按站点 origin 缓存恢复（无网也保持上次拉取到的白名单），
///   再后台拉取 `GET /api/external-link-policy` 刷新并回写；
/// - 失败静默：保留当前清单；首次未配置或离线时维持「全部站外链接需确认」的安全默认。
class ExternalLinkPolicyController extends Notifier<void> {
  static const String _cacheKeyPrefix = 'external_link_policy_cache:';

  String _cacheKey() {
    final base = AppConfig.apiBaseUrl.isNotEmpty
        ? AppConfig.apiBaseUrl
        : GfApiClient.defaultBaseUrl;
    final origin = Uri.tryParse(base)?.origin ?? base;
    return '$_cacheKeyPrefix$origin';
  }

  @override
  void build() {
    _restoreAndRefresh();
  }

  Future<void> _restoreAndRefresh() async {
    final cacheKey = _cacheKey();
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString(cacheKey);
      if (raw != null) {
        final decoded = jsonDecode(raw);
        if (decoded is List) {
          LinkNavigation.applyTrustedDomains(
            decoded.whereType<String>().toList(growable: false),
          );
        }
      }
    } catch (_) {
      // 无本地存储或损坏时静默回退默认安全清单。
    }
    try {
      final List<String>? domains = await ref
          .read(externalLinkPolicyRepositoryProvider)
          .fetchTrustedDomains();
      // 旧后端无该端点时返回 null：保留当前清单，不改变豁免面。
      if (domains == null) return;
      LinkNavigation.applyTrustedDomains(domains);
      try {
        final prefs = await SharedPreferences.getInstance();
        await prefs.setString(cacheKey, jsonEncode(domains));
      } catch (_) {
        // 持久化失败不影响本次会话。
      }
    } catch (_) {
      // 传输层失败（离线/超时）：保留当前清单，不放大豁免面。
    }
  }
}

final externalLinkPolicyProvider =
    NotifierProvider<ExternalLinkPolicyController, void>(
      ExternalLinkPolicyController.new,
    );
