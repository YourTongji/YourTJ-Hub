# Stable Android download links

## Status

Accepted
Class: process

## Context and Problem Statement

Versioned Android assets include both version and ABI build code in their names. A link to an APK
therefore changes at every release. GitHub's repository-wide latest release belongs to the server;
using it for Android would disrupt existing server download links.

## Decision Drivers

- Provide a fixed URL for each supported Android ABI without another hosting service.
- Preserve signed APK bytes, versioned release assets and the server latest marker.
- Recover failed updates without rebuilding APKs or rolling the download channel backward.
- Keep in-app update discovery tied to immutable, versioned release metadata.

## Considered Options

1. A separate mutable GitHub release named `mobile-latest` with fixed asset names.
2. Mark mobile releases as the repository-wide latest and add fixed asset names there.
3. Operate a separate HTTP redirect endpoint that resolves the latest mobile release.

## Decision Outcome

Choose option 1 as a download convenience channel extending
[versioned mobile distribution](0014-mobile-release-distribution.md). The Android release job runs
the alias publisher after the original release is public and verified, within the existing
`mobile-release` concurrency group. It selects the highest stable `mobile-vX.Y.Z` among the most
recent 100 releases, downloads all three APKs, verifies their sizes and GitHub SHA-256 digests and
consistent ABI build numbers, then updates only `mobile-latest` assets. Aliases are named
`YourTJ-ABI.apk`; a checksum file and release notes identify the original version. Retries skip
matching bytes, and a recorded newer source prevents older recovery from replacing the channel.

The channel is a pre-release with `latest=false` so repository latest and in-app update discovery
ignore it. Its APK bytes come from the stable release; the pre-release flag classifies the channel,
not the binary. Initial creation uses a draft and publishes after GitHub confirms every uploaded
digest. Subsequent updates replace alias files individually, so a fixed URL can briefly be
unavailable and the checksum file may lag an APK. Original version links remain available for
consistent downloads. A failed update is resumed from the original assets, never a rebuild.

The alias tag anchors its initial source commit and is never moved. Generated source archives on
that page are not the current APK source; notes link the exact versioned source tag. Version tags
and assets remain immutable. The publisher refuses immutable releases, unrelated channel metadata
and stable releases at the alias name; it does not weaken repository release protections.
Standalone recovery requires that no mobile release publisher is running.

## Pros and Cons of the Options

- Option 1 uses existing infrastructure and signing provenance, but duplicates APK storage and
  accepts brief per-file update gaps rather than an atomic redirect.
- Option 2 is simpler for GitHub's generic latest URL but makes server and mobile releases compete
  for one marker; rejected.
- Option 3 can redirect atomically to immutable assets, but adds a serving endpoint, availability
  dependency and metadata caching solely for downloads; deferred.

## Links

- [Mobile release runbook](../operations/mobile-releases.md#android-apk-and-in-app-update)
- [GitHub release links](https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases)
- [GitHub releases API](https://docs.github.com/en/rest/releases/releases)
- [GitHub release asset digests](https://docs.github.com/en/rest/releases/assets)
