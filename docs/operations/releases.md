# RC and confirmed releases

> Doc type: operations reference
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-11-03

`Current`: new releases build a production RC, then an explicit operator Confirm seals the exact
source, final notes, targets and original artifacts. Confirm promotes those artifacts without another
compile, source test matrix or dev QA gate. Normal development CI runs independently. Apple review,
configured credentials and physical-device acceptance remain separate outcomes.

## Entrypoints

Use **Release / RC** and **Release / Confirm** on **main**, or the portable CLI:

```sh
python3 scripts/release/cli.py doctor --scope mobile --json
python3 scripts/release/cli.py rc --scope mobile --ios-destination app-store --apply
python3 scripts/release/cli.py status --candidate mobile-X.Y.Z-N --json
python3 scripts/release/cli.py validate --candidate mobile-X.Y.Z-N --json
python3 scripts/release/cli.py confirm --candidate mobile-X.Y.Z-N --apply
```

Commands with side effects default to dry-run; `--apply` executes existing user authorization.
`mobile` means Android and iOS; `web`, `android` and `ios` select only that scope. The RC entry defaults
to both mobile platforms and App Store (including production TestFlight); explicitly choose TestFlight
for beta-only distribution. Android-only RCs must use `--ios-destination testflight`.

Source must already be in main history. `main` resolves once to a full application SHA; preparing an RC
never implicitly merges dev. Source promotion uses an ordinary reviewed code PR. Dev delivery is
automatic and has [its own model](release-cycles.md), not a prerelease command.

`prepare` is internal to RC. `prerelease` refuses new manual preview cycles. `publish` is an initial
publication recovery entry, not a way to bypass Confirm or build a new schema-4 final release directly.
Legacy schema-1/2/3 requests retain their existing authorization and original-artifact recovery rules.

## Request and review

RC preparation queries actual per-channel baselines, reserves identity under a short shared lock,
collects source evidence and opens `codex/release/<id>` against main. Only `releases/requests/<id>/`
may change. Editable drafts do not delay building the RC. Signing executes trusted-main controller
code with the selected source in a separate checkout; release data cannot edit publisher workflows.

| File | Purpose |
|---|---|
| manifest.json | Source, final version, native counter, display ID, iOS number, channels and selected RC run |
| changelog.json | Structured mobile facts and optional display ID; source for separate channel text |
| evidence.json | Baseline-relative source and platform evidence |
| android.zh-CN.md | Android What's New |
| ios.zh-Hans.txt | App Store plain-text What's New |
| testflight.en-US.txt | Separate English testing instructions |
| web.zh-CN.md | Web release notes |
| operators.zh-CN.md | Server prerequisites and operator actions |

Review platform applicability, mandatory disclosures and server dependencies. Draft markers or empty
channel entries block confirmation. iOS/TestFlight text must not mention Android/安卓/APK; Android text
must not mention iOS/iPhone/iPad/iPadOS/App Store/TestFlight/苹果. Structured titles and summaries enforce
the same rule for every listed platform. Frozen historical exemptions bind both exact bytes and exact
candidate ID; they cannot excuse copied wording in a new request. Other semantic correctness requires
review of the evidence.

New schema-4 requests require **Release / Confirm** after the RC and final notes are ready. It records
an immutable Actions artifact sealing candidate, source, version, counters, selected RC, targets,
current PR head, complete content digest, operator, attempt and trusted controller SHA. A deployment
pointer locates that seal but cannot authorize a release by itself. A changed head or note digest
requires a new Confirm. The operator must have repository write/maintain/admin permission. An agent
may execute it within an explicit user's release authorization; it never fabricates a GitHub APPROVED
review. Legacy requests still require their eligible current-head human GitHub review.

Confirm rechecks successful metadata-only `ci-required`, validates the original RC and exact PR paths,
then merges only that sealed head. Publication independently validates the seal, source ancestry and
merged bytes. A seal/merge failure is recoverable without rebuilding. The `release-authorized` check
is derived from this proof; manually setting a check or changing a label is insufficient.

