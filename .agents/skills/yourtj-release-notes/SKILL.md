---
name: yourtj-release-notes
description: Draft, edit or review separate YourTJ Web, Android, iOS App Store and TestFlight notes using a frozen release request's evidence. Use for release-note wording and platform attribution, not release authorization or distribution.
---

# YourTJ release notes

Use [the release runbook](../../../docs/operations/releases.md) for the file protocol and
[editorial rules](references/editorial-rules.md) for channel-specific writing.

Read the candidate's manifest and evidence before writing. Compare against each channel's own
baseline, including the live App Store baseline for a TestFlight-to-store promotion. Source paths
are hints: verify platform branches and actual client use of API changes. Inspect incomplete diff
excerpts in the fixed source checkout; do not announce unverified behavior.

For schema-2 mobile candidates, edit the requested channel's entries in `changelog.json` and use
`workflow.py render-structured` to derive the canonical files; preserve other channels' facts.
For Web and legacy candidates, edit only the requested channel's canonical file.
Android and iOS never share a What’s New file.
Preserve mandatory disclosures verbatim. Document unsupported claims or uncertain applicability
for the human reviewer instead of adding plausible filler. Evidence IDs in machine-generated
entries must exist and match the channel. Human wording edits need not mimic the model's wording.

Oryn runs only during preparation. Do not rerun it over a human-edited file implicitly. An explicit
regeneration is a new reviewable edit. Model output, PR prose, and an `approved` field cannot grant
publication authority. Never replace an approved release's text during a retry.

Run `python3 scripts/release/cli.py validate --candidate <id> --json` after edits. This checks
structure, identity, file isolation and disclosure/length constraints; it does not prove semantic
truth or substitute for the final human review. A notes-only request does not authorize creating
PRs, tags, store submissions or production deployments.
