import '../../private_notes.dart';
import '../../navigation/auth_navigation.dart';
import 'dart:math' as math;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../current_user.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../profile_links.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';
import '../../widgets/skeletons.dart';
import '../../widgets/topic_list.dart';
import '../../widgets/user_badge.dart';

typedef _ContentKey = (bool, int); // isReply, content ID
_ContentKey? _activityKey(UserActivityPayload activity) {
  final type = activity.subjectType.toLowerCase();
  if (activity.action == 5 && type == 'post') return (true, activity.subjectId);
  if ((activity.action == 2 || activity.action == 3) &&
      (type == 'topic' || type == 'post')) {
    return (false, activity.subjectId);
  }
  return null;
}

/// User profile aligned with the web identity card while keeping mobile
/// navigation, actions and content streams clear and thumb-friendly.
class ProfilePage extends ConsumerStatefulWidget {
  const ProfilePage({super.key, this.userId, this.initialStream = 'timeline'})
    : connectionsOnly = false;

  const ProfilePage.connections({
    super.key,
    this.userId,
    this.initialStream = 'following',
  }) : connectionsOnly = true;

  final bool connectionsOnly;

  final int? userId;
  final String initialStream;

  @override
  ConsumerState<ProfilePage> createState() => _ProfilePageState();
}

class _ProfileStreamState {
  UserProfileProps? props;
  bool loading = false;
  bool loadingMore = false;
  Object? error;
  Object? paginationError;
  int request = 0;
  double? offset;
  CancelToken? cancelToken;
}

class _ProfilePageState extends ConsumerState<ProfilePage> {
  final _streams = <String, _ProfileStreamState>{};
  _ProfileStreamState get _active =>
      _streams.putIfAbsent(_stream, _ProfileStreamState.new);
  AsyncValue<UserProfileProps> get _page {
    final props = _active.props ?? _headerProps;
    if (props != null) return AsyncValue.data(props);
    if (_active.error != null) {
      return AsyncValue.error(_active.error!, StackTrace.current);
    }
    return const AsyncValue.loading();
  }

  bool get _loadingMore => _active.loadingMore;
  bool get _streamLoading => _active.loading && _active.props == null;
  Object? get _streamError => _active.error;
  int _followRevision = 0;
  UserProfileProps? _headerProps;
  double _minimumScrollOffset = 0;
  String _stream = 'timeline';
  bool _following = false;
  bool _followBusy = false;
  final _connectionFollowing = <int, bool>{};
  final _connectionBusy = <int>{};
  int _connectionEpoch = 0;
  int _connectionRead = 0;
  final _connectionRevisions = <int, int>{};
  final _connectionAcceptedReads = <int, int>{};
  final _seenTopicReturns = <int, TopicReturnState>{};
  final _seenPostReturns = <int, PostReturnState>{};
  int _interactionRevision = 0;
  final _interactionBusy = <_ContentKey>{};
  final _interactions =
      <_ContentKey, ({int revision, PostReturnState state})>{};
  bool _loginRequired = false;
  bool _canAccessAdmin = false;
  bool _canModerate = false;
  bool _canManageCourses = false;

  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();

  bool get _isShellProfile => widget.userId == null;

  @override
  void initState() {
    super.initState();
    _stream = widget.initialStream;
    _seenTopicReturns.addAll(ref.read(topicReturnStatesProvider));
    _seenPostReturns.addAll(ref.read(postReturnStatesProvider));

    _load();
  }

