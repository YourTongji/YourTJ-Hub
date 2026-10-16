# Forum-owned cross-platform passkeys

## Status

Accepted
Class: feature

## Context and Problem Statement

Issue #1151 requests passkey sign-in across Web, iOS and Android. The forum owns account identity,
numeric user IDs, sessions and TOTP. A passkey implementation therefore needs to fit the existing
identity boundary, support discoverable credentials, and avoid introducing a second session system.
Passkeys also require explicit per-instance RP domains and Apple/Android app association values.

## Decision Drivers

- Keep the forum `users` table as the sole identity source and issue the normal revocable forum session.
- Support a discoverable, passwordless login on all requested clients with authenticator user
  verification required.
- Treat a verified passkey with UV as a complete login path, including for accounts that have TOTP
  enabled, while leaving password/TOTP and social/OIDC policies intact.
- Keep account recovery available through the existing email/password flow.
- Require fresh password authentication, plus TOTP when enabled, before credential management.
- Fail closed when RP origin or app signing association is incomplete.
- Keep the forum's single-binary deployment shape and avoid an external identity dependency.

## Considered Options

- Make the forum the WebAuthn relying party and store credentials beside forum accounts.
- Delegate passkey authentication to an external identity provider or to the built-in OIDC provider.
- Add passkeys only as another factor after password/TOTP login.

## Decision Outcome

Chosen option: "Make the forum the WebAuthn relying party and store credentials beside forum
accounts." The Go backend uses `go-webauthn`; browser clients use the WebAuthn API and Flutter
clients use `passkeys`, whose iOS implementation calls AuthenticationServices and Android
implementation uses AndroidX Credential Manager. Discoverable resident credentials and user
verification are required. A successful assertion issues the same `jti`-backed forum session as
password or OAuth login.

A verified passkey assertion with UV is a complete login credential. It does not ask for TOTP when
the account has TOTP enabled; password login keeps its existing TOTP challenge. This is an explicit
alternative sign-in path and never disables, resets, or recovers TOTP. Social and OIDC MFA behavior
is unchanged. Recovery stays on the existing email/password path.

Passkey enrollment is available only for an existing signed-in account. A short-lived, one-use
proof binds password authentication and the current session, and also requires a TOTP code when
TOTP is enabled. The same proof protects rename and revoke operations. Passkeys do not replace the
existing email/password recovery flow and do not create accounts. Account closure removes stored
credentials and pending ceremonies.

Configuration is disabled by default. Enabling an instance requires an RP ID, explicit HTTPS origins,
Apple Team ID, iOS bundle ID, Android package name, and verified Android signing-certificate
fingerprints. The service publishes the corresponding Apple and Android association documents.
Android's opaque app origin is admitted only when derived from an explicitly configured signing
fingerprint; web origins remain exact HTTPS values within the RP domain. Apple AASA and Android
Digital Asset Links association are deployment requirements. The Android plugin documents a fix for
devices without Google Play Services in its 2.10.0 release notes and requires Android API 28 at
runtime. Its README also describes Google account requirements for Google Password Manager sync;
this does not establish that every OEM provider works. Xiaomi mainland support still requires real
GMS-free device acceptance. Other domestic Android brands remain best effort and no vendor SDK is
used. Production device acceptance remains `Partial` until iPhone/iPad and Android credential
providers are verified.

## Pros and Cons of the Options

### Forum-owned WebAuthn relying party

- Good, because account identity, credential lifecycle and session issuance stay inside the forum's
  existing ownership boundary.
- Good, because discoverable credentials let users sign in without first entering a username.
- Good, because browser and native clients share the same ceremonies and recovery path.
- Bad, because the operator must configure correct Apple Team ID and Android signing fingerprints
  for each relying-party domain.
- Bad, because platform credential providers and domain association need real-device acceptance.

### External provider or OIDC delegation

- Good, because a provider may own platform association and credential lifecycle.
- Bad, because it adds another identity binding and recovery surface, while the forum still needs to
  map the external subject to its numeric user ID and issue a forum session.
- Bad, because it makes the first-party login contract depend on an external provider.

### Passkey as a password second factor

- Good, because it layers onto the existing password challenge.
- Bad, because it does not provide the requested passwordless sign-in and still asks users to retain
  and enter a password before using the credential.

### Vendor-specific mobile authentication SDKs

- Good, because a vendor could provide a tailored integration for its own credential manager.
- Bad, because it would make account authentication depend on vendor accounts or services, exclude
  GMS-free devices, and fragment the shared credential flow. The implementation uses platform APIs
  through a maintained Flutter plugin instead.

## Links

- [Issue #1151](https://github.com/YourTongji/YourTJ-Hub/issues/1151)
- [go-webauthn v0.18.2](https://pkg.go.dev/github.com/go-webauthn/webauthn@v0.18.2)
- [Flutter passkeys 2.23.1](https://pub.dev/packages/passkeys/versions/2.23.1)
- [Android plugin GMS-free fix and API 28 runtime requirement](https://pub.dev/packages/passkeys_android/changelog)
- [Xiaomi HyperOS passkey integration guidance](https://dev.mi.com/xiaomihyperos/documentation/detail?pId=1939)
