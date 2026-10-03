---
name: yourtj-release-recovery
description: Diagnose or resume a failed YourTJ release channel while preserving approved source, notes and signed artifacts. Use for partial Android/iOS/Web releases, uncertain uploads or rollback analysis; new features and new binary identities use normal preparation.
---

# YourTJ release recovery

Read [the release runbook](../../../docs/operations/releases.md) and the relevant
[failure cases](references/failure-cases.md). Start with
`python3 scripts/release/cli.py status --candidate <id> --json` and inspect the linked Actions run,
GitHub Release and Apple state when applicable. Query failures mean unknown, not absence/success.

Use `retry --candidate <id> --channel <channel> --dry-run` to review the exact action. Existing
release authority permits an idempotent retry of the same approved channel; use `--apply` within
that scope. Successful channels remain intact. Do not expand TestFlight approval to App Store,
withdraw another store version, overwrite APKs, recycle build numbers or choose a new source SHA.

Recover retained original build artifacts or the exact Apple build. If required artifacts expired
and the external platform has no usable original, stop that recovery and prepare a new reviewed
version when authorized. Rebuilding the same identity is not a retry. After an uncertain Apple
upload, query before transferring again. A recorded submission alone never means App Store live.

Production rollback is a separate operational decision. Identify the target image digest,
configuration and database compatibility; obtain explicit rollback intent if absent. Never restore
or overwrite production DB automatically. The release retry command resumes the same approved
image and does not authorize an arbitrary older image.
