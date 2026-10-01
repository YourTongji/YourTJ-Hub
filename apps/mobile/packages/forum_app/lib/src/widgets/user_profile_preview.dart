import 'dart:math' as math;
import 'dart:ui' show ImageFilter;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../format.dart';
import '../current_user.dart';
import '../../l10n/app_localizations.dart';
import '../navigation/auth_navigation.dart';
import '../private_notes.dart';
import '../profile_links.dart';
import '../providers.dart';
import '../server_messages.dart';
import '../user_blocks.dart';
import '../asset_url.dart';
import 'status_views.dart';
import 'user_badge.dart';

typedef _CachedUserCard = ({
  UserCardPayload user,
  DateTime fetchedAt,
  int sessionEpoch,
});

final _userCardPreviewCache = <int, _CachedUserCard>{};
const _previewCacheTtl = Duration(seconds: 60);

void invalidateUserProfilePreview(int userId) =>
    _userCardPreviewCache.remove(userId);

Future<void> showUserProfilePreview(
  BuildContext context, {
  required int userId,
  required String username,
  String? nickname,
  required String avatarUrl,
  UserBadgePayload? wornBadge,
}) {
  final l10n = AppLocalizations.of(context);
  return showGeneralDialog<void>(
    context: context,
    useRootNavigator: true,

    barrierDismissible: true,
    barrierLabel: l10n.commonClose,
    barrierColor: Colors.transparent,
    transitionDuration: GfMotion.duration(context, GfMotion.layout),
    transitionBuilder: (context, animation, secondaryAnimation, child) =>
        GfFadeTransition(animation: animation, child: child),
    pageBuilder: (dialogContext, animation, secondaryAnimation) => Stack(
      children: [
        Positioned.fill(
          child: GestureDetector(
            onTap: () => Navigator.of(dialogContext).pop(),
            child: ClipRect(
              child: BackdropFilter(
                filter: ImageFilter.blur(sigmaX: 3, sigmaY: 3),
                child: ColoredBox(
                  color: Theme.of(
                    dialogContext,
                  ).colorScheme.scrim.withValues(alpha: .46),
                ),
              ),
            ),
          ),
        ),
        SafeArea(
          child: Align(
            alignment: Alignment.topCenter,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(12, 14, 12, 12),
              child: UserProfilePreview(
                rootContext: context,
                userId: userId,
                username: username,
                nickname: nickname,
                avatarUrl: resolveApiAssetUrl(avatarUrl),
                wornBadge: wornBadge,
              ),
            ),
          ),
        ),
      ],
    ),
  );
}

class UserProfilePreview extends ConsumerStatefulWidget {
  const UserProfilePreview({
    super.key,
    required this.rootContext,
    required this.userId,
    required this.username,
    required this.avatarUrl,
    this.nickname,
    this.wornBadge,
  });

  final BuildContext rootContext;
  final int userId;
  final String username;
  final String? nickname;
  final String avatarUrl;
  final UserBadgePayload? wornBadge;

  @override
  ConsumerState<UserProfilePreview> createState() => _UserProfilePreviewState();
}

