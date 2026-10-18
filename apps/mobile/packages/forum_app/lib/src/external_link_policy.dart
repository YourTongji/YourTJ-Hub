import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'link_navigation.dart';
import 'providers.dart';

/// 外链免确认白名单同步（issue #1200）。
///
/// - 启动时拉取 `GET /api/external-link-policy`（管理员配置，与 Web 端 SSR
///   payload `layout.site.trustedExternalDomains` 同源）并写入 [LinkNavigation]；
/// - 失败静默：保留当前清单；未配置或离线时维持「全部站外链接需确认」的安全默认。
class ExternalLinkPolicyController extends Notifier<void> {
  @override
  void build() {
    _refresh();
  }

  Future<void> _refresh() async {
    try {
      final List<String>? domains = await ref
          .read(externalLinkPolicyRepositoryProvider)
          .fetchTrustedDomains();
      // 旧后端无该端点时返回 null：保留当前清单，不改变豁免面。
      if (domains == null) return;
      LinkNavigation.applyTrustedDomains(domains);
    } catch (_) {
      // 传输层失败（离线/超时）：保留当前清单，不放大豁免面。
    }
  }
}

final externalLinkPolicyProvider =
    NotifierProvider<ExternalLinkPolicyController, void>(
      ExternalLinkPolicyController.new,
    );
