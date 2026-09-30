# Automatic mobile visitor statistics

## Status

Accepted
Class: feature

## Context and Problem Statement

Mobile public-page statistics require a device-local opt-in while the Web public-page
collector starts automatically. Issue #937 requests automatic mobile statistics with no
settings switch. The maintainer selected that behavior for both new and existing installations.

## Decision Drivers

- Count public-page usage without a separate setup step for guests or signed-in users.
- Keep the existing narrow payload, platform restrictions and best-effort transport.
- Make upgrade behavior and the absence of an in-app switch explicit in the privacy disclosure.

## Considered Options

- Retain explicit opt-in.
- Enable by default and retain an opt-out switch and persisted preference.
- Collect automatically without a switch or a preference gate.

## Decision Outcome

Collect automatically without a switch in production Android/iOS release builds. Remove
the consent provider, settings control and its translations. The host gates collection by
foreground lifecycle; the provider gates it by platform, build mode and production origin.
The old `visitor_analytics_opt_in` preference is neither read nor written, including when
its stored value is `false`. Upgrades and fresh installations therefore behave identically.

Keep the public-route allowlist, sanitized fixed categories, independent credential-free
transport, memory-only queue/token, bounded timeouts, cancellation and no retries. This
does not add account identifiers, content, private screens or advertising tracking.
The product specification owns the full data boundary. The embedded privacy supplement
discloses automatic collection and no switch; release notes disclose the changed default.
App Store privacy metadata and production reporting still require release-time verification.

## Pros and Cons of the Options

- Explicit opt-in preserves the previous user choice but does not meet the requested automatic
  reporting behavior and leaves statistics limited to a self-selected sample.
- Default-on with opt-out improves coverage and retains user control, but keeps the settings
  control and preference lifecycle that the maintainer explicitly chose to remove.
- Automatic collection removes configuration and preference failure paths. It gives up in-app
  opt-out, including previously saved choices; that trade-off must remain disclosed. Expanded
  reporting coverage can change aggregate trends without a corresponding change in actual usage.

## Links

- [Issue #937: automatic mobile visitor statistics](https://github.com/YourTongji/YourTJ-Hub/issues/937)
- [Mobile data boundary](../product/mobile-experience.md#profile-and-privacy)
- [Distribution verification](../operations/mobile-releases.md#native-visitor-statistics)
- [Embedded App privacy disclosure](../../apps/gooseforum/app/models/defaultconfig/pageconfig/app_privacy.md)
