# Reviewed releases

> Doc type: operations runbook
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-01

`Current`: release requests freeze source, platform notes and distribution targets; a live final-head
human review is required by the publisher. `Partial`: distribution also requires configured GitHub
App/tag permissions, environment secrets and successful platform runs. Apple review is external.
The source of truth is [the CLI](../../scripts/release/cli.py), [protocol validator](../../scripts/release/model.py)
and [controller](../../scripts/release/controller.py). See [decision 0054](../decisions/0054-reviewed-release-pipeline.md).

## Entrypoints

Use **Release / Prepare** on branch **main**, or run from a repository checkout:

```sh
python3 scripts/release/cli.py doctor --scope android --json
python3 scripts/release/cli.py plan --scope android --source main --bump patch --json
python3 scripts/release/cli.py prepare --scope mobile --source main --bump patch --ios-destination testflight --apply
python3 scripts/release/cli.py validate --candidate mobile-1.0.15-15 --json
python3 scripts/release/cli.py status --candidate mobile-1.0.15-15 --json
python3 scripts/release/cli.py retry --candidate mobile-1.0.15-15 --channel ios-testflight --dry-run
python3 scripts/release/cli.py promote-ios --release mobile-1.0.15-15 --to app-store --apply
```

IDs above are examples, not a command to publish that version. Query commands are read-only. Mutating
commands default to a dry-run; `--apply` dispatches the corresponding workflow explicitly on main.
JSON output carries `schemaVersion`, `ok`, result fields, and actionable errors. API errors remain
unknown/blocked. Status/validate/retry can read a remote Release PR into a temporary directory when
its candidate is absent locally, without switching the agent's checkout. A local Apple plan needs
`--apple-state` from the authenticated read-only ASC discovery;
Prepare performs that discovery on Actions without requiring keys on the caller's computer.

`scope` is web, android, ios or mobile. Mobile selects Android and iOS. TestFlight is the iOS default;
app-store explicitly includes its store submission. An existing-build promotion creates a fresh
App Store request without allocating another binary identity. The ordinary release entry does not
merge dev: promote intended code using a normal dev → main PR first. The default branch remains dev;
release workflow discovery requires the implementation on dev and execution requires it on main.

## Request and review

Prepare resolves a full main-history `sourceSha`, determines actual per-channel baselines, and reserves
the preparation slot while collecting evidence and opening `codex/release/<candidate-id>` against main.
Only one undecided request per product (web/mobile) is admitted. Human and Apple review do not hold a
runner lock. Server tags remain `vX.Y.Z`; mobile tags remain `mobile-vX.Y.Z`, with a shared increasing
base build number. Native platform version codes retain their existing rules. Tags are created only
after approval/source verification, and consumed build numbers are never recycled.

The Release PR contains only `releases/requests/<id>/`:

| File | Purpose |
| --- | --- |
| manifest.json | Source, version/build, channels, baselines, dependencies and required disclosures |
| evidence.json | Complete changed-file inventory, bounded net-diff excerpts, PR/commit context and Oryn provenance/uncertainties |
| web.zh-CN.md | Forum web changes |
| operators.zh-CN.md | Web deployment/configuration/migration notes |
| android.zh-CN.md | Android changes only |
| ios.zh-Hans.txt | iOS App Store What's New, plain text, at most 4000 characters |
| testflight.en-US.txt | iOS testing instructions, independently reviewed |

Only selected platform files are created. Source SHA controls the application build/tag; the reviewed
PR head controls release metadata. A metadata PR merge is not a new application version. Baselines
are distribution-specific: Android public verified APK release; web successful production deployment;
iOS actual TestFlight distribution or live App Store version. Unknown Apple build/tag mappings block
preparation and require reconciliation; an uploaded or in-review version is not a live-store baseline.

A reviewer edits files, checks evidence/platform attribution, fills server prerequisites where required,
then approves the final head. `serverRequirement` is null or `{sourceSha, reason}`; the required source
must be an ancestor of the confirmed deployed server. Never use main membership as proof of deployment.
Mandatory disclosures are `{id, channels, text}` and must appear verbatim in the corresponding files.

At least one current GitHub user with write/maintain/admin permission must have APPROVED the exact head.
Bot/App reviews and stale reviews are rejected. An outstanding eligible maintainer's change request
blocks publication. A manifest cannot self-declare approval. A Release PR cannot include application,
workflow or publisher edits. `release-authorized` recomputes on reviews and head changes from the trusted
main controller. Publication independently rechecks live reviews, paths, merged content and `ci-required`;
a ruleset bypass or direct metadata push does not bypass that check.

## Oryn drafting

Prepare uses [the pinned Oryn runtime](../../.github/actions/setup-oryn/action.yml)'s independent
`release-notes` task. It shares Oryn's isolated Core execution, provider configuration and bounded
structured output, but has no issue-repair authority, repository tools, signing inputs or GitHub token.
The collector supplies final net diffs relative to each platform's baseline, including removed/moved
paths. Excerpts are bounded; incomplete evidence is explicit. Paths identify candidate applicability,
not semantic proof. Human review resolves uncertain platform behavior and dependencies.

