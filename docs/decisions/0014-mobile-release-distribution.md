# Mobile distribution through versioned GitHub releases

## Status

Accepted
Class: process

## Context and Problem Statement

YourTJ needs installable Android APKs, in-app update discovery and coordinated TestFlight/App Store
submission without mixing mobile versions into the server release namespace. Mainland users can
experience slow GitHub asset downloads, but there is no owned CDN. Signing identities and Apple
review credentials must remain outside public source and release metadata.

## Decision Drivers

- Preserve Android signing continuity and verify untrusted mirror downloads.
- Build only reviewed production source with reproducible version identities.
- Keep public infrastructure optional and fail closed when trusted metadata is unavailable.
- Make uncertain uploads recoverable without duplicate Apple builds or overwritten APKs.
- Use the existing Apple account and GitHub Actions instead of another hosted release service.

## Considered Options

1. GitHub release metadata/digests, optional public APK mirrors, platform signing and ASC CLI.
2. An owned CDN and separately signed update manifest.
3. Public proxies as both update metadata and binary sources.
4. Manual APK uploads and manual iOS packaging for every release.

## Decision Outcome

Choose option 1. A patch/minor/major action reserves an annotated `mobile-vX.Y.Z` tag on a reviewed
main commit. The tag records the immutable version and increasing build number passed to Flutter;
pubspec supplies only the bootstrap/development floor. Recovery reuses the recorded identity. A protected environment supplies Android/iOS signing inputs and Apple review credentials.
Android assets are validated before publishing, retain GitHub's computed SHA-256, and never replace
same-name files. Only GitHub's HTTPS API supplies update metadata; public mirrors transport APK
bytes. The client checks the digest before Android verifies package/version/signing identity and
opens the system installer. iOS uses a checksum-pinned ASC CLI to upload once and submit the same
build to TestFlight and App Store review. Server `vX.Y.Z` releases retain their independent flow.
An explicit iOS-only recovery uses corrected publisher tools from the reviewed main dispatch,
tagged store metadata and Apple's existing exact version/build, without rebuilding or uploading.
TestFlight-only submission is opt-in and leaves pending App Store reviews unchanged; the default
combined submission reports an outstanding App Store version as a blocker.

Release preparation may run from dev with the repository-level `RELEASE_TOKEN`. This deliberately
trusts maintainers who can merge workflow or release-script changes into dev with that credential's
production capabilities. The main promotion check governs the selected release source; it is not
an isolation boundary for the token. The owner accepts this operational trust model.

This gives up update discovery when GitHub's API is unreachable. Public mirror availability and
Apple approval are not guaranteed. An owned CDN can replace the mirror list through a reviewed
app change; a future independent metadata authority needs its own signing and rotation decision.

## Pros and Cons of the Options

- **Option 1:** uses available infrastructure and platform trust without another signing-key scheme;
  requires GitHub metadata access and explicit mirror failure handling. Apple API changes and
  certificate expiry still need operational maintenance.
- **Option 2:** controls download availability and metadata reachability; currently lacks an owned
  service and adds CDN operations, manifest signing, revocation and key rotation.
- **Option 3:** makes metadata easier to reach but lets an untrusted proxy advertise updates and
  hashes together; rejected because a hash from the same untrusted source is not authenticity.
- **Option 4:** has low initial setup cost but repeats credential handling, metadata entry and
  version mistakes; retained only as a documented recovery route using the same validated artifacts.

## Links

- [Mobile release runbook](../operations/mobile-releases.md)
- [GitHub release asset digest API](https://docs.github.com/en/rest/releases/assets)
- [ASC CLI 5.0.0](https://github.com/rorkai/App-Store-Connect-CLI/releases/tag/5.0.0)
- [Android app signing](https://developer.android.com/studio/publish/app-signing)
- [Flutter 3.44.9 split ABI version mapping](https://github.com/flutter/flutter/blob/3.44.9/packages/flutter_tools/gradle/src/main/kotlin/FlutterPluginConstants.kt)
