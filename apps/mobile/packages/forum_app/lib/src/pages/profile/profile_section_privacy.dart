import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../current_user.dart';
import '../../navigation/auth_navigation.dart';

/// The privacy state of one profile stream (issue #1124); `null` for streams
/// that profile privacy does not cover.
ProfileSectionVisibilityPayload? profileStreamVisibility(
  UserProfileProps props,
  String stream,
) => switch (stream) {
  'timeline' => props.visibility.profileActivity,
  'topics' => props.visibility.profileTopics,
  'likes' => props.visibility.profileLikes,
  'following' => props.visibility.followingList,
  'followers' => props.visibility.followersList,
  _ => null,
};

/// Shown to other members and guests instead of a hidden section's list.
class ProfileSectionHidden extends ConsumerWidget {
  const ProfileSectionHidden({
    super.key,
    required this.state,
    required this.name,
    required this.returnTo,
  });

  final ProfileSectionVisibilityPayload state;

  /// The profile owner's display name.
  final String name;
  final String returnTo;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final signedIn = ref.watch(currentUserProvider).valueOrNull != null;
    final reason = switch (state.reason) {
      'signin' => l10n.profileHiddenSignin(name),
      'followers' =>
        signedIn
            ? l10n.profileHiddenFollowers(name)
            : l10n.profileHiddenFollowersGuest(name),
      'mutual' => l10n.profileHiddenMutual(name),
      _ => l10n.profileHiddenPrivate(name),
    };
    // Only offer sign-in when signing in could actually reveal the section.
    final canSignIn =
        !signedIn && (state.reason == 'signin' || state.reason == 'followers');
    return GfEmpty(
      key: const ValueKey('profile-section-hidden'),
      symbol: 'eye-off',
      message: l10n.profileHiddenTitle,
      description: reason,
      action: canSignIn
          ? GfButton(
              key: const ValueKey('profile-section-sign-in'),
              label: l10n.profileHiddenSignIn,
              onPressed: () =>
                  context.push(authLoginLocation(returnTo: returnTo)),
            )
          : null,
    );
  }
}

/// Reminds the owner who else can see a restricted section.
class ProfileSectionOwnerNote extends StatelessWidget {
  const ProfileSectionOwnerNote({super.key, required this.message});

  final String message;

  /// The note for [audience], or `null` when everyone can see the section.
  static String? messageFor(AppLocalizations l10n, String? audience) =>
      switch (audience) {
        'members' => l10n.profileOwnerMembers,
        'followers' => l10n.profileOwnerFollowers,
        'mutual' => l10n.profileOwnerMutual,
        'self' => l10n.profileOwnerSelf,
        _ => null,
      };

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return Padding(
      key: const ValueKey('profile-section-owner-note'),
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: colors.base200,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Padding(
          padding: const EdgeInsetsDirectional.fromSTEB(12, 4, 4, 4),
          child: Row(
            children: [
              GfSymbol('eye', size: 16, color: colors.iconMuted),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  message,
                  style: type.caption.copyWith(
                    color: colors.baseContent.withValues(alpha: .75),
                  ),
                ),
              ),
              TextButton(
                onPressed: () => context.push('/settings/profile-privacy'),
                child: Text(l10n.profilePrivacyManage),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
