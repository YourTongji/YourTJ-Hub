# Profile decorations

> Doc type: product reference
>
> Status: Active (administrator grants and equipment `Current`; XP, trust and Mall integration `Planned`)
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-19

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

A frame definition is configured as either a bounded gradient border or a transparent PNG/WebP
around the avatar; the two choices are exclusive and the admin form shows only the selected kind's
fields (palette for a gradient border, artwork and scale for an image frame). A gradient border uses
its palette only and draws no artwork; an image frame uses its artwork only, ignores the palette and
draws no colored ring — not while the artwork loads, and not when it is unavailable or fails to
decode; there is no fallback border. Both kinds preserve the avatar's layout and badge overlay.
Both kinds also scale with the avatar: an image frame's diameter is always the configured scale
times the avatar diameter, and a gradient border's thickness is a fixed 7.5% of the avatar diameter
(3 px on a 40 px avatar). List rows use 20–24 px avatars and profile headers 108 px, so neither the
border nor the artwork may be sized in fixed pixels — that produced a fat ring in lists and a
hairline ring on the profile.
Artwork can be scaled from 50% to 200% of the avatar diameter, centered on the portrait; the default
is 125%. This saved setting is shared by Web and Flutter and applies to image artwork only; gradient
borders ignore it. SVG, PNG, WebP and GIF uploads are supported. GIF animation is preserved and every composited
frame must keep the central opening transparent. SVG is rendered to PNG before validation and
storage. Site managers upload artwork in `/admin/decorations` and save its generated
`/static/decorations/` path without rebuilding the app. Existing embedded artwork under
`apps/gooseforum/resource/static/decorations/` remains valid. External URLs and unsafe paths are
rejected. See the [avatar-frame artwork generation spec](avatar-frame-generation.md).
Nickname styles use validated hexadecimal colors with separate light/dark palettes; custom HTML,
CSS, external assets and animations are not accepted. Definition validation samples only the
gradient's two endpoint stops against the supported light and dark theme backgrounds, at the WCAG
large-text floor of 3:1; the ramp between the stops is not interpolated. The admin console's live
preview always paints the entered gradient regardless of the floor, shows the measured lowest
contrast below the preview, and flags a palette below 3:1 there instead of letting it be saved.
Clients still check the actual background and fall back to normal text when it
cannot meet 3:1. Web also falls back in forced-color mode or on backgrounds it cannot safely
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
