import 'package:core/core.dart';
import 'package:passkeys/exceptions.dart' as passkey_exceptions;

import '../../../l10n/app_localizations.dart';
import '../../server_messages.dart';

/// Converts platform authenticator failures into a safe, localized fallback.
/// In particular, no-provider and unsupported-device paths return users to the
/// existing password/OIDC options without leaking native exception text.
String resolvePasskeyErrorMessage(
  AppLocalizations l10n,
  Object error, {
  required String fallback,
}) {
  if (error is StateError) return l10n.authPasskeyUnavailable;
  if (error is passkey_exceptions.PasskeyAuthCancelledException ||
      error is passkey_exceptions.NoCredentialsAvailableException) {
    return l10n.authPasskeyCancelled;
  }
  if (error is passkey_exceptions.DeviceNotSupportedException ||
      error is passkey_exceptions.PasskeyUnsupportedException ||
      error is passkey_exceptions.NoCreateOptionException ||
      error is passkey_exceptions.MissingGoogleSignInException ||
      error is passkey_exceptions.SyncAccountNotAvailableException) {
    return l10n.authPasskeyUnavailable;
  }
  if (error is passkey_exceptions.AuthenticatorException) return fallback;
  if (error is ApiException) return resolveErrorMessage(l10n, error);
  return fallback;
}
