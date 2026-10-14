import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../current_user.dart';
import '../../navigation/auth_navigation.dart';
import '../../providers.dart';
import '../../widgets/skeletons.dart';
import '../../widgets/status_views.dart';

/// The owner's saved profile privacy (issue #1124), reloaded per login epoch.
final profilePrivacySettingsProvider =
    FutureProvider.autoDispose<UserPrivacySettingsPayload?>((ref) async {
      ref.watch(offlineCacheEpochProvider);
      if (await ref.watch(currentUserProvider.future) == null) return null;
      return ref.read(userRepositoryProvider).getPrivacySettings();
    });

const List<String> _audienceRank = [
  'everyone',
  'members',
  'followers',
  'mutual',
  'self',
];

/// True when [to] lets more people see a section than [from].
bool profilePrivacyWidens(String from, String to) =>
    _audienceRank.indexOf(to) < _audienceRank.indexOf(from);

String profilePrivacySectionLabel(AppLocalizations l10n, String key) =>
    switch (key) {
      'profileTopics' => l10n.profilePrivacyTopics,
      'profileActivity' => l10n.profilePrivacyActivity,
      'profileLikes' => l10n.profilePrivacyLikes,
      'followingList' => l10n.profilePrivacyFollowing,
      _ => l10n.profilePrivacyFollowers,
    };

String _sectionDescription(AppLocalizations l10n, String key) => switch (key) {
  'profileTopics' => l10n.profilePrivacyTopicsDescription,
  'profileActivity' => l10n.profilePrivacyActivityDescription,
  'profileLikes' => l10n.profilePrivacyLikesDescription,
  'followingList' => l10n.profilePrivacyFollowingDescription,
  _ => l10n.profilePrivacyFollowersDescription,
};

String profilePrivacyAudienceLabel(AppLocalizations l10n, String audience) =>
    switch (audience) {
      'members' => l10n.profilePrivacyMembers,
      'followers' => l10n.profilePrivacyFollowersOnly,
      'mutual' => l10n.profilePrivacyMutual,
      'self' => l10n.profilePrivacySelf,
      _ => l10n.profilePrivacyEveryone,
    };

/// Choose who can see each profile section. A row shows the saved value
/// until the server confirms a change, so a failure needs no rollback.
class ProfilePrivacyPage extends ConsumerStatefulWidget {
  const ProfilePrivacyPage({super.key});

  @override
  ConsumerState<ProfilePrivacyPage> createState() => _ProfilePrivacyPageState();
}

class _ProfilePrivacyPageState extends ConsumerState<ProfilePrivacyPage> {
  UserPrivacySettingsPayload? _confirmed;
  String? _saving;

