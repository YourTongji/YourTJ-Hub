import 'package:dio/dio.dart';

import '../gen/response.dart';
import '../token/token_storage.dart';
import 'api_error.dart';

typedef JsonParser<T> = T Function(Object? json);

/// GooseForum 后端 JSON API 客户端。
///
/// 统一处理:Bearer 令牌注入、New-Token 头滑动续期、401 会话失效、
/// 429 限流(Retry-After)、`{code, messageCode, params, result}` 错误协议。
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
  final void Function(String newToken)? onTokenRenewed;

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
      throw NetworkException(
        fallbackMessage: 'Network connection failed, please check your network',
        statusCode: status,
      );
    }
  }

  T _resolve<T>(Response<dynamic> response, JsonParser<T>? parser) {
    _handleTokenRenewal(response);
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

    // 页面级数据通道(X-Goose-Page)直接返回裸 PagePayload,无 code 字段,按成功处理。
    if (!data.containsKey('code')) {
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

  GfResponse<Object?>? _tryParseEnvelope(Object? data) {
    if (data is! Map<String, dynamic>) return null;
    try {
      return GfResponse<Object?>.fromJson(data, (json) => json);
    } catch (_) {
      return null;
    }
  }

  void _handleTokenRenewal(Response<dynamic> response) {
    final newToken = response.headers.value('New-Token');
    if (newToken != null && newToken.trim().isNotEmpty) {
      onTokenRenewed?.call(newToken.trim());
    }
  }

  int? _retryAfter(Response<dynamic>? response) {
    final value = response?.headers.value('Retry-After');
    final seconds = int.tryParse(value ?? '');
    return (seconds != null && seconds > 0) ? seconds : null;
  }
}
