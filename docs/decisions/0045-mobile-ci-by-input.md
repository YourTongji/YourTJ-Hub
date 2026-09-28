# Mobile CI separates behavior tests from native compilation

## Status
Superseded by [0047](0047-mobile-visual-acceptance.md)
Class: testing

## Context and Problem Statement

Running both native builds for every Dart, test, documentation or release-script change delays
feedback without providing the same value for each input. Serial package tests also leave independent
work waiting on one runner. Pixel baselines require a matching rendering environment.

## Decision Drivers

- Give source changes fast analysis and complete behavior-test coverage.
- Keep compile checks for native code, plugin dependencies and release build configuration.
- Avoid concurrent Flutter commands competing for one SDK startup lock.
- Keep native builds and Android size reports available on demand.

## Considered Options

- Run every check and both native builds for every mobile change.
- Split behavior tests and native compilation, selecting jobs by their inputs.
- Remove native compilation from pull requests entirely.

## Decision Outcome

Select independent analysis, package test jobs and a Python release-tool job. Split `forum_app` tests
into two Flutter shards; every test job has its own runner and SDK. All four package suites run for
Flutter changes so changes to shared packages exercise their consumers.

A separate native workflow selects Android and/or iOS from changed platform paths. Shared dependency
manifests and lockfiles, native build hooks/scripts and release tooling select both. It retains Android
release shrinking, all push adapters and size artifacts, and iOS compilation/signing-setting checks.
Manual dispatch selects both platforms. Signed release verification and builds remain mandatory.

Pure Dart or asset changes rely on analysis and behavior tests until a native build is requested or a
release is built. Platform-conditional compilation and release-only failures can therefore surface at
that later build; maintainers can dispatch native CI for changes needing this evidence.

Pixel goldens remain excluded from behavior tests, and there is no golden-refresh workflow. Intentional
baseline generation is a local Linux/Flutter task. This replaces the golden-refresh guidance in the
already-superseded [0010](0010-mobile-route-a-native-alignment.md); other mobile architecture decisions
are unaffected.

## Pros and Cons of the Options

- Running everything provides native compile evidence on each PR, but repeatedly spends build time
  on changes with no native impact and serializes unrelated tests.
- Selecting independent jobs reduces that unnecessary work and shortens test feedback, but more test
  runners repeat SDK/bootstrap setup and Dart-only PRs lack automatic native compile evidence.
- Removing native PR builds is cheapest, but leaves native/plugin regressions until release; rejected.

## Links

- [Behavior workflow](../../.github/workflows/ci-mobile.yml)
- [Native workflow](../../.github/workflows/ci-mobile-native.yml)
- [Testing commands and CI mapping](../development/testing.md)
- [Release verification](../operations/mobile-releases.md#release-verification-and-privacy-disclosures)
