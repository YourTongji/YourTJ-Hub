# Build-only publication and independent production RC artifacts

## Status

Superseded by [0124](0124-continuous-dev-and-confirmed-rc.md)
Class: process

## Context and Problem Statement

The release pipeline repeats source CI and native builds before signed packaging. Three comparable
successful three-platform previews take 29:00, 34:01 and 35:20. In the latest, native iOS verification
takes 22:25 and Web CI 19:19; only then does Android signing take another 11:29. Runner queue time is
seconds. Production rehearsal uses preview compilation flags and a preview build-number reservation,
so formal publication rebuilds a package that is different from the rehearsed one.

The maintainer explicitly chooses build-only publication, separate dev prerelease and production RC
entries, optional RC use and reuse of final artifacts. Development CI remains independent.

## Decision Drivers

- Remove repeated release-time test matrices and cross-platform waiting.
- Test/distribute the same production bytes that can later be published.
- Retain final request review, signing isolation, immutable identity and original-artifact recovery.
- Reuse existing builders, receipts and release request metadata rather than add a service/state store.

## Considered Options

1. Keep source checks, dev QA and production rehearsal before every formal build.
2. Build directly, with optional production RCs selected in the final release request.
3. Promote development packages to production by editing distribution metadata.

## Decision Outcome

Choose option 2. Prerelease builds dev configuration. Schema-3 release requests resolve fixed
main-history source, reserve final production version/build identity and have nullable `rcRun`.
They do not depend on dev-cycle or production-rehearsal QA. Direct publication builds once; an
optional RC uses the same platform builders with production configuration and `preview=false`.
Release workflows do not invoke source CI, browser suites or native simulator tests.

RCs distribute Android candidate downloads, normal Apple uploads assigned only to the internal QA
group, and Web archives/images without production deployment. A successful trusted-main RC seals
source, candidate, controller and platform artifact references in an immutable Actions artifact.
It attaches that run ID to the editable request through the existing preparation App identity;
formal publication still requires human approval of the final request head. Notes can change without
rebuilding. Source/config/version changes require a new candidate/build identity.

Promotion restores original APKs, submits the exact Apple build and deploys the original Web image
digest. Final version tags are reserved at formal publication, while RC tags identify candidate/run.
Missing/expired original artifacts and uncertain uploads do not authorize a silent rebuild under
an already distributed identity. New schemas retain compatibility recovery of schema-1/2 requests.
The legacy dev-preview/cycle controller remains responsible only for its existing supported paths.
This supersedes source-test/rehearsal prerequisites for new releases in
[0055](0055-reviewed-release-pipeline.md) and [0099](0099-shared-dev-release-cycles.md);
their final-review and legacy recovery boundaries remain applicable.

## Pros and Cons of the Options

- Option 2 removes duplicated tests/builds and makes RC-to-final identity explicit. It relies on
  normal development CI and maintainer source selection, and retained artifacts have finite lifetime.
  A failed build can spend signing-runner time without a preceding functional-test gate; this tradeoff
  is explicitly accepted. It does not prove device acceptance or Apple approval.
- Option 1 gives another automated test pass but adds repeated startup/compilation and tests a
  different binary before rebuilding formal production artifacts.
- Option 3 is inexpensive but development endpoints, package IDs, storage namespaces and preview
  flags are compiled inputs. Metadata edits cannot turn those bytes into production configuration.

## Links

- [Release runbook](../operations/releases.md)
- [Latest three-platform preview timing](https://github.com/YourTongji/YourTJ-Hub/actions/runs/41388074061)
- [Previous three-platform preview timing](https://github.com/YourTongji/YourTJ-Hub/actions/runs/41328766604)
- [Initial three-platform preview timing](https://github.com/YourTongji/YourTJ-Hub/actions/runs/41163657174)
- [GitHub immutable release assets and tags](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases)
- [Apple choosing an already uploaded build](https://developer.apple.com/help/app-store-connect/manage-builds/choose-a-build-to-submit)
- [Apple Internal Only build restriction](https://developer.apple.com/help/app-store-connect/test-a-beta-version/add-internal-testers)
