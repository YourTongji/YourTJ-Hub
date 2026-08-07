import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../format.dart';
import 'status_views.dart';

/// 话题列表行(web TopicRow.vue 的移动端形态)。
/// 复用 ui_kit GfTopicRow,补充分类 chip 与导航。
class GfTopicListRow extends StatelessWidget {
  const GfTopicListRow({super.key, required this.topic});

  final dynamic topic;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final int replyCount = topic.replyCount is int
        ? topic.replyCount as int
        : int.tryParse('${topic.replyCount}') ?? 0;

    return GfCard(
      onTap: () => context.push('/p/${topic.id}'),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                if (topic.pinWeight > 0) ...[
                  Icon(Icons.push_pin, size: 14, color: colors.error),
                  const SizedBox(width: 4),
                ],
                Expanded(
                  child: Text(
                    topic.title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: colors.baseContent,
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                if (topic.unseen == true) ...[
                  const SizedBox(width: 6),
                  Container(
                    width: 8,
                    height: 8,
                    decoration: BoxDecoration(
                      color: colors.primary,
                      shape: BoxShape.circle,
                    ),
                  ),
                ],
              ],
            ),
            if (topic.description.isNotEmpty) ...[
              const SizedBox(height: 4),
              Text(
                topic.description,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(color: colors.iconMuted, fontSize: 13),
              ),
            ],
            const SizedBox(height: 8),
            Row(
              children: [
                for (final cat
                    in (topic.categories as List<dynamic>? ?? const []))
                  Padding(
                    padding: const EdgeInsets.only(right: 6),
                    child: GfChip(
                      label: cat.name,
                      color: colorFromHex(cat.color as String? ?? ''),
                    ),
                  ),
                const Spacer(),
                Text(
                  '${timeAgo(topic.activityText is String ? topic.activityText : topic.lastUpdateTime ?? '')} · ${l10n.topicReplies(replyCount)}',
                  style: TextStyle(color: colors.iconMuted, fontSize: 12),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// 话题列表(无限分页)。
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
        return GfTopicListRow(topic: topics[index]);
      },
    );
  }
}