## Production RC

RC uses final production endpoints, app identity, `preview=false`, final marketing version and final
native build numbers. It distributes candidate APKs under an RC tag, uploads a normal Apple build only
to the configured internal QA group, and preserves Web archives/images without production deployment.
It does not publish stable aliases or submit the App Store version yet.

Each trusted-main RC records source, identity, controller, complete platform set, original archive
references and asset digests in immutable Actions artifacts. The collector seals only successful
platform results, then attaches `rcRun` to the editable request. A complete seal remains reusable if a
later metadata attachment step fails; an incomplete or failed platform has no reusable complete seal.
Retry only failed RC jobs in the original run. Existing artifacts are restored, Apple is queried before
any upload and already uploaded builds are reused. Missing required originals fail rather than silently
rebuilding an already distributed identity. Notes may change without rebuilding; binary/source/config
changes need a fresh candidate and number. A replacement RC may explicitly retain an unpublished
marketing version with `--version X.Y.Z` after closing its undecided request.

If only attachment failed, download the exact `rc-record-<attempt>` artifact, validate with
`rc.read_record(GitHub(), RUN_ID, identity)`, then run `GITHUB_RUN_ID=RUN_ID python3 scripts/release/rc.py
attach` from a full checkout. This updates only the matching request's `rcRun`; final Confirm is still
required. Do not substitute a mutable deployment receipt for the sealed original record.

### Build identities

`displayBuild` is `YYYYMMDDNNNN`, in UTC+8, resetting daily at 0001. Dev bundles and production RCs share
one atomic allocation lock and annotated `build-id-...` reservations. Failed reservations are consumed;
9999 fails rather than wrapping. A retry reuses its original identity. Source and environment bind each
reservation. A Confirm does not consume another number.

Mobile also retains a global increasing internal counter (`mobile-build-N`) for Android ABI version
codes and release-catalog ordering. Twelve-digit IDs do not fit Android's native versionCode.
iOS uses the display ID as native `iosBuildNumber`; the compiled internal counter separately keeps
in-app update/history comparisons consistent across platforms. App history and release pages show the
date ID in parentheses; old history without it shows its recorded counter. Stable tags remain
`vX.Y.Z` and `mobile-vX.Y.Z`, created only on final publication. Dev and RC tags do not consume a stable
marketing version. Actual Apple upload validation, not the local metadata validator, establishes
acceptance of a native number.

Dev bundles and RCs hold one shared packaging lock from allocation through native upload, so Apple
cannot receive a later reserved number before an earlier in-flight bundle. This does not block dev
Web. Confirm's existing-build metadata publisher uses a separate queue: it neither allocates a new
number nor uploads another binary, and does not wait behind a new dev compile.

## Oryn drafting

RC preparation defaults to deterministic editable drafts. Set `draft_notes=true` (CLI `--draft-notes`)
to opt into [the pinned Oryn runtime](../../.github/actions/setup-oryn/action.yml)'s independent
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

When enabled, Prepare runs `workflow.py draft-notes`, which retries Oryn (`ORYN_NOTES_ATTEMPTS`, default 3, with
backoff inside `ORYN_NOTES_BUDGET_SECONDS`, default 2400). Each attempt is rendered into a scratch copy
first: a provider error, timeout, malformed JSON, schema violation, foreign channel or unknown evidence
is retried without touching the request. An attempt is complete when every channel's platform note
renders from entries. Otherwise the most complete valid draft is kept, still marked `[DRAFT:` for the
missing channels. The release PR opens with a warning naming the missing channels and the last error.
Failed generation leaves explicit drafts for human completion; it never blocks the PR from opening.
A candidate cannot publish while any channel lacks notes: PR validation and every publisher reject a
`[DRAFT:` note, and a schema-2 mobile channel with no reviewed changelog entry. Rendering preserves
human-edited files; regeneration must be deliberate and reviewed. No model runs after confirmation. Static validation
checks shape, evidence references, and store-channel isolation: iOS What's New and TestFlight notes must not name Android, and Android notes must not name Apple's mobile platforms. A structured changelog entry is rejected when its title or summary names the opposite platform for a platform it lists. Other semantic correctness still requires a person.

