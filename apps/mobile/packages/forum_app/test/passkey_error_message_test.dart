import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/auth/passkey_error_message.dart';
import 'package:passkeys/exceptions.dart';

void main() {
  late AppLocalizations l10n;

  setUpAll(() async {
    l10n = await AppLocalizations.delegate.load(const Locale('en'));
  });

  test('maps user cancellation and missing credentials to a safe retry hint', () {
    expect(
      resolvePasskeyErrorMessage(
        l10n,
        PasskeyAuthCancelledException(),
        fallback: 'generic',
      ),
      l10n.authPasskeyCancelled,
    );
    expect(
      resolvePasskeyErrorMessage(
        l10n,
        NoCredentialsAvailableException(),
        fallback: 'generic',
      ),
      l10n.authPasskeyCancelled,
    );
  });

  test('maps unsupported providers to the existing sign-in fallback', () {
    expect(
      resolvePasskeyErrorMessage(
        l10n,
        NoCreateOptionException('provider details'),
        fallback: 'generic',
      ),
      l10n.authPasskeyUnavailable,
    );
  });

  test('does not expose native domain association diagnostics', () {
    expect(
      resolvePasskeyErrorMessage(
        l10n,
        DomainNotAssociatedException('internal association details'),
        fallback: 'generic',
      ),
      'generic',
    );
  });
}
