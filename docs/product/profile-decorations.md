# Profile decorations

> Doc type: product reference
>
> Status: Active (administrator grants and equipment `Current`; XP, trust and Mall integration `Planned`)
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-13

## Selection and presentation

`Current`: Web profile settings and native Flutter settings expose an avatar-frame slot and a
nickname-style slot. Each accepts one owned, enabled decoration or an empty selection. The slots
are independent and can be combined. Preview does not mutate equipment; saving changes both slots
atomically, and failure retains the last saved server state and the user's draft for retry.

The catalog starts empty. Site managers create and configure each avatar frame or nickname style at
`/admin/decorations`; there are no bundled starter styles. Catalog entries do not confer ownership:
all entitlements require administrator grants. No free styles, automated sponsor migration, prices
or purchase flow are enabled.

Decoration changes preserve the original avatar file, nickname and username. Web and Flutter apply
the current effective appearance across topic/reply authors, participants, profiles/cards,
connections, search, chat peers and the signed-in account, notification actors, mention candidates
and member course-review authors. Private notes retain their existing display-name rules. Existing
rendered historical exports, Markdown and forwarded-message snapshots are not rewritten.

Frames use bounded gradient borders or an optional transparent PNG/WebP around the avatar,
preserving its layout and badge overlay. Artwork can be scaled from 50% to 200% of the avatar
diameter, centered on the portrait; the default is 125%. This saved setting is shared by Web and
Flutter. Scaling affects artwork only, not the fallback gradient border. Artwork is installed under
`apps/gooseforum/resource/static/decorations/` and configured by its same-origin static path;
external URLs and unsafe paths are rejected. Loading failure falls back to the configured gradient
border. See the [avatar-frame artwork generation spec](avatar-frame-generation.md).
Nickname styles use validated hexadecimal colors with separate light/dark palettes; custom HTML,
CSS, external assets and animations are not accepted. Definition validation checks ordinary-text
contrast on supported backgrounds; clients fall back to normal text when the actual background
cannot meet 4.5:1. Web also falls back in forced-color mode or on backgrounds it cannot safely
resolve. Missing/unknown/invalid decorations use the ordinary appearance. Decoration colors never
confer official status, trust or administrative permissions.

## Entitlements and administration

`Current`: Hub owns definitions, entitlements and equipment. Site managers edit definitions at
`/admin/decorations`, including name, theme palettes, enabled state and order. A code is stable and
its kind cannot change. Definitions are disabled rather than deleted, preserving references and
audit records. User managers grant or revoke at `/admin/user-decorations`; a reason is required.

The catalog supports name/code search and type/status filters. Creating or editing a definition
opens a labeled form with color pickers, hexadecimal inputs and simultaneous light/dark previews.
Avatar artwork has a live scale slider, percentage input and default reset. Validation feedback keeps
the draft available for correction; failed saves do not close the editor. Disabled definitions remain
previewable for administration but do not display on user profiles. A confirmed save updates the list
from its response rather than depending on another read.

The grants view identifies the loaded user ID and shows inventory and stored equipment. Changing the
ID blocks mutations until that account is loaded. Revocation requires confirmation of the account,
style and reason. Successful changes show a receipt; if the subsequent audit refresh fails, the view
retains the confirmed state and offers a read-only refresh rather than repeating the mutation.
Visible labels and status/error feedback follow the
[WAI form labels](https://www.w3.org/WAI/tutorials/forms/labels/) and
[notifications guidance](https://www.w3.org/WAI/tutorials/forms/notifications/).

The account view shows its latest 100 audit entries, newest first.

A grant/revoke carries a durable request key. Repeating identical input returns current state
without repeating its mutation. Reusing the key with different input is rejected. Replaying a grant
after a later revoke does not restore ownership. Equipment saves verify the caller's own active
entitlements; supplying another user ID cannot change that user's equipment. Cookie writes retain
CSRF protection, and frozen/pending account write restrictions apply.

Each account/style can hold separate entitlement sources. An administrator revoke removes all
active sources of that style and clears its equipped slot when no entitlement remains. Trusted
business integrations revoke only their own source: removing XP or trust qualification does not
erase a concurrently active Credit entitlement. Re-granting with a new key restores availability,
without automatically equipping a revoked item. Disabling a definition hides its effective appearance
without deleting ownership or the stored selection; re-enabling it makes the retained selection
eligible again. Clients re-read on entry, and existing profile caches converge through their normal
refresh/expiry behavior; already visible remote pages are not pushed a global appearance update.

Mutations and their audit/idempotency receipt commit in the same transaction. Account-row locking
serializes equipment changes and entitlement revocation. Definition updates and equipment changes
also leave audit entries. Database migration creates the decoration tables without changing existing
avatars, names, badges, XP or point balances. Closed accounts retain historical records but show no
decorations.

## Anonymous identities

`Current`: public persona/anonymous topic and reply authors and anonymous/legacy course-review
authors have no main-account decoration payload. Clients also ignore owner decorations for those
identity kinds. The main-account selection does not affect anonymous avatars or names; anonymous
identity receives no separate decoration inventory or equipment editor. See
[anonymous identity](anonymous-identity.md).

## Business integration boundary

`Planned`: XP/level, trust and Credit/Mall integrations call the Hub-owned, transaction-bound grant
or revoke service with a source and durable business key. The public administrator API always uses
manual grants; ordinary clients cannot select an integration source or issue themselves entitlements.
Mall can own products/orders/payments while consuming Hub's entitlement and equipment API. It must
not maintain a second authoritative inventory or write Hub tables directly. See the
[ownership decision](../decisions/0082-hub-decoration-entitlement-ownership.md).

`Decision needed`: unlock thresholds, trust data source, prices and source-specific qualification
rules. These rules are not enabled by providing the integration boundary. Permanent purchased
ownership and ongoing XP/trust qualification are distinct; a qualification decrease does not itself
revoke a Credit source. Credit accounting, Mall orders/refunds and historical sponsor migration
remain outside the supported decoration flow.

HTTP request/response shapes live in the [OpenAPI contract](../../packages/api-contract/openapi.yaml);
table definitions and migration registration live in the backend models/migrations.
