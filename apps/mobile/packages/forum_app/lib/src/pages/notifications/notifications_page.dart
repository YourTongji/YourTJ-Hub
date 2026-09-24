import 'package:dio/dio.dart';
import '../../private_notes.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import 'notification_text.dart';
import 'notification_target.dart';
import '../../widgets/app_refresh_indicator.dart';
import '../../server_messages.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../navigation/tab_scroll_registry.dart';
import '../../widgets/root_surface.dart';
import '../../widgets/status_views.dart';
import '../../realtime/realtime_updates.dart';

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
  bool _refreshing = false;
  bool _markingAll = false;
  int _generation = 0;
  CancelToken? _loadCancel;
  CancelToken? _loadMoreCancel;
  String? _loadMoreError;
  final Set<int> _reading = {};
  final Set<int> _acknowledged = {};
  final Map<int, String> _readErrors = {};
  final _scroll = GfScrollToTopController();
  late final GfTabScrollRegistry _registry;
  int _seenRealtimeRevision = 0;
  bool _realtimeDirty = false;

  @override
  void dispose() {
    _generation++;
    _loadCancel?.cancel('notifications disposed');
    _loadMoreCancel?.cancel('notifications disposed');
    _registry.unregister(GfShellDestination.notifications, _scroll);
    super.dispose();
  }

  @override
  void initState() {
    super.initState();
    _registry = ref.read(tabScrollRegistryProvider)
      ..register(GfShellDestination.notifications, _scroll);
    _seenRealtimeRevision = ref
        .read(realtimeInvalidationsProvider)
        .notificationsRevision;
    _load();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final revision = ref
        .read(realtimeInvalidationsProvider)
        .notificationsRevision;
    if (TickerMode.valuesOf(context).enabled &&
        revision != _seenRealtimeRevision) {
      _seenRealtimeRevision = revision;
      _load(silent: true);
    }
  }

  Future<void> _load({bool silent = false}) async {
    if (silent && _refreshing) {
      _realtimeDirty = true;
      return;
    }
    _loadCancel?.cancel('notifications request superseded');
    _loadMoreCancel?.cancel('notifications refresh superseded');
    final cancel = _loadCancel = CancelToken();
    final generation = ++_generation;
    final epoch = ref.read(offlineCacheEpochProvider);
    final filter = _filter;
    _refreshing = true;
    _loadingMore = false;
    if (!silent) {
      setState(() {
        _loadMoreError = null;
        _list = const AsyncValue.loading();
      });
    }
    try {
      final resp = await ref
          .read(notificationRepositoryProvider)
          .fetchNotifications(filter: filter, cursor: 0, cancelToken: cancel);
      if (!_current(generation, epoch)) return;
      final confirmedRead = resp.items
          .where((item) => item.isRead)
          .map((item) => item.id)
          .toSet();
      setState(() {
        _readErrors.removeWhere((id, _) => confirmedRead.contains(id));
        _loadMoreError = null;
        _list = AsyncValue.data(resp);
        _items.clear();
        _items.addAll(_mergeRead(resp.items));
        _cursor = resp.nextCursor;
      });
    } catch (e, st) {
      if (!_current(generation, epoch) || cancel.isCancelled) return;
      if (silent && _list.hasValue) {
        _showError(e);
      } else {
        setState(() => _list = AsyncValue.error(e, st));
      }
    } finally {
      if (identical(_loadCancel, cancel)) _loadCancel = null;
      if (_current(generation, epoch)) {
        setState(() => _refreshing = false);
        if (_realtimeDirty) {
          _realtimeDirty = false;
          _load(silent: true);
        }
      }
    }
  }

  bool _current(int generation, int epoch) =>
      mounted &&
      generation == _generation &&
      epoch == ref.read(offlineCacheEpochProvider);

  Iterable<NotificationPayload> _mergeRead(List<NotificationPayload> items) =>
      items
          .map(
            (item) => _acknowledged.contains(item.id)
                ? item.copyWith(isRead: true)
                : item,
          )
          .where((item) => _filter != 'unread' || !item.isRead);

  void _showError(Object error) => showGfToast(
    context,
    resolveErrorMessage(AppLocalizations.of(context), error),
    error: true,
  );

  Future<void> _loadMore() async {
    final l10n = AppLocalizations.of(context);
    final resp = _list.valueOrNull;
    if (resp == null || !resp.hasNext || _loadingMore || _refreshing) return;
    final generation = _generation;
    final epoch = ref.read(offlineCacheEpochProvider);
    final cancel = _loadMoreCancel = CancelToken();
    final cursor = _cursor;
    final filter = _filter;
    setState(() {
      _loadingMore = true;
      _loadMoreError = null;
    });
    try {
      final next = await ref
          .read(notificationRepositoryProvider)
          .fetchNotifications(
            filter: filter,
            cursor: cursor,
            cancelToken: cancel,
          );
      if (!_current(generation, epoch)) return;
      final seen = _items.map((item) => item.id).toSet();
      final additions = _mergeRead(
        next.items,
      ).where((item) => seen.add(item.id)).toList();
      if (next.hasNext && next.nextCursor == cursor && additions.isEmpty) {
        setState(() => _loadMoreError = l10n.commonLoadFailed);
        return;
      }
      setState(() {
        _items.addAll(additions);
        _cursor = next.nextCursor;
        _list = AsyncValue.data(next);
      });
    } catch (error) {
      if (!_current(generation, epoch) || cancel.isCancelled) return;
      setState(
        () => _loadMoreError = resolveErrorMessage(
          AppLocalizations.of(context),
          error,
        ),
      );
    } finally {
      if (identical(_loadMoreCancel, cancel)) _loadMoreCancel = null;
      if (_current(generation, epoch)) setState(() => _loadingMore = false);
    }
  }

  Future<void> _markAllRead() async {
    if (_markingAll || _reading.isNotEmpty) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    final ids = _items.map((item) => item.id).toSet();
    setState(() => _markingAll = true);
    try {
      final ok = await ref
          .read(notificationRepositoryProvider)
          .markAllNotificationsRead();
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      if (!ok) {
        throw const ApiException(
          fallbackMessage: 'Notification read was not acknowledged',
        );
      }
      setState(() {
        _acknowledged.addAll(ids);
        _readErrors.clear();
        final items = _mergeRead(_items).toList();
        _items
          ..clear()
          ..addAll(items);
        // Supersede a read started before this acknowledged mutation.
        _generation++;
        _refreshing = false;
      });
      await _load(silent: true);
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        _showError(error);
      }
    } finally {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _markingAll = false);
      }
    }
  }

  Future<void> _markRead(NotificationPayload n) async {
    if (n.isRead ||
        _acknowledged.contains(n.id) ||
        _reading.contains(n.id) ||
        _markingAll) {
      return;
    }
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() {
      _reading.add(n.id);
      _readErrors.remove(n.id);
    });
    try {
      final ok = await ref
          .read(notificationRepositoryProvider)
          .markNotificationRead(notificationId: n.id);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      if (!ok) {
        throw const ApiException(
          fallbackMessage: 'Notification read was not acknowledged',
        );
      }
      setState(() {
        _acknowledged.add(n.id);
        for (int i = 0; i < _items.length; i++) {
          if (_items[i].id == n.id) {
            _items[i] = _items[i].copyWith(isRead: true);
          }
        }
        if (_filter == 'unread') _items.removeWhere((item) => item.id == n.id);
      });
    } catch (error) {
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(
        () => _readErrors[n.id] = resolveErrorMessage(
          AppLocalizations.of(context),
          error,
        ),
      );
      _showError(error);
    } finally {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _reading.remove(n.id));
      }
    }
  }

  /// 点击通知:标记已读 + 跳转(web targetURL/actorURL 语义)。
  /// - 话题通知 → /p/:topicId
  /// - 关注/徽章 → /u/:actorId
  void _openNotification(NotificationPayload n) {
    _markRead(n);
    final target = notificationTarget(n);
    if (target != null) context.push(target);
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(realtimeInvalidationsProvider, (previous, next) {
      if (previous?.notificationsRevision == next.notificationsRevision ||
          !TickerMode.valuesOf(context).enabled) {
        return;
      }
      _seenRealtimeRevision = next.notificationsRevision;
      _load(silent: true);
    });
    final AppLocalizations l10n = AppLocalizations.of(context);
    ref.listen<int>(offlineCacheEpochProvider, (_, _) {
      _generation++;
      _loadCancel?.cancel('notifications session changed');
      _loadMoreCancel?.cancel('notifications session changed');
      _loadingMore = false;
      _items.clear();
      _acknowledged.clear();
      _reading.clear();
      _readErrors.clear();
      _markingAll = false;
      _refreshing = false;
      _realtimeDirty = false;
      _cursor = 0;
      _load();
    });

    return RootSurface(
      title: l10n.notificationsTitle,
      actions: <Widget>[
        if (_markingAll)
          SizedBox(
            width: 44,
            height: 44,
            child: Center(
              child: Semantics(
                label: l10n.commonLoading,
                liveRegion: true,
                child: const GfLoadingIndicator(small: true),
              ),
            ),
          )
        else
          GfIconButton(
            icon: Icons.done_all_rounded,
            size: 44,
            tooltip: l10n.notificationsMarkAllRead,
            onPressed: _reading.isNotEmpty ? null : _markAllRead,
          ),
      ],
      toolbarHeight: GfTabBar.heightFor(context),
      toolbar: Container(
        height: GfTabBar.heightFor(context),
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
      body: (top, bottom) => Column(
        children: [
          Expanded(
            child: _list.when(
              loading: () => const GfLoading(),
              error: (e, _) => GfErrorRetry(
                message: resolveErrorMessage(l10n, e),
                onRetry: _load,
              ),
              data: (resp) => GfScrollToTop(
                controller: _scroll,
                semanticLabel: l10n.commonBackToTop,
                threshold: 360,
                showButton: false,
                builder: (BuildContext context, ScrollController controller) {
                  return AppRefreshIndicator(
                    edgeOffset: top,
                    onRefresh: () => _load(silent: true),
                    child: _items.isEmpty && !resp.hasNext
                        ? CustomScrollView(
                            controller: controller,
                            physics: const AlwaysScrollableScrollPhysics(),
                            slivers: <Widget>[
                              SliverFillRemaining(
                                hasScrollBody: false,
                                child: Padding(
                                  padding: EdgeInsets.only(
                                    top: top,
                                    bottom: bottom,
                                  ),
                                  child: GfEmpty(
                                    message: l10n.notificationsEmpty,
                                    description:
                                        l10n.notificationsEmptyDescription,
                                    icon: Icons.notifications_none_rounded,
                                    action: GfButton(
                                      label: l10n.navHome,
                                      onPressed: () => context.go('/'),
                                    ),
                                  ),
                                ),
                              ),
                            ],
                          )
                        : ListView.separated(
                            controller: controller,
                            physics: const AlwaysScrollableScrollPhysics(),
                            padding: EdgeInsets.only(top: top, bottom: bottom),
                            itemCount: _items.length + 1,
                            separatorBuilder: (_, _) => const GfDivider(),
                            itemBuilder: (context, i) {
                              if (i == _items.length) {
                                return GfListFooter(
                                  key: ValueKey((_filter, _generation)),
                                  progressKey: (_cursor, _items.length),
                                  error: _loadMoreError,
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
                              final (title, subtitle) = notificationText(
                                n,
                                l10n,
                                displayName: (id, name) =>
                                    privateDisplayName(context, id, name),
                              );
                              return Column(
                                key: ValueKey(n.id),
                                children: [
                                  GfNotificationRow(
                                    icon: icon,
                                    tone: tone,
                                    title: title,
                                    subtitle: subtitle,
                                    time: timeAgo(n.createdAt, l10n: l10n),
                                    unread: !n.isRead,
                                    onMarkRead:
                                        n.isRead ||
                                            _reading.contains(n.id) ||
                                            _markingAll
                                        ? null
                                        : () => _markRead(n),
                                    markReadLabel: l10n.notificationsMarkRead,
                                    onTap: () => _openNotification(n),
                                  ),
                                  if (!n.isRead &&
                                      (_reading.contains(n.id) ||
                                          _readErrors.containsKey(n.id)))
                                    Padding(
                                      padding: const EdgeInsets.fromLTRB(
                                        16,
                                        0,
                                        16,
                                        12,
                                      ),
                                      child: Row(
                                        children: [
                                          if (_reading.contains(n.id)) ...[
                                            const GfLoadingIndicator(
                                              small: true,
                                            ),
                                            const SizedBox(width: 8),
                                            Expanded(
                                              child: Text(l10n.commonLoading),
                                            ),
                                          ] else ...[
                                            Expanded(
                                              child: Text(
                                                _readErrors[n.id]!,
                                                style: TextStyle(
                                                  color: GfTheme.colorsOf(
                                                    context,
                                                  ).error,
                                                ),
                                              ),
                                            ),
                                            TextButton(
                                              onPressed: () => _markRead(n),
                                              child: Text(l10n.commonRetry),
                                            ),
                                          ],
                                        ],
                                      ),
                                    ),
                                ],
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
