import 'package:core/core.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../current_user.dart';
import '../format.dart';
import '../navigation/tab_swipe_surface.dart';
import '../providers.dart';
import '../theme_mode.dart';

final accountDrawerLayerKey = GlobalKey<AccountDrawerLayerState>();

class AccountDrawerLayer extends StatefulWidget {
  const AccountDrawerLayer({
    super.key,
    required this.child,
    required this.onChanged,
  });

  final Widget child;
  final ValueChanged<bool> onChanged;

  @override
  State<AccountDrawerLayer> createState() => AccountDrawerLayerState();
}

class AccountDrawerLayerState extends State<AccountDrawerLayer>
    with SingleTickerProviderStateMixin {
  late final AnimationController _progress = AnimationController(
    vsync: this,
    duration: GfMotion.overlay,
  )..addListener(_onProgress);
  final FocusScopeNode _focusScopeNode = FocusScopeNode();
  final DrawerGesturePolicy _drawerGesturePolicy = DrawerGesturePolicy();
  LocalHistoryEntry? _historyEntry;
  double _drawerWidth = 0;
  double _openingWidth = 0;
  int _direction = 1;
  bool _reportedOpen = false;
  bool _disposing = false;
  bool _dragActive = false;

  void _onProgress() {
    final open = _progress.value >= .5;
    if (open != _reportedOpen) {
      _reportedOpen = open;
      widget.onChanged(open);
    }
    setState(() {});
  }

  void open() {
    _ensureHistoryEntry();
    _progress.animateTo(
      1,
      duration: GfMotion.duration(context, GfMotion.overlay),
      curve: GfMotion.enterCurve,
    );
  }

  void close() {
    final entry = _historyEntry;
    if (entry == null) {
      _animateClosed();
      return;
    }
    _historyEntry = null;
    entry.remove();
  }

  void _animateClosed() {
    _progress.animateTo(
      0,
      duration: GfMotion.duration(context, GfMotion.overlay),
      curve: GfMotion.layoutCurve,
    );
  }

  void _ensureHistoryEntry() {
    if (_historyEntry != null) return;
    final route = ModalRoute.of(context);
    if (route == null) return;
    late final LocalHistoryEntry entry;
    entry = LocalHistoryEntry(
      onRemove: () {
        if (identical(_historyEntry, entry)) _historyEntry = null;
        if (!_disposing && mounted && _progress.value > 0) _animateClosed();
      },
      impliesAppBarDismissal: false,
    );
    _historyEntry = entry;
    route.addLocalHistoryEntry(entry);
    FocusScope.of(context).setFirstFocus(_focusScopeNode);
  }

  void _startDrag(DragStartDetails _) {
    _progress.stop();
    _dragActive = true;
    if (_progress.value == 0) _ensureHistoryEntry();
  }

  void _updateDrag(DragUpdateDetails details) {
    if (_drawerWidth == 0) return;
    _progress.value =
        (_progress.value +
                (details.primaryDelta ?? 0) * _direction / _drawerWidth)
            .clamp(0, 1);
  }

  void _endDrag(DragEndDetails details) {
    _dragActive = false;
    final velocity = details.velocity.pixelsPerSecond.dx * _direction;
    if (velocity.abs() >= 365) {
      if (velocity > 0) {
        open();
      } else {
        close();
      }
    } else if (_progress.value >= .5) {
      open();
    } else {
      close();
    }
  }

  void _cancelDrag() {
    if (!_dragActive) return;
    _dragActive = false;
    if (_progress.value >= .5) {
      open();
    } else {
      close();
    }
  }

  @override
  void dispose() {
    _disposing = true;
    final entry = _historyEntry;
    _historyEntry = null;
    entry?.remove();
    _progress
      ..removeListener(_onProgress)
      ..dispose();
    _focusScopeNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      _drawerWidth = (constraints.maxWidth * .84).clamp(0, 400);
      _openingWidth = constraints.maxWidth * drawerSwipeOpeningFraction;
      _direction = Directionality.of(context) == TextDirection.ltr ? 1 : -1;
      return RawGestureDetector(
        behavior: HitTestBehavior.translucent,
        excludeFromSemantics: true,
        gestures: {
          _DirectionalDrawerGestureRecognizer:
              GestureRecognizerFactoryWithHandlers<
                _DirectionalDrawerGestureRecognizer
              >(
                _DirectionalDrawerGestureRecognizer.new,
                (recognizer) => recognizer
                  ..screenWidth = constraints.maxWidth
                  ..drawerWidth = _drawerWidth
                  ..openingWidth = _openingWidth
                  ..openingDirection = _direction
                  ..isOpen = (() => _progress.value > 0)
                  ..canOpen = _drawerGesturePolicy.canOpen
                  ..dragStartBehavior = DragStartBehavior.down
                  ..onStart = _startDrag
                  ..onUpdate = _updateDrag
                  ..onEnd = _endDrag
                  ..onCancel = _cancelDrag,
              ),
        },
        child: DrawerGestureGate(
          policy: _drawerGesturePolicy,
          child: Stack(
            fit: StackFit.expand,
            clipBehavior: Clip.hardEdge,
            children: [
              IgnorePointer(
                ignoring: _progress.value > 0,
                child: ExcludeSemantics(
                  excluding: _progress.value > 0,
                  child: widget.child,
                ),
              ),
              if (_progress.value > 0) ...[
                Positioned.fill(
                  child: BlockSemantics(
                    child: GestureDetector(
                      behavior: HitTestBehavior.opaque,
                      onTap: close,
                      child: Semantics(
                        label: MaterialLocalizations.of(
                          context,
                        ).modalBarrierDismissLabel,
                        child: ColoredBox(
                          color: Colors.black.withValues(
                            alpha: .54 * _progress.value,
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
                PositionedDirectional(
                  top: 0,
                  bottom: 0,
                  start: -_drawerWidth * (1 - _progress.value),
                  width: _drawerWidth,
                  child: FocusScope(
                    node: _focusScopeNode,
                    child: const AccountDrawer(),
                  ),
                ),
              ],
            ],
          ),
        ),
      );
    },
  );
}

class _DirectionalDrawerGestureRecognizer
    extends HorizontalDragGestureRecognizer {
  _DirectionalDrawerGestureRecognizer()
    : super(supportedDevices: const {PointerDeviceKind.touch}) {
    onlyAcceptDragOnThreshold = true;
  }

  double screenWidth = 0;
  double drawerWidth = 0;
  double openingWidth = 0;
  int openingDirection = 1;
  bool Function() isOpen = () => false;
  bool Function(int) canOpen = (_) => true;
  final Map<int, Offset> _origins = {};
  final Map<int, int> _directions = {};
  final Set<int> _acceptedPointers = {};

  @override
  bool isPointerAllowed(PointerEvent event) {
    final x = event.localPosition.dx;
    final open = isOpen();
    final bandWidth = open ? drawerWidth : openingWidth;
    final inLeadingBand = openingDirection > 0
        ? x >= 0 && x <= bandWidth
        : x <= screenWidth && x >= screenWidth - bandWidth;
    return inLeadingBand && super.isPointerAllowed(event);
  }

  @override
  void addAllowedPointer(PointerDownEvent event) {
    _origins[event.pointer] = event.position;
    _directions[event.pointer] = isOpen()
        ? -openingDirection
        : openingDirection;
    super.addAllowedPointer(event);
  }

  @override
  void handleEvent(PointerEvent event) {
    final origin = _origins[event.pointer];
    // Arena rejection cannot undo an accepted drag. Keep all subsequent moves
    // flowing, including reversals past the origin and diagonal adjustments.
    if (origin != null &&
        event is PointerMoveEvent &&
        !_acceptedPointers.contains(event.pointer)) {
      if (!isOpen() && !canOpen(event.pointer)) {
        resolve(GestureDisposition.rejected);
        return;
      }
      final delta = event.position - origin;
      final slop = computeHitSlop(event.kind, gestureSettings);
      if (delta.dx * _directions[event.pointer]! < -slop ||
          (delta.dy.abs() >= slop && delta.dy.abs() > delta.dx.abs())) {
        resolve(GestureDisposition.rejected);
        return;
      }
    }
    if (event is PointerUpEvent || event is PointerCancelEvent) {
      _origins.remove(event.pointer);
      _directions.remove(event.pointer);
      _acceptedPointers.remove(event.pointer);
    }
    super.handleEvent(event);
  }

  @override
  void acceptGesture(int pointer) {
    _acceptedPointers.add(pointer);
    super.acceptGesture(pointer);
  }

  @override
  void rejectGesture(int pointer) {
    _origins.remove(pointer);
    _directions.remove(pointer);
    _acceptedPointers.remove(pointer);
    super.rejectGesture(pointer);
  }

  @override
  void dispose() {
    _origins.clear();
    _directions.clear();
    _acceptedPointers.clear();
    super.dispose();
  }
}

final accountLayoutProvider = FutureProvider.autoDispose<LayoutPayload>((
  ref,
) async {
  ref.watch(currentUserProvider);
  return (await ref.watch(pageRepositoryProvider).home()).layout;
});

/// Marks the decorative hairline separating the account group (sign-in or the
/// signed-in entries) from the settings, about and appearance entries below.
const Key drawerSectionDividerKey = ValueKey('drawer-section-divider');

// Keyed by the server viewer ID; a session change discards cached identity.
final accountCardProvider = FutureProvider.autoDispose
    .family<UserCardPayload, int>((ref, id) {
      ref.watch(currentUserProvider);
      return ref.watch(userRepositoryProvider).getUserCard(id);
    });

class AccountAvatar extends ConsumerWidget {
  const AccountAvatar({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final layout = ref.watch(accountLayoutProvider);
    final viewer = layout.isLoading ? null : layout.asData?.value.viewer;
    return GfAvatar(src: resolveApiAssetUrl(viewer?.avatarUrl ?? ''), size: 30);
  }
}

class AccountDrawer extends ConsumerWidget {
  const AccountDrawer({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final layout = ref.watch(accountLayoutProvider);
    final viewer = layout.isLoading ? null : layout.asData?.value.viewer;
    final signedIn = viewer?.isAuthenticated == true;
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final card = signedIn ? ref.watch(accountCardProvider(viewer!.id)) : null;
    final user = card?.isLoading == true ? null : card?.asData?.value;
    void open(String path) {
      Navigator.of(context).pop();
      context.push(path);
    }

    Widget connection(String label, int? count, String stream) => TextButton(
      style: TextButton.styleFrom(
        foregroundColor: colors.baseContent,
        padding: EdgeInsets.zero,
        minimumSize: const Size(0, 44),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      ),
      onPressed: () => open('/profile?stream=$stream'),
      child: Text.rich(
        TextSpan(
          children: [
            TextSpan(
              text: count == null ? '—' : formatNumber(count),
              style: const TextStyle(fontWeight: FontWeight.w700),
            ),
            TextSpan(
              text: ' $label',
              style: TextStyle(color: colors.iconMuted),
            ),
          ],
        ),
        style: type.small.copyWith(fontSize: 15),
      ),
    );
    Widget entry(
      String icon,
      String title,
      String? path, {
      Widget? trailing,
      VoidCallback? onTap,
    }) => ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 24),
      minTileHeight: 56,
      minLeadingWidth: 24,
      horizontalTitleGap: 16,
      leading: GfSymbol(icon, size: 24, color: colors.baseContent),
      title: Text(
        title,
        style: type.bodyStrong.copyWith(fontSize: 18, height: 1.3),
      ),
      trailing: trailing,
      onTap: onTap ?? (path == null ? null : () => open(path)),
    );
    return Drawer(
      width: (MediaQuery.sizeOf(context).width * .84).clamp(0.0, 400.0),
      backgroundColor: GfTheme.colorsOf(context).base100,
      shape: const RoundedRectangleBorder(),
      child: SafeArea(
        child: ListView(
          padding: const EdgeInsets.symmetric(vertical: 12),
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 12, 24, 12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  InkWell(
                    borderRadius: BorderRadius.circular(30),
                    onTap: () => open(signedIn ? '/profile' : '/login'),
                    child: GfAvatar(
                      src: resolveApiAssetUrl(
                        user?.avatarUrl ?? viewer?.avatarUrl ?? '',
                      ),
                      size: 56,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    signedIn
                        ? (user?.nickname.isNotEmpty == true
                              ? user!.nickname
                              : viewer!.username)
                        : 'YourTJ',
                    style: type.title2.copyWith(fontSize: 20),
                  ),
                  if (signedIn)
                    Text(
                      '@${viewer!.username}',
                      style: type.small.copyWith(color: colors.iconMuted),
                    ),
                  if (signedIn) ...[
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 18,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        connection(
                          l10n.accountFollowing,
                          user?.followingCount,
                          'following',
                        ),
                        connection(
                          l10n.accountFollowers,
                          user?.followerCount,
                          'followers',
                        ),
                        if (card?.hasError == true)
                          IconButton(
                            tooltip: l10n.commonRetry,
                            onPressed: () =>
                                ref.invalidate(accountCardProvider(viewer!.id)),
                            icon: const GfSymbol('refresh-cw', size: 18),
                          ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
            if (!signedIn) entry('user-round', l10n.loginModeLogin, '/login'),
            if (signedIn) ...[
              entry('user-round', l10n.settingsSectionProfile, '/profile'),
              entry(
                'bookmark',
                l10n.profileBookmarks,
                '/profile?stream=bookmarks',
              ),
              entry('file-text', l10n.draftsTitle, '/drafts'),
              entry('archive', l10n.accountContent, '/my-content'),
              entry('trash-2', l10n.profileTrash, '/recycle-bin'),
              entry(
                'graduation-cap',
                l10n.myCourseReviewsTitle,
                '/my-course-reviews',
              ),
              if (viewer!.canAccessAdmin)
                entry('shield-check', l10n.profileAdmin, '/admin'),
              if (viewer.isModerator || viewer.canAccessAdmin)
                entry('shield-check', l10n.profileModeration, '/moderation'),
              if (viewer.canManageCourses) ...[
                entry(
                  'graduation-cap',
                  l10n.coursesManagement,
                  '/moderation/courses',
                ),
                entry(
                  'shield-check',
                  l10n.coursesReviewModeration,
                  '/moderation/course-reviews',
                ),
              ],
            ],
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 8),
              child: GfDivider(key: drawerSectionDividerKey, inset: 24),
            ),
            entry('settings', l10n.settingsTitle, '/settings'),
            entry('book-open', l10n.siteInfoTitle, '/about'),
            entry(
              'moon',
              l10n.settingsAppearance,
              null,
              trailing: GfSymbol(
                'chevron-down',
                size: 18,
                color: colors.iconMuted,
              ),
              onTap: () => _showThemeModeSheet(context),
            ),
          ],
        ),
      ),
    );
  }
}

Future<void> _showThemeModeSheet(BuildContext context) =>
    showGfBottomSheet<void>(
      context,
      builder: (_) => Consumer(
        builder: (context, ref, _) {
          final l10n = AppLocalizations.of(context);
          final mode = ref.watch(themeModeProvider);
          final type = GfTheme.typographyOf(context);
          const choices = <ThemeMode>[
            ThemeMode.light,
            ThemeMode.dark,
            ThemeMode.system,
          ];
          return SingleChildScrollView(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 20, 16, 12),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(l10n.settingsAppearance, style: type.title2),
                  const SizedBox(height: 12),
                  RadioGroup<ThemeMode>(
                    groupValue: mode,
                    onChanged: (value) {
                      if (value != null) {
                        ref.read(themeModeProvider.notifier).setMode(value);
                      }
                    },
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        for (final choice in choices)
                          RadioListTile<ThemeMode>(
                            value: choice,
                            controlAffinity: ListTileControlAffinity.trailing,
                            contentPadding: EdgeInsets.zero,
                            title: Text(switch (choice) {
                              ThemeMode.system => l10n.settingsLanguageSystem,
                              ThemeMode.light => l10n.settingsThemeLight,
                              ThemeMode.dark => l10n.settingsThemeDark,
                            }, style: type.body),
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
