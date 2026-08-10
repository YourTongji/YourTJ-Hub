import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';

/// 通知页(web notifications.index 的移动端形态):
/// 通知列表 + 未读标记 + 全部已读 + all/unread 筛选 + 点击跳转。
class NotificationsPage extends ConsumerStatefulWidget {
  const NotificationsPage({super.key});

  @override
  ConsumerState<NotificationsPage> createState() => _NotificationsPageState();
}

class _NotificationsPageState extends ConsumerState<NotificationsPage> {
  AsyncValue<NotificationListResponse> _list = const AsyncValue.loading();
  String _filter = 'all';
  int _cursor = 0;
  final List<NotificationPayload> _items = [];
  bool _loadingMore = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({bool silent = false}) async {
    if (!silent) setState(() => _list = const AsyncValue.loading());
    try {
      final resp = await ref
          .read(notificationRepositoryProvider)
          .fetchNotifications(filter: _filter, cursor: 0);
      if (!mounted) return;
      setState(() {
        _list = AsyncValue.data(resp);
        _items.clear();
        _items.addAll(resp.items);
        _cursor = resp.nextCursor;
      });
    } catch (e, st) {
      if (mounted) setState(() => _list = AsyncValue.error(e, st));
    }
  }

  Future<void> _loadMore() async {
    final resp = _list.value;
    if (resp == null || !resp.hasNext || _loadingMore) return;
    setState(() => _loadingMore = true);
    try {
      final next = await ref
          .read(notificationRepositoryProvider)
          .fetchNotifications(filter: _filter, cursor: _cursor);
      if (!mounted) return;
      setState(() {
        _items.addAll(next.items);
        _cursor = next.nextCursor;
        _list = AsyncValue.data(next);
      });
    } catch (_) {
      // 静默。
    } finally {
      if (mounted) setState(() => _loadingMore = false);
    }
  }

  Future<void> _markAllRead() async {
    try {
      await ref.read(notificationRepositoryProvider).markAllNotificationsRead();
      await _load(silent: true);
    } catch (_) {
      // 静默。
    }
  }

  Future<void> _markRead(NotificationPayload n) async {
    if (n.isRead) return;
    try {
      await ref
          .read(notificationRepositoryProvider)
          .markNotificationRead(notificationId: n.id);
      if (!mounted) return;
      setState(() {
        for (int i = 0; i < _items.length; i++) {
          if (_items[i].id == n.id) {
            _items[i] = _items[i].copyWith(isRead: true);
          }
        }
      });
    } catch (_) {
      // 静默。
    }
  }

  /// 点击通知:标记已读 + 跳转(web targetURL/actorURL 语义)。
  /// - 话题通知 → /p/:topicId
  /// - 关注/徽章 → /u/:actorId
  void _openNotification(NotificationPayload n) {
    _markRead(n);
    final int? topicId = n.topic?.id ?? n.payload.topicId;
    if (topicId != null && topicId > 0) {
      context.push('/p/$topicId');
      return;
    }
    if (n.eventType == 'follow' ||
        n.eventType == 'badge' ||
        n.payload.metadata?.profileUrl != null) {
      final int actorId = n.actor.id;
      if (actorId > 0) {
        context.push('/u/$actorId');
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Scaffold(
      appBar: GfAppBar(
        title: Text(l10n.notificationsTitle),
        actions: <Widget>[
          GfIconButton(
            icon: Icons.done_all_rounded,
            size: 44,
            tooltip: l10n.notificationsMarkAllRead,
            onPressed: _markAllRead,
          ),
        ],
      ),
      body: Column(
        children: [
          // 筛选:全部 / 未读(web all/unread tabs)。
          Container(
            height: 44,
            alignment: Alignment.centerLeft,
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: GfTabBar(
              tabs: <GfTab>[
                GfTab(label: l10n.notificationsAll, value: 'all'),
                GfTab(label: l10n.notificationsUnread, value: 'unread'),
              ],
              selected: _filter,
              onSelected: (Object value) {
                if (value == _filter) return;
                setState(() => _filter = value as String);
                _load();
              },
            ),
          ),
          const GfDivider(),
          Expanded(
            child: _list.when(
              loading: () => const GfLoading(),
              error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
              data: (resp) => GfScrollToTop(
                semanticLabel: l10n.commonBackToTop,
                threshold: 360,
                builder: (BuildContext context, ScrollController controller) {
                  return RefreshIndicator(
                    onRefresh: () => _load(silent: true),
                    child: _items.isEmpty
                        ? CustomScrollView(
                            controller: controller,
                            physics: const AlwaysScrollableScrollPhysics(),
                            slivers: <Widget>[
                              SliverFillRemaining(
                                hasScrollBody: false,
                                child: GfEmpty(
                                  message: l10n.notificationsEmpty,
                                ),
                              ),
                            ],
                          )
                        : ListView.separated(
                            controller: controller,
                            physics: const AlwaysScrollableScrollPhysics(),
                            itemCount: _items.length + 1,
                            separatorBuilder: (_, _) => const GfDivider(),
                            itemBuilder: (context, i) {
                              if (i == _items.length) {
                                return GfListFooter(
                                  loading: _loadingMore,
                                  hasMore: resp.hasNext,
                                  onLoadMore: _loadMore,
                                );
                              }
                              final NotificationPayload n = _items[i];
                              final (
                                IconData icon,
                                GfNotificationTone tone,
                              ) = switch (n.eventType) {
                                'follow' => (
                                  Icons.person_add,
                                  GfNotificationTone.success,
                                ),
                                'badge' => (
                                  Icons.workspace_premium,
                                  GfNotificationTone.warning,
                                ),
                                'system' => (
                                  Icons.info_outline,
                                  GfNotificationTone.info,
                                ),
                                _ => (
                                  Icons.message,
                                  GfNotificationTone.primary,
                                ),
                              };
                              return GfNotificationRow(
                                icon: icon,
                                tone: tone,
                                title: n.title,
                                subtitle: n.content,
                                time: timeAgo(n.createdAt, l10n: l10n),
                                unread: !n.isRead,
                                onTap: () => _openNotification(n),
                              );
                            },
                          ),
                  );
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}
