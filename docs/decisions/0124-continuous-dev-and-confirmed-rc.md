# Continuous dev distribution and RC confirmation

## Status

Accepted
Class: process

## Context and Problem Statement

Development Web deployment and manually owned/frozen mobile previews serve the same need: testing the
current dev source. Separate cycles, QA admission and expiry make that path slower and obscure which
source each platform runs. Optional direct production builds introduce a second final release path.
The maintainer explicitly authorizes automatic same-source dev delivery and production RC then Confirm.

## Decision Drivers

- Make every merged dev change available to developers without another release ceremony.
- Keep Web and native build latency independent while identifying their actual sources.
- Build production bytes once, then confirm final notes and targets without compiling again.
- Display readable date/daily build identity while preserving native upgrade ordering.
- Retain trusted signing, source isolation, exact confirmation and original-artifact recovery.

## Considered Options

1. Keep manual prereleases, release owners, dev freeze and optional production RC.
2. Automatically distribute dev; make production RC and explicit Confirm the only new release path.
3. Publish production automatically from dev or reuse dev packages for production.

## Decision Outcome

Choose option 2. Every dev push requests the trusted-main distributor with the same source SHA.
Web and mobile start independently without waiting for a repeated source test matrix. Normal PR CI
remains independent. One mobile bundle runs and only the latest pending bundle remains; active uploads
finish. Platform and instance transactions still serialize safely. Public dev status records current
dev source and each platform's latest successful source. New builds have no cycle owner, freeze or expiry.

Production RC prepares a main-history source request and builds final configuration/version once.
Schema 4 requires a selected immutable RC before explicit operator Confirm. Confirm seals exact source,
notes/content digest, scope, head, numbers, RC and operator in a trusted immutable Actions artifact,
then merges only that head. Publishers independently verify it and promote original bytes. A deployment
pointer alone is not authority. An authorized agent may execute the user's Confirm; it cannot fabricate
a GitHub human review. Legacy schemas retain their original authorization and recovery protocol.

UTC+8 date/daily IDs reset at 0001 and are atomically reserved across dev and RC. Failed allocations are
not reused; retries and Confirm reuse original identity. Android keeps an internal increasing counter
because its versionCode cannot hold a twelve-digit date ID. iOS uses the date ID natively, with the
separate compiled counter keeping update/catalog comparisons consistent. Actual Apple upload validation
establishes native acceptance. Source/config changes require a fresh candidate, not artifact replacement.

This supersedes [0123](0123-build-only-releases-and-production-rc.md)'s optional direct-build choice and
[0099](0099-shared-dev-release-cycles.md)'s new-cycle ownership/freeze model. Build-only publication and
original-artifact recovery remain. Existing legacy operations are recoverable but cannot start new cycles.

## Pros and Cons of the Options

- Option 2 removes duplicated ceremonies and tests the production bytes that become final. Native
  packaging still takes compiler/signing/Apple time; platforms can temporarily show different sources.
  Maintainers own source selection and explicit release authorization. Artifacts have finite retention.
- Option 1 retains more checkpoints but requires manual coordination for ordinary development and
  leaves two production paths to understand and recover.
- Option 3 is automatic but dev endpoints, preview flags and package/storage identity are compiled
  inputs; publishing those bytes would not produce the intended production application.

## Links

- [Dev delivery](../operations/release-cycles.md)
- [RC and Confirm protocol](../operations/releases.md)
- [GitHub concurrency](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency)
- [Android versionCode limit](https://developer.android.com/studio/publish/versioning)
- [Apple CFBundleVersion](https://developer.apple.com/documentation/bundleresources/information-property-list/cfbundleversion)
- [Apple choosing an uploaded build](https://developer.apple.com/help/app-store-connect/manage-builds/choose-a-build-to-submit)
