# Mobile performance

> Doc type: reference
>
> Status: Active
>
> Owner: Mobile maintainers
>
> Last verified: 2026-09-25

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

`Current`: mobile CI retains an Android arm64 release size report with commit, Flutter version,
build mode and APK size. Its build-only signing key and OEM configuration are not distribution
credentials. Compare reports with matching toolchain, ABI and build configuration.

`Partial`: reproducible physical-device cold/warm startup distributions, frame-time percentiles,
and production-equivalent size budgets have not been established. Simulator debug timings and
a passing compile do not establish production performance. Capture profile/release timelines
on target devices before assigning latency or frame-rate guarantees.

Run the relevant package tests and analyzer using the [testing guide](testing.md). Pixel goldens
are platform-specific and are separate from behavior regression tests.
