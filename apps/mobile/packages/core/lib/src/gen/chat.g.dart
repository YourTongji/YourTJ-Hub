// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'chat.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$ChatItemPayloadImpl _$$ChatItemPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$ChatItemPayloadImpl(
  id: (json['id'] as num).toInt(),
  peerId: (json['peerId'] as num).toInt(),
  peerUsername: json['peerUsername'] as String,
  peerNickname: json['peerNickname'] as String?,
  peerAvatar: json['peerAvatar'] as String,
  lastMsg: json['lastMsg'] as String,
  lastMsgTime: json['lastMsgTime'] as String,
  unreadCount: (json['unreadCount'] as num).toInt(),
  convId: (json['convId'] as num).toInt(),
  peerUrl: json['peerUrl'] as String,
);

Map<String, dynamic> _$$ChatItemPayloadImplToJson(
  _$ChatItemPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'peerId': instance.peerId,
  'peerUsername': instance.peerUsername,
  'peerNickname': instance.peerNickname,
  'peerAvatar': instance.peerAvatar,
  'lastMsg': instance.lastMsg,
  'lastMsgTime': instance.lastMsgTime,
  'unreadCount': instance.unreadCount,
  'convId': instance.convId,
  'peerUrl': instance.peerUrl,
};

_$MessagesPagePropsImpl _$$MessagesPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$MessagesPagePropsImpl(
  conversations: (json['conversations'] as List<dynamic>)
      .map((e) => ChatItemPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  suggestedUsers: (json['suggestedUsers'] as List<dynamic>)
      .map((e) => UserConnectionPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$$MessagesPagePropsImplToJson(
  _$MessagesPagePropsImpl instance,
) => <String, dynamic>{
  'conversations': instance.conversations,
  'suggestedUsers': instance.suggestedUsers,
};

_$ChatMessagePayloadImpl _$$ChatMessagePayloadImplFromJson(
  Map<String, dynamic> json,
) => _$ChatMessagePayloadImpl(
  id: (json['id'] as num).toInt(),
  senderId: (json['senderId'] as num).toInt(),
  content: json['content'] as String,
  msgType: (json['msgType'] as num).toInt(),
  isRead: (json['isRead'] as num).toInt(),
  createdAt: json['createdAt'] as String,
  isSelf: json['isSelf'] as bool,
  forwarded: json['forwarded'] == null
      ? null
      : ChatForwardBundle.fromJson(json['forwarded'] as Map<String, dynamic>),
);

Map<String, dynamic> _$$ChatMessagePayloadImplToJson(
  _$ChatMessagePayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'senderId': instance.senderId,
  'content': instance.content,
  'msgType': instance.msgType,
  'isRead': instance.isRead,
  'createdAt': instance.createdAt,
  'isSelf': instance.isSelf,
  'forwarded': instance.forwarded,
};

_$ChatMessagesResponseImpl _$$ChatMessagesResponseImplFromJson(
  Map<String, dynamic> json,
) => _$ChatMessagesResponseImpl(
  list: (json['list'] as List<dynamic>)
      .map((e) => ChatMessagePayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  hasMoreBefore: json['hasMoreBefore'] as bool,
  hasMoreAfter: json['hasMoreAfter'] as bool,
  nextBeforeId: (json['nextBeforeId'] as num).toInt(),
  latestId: (json['latestId'] as num).toInt(),
);

Map<String, dynamic> _$$ChatMessagesResponseImplToJson(
  _$ChatMessagesResponseImpl instance,
) => <String, dynamic>{
  'list': instance.list,
  'hasMoreBefore': instance.hasMoreBefore,
  'hasMoreAfter': instance.hasMoreAfter,
  'nextBeforeId': instance.nextBeforeId,
  'latestId': instance.latestId,
};

_$ChatVisibleReadResultImpl _$$ChatVisibleReadResultImplFromJson(
  Map<String, dynamic> json,
) => _$ChatVisibleReadResultImpl(
  convId: (json['convId'] as num).toInt(),
  acknowledgedMessageIds: (json['acknowledgedMessageIds'] as List<dynamic>)
      .map((e) => (e as num).toInt())
      .toList(),
  unreadCount: (json['unreadCount'] as num).toInt(),
);

Map<String, dynamic> _$$ChatVisibleReadResultImplToJson(
  _$ChatVisibleReadResultImpl instance,
) => <String, dynamic>{
  'convId': instance.convId,
  'acknowledgedMessageIds': instance.acknowledgedMessageIds,
  'unreadCount': instance.unreadCount,
};

_$ChatMessageReadStateImpl _$$ChatMessageReadStateImplFromJson(
  Map<String, dynamic> json,
) => _$ChatMessageReadStateImpl(
  id: (json['id'] as num).toInt(),
  isRead: (json['isRead'] as num).toInt(),
);

Map<String, dynamic> _$$ChatMessageReadStateImplToJson(
  _$ChatMessageReadStateImpl instance,
) => <String, dynamic>{'id': instance.id, 'isRead': instance.isRead};

_$ChatMessageReadStatesResultImpl _$$ChatMessageReadStatesResultImplFromJson(
  Map<String, dynamic> json,
) => _$ChatMessageReadStatesResultImpl(
  items: (json['items'] as List<dynamic>)
      .map((e) => ChatMessageReadState.fromJson(e as Map<String, dynamic>))
      .toList(),
  unreadCount: (json['unreadCount'] as num).toInt(),
);

Map<String, dynamic> _$$ChatMessageReadStatesResultImplToJson(
  _$ChatMessageReadStatesResultImpl instance,
) => <String, dynamic>{
  'items': instance.items,
  'unreadCount': instance.unreadCount,
};

_$ChatForwardBundleImpl _$$ChatForwardBundleImplFromJson(
  Map<String, dynamic> json,
) => _$ChatForwardBundleImpl(
  version: (json['version'] as num).toInt(),
  messages: (json['messages'] as List<dynamic>)
      .map((e) => ChatForwardEntry.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$$ChatForwardBundleImplToJson(
  _$ChatForwardBundleImpl instance,
) => <String, dynamic>{
  'version': instance.version,
  'messages': instance.messages,
};

_$ChatForwardEntryImpl _$$ChatForwardEntryImplFromJson(
  Map<String, dynamic> json,
) => _$ChatForwardEntryImpl(
  senderName: json['senderName'] as String,
  avatarUrl: json['avatarUrl'] as String? ?? '',
  content: json['content'] as String,
  createdAt: json['createdAt'] as String,
  msgType: (json['msgType'] as num).toInt(),
  forwarded: json['forwarded'] == null
      ? null
      : ChatForwardBundle.fromJson(json['forwarded'] as Map<String, dynamic>),
);

Map<String, dynamic> _$$ChatForwardEntryImplToJson(
  _$ChatForwardEntryImpl instance,
) => <String, dynamic>{
  'senderName': instance.senderName,
  'avatarUrl': instance.avatarUrl,
  'content': instance.content,
  'createdAt': instance.createdAt,
  'msgType': instance.msgType,
  'forwarded': instance.forwarded,
};

_$ChatForwardResultImpl _$$ChatForwardResultImplFromJson(
  Map<String, dynamic> json,
) => _$ChatForwardResultImpl(
  convId: (json['convId'] as num).toInt(),
  messageIds: (json['messageIds'] as List<dynamic>)
      .map((e) => (e as num).toInt())
      .toList(),
);

Map<String, dynamic> _$$ChatForwardResultImplToJson(
  _$ChatForwardResultImpl instance,
) => <String, dynamic>{
  'convId': instance.convId,
  'messageIds': instance.messageIds,
};
