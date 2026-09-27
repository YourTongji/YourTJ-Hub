# Native Apple login with explicit account binding and revocable grants

## Status
Accepted
Class: architecture

## Context and Problem Statement

The iOS app offers Google and GitHub alongside password and Tongji login. It needs a privacy-preserving
Apple option while retaining existing account access. The forum's numeric user ID remains the identity
source, and social login must not bypass the controlled registration path. Apple requires a way to
revoke its authorization when the user disconnects or deletes the account.

## Decision Drivers

- Retain existing login methods and use Apple's native authorization UI.
- Do not infer account ownership from an email address or collect an unnecessary name/email.
- Validate signed identity and single-use authorization codes on the server.
- Revoke Apple authorization without requiring users to visit a second settings app during deletion.
- Keep provider secrets off clients, public API payloads, logs and source control.

## Considered Options

- Remove Google/GitHub from iOS and rely on forum and school accounts.
- Add native Apple authorization with explicit binding, signed-token verification and an encrypted revocation grant.
- Use a web Apple OAuth callback and automatically merge accounts with matching email addresses.

## Decision Outcome

Use native AuthenticationServices on iOS with Apple's system sign-in button. Request no profile scopes.
A cryptographically random client nonce is hashed into the Apple request; a separate state binds the
native callback. The server checks RS256 signature, issuer, bundle-ID audience, expiration, subject and
nonce, redeems the single-use code with a short-lived ES256 client secret, and verifies the exchange
identity has the same subject and nonce. Only then can it issue a numeric forum session.

Users first connect Apple from an authenticated, writable account's settings. Unbound Apple login
returns an explicit binding instruction; it cannot create or merge a forum account. Binding, login,
unlink and closure serialize on the owning user row. Closed, frozen and agent accounts cannot log in.

Unlike unused GitHub/Google credentials, Apple refresh grants have a required revocation purpose.
The binding stores only a purpose- and account-scoped encrypted refresh token, excluded from JSON.
No access token, raw ID token, name or Apple email is retained. Unlink and closure revoke the grant
before deleting the binding. Failure keeps the local account/binding retryable. The iOS device keeps
only the Apple subject and matching forum session jti in Keychain, checks credential state on foreground
and revocation notifications, and invalidates only that Apple-authenticated session. Other login
methods and renewed tokens with the same jti remain correctly distinguished.

The signing pipeline requires Apple login entitlements in both the distribution profile and exported
app. Server private keys belong to the production deployment environment. Dev uses production database
snapshots and receives no production Apple credentials, so it cannot revoke production authorizations.

## Pros and Cons of the Options

- Removing social login simplifies setup but breaks established user paths.
- Native authorization avoids a web service ID, callback-cookie transport and private-email relay
  configuration. It requires native UI integration, a server signing key and encrypted grant lifecycle.
  Apple users must explicitly bind an existing account first; device and service configuration remain
  release acceptance requirements.
- A web callback could serve more platforms, but adds configuration and cookie/redirect handling.
  Email-based merging would confuse Apple relay aliases with the forum's identity proof and is rejected.

## Links

- [Identity and account lifecycle](../product/identity-and-access.md)
- [Mobile release operations](../operations/mobile-releases.md)
- [Apple login services guideline](https://developer.apple.com/app-store/review/guidelines/#login-services)
- [Apple user verification](https://developer.apple.com/documentation/signinwithapple/verifying-a-user)
- [Apple account deletion and revocation](https://developer.apple.com/documentation/technotes/tn3194-handling-account-deletions-and-revoking-tokens-for-sign-in-with-apple)
