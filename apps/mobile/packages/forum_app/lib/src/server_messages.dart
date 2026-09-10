import 'package:core/core.dart';

import '../l10n/app_localizations.dart';

import 'server_message_catalog.dart';

/// 把 [error] 解析为用户可读的本地化文案。
///
/// - [ApiException] 且 messageCode 命中本地化目录:返回目录文案;
/// - [ApiException] 但 messageCode 未知/缺失/未命中:统一回退
///   [AppLocalizations.commonLoadFailed](zh: 加载失败 / en: Failed to load),
///   绝不返回英文硬编码 fallbackMessage;
/// - 其它错误(页面自身的业务错误文案等):保留原样。
/// 任何情况下都不会把原始 messageCode 或 `ApiException: ...` 前缀展示给用户。
String resolveErrorMessage(AppLocalizations l10n, Object error) {
  if (error is! ApiException) return error.toString();

  final String? code = error.messageCode?.trim();
  if (code != null && code.isNotEmpty) {
    final Map<String, String> catalog =
        serverMessageCatalog[l10n.localeName] ??
        serverMessageCatalog['en'] ??
        const <String, String>{};
    final String? localized = catalog[code];
    if (localized != null) {
      final params = <String, dynamic>{
        if (error is RateLimitException && error.retryAfterSeconds != null)
          'retryAfterSeconds': error.retryAfterSeconds,
        ...?error.params,
      };
      var missingParameter = false;
      final resolved = localized.replaceAllMapped(RegExp(r'\{(\w+)\}'), (
        match,
      ) {
        final value = params[match.group(1)];
        if (value == null ||
            (value is! String && value is! num && value is! bool)) {
          missingParameter = true;
          return '';
        }
        return value.toString();
      });
      if (!missingParameter) return resolved;
    }
  }

  return l10n.commonLoadFailed;
}
