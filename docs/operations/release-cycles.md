# Release cycles: dev freeze, previews and hotfixes

> Doc type: operations runbook
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-19

`Current`: the trusted-main controller binds a release owner, full source SHA, target channels,
tracking issue, freeze deadline and QA conclusions. A changed source or repeated dev deployment
invalidates QA. Production publication additionally retains the
[reviewed Release PR protocol](releases.md).
`Partial`: enforcement of the merge check depends on repository rules; signed distribution depends
on configured credentials/TestFlight groups and device acceptance. Workflow code alone cannot
prevent a direct push or a merge that bypasses branch rules.

The implementation is [cycle.py](../../scripts/release/cycle.py),
[cycle_control.py](../../scripts/release/cycle_control.py),
[Release / Prerelease](../../.github/workflows/release-prerelease.yml) and
[Release / Control](../../.github/workflows/release-control.yml).
[Issue #1240](https://github.com/YourTongji/YourTJ-Hub/issues/1240) owns the acceptance criteria;
[0099](../decisions/0099-shared-dev-release-cycles.md) records the tradeoffs.

## Ownership and environments

The human who starts the prerelease owns production deployment, smoke, channel failures and any
later store-review follow-up. The ordinary cycle ends automatically when its formal Release request
completes; Apple approval and public availability do not hold dev frozen. A rerun preserves the
initiating identity. Only a human with write/maintain/admin permission can initiate a cycle or accept ownership. `offer` names a
successor; `accept` must be dispatched by that successor. Asking another contributor to fix a bug
does not transfer ownership. Bots cannot initiate or accept ownership.

A repository **admin** may use **reassign** with `assignee` and a reason to transfer ownership
immediately, including after freeze expiry or during a failure. The old owner need not respond or
offer a handoff. The controller checks both the initiating and rerunning actor's live `admin`
permission, checks that the successor is a human maintainer, and preserves previous/new owner,
admin actor, time and reason in the sealed ownership history. Write/maintain permission alone is
insufficient. Reassignment neither extends the deadline nor supplies QA or makes a failed channel
successful. The successor takes responsibility for the remaining work, including an aborted hotfix's
backflow. Ended closed/cancelled cycles cannot be reassigned.

There is one dev instance and at most one cycle occupying it. There are no stabilization branches
or per-release sites. Developers keep working on feature/fix branches while ordinary dev merges
are frozen. The freeze defaults to 24 hours. Expiry blocks further release actions; it does not
automatically unlock merges, bless QA or publish. The owner explicitly extends (with a reason and
deadline), cancels before publication, or aborts after reconciling publication facts. Each extension
is limited to seven days. An unavailable owner can be replaced by an admin without waiting for expiry.

| Build | API / OIDC | Distribution |
| --- | --- | --- |
| Dev prerelease | `https://dev.yourtj.de` / `/api/oauth` | Existing dev site, signed Android GitHub prerelease, dedicated internal TestFlight QA group |
| Production configuration rehearsal | `https://f.yourtj.de` / `/api/oauth` | Signed Android GitHub prerelease and the internal QA group; no stable download/catalog updates |
| Formal production | `https://f.yourtj.de` / `/api/oauth` | Reviewed web/mobile publishers and their approved public channels |

All builds explicitly select the cycle's SHA. Rehearsal and formal builds use production configuration;
dev IPAs cannot be promoted through the formal existing-build entry. Rehearsal is a separate test
distribution, not a promise of identical binary bytes after rebuilding. The protocol binds source
and environment; automated artifact/signature checks and formal recovery still protect published
bytes. Operators do not calculate artifact hashes or allocate build numbers manually.

Android dev previews install beside the stable app as `tj.yourtj.forum_app.dev`, named
`YourTJ Dev` with a badged DEV icon. Production rehearsals and iOS previews use the existing app IDs
and replace an installed production app. Previews display `DEV <short SHA>` or
`PROD TEST <short SHA>`. Dev uses separate secure-session keys, preferences,
encrypted local database paths and encryption keys; it does not migrate production local data.
Offline/media scopes retain their site/account boundaries. Preview update checks do not query or
install stable releases. Scheduler links follow the configured API origin. Dev native push is
disabled; production configuration rehearsals use dedicated test accounts and can exercise
production-only integrations deliberately.

Preview apps check the public `preview-status` GitHub prerelease index at startup, on resume and
every five minutes. A changed source, ended/paused candidate, removed platform or expired freeze
shows an old-candidate notice. Rebuilds of the same source remain eligible; package bytes/build
numbers do not determine this notice. Missing/malformed/offline or more than 26-hour-old status is
displayed as **unverified**, never as current. The notice does not block the app or switch its server.
Its link opens the current candidate, owner, checklist/run links and production download instructions.
Android download notes and TestFlight testing notes include the same link.

The controller refreshes the public index after transitions and preview runs, and daily with state
renewal. This is an informational projection, not QA or publication authority; it contains no QA
free text, private channel details or Apple responses. A failed refresh is visible as a failed
workflow and can be repaired with **Control → refresh**. Existing TestFlight/download history is
not deleted. To leave testing, use the normal App Store/Android entry; an Android downgrade may
require uninstalling the preview, so preserve local-only data first. No automatic production install
is attempted. Builds predating the notice require testers to follow the release page manually.

Control saves and seals the transition before a separate `project` job re-runs freeze checks and
the public notice. If that job fails, use **Re-run failed jobs** to retry these projections without
reapplying the transition. Both jobs remain under the dev instance lock. Do not rerun all jobs for
an already sealed `start`, handoff or cancellation; **Control → refresh** also repairs projections
from the latest state. An upload failure before sealing retains the same-dispatch recovery path.

## Start and build

Select **main** for every release workflow. The workflows must exist on the repository default
branch for discovery and on main for execution. Candidate scripts cannot replace the main controller.
An eligible human must have reviewed the candidate's exact PR head before it gets signing access;
the workflow runs backend/PostgreSQL, frontend and contract checks first, plus mobile checks and
the selected native platform builds when mobile artifacts are selected.
Normal branch review/CI remain mandatory.

For an explicitly authorized emergency **dev preview**, a repository admin may supply
`source_review_bypass_reason` (CLI: `--source-review-bypass-reason`). The sealed cycle records
source SHA, cycle/run ID, initiating and rerunning actors, time and reason. Identity allocation and
signing recheck that exact binding and both actors' live admin permission. A different SHA, cycle
or run cannot reuse it; a new run requires a new explicit reason. Rerunning the same authorized
run remains possible. Ordinary previews still require the exact human source review. This exception
does not authorize production-configuration rehearsal, formal Release, QA, hotfix backflow or
branch merges; their existing gates remain in force.

Start **Release / Prerelease** with the full current dev SHA, an open tracking issue, production
channels and purpose `dev`. Omit `cycle` to create a cycle; its ID is the initiating Actions run ID.
Starting another ordinary cycle while dev is occupied fails. Each run distributes only the
artifacts selected for that run.

`channels` defines the **permitted production scope**; `artifacts` selects **this preview run**.
One formal Release request may select a subset of that scope. Its successful completion ends the
ordinary cycle, even if other tested platforms have not been formally released. Later platforms
start a new cycle with fresh QA; they do not keep the first cycle open.
Use a comma-separated subset of `web,android,ios` for dev, or `android,ios` for production rehearsal.
Empty defaults to all artifacts supported by that purpose. Web-only runs skip mobile identity
allocation, signing and native builds; Android-only runs do not require the iOS build or TestFlight
group. An initial dev preview must include `web` to establish the shared site's SHA. A later mobile-only
run may reuse a successful Web deployment for the same candidate, avoiding another database reset.
For example, use `web,android` for a Web/Android release, `web` for an API-only release with existing-client
regression, or build `web` first and add `ios` separately.

Each artifact retains its latest distribution run for this source. Successful Web/Android work can
be reused when another platform fails; QA verifies each required artifact's validation/distribution
jobs, rather than requiring every unrelated job in its original run to succeed. A failed latest
attempt cannot fall back to an older success. `qa` requires Web and every mobile platform in the
production scope; `production-qa` requires those mobile platforms on production configuration.
Selecting fewer artifacts for one run cannot bypass this coverage. Additional selected artifacts
must also succeed before QA. To discard an optional failed selection before formal publication,
explicitly `advance` at the same SHA with a reason and rebuild the reduced selection; this clears QA
and all prior preview bindings. Any changed SHA also clears all bindings.

The CLI has the same inputs. Commands below are dry runs; replace the example SHA/IDs with actual
values. Add `--apply` only for an explicitly authorized workflow dispatch:

```sh
python3 scripts/release/cli.py prerelease --source FULL_DEV_SHA --issue RELEASE_ISSUE --channels web,android --artifacts web,android --dry-run
python3 scripts/release/cli.py cycle --json
python3 scripts/release/cli.py prerelease --cycle CYCLE_ID --purpose dev --artifacts android --dry-run
python3 scripts/release/cli.py control --cycle CYCLE_ID --action qa --note 'Device checklist result and unresolved limits' --dry-run
python3 scripts/release/cli.py prerelease --cycle CYCLE_ID --purpose production --artifacts android --dry-run
python3 scripts/release/cli.py control --cycle CYCLE_ID --action production-qa --note 'Production configuration differences checked' --dry-run
python3 scripts/release/cli.py control --cycle CYCLE_ID --action reassign --assignee NEW_OWNER --note 'Admin takeover: current owner unavailable' --dry-run
python3 scripts/release/cli.py control --cycle CYCLE_ID --action abort --note 'Stop publication; channel follow-up recorded in release issue' --dry-run
```

During a freeze, automatic dev deployments pause. The prerelease deploy reuses the existing dev
deployment transaction and instance lock, with explicit source and cycle checks before mutation.
The binary embeds the full source commit in buildinfo; verify it in the admin server-version view
alongside the Actions summary. Deployment repeats the existing consistent production DB snapshot
sync, so **tell testers that dev data was reset and recreate test fixtures before restarting QA**.
Every new dev preview run revokes both QA conclusions, even when the SHA did not change; only
selecting Web actually repeats the deployment/database reset.
Once QA is recorded, rerunning just the old deploy job is rejected; start a new dev prerelease run
so invalidation occurs before the database snapshot is reset again.

Android preview assets are attached only to `preview-*` GitHub prereleases with `latest=false`.
They never update `mobile-latest`, the formal mobile catalog or stable update notifications.
TestFlight previews use a dedicated **internal** group distinct from the existing production group.
Before uploading, the publisher rejects any other group with automatic or unknown all-build access.
It does not change groups, production reviewer credentials or App Store submissions.

The iOS result artifact records upload/processing/assignment/Apple availability separately.
`available_for_testing` and a successful workflow mean the channels are ready to test, not that a
device has passed QA. Any failed/skipped required channel blocks `qa`. External testers are not
supported by this preview entry: internal testing requires invited App Store Connect users.

## Freeze, fixes and QA

The owner admits a blocker with **Release / Control → admit**, supplying the dev PR number and
reason. Admission binds the current PR head; another push revokes it. Admission does not replace
review or `ci-required`. Labels alone confer no permission. The `release-freeze` check is the
GitHub Actions job of that name in **Release / Freeze gate**: it runs trusted-main
`cycle_control.py check-pr` on PR head/label changes and merge-group requests, and its exit code is
the verdict. Branch rulesets accept only job check runs from GitHub Actions, so the controller never
writes the check through the Checks API. When state changes, Control and dev pushes re-run the latest
completed `release-freeze` run of every open dev PR head and outstanding merge-group SHA, including
previously green groups; a queued run reads the new state itself, and a head without any run is
logged and needs a new PR event (push or reopen). A re-run reuses the workflow file of its original
run. Starting a cycle requires an empty dev merge queue. Groups created during a freeze fail the
check; admitted fixes use reviewed PR merges.

After a blocker merges, select **advance** with the new full current dev SHA and a reason, then run
the dev prerelease again. Old QA cannot authorize the changed source. Before recording `qa`, the
owner checks that the dev site and selected installed apps show the expected source, every required
artifact succeeded and dev has not moved. `production-qa` similarly requires successful production rehearsals
after dev QA. Both actions require a nonempty human summary; software cannot prove that someone
performed a physical-device check.

Use this checklist in the tracking issue; record failures, unsupported cases and retest results:

- [ ] Scope, owner, full SHA, freeze start/deadline, blocker list and rollback target are explicit.
- [ ] Dev server/Web and every selected TestFlight/Android artifact match the SHA; omitted platforms
  and older-client compatibility coverage are explicitly recorded.
- [ ] Password/campus login, logout, expired session and relogin work; tokens stay within their site.
- [ ] Feed/topic browsing, publishing/editing/replies, uploads, likes/bookmarks and permissions work.
- [ ] Search, notifications/chat, course/scheduler and campus data cover the changed paths and core smoke.
- [ ] Dev/production app replacement does not expose the other site's session, drafts or offline data;
  scheduler/site links, widgets and the environment badge point to the intended environment.
- [ ] One older supported production client still works with the changed API and migration.
- [ ] Fix verification, affected-area regression and a final core smoke are complete on the final SHA.
- [ ] Production configuration rehearsal covers API/OIDC, native callbacks, signing/install/upgrade,
  site links and production-only login/push behavior using authorized test accounts.
- [ ] Dev restrictions are explicit: snapshots reset fixtures; native push and production-only OAuth
  grants are not validated by a dev pass. Record the separate production checks or their blockers.

An absent item is not a pass. Update the existing issue with the testing context; no separate manual
artifact digest register is required.

## Enter main, publish and finish

Promote the tested source through a reviewed PR to main using a merge commit that preserves the
source SHA. A clean merge is supported; fast-forward-only is not required. The controller rejects
squash/rebase replacement of the candidate identity and conflict/manual edits to the integration
merge tree. Resolve such changes on dev, advance the cycle, rebuild and repeat QA.

Use **Release / Prepare** for this formal request's channels. Its default `source=main` selects the
active cycle SHA, not the moving main head; an explicit full SHA also needs the cycle's approval.
Main may advance without changing the selected application source. Release-data PR approval,
source CI, immutable formal tags and independent publisher checks still apply. Every publisher
rechecks cycle state after building and before publication. A partial channel failure stays a failure;
use the existing [recovery procedure](releases.md#recovery-and-completion) for its original signed artifacts.
The prepared PR links the tracking issue and identifies the cycle and owner at preparation; GitHub
shows the cross-reference in that issue. Keep test results and channel follow-up there.

After **Release / Publish** succeeds, **Release / Control** automatically finishes that request's
cycle and restores ordinary dev integration. There is no separate manual unfreeze/close step.
The completion boundary is the formal request, not all platforms selected during preview: a
successful Web request ends its ordinary cycle even if Android/iOS were also tested. Web must have
passed production deployment/health checks, Android its publisher, and selected Apple channels their
publication/submission steps. TestFlight Beta App Review and App Store approval or actual availability
are not completion prerequisites. Submission remains submission in channel receipts and the public
catalog; ending the cycle does not label an unapproved build as downloadable.

If mobile requires a new server API, publish Web first, verify production Web and older-client
compatibility, then start a new mobile cycle. Select the current dev SHA (the same SHA is allowed if
dev has not moved), perform fresh dev QA and the production-configuration rehearsal against the
updated server, then prepare the mobile request. Web publication requires dev QA; mobile publication
also requires `production-qa`. A web-only cycle does not require new mobile builds, but still needs
older-client compatibility coverage. The owner records production smoke and subsequent Apple
rejections or release results in the tracking issue; these follow-ups do not prolong the ordinary freeze.
Changing code for a rejected review needs a new candidate and new QA.

Automatic completion runs after the formal workflow has completed, outside its publication lock.
The main controller verifies the successful Publish/Recover run, its immutable cycle/request binding,
reviewed source/metadata and latest successful receipts for that request's channels. A failed,
cancelled or skipped publisher cannot end the cycle. Recovery of one channel only finishes the cycle
once all channels of that same request succeeded; unrelated future requests are not prerequisites.
Old callbacks cannot unlock another cycle or revive an aborted/paused candidate. If the completion
callback fails, rerun **Control → complete** with the successful formal run ID; it repeats only
completion, never builds, uploads or deploys again:

```sh
python3 scripts/release/cli.py control --action complete --release-run RELEASE_RUN_ID --note 'Retry automatic completion' --dry-run
```

Use **cancel** only before any formal publication has started. It invalidates QA and releases dev.
Once publication has started, use recovery while retaining the candidate, or **abort** to stop it.
Abort waits in the same `release-production` queue as formal publishers, then records each selected
channel's latest receipt, actual available source/tag and relevant Apple state. An external API or
unknown baseline is recorded as `unknown`; it is not silently called failed, rolled back or successful.
The owner must record follow-up in the tracking issue. Abort revokes both QA conclusions and blocker
admissions, releases this cycle's dev freeze and leaves its outcomes and owner on record. It works
after freeze expiry while the cycle is still open. The aborted candidate
cannot publish/recover using old QA or historical-release compatibility; use a newly tested candidate
for further publication. Cancel/abort do not roll back a successful channel, withdraw Apple review or
expire distributed previews. An already submitted Apple build may still become available and remains
the owner's follow-up obligation.

The same before-publication restriction applies to advancing/redeploying a dev candidate;
a code defect discovered after Web publication needs a new hotfix, not replacement of that source.
A cancelled preview may still exist in TestFlight/download history and
must not be treated as the current candidate. Starting a fresh preview consumes new automated build
numbers; do not repoint tags or overwrite signed assets.

## Hotfix

Find the **actually distributed source** for the affected channel using the release receipts and
authenticated Apple discovery; the newest main commit or tag is not sufficient. Create a short-lived
`hotfix/<topic>` from that SHA. Keep it to the necessary production fix and review its diff against
that base. A hotfix is not a release stabilization branch.

Use **Control → hotfix** with the exact hotfix source, actual base SHA, tracking issue, targets and
incident reason. The controller waits for in-flight formal publication, then verifies ancestry and
every selected channel's live baseline before changing ownership. All selected channels must have
the same nonempty actual base SHA; split different baselines into separate hotfixes. It
pauses an occupying ordinary candidate, revokes that candidate's QA, and gives the hotfix exclusive
dev and publication ownership. The hotfix follows the same preview, required regression, main PR,
production rehearsal and Release PR approval path. Production publishes the hotfix source rather
than unrelated main changes. Before publication, `advance` can select a reviewed revision on the
hotfix line that still descends from its actual base; it need not equal dev.

Successful formal publication moves a hotfix to `backflow`; it does not wait for Apple approval.
Dev remains reserved for the required hotfix return. After production smoke, admit and merge the
hotfix PR back to dev preserving ancestry, complete CI and regression, then use **backflow**
with the merged dev SHA and test summary. The controller verifies ancestry/current dev/successful CI.
It records the required hotfix ancestor for subsequent production releases. If an ordinary candidate
was paused, it resumes at the merged dev SHA with no QA and a new 24-hour deadline; rerun dev preview,
regression and production rehearsal. Without a paused candidate, dev is released. Backflow also
closes the hotfix cycle; its owner follows up any store decision in the issue. Ordinary publication
cannot bypass an active hotfix, and subsequent ordinary candidates must include every completed hotfix. A later
channel-specific hotfix starts from that channel's actual baseline; backflow restores all fixes to dev.
Cancelling a paused ordinary candidate prevents it from being resumed by backflow.
An aborted hotfix with any started publication, observed availability/submission or unknown outcome
keeps a pending backflow obligation. Dev integration is released so the fix can be merged normally,
but new cycles and ordinary publication are blocked until its owner completes **backflow** with
the merged, reviewed and CI-passing dev SHA. A paused ordinary cycle then resumes with fresh QA and
a new deadline. The hotfix remains `aborted`, preserving the failure record. An aborted hotfix with
no publication side effects resumes its paused parent immediately, also without QA. An admin can
reassign an aborted hotfix so an unavailable owner cannot strand this obligation.
Formal publication runs share one `release-production` queue across channels and candidates;
hotfix publication waits for in-flight ordinary writes to finish or fail rather than interleaving them.

## Failure, rollback and state recovery

- A source check or distribution failure leaves the freeze and ownership intact. Read the channel
  result; repeat selected artifacts for the same cycle after fixing the cause, cancel before
  publication, or abort after reconciling channel outcomes.
  Whole-run retries allocate another build number; original assets/tags are not reused or overwritten.
- A processing timeout is not Apple rejection. Check the recorded build and upload state. Retrying
  only a failed iOS publisher uses its existing version/build and reconciles the upload before adding
  the internal group. An uncertain upload blocks another upload.
- A failed formal channel uses **Release / Recover** and the original artifact. Never rebuild a
  consumed formal version because a different channel failed. Store submission and live availability
  are separate facts.
- Historical recovery is restricted to an already reserved formal release whose original reviewed
  release-data PR merge predates the cycle controller (`scripts/release/cycle.py` is absent from that
  immutable merge tree). The original tag/review binding, scope and build number are reverified.
  Recovery does not require inventing a historical owner or QA record. Existing TestFlight-to-store
  promotion keeps the original Apple build and still requires a newly reviewed promotion PR.
  A missing cycle alone is not an exemption: newer release requests, new unreserved versions,
  aborted/cancelled cycles and changed sources cannot use this path. Active/pending hotfixes block
  it, and completed hotfix ancestry still applies. This is failed-publication recovery/promotion,
  not permission to deploy an arbitrary older source or rebuild a consumed formal version.
- Roll back only to a known stable image compatible with the current database and older clients,
  using [deployment recovery](deployment.md). Image rollback does not restore or overwrite the DB.
  Forward-fix an incompatible migration; document the incident and affected channel states.
- Release state is stored in GitHub Deployments, with an immutable Actions artifact binding the
  deployment ID and trusted controller run. Editable issue text/deployment payloads alone cannot
  authorize production. The daily control job renews the seal before its 90-day retention expires.
  A missing, expired or altered seal blocks all release/deploy readers; it never means “unfrozen”.
  If a controller failed between writing the deployment and uploading its seal, rerun that same
  trusted dispatch: it can replay its unchanged inputs from the previous sealed snapshot. Do not
  delete state or paste an older snapshot to clear a freeze. If the original seal expired or was
  deleted, reconcile the audit trail and repair the controller explicitly before any publication.

## Activation and rehearsal boundary

Install the controller through reviewed code PRs before requiring its new check. The freeze
workflow is discovered from dev and executes controller code from main; neither a pending feature
PR nor installing it only on dev can supply a working trusted-main freeze check. Configure the
`release-freeze` requirement separately with enforcement disabled during this bootstrap. After the
workflow/controller exist on both branches, obtain a successful GitHub Actions `release-freeze`
check on a current dev PR head and enable that requirement without bypass actors. Do not fabricate
a successful check or enable a requirement that has no working producer. Existing PR review and
`ci-required` protections stay active throughout; ordinary main integration requires a merge commit
that preserves the reviewed source. Installing controller code does not itself start a release.

`Partial`: local tests verify state transitions, wrong-source/owner rejection, environment selection
and workflow contracts. They do not establish live signing, installation or branch enforcement.
Before using the workflows operationally, maintainers verify these external prerequisites:

- `release-freeze` and ordinary CI/review are required on dev; direct pushes/bypass are disabled;
  merge queues are drained when freezing. Re-evaluation cannot revoke a merge already accepted by
  GitHub. A detected dev drift blocks QA and deployment and requires owner reconciliation.
- Main and release metadata keep their existing human review/CI rules. Signing/deployment
  environments permit only trusted main workflows; candidate/fork workflows cannot read credentials.
- `PRERELEASE_TESTFLIGHT_GROUP_ID` names the isolated internal QA group; other groups do not receive
  all builds automatically. Testers can install through this group. Signing secrets/profiles remain
  the existing `mobile-release` inputs. No group/credential configuration is mutated by these scripts.
- Retain the successful Publish/Recover completion callback and daily state-seal/status renewal job;
  monitor failed control runs. Both events are discovered on dev and dispatch the trusted main controller.
  Dispatchers need Actions write. The main controller needs Actions read, Deployments write,
  Checks write and Contents write for the public status
  prerelease; preview number reservations/downloads also need Contents write. The status prerelease
  always uses `latest=false` and cannot replace stable download/catalog entries.
- Rehearse failed preview, changed SHA, timeout/cancellation, partial formal failure, hotfix/backflow,
  automatic completion during pending Apple review and compatible rollback with the device checklist
  and real environment owners.

Changing repository protection rules, configuring TestFlight and performing a real distribution are
separate operational actions; changing workflow files does not perform them.
