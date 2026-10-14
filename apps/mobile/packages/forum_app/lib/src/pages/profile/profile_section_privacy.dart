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

/// Who else can see a section the owner restricted; `null` when everyone can.
String? profileOwnerAudienceNote(AppLocalizations l10n, String? audience) =>
    switch (audience) {
      'members' => l10n.profileOwnerMembers,
      'followers' => l10n.profileOwnerFollowers,
      'mutual' => l10n.profileOwnerMutual,
      'self' => l10n.profileOwnerSelf,
      _ => null,
    };

/// Lock reasons for the profile tab bar: visitors see which sections are not
/// public to them, the owner sees which ones have a narrower audience.
Map<String, String> profileTabLocks(
  AppLocalizations l10n,
  UserProfileProps props,
) {
  final locks = <String, String>{};
  for (final key in const [
    'timeline',
    'topics',
    'likes',
    'following',
    'followers',
  ]) {
    final state = profileStreamVisibility(props, key);
    if (state == null) continue;
    final note = !state.visible
        ? l10n.profileHiddenTitle
        : props.isOwnProfile
        ? profileOwnerAudienceNote(l10n, state.audience)
        : null;
    if (note != null) locks[key] = note;
  }
  // Bookmarks are never shown to anyone else, whatever the settings say.
  if (props.isOwnProfile) locks['bookmarks'] = l10n.profileOwnerSelf;
  return locks;
}
