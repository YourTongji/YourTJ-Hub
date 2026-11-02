# Mobile performance

> Doc type: reference
>
> Status: Active
>
> Owner: Mobile maintainers
>
> Last verified: 2026-10-30

## Startup and rendering

`Current`: native launch markers and Flutter timeline events distinguish the first rasterized frame,
home skeleton, parsed home data, first content and first image. Android reports fully drawn when
useful home content is submitted. These are measurement hooks, not measured startup guarantees.

`Current`: session tokens use serialized secure-storage mutations and an in-memory read cache.
Home can show an account/site/sort-scoped cached first page while refreshing from the server.
The shared offline-cache clear operation removes those pages. Topic details publish network content
before awaiting optional cache persistence. Search and course results use lazy scrolling lists.

`Current`: foreground conversation polling preserves the initial history request and avoids
overlapping loads. Hiding a conversation cancels its request; returning resumes polling.

`Current`: feed media accepts intrinsic dimensions and server variants, selects a suitable source
and bounds decoded image sizes. Gallery and small-image decoding preserve the source aspect ratio.
Legacy URLs retain a fallback layout. The derivative storage and cleanup contract belongs to
[object storage](../operations/object-storage.md).

## Measurement boundaries

`Current`: the native mobile CI workflow retains an Android arm64 release size report with commit,
Flutter version, build mode and APK size. Its build-only signing key and OEM configuration are not distribution
credentials. It runs for native/dependency changes or manual dispatch, not every Dart change;
see the [CI mapping](testing.md#ci-mapping). Compare reports with matching toolchain, ABI and build
configuration.

`Current`: native Android CI rejects APK growth above 3% of the checked-in
[`android_apk_size_baseline.json`](../../scripts/ci/android_apk_size_baseline.json).
The ceiling is inclusive and measured in whole bytes. The report is uploaded before the gate runs,
so a size regression retains its analysis. This is a fixed baseline, not an automatically advancing
comparison with the previous build. Ordinary builds never rewrite it.

The baseline records the measured source commit, CI run, artifact ID and archive SHA-256.
Flutter version, ABI and build mode must match. The recorded Android build-configuration hash
covers the Gradle wrapper, settings, root/app build scripts and Gradle properties; changes fail
closed rather than compare silently across build settings. This conservative whole-file hash also
detects native dependency versions and comment-only changes in those files; Dart dependency changes
alone remain subject to the size comparison. The initial hash was reconstructed
from its recorded source commit because that older artifact did not contain the hash.
For a toolchain/configuration change or an accepted size increase, inspect the uploaded report and
submit a reviewed baseline update with the new measurement and provenance. Do not raise the limit
or copy a failing size into the baseline merely to make CI green. Build-only signing and OEM inputs
must remain comparable; changing that workflow setup also requires reviewing the baseline.

`Partial`: this budget covers builds selected by the native CI policy, not every Dart/asset-only PR.
Budget checker, test and baseline changes select an Android build. It is not a production APK,
store-download or installed-size budget; distribution signing and configuration can change size.

`Partial`: reproducible physical-device cold/warm startup distributions, frame-time percentiles,
and production-equivalent size budgets have not been established. Simulator debug timings and
a passing compile do not establish production performance. Capture profile/release timelines
on target devices before assigning latency or frame-rate guarantees.

Run the relevant package tests and analyzer using the [testing guide](testing.md). Visual acceptance uses a simulator
or device; automated regression tests cover behavior and layout without screenshot baselines.
