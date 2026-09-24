import 'package:freezed_annotation/freezed_annotation.dart';

import 'user.dart';

part 'chat.freezed.dart';
part 'chat.g.dart';

@freezed
abstract class ChatItemPayload with _$ChatItemPayload {
  const factory ChatItemPayload({
    required int id,
    required int peerId,
    required String peerUsername,
    required String peerAvatar,
    required String lastMsg,
    required String lastMsgTime,
    required int unreadCount,
    required int convId,
    required String peerUrl,
  }) = _ChatItemPayload;

  factory ChatItemPayload.fromJson(Map<String, dynamic> json) =>
      _$ChatItemPayloadFromJson(json);
}

@freezed
abstract class MessagesPageProps with _$MessagesPageProps {
  const factory MessagesPageProps({
    required List<ChatItemPayload> conversations,
    required List<UserConnectionPayload> suggestedUsers,
  }) = _MessagesPageProps;

  factory MessagesPageProps.fromJson(Map<String, dynamic> json) =>
      _$MessagesPagePropsFromJson(json);
}

@freezed
abstract class ChatMessagePayload with _$ChatMessagePayload {
  const factory ChatMessagePayload({
    required int id,
    required int senderId,
    required String content,
    required int msgType,
    required int isRead,
    required String createdAt,
    required bool isSelf,
  }) = _ChatMessagePayload;

  factory ChatMessagePayload.fromJson(Map<String, dynamic> json) =>
      _$ChatMessagePayloadFromJson(json);
}

@freezed
abstract class ChatMessagesResponse with _$ChatMessagesResponse {
  const factory ChatMessagesResponse({
    required List<ChatMessagePayload> list,
    required bool hasMoreBefore,
    required bool hasMoreAfter,
    required int nextBeforeId,
    required int latestId,
  }) = _ChatMessagesResponse;

  factory ChatMessagesResponse.fromJson(Map<String, dynamic> json) =>
      _$ChatMessagesResponseFromJson(json);
}

/// Acknowledgement of the exact incoming IDs the client displayed.
/// The stored unread counter decreases only for newly read rows.
@freezed
abstract class ChatVisibleReadResult with _$ChatVisibleReadResult {
  const factory ChatVisibleReadResult({
    required int convId,
    required List<int> acknowledgedMessageIds,
    required int unreadCount,
  }) = _ChatVisibleReadResult;

  factory ChatVisibleReadResult.fromJson(Map<String, dynamic> json) =>
      _$ChatVisibleReadResultFromJson(json);
}

@freezed
abstract class ChatMessageReadState with _$ChatMessageReadState {
  const factory ChatMessageReadState({required int id, required int isRead}) =
      _ChatMessageReadState;

  factory ChatMessageReadState.fromJson(Map<String, dynamic> json) =>
      _$ChatMessageReadStateFromJson(json);
}

/// Read flags and the stored unread counter from one conversation lock scope.
/// This bounded lookup does not recount historical unread messages.
@freezed
abstract class ChatMessageReadStatesResult with _$ChatMessageReadStatesResult {
  const factory ChatMessageReadStatesResult({
    required List<ChatMessageReadState> items,
    required int unreadCount,
  }) = _ChatMessageReadStatesResult;

  factory ChatMessageReadStatesResult.fromJson(Map<String, dynamic> json) =>
      _$ChatMessageReadStatesResultFromJson(json);
}
