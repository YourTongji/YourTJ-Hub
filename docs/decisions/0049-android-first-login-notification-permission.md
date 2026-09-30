# Android first-login notification permission

## Status
Accepted
Class: feature

## Context and Problem Statement

Android users who do not visit Settings can remain unaware of system notification permission. The
app already has a one-time post-login permission request on iOS and a native Android bridge that can
request `POST_NOTIFICATIONS` without initializing JPush. Android must request permission at a useful
time while preserving consent and device-collection boundaries.

## Decision Drivers

- Give authenticated Android users one clear opportunity to enable useful notifications.
- Do not interrupt login, OAuth return, or the initial main-screen frame.
- Never initialize JPush, obtain its token, or register a device before OS permission is granted.
- Avoid repeating the system prompt after denial, dismissal, restart, or upgrade.
- Keep OS permission, app preference, provider build configuration, and server channel status
  distinguishable.

## Considered Options

- Keep Android strictly opt-in through the Settings switch.
- Request once after login on both platforms, sharing the existing local marker and Android bridge.
- Add a separate in-app rationale dialog before the system prompt.

## Decision Outcome

After a valid Android session reaches the main app and its first frame is stable, request system
notification permission once if the device has not been authorized and the shared
`push_permission_requested` marker is absent, unless the stored app preference is explicitly false.
Existing Android installations without that marker receive the same one-time request after upgrading
when they have not explicitly disabled push in Settings. Preserve that opt-out across upgrade; the
shared one-shot marker is stored only after the OS permission call returns on either platform. If the
app process exits while a permission prompt is open, the next launch repeats the native permission
check/request and lets the OS determine whether another prompt is needed. The Settings switch remains
available for explicit retry and disabling.

Treat a system grant as push consent: set the app preference and continue through the existing
provider configuration, token, and device-registration path. Preserve the established iOS denial
semantics: after a denial the preference remains enabled and the channel state is `permissionDenied`;
the app does not initialize JPush or register a device, and Settings offers system notification
settings recovery. On Android versions below 13, read the existing system notification state without
calling the runtime permission API. Check OS permission before local JPush configuration so an
unconfigured build may request OS permission while still reporting `unsupported`; a disabled server
channel remains `serverDisabled`.

Do not add a second in-app rationale prompt. The authenticated main experience supplies context, and
Settings already explains provider processing. Keep JPush auto-initialization disabled; native
registration is still the only SDK initialization path.

## Pros and Cons of the Options

- Settings-only opt-in avoids an automatic prompt but leaves the permission gap for users who never
  discover Settings.
- One login-time request makes the choice visible at a relevant point and reuses the existing bridge
  and marker. Upgraded installations without the marker see the prompt once after their next login,
  unless they previously turned push off in Settings.
- A custom rationale can add context but creates another dialog and decline path without changing
  the system permission or provider contract.

## Links

- [Issue #943](https://github.com/YourTongji/YourTJ-Hub/issues/943)
- [Native push provider decision](0019-native-push-providers.md)
- [Mobile experience](../product/mobile-experience.md#system-notifications)
- [Android notification permission](https://developer.android.com/develop/ui/compose/notifications/notification-permission)
