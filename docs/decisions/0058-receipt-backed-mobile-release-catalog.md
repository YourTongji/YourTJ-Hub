# Receipt-backed mobile release catalog

## Status

Proposed
Class: feature

## Context and Problem Statement

The mobile client needs human-reviewed release notes, but channel releases can complete independently
and older published builds may lack structured notes. A public catalog must not imply a channel was
released merely because a candidate exists or an App Store submission is pending.

## Decision Drivers

- Bind displayed notes to reviewed candidate bytes and successful channel receipts.
- Keep Android, TestFlight and public App Store availability distinct.
- Represent incomplete historical coverage without inventing notes.
- Keep the status endpoint display-only and bounded.

## Considered Options

1. Generate one catalog from reviewed candidates and successful per-channel receipts.
2. Publish notes directly from candidate metadata or latest store status.
3. Let mobile clients infer distribution from release tags.

## Decision Outcome

Choose option 1. New mobile release requests carry structured changelog facts and evidence
references. The trusted-main publisher validates receipt source SHA and candidate content digest,
requires verified APK digests for Android, `APPROVED` for TestFlight, and public storefront evidence
for new App Store builds. Previously verified public App Store history remains covered after the
store advances. The catalog records exact channel IDs and per-channel history coverage. The status
site proxies one fixed GitHub Release asset using bounded reads, redirects, and caching.

## Pros and Cons of the Options

- Option 1 creates an auditable source for client history and makes independent channel completion
  explicit; it adds a receipt-aware catalog publication step.
- Option 2 is simpler but treats pending review or candidate preparation as distribution evidence.
- Option 3 makes clients duplicate platform-specific release verification and cannot provide reviewed
  platform-specific notes.

## Links

- [Release operations](../operations/releases.md)
- [Mobile release operations](../operations/mobile-releases.md)
- [Status API contract](../../apps/status/api/openapi.yaml)
- [Issue #1016](https://github.com/YourTongji/YourTJ-Hub/issues/1016)
