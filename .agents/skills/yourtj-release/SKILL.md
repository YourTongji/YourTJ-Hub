---
name: yourtj-release
description: Inspect release readiness or prepare reviewed Web, Android or iOS release requests in YourTJ Hub. Use for version planning, Release PRs, channel status and promoting an existing iOS build. Source development and ordinary PR publication use yourtj-development.
---

# YourTJ release

Read [the release runbook](../../../docs/operations/releases.md). The public interface is
`python3 scripts/release/cli.py` from the repository root; `--help` and subcommand `--help`
are authoritative. Git, Python and authenticated `gh` are sufficient. Signing and Oryn's private
runtime stay in Actions. No Codex-specific tool is required.

- Inspect intent first: a status/analysis request is read-only. `plan`, `doctor`, `status`, `validate`
  do not publish. `prepare`, `publish`, `retry` and `promote-ios` default to dry-run; use `--apply` within the
  user's existing authorization. Preparing a release creates a review request; it does not approve it.
- Identify platform and destination. `mobile` means Android plus iOS; iOS defaults to TestFlight.
  Don't widen Android to both platforms or TestFlight to App Store. Ask only if those choices are
  genuinely unspecified and cannot be inferred.
- Freeze a SHA already in main. A release never implicitly merges dev. If intended work is only in
  dev, report that dependency and follow the separately authorized promotion PR process.
- Run `doctor --scope <scope> --json`, then `plan` when baselines are locally available. Apple
  baseline discovery uses authenticated Prepare Actions; a local unknown Apple state is not proof
  of no previous release. Dispatch `prepare --scope ... --apply` after explicit preparation authority.
- Open the resulting `codex/release/<id>` PR against main. It contains only its candidate directory.
  Use [yourtj-release-notes](../yourtj-release-notes/SKILL.md) for edits. Validate the final files.
  One eligible human must APPROVE the final head; an agent's approval, label or manifest flag does
  not satisfy this requirement. Any source, target, evidence or note edit needs fresh review.
- After an authorized merge, inspect channel receipts with `status --candidate <id> --json`.
  If publication never started and no channel has an execution receipt, `publish --candidate <id>
  --apply` resumes initial publication from the current main controller with the same approved source
  and notes. A reserved tag is not a build. Existing execution receipts require `retry`/Recover.
  App Store submission is not live distribution. For partial outcomes route to
  [yourtj-release-recovery](../yourtj-release-recovery/SKILL.md).

Read [scenarios](references/scenarios.md) when deciding between a new version, a draft edit and
an existing-build promotion. Report source SHA, PR link, channels and actual observed outcomes.