  Future<void> _choose(UserPrivacySettingsPayload settings, String key) async {
    if (_saving != null) return;
    final l10n = AppLocalizations.of(context);
    final current = settings.audienceOf(key);
    final epoch = ref.read(offlineCacheEpochProvider);
    final picked = await showGfBottomSheet<String>(
      context,
      showDragHandle: true,
      builder: (_) => _AudienceSheet(section: key, selected: current),
    );
    if (!mounted ||
        picked == null ||
        picked == current ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    // Restoring a wider audience re-exposes history, so confirm it first.
    if (profilePrivacyWidens(current, picked) &&
        !await _confirmWiden(l10n, key, picked)) {
      return;
    }
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
    setState(() => _saving = key);
    try {
      final saved = await ref
          .read(userRepositoryProvider)
          .updatePrivacySetting(key, picked);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() => _confirmed = saved);
      showGfToast(context, l10n.profilePrivacySaved);
    } catch (_) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(context, l10n.profilePrivacySaveFailed, error: true);
      }
    } finally {
      if (mounted) setState(() => _saving = null);
    }
  }

  Future<bool> _confirmWiden(
    AppLocalizations l10n,
    String key,
    String audience,
  ) async =>
      await showGfAlertDialog<bool>(
        context,
        barrierDismissible: true,
        builder: (ctx) => GfAlertDialog(
          key: const ValueKey('profile-privacy-widen'),
          title: Text(l10n.profilePrivacyWidenTitle),
          content: Text(
            l10n.profilePrivacyWidenDescription(
              profilePrivacySectionLabel(l10n, key),
              profilePrivacyAudienceLabel(l10n, audience),
            ),
          ),
          actions: [
            GfButton(
              label: l10n.commonCancel,
              variant: GfButtonVariant.ghost,
              onPressed: () => Navigator.pop(ctx, false),
            ),
            GfButton(
              label: l10n.profilePrivacyWidenConfirm,
              onPressed: () => Navigator.pop(ctx, true),
            ),
          ],
        ),
      ) ==
      true;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    // A different account never inherits the previous one's saved values.
    ref.listen(offlineCacheEpochProvider, (_, _) {
      setState(() {
        _confirmed = null;
        _saving = null;
      });
    });
    final state = ref.watch(profilePrivacySettingsProvider);
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.profilePrivacyTitle)),
      body: SafeArea(
        top: false,
        child: state.when(
          loading: () => const GfSettingsSkeleton(),
          error: (error, _) => GfErrorRetry(
            message: l10n.profilePrivacyLoadFailed,
            onRetry: () => ref.invalidate(profilePrivacySettingsProvider),
          ),
          data: (loaded) => loaded == null
              ? GfEmpty(
                  symbol: 'login4',
                  message: l10n.profileNotLoggedIn,
                  action: GfButton(
                    label: l10n.authLoginTitle,
                    onPressed: () => context.push(
                      authLoginLocation(returnTo: '/settings/profile-privacy'),
                    ),
                  ),
                )
              : _content(context, l10n, _confirmed ?? loaded),
        ),
      ),
    );
  }

  Widget _content(
    BuildContext context,
    AppLocalizations l10n,
    UserPrivacySettingsPayload settings,
  ) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final muted = type.caption.copyWith(
      color: colors.baseContent.withValues(alpha: .65),
    );
    final keys = profilePrivacyAudiences.keys.toList();
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Padding(
          padding: const EdgeInsetsDirectional.only(start: 4, bottom: 12),
          child: Text(l10n.profilePrivacyDescription, style: muted),
        ),
        Material(
          color: colors.base100,
          borderRadius: BorderRadius.circular(16),
          clipBehavior: Clip.antiAlias,
          child: Column(
            children: [
              for (final key in keys) ...[
                if (key != keys.first) const GfDivider(),
                Semantics(
                  value: profilePrivacyAudienceLabel(
                    l10n,
                    settings.audienceOf(key),
                  ),
                  child: GfSettingRow(
                    key: ValueKey('profile-privacy-row-$key'),
                    title: profilePrivacySectionLabel(l10n, key),
                    description: _sectionDescription(l10n, key),
                    trailing: _saving == key
                        ? const SizedBox.square(
                            dimension: 18,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              ConstrainedBox(
                                constraints: const BoxConstraints(
                                  maxWidth: 120,
                                ),
                                child: Text(
                                  profilePrivacyAudienceLabel(
                                    l10n,
                                    settings.audienceOf(key),
                                  ),
                                  textAlign: TextAlign.end,
                                  maxLines: 2,
                                  overflow: TextOverflow.ellipsis,
                                  style: type.small.copyWith(
                                    color: colors.baseContent.withValues(
                                      alpha: .7,
                                    ),
                                  ),
                                ),
                              ),
                              const SizedBox(width: 4),
                              GfSymbol(
                                'chevron-right',
                                size: 18,
                                color: colors.iconMuted,
                              ),
                            ],
                          ),
                    onTap: _saving == null
                        ? () => _choose(settings, key)
                        : null,
                  ),
                ),
              ],
              // Bookmarks have no audience to choose: always only the owner.
              // A fixed row keeps that from reading as public by omission.
              const GfDivider(),
              Semantics(
                value: l10n.profilePrivacySelf,
                child: GfSettingRow(
                  key: const ValueKey('profile-privacy-row-bookmarks'),
                  title: l10n.profileBookmarks,
                  description: l10n.profilePrivacyBookmarksDescription,
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        l10n.profilePrivacySelf,
                        style: type.small.copyWith(
                          color: colors.baseContent.withValues(alpha: .7),
                        ),
                      ),
                      const SizedBox(width: 4),
                      // The chevron's slot, so the label lines up with the
                      // editable rows above.
                      SizedBox.square(
                        dimension: 18,
                        child: GfSymbol(
                          'lock-filled',
                          size: 14,
                          color: colors.iconMuted,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        Padding(
          padding: const EdgeInsetsDirectional.symmetric(horizontal: 4),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Padding(
                padding: const EdgeInsets.only(top: 1),
                child: GfSymbol('info', size: 16, color: colors.iconMuted),
              ),
              const SizedBox(width: 8),
              Expanded(child: Text(l10n.profilePrivacyScope, style: muted)),
            ],
          ),
        ),
        const SizedBox(height: 16),
        Align(
          alignment: AlignmentDirectional.centerStart,
          child: TextButton(
            key: const ValueKey('profile-privacy-view-profile'),
            onPressed: () => context.go('/profile'),
            child: Text(l10n.profilePrivacyViewProfile),
          ),
        ),
      ],
    );
  }
}

class _AudienceSheet extends StatelessWidget {
  const _AudienceSheet({required this.section, required this.selected});
  final String section;
  final String selected;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return SafeArea(
      top: false,
      child: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(8, 0, 8, 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 4, 12, 8),
              child: Text(
                l10n.profilePrivacyAudienceTitle(
                  profilePrivacySectionLabel(l10n, section),
                ),
                style: type.title3,
              ),
            ),
            for (final audience in profilePrivacyAudiences[section]!)
              Semantics(
                selected: audience == selected,
                child: GfSettingRow(
                  key: ValueKey('profile-privacy-option-$audience'),
                  title: profilePrivacyAudienceLabel(l10n, audience),
                  trailing: audience == selected
                      ? GfSymbol('check', size: 18, color: colors.primary)
                      : const SizedBox.shrink(),
                  onTap: () => Navigator.of(context).pop(audience),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
