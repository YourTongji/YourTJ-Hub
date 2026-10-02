# Reviewed release requests, platform notes and deterministic publication

## Status

Accepted
Class: process

## Context and Problem Statement

Release preparation implicitly promotes dev, notes mix independent platforms, source checks have
multiple entrypoints and production rebuilds a binary already built for release archives. Mobile
and server recovery need stable source, approved notes and original signed artifact identities.

## Decision Drivers

- Require a human to review final notes and targets while automating mechanical work.
- Keep Android and iOS wording and distribution baselines independent.
- Preserve existing single-binary deployment, APK verification and Apple review behavior.
- Give external coding agents one documented, credential-light CLI.
- Make required CI completeness and exact-source deployment eligibility explicit.

## Considered Options

1. Reviewed release-data PR, deterministic controller and reusable platform publishers.
2. AI directly edits published Releases and App Store metadata.
3. Environment approval per job over the existing implicit promotion workflow.

## Decision Outcome

Choose option 1. Freeze main-history source and separate release metadata in a main-targeted request
PR. A fresh eligible human review is checked both by a status check and by the publisher. Oryn drafts
through a dedicated evidence-to-notes task; it cannot approve or publish. Preserve existing version
namespaces and signing identities. Source verification is independent of metadata CI. Recovery uses
original artifacts and checks actual external state.

CI has one domain selector and an always-running result aggregate. Deploy dev only after a current,
successful dev-push CI run. Production consumes the binary from GoReleaser and a digest-qualified image.
Existing dev check names remain compatibility aliases until protection settings use the aggregate.

This supersedes the orchestration and notes rules of [0014](0014-mobile-release-distribution.md);
its signing, APK verification and Android update-discovery constraints remain in the current runbook.

## Pros and Cons of the Options

- Option 1 makes source/notes/approval reviewable in Git and supports platform-specific retries;
  it adds a release-data PR and requires a trusted review controller and durable execution records.
- Option 2 removes a review step but cannot prove independent platform prose was human-approved
  and couples uncertain generated content to irreversible distribution.
- Option 3 controls execution timing but does not expose editable final notes or remove implicit
  dev promotion, duplicate builds and scattered CI selection.

## Links

- [Release operations](../operations/releases.md)
- [Unified CI](../../.github/workflows/ci.yml)
- [GitHub workflow token event behavior](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow)
- [GitHub concurrency](https://docs.github.com/en/actions/concepts/workflows-and-actions/concurrency)
- [Apple platform-version fields](https://developer.apple.com/help/app-store-connect/reference/app-information/platform-version-information)
