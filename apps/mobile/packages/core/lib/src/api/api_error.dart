/// API 错误类型。
///
/// 后端错误协议:HTTP 200 + `{code != 0, messageCode, params}` 表示业务失败;
/// HTTP 429 限流(带 Retry-After);HTTP 401 未授权;其它为网络/传输错误。
library;

/// 业务失败(code != 0)或传输层错误(messageCode 为空)。
class ApiException implements Exception {
  const ApiException({
    required this.fallbackMessage,
    this.messageCode,
    this.params,
    this.statusCode,
  });

  /// 后端稳定的 messageCode(如 `auth.login.invalidRequest`),可映射为本地化 key。
  final String? messageCode;

  final Map<String, dynamic>? params;

  final String fallbackMessage;

  final int? statusCode;

  /// 本地化 key 形如 `server.<messageCode>`;无 messageCode 时返回 fallbackMessage。
  String get messageKey =>
      messageCode == null ? fallbackMessage : 'server.$messageCode';

  @override
  String toString() {
    final code = messageCode == null ? '' : ' ($messageCode)';
    return 'ApiException: $fallbackMessage$code';
  }
}

/// HTTP 429 限流。
class RateLimitException extends ApiException {
  const RateLimitException({
    required super.fallbackMessage,
    super.messageCode,
    super.params,
    super.statusCode = 429,
    this.retryAfterSeconds,
  });

  /// Retry-After 响应头秒数;缺失时为 null。
  final int? retryAfterSeconds;

  @override
  String toString() =>
      'RateLimitException: $fallbackMessage'
      '${retryAfterSeconds == null ? '' : ' (retry after ${retryAfterSeconds}s)'}';
}

/// HTTP 401 未授权(会话失效,应清除本地令牌)。
class UnauthorizedException extends ApiException {
  const UnauthorizedException({
    super.fallbackMessage = 'Not signed in or session expired',
  }) : super(statusCode: 401);
}

/// 网络层失败(DNS、连接拒绝、超时等)。
class NetworkException extends ApiException {
  const NetworkException({required super.fallbackMessage, super.statusCode});
}
