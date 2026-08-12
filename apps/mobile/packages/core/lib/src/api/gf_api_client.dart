import 'dart:async';

import 'package:dio/dio.dart';

import '../gen/page.dart';
import '../gen/response.dart';
import '../token/token_storage.dart';
import 'api_error.dart';

typedef JsonParser<T> = T Function(Object? json);
typedef TokenRenewedCallback = FutureOr<void> Function(String newToken);

/// GooseForum 后端 JSON API 客户端。
///
/// 统一处理:Bearer 令牌注入、New-Token 头滑动续期、401 会话失效、
/// 429 限流(Retry-After)、`{code, messageCode, params, result}` 错误协议。
/// 后端所有失败(HTTP 2xx 业务失败或 HTTP 4xx/5xx 的 `ResultStruct` envelope)
/// 都解析为带 messageCode 的 [ApiException];只有真正的传输层故障
/// (DNS、连接拒绝、超时)才是 [NetworkException]。
class GfApiClient {
  GfApiClient({
    required this.dio,
    required this.tokenStorage,
    this.baseUrl = defaultBaseUrl,
    this.onTokenRenewed,
    this.onUnauthorized,
  }) {
    dio.options.baseUrl = baseUrl;
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          final token = await tokenStorage.read();
          if (token != null && token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }
          handler.next(options);
        },
      ),
    );
  }

  /// 默认后端地址:Android 模拟器访问宿主机。
  static const defaultBaseUrl = 'http://10.0.2.2:5234';

  /// 页面级数据通道请求头(X-Goose-Page: true 直接返回 PagePayload JSON)。
  static const pageRequestHeader = 'X-Goose-Page';

  final Dio dio;
  final TokenStorage tokenStorage;
  final String baseUrl;

  /// 收到 New-Token 响应头时回调(滑动续期,上层负责持久化)。
  final TokenRenewedCallback? onTokenRenewed;

  /// 收到 401 时回调(会话失效,上层负责清理)。
  final void Function()? onUnauthorized;

  Future<T> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
    Map<String, dynamic>? headers,
    JsonParser<T>? parser,
  }) async {
    final response = await _request(
      () => dio.get(
        path,
        queryParameters: queryParameters,
        options: Options(headers: headers),
      ),
    );
    return _resolve(response, parser);
  }

  Future<T> post<T>(
    String path, {
    Object? body,
    Map<String, dynamic>? headers,
    JsonParser<T>? parser,
  }) async {
    final response = await _request(
      () => dio.post(
        path,
        data: body,
        options: Options(headers: headers),
      ),
    );
    return _resolve(response, parser);
  }

  Future<T> postMultipart<T>(
    String path, {
    required FormData formData,
    JsonParser<T>? parser,
  }) async {
    final response = await _request(() => dio.post(path, data: formData));
    return _resolve(response, parser);
  }

  Future<Response<dynamic>> _request(
    Future<Response<dynamic>> Function() run,
  ) async {
    try {
      return await run();
    } on DioException catch (e) {
      final status = e.response?.statusCode;
      if (status == 401) {
        onUnauthorized?.call();
        throw const UnauthorizedException();
      }
      if (status == 429) {
        final envelope = _tryParseEnvelope(e.response?.data);
        throw RateLimitException(
          fallbackMessage: 'Too many requests, please retry later',
          messageCode: envelope?.messageCode,
          params: envelope?.params,
          retryAfterSeconds: _retryAfter(e.response),
        );
      }
      if (status != null) {
        // 页面级数据通道的错误页(HTTP 404/500 + error.index PagePayload):
        // dio 视非 2xx 为异常,但 body 仍是可解析的页面负载,提升为
        // ApiException 以便页面层展示本地化错误文案。
        final data = e.response?.data;
        if (data is Map<String, dynamic> && !data.containsKey('code')) {
          final ApiException? pageError = _pageErrorException(status, data);
          if (pageError != null) throw pageError;
        }
        // 后端已应答的 4xx/5xx:解析 ResultStruct envelope,保留
        // status/messageCode/params,不再伪装成 NetworkException。
        // 无 body 或非 envelope 时退化为无 messageCode 的 ApiFailureException,
        // 仍是"服务端错误"而非网络故障。
        final envelope = _tryParseEnvelope(e.response?.data);
        throw ApiFailureException(
          fallbackMessage: 'Request failed with status $status',
          messageCode: envelope?.messageCode,
          params: envelope?.params,
          statusCode: status,
        );
      }
      // 无 HTTP 状态码:传输层故障(DNS、连接拒绝、超时等)。
      throw NetworkException(
        fallbackMessage: 'Network connection failed, please check your network',
      );
    }
  }

  Future<T> _resolve<T>(
    Response<dynamic> response,
    JsonParser<T>? parser,
  ) async {
    // Password login and TOTP return their session only through New-Token.
    // Persist it before resolving the request so callers can immediately read
    // TokenStorage without racing flutter_secure_storage.
    await _handleTokenRenewal(response);
    final statusCode = response.statusCode;
    final data = response.data;

    if (statusCode == 401) {
      onUnauthorized?.call();
      throw const UnauthorizedException();
    }
    if (statusCode == 429) {
      final envelope = _tryParseEnvelope(data);
      throw RateLimitException(
        fallbackMessage: 'Too many requests, please retry later',
        messageCode: envelope?.messageCode,
        params: envelope?.params,
        retryAfterSeconds: _retryAfter(response),
      );
    }
    if (data is! Map<String, dynamic>) {
      throw ApiException(
        fallbackMessage: 'Invalid response format',
        statusCode: statusCode,
      );
    }

    // 页面级数据通道(X-Goose-Page)直接返回裸 PagePayload:
    // 1. 非 2xx 时若为 error.index 页面(404/500 等),把 messageCode/params
    //    提升为 ApiException,让页面层展示本地化错误文案而不是原始字符串;
    // 2. 2xx 页面按成功处理。
    if (!data.containsKey('code')) {
      if (statusCode != null && (statusCode < 200 || statusCode >= 300)) {
        final ApiException? pageError = _pageErrorException(statusCode, data);
        if (pageError != null) throw pageError;
      }
      if (parser != null) {
        return parser(data);
      }
      return data as T;
    }

    final envelope = GfResponse<Object?>.fromJson(data, (json) => json);
    if (!envelope.isSuccess) {
      throw ApiException(
        fallbackMessage: 'Request failed',
        messageCode: envelope.messageCode,
        params: envelope.params,
        statusCode: statusCode,
      );
    }
    final result = envelope.result;
    if (parser != null) {
      return parser(result);
    }
    return result as T;
  }

  /// 把 page-channel 的 error.index payload 转成 [ApiException]。
  /// payload 形如 `{component: "error.index", props: {code, title, messageCode, params}}`。
  /// 非 error.index 页面(或 props 缺失)返回 null,调用方按普通传输错误处理;
  /// 解析出的 messageCode 缺失时退回通用的加载失败文案,绝不把原始字符串抛给 UI。
  ApiException? _pageErrorException(int statusCode, Map<String, dynamic> data) {
    if (data['component'] != PageComponent.error) return null;
    final Object? props = data['props'];
    if (props is! Map<String, dynamic>) {
      return ApiException(
        fallbackMessage: 'Failed to load',
        statusCode: statusCode,
      );
    }
    final String? code = (props['messageCode'] as String?)?.trim();
    if (code == null || code.isEmpty) {
      return ApiException(
        fallbackMessage: 'Failed to load',
        statusCode: statusCode,
      );
    }
    return ApiException(
      fallbackMessage: 'Failed to load',
      messageCode: code,
      params: (props['params'] as Map<String, dynamic>?)?.cast<String, dynamic>(),
      statusCode: statusCode,
    );
  }

  GfResponse<Object?>? _tryParseEnvelope(Object? data) {
    if (data is! Map<String, dynamic>) return null;
    try {
      return GfResponse<Object?>.fromJson(data, (json) => json);
    } catch (_) {
      return null;
    }
  }

  Future<void> _handleTokenRenewal(Response<dynamic> response) async {
    final newToken = response.headers.value('New-Token');
    if (newToken != null && newToken.trim().isNotEmpty) {
      final callback = onTokenRenewed;
      if (callback != null) {
        await callback(newToken.trim());
      }
    }
  }

  int? _retryAfter(Response<dynamic>? response) {
    final value = response?.headers.value('Retry-After');
    final seconds = int.tryParse(value ?? '');
    return (seconds != null && seconds > 0) ? seconds : null;
  }
}
