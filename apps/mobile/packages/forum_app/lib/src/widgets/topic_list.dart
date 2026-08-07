import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../format.dart';
import 'status_views.dart';

/// 话题列表(无限分页),行复用 ui_kit [GfTopicRow]。
class GfTopicList extends StatelessWidget {
  const GfTopicList({
    super.key,
    required this.loading,
    required this.topics,
    required this.hasMore,
    required this.onLoadMore,
  });

  final bool loading;
  final List<dynamic> topics;
  final bool hasMore;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    if (loading && topics.isEmpty) return const GfLoading();
    if (topics.isEmpty) {
      return GfEmpty(message: AppLocalizations.of(context).topicEmpty);
    }
    return ListView.separated(
      itemCount: topics.length + 1,
      separatorBuilder: (_, _) => const SizedBox(height: 1),
      itemBuilder: (context, index) {
        if (index == topics.length) {
          return GfListFooter(
            loading: loading,
            hasMore: hasMore,
            onLoadMore: onLoadMore,
          );
        }
        return _topicRow(context, topics[index]);
      },
    );
  }
}

/// 把后端话题 payload 映射为 [GfTopicRow](对齐 web TopicRow.vue 语义)。
Widget _topicRow(BuildContext context, dynamic topic) {
  final int replyCount = topic.replyCount is int
      ? topic.replyCount as int
      : int.tryParse('${topic.replyCount}') ?? 0;
  final int viewCount = topic.viewCount is int
      ? topic.viewCount as int
      : int.tryParse('${topic.viewCount}') ?? 0;

  final List<GfTopicCategory> categories = <GfTopicCategory>[
    for (final cat in (topic.categories as List<dynamic>? ?? const []))
      GfTopicCategory(
        name: cat.name as String? ?? '',
        color: colorFromHex(cat.color as String? ?? ''),
      ),
  ];

  final List<Widget> participantAvatars = <Widget>[
    for (final p in (topic.participants as List<dynamic>? ?? const []))
      _avatar(p.avatarUrl as String?),
  ];

  return GfTopicRow(
    title: topic.title as String? ?? '',
    description: topic.description as String? ?? '',
    categories: categories,
    participantAvatars: participantAvatars,
    activityText: timeAgo(
      topic.activityText is String
          ? topic.activityText
          : topic.lastUpdateTime ?? '',
    ),
    replyCount: replyCount,
    viewCount: viewCount,
    hot: viewCount > 500,
    pinned: (topic.pinWeight ?? 0) > 0,
    unseen: topic.unseen == true,
    onTap: () => context.push('/p/${topic.id}'),
  );
}

/// 参与者小头像(web AvatarStack sm 语义)。
Widget _avatar(String? url) {
  if (url == null || url.isEmpty) {
    return CircleAvatar(radius: 10, backgroundColor: Colors.transparent);
  }
  return CircleAvatar(
    radius: 10,
    backgroundImage: NetworkImage(url),
    backgroundColor: Colors.transparent,
  );
}
