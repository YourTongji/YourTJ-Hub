import 'package:core/core.dart';

import '../l10n/app_localizations.dart';

/// 服务端 messageCode → 本地化文案目录。
///
/// 镜像 web 端 `resolveApiMessage` 的 flat lookup(`serverMessages.<code>`),
/// 仅覆盖页面级错误常见码;未知码统一回退到 [AppLocalizations.commonLoadFailed]。
/// 文案与 `apps/gooseforum/app/bundles/i18n/locales/*.json` 保持一致。
const Map<String, Map<String, String>> _serverMessages = {
  'zh': <String, String>{
    'page.notFound': '页面不存在，或已经被删除。',
    'route.notFound': '路由未定义，请确认 URL 和请求方法是否正确。',
    'topic.notFound': '话题不存在，或已经被删除。',
    'user.notFound': '用户不存在。',
    'auth.required': '请先登录后再访问此页面。',
    'permission.denied': '当前用户没有执行该操作的权限。',
    'permission.userFrozen': '您的账号已被封禁，无法继续此操作。',
    'common.operation.failed': '操作失败，请稍后重试。',
    'oauth.callback.failed': '第三方账号登录回调失败，请稍后重试。',
    'oauth.process.failed': '第三方账号登录处理失败，请稍后重试。',
    'oauth.activation.updateFailed': '登录账号激活状态更新失败，请稍后重试。',
    'oauth.token.failed': '登录凭据生成失败，请稍后重试。',
  },
  'en': <String, String>{
    'page.notFound': 'The page does not exist or has been deleted.',
    'route.notFound': 'Route not found. Please check the URL and request method.',
    'topic.notFound': 'The topic does not exist or has been deleted.',
    'user.notFound': 'The user does not exist.',
    'auth.required': 'Please sign in to view this page.',
    'permission.denied': 'You do not have permission to perform this action.',
    'permission.userFrozen': 'Your account is banned and cannot continue.',
    'common.operation.failed': 'Operation failed. Please try again later.',
    'oauth.callback.failed': 'OAuth callback failed. Please try again later.',
    'oauth.process.failed': 'OAuth sign-in failed. Please try again later.',
    'oauth.activation.updateFailed':
        'Failed to update the account activation status. Please try again later.',
    'oauth.token.failed': 'Failed to create sign-in credentials. Please try again later.',
  },
};

/// 把 [error] 解析为用户可读的本地化文案。
///
/// - [ApiException] 且携带 messageCode:flat lookup 本地化目录,未知码回退
///   到其 fallbackMessage(不含原始 messageCode);
/// - [ApiException] 无 messageCode(网络错误等):直接使用 fallbackMessage;
/// - 其它错误(页面自身的业务错误文案等):保留原样。
/// 任何情况下都不会把原始 messageCode 或 `ApiException: ...` 前缀展示给用户。
String resolveErrorMessage(AppLocalizations l10n, Object error) {
  if (error is! ApiException) return error.toString();

  final String? code = error.messageCode?.trim();
  if (code == null || code.isEmpty) {
    return error.fallbackMessage;
  }

  final Map<String, String> catalog = _serverMessages[l10n.localeName] ??
      _serverMessages['en'] ??
      const <String, String>{};
  return catalog[code] ?? error.fallbackMessage;
}
