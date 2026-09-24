import 'dart:async';

import 'package:dio/dio.dart';

import 'forum_sse.dart';
import 'gf_api_client.dart';

class ForumRealtimeHttpException implements Exception {
  const ForumRealtimeHttpException(this.statusCode, {this.retryAfter});

  final int statusCode;
  final Duration? retryAfter;
}

class ForumRealtimeConnection {
  ForumRealtimeConnection(this.events, this._cancelToken);

  final Stream<ForumSseFrame> events;
  final CancelToken _cancelToken;

  void close() {
    if (!_cancelToken.isCancelled) _cancelToken.cancel('SSE connection closed');
  }
}

/// Owns one HTTP stream; the caller owns retry, lifecycle and REST resync.
/// The explicit bearer token binds the request to the session that opened it,
/// even when another account logs in while the connection is pending.
class ForumRealtimeTransport {
  const ForumRealtimeTransport(this._client);

  final GfApiClient _client;

  Future<ForumRealtimeConnection> connect({
    required String token,
    required CancelToken cancelToken,
    void Function()? onActivity,
  }) async {
    final Response<ResponseBody> response;
    try {
      response = await _client.dio.get<ResponseBody>(
        '/api/forum/events',
        cancelToken: cancelToken,
        options: Options(
          responseType: ResponseType.stream,
          receiveTimeout: Duration.zero,
          validateStatus: (_) => true,
          headers: {
            'Authorization': 'Bearer $token',
            'Accept': 'text/event-stream',
          },
        ),
      );
    } on DioException {
      rethrow;
    }
    final newToken = response.headers.value('new-token');
    if (newToken != null && newToken.isNotEmpty) {
      await _client.onTokenRenewed?.call(newToken);
    }
    final status = response.statusCode ?? 0;
    if (status != 200) {
      if (status == 401) _client.onUnauthorized?.call();
      cancelToken.cancel('SSE rejected with HTTP $status');
      throw ForumRealtimeHttpException(
        status,
        retryAfter: _retryAfter(response.headers.value('retry-after')),
      );
    }
    if (!response.headers
        .value(Headers.contentTypeHeader)
        .toString()
        .toLowerCase()
        .contains('text/event-stream')) {
      cancelToken.cancel('SSE content type mismatch');
      throw const FormatException('SSE content type mismatch');
    }
    final body = response.data;
    if (body == null) {
      cancelToken.cancel('SSE response has no body');
      throw const FormatException('SSE response has no body');
    }
    return ForumRealtimeConnection(
      decodeForumSse(
        // Lifecycle shutdown cancels the decoder and HTTP request together.
        // The decoder may still be awaiting bytes while its cancel Future is
        // pending: consume only the expected transport cancellation here so
        // that Future completes normally. Real network errors still propagate.
        body.stream.handleError(
          (Object _) {},
          test: (error) => error is DioException && CancelToken.isCancel(error),
        ),
        onActivity: onActivity,
      ),
      cancelToken,
    );
  }

  static Duration? _retryAfter(String? value) {
    final seconds = int.tryParse(value ?? '');
    if (seconds == null || seconds < 0) return null;
    return Duration(seconds: seconds);
  }
}
