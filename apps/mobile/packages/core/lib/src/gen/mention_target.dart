/// Public identity fields returned by the dedicated forum mention endpoint.
class MentionTarget {
  const MentionTarget({
    required this.userId,
    required this.username,
    required this.nickname,
    required this.avatarUrl,
    required this.actorType,
  });

  final int userId;
  final String username;
  final String nickname;
  final String avatarUrl;
  final String actorType;

  factory MentionTarget.fromJson(Map<String, dynamic> json) => MentionTarget(
    userId: (json['userId'] as num).toInt(),
    username: json['username'] as String,
    nickname: json['nickname'] as String? ?? '',
    avatarUrl: json['avatarUrl'] as String? ?? '',
    actorType: json['actorType'] as String,
  );
}
