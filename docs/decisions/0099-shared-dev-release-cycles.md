# Shared dev release cycles with exact-source QA

## Status

Proposed
Class: process

## Context and Problem Statement

An expanding team integrates continuously into dev while releases span Web, Android and iOS.
Moving branch heads and incomplete device regression make it difficult to identify what was tested
and who owns unfinished channel publication. The team operates one dev site and does not want a
separate stabilization branch and server lifecycle for each release.

## Decision Drivers

- Keep one dev site and preserve the existing reviewed production publication boundary.
- Bind QA and production configuration checks to one source commit without manual artifact accounting.
- Make release ownership, freeze expiry, partial failure and hotfix backflow explicit.
- Keep dev builds out of stable downloads, production sessions and public testing groups.
- End each formal release independently; external store review must not hold the integration freeze.

## Considered Options

- Freeze dev briefly, pin a source SHA and distribute isolated previews under one owner.
- Create a short-lived release branch and an isolated environment for each candidate.
- Continue merging dev during QA and release its latest head with a manual checklist.
- Keep one cycle open until every previewed platform is publicly available, or end it after each
  successful formal Release request and start fresh cycles for later platforms.

## Decision Outcome

Use one owner-controlled cycle occupying dev. All preflight, preview and production builds select
the full source SHA. A changed source or dev redeployment revokes QA. A separate production
configuration rehearsal checks mobile environment differences before the existing Release PR protocol
can publish mobile. Web can publish after dev QA to supply the API needed by a subsequent mobile cycle.
Every successful formal request automatically ends its ordinary cycle and unfreezes dev, even if
other platforms were previewed. TestFlight/App Store submission does not wait for approval or public
availability. Later platforms require a new cycle and fresh QA. The initiating human retains smoke,
failure and store-review follow-up responsibility in the tracking issue; ownership
normally transfers when the named successor accepts. A repository admin can directly reassign a
human maintainer with an audited reason, including on expiry or after failure. Reassignment does
not supply QA. Successful completion closes and unfreezes together; abort preserves failure facts.

Preview artifacts are selectable independently of production scope. The same-source dev site can be
reused by later mobile-only runs; every production target still requires its corresponding preview
coverage. Rebuilding a fixed commit is supported, without requiring identical dev and production
package bytes. Failed formal versions retain their original-artifact recovery contract.

Hotfix admission and abort wait in the formal publication queue. Admission verifies every selected
channel shares the actual base before taking ownership. Abort preserves partial and unknown channel facts, revokes QA and
releases the cycle's dev freeze. Published/uncertain hotfixes retain backflow obligations that block
new cycles and ordinary publication. Historical recovery requires an original reviewed, reserved
release predating this controller; absence of cycle data is insufficient and cannot bypass an abort.

Main runs the controller. Deployments provide the state pointer; an immutable Actions artifact
binds its deployment ID to the trusted controller run, because editable metadata is not authorization.
The controller renews that artifact periodically and fails closed if its seal is unavailable.
Control projects merge checks and the public notice in a separate retryable job after sealing;
retrying failed projections never replays an already applied transition. New freezes require the
dev merge queue to drain, and state refreshes include synthetic merge-group SHAs.
Successful formal runs seal the exact cycle/request binding. A completion callback dispatches main
after the publication lock is released; the controller verifies that successful run, immutable seal
and the request's latest receipts before ending the cycle. This keeps delayed callbacks from
unlocking another cycle without broadening authority to default-branch code or editable receipts.
Build numbers use automatically reserved annotated tags shared by preview and formal preparation;
operators track source and QA rather than calculate hashes. Existing automatic binary checks stay.

Android dev previews use their own application ID (`tj.yourtj.forum_app.dev`), name and badged
icon, so developers install them beside the stable app on real devices. Production configuration
rehearsals and iOS previews retain the existing application IDs and replace installed apps; iOS
previews require an internal TestFlight QA group. All previews isolate dev storage and stable
updates and identify their environment.
An informational public status projection warns installed previews when the selected source is no
longer current and links testers to the current candidate or production entry. Missing or stale
status remains unverified. The notice follows source identity, permits rebuilding, and supplies no
release authorization. It does not delete old distributions or force an installed app to change sites.
Hotfixes originate at the actual distributed source, preempt ordinary candidates, and must merge
back to dev before ordinary publication resumes. Future ordinary sources retain hotfix ancestry;
another channel's hotfix still starts from that channel's actual distributed baseline.

## Pros and Cons of the Options

- A shared dev freeze has low infrastructure overhead and a clear QA target, but briefly delays
  integration, requires explicit expiry handling, and cannot offer simultaneous candidate sites.
  Workflow checks require corresponding GitHub branch/environment rules to prevent bypasses.
- A release branch plus isolated environment supports parallel releases and continuous dev merges,
  but adds branch, database, server and cleanup ownership the team does not currently want.
- A moving dev head has little process overhead, but a checklist cannot authorize untested commits
  arriving after QA. It leaves the original regression and accountability problem unresolved.
- Internal TestFlight groups avoid changing the public beta review account/queue, but testers need
  App Store Connect membership. Supporting external dev testers needs a separately reviewed design.
- A separate Android dev application ID lets one device keep stable and dev installs, but the dev
  app has no OEM push registration (dev push is already disabled) and shares the OAuth callback
  scheme, so a sign-in return may show an app chooser. Rehearsals keep the production ID because
  they verify the production package, signing and push configuration.
- Completing each formal request bounds the freeze to work controlled by the team. It requires fresh
  QA for a later platform and keeps Apple-review follow-up separate from cycle completion. Waiting
  for all platforms to become public would keep integration tied to external review latency.

## Links

- [Release-cycle runbook](../operations/release-cycles.md)
- [Reviewed release boundary](0055-reviewed-release-pipeline.md)
- [Issue #1240](https://github.com/YourTongji/YourTJ-Hub/issues/1240)
- [GitHub deployment API](https://docs.github.com/en/rest/deployments/deployments)
- [GitHub immutable Actions artifacts](https://docs.github.com/en/actions/using-workflows/storing-workflow-data-as-artifacts)
- [Apple TestFlight beta groups](https://developer.apple.com/documentation/appstoreconnectapi/beta-groups)
- [GitHub workflow jobs and rerun history](https://docs.github.com/en/rest/actions/workflow-jobs#list-jobs-for-a-workflow-run)
- [GitHub workflow completion events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#workflow_run)
- [Android version codes and downgrade prevention](https://developer.android.com/studio/publish/versioning)
- [ASC 5.0.0 source](https://github.com/rorkai/App-Store-Connect-CLI/tree/5.0.0)