Oryn returns channel-specific entries with eligible evidence IDs. The host renderer validates their
scope and the input digest before writing canonical filenames. Model, prompt version and input digest
are retained. `ORYN_MODEL`/`ORYN_BASE_URL` override defaults; do not assume the default profile is the
active provider. The currently configured deployment uses `oryn/glm-5.3-flash` through the configured
YourTJ endpoint. No private provider or review credentials enter the request/evidence files.

Failed generation leaves explicit drafts for human completion. Rendering preserves human-edited files;
regeneration must be deliberate and reviewed. No model runs after release approval. Static validation
checks shape and evidence references; semantic correctness still requires a person.

## Verification and publication

`Release / Publish` consumes the merged request and rechecks approval, source ancestry, version identity,
baselines and any server dependency. It separately verifies the fixed source: web backend/race/PG,
frontend/browser/i18n, contract and current govulncheck; mobile analysis/package tests and selected
platform signing/build validation. Green metadata-only PR CI is not application verification.

Platform publishers are reusable workflows with main-only guards. They cannot be launched with arbitrary
public workflow-dispatch inputs. Each rechecks approval before side effects. Concurrent execution is
serialized per platform; deploy/config transactions share `instance-mutate-main` or `instance-mutate-dev`
and are not automatically cancelled. Queued concurrency uses GitHub's `queue: max` policy.

- Web: GoReleaser builds archives once with explicit source/version metadata and approved notes. The
  exact Linux binary is packaged into GHCR. Production consumes a digest-qualified image reference,
  waits for health, and compares the running binary digest. A failed deployment records failure even
  when release archives are already downloadable. Image/config rollback does not restore the database.
- Android: signed APK package, architecture, version code, certificate and server-side asset digests
  are checked. Existing assets must match exactly. Public version notes come only from android.zh-CN.md.
  Updating `mobile-latest` is a separate last step; it never overwrites immutable version APKs.
- iOS: publisher queries exact version/build before uploading, preserves an existing processing build,
  and uses separate TestFlight and zh-Hans store text. Existing store metadata/screenshots remain in
  `apps/mobile/store/`, but static `whatsNew` is not a publication fallback. Other in-review versions
  are not automatically withdrawn. An existing-build promotion validates its approved Apple build ID.

Execution records use GitHub Deployments under `release-web`, `release-android`, `release-ios-testflight`
and `release-ios-app-store`. Payloads record source, reviewed content digest/head, controller SHA,
run/build identity and channel details. Read them together with immutable Git/tag contents, actual
assets and Apple state; an editable receipt alone is not proof. Git contains requests and approved
notes; temporary Actions artifacts are not their only durable source.

## Recovery and completion

Use **Release / Recover** on main with the original candidate and an approved channel. It rechecks the
original approval, restores the original build artifact, and cannot add targets. `android-alias` only
refreshes stable download links. iOS queries Apple before restoring an IPA; an existing Apple build can
resume even when the temporary archive has expired. If required original artifacts are unavailable,
stop and prepare a newly reviewed identity rather than rebuilding the same version silently.
An existing production image without its original digest receipt also blocks another push; reconcile
the original identity/receipt or prepare a new release. Registry errors are not absence proofs.

Store submission and availability are separate: submitted/in_review is not live. `status` reports
recorded Apple results explicitly as observations, not an always-current store status. Query ASC for
current external state. No permanent monitor is installed by this pipeline.

A notes-only store correction requires another reviewed existing-build promotion request; a binary
change requires a new release. Production rollback requires an explicitly chosen compatible image and
configuration and database compatibility assessment. It is not an arbitrary-image option on Recover,
and never automatically restores production data.

## GitHub configuration

Use the existing Oryn GitHub App for request PR creation, narrowed to Contents/Pull requests write so
PR CI triggers normally. The model child receives no installation token. `RELEASE_TOKEN` is limited to
the trusted main tag-reservation step; its identity must satisfy the existing mobile-tag ruleset. Keep
protected tag updates/deletion forbidden. Signing/Apple keys stay in `mobile-release`; SSH/config keys
stay in production/dev environments. PR checks never receive those secrets.

The publisher's human gate is mandatory regardless of UI protection. Configure main to require PRs,
`ci-required` and `release-authorized`, and dev to require `ci-required`. The compatibility checks
`ci-backend`, `ci-frontend`, `ci-contract` remain available while existing dev rules are in use. Change
required names only after the new checks have appeared successfully. Repository ruleset changes are
an operator action after the corresponding controller is available on the protected branches.

The retired Release / mobile and Release / main entries refuse new releases. Do not rerun historical
workflow versions as a substitute for the new approval process. Historical successful releases remain
valid baselines; continuing an unapproved historical operation requires a reviewed request and explicit
identity reconciliation. Runtime upgrades and source changes use normal reviewed code PRs.

External coding agents can start with [yourtj-release](../../.agents/skills/yourtj-release/SKILL.md),
[yourtj-release-notes](../../.agents/skills/yourtj-release-notes/SKILL.md) or
[yourtj-release-recovery](../../.agents/skills/yourtj-release-recovery/SKILL.md).