New mobile candidates use schema 4 (structured notes plus `displayBuild`, `iosBuildNumber` and mandatory selected `rcRun` before confirmation) and retain the exact reviewed changelog and evidence bytes in the
candidate digest. Android, App Store, and TestFlight notes are rendered from that structure; a
publisher rejects drift between the structured facts and platform text. Legacy schema-1 candidates
remain valid for original-artifact recovery. Existing schema-2/3 requests remain unchanged.
TestFlight notes have stable IDs and evidence references tied to the iOS TestFlight channel.

The collector links each changed file to the feature pull requests that changed it within that
channel's range. Feature PRs are read from merged history, since main's first-parent history only
shows dev promotions; promotions from dev or main are ignored. The links stay in `evidence.json`,
outside the model request. Commit references are PR candidates only: an explicit PR-endpoint 404
omits that reference from PR metadata and entry identity, retains the full source evidence, and records
an uncertainty. Authentication, rate-limit and other API errors still block preparation.
When Oryn's draft is converted, each entry takes its kind from the linked
PRs' conventional title type or branch prefix (`feat` new, `fix` fixed, security wording or type
security, otherwise improved). Its ID is `pr-<number>`, with `-ios` or `-beta` for iOS channels.
A breaking `type!:` PR places it under required changes. Title and summary are split at a leading
`标题：` or the first short sentence. Manifest disclosures become required entries with their
exact text. Optional mobile `kind`, `title`, `summary` and `required` fields from Oryn take precedence;
invalid supplied fields fail the attempt and trigger retry. Only absent fields use inference.
Android/App Store text is at most 400 characters, TestFlight text at most 500, titles 1–100,
summaries 1–400, and `required` must be boolean. Web/operators retain their three-field entry format.
Entries without a linked PR keep a text-hash `oryn-` ID, which the reviewer replaces
with a concise semantic slug. The reviewer still checks wording, kind and grouping before confirmation.
Android and App Store notes use the app's sections: required, security, new, improved and fixed,
omitting empty sections. Android uses Markdown headings and bullets; App Store uses plain-text
headings, blank lines and `•` bullets. A summary that only restates its title is rendered once.
New mobile changelogs set `notesFormatVersion: 2`; the controller owns this field, not Oryn.
An absent field or value `1` retains the original flat App Store text byte-for-byte for existing
candidates and recovery. To upgrade an unapproved candidate, explicitly set the field to `2` and run
`python3 scripts/release/workflow.py render-structured --candidate <id> --replace-structured`.
Review the resulting diff and confirm the new head; this changes the candidate
digest. Never upgrade an already approved candidate during recovery. Android and TestFlight formats
are unchanged, and section labels and bullets count towards the App Store's 4000-character limit.
The public mobile catalog contains structured entries, not this store-copy formatting field; the
app renders formal release entries with its native update-history and prompt views. iOS public
history always selects App Store entries, with no distribution-channel picker; TestFlight-only
releases and testing instructions are not substituted for missing public history. The three
presentation surfaces are separate: the app consumes structured facts, App Store receives
`ios.zh-Hans.txt`, and TestFlight receives `testflight.en-US.txt` for Apple's testing UI.

