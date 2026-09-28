import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../apple/apple_sign_in.dart';
import '../../apple/apple_sign_in_button.dart';

/// Sign-in choices other than the account/password form.
enum SignInMethod { apple, tongji, google, github }

/// Providers the published login options expose for the current mode.
///
/// Registration creates the account first, so it offers the school identity
/// only; Google, GitHub and Apple belong to the login mode.
List<SignInMethod> availableSignInMethods(
  LoginPageProps options, {
  required bool loginMode,
  bool? nativeAppleAvailable,
}) {
  final bool appleAvailable = nativeAppleAvailable ?? supportsNativeAppleSignIn;
  return <SignInMethod>[
    if (loginMode && options.appleReady && appleAvailable) SignInMethod.apple,
    if (options.tongjiReady) SignInMethod.tongji,
    if (loginMode && options.googleReady) SignInMethod.google,
    if (loginMode && options.githubUrl.isNotEmpty) SignInMethod.github,
  ];
}

/// Secondary sign-in providers behind a short bottom sheet.
///
/// Pops with the chosen [SignInMethod]; the page owns the network flow so the
/// sheet never outlives the choice. Tongji keeps its notice and policy links,
/// which describe that one flow.
class SignInMethodsSheet extends StatelessWidget {
  const SignInMethodsSheet({
    super.key,
    required this.methods,
    required this.termsOfServiceEnabled,
    required this.privacyPolicyEnabled,
  });

  final List<SignInMethod> methods;
  final bool termsOfServiceEnabled;
  final bool privacyPolicyEnabled;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 20, 16, 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            Text(
              l10n.authSignInMethods,
              style: GfTheme.typographyOf(context).title2,
            ),
            const SizedBox(height: 12),
            for (final method in methods) ...<Widget>[
              _methodButton(context, l10n, method),
              const SizedBox(height: 8),
            ],
            if (methods.contains(SignInMethod.tongji)) ...<Widget>[
              const SizedBox(height: 4),
              Text(
                l10n.loginTongjiHint,
                style: Theme.of(context).textTheme.bodySmall,
              ),
              if (termsOfServiceEnabled || privacyPolicyEnabled) ...<Widget>[
                const SizedBox(height: 8),
                Text(
                  l10n.loginTongjiPolicies,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                Wrap(
                  children: <Widget>[
                    if (termsOfServiceEnabled)
                      TextButton(
                        onPressed: () => context.push('/terms'),
                        child: Text(l10n.siteInfoTerms),
                      ),
                    if (privacyPolicyEnabled)
                      TextButton(
                        onPressed: () => context.push('/privacy'),
                        child: Text(l10n.siteInfoPrivacy),
                      ),
                  ],
                ),
              ],
            ],
          ],
        ),
      ),
    );
  }

  Widget _methodButton(
    BuildContext context,
    AppLocalizations l10n,
    SignInMethod method,
  ) {
    void select() => Navigator.pop(context, method);
    if (method == SignInMethod.apple) {
      return AppleSignInButton(onPressed: select);
    }
    return OutlinedButton.icon(
      icon: method == SignInMethod.tongji
          ? SvgPicture.asset(
              'assets/images/tongji-university.svg',
              width: 32,
              height: 32,
              excludeFromSemantics: true,
              colorFilter: Theme.of(context).brightness == Brightness.dark
                  ? ColorFilter.mode(
                      GfTheme.colorsOf(context).info,
                      BlendMode.srcIn,
                    )
                  : null,
            )
          : GfSymbol(method.name, size: 22),
      label: Text(switch (method) {
        SignInMethod.tongji => l10n.loginTongji,
        SignInMethod.google => l10n.loginGoogle,
        SignInMethod.github => l10n.loginGithub,
        SignInMethod.apple => '',
      }, textAlign: TextAlign.center),
      onPressed: select,
    );
  }
}
