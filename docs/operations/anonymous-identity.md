# Anonymous identity operations

> Doc type: operations reference
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-07

## Permissions and private evidence

`Current`: the role permission editor grants `anonymous.identity.reveal` (ID 7) explicitly. Ordinary
Admin permission does not imply this grant. Assign it only to approved audit operators, review the
grants on role changes, and remove it when audit responsibility ends. Admin → Anonymous identities
requires this explicit grant together with user management (ID 1, or Admin's management wildcard).
Operators enter a viewing reason before loading the list; every search, filter, refresh and page
records its returned mappings in the private audit with action `admin.list` and a shared trace ID.
An empty result still records the access action. Bans/restores require a separate action reason.
Ordinary moderators retain content moderation without access to private mappings.
The restricted audit stores action, operator, persona UID, owner ID, reason, trace ID and timestamp
before returning a reveal or committing a governance restriction. The restricted list returns only
persona display/status fields, numeric owner ID, username and closed/frozen flags. Audit records
have no ordinary admin-list endpoint. Database and backup credentials able to read them are
restricted operator credentials and must not be distributed to content moderators.

`Current`: bindings and reveal/governance audits are retained indefinitely, including after account
closure, under the approved product retention policy. Do not cascade account cleanup into these
tables or delete the occupied persona slot. Candidate batches are usable only through their Shanghai
day and are removed by the existing hourly cron once expired for over seven days. The same cron
removes quota rows whose Shanghai day is strictly earlier than the day seven days ago; current-day
counters and the cutoff day's rows remain intact. Quotas never become public data. Binding/audit
retention changes need a product decision because they change audit availability and the
one-identity guarantee.

`Current`: private state reads share the configurable `interact` HTTP rate limit with persona
writes. Excess requests return HTTP 429 with `Retry-After`, independently of the daily name-draw
quota. This bounds polling while the state transaction serializes with writes to keep remaining
draws and persisted batches consistent. Rate-limit settings apply to this read path too.

## Migration, backup and recovery

`Current`: append-only AutoMigrate adds persona/binding/quota/batch/audit tables, the topic/post
persona UID, notification private-actor fields and the independent account governance restriction.
Existing numeric user IDs, OIDC subjects and legacy anonymous content remain unchanged. The versioned
phrase6-v1 name components are embedded in the same binary; no runtime download is required.
A source update does not rewrite confirmed names or persisted candidate batches.

Back up the complete main database with restricted access before upgrading. A complete database
backup must preserve the five anonymous tables, content persona UIDs, notification actor projection
and the account restriction together. Restore them as one consistent database snapshot and verify
one binding per owner, one owner per UID, audit readability for an explicit operator, and blocked
writes for restricted/closed accounts. Avatar seeds are private backup data.

`Current`: ordinary administrative JSON/CSV exports redact anonymous numeric authors/editors and
include public persona UIDs only. They exclude binding, candidate, quota and restricted audit tables.
Import rejects a snapshot containing persistent persona UIDs because a public export cannot restore
the private attribution safely. Use the restricted complete database backup for recovery. Do not
use an ordinary content export as an anonymous identity backup or manufacture numeric authors.

Do not downgrade to a binary that predates persona projection: it can display stored numeric owners.
Keep the database and serving binary consistent; an emergency rollback should restore a complete
pre-feature snapshot with its corresponding binary, with the usual loss of subsequent writes.

## Cache and logging boundary

`Current`: anonymous settings, candidate, confirmation, toggle, governance and reveal responses send
`Cache-Control: private, no-store`, including the restricted admin list and governance responses.
Reverse proxies must preserve it. Anonymous API and `/a/` access and panic logs omit account and IP,
use the route pattern and discard query values. Restricted evidence never
enters ordinary operation/moderation exports. Authenticated Web layouts disable analytics, replay
and configured script injection; returning between tracking states reloads the whole page.
Identity settings/legacy governance routes live under `/api/forum/anonymous/`, restricted admin
operations under `/api/admin/anonymous-identities/`, and public personas under `/a/`. Extensions
to these namespaces preserve this logging boundary. The admin page does not export mappings or
persist them in browser storage; each revisit requires a new viewing reason.

An avatar URL is immutable and publicly cacheable for one year. Public name/profile payloads must
be refreshed after a rename; clients must not publish private settings state to a shared cache.
Notification hydration and push retries use the public persona projection, not its private owner.

See [product behavior](../product/anonymous-identity.md) and
[deployment backup procedures](deployment.md#backup-and-sync-scripts-under-postgresql).
