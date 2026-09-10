import '../../gen/chat.dart';
import '../gf_api_client.dart';

/// 私信(IM)接口。
class ChatRepository {
  ChatRepository(this._client);

  final GfApiClient _client;

  /// 发送私信,成功返回会话 id(result 为 `{convId}` 映射)。
  Future<int> sendMessage({
    required int peerId,
    required String content,
    int msgType = 1,
  }) {
    return _client.post<int>(
      '/api/forum/chat/send',
      body: {'peerId': peerId, 'content': content, 'msgType': msgType},
      parser: (json) {
        if (json is int) return json;
        return (json as Map<String, dynamic>)['convId'] as int? ?? 0;
      },
    );
  }

  /// 拉取会话消息(beforeId/afterId 游标,limit 页大小)。
  Future<ChatMessagesResponse> getMessages({
    required int convId,
    int beforeId = 0,
    int afterId = 0,
    int limit = 30,
  }) {
    return _client.post<ChatMessagesResponse>(
      '/api/forum/chat/messages',
      body: {'convId': convId, 'beforeId': beforeId, 'afterId': afterId, 'limit': limit},
      parser: (json) => ChatMessagesResponse.fromJson(json as Map<String, dynamic>),
    );
  }

  Future<bool> markRead({required int convId}) async {
    await _client.post<Object?>('/api/forum/chat/mark-read', body: {'convId': convId});
    return true;
  }
}
