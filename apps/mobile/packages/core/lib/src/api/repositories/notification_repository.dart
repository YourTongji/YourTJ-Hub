import '../../gen/layout.dart';
import '../../gen/notification.dart';
import '../gf_api_client.dart';

/// 通知与未读状态接口。
class NotificationRepository {
  NotificationRepository(this._client);

  final GfApiClient _client;

  /// 通知列表(filter: all | unread;cursor 游标;limit 页大小)。
  Future<NotificationListResponse> fetchNotifications({
    String filter = 'all',
    int cursor = 0,
    int limit = 20,
  }) {
    return _client.get<NotificationListResponse>(
      '/api/forum/notifications',
      queryParameters: {'filter': filter, 'cursor': cursor, 'limit': limit},
      parser: (json) => NotificationListResponse.fromJson(json as Map<String, dynamic>),
    );
  }

  Future<bool> markNotificationRead({required int notificationId}) async {
    await _client.post<Object?>(
      '/api/forum/notification/mark-read',
      body: {'notificationId': notificationId},
    );
    return true;
  }

  Future<bool> markAllNotificationsRead() async {
    await _client.post<Object?>('/api/forum/notification/mark-all-read');
    return true;
  }

  /// 未读状态(通知/私信/待审核)。
  Future<UnreadStatusPayload> getUnreadStatus() {
    return _client.get<UnreadStatusPayload>(
      '/api/forum/unread-status',
      parser: (json) => UnreadStatusPayload.fromJson(json as Map<String, dynamic>),
    );
  }
}
