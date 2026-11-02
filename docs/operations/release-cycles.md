# Continuous dev delivery

> Doc type: operations reference
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-11-03

`Current`: every push merged into dev requests the trusted-main **Dev / Distribute** workflow with the
same full source SHA. Web deployment and Android/iOS developer packages start independently from that
source. They do not wait for each other or rerun/wait for the source CI matrix. Normal PR CI remains
independent. There is no new prerelease owner, freeze, expiry, QA admission or manual version selection.
Production uses [RC then Confirm](releases.md).

## Ownership and environments

The dev push requester has only Actions dispatch permission, no signing or deployment keys. The main
controller validates the same-repository dev push provenance and current dev head before admitting the
source. Manual retries require a repository write/maintain/admin operator. Signing code always comes
from main, with application source in a separate checkout.

Web connects to dev.yourtj.de and retains its consistent main DB snapshot sync, configuration transaction
and health checks. Native dev builds use dev endpoints and isolated sessions/preferences/databases.
Android is `YourTJ Dev` (`tj.yourtj.forum_app.dev`) and installs alongside production. iOS uses the existing
app ID and replaces the installed version; only the dedicated internal developer group receives it.
Existing automatic/external TestFlight access must not leak this build. Dev never updates stable APK
aliases, the formal note catalog or an App Store submission.

## Start and build

A merged dev push is sufficient. **Dev / Distribute** on main or
`python3 scripts/release/cli.py dev --apply` retries the current dev head manually. Obsolete sources
are rejected before allocation/deploy mutation. The Web branch has its own instance lock and is not
blocked by mobile signing. One mobile bundle can run; only the newest pending dev bundle remains.
Active signing/upload work finishes safely. Shared native platform queues and instance locks retain
serialization with production work; cancelling a queued dev request never interrupts a production upload.

The shared packaging lock encloses allocation and upload for both dev and RC, preserving Apple build
number order. Existing-build formal promotion has its own metadata queue and does not wait for a new
development compile. Web remains independent of both native queues.

The short shared allocator reserves one date ID plus a global mobile native counter. Android/iOS use
the same immutable identity; [numbering rules](releases.md#build-identities) own the exact format.
Retrying the original run restores original artifacts or the exact Apple build. Missing originals after
distribution fail rather than rebuilding the same identity. A new current-dev run consumes a new ID.

## Freeze, fixes and QA

Ordinary dev PRs are not frozen. The compatibility `release-freeze` check now acknowledges normal
review/CI; it does not read old release-cycle state. A new merge may supersede a queued build. Actual
platform completion may differ temporarily; do not claim all platforms are on the newest source until
the recorded platform sources agree.

The `dev-latest` GitHub prerelease contains current dev SHA, latest successful source/run/downloads
for each platform and `dev-status.json`. Every successful platform receipt triggers a short serialized
status refresh; Android discovery does not wait for iOS or Web to finish. Native dev UI uses an informational notice for a newer ready
build, a build still running or unavailable status. There is no time expiry and no app-use restriction.
A date/SHA badge identifies the installed build. Offline status does not certify availability or QA.

`Partial`: signed upgrade/install, endpoint isolation, OAuth, push and physical-device acceptance need
actual candidate evidence. Build and CI success do not establish these user journeys. Follow
[mobile release validation](mobile-releases.md#release-verification-and-privacy-disclosures).

## Enter main, publish and finish

Promote code to main through a normal PR when authorized. Development package bytes cannot become a
production release because endpoints, identity and preview configuration are compiled inputs. Build a
production RC once and Confirm the original artifacts. Dev delivery continues independently during
production packaging or Apple review; there is no owner/cycle completion action for new releases.

## Failure, rollback and state recovery

Inspect Web, Android and iOS results separately. Retry failed jobs in their original run to reuse
identity and originals. A new manual current-dev run is a new build, not a recovery overwrite.
Web uses the shared instance/config transaction and rolls back image/config without restoring data.

Legacy cycle artifacts remain available only to recover or close already existing schema-1/2 operations.
The recovery control retains completion, cancellation, reassignment and immutable-state retention; it
cannot start a new freeze/preview. Existing source/notes/signature identity remains immutable. New dev
admission, deploy, status and production RCs do not consult that state. Do not restart a historical
workflow to bypass the current publication protocol.
