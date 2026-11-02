---
name: yourtj-release
description: Inspect release readiness or prepare reviewed Web, Android or iOS release requests in YourTJ Hub. Use for version planning, Release PRs, channel status and promoting an existing iOS build. Source development and ordinary PR publication use yourtj-development.
---

# YourTJ release

Read [the release runbook](../../../docs/operations/releases.md). The public interface is
`python3 scripts/release/cli.py` from the repository root; `--help` and subcommand `--help`
are authoritative. Git, Python and authenticated `gh` are sufficient. Signing and Oryn's private
runtime stay in Actions. No Codex-specific tool is required.

- Read-only status/planning never publishes. `rc`, `confirm`, `dev`, `publish`, `retry` and
  `promote-ios` default to dry-run; use `--apply` within existing explicit user authority.
- Dev builds follow every dev merge automatically. `dev --apply` retries the current dev source;
  do not start a prerelease cycle, freeze or QA admission.
- Select scope and destination from user intent. RC defaults to mobile/App Store; explicitly choose
  TestFlight-only for beta-only intent. Source must already be in main; promote source through a
  separately authorized code PR, never implicitly inside release preparation.
- Run `doctor`, then `rc --scope ... --apply`. Apple baseline discovery stays in authenticated Actions.
  RC prepares an editable release-data PR and builds final production configuration/version once.
- Use [yourtj-release-notes](../yourtj-release-notes/SKILL.md), validate final notes and inspect the
  immutable complete RC record. Schema-4 requests require explicit `confirm --candidate ID --apply`,
  sealing exact head/source/notes/targets/identity/RC before merge and original-artifact promotion.
  An agent may execute Confirm within explicit user publication authority. Do not fabricate a GitHub
  APPROVED review. Legacy schemas retain their current-head eligible human-review rule.
- Source/config/binary changes require a new RC identity; notes edits do not rebuild. A retry or Confirm
  reuses the original date ID, internal counter and signed artifacts. Never promote dev configuration.
- Inspect actual channel receipts and Apple state. Submission is not live distribution. If no publisher
  started after an authorized merge, `publish` resumes initial promotion; existing execution receipts
  use `retry`/Recover. Missing required originals never authorize rebuilding a distributed identity.
- `promote-ios` enters metadata-only RC preparation for a published schema-4 TestFlight build, followed
  by Confirm of the exact original Apple build. Legacy recovery stays with the original request.

Report source SHA, PR, displayed build ID, channels and actual outcomes. See
[yourtj-release-recovery](../yourtj-release-recovery/SKILL.md) for partial failures.