  @override
  void didUpdateWidget(covariant ProfilePage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.userId != widget.userId ||
        oldWidget.initialStream != widget.initialStream ||
        oldWidget.connectionsOnly != widget.connectionsOnly) {
      _stream = widget.initialStream;
      _resetStreams();
      _load();
    }
  }

  void _resetStreams() {
    for (final state in _streams.values) {
      _cancelStreamRead(state);
    }
    _streams.clear();
    _interactionBusy.clear();
    _interactions.clear();
    _seenTopicReturns.clear();
    _seenPostReturns.clear();
    _interactionRevision++;
    _connectionFollowing.clear();
    _connectionBusy.clear();
    _connectionRevisions.clear();
    _connectionAcceptedReads.clear();
    _connectionEpoch++;
    _headerProps = null;
    _minimumScrollOffset = 0;
    _followBusy = false;
    _following = false;
    _canAccessAdmin = false;
    _canModerate = false;
    _canManageCourses = false;
    _followRevision++;
  }

  void _cancelStreamRead(_ProfileStreamState state) {
    final token = state.cancelToken;
    if (token == null) return;
    state.cancelToken = null;
    state.request++;
    state.loading = false;
    state.loadingMore = false;
    token.cancel();
  }

  @override
  void dispose() {
    for (final state in _streams.values) {
      _cancelStreamRead(state);
    }
    super.dispose();
  }

  Future<void> _load({String? nextUrl, bool streamChange = false}) async {
    final key = _stream;
    final state = _active;
    _cancelStreamRead(state);
    final cancelToken = CancelToken();
    state.cancelToken = cancelToken;
    final request = ++state.request;
    final epoch = ref.read(offlineCacheEpochProvider);
    final followRevision = _followRevision;
    final interactionRevision = _interactionRevision;
    final connectionRevisions = Map<int, int>.of(_connectionRevisions);
    final connectionRead = ++_connectionRead;
    final previous = state.props;
    bool current() =>
        mounted &&
        !cancelToken.isCancelled &&
        request == state.request &&
        identical(_streams[key], state) &&
        ref.read(offlineCacheEpochProvider) == epoch;
    setState(() {
      state.loading = nextUrl == null;
      state.loadingMore = nextUrl != null;
      state.error = null;
      state.paginationError = null;
      _loginRequired = false;
    });
    try {
      final int? currentId = (await ref.read(currentUserProvider.future))?.id;
      if (!mounted || !current()) return;
      final int uid = widget.userId ?? currentId ?? 0;
      if (uid == 0) {
        setState(() {
          _loginRequired = true;
          state.error = AppLocalizations.of(context).profileNotLoggedIn;
        });
        return;
      }
      final path = nextUrl ?? _streamPath(uid, key);
      final payload = await ref
          .read(pageRepositoryProvider)
          .fetch(path, cancelToken: cancelToken);
      if (!mounted || !current()) return;
      var props = parsePageProps<UserProfileProps>(payload);
      if (props == null) {
        throw FormatException(AppLocalizations.of(context).commonParseFailed);
      }
      void accept(_ContentKey? key, bool? liked, bool? bookmarked, int count) {
        if (key == null ||
            liked == null ||
            bookmarked == null ||
            _interactionBusy.contains(key)) {
          return;
        }
        final update = _interactions[key];
        if (update == null || update.revision <= interactionRevision) {
          _setInteraction(key, (
            liked: liked,
            bookmarked: bookmarked,
            likeCount: count,
          ));
        }
      }

      for (final topic in props.topics) {
        accept(
          (false, topic.id),
          topic.liked,
          topic.bookmarked,
          topic.likeCount,
        );
      }
      for (final activity in props.activities) {
        accept(
          _activityKey(activity),
          activity.liked,
          activity.bookmarked,
          activity.likeCount ?? 0,
        );
      }
      // Accept only rows actually returned by this read, not retained pages.
      // Reads started before/during a mutation cannot undo it; a later refresh
      // can reconcile changes made elsewhere, shared across both retained tabs.
      final connections = switch (key) {
        'following' => props.following,
        'followers' => props.followers,
        _ => const <UserConnectionPayload>[],
      };
      for (final user in connections) {
        if (user.isFollowing != null &&
            !_connectionBusy.contains(user.id) &&
            (connectionRevisions[user.id] ?? 0) ==
                (_connectionRevisions[user.id] ?? 0) &&
            connectionRead > (_connectionAcceptedReads[user.id] ?? 0)) {
          _connectionFollowing[user.id] = user.isFollowing!;
          _connectionAcceptedReads[user.id] = connectionRead;
        }
      }
      if (nextUrl != null && previous != null) {
        props = props.copyWith(
          topics: _merge(previous.topics, props.topics, (item) => item.id),
          activities: _merge(
            previous.activities,
            props.activities,
            (item) => item.id,
          ),
          likes: _merge(previous.likes, props.likes, (item) => item.id),
          bookmarks: _merge(
            previous.bookmarks,
            props.bookmarks,
            (item) => item.id,
          ),
          following: _merge(
            previous.following,
            props.following,
            (item) => item.id,
          ),
          followers: _merge(
            previous.followers,
            props.followers,
            (item) => item.id,
          ),
        );
      }
      final loaded = _mergeInteractionState(props, interactionRevision);
      setState(() {
        state.props = loaded;
        // Keep the visible identity stable while tabs load, and never undo a
        // follow action started after this read.
        if (key == _stream && nextUrl == null && !streamChange) {
          _headerProps = loaded;
          if (!_followBusy && followRevision == _followRevision) {
            _following = loaded.user.isFollowing;
          }
          _canAccessAdmin = payload.layout.viewer.canAccessAdmin;
          _canModerate = payload.layout.viewer.isModerator;
          _canManageCourses = payload.layout.viewer.canManageCourses;
        }
      });
    } catch (error) {
      if (mounted && current()) {
        setState(() {
          if (nextUrl != null) {
            state.paginationError = error;
          } else {
            state.error = error;
          }
        });
      }
    } finally {
      if (mounted && current()) {
        setState(() {
          state.cancelToken = null;
          state.loading = false;
          state.loadingMore = false;
        });
      }
    }
  }

  List<T> _merge<T>(List<T> previous, List<T> next, int Function(T) id) => {
    for (final item in previous) id(item): item,
    for (final item in next) id(item): item,
  }.values.toList();

  void _selectStream(
    String key,
    ScrollController controller,
    double headerExtent,
  ) {
    if (key == _stream) return;
    _active.offset = controller.offset;
    _cancelStreamRead(_active);
    final state = _streams.putIfAbsent(key, _ProfileStreamState.new);
    final offset =
        state.offset ??
        math.min(math.max(0.0, controller.offset), headerExtent);
    setState(() {
      _stream = key;
      _minimumScrollOffset = offset;
    });
    // Restore after the new slivers have laid out, preserving deep offsets for
    // visited streams and only the collapsed header for a first visit.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _stream == key && controller.hasClients) {
        controller.jumpTo(offset);
      }
    });
    if (state.props == null && !state.loading && state.error == null) {
      _load(streamChange: true);
    }
  }

  String _streamPath(int uid, String key) => switch (key) {
    'bookmarks' || 'badges' || 'following' || 'followers' => '/u/$uid/$key',
    'timeline' => '/u/$uid/activity',
    _ => '/u/$uid/activity/$key',
  };

  List<TabItemPayload> _tabs(UserProfileProps props, AppLocalizations l10n) => [
    for (final (key, label)
        in widget.connectionsOnly
            ? [
                ('following', l10n.profileFollowingCount),
                ('followers', l10n.profileFollowers),
              ]
            : [
                ('timeline', l10n.profileActivity),
                ('topics', l10n.profilePosts),
                ('likes', l10n.profileLikedPosts),
                if (props.isOwnProfile) ('bookmarks', l10n.profileBookmarks),
                ('badges', l10n.profileBadges),
              ])
      TabItemPayload(key: key, label: label, url: '', active: key == _stream),
  ];

  Future<void> _loadMore(UserProfileProps props) async {
    if (_active.loading || _loadingMore || !props.pagination.hasNext) return;
    final uri = Uri.tryParse(props.pagination.nextUrl);
    // Follow only relative pagination URLs for this exact user and stream.
    if (uri == null ||
        uri.hasScheme ||
        uri.hasAuthority ||
        uri.path != _streamPath(props.user.userId, _stream)) {
      return;
    }
    await _load(nextUrl: uri.toString());
  }

  Future<void> _toggleFollow(UserCardPayload user) async {
    if (user.isSelf || _followBusy) return;
    final revision = ++_followRevision;
    final epoch = ref.read(offlineCacheEpochProvider);
    bool current() =>
        mounted &&
        revision == _followRevision &&
        ref.read(offlineCacheEpochProvider) == epoch;
    final bool wasFollowing = _following;
    final bool target = !wasFollowing;
    setState(() {
      _following = target;
      _followBusy = true;
    });
    try {
      await ref
          .read(topicRepositoryProvider)
          .followUser(userId: user.userId, isFollowing: wasFollowing);
    } catch (error) {
      if (mounted && current()) {
        setState(() => _following = wasFollowing);
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (current()) {
        setState(() {
          // Reads begun during the mutation also precede its settled truth.
          _followRevision++;
          _followBusy = false;
        });
      }
    }
  }

  Future<void> _toggleConnection(UserConnectionPayload user) async {
    if (user.isSelf || _connectionBusy.contains(user.id)) return;
    if (ref.read(currentUserProvider).valueOrNull == null) {
      await context.push(
        authLoginLocation(returnTo: GoRouterState.of(context).uri.toString()),
      );
      return;
    }
    final previous = _connectionFollowing[user.id] ?? user.isFollowing;
    if (previous == null) return;
    final epoch = _connectionEpoch;
    final session = ref.read(offlineCacheEpochProvider);
    bool current() =>
        mounted &&
        epoch == _connectionEpoch &&
        session == ref.read(offlineCacheEpochProvider);
    setState(() {
      _connectionFollowing[user.id] = !previous;
      _connectionBusy.add(user.id);
      _connectionRevisions[user.id] = (_connectionRevisions[user.id] ?? 0) + 1;
    });
    try {
      await ref
          .read(topicRepositoryProvider)
          .followUser(userId: user.id, isFollowing: previous);
    } catch (error) {
      if (mounted && current()) {
        setState(() => _connectionFollowing[user.id] = previous);
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (current()) {
        setState(() {
          _connectionBusy.remove(user.id);
          _connectionRevisions[user.id] =
              (_connectionRevisions[user.id] ?? 0) + 1;
        });
      }
    }
  }

  UserProfileProps _mergeInteractionState(
    UserProfileProps props,
    int readRevision, {
    _ContentKey? only,
  }) {
    PostReturnState? state(_ContentKey? key) {
      if (key == null) return null;
      if (only != null && only != key) return null;
      final value = _interactions[key];
      return value != null &&
              (_interactionBusy.contains(key) || value.revision > readRevision)
          ? value.state
          : null;
    }

    return props.copyWith(
      topics: [
        for (final topic in props.topics)
          if (state((false, topic.id)) case final update?)
            topic.copyWith(
              liked: update.liked,
              bookmarked: update.bookmarked,
              likeCount: update.likeCount,
            )
          else
            topic,
      ],
      activities: [
        for (final activity in props.activities)
          if (state(_activityKey(activity)) case final update?)
            activity.copyWith(
              liked: update.liked,
              bookmarked: update.bookmarked,
              likeCount: update.likeCount,
            )
          else
            activity,
      ],
    );
  }

  void _setInteraction(_ContentKey key, PostReturnState value) {
    _interactions[key] = (revision: ++_interactionRevision, state: value);
    for (final stream in _streams.values) {
      if (stream.props != null) {
        stream.props = _mergeInteractionState(stream.props!, -1, only: key);
      }
    }
  }

  void _syncReturnedInteractions() {
    if (!mounted || ref.read(currentUserProvider).valueOrNull == null) return;
    setState(() {
      for (final entry in ref.read(topicReturnStatesProvider).entries) {
        final state = entry.value;
        if (_seenTopicReturns[entry.key] == state) continue;
        _seenTopicReturns[entry.key] = state;
        if (state.liked != null && state.bookmarked != null) {
          _setInteraction(
            (false, entry.key),
            (
              liked: state.liked!,
              bookmarked: state.bookmarked!,
              likeCount: state.likeCount,
            ),
          );
        }
      }
      for (final entry in ref.read(postReturnStatesProvider).entries) {
        if (_seenPostReturns[entry.key] == entry.value) continue;
        _seenPostReturns[entry.key] = entry.value;
        _setInteraction((true, entry.key), entry.value);
      }
    });
  }

  Future<bool> _toggleInteraction(
    _ContentKey key,
    PostReturnState previous,
    bool bookmark,
    bool target,
  ) async {
    if (ref.read(currentUserProvider).valueOrNull == null) {
      await context.push(
        authLoginLocation(returnTo: GoRouterState.of(context).uri.toString()),
      );
      return false;
    }
    if (!_interactionBusy.add(key)) return false;
    final failedMessage = AppLocalizations.of(context).commonLoadFailed;
    final epoch = ref.read(offlineCacheEpochProvider);
    final generation = _connectionEpoch;
    bool current() =>
        mounted &&
        generation == _connectionEpoch &&
        epoch == ref.read(offlineCacheEpochProvider);
    final next = (
      liked: bookmark ? previous.liked : target,
      bookmarked: bookmark ? target : previous.bookmarked,
      likeCount:
          previous.likeCount +
          (bookmark || previous.liked == target
              ? 0
              : target
              ? 1
              : -1),
    );
    setState(() => _setInteraction(key, next));
    try {
      bool success;
      if (key.$1) {
        final repo = ref.read(postRepositoryProvider);
        if (bookmark) {
          success = await repo.bookmarkPost(
            postId: key.$2,
            action: target ? 1 : 2,
          );
        } else {
          success = await repo.likePost(postId: key.$2, action: target ? 1 : 2);
        }
      } else {
        final repo = ref.read(topicRepositoryProvider);
        if (bookmark) {
          success = await repo.bookmarkTopic(
            topicId: key.$2,
            action: target ? 1 : 2,
          );
        } else {
          success = await repo.likeTopic(
            topicId: key.$2,
            action: target ? 1 : 2,
          );
        }
      }
      if (!current()) return false;
      if (!success) throw StateError(failedMessage);
      // Publish successful state for the home/detail return handoff as well.
      if (key.$1) {
        ref.read(postReturnStatesProvider)[key.$2] = next;
      } else {
        final old = ref.read(topicReturnStatesProvider)[key.$2];
        if (old != null) {
          ref.read(topicReturnStatesProvider)[key.$2] = (
            unseen: old.unseen,
            liked: next.liked,
            bookmarked: next.bookmarked,
            likeCount: next.likeCount,
            replyCount: old.replyCount,
            viewCount: old.viewCount,
          );
        }
      }
      return true;
    } catch (error) {
      if (mounted && current()) {
        setState(() => _setInteraction(key, previous));
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
      return false;
    } finally {
      if (current()) {
        setState(() {
          _interactionBusy.remove(key);
          // Fence reads started while the write was in flight, including rollback.
          final value = _interactions[key]!;
          _interactions[key] = (
            revision: ++_interactionRevision,
            state: value.state,
          );
        });
      }
    }
  }

  Future<void> _openProfileTool(String route) async {
    final saved = await context.push<Object?>(route);
    if (!mounted) return;
    if (route == '/settings/profile?edit=1' && saved == true) {
      showGfToast(context, AppLocalizations.of(context).settingsInfoSaved);
    }
    await _load();
  }

  Widget _profileTitle(BuildContext context, AppLocalizations l10n) =>
      _page.valueOrNull == null
      ? Text(l10n.profileTitle)
      : Builder(
          builder: (context) {
            final user = (_headerProps ?? _page.valueOrNull!).user;
            return Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  privateDisplayName(
                    context,
                    user.userId,
                    user.username,
                    user.nickname,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                if (MediaQuery.textScalerOf(context).scale(13) <= 18)
                  Text(
                    widget.connectionsOnly
                        ? '@${user.username}'
                        : '${formatNumber(user.topicCount)} ${l10n.profileTopics}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 13,
                      height: 1.2,
                      fontWeight: FontWeight.w400,
                      color: GfTheme.colorsOf(context).iconMuted,
                    ),
                  ),
              ],
            );
          },
        );

  List<Widget> _profileActions(
    BuildContext context,
    AppLocalizations l10n, {
    bool glass = false,
  }) =>
      !widget.connectionsOnly &&
          (_isShellProfile || _page.valueOrNull?.isOwnProfile == true)
      ? <Widget>[
          _profileMenu(context, l10n, glass: glass),
          if (glass)
            GfGlassIconButton(
              symbol: 'bell',
              tooltip: l10n.notificationsTitle,
              onPressed: () => context.go('/notifications'),
            )
          else
            GfIconButton(
              symbol: 'bell',
              tooltip: l10n.notificationsTitle,
              onPressed: () => context.go('/notifications'),
            ),
        ]
      : const <Widget>[];

  Widget _profileMenu(
    BuildContext context,
    AppLocalizations l10n, {
    required bool glass,
  }) {
    final menu = PopupMenuButton<String>(
      tooltip: l10n.profileMore,
      icon: GfSymbol(
        'ellipsis',
        color: glass ? Colors.white : GfTheme.colorsOf(context).baseContent,
      ),
      useRootNavigator: true,
      onSelected: _openProfileTool,
      itemBuilder: (_) => [
        PopupMenuItem(value: '/drafts', child: Text(l10n.draftsTitle)),
        PopupMenuItem(
          value: '/my-course-reviews',
          child: Text(l10n.myCourseReviewsTitle),
        ),
        PopupMenuItem(value: '/my-content', child: Text(l10n.profileContent)),
        PopupMenuItem(value: '/recycle-bin', child: Text(l10n.profileTrash)),
        if (_canModerate)
          PopupMenuItem(
            value: '/moderation',
            child: Text(l10n.profileModeration),
          ),
        if (_canAccessAdmin)
          PopupMenuItem(value: '/admin', child: Text(l10n.profileAdmin)),
        if (_canManageCourses) ...[
          PopupMenuItem(
            value: '/moderation/courses',
            child: Text(l10n.coursesManagement),
          ),
          PopupMenuItem(
            value: '/moderation/course-reviews',
            child: Text(l10n.coursesReviewModeration),
          ),
        ],
        const PopupMenuDivider(),
        PopupMenuItem(
          value: '/settings/account',
          child: Text(l10n.profileSecurity),
        ),
        PopupMenuItem(value: '/settings', child: Text(l10n.settingsTitle)),
      ],
    );
    return glass ? GfGlassSurface(child: menu) : menu;
  }

  Widget? _profileButtons(UserProfileProps props) {
    final l10n = AppLocalizations.of(context);
    final user = props.user;
    final List<Widget> actions = <Widget>[];
    if (user.isSelf || props.isOwnProfile) {
      actions.add(
        OutlinedButton(
          style: OutlinedButton.styleFrom(
            minimumSize: const Size(96, 44),
            shape: const StadiumBorder(),
            foregroundColor: GfTheme.colorsOf(context).baseContent,
            side: BorderSide(color: GfTheme.colorsOf(context).line),
            textStyle: Theme.of(context).textTheme.labelLarge?.copyWith(
              fontSize: 14,
              fontWeight: FontWeight.w700,
            ),
          ),
          onPressed: () => _openProfileTool('/settings/profile?edit=1'),
          child: Text(l10n.settingsEditProfile, textAlign: TextAlign.center),
        ),
      );
    } else {
      final notes = PrivateNotesScope.of(context);
      if (notes != null && notes.ownerId > 0 && notes.ownerId != user.userId) {
        actions.add(
          PrivateNoteButton(userId: user.userId, username: user.username),
        );
      }
      if (props.canFollow) {
        actions.add(
          GfFollowButton(
            following: _following,
            label: _following ? l10n.profileFollowing : l10n.profileFollow,
            busy: _followBusy,
            onPressed: () => _toggleFollow(user),
          ),
        );
      }
      if (props.canMessage && props.messageUrl.trim().isNotEmpty) {
        actions.add(
          IconButton.outlined(
            icon: const GfSymbol('mail', size: 20),
            tooltip: l10n.messagesNew,
            constraints: const BoxConstraints(minWidth: 44, minHeight: 44),
            onPressed: () => context.push(props.messageUrl),
          ),
        );
      }
    }

    return actions.isEmpty
        ? null
        : Wrap(
            alignment: WrapAlignment.end,
            spacing: 8,
            runSpacing: 8,
            children: actions,
          );
  }

  double _actionBandHeight(
    BuildContext context,
    UserProfileProps props,
    double width,
  ) {
    final scaler = MediaQuery.textScalerOf(context);
    if (props.user.isSelf || props.isOwnProfile) {
      return GfUserCardHeader.actionHeightFor(scaler);
    }

    // Match the action Wrap's available width, padding and actual labels before
    // assigning the sliver extent. A fixed band clips multi-row peer actions.
    final availableWidth = math.max(
      1.0,
      width - GfUserCardHeader.actionPadding.horizontal,
    );
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final direction = Directionality.of(context);
    final labelStyle =
        theme.textTheme.labelLarge ?? const TextStyle(fontSize: 14);
    final buttons = <Size>[];

    Size measure(
      String label, {
      required TextStyle textStyle,
      required EdgeInsetsGeometry padding,
      required Size minimumSize,
      ButtonStyle? themeStyle,
      double extraWidth = 0,
    }) {
      final density = themeStyle?.visualDensity ?? theme.visualDensity;
      final adjustment = density.baseSizeAdjustment;
      final insets = padding
          .add(
            EdgeInsets.symmetric(
              horizontal: math.max(0, adjustment.dx),
              vertical: adjustment.dy,
            ),
          )
          .clamp(EdgeInsets.zero, EdgeInsetsGeometry.infinity)
          .resolve(direction);
      final painter =
          TextPainter(
            text: TextSpan(
              text: label,
              style: MediaQuery.boldTextOf(context)
                  ? textStyle.merge(
                      const TextStyle(fontWeight: FontWeight.bold),
                    )
                  : textStyle,
            ),
            textDirection: direction,
            textScaler: scaler,
            locale: Localizations.localeOf(context),
          )..layout(
            maxWidth: math.max(
              1,
              availableWidth - insets.horizontal - extraWidth,
            ),
          );
      final padded =
          (themeStyle?.tapTargetSize ?? theme.materialTapTargetSize) ==
          MaterialTapTargetSize.padded;
      final size = Size(
        math
            .min(
              availableWidth,
              math.max(
                minimumSize.width + adjustment.dx,
                painter.width + insets.horizontal + extraWidth,
              ),
            )
            .ceilToDouble(),
        math
            .max(
              padded ? 48 + adjustment.dy : 0,
              math.max(
                minimumSize.height + adjustment.dy,
                painter.height + insets.vertical,
              ),
            )
            .ceilToDouble(),
      );
      painter.dispose();
      return size;
    }

    final notes = PrivateNotesScope.of(context);
    if (notes != null &&
        notes.ownerId > 0 &&
        notes.ownerId != props.user.userId) {
      final style = TextButtonTheme.of(context).style;
      final states = <WidgetState>{
        if (!notes.ready && !notes.failed) WidgetState.disabled,
      };
      buttons.add(
        measure(
          notes.failed ? l10n.commonRetry : l10n.privateNoteEdit,
          textStyle: style?.textStyle?.resolve(states) ?? labelStyle,
          padding:
              style?.padding?.resolve(states) ??
              ButtonStyleButton.scaledPadding(
                theme.useMaterial3
                    ? const EdgeInsets.symmetric(horizontal: 12, vertical: 8)
                    : const EdgeInsets.all(8),
                const EdgeInsets.symmetric(horizontal: 8),
                const EdgeInsets.symmetric(horizontal: 4),
                scaler.scale(labelStyle.fontSize ?? 14) / 14,
              ),
          minimumSize:
              style?.minimumSize?.resolve(states) ??
              Size(64, theme.useMaterial3 ? 40 : 36),
          themeStyle: style,
        ),
      );
    }
    if (props.canFollow) {
      buttons.add(
        measure(
          _following ? l10n.profileFollowing : l10n.profileFollow,
          textStyle: labelStyle.copyWith(
            fontSize: 14,
            fontWeight: FontWeight.w700,
          ),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          minimumSize: const Size(96, 44),
          themeStyle: OutlinedButtonTheme.of(context).style,
          extraWidth: _followBusy ? 24 : 0,
        ),
      );
    }
    if (props.canMessage && props.messageUrl.trim().isNotEmpty) {
      buttons.add(const Size(48, 48));
    }

    double rowWidth = 0;
    double rowHeight = 0;
    double totalHeight = 0;
    for (final button in buttons) {
      if (rowWidth > 0 && rowWidth + 8 + button.width > availableWidth) {
        totalHeight += rowHeight + 8;
        rowWidth = 0;
        rowHeight = 0;
      }
      rowWidth += (rowWidth > 0 ? 8 : 0) + button.width;
      rowHeight = math.max(rowHeight, button.height);
    }
    return math.max(
      GfUserCardHeader.minimumActionHeight,
      totalHeight + rowHeight + GfUserCardHeader.actionPadding.vertical,
    );
  }

  Widget? _wornBadge(UserCardPayload user) => user.wornBadge == null
      ? null
      : UserWornBadge(user.wornBadge!, avatarSize: 88);

  Widget _immersiveHeader(
    BuildContext context,
    UserProfileProps props,
  ) => SliverLayoutBuilder(
    builder: (context, constraints) {
      final colors = GfTheme.colorsOf(context);
      final l10n = AppLocalizations.of(context);
      final topInset = MediaQuery.paddingOf(context).top;
      final width = constraints.crossAxisExtent;
      final coverHeight = GfUserCard.coverHeightFor(width, topInset: topInset);
      bool expanded(BuildContext context) {
        final settings = context
            .dependOnInheritedWidgetOfExactType<FlexibleSpaceBarSettings>();
        return settings == null ||
            settings.currentExtent > settings.minExtent + 12;
      }

      final actionHeight = _actionBandHeight(context, props, width);
      return SliverAppBar(
        key: const Key('profile-cover-navigation'),
        pinned: true,
        expandedHeight: coverHeight + actionHeight - topInset,
        toolbarHeight: 56,
        elevation: 0,
        scrolledUnderElevation: 0,
        backgroundColor: colors.base100,
        foregroundColor: colors.baseContent,
        automaticallyImplyLeading: false,
        centerTitle: false,
        titleSpacing: Navigator.canPop(context) ? 0 : 16,
        titleTextStyle: TextStyle(
          color: colors.baseContent,
          fontSize: 18,
          fontWeight: FontWeight.w700,
        ),
        leadingWidth: 64,
        leading: Navigator.canPop(context)
            ? Builder(
                builder: (context) => Padding(
                  padding: const EdgeInsetsDirectional.only(
                    start: 12,
                    top: 6,
                    bottom: 6,
                  ),
                  child: expanded(context)
                      ? GfGlassIconButton(
                          symbol: 'arrow-left',
                          tooltip: l10n.commonBack,
                          onPressed: () => Navigator.maybePop(context),
                        )
                      : GfIconButton(
                          symbol: 'arrow-left',
                          tooltip: l10n.commonBack,
                          onPressed: () => Navigator.maybePop(context),
                        ),
                ),
              )
            : null,
        title: Builder(
          builder: (context) => expanded(context)
              ? const SizedBox.shrink()
              : _profileTitle(context, l10n),
        ),
        actions: [
          Builder(
            builder: (context) => Padding(
              padding: const EdgeInsetsDirectional.only(end: 12),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                spacing: 8,
                children: _profileActions(
                  context,
                  l10n,
                  glass: expanded(context),
                ),
              ),
            ),
          ),
        ],
        flexibleSpace: Builder(
          builder: (context) {
            final isExpanded = expanded(context);
            return AnnotatedRegion<SystemUiOverlayStyle>(
              value:
                  isExpanded || Theme.of(context).brightness == Brightness.dark
                  ? SystemUiOverlayStyle.light
                  : SystemUiOverlayStyle.dark,
              child: Stack(
                fit: StackFit.expand,
                children: [
                  FlexibleSpaceBar(
                    collapseMode: CollapseMode.pin,
                    // Keep the cover, complete avatar and action band in the same
                    // sliver; a negative overflow into a later sliver is occluded.
                    background: IgnorePointer(
                      ignoring: !isExpanded,
                      child: ExcludeSemantics(
                        excluding: !isExpanded,
                        child: GfUserCardHeader(
                          coverHeight: coverHeight,
                          actionHeight: actionHeight,
                          coverUrl: resolveApiAssetUrl(
                            props.user.profileCoverUrl,
                          ),
                          avatarUrl: resolveApiAssetUrl(props.user.avatarUrl),
                          avatarBadge: _wornBadge(props.user),
                          actions: _profileButtons(props),
                        ),
                      ),
                    ),
                  ),
                  if (isExpanded)
                    Positioned(
                      left: 0,
                      right: 0,
                      top: 0,
                      height: topInset + 64,
                      child: const IgnorePointer(
                        child: DecoratedBox(
                          decoration: BoxDecoration(
                            gradient: LinearGradient(
                              begin: Alignment.topCenter,
                              end: Alignment.bottomCenter,
                              colors: [
                                Color(0x99000000),
                                Color(0x88000000),
                                Colors.transparent,
                              ],
                              stops: [0, .6, 1],
                            ),
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            );
          },
        ),
      );
    },
  );

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    ref.listen(offlineCacheEpochProvider, (previous, next) {
      if (previous == next) return;
      setState(_resetStreams);
      _load();
    });
    return Scaffold(
      appBar:
          !widget.connectionsOnly &&
              (_page.isLoading || _page.valueOrNull != null)
          ? null
          : GfAppBar(
              centerTitle: false,
              title: _profileTitle(context, l10n),
              actions: _profileActions(context, l10n),
            ),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: _page.when(
            loading: () => _profileLoading(context, l10n),
            error: (e, _) => _isShellProfile
                ? _ProfileErrorBody(
                    message: resolveErrorMessage(l10n, e),
                    onRetry: _load,
                    showLogin: _loginRequired,
                    l10n: l10n,
                  )
                : GfErrorRetry(
                    message: resolveErrorMessage(l10n, e),
                    onRetry: _load,
                  ),
            data: (UserProfileProps props) {
              final tabs = _tabs(props, l10n);
              return GfScrollToTop(
                semanticLabel: l10n.commonBackToTop,
                controller: _isShellProfile ? _scrollToTopController : null,
                threshold: 360,
                builder: (BuildContext context, ScrollController controller) {
                  return AppRefreshIndicator(
                    edgeOffset: widget.connectionsOnly
                        ? 0
                        : MediaQuery.paddingOf(context).top + 56,
                    onRefresh: () => _load(),
                    child: CustomScrollView(
                      controller: controller,
                      physics: const AlwaysScrollableScrollPhysics(),
                      slivers: <Widget>[
                        if (!widget.connectionsOnly)
                          _immersiveHeader(context, _headerProps ?? props),
                        if (!widget.connectionsOnly)
                          SliverToBoxAdapter(
                            child: _profileCard(_headerProps ?? props),
                          ),
                        const SliverToBoxAdapter(child: GfDivider()),
                        if (tabs.isNotEmpty)
                          SliverLayoutBuilder(
                            builder: (context, constraints) =>
                                SliverPersistentHeader(
                                  pinned: true,
                                  delegate: _ProfileTabsHeader(
                                    height: math.max(
                                      52,
                                      MediaQuery.textScalerOf(
                                                context,
                                              ).scale(16) *
                                              1.4 +
                                          24,
                                    ),
                                    child: _ProfileTabs(
                                      tabs: tabs,
                                      index: tabs.indexWhere(
                                        (tab) => tab.key == _stream,
                                      ),
                                      onChanged: (index) => _selectStream(
                                        tabs[index].key,
                                        controller,
                                        math.max(
                                          0,
                                          constraints.precedingScrollExtent -
                                              (widget.connectionsOnly
                                                  ? 0
                                                  : MediaQuery.paddingOf(
                                                          context,
                                                        ).top +
                                                        56),
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                          ),
                        if (_streamLoading)
                          _ProfileStreamSkeleton(
                            connectionsOnly: widget.connectionsOnly,
                          )
                        else if (_streamError != null)
                          SliverToBoxAdapter(
                            child: GfErrorRetry(
                              message: resolveErrorMessage(l10n, _streamError!),
                              onRetry: () {
                                _load(streamChange: true);
                              },
                            ),
                          ),
                        if (!_streamLoading && _active.props != null)
                          // Key each stream so recycled SliverList geometry cannot
                          // shift its restored scroll offset.
                          _ProfileBody(
                            key: ValueKey(_stream),
                            props: props,
                            selectedKey: _stream,
                            following: _connectionFollowing,
                            busy: _connectionBusy,
                            onFollow: _toggleConnection,
                            onReturn: _syncReturnedInteractions,
                            onInteraction: _toggleInteraction,
                            interactionBusy: _interactionBusy,
                          ),
                        if (!_streamLoading &&
                            _streamError == null &&
                            props.pagination.hasNext)
                          SliverToBoxAdapter(
                            child: GfListFooter(
                              key: ValueKey(_stream),
                              error: _active.paginationError == null
                                  ? null
                                  : resolveErrorMessage(
                                      l10n,
                                      _active.paginationError!,
                                    ),
                              progressKey: (_stream, props.pagination.nextUrl),
                              hasMore: props.pagination.hasNext,
                              loading: _loadingMore,
                              onLoadMore: () => _loadMore(props),
                            ),
                          ),

                        // Keep short streams from clamping a restored offset.
                        // First visits retain only the header-collapse offset.
                        SliverLayoutBuilder(
                          builder: (context, constraints) => SliverToBoxAdapter(
                            child: SizedBox(
                              height: math.max(
                                32,
                                _minimumScrollOffset +
                                    constraints.viewportMainAxisExtent -
                                    constraints.precedingScrollExtent,
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  );
                },
              );
            },
          ),
        ),
      ),
    );
  }

  Widget _profileLoading(BuildContext context, AppLocalizations l10n) {
    if (widget.connectionsOnly) {
      return const GfProfileConnectionsSkeleton(
        key: Key('profile-connections-skeleton'),
      );
    }
    return Stack(
      fit: StackFit.expand,
      children: [
        const GfProfileSkeleton(),
        Positioned(
          left: 0,
          right: 0,
          top: 0,
          child: SafeArea(
            bottom: false,
            child: SizedBox(
              height: 56,
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: Row(
                  spacing: 8,
                  children: [
                    if (Navigator.canPop(context))
                      GfGlassIconButton(
                        symbol: 'arrow-left',
                        tooltip: l10n.commonBack,
                        onPressed: () => Navigator.maybePop(context),
                      ),
                    const Spacer(),
                    ..._profileActions(context, l10n, glass: true),
                  ],
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }

  String _compactProfileDate(String value, AppLocalizations l10n) {
    final date = DateTime.tryParse(formatDate(value));
    if (date == null) return formatDate(value);
    return DateFormat.yMd(l10n.localeName)
        .format(date)
        .replaceFirst(
          date.year.toString(),
          (date.year % 100).toString().padLeft(2, '0'),
        );
  }

  Widget _profileMetaRow({
    required UserCardPayload user,
    required List<(String, Uri, String?)> links,
    required String? joinedAt,
    required String? lastActive,
    required TextStyle textStyle,
    required AppLocalizations l10n,
  }) {
    final compactJoined = joinedAt == null
        ? null
        : _compactProfileDate(user.createdAt, l10n);
    final activeDate = DateTime.tryParse(
      formatDateTime(user.lastActiveTime).replaceFirst(' ', 'T'),
    );
    final compactActive = lastActive == null
        ? null
        : activeDate == null
        ? timeAgo(user.lastActiveTime, l10n: l10n)
        : DateUtils.isSameDay(activeDate, DateTime.now())
        ? DateFormat.Hm(l10n.localeName).format(activeDate)
        : _compactProfileDate(user.lastActiveTime, l10n);
    final iconColor = GfTheme.colorsOf(context).iconMuted;

    return LayoutBuilder(
      builder: (layoutContext, constraints) {
        final direction = Directionality.of(layoutContext);
        final scaler = MediaQuery.textScalerOf(layoutContext);
        double textWidth(String value) {
          final painter = TextPainter(
            text: TextSpan(text: value, style: textStyle),
            textDirection: direction,
            textScaler: scaler,
            maxLines: 1,
          )..layout();
          final width = painter.width;
          painter.dispose();
          return width;
        }

        final iconExtent = links.length >= 6
            ? 24.0
            : links.length >= 4
            ? 28.0
            : 32.0;
        double requiredWidth(String? joined, String? active) {
          var width = links.length * iconExtent;
          if (joined != null) width += 18 + textWidth(joined);
          if (active != null) width += 18 + textWidth(active);
          if (joined != null && active != null) width += 8;
          if (links.isNotEmpty && (joined != null || active != null)) {
            width += 8;
          }
          return width;
        }

        final compact =
            requiredWidth(joinedAt, lastActive) > constraints.maxWidth;
        final joinedText = compact ? compactJoined : joinedAt;
        final activeText = compact ? compactActive : lastActive;
        final rowWidth = math.max(
          constraints.maxWidth,
          requiredWidth(joinedText, activeText),
        );
        Widget timestamp(String icon, String visible, String full) => Tooltip(
          message: full,
          excludeFromSemantics: true,
          child: Semantics(
            label: full,
            child: ExcludeSemantics(
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  GfSymbol(icon, size: 14, color: iconColor),
                  const SizedBox(width: 4),
                  Text(visible, maxLines: 1, softWrap: false, style: textStyle),
                ],
              ),
            ),
          ),
        );

        return SizedBox(
          width: constraints.maxWidth,
          child: FittedBox(
            fit: BoxFit.scaleDown,
            alignment: Alignment.centerLeft,
            child: SizedBox(
              width: rowWidth,
              child: Row(
                children: [
                  if (joinedText != null)
                    timestamp('calendar-days', joinedText, joinedAt!),
                  if (joinedText != null && activeText != null)
                    const SizedBox(width: 8),
                  if (activeText != null)
                    timestamp('clock', activeText, lastActive!),
                  if (links.isNotEmpty &&
                      (joinedText != null || activeText != null)) ...[
                    const SizedBox(width: 8),
                    const Spacer(),
                  ],
                  for (final (label, uri, provider) in links)
                    Semantics(
                      label: label,
                      button: true,
                      child: Tooltip(
                        message: label,
                        excludeFromSemantics: true,
                        child: IconButton(
                          style: IconButton.styleFrom(
                            padding: EdgeInsets.zero,
                            fixedSize: Size.square(iconExtent),
                            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                            visualDensity: VisualDensity.compact,
                          ),
                          constraints: BoxConstraints.tightFor(
                            width: iconExtent,
                            height: iconExtent,
                          ),
                          icon: GfSocialIcon(
                            provider,
                            size: iconExtent == 24 ? 18 : 20,
                          ),
                          onPressed: () async {
                            try {
                              if (!await launchUrl(
                                uri,
                                mode: LaunchMode.externalApplication,
                              )) {
                                throw StateError('Could not open profile link');
                              }
                            } catch (error) {
                              if (mounted) {
                                showGfToast(
                                  context,
                                  resolveErrorMessage(l10n, error),
                                  error: true,
                                );
                              }
                            }
                          },
                        ),
                      ),
                    ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _profileCard(UserProfileProps props) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final UserCardPayload user = props.user;
    final colors = GfTheme.colorsOf(context);
    final Map<String, GfUserBadge> badges = <String, GfUserBadge>{};
    for (final badge in (user.displayBadges ?? user.badges.take(5))) {
      badges['earned:${badge.code}'] = GfUserBadge(
        label: badge.name,
        color: userBadgeColor(badge),
        icon: UserBadgeArtwork(badge, size: 24),
        description: badge.description,
        onTap: () => showUserBadgeDetails(context, badge),
      );
    }
    final links = publicProfileLinks(user);
    final joinedAt = user.createdAt.trim().isEmpty
        ? null
        : l10n.profileJoinedAt(formatDate(user.createdAt));
    final lastActive = user.lastActiveTime.trim().isEmpty
        ? null
        : l10n.profileLastActive(timeAgo(user.lastActiveTime, l10n: l10n));
    final metaStyle = GfTheme.typographyOf(
      context,
    ).caption.copyWith(color: colors.baseContent.withValues(alpha: .55));
    return GfUserCard(
      showHeader: false,
      coverUrl: resolveApiAssetUrl(user.profileCoverUrl),
      avatarUrl: resolveApiAssetUrl(user.avatarUrl),
      avatarBadge: _wornBadge(user),
      name: privateDisplayName(
        context,
        user.userId,
        user.username,
        user.nickname,
      ),
      nameBadges: [
        if (user.isAdmin)
          GfBadge(
            label: l10n.profileRoleAdmin,
            variant: GfBadgeVariant.warning,
            radius: 4,
          ),
        if (user.isOnline)
          GfBadge(
            label: l10n.profileOnline,
            variant: GfBadgeVariant.success,
            icon: const GfSymbol('signal-stream', size: 12),
          ),
      ],
      username: user.username,
      bio: user.bio,
      signature: user.signature,
      details: links.isEmpty && joinedAt == null && lastActive == null
          ? null
          : _profileMetaRow(
              user: user,
              links: links,
              joinedAt: joinedAt,
              lastActive: lastActive,
              textStyle: metaStyle,
              l10n: l10n,
            ),
      coloredBadges: badges.values.toList(growable: false),
      stats: <(String, String)>[
        (l10n.profileFollowingCount, formatNumber(user.followingCount)),
        (l10n.profileFollowers, formatNumber(user.followerCount)),
        (l10n.profileTopics, formatNumber(user.topicCount)),
        (l10n.profileReplies, formatNumber(user.replyCount)),
        (l10n.profileLikes, formatNumber(user.likeReceivedCount)),
      ],
      statActions: {
        0: () => context.push('/u/${user.userId}/following'),
        1: () => context.push('/u/${user.userId}/followers'),
      },
    );
  }
}

class _ProfileErrorBody extends StatelessWidget {
  const _ProfileErrorBody({
    required this.message,
    required this.onRetry,
    required this.showLogin,
    required this.l10n,
  });

  final String message;
  final VoidCallback onRetry;
  final bool showLogin;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.only(top: 12, bottom: 32),
      children: <Widget>[
        GfErrorRetry(message: message, onRetry: onRetry),
        if (showLogin) ...<Widget>[
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: GfButton(
              icon: const GfSymbol('user-round', size: 18),
              label: l10n.loginModeLogin,
              onPressed: () => context.push('/login'),
            ),
          ),
          const SizedBox(height: 12),
        ],
        _AccountShortcuts(l10n: l10n),
      ],
    );
  }
}

class _ProfileTabsHeader extends SliverPersistentHeaderDelegate {
  const _ProfileTabsHeader({required this.height, required this.child});
  final double height;
  final Widget child;
  @override
  double get minExtent => height;
  @override
  double get maxExtent => height;
  @override
  Widget build(
    BuildContext context,
    double shrinkOffset,
    bool overlapsContent,
  ) => Material(
    color: GfTheme.colorsOf(context).base100,
    child: Column(
      children: [
        Expanded(child: child),
        const GfDivider(),
      ],
    ),
  );
  @override
  bool shouldRebuild(covariant _ProfileTabsHeader oldDelegate) => true;
}

class _ProfileTabs extends StatefulWidget {
  const _ProfileTabs({
    required this.tabs,
    required this.index,
    required this.onChanged,
  });

  final List<TabItemPayload> tabs;
  final int index;
  final ValueChanged<int> onChanged;

  @override
  State<_ProfileTabs> createState() => _ProfileTabsState();
}

class _ProfileTabsState extends State<_ProfileTabs>
    with SingleTickerProviderStateMixin {
  static const _animationDuration = Duration(milliseconds: 220);
  static const _animationCurve = Curves.easeInOutCubic;

  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: _animationDuration,
    value: 1,
  );
  List<double> _fromShares = [];
  List<double> _displayedShares = [];
  Offset? _fromSegmentShares;
  Offset? _displayedSegmentShares;
  bool _disableAnimations = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final disableAnimations = MediaQuery.disableAnimationsOf(context);
    if (disableAnimations && !_disableAnimations) {
      _fromShares = List.of(_displayedShares);
      _fromSegmentShares = _displayedSegmentShares;
      _controller.value = 1;
    }
    _disableAnimations = disableAnimations;
  }

  @override
  void didUpdateWidget(covariant _ProfileTabs oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.index == oldWidget.index) return;
    _fromShares = List.of(_displayedShares);
    _fromSegmentShares = _displayedSegmentShares;
    if (_disableAnimations) {
      _controller.value = 1;
    } else {
      _controller.forward(from: 0);
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  String _iconFor(String key) => switch (key) {
    'timeline' => 'activity',
    'topics' => 'file-text',
    'likes' => 'heart',
    'bookmarks' => 'bookmark',
    'badges' => 'award',
    'following' => 'user-round-check',
    'followers' => 'users-round',
    _ => 'circle-user-round',
  };

  double _labelWidth(BuildContext context, String label) {
    final style = DefaultTextStyle.of(
      context,
    ).style.copyWith(fontSize: 16, fontWeight: FontWeight.w600);
    final painter = TextPainter(
      text: TextSpan(text: label, style: style),
      maxLines: 1,
      textDirection: Directionality.of(context),
      textScaler: MediaQuery.textScalerOf(context),
    )..layout();
    final width = painter.width;
    painter.dispose();
    return width;
  }

  @override
  Widget build(BuildContext context) {
    final selectedIndex = widget.index < 0 ? 0 : widget.index;
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, _) => LayoutBuilder(
        builder: (context, constraints) {
          final colors = GfTheme.colorsOf(context);
          final animationDuration = _disableAnimations
              ? Duration.zero
              : _animationDuration;
          final labelWidths = [
            for (final tab in widget.tabs)
              _labelWidth(context, tab.label ?? tab.key),
          ];
          final expandedWidths = [
            for (final width in labelWidths) math.max(48.0, width + 42),
          ];
          final rowWidth = math.max(
            constraints.maxWidth,
            widget.tabs.length <= 2
                ? expandedWidths.reduce((a, b) => a > b ? a : b) *
                      widget.tabs.length
                : expandedWidths[selectedIndex] +
                      48.0 * (widget.tabs.length - 1),
          );
          final targetWidths = widget.tabs.length <= 2
              ? List<double>.filled(
                  widget.tabs.length,
                  rowWidth / widget.tabs.length,
                )
              : [
                  for (int i = 0; i < widget.tabs.length; i++)
                    i == selectedIndex
                        ? expandedWidths[i]
                        : (rowWidth - expandedWidths[selectedIndex]) /
                              (widget.tabs.length - 1),
                ];
          final targetShares = [
            for (final width in targetWidths) width / rowWidth,
          ];
          var targetSegmentLeft = 0.0;
          for (int i = 0; i < selectedIndex; i++) {
            targetSegmentLeft += targetShares[i];
          }
          final targetSegment = Offset(
            targetSegmentLeft,
            targetShares[selectedIndex],
          );
          if (_fromShares.length != targetShares.length ||
              _fromSegmentShares == null) {
            _fromShares = List.of(targetShares);
            _fromSegmentShares = targetSegment;
          }

          final progress = _disableAnimations
              ? 1.0
              : _animationCurve.transform(_controller.value);
          final shares = [
            for (int i = 0; i < targetShares.length; i++)
              _fromShares[i] + (targetShares[i] - _fromShares[i]) * progress,
          ];
          _displayedShares = List.of(shares);
          final widths = [for (final share in shares) share * rowWidth];
          final fromSegment = _fromSegmentShares!;
          final activeSegmentShares = Offset(
            fromSegment.dx + (targetSegment.dx - fromSegment.dx) * progress,
            fromSegment.dy + (targetSegment.dy - fromSegment.dy) * progress,
          );
          _displayedSegmentShares = activeSegmentShares;
          final activeLeft = activeSegmentShares.dx * rowWidth;
          final activeWidth = activeSegmentShares.dy * rowWidth;
          final indicatorWidth = math.min(
            64.0,
            math.max(40.0, activeWidth * .72),
          );
          final indicatorLeft = activeLeft + (activeWidth - indicatorWidth) / 2;
          final height = math.max(
            52.0,
            MediaQuery.textScalerOf(context).scale(16) * 1.4 + 24,
          );

          return SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: SizedBox(
              width: rowWidth,
              height: height,
              child: Stack(
                children: [
                  Row(
                    children: [
                      for (int i = 0; i < widget.tabs.length; i++)
                        SizedBox(
                          key: i == selectedIndex
                              ? const ValueKey('profile-tab-active-segment')
                              : null,
                          width: widths[i],
                          height: height,
                          child: Tooltip(
                            message: widget.tabs[i].label ?? widget.tabs[i].key,
                            excludeFromSemantics: true,
                            child: Semantics(
                              selected: i == selectedIndex,
                              button: true,
                              label: widget.tabs[i].label ?? widget.tabs[i].key,
                              child: InkWell(
                                borderRadius: BorderRadius.circular(8),
                                onTap: () => widget.onChanged(i),
                                child: Padding(
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 8,
                                  ),
                                  child: Row(
                                    mainAxisAlignment: MainAxisAlignment.center,
                                    mainAxisSize: MainAxisSize.max,
                                    children: [
                                      GfSymbol(
                                        _iconFor(widget.tabs[i].key),
                                        size: 20,
                                        color: i == selectedIndex
                                            ? colors.baseContent
                                            : colors.iconMuted,
                                      ),
                                      Flexible(
                                        fit: FlexFit.loose,
                                        child: AnimatedContainer(
                                          duration: animationDuration,
                                          curve: _animationCurve,
                                          width: i == selectedIndex
                                              ? labelWidths[i] + 6
                                              : 0,
                                          child: ClipRect(
                                            child: AnimatedOpacity(
                                              duration: animationDuration,
                                              curve: _animationCurve,
                                              opacity: i == selectedIndex
                                                  ? 1
                                                  : 0,
                                              child: Padding(
                                                padding: const EdgeInsets.only(
                                                  left: 6,
                                                ),
                                                child: ExcludeSemantics(
                                                  child: Text(
                                                    widget.tabs[i].label ??
                                                        widget.tabs[i].key,
                                                    maxLines: 1,
                                                    softWrap: false,
                                                    style:
                                                        DefaultTextStyle.of(
                                                          context,
                                                        ).style.copyWith(
                                                          fontSize: 16,
                                                          fontWeight:
                                                              FontWeight.w600,
                                                          color: colors
                                                              .baseContent,
                                                        ),
                                                  ),
                                                ),
                                              ),
                                            ),
                                          ),
                                        ),
                                      ),
                                    ],
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
                  Positioned(
                    left: indicatorLeft,
                    bottom: 0,
                    width: indicatorWidth,
                    height: 3,
                    child: DecoratedBox(
                      key: const ValueKey('profile-tab-indicator'),
                      decoration: BoxDecoration(
                        color: colors.primary,
                        borderRadius: BorderRadius.circular(3),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}

class _ProfileStreamSkeleton extends StatelessWidget {
  const _ProfileStreamSkeleton({this.connectionsOnly = false});
  final bool connectionsOnly;

  @override
  Widget build(BuildContext context) => SliverList.separated(
    itemCount: 3,
    separatorBuilder: (_, _) => const GfDivider(),
    itemBuilder: (_, _) =>
        GfProfileStreamRowSkeleton(connectionsOnly: connectionsOnly),
  );
}

class _ProfileBody extends StatelessWidget {
  const _ProfileBody({
    super.key,
    required this.props,
    required this.selectedKey,
    required this.following,
    required this.busy,
    required this.onFollow,
    required this.onReturn,
    required this.onInteraction,
    required this.interactionBusy,
  });

  final UserProfileProps props;
  final String selectedKey;
  final Map<int, bool> following;
  final Set<int> busy;
  final ValueChanged<UserConnectionPayload> onFollow;
  final VoidCallback onReturn;
  final Set<_ContentKey> interactionBusy;
  final Future<bool> Function(_ContentKey, PostReturnState, bool, bool)
  onInteraction;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return switch (selectedKey) {
      'badges' =>
        props.badges.isEmpty
            ? _empty('award', l10n.profileNoBadges)
            : _badgeGallery(context),
      'topics' => _topicRows(context, l10n),
      'likes' => _likeRows(context, l10n),
      'bookmarks' => _bookmarkRows(context, l10n),
      'following' => _connectionRows(
        context,
        props.following,
        l10n.profileEmptyFollowing,
      ),
      'followers' => _connectionRows(
        context,
        props.followers,
        l10n.profileEmptyFollowers,
      ),
      _ => _activityRows(context, l10n),
    };
  }

  Widget _badgeGallery(BuildContext context) => SliverLayoutBuilder(
    builder: (context, constraints) {
      final width = constraints.crossAxisExtent - 32;
      final textScale = MediaQuery.textScalerOf(context).scale(14) / 14;
      final columns = ((width + 12) / (148 * textScale + 12)).floor().clamp(
        1,
        4,
      );
      final rows = (props.badges.length / columns).ceil();
      return SliverPadding(
        padding: const EdgeInsets.all(16),
        sliver: SliverList.separated(
          itemCount: rows,
          separatorBuilder: (_, _) => const SizedBox(height: 12),
          itemBuilder: (context, row) => IntrinsicHeight(
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                for (var column = 0; column < columns; column++) ...[
                  if (column > 0) const SizedBox(width: 12),
                  Expanded(
                    child: row * columns + column >= props.badges.length
                        ? const SizedBox.shrink()
                        : _badgeCard(
                            context,
                            props.badges[row * columns + column],
                          ),
                  ),
                ],
              ],
            ),
          ),
        ),
      );
    },
  );

  Widget _badgeCard(BuildContext context, UserBadgePayload badge) =>
      GfAchievementCard(
        title: badge.name,
        description: badge.description,
        color: userBadgeColor(badge),
        icon: UserBadgeArtwork(badge),
        onTap: () => showUserBadgeDetails(context, badge),
      );

  Widget _empty(String symbol, String message) {
    return SliverToBoxAdapter(
      child: GfEmpty(symbol: symbol, message: message),
    );
  }

  Widget _activityRows(BuildContext context, AppLocalizations l10n) {
    if (props.activities.isEmpty) {
      return _empty('sparkles', l10n.profileEmptyActivity);
    }
    return SliverList.builder(
      itemCount: props.activities.length,
      itemBuilder: (BuildContext context, int index) {
        final UserActivityPayload activity = props.activities[index];
        final String? route = _activityRoute(activity);
        final action = switch (activity.action) {
          1 => 'signup',
          2 => 'post',
          3 => 'like',
          4 => 'follow',
          5 => 'comment',
          _ => activity.label,
        };
        return GfContentRow(
          author: privateDisplayName(
            context,
            props.user.userId,
            props.user.username,
            props.user.nickname,
          ),
          avatarUrl: resolveApiAssetUrl(props.user.avatarUrl),
          contextSymbol: switch (action) {
            'signup' => 'user-round',
            'post' => 'square-pen',
            'like' => 'heart',
            'follow' => 'user-round-plus',
            'comment' => 'message-circle',
            _ => 'sparkles',
          },
          contextLabel: switch (action) {
            'signup' => l10n.profileActionSignup,
            'post' => l10n.profileActionPost,
            'like' => l10n.profileActionLike,
            'follow' => l10n.profileActionFollow,
            'comment' => l10n.profileActionComment,
            _ => activity.label,
          },
          text: activity.contentPreview,
          time: timeAgo(activity.createdAt, l10n: l10n),
          footer:
              _activityKey(activity) == null ||
                  activity.liked == null ||
                  activity.bookmarked == null
              ? null
              : Wrap(
                  alignment: WrapAlignment.start,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  spacing: 8,
                  runSpacing: 4,
                  children: [
                    Tooltip(
                      message: l10n.topicLike,
                      child: TextButton.icon(
                        onPressed:
                            interactionBusy.contains(_activityKey(activity))
                            ? null
                            : () => onInteraction(
                                _activityKey(activity)!,
                                (
                                  liked: activity.liked!,
                                  bookmarked: activity.bookmarked!,
                                  likeCount: activity.likeCount ?? 0,
                                ),
                                false,
                                !activity.liked!,
                              ),
                        icon: GfSymbol(
                          activity.liked! ? 'heart-filled' : 'heart',
                          size: 18,
                          color: activity.liked!
                              ? GfTheme.colorsOf(context).error
                              : GfTheme.colorsOf(context).iconMuted,
                        ),
                        label: Text(
                          formatNumber(activity.likeCount ?? 0),
                          semanticsLabel: '${activity.likeCount ?? 0}',
                        ),
                        style: TextButton.styleFrom(
                          minimumSize: const Size(44, 44),
                          padding: const EdgeInsets.symmetric(horizontal: 4),
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                          textStyle: const TextStyle(fontSize: 13),
                          foregroundColor: GfTheme.colorsOf(context).iconMuted,
                        ),
                      ),
                    ),
                    IconButton(
                      tooltip: activity.bookmarked!
                          ? l10n.topicBookmarked
                          : l10n.topicBookmark,
                      onPressed:
                          interactionBusy.contains(_activityKey(activity))
                          ? null
                          : () => onInteraction(
                              _activityKey(activity)!,
                              (
                                liked: activity.liked!,
                                bookmarked: activity.bookmarked!,
                                likeCount: activity.likeCount ?? 0,
                              ),
                              true,
                              !activity.bookmarked!,
                            ),
                      icon: GfSymbol(
                        activity.bookmarked! ? 'bookmark-filled' : 'bookmark',
                        size: 18,
                        color: activity.bookmarked!
                            ? GfTheme.colorsOf(context).primary
                            : GfTheme.colorsOf(context).iconMuted,
                      ),
                    ),
                  ],
                ),
          onTap: route == null
              ? null
              : () async {
                  await context.push(route);
                  onReturn();
                },
        );
      },
    );
  }

  Widget _topicRows(BuildContext context, AppLocalizations l10n) {
    if (props.topics.isEmpty) {
      return _empty('file-text', l10n.profileEmptyTopics);
    }
    return SliverList.builder(
      itemCount: props.topics.length,
      itemBuilder: (BuildContext context, int index) {
        final topic = props.topics[index];
        final key = (false, topic.id);
        final known = topic.liked != null && topic.bookmarked != null;
        final state = (
          liked: topic.liked ?? false,
          bookmarked: topic.bookmarked ?? false,
          likeCount: topic.likeCount,
        );
        return buildTopicFeedCard(
          context,
          topic,
          onReturn: onReturn,
          onLike: known
              ? (target) => onInteraction(key, state, false, target)
              : null,
          onBookmark: known
              ? (target) => onInteraction(key, state, true, target)
              : null,
        );
      },
    );
  }

  Widget _likeRows(BuildContext context, AppLocalizations l10n) {
    if (props.likes.isEmpty) {
      return _empty('heart', l10n.profileEmptyLikes);
    }
    return SliverList.builder(
      itemCount: props.likes.length,
      itemBuilder: (BuildContext context, int index) {
        final UserLikePayload like = props.likes[index];
        return _contentRow(
          context,
          author: like.author,
          title: like.title,
          excerpt: like.excerpt ?? '',
          thumbnail: like.thumbnailUrl ?? '',
          time: like.likedAt,
          route: '/p/${like.topicId}',
        );
      },
    );
  }

  Widget _bookmarkRows(BuildContext context, AppLocalizations l10n) {
    if (props.bookmarks.isEmpty) {
      return _empty('bookmark', l10n.profileEmptyBookmarks);
    }
    return SliverList.builder(
      itemCount: props.bookmarks.length,
      itemBuilder: (BuildContext context, int index) {
        final UserBookmarkPayload bookmark = props.bookmarks[index];
        return _contentRow(
          context,
          author: bookmark.author,
          title: bookmark.title,
          excerpt: bookmark.excerpt ?? '',
          thumbnail: bookmark.thumbnailUrl ?? '',
          time: bookmark.bookmarkedAt,
          route: bookmark.postNo != null && bookmark.postNo! > 0
              ? '/p/${bookmark.topicId}?postNo=${bookmark.postNo}'
              : '/p/${bookmark.topicId}',
        );
      },
    );
  }

  Widget _connectionRows(
    BuildContext context,
    List<UserConnectionPayload> users,
    String emptyMessage,
  ) {
    if (users.isEmpty) return _empty('users-round', emptyMessage);
    return SliverList.builder(
      itemCount: users.length,
      itemBuilder: (BuildContext context, int index) {
        final UserConnectionPayload user = users[index];
        return GfConnectionRow(
          avatarUrl: resolveApiAssetUrl(user.avatarUrl),
          name: privateDisplayName(
            context,
            user.id,
            user.username,
            user.nickname,
          ),
          username: user.username,
          bio: user.bio,
          action: user.isSelf || user.isFollowing == null
              ? null
              : GfFollowButton(
                  following: following[user.id] ?? user.isFollowing!,
                  label: (following[user.id] ?? user.isFollowing!)
                      ? AppLocalizations.of(context).profileFollowing
                      : AppLocalizations.of(context).profileFollow,
                  busy: busy.contains(user.id),
                  onPressed: () => onFollow(user),
                ),
          onTap: () => context.push('/u/${user.id}'),
        );
      },
    );
  }

  Widget _contentRow(
    BuildContext context, {
    required UserBriefPayload? author,
    required String title,
    required String excerpt,
    required String thumbnail,
    required String time,
    required String route,
  }) => GfContentRow(
    author: author == null
        ? ''
        : privateDisplayName(
            context,
            author.id,
            author.username,
            author.nickname,
          ),
    avatarUrl: resolveApiAssetUrl(author?.avatarUrl ?? ''),
    title: title,
    text: excerpt,
    thumbnailUrl: resolveApiAssetUrl(thumbnail),
    time: timeAgo(time, l10n: AppLocalizations.of(context)),
    onAuthorTap: author == null || author.id <= 0
        ? null
        : () => context.push('/u/${author.id}'),
    onTap: () => context.push(route),
  );

  String? _activityRoute(UserActivityPayload activity) {
    final route = profileActivityRoute(activity.url);
    if (route != null) return route;
    if (activity.subjectType == 'topic' && activity.subjectId > 0) {
      return '/p/${activity.subjectId}';
    }
    return null;
  }
}

class _AccountShortcuts extends StatelessWidget {
  const _AccountShortcuts({required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: GfCardList(
        children: <Widget>[
          GfSettingRow(
            symbol: 'settings',
            title: l10n.settingsTitle,
            onTap: () => context.push('/settings'),
          ),
          GfSettingRow(
            symbol: 'bell',
            title: l10n.notificationsTitle,
            onTap: () => context.go('/notifications'),
          ),
          GfSettingRow(
            symbol: 'file-text',
            title: l10n.draftsTitle,
            onTap: () => context.push('/drafts'),
          ),
        ],
      ),
    );
  }
}
