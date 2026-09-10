/// API 错误类型。
///
/// 后端错误协议:HTTP 2xx + `{code != 0, messageCode, params}` 表示业务失败;
/// 中间件/控制器直接以 HTTP 4xx/5xx 返回同一 `ResultStruct` envelope
/// (`{code: 1, messageCode, params, result: null}`,见
/// `component.FailDataCode`);HTTP 429 限流(带 Retry-After);HTTP 401 未授权;
/// 只有真正的传输层故障(DNS、连接拒绝、超时等)才是 [NetworkException]。
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

/// HTTP 4xx/5xx 且响应体可解析为 `ResultStruct` envelope 的服务端错误
/// (如 OIDC 400/403、冻结 `permission.userFrozen`、500)。区别于
/// [NetworkException]:这是**后端已应答**的业务失败,messageCode 稳定。
class ApiFailureException extends ApiException {
  const ApiFailureException({
    required super.fallbackMessage,
    super.messageCode,
    super.params,
    super.statusCode,
  });

  @override
  String toString() {
    final code = messageCode == null ? '' : ' ($messageCode)';
    return 'ApiFailureException: $fallbackMessage$code';
  }
}

/// 网络层失败(DNS、连接拒绝、超时等),后端未产生业务响应。
class NetworkException extends ApiException {
  const NetworkException({
    required super.fallbackMessage,
    super.statusCode,
    super.messageCode,
    super.params,
  });
}
