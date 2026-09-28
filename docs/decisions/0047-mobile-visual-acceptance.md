# Mobile visual acceptance without screenshot baselines

## Status

Accepted
Class: testing

## Context and Problem Statement

Screenshot baselines require a matching Flutter renderer and font environment, while routine mobile
changes already have behavior, layout, accessibility and design-token assertions. Maintaining PNG
baselines separately from CI still imposes recurring work without establishing native device quality.

## Decision Drivers

- Keep automated checks focused on observable behavior and meaningful layout constraints.
- Remove screenshot maintenance and rendering-environment coupling from the repository.
- Verify visual changes on the affected simulator or device screens in both themes.
- Preserve native compilation and signed release validation.

## Considered Options

- Run screenshot comparisons in a pinned CI rendering environment.
- Keep screenshot tests and PNG baselines as an optional local suite.
- Remove screenshot tests and baselines, retaining targeted assertions and manual visual acceptance.

## Decision Outcome

Remove the mobile screenshot golden tests, PNG fixtures and dedicated tag/exclusion configuration.
Package tests run normally in local, PR and release workflows. Keep behavior, geometry, accessibility,
font-inheritance and token assertions, including generated-image checks that do not compare against
stored screenshots. Fonts still used by text-layout tests remain test fixtures.

Visual changes receive focused simulator/device inspection in both themes. Acceptance captures stay
outside Git; App Store screenshots remain distribution assets and are unaffected.

This supersedes [0045](0045-mobile-ci-by-input.md). Its input-based job selection, independent package
tests, two `forum_app` shards, native builds on native/dependency changes, optional native dispatch,
and mandatory signed release verification remain the current model. Only screenshot-baseline
maintenance is removed.

## Pros and Cons of the Options

- Pinned screenshot CI detects pixel changes, but couples maintenance to the rendering toolchain and
  still cannot verify physical-device interactions.
- Optional local baselines avoid a CI gate but continue accumulating stale screenshots and special
  regeneration instructions; rejected.
- Targeted assertions plus visual inspection reduce artifacts and make automated failures actionable,
  but arbitrary pixel shifts require human inspection and are not automatically detected.

## Links

- [Testing guide](../development/testing.md)
- [Mobile CI](../../.github/workflows/ci-mobile.yml)
- [Mobile release verification](../operations/mobile-releases.md#release-verification-and-privacy-disclosures)
- [Implementation and maintainer-requested removal](https://github.com/YourTongji/YourTJ-Hub/pull/925)
