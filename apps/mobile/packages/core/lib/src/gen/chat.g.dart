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
