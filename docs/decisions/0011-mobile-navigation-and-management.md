# Mobile navigation and shared management workspaces

## Status
Superseded by [0012](0012-unified-mobile-reading-navigation.md)
Class: architecture

## Context and Problem Statement

The mobile scope includes the forum, courses, scheduler, Wiki and every Web administration function.
The native shell needs clear campus entry points, while the complete administration console includes
many role-sensitive forms, configuration panels and data operations. Reimplementing each console form
creates a second place for permission and configuration behavior to diverge.

## Decision Drivers

- Native reading, publishing and campus workflows remain the main product experience.
- Every existing management function must remain accessible to authorized users.
- Keep numeric identity, server-side authorization, revocation and CSRF boundaries intact.
- Avoid duplicating backend business rules or building a generic raw-API editor.

## Considered Options

- Native primary workflows plus embedded first-party Web management workspaces.
- Reimplement every management page and setting in Flutter.
- Open administration in an external browser with a separate login.

## Decision Outcome

Use Home, Campus, Messages and Profile as the four persistent destinations; push search and editors
above the shell. Keep forum, account/content lifecycle, courses, scheduler and Wiki native. Reuse the
complete administration and moderation workspaces in a controlled in-app browser. OAuth account
connections reuse the site settings in the system browser with an explicit matching-account login;
returning refreshes native status. This avoids adding a second provider-binding protocol, at the
cost of a possible separate Web login for this infrequent account operation.

The authenticated handoff validates the native Bearer session and selected workspace permission,
sets an HttpOnly cookie and redirects to an allowlisted path. It neither issues an independent
identity nor puts a credential in a URL or script. The wrapper restricts origin navigation, handles
file selection/export and clears browser state when leaving. Management API gates remain Web's.

This replaces the navigation and management exclusions in
[0010](0010-mobile-route-a-native-alignment.md). Its scheduler-algorithm, theme-token and push-channel
choices remain the architectural basis; the scope change does not replace those implementations.

The cost is a distinct management rendering surface and native/WebView integration testing. The
benefit is that the full console continues to use one set of forms, validations and permissions.

## Pros and Cons of the Options

### Native core with shared management

- Good: native primary flows; complete management UI follows the same server and Web behavior.
- Good: existing admin fixes reach both clients without duplicating each form.
- Bad: browser session cleanup, navigation and file handling require explicit device tests.

### Entirely native management

- Good: one rendering technology and full control over mobile layouts.
- Bad: duplicates the large evolving configuration surface, including sensitive permission semantics.

### External browser

- Good: reuses all Web behavior with little app integration.
- Bad: separates session and navigation from the App, and breaks the requested integrated experience.

## Links

- [Mobile experience](../product/mobile-experience.md)
- [Identity and account lifecycle](../product/identity-and-access.md)
- [Native alignment foundation](0010-mobile-route-a-native-alignment.md)
- [Flutter WebView package](https://pub.dev/packages/webview_flutter)