class _UserProfilePreviewState extends ConsumerState<UserProfilePreview> {
  UserCardPayload? _user;
  Object? _error;
  bool _loading = true;
  bool _refreshing = false;
  bool _closing = false;
  int _request = 0;
  final _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    final cached = _userCardPreviewCache[widget.userId];
    final sessionEpoch = ref.read(offlineCacheEpochProvider);
    final validCached = cached?.sessionEpoch == sessionEpoch ? cached : null;
    if (validCached != null) {
      _user = validCached.user;
      _loading = false;
    } else if (cached != null) {
      _userCardPreviewCache.remove(widget.userId);
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (validCached == null) {
        _load();
      } else if (DateTime.now().difference(validCached.fetchedAt) >=
          _previewCacheTtl) {
        _load();
      }
    });
  }

  Future<void> _load() async {
    final request = ++_request;
    final sessionEpoch = ref.read(offlineCacheEpochProvider);
    final read = ref.read(userFollowStateProvider(widget.userId)).beginRead();
    setState(() {
      _refreshing = true;
      if (_user == null) _loading = true;
      _error = null;
    });
    try {
      final user = await ref
          .read(userRepositoryProvider)
          .getUserCard(widget.userId);
      if (!mounted ||
          request != _request ||
          ref.read(offlineCacheEpochProvider) != sessionEpoch) {
        return;
      }
      _userCardPreviewCache[widget.userId] = (
        user: user,
        fetchedAt: DateTime.now(),
        sessionEpoch: sessionEpoch,
      );
      ref
          .read(userFollowStateProvider(widget.userId))
          .acceptServerValue(user.isFollowing, read);
      setState(() {
        _user = user;
        _error = null;
      });
    } catch (error) {
      if (mounted && request == _request && _user == null) {
        setState(() => _error = error);
      }
    } finally {
      if (mounted && request == _request) {
        setState(() {
          _loading = false;
          _refreshing = false;
        });
      }
    }
  }

  Future<void> _toggleFollow(UserCardPayload user) async {
    if (user.isSelf) return;
    final router = GoRouter.of(widget.rootContext);
    final location = router.routeInformationProvider.value.uri.toString();
    final current = await ref.read(currentUserProvider.future);
    if (!mounted) return;
    if (current == null) {
      router.push(authLoginLocation(returnTo: location));
      return;
    }
    try {
      await ref
          .read(userFollowStateProvider(user.userId).notifier)
          .toggle(userId: user.userId, fallback: user.isFollowing);
      invalidateUserProfilePreview(user.userId);
      await _load();
    } catch (error) {
      if (mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    }
  }

  Future<void> _changeBlock(int userId, bool blocked) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    try {
      await changeUserBlock(context, ref, userId: userId, blocked: blocked);
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    }
  }

  Future<void> _openProfile(int userId) async {
    Navigator.of(context, rootNavigator: true).pop();
    await widget.rootContext.push('/u/$userId');
  }

  Future<void> _openMessage(UserCardPayload user, String avatarUrl) async {
    final location = Uri(
      path: '/chat',
      queryParameters: {
        'userId': '${user.userId}',
        'username': user.username,
        if (avatarUrl.isNotEmpty) 'avatar': avatarUrl,
      },
    ).toString();
    Navigator.of(context, rootNavigator: true).pop();
    await widget.rootContext.push(location);
  }

  void _close() {
    if (_closing) return;
    setState(() => _closing = true);
    Navigator.of(context, rootNavigator: true).pop();
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.listen<int>(offlineCacheEpochProvider, (previous, next) {
      if (previous == next) return;
      invalidateUserProfilePreview(widget.userId);
      setState(() {
        _user = null;
        _error = null;
        _loading = true;
      });
      _load();
    });
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final follow = ref.watch(userFollowStateProvider(widget.userId));
    final viewer = ref.watch(currentUserProvider).valueOrNull;
    final user = _user;
    final userBlocks =
        viewer != null && user != null && viewer.id != user.userId
        ? ref.watch(userBlocksProvider)
        : null;
    final displayName = user == null
        ? privateDisplayName(
            context,
            widget.userId,
            widget.username,
            widget.nickname,
          )
        : privateDisplayName(
            context,
            user.userId,
            user.username,
            user.nickname,
          );
    final avatarUrl = user == null || user.avatarUrl.isEmpty
        ? widget.avatarUrl
        : resolveApiAssetUrl(user.avatarUrl);
    final wornBadge = user?.isAccountClosed == true
        ? null
        : user?.wornBadge ?? widget.wornBadge;
    final cover = user?.profileCoverUrl.trim();
    final coverUrl = cover?.isNotEmpty == true
        ? resolveApiAssetUrl(cover!)
        : null;
    final isLightTheme = Theme.of(context).brightness == Brightness.light;
    final actionSurface = colors.base100.withValues(alpha: .66);
    // Strengthen the hairline with the active foreground token so it survives
    // both ends of the cover-derived glass field.
    final actionBorder = Color.lerp(colors.line, colors.baseContent, .48)!;
    final radii = BorderRadius.circular(24);
    final viewportHeight = MediaQuery.sizeOf(context).height;
    final availableHeight =
        viewportHeight - MediaQuery.paddingOf(context).vertical - 44;
    final maxHeight = math.max(0.0, availableHeight - 16);

    return ConstrainedBox(
      constraints: BoxConstraints(maxWidth: 460, maxHeight: maxHeight),
      child: SizedBox(
        width: double.infinity,
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            Padding(
              padding: const EdgeInsets.only(top: 38),
              child: Material(
                color: colors.base100,
                elevation: isLightTheme ? 2 : 0,
                shape: RoundedRectangleBorder(
                  borderRadius: radii,
                  side: BorderSide(
                    color: colors.line.withValues(
                      alpha: isLightTheme ? 1 : .72,
                    ),
                  ),
                ),
                clipBehavior: Clip.antiAlias,
                child: Stack(
                  children: [
                    if (coverUrl != null) ...[
                      Positioned.fill(
                        child: ImageFiltered(
                          imageFilter: ImageFilter.blur(sigmaX: 34, sigmaY: 42),
                          child: GfNetworkImage(
                            coverUrl,
                            // Stretching only the blurred duplicate lets its
                            // full-width color field flow through the tall card.
                            fit: BoxFit.fill,
                            errorBuilder: (_, _, _) => const SizedBox.shrink(),
                          ),
                        ),
                      ),
                      Positioned.fill(
                        child: ColoredBox(
                          color: colors.base100.withValues(
                            alpha: isLightTheme ? .60 : .52,
                          ),
                        ),
                      ),
                      Positioned(
                        top: 0,
                        left: 0,
                        right: 0,
                        child: AspectRatio(
                          // Covers are uploaded at 5:1; keep this banner sharp
                          // while its lower edge fades into the full-card field.
                          aspectRatio: 5,
                          child: ShaderMask(
                            blendMode: BlendMode.dstIn,
                            shaderCallback: (bounds) => const LinearGradient(
                              begin: Alignment.topCenter,
                              end: Alignment.bottomCenter,
                              colors: [
                                Colors.white,
                                Colors.white,
                                Colors.transparent,
                              ],
                              stops: [0, .58, 1],
                            ).createShader(bounds),
                            child: GfNetworkImage(
                              coverUrl,
                              fit: BoxFit.cover,
                              errorBuilder: (_, _, _) =>
                                  ColoredBox(color: colors.base200),
                            ),
                          ),
                        ),
                      ),
                    ],
                    ConstrainedBox(
                      constraints: BoxConstraints(
                        maxHeight: math.max(0.0, maxHeight - 38),
                      ),
                      child: Scrollbar(
                        controller: _scrollController,
                        thumbVisibility: true,
                        thickness: 3,
                        radius: const Radius.circular(1),
                        interactive: true,
                        child: SingleChildScrollView(
                          controller: _scrollController,
                          child: Padding(
                            padding: const EdgeInsets.only(top: 54),
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              crossAxisAlignment: CrossAxisAlignment.stretch,
                              children: [
                                if (_loading)
                                  Column(
                                    children: [
                                      _identity(displayName, widget.username),
                                      SizedBox(
                                        height: 104,
                                        child: GfLoading(
                                          message: l10n.commonLoading,
                                        ),
                                      ),
                                    ],
                                  )
                                else if (_error != null && user == null)
                                  Column(
                                    children: [
                                      _identity(displayName, widget.username),
                                      SizedBox(
                                        height: 124,
                                        child: GfErrorRetry(
                                          message: resolveErrorMessage(
                                            l10n,
                                            _error!,
                                          ),
                                          onRetry: _load,
                                        ),
                                      ),
                                    ],
                                  )
                                else if (user?.isAccountClosed == true)
                                  _closedAccount(user!, l10n)
                                else if (user != null)
                                  _profileDetails(context, user, l10n)
                                else
                                  const SizedBox(height: 164),
                                if (user != null && !user.isAccountClosed)
                                  Padding(
                                    padding: const EdgeInsets.fromLTRB(
                                      16,
                                      2,
                                      16,
                                      8,
                                    ),
                                    child: LayoutBuilder(
                                      builder: (context, constraints) => GfHorizontalScrollView(
                                        scrollViewKey: const ValueKey(
                                          'profile-preview-actions',
                                        ),
                                        child: ConstrainedBox(
                                          constraints: BoxConstraints(
                                            minWidth: constraints.maxWidth,
                                          ),
                                          child: Row(
                                            mainAxisSize: MainAxisSize.min,
                                            mainAxisAlignment:
                                                MainAxisAlignment.end,
                                            children: [
                                              if (!user.isSelf) ...[
                                                GfFollowButton(
                                                  following:
                                                      follow.following ??
                                                      user.isFollowing,
                                                  label:
                                                      (follow.following ??
                                                          user.isFollowing)
                                                      ? l10n.profileFollowing
                                                      : l10n.profileFollow,
                                                  busy: follow.busy,
                                                  onPressed: () =>
                                                      _toggleFollow(user),
                                                ),
                                                const SizedBox(width: 6),
                                              ],
                                              OutlinedButton.icon(
                                                onPressed: () =>
                                                    _openProfile(user.userId),
                                                icon: const GfSymbol(
                                                  'external-link',
                                                  size: 16,
                                                ),
                                                label: Text(l10n.profileTitle),
                                                style: OutlinedButton.styleFrom(
                                                  minimumSize: const Size(
                                                    44,
                                                    44,
                                                  ),
                                                  foregroundColor:
                                                      colors.baseContent,
                                                  backgroundColor:
                                                      actionSurface,
                                                  side: BorderSide(
                                                    color: actionBorder,
                                                  ),
                                                ),
                                              ),
                                              if (!user.isSelf) ...[
                                                const SizedBox(width: 6),
                                                IconButton.outlined(
                                                  tooltip: l10n.messagesNew,
                                                  constraints:
                                                      const BoxConstraints.tightFor(
                                                        width: 44,
                                                        height: 44,
                                                      ),
                                                  style: IconButton.styleFrom(
                                                    foregroundColor:
                                                        colors.baseContent,
                                                    backgroundColor:
                                                        actionSurface,
                                                    side: BorderSide(
                                                      color: actionBorder,
                                                    ),
                                                    shape: const CircleBorder(),
                                                  ),
                                                  onPressed: () => _openMessage(
                                                    user,
                                                    avatarUrl,
                                                  ),
                                                  icon: const GfSymbol(
                                                    'mail',
                                                    size: 20,
                                                  ),
                                                ),
                                              ],
                                              if (viewer != null &&
                                                  viewer.id != user.userId) ...[
                                                const SizedBox(width: 6),
                                                GfActionMenuButton<String>(
                                                  tooltip: l10n.profileMore,

                                                  icon: const GfSymbol(
                                                    'ellipsis',
                                                    size: 20,
                                                  ),
                                                  style: ButtonStyle(
                                                    minimumSize:
                                                        const WidgetStatePropertyAll(
                                                          Size(44, 44),
                                                        ),
                                                    maximumSize:
                                                        const WidgetStatePropertyAll(
                                                          Size(44, 44),
                                                        ),
                                                    padding:
                                                        const WidgetStatePropertyAll(
                                                          EdgeInsets.zero,
                                                        ),
                                                    tapTargetSize:
                                                        MaterialTapTargetSize
                                                            .shrinkWrap,
                                                    foregroundColor:
                                                        WidgetStatePropertyAll(
                                                          colors.baseContent,
                                                        ),
                                                    backgroundColor:
                                                        WidgetStatePropertyAll(
                                                          actionSurface,
                                                        ),
                                                    side:
                                                        WidgetStatePropertyAll(
                                                          BorderSide(
                                                            color: actionBorder,
                                                          ),
                                                        ),
                                                    shape:
                                                        const WidgetStatePropertyAll(
                                                          CircleBorder(),
                                                        ),
                                                  ),
                                                  itemBuilder: (_) {
                                                    if (userBlocks?.hasError ==
                                                        true) {
                                                      return [
                                                        GfContextAction(
                                                          value: 'retry',
                                                          label:
                                                              l10n.commonRetry,
                                                          symbol: 'refresh-cw',
                                                        ),
                                                      ];
                                                    }
                                                    final blockPayload =
                                                        userBlocks?.valueOrNull;
                                                    if (blockPayload == null) {
                                                      return [
                                                        GfContextAction(
                                                          value: 'loading',
                                                          enabled: false,
                                                          label: l10n
                                                              .commonLoading,
                                                        ),
                                                      ];
                                                    }
                                                    final isBlocked =
                                                        blockPayload.blocks.any(
                                                          (item) =>
                                                              item.targetUserId ==
                                                              user.userId,
                                                        );
                                                    return [
                                                      GfContextAction(
                                                        value: isBlocked
                                                            ? 'unblock'
                                                            : 'block',
                                                        label: isBlocked
                                                            ? l10n.userUnblock
                                                            : l10n.userBlock,
                                                        symbol: 'user-round',
                                                      ),
                                                    ];
                                                  },
                                                  onSelected: (action) {
                                                    if (action == 'retry') {
                                                      ref.invalidate(
                                                        userBlocksProvider,
                                                      );
                                                      return;
                                                    }
                                                    _changeBlock(
                                                      user.userId,
                                                      action == 'block',
                                                    );
                                                  },
                                                ),
                                              ],
                                            ],
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
                    Positioned(
                      top: 8,
                      right: 8,
                      child: GfGlassSurface(
                        blurSigma: 4,
                        child: IconButton(
                          constraints: const BoxConstraints.tightFor(
                            width: 44,
                            height: 44,
                          ),
                          padding: EdgeInsets.zero,
                          tooltip: l10n.commonClose,
                          onPressed: _closing ? null : _close,
                          icon: AnimatedRotation(
                            turns: _closing ? .125 : 0,
                            duration: GfMotion.duration(
                              context,
                              GfMotion.press,
                            ),
                            curve: GfMotion.enterCurve,
                            child: AnimatedScale(
                              scale: _closing ? .84 : 1,
                              duration: GfMotion.duration(
                                context,
                                GfMotion.press,
                              ),
                              curve: GfMotion.enterCurve,
                              child: const GfSymbol(
                                'x',
                                size: 18,
                                color: Colors.white,
                              ),
                            ),
                          ),
                        ),
                      ),
                    ),
                    if (_refreshing && user != null)
                      Positioned(
                        top: 12,
                        left: 84,
                        child: GfLoadingIndicator(small: true),
                      ),
                  ],
                ),
              ),
            ),
            Positioned(
              left: 20,
              top: 0,
              child: DecoratedBox(
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: colors.base100,
                  boxShadow: GfTheme.shadowsOf(context).floating,
                ),
                child: Padding(
                  padding: const EdgeInsets.all(4),
                  child: GfAvatar(
                    src: avatarUrl,
                    size: 76,
                    badge: wornBadge == null
                        ? null
                        : UserWornBadge(wornBadge, avatarSize: 76),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _identity(String name, String username) => GfUserCard(
    showHeader: false,
    compact: true,
    avatarUrl: '',
    name: name,
    username: username,
  );

  Widget _closedAccount(UserCardPayload user, AppLocalizations l10n) {
    final context = this.context;
    final colors = GfTheme.colorsOf(context);
    return Column(
      children: [
        GfUserCard(
          showHeader: false,
          compact: true,
          avatarUrl: '',
          name: privateDisplayName(
            context,
            user.userId,
            user.username,
            user.nickname,
          ),
          username: user.username,
          nameBadges: [
            GfBadge(
              label: l10n.profileAccountClosedBadge,
              variant: GfBadgeVariant.muted,
              radius: 4,
            ),
          ],
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 0, 24, 16),
          child: Column(
            children: [
              GfSymbol('user-round', size: 34, color: colors.iconMuted),
              const SizedBox(height: 8),
              Text(
                l10n.profileAccountClosedTitle,
                style: Theme.of(context).textTheme.titleSmall?.copyWith(
                  color: colors.baseContent,
                  fontWeight: FontWeight.w700,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 6),
              Text(
                l10n.profileAccountClosedDescription,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: colors.iconMuted,
                  height: 1.5,
                ),
                textAlign: TextAlign.center,
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _profileDetails(
    BuildContext context,
    UserCardPayload user,
    AppLocalizations l10n,
  ) {
    final colors = GfTheme.colorsOf(context);
    final isLightTheme = Theme.of(context).brightness == Brightness.light;
    final notes = PrivateNotesScope.of(context);
    final links = publicProfileLinks(user).take(8).toList();
    final joinedAt = user.createdAt.trim().isEmpty
        ? null
        : l10n.profileJoinedAt(formatDate(user.createdAt));
    final lastActive = !user.isOnline && user.lastActiveTime.trim().isNotEmpty
        ? l10n.profileLastActive(timeAgo(user.lastActiveTime, l10n: l10n))
        : null;
    final badges = (user.displayBadges ?? user.badges)
        .take(5)
        .map((badge) {
          return GfUserBadge(
            label: badge.name,
            color: userBadgeColor(badge),
            icon: UserBadgeArtwork(badge, size: 24),
            description: badge.description,
            onTap: () => showUserBadgeDetails(context, badge),
          );
        })
        .toList(growable: false);
    Widget statusBadge(GfBadge badge, {required double radius}) => isLightTheme
        ? DecoratedBox(
            decoration: BoxDecoration(
              color: colors.base100.withValues(alpha: .72),
              borderRadius: BorderRadius.circular(radius),
              border: Border.all(color: colors.line.withValues(alpha: .72)),
            ),
            child: badge,
          )
        : badge;
    return GfUserCard(
      showHeader: false,
      compact: true,
      avatarUrl: '',
      name: privateDisplayName(
        context,
        user.userId,
        user.username,
        user.nickname,
      ),
      username: user.username,
      usernameAction:
          notes != null && notes.ownerId > 0 && notes.ownerId != user.userId
          ? PrivateNoteButton(
              userId: user.userId,
              username: user.username,
              compact: true,
            )
          : null,
      nameBadges: [
        if (user.isAdmin)
          statusBadge(
            GfBadge(
              label: l10n.profileRoleAdmin,
              variant: GfBadgeVariant.warning,
              color: isLightTheme
                  ? Color.lerp(colors.warning, colors.baseContent, .6)
                  : null,
              radius: 4,
            ),
            radius: 4,
          ),
        if (user.isOnline)
          statusBadge(
            GfBadge(
              label: l10n.profileOnline,
              variant: GfBadgeVariant.success,
              color: isLightTheme
                  ? Color.lerp(colors.success, colors.baseContent, .4)
                  : null,
              icon: const GfSymbol('signal-stream', size: 12),
            ),
            radius: 999,
          ),
      ],
      bio: user.bio,
      signature: user.signature,
      coloredBadges: badges,
      stats: [
        (l10n.profilePosts, formatNumber(user.topicCount)),
        (l10n.profileReplies, formatNumber(user.replyCount)),
        (l10n.profileLikes, formatNumber(user.likeReceivedCount)),
        (l10n.profileFollowers, formatNumber(user.followerCount)),
      ],
      details: links.isEmpty && joinedAt == null && lastActive == null
          ? null
          : Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (lastActive != null || joinedAt != null)
                    GfHorizontalScrollView(
                      scrollViewKey: const ValueKey('profile-preview-date-row'),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          if (lastActive != null)
                            _metadata(context, 'clock', lastActive),
                          if (lastActive != null && joinedAt != null)
                            const SizedBox(width: 8),
                          if (joinedAt != null)
                            _metadata(context, 'calendar-days', joinedAt),
                        ],
                      ),
                    ),
                  if (links.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 2),
                      child: GfHorizontalScrollView(
                        scrollViewKey: const ValueKey(
                          'profile-preview-links-row',
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            for (final link in links)
                              IconButton(
                                constraints: const BoxConstraints.tightFor(
                                  width: 44,
                                  height: 44,
                                ),
                                padding: EdgeInsets.zero,
                                visualDensity: VisualDensity.compact,
                                tooltip: link.$1,
                                onPressed: () => _openLink(link.$2),
                                icon: GfSocialIcon(link.$3, size: 18),
                              ),
                          ],
                        ),
                      ),
                    ),
                ],
              ),
            ),
    );
  }

  Widget _metadata(BuildContext context, String icon, String label) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 2),
    child: Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        GfSymbol(icon, size: 14, color: GfTheme.colorsOf(context).iconMuted),
        const SizedBox(width: 5),
        Text(
          label,
          style: GfTheme.typographyOf(
            context,
          ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
        ),
      ],
    ),
  );

  Future<void> _openLink(Uri uri) async {
    final l10n = AppLocalizations.of(context);
    try {
      if (!await launchUrl(uri, mode: LaunchMode.externalApplication)) {
        throw StateError('Could not open profile link');
      }
    } catch (error) {
      if (mounted) {
        showGfToast(context, resolveErrorMessage(l10n, error), error: true);
      }
    }
  }
}