The trusted-main catalog publisher derives `releases.json` from merged reviewed candidates and
successful channel deployment receipts. It validates each receipt against the candidate source SHA
and content digest, keeps platform channels separate, and publishes to the `mobile-notes` GitHub
Release. Android requires all three APK SHA-256 values; TestFlight requires `APPROVED`; App Store
submission states do not count as public availability. A matching exact China storefront lookup may
add a public App Store build, and a prior catalog's covered public builds remain covered after the
store advances. Missing structured channel history is omitted and expressed in per-channel coverage.
The status Worker's `/mobile/releases.json` route proxies this fixed asset with bounded reads, a short
timeout and conditional ETags; it sets client cache headers but writes no edge cache or snapshot storage.
It is display-only and contains no evidence.
Normal release publication refreshes the catalog; operators may run the trusted-main manual refresh
workflow after reconciling a receipt or catalog failure.
For multiple successful App Store requests for the same binary, the most recently merged reviewed
request supplies its store copy. Supersession requires the same source/tag/version/build and a promotion
of the original candidate and Apple build. Publisher retries cannot restore older copy, and another
platform's notes are preserved. TestFlight-only requests are complete with reviewed TestFlight entries;
they do not require artificial Android or App Store highlights.

## Verification and publication

Confirm merges the sealed request and triggers **Release / Promote original artifacts**. It checks
authorization, baselines and declared server prerequisites. There is no release-time source CI or
native test/compile stage. Android restores original signed APKs and verifies their digests before
stable publication; `mobile-latest` is a separate alias update. iOS selects the exact uploaded Apple
build, applies the separate channel notes and submits the selected App Store target. Web deploys the
original digest-qualified RC image and verifies health and running binary identity.

Platform publishers are main-only reusable workflows. They independently verify source, candidate,
note digests and original artifacts. Selected skipped/failed channels fail the overall result.
Instance mutations serialize with config applies and never interrupt an active transaction. An image
rollback does not restore the database. Execution records use `release-web`, `release-android`,
`release-ios-testflight` and `release-ios-app-store` Deployments and bind source, content, controller,
run and channel details. Receipts are observations, not standalone authority.

## Recovery and completion

New releases have no dev cycle to complete or freeze to lift. Use **Release / Recover** with the original
candidate and selected channel after a publisher starts. It verifies the original authorization and
reuses signed artifacts or an exact Apple build. It cannot widen targets. `android-alias` only refreshes
stable links. If nothing started after an authorized merge, `publish --candidate ID --apply` resumes
initial promotion. Missing required original artifacts require a new identity, never a silent rebuild.
Legacy requests keep their original recovery rules, including the strictly bounded Web build exception
when every original archive upload was skipped and no release/image was published.

`promote-ios --release ID --apply` enters RC preparation for a published schema-4 TestFlight request,
prepares App Store notes for its exact original build and selects its original RC without compiling.
Confirm authorizes that metadata snapshot. It does not assign a new date or native counter. Legacy
metadata corrections use their already prepared requests and original recovery paths.

App Store submission, processing, beta installability and public availability are separate facts.
Query ASC for current state; submitted/in-review is not live. Existing conflicting review queues are
reported, never automatically withdrawn. A notes mismatch on a submitted version requires resolving
Apple metadata editability before recovery. A binary change requires a new RC. Production rollback is
an explicit compatible image/config choice with database assessment, never automatic data restoration.

## GitHub configuration

The existing Oryn App creates request PRs with narrowed Contents/Pull requests permissions so PR checks
run. Models receive no installation token. `RELEASE_TOKEN` is used only for protected final-tag creation
and merging the exact confirmed release-data head; its existing authorized identity must satisfy those
rules. Do not broaden it or export a local credential to runners. Signing/Apple keys stay in
`mobile-release`; SSH/config keys stay in production/dev environments. Untrusted PR checks receive none.

Main requires code PRs and normal CI. For release-data PRs `ci-required` is metadata-only and
`release-authorized` reflects Confirm (or a legacy human review). A bypass merge cannot bypass the
publisher's independent proof checks. Review changes to controller, classifiers and workflows as code;
release-data PRs cannot change them. Historical retired workflows are not alternate publication paths.

See [yourtj-release](../../.agents/skills/yourtj-release/SKILL.md),
[yourtj-release-notes](../../.agents/skills/yourtj-release-notes/SKILL.md) and
[yourtj-release-recovery](../../.agents/skills/yourtj-release-recovery/SKILL.md).
