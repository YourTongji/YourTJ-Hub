import '../../gen/layout.dart';
import '../../gen/notification.dart';
import '../gf_api_client.dart';

import 'package:dio/dio.dart';

/// 通知与未读状态接口。
class NotificationRepository {
  NotificationRepository(this._client) : _sessionToken = null;
  NotificationRepository._session(this._client, this._sessionToken);

  final GfApiClient _client;
  final String? _sessionToken;
  Map<String, dynamic>? get _headers =>
      _sessionToken == null ? null : {'Authorization': 'Bearer $_sessionToken'};

  /// Freeze the submitting account before an asynchronous preference save.
  Future<NotificationRepository> forSession() async {
    final token = _sessionToken ?? await _client.tokenStorage.read();
    if (token == null || token.isEmpty) throw StateError('No session');
    return NotificationRepository._session(_client, token);
  }

  Future<NotificationPreferences> preferences({CancelToken? cancelToken}) =>
      _client.get<NotificationPreferences>(
        '/api/forum/notification-preferences',
        headers: _headers,
        cancelToken: cancelToken,
        parser: (json) => NotificationPreferences.fromJson(
          Map<String, dynamic>.from(json as Map),
        ),
      );

  Future<NotificationPreferences> updatePreferences(UnreadBadgeStyle style) =>
      _client.put<NotificationPreferences>(
        '/api/forum/notification-preferences',
        headers: _headers,
        body: {'badgeStyle': style.name},
        parser: (json) => NotificationPreferences.fromJson(
          Map<String, dynamic>.from(json as Map),
        ),
      );

  /// 通知列表(filter: all | unread;cursor 游标;limit 页大小)。
  Future<NotificationListResponse> fetchNotifications({
    String filter = 'all',
    String category = 'all',
    int cursor = 0,
    int limit = 20,
    CancelToken? cancelToken,
  }) {
    return _client.get<NotificationListResponse>(
      '/api/forum/notifications',
      headers: _headers,
      cancelToken: cancelToken,
      queryParameters: {
        'filter': filter,
        'category': category,
        'cursor': cursor,
        'limit': limit,
      },
      parser: (json) =>
          NotificationListResponse.fromJson(json as Map<String, dynamic>),
    );
  }

  Future<bool> markNotificationRead({required int notificationId}) async {
    await _client.post<Object?>(
      '/api/forum/notification/mark-read',
      headers: _headers,
      body: {'notificationId': notificationId},
    );
    return true;
  }

  Future<bool> markAllNotificationsRead() async {
    await _client.post<Object?>(
      '/api/forum/notification/mark-all-read',
      headers: _headers,
    );
    return true;
  }

  /// 未读状态(通知/私信/待审核)。
  Future<UnreadStatusPayload> getUnreadStatus({CancelToken? cancelToken}) {
    return _client.get<UnreadStatusPayload>(
      '/api/forum/unread-status',
      headers: _headers,
      cancelToken: cancelToken,
      parser: (json) =>
          UnreadStatusPayload.fromJson(json as Map<String, dynamic>),
    );
  }
}
