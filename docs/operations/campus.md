# Campus connection operations

> Doc type: operations guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-19

## Configuration

`Current`: the Go binary accepts the following environment variables (or their `[campus]` TOML equivalents). An absent client, invalid callback, or key shorter than 32 characters disables the connection. Existing installations remain disabled until configured. No access token is placed in application configuration.

| Environment secret | TOML key | Meaning |
|---|---|---|
| `CAMPUS_CLIENT_ID` | `clientId` | Authorized OneTJ-compatible school client |
| `CAMPUS_CLIENT_SECRET` | `clientSecret` | Optional confidential-client credential; the current public client needs none |
| `CAMPUS_REDIRECT_URI` | `redirectUri` | This instance's exact `/api/campus/tongji/callback` URL |
| `CAMPUS_ENCRYPTION_KEY` | `encryptionKey` | Random, environment-specific encryption secret, at least 32 characters |
| `CAMPUS_IDENTITY_KEY` | `identityKey` | Stable random HMAC secret for identity uniqueness, at least 32 characters |

Use HTTPS for deployed callbacks. Loopback HTTP is allowed for local development. Configure the five values in the corresponding GitHub Environment; deploy/apply-config renders escaped TOML. Leave all values empty to disable. Keep both keys out of git and separate between main/dev. The OneTJ WebView callback interception is not a callback relay: YourTJ uses its own server callback with the same authorized client.

Generate independent keys with a cryptographic generator (for example `openssl rand -hex 32`). Back them up in the environment's secret store. A key rotation needs a controlled migration: decrypt and reseal each row for encryption-key rotation; recompute each identity fingerprint from decrypted student ID for identity-key rotation, then switch configuration while writes are stopped. Simply changing the HMAC key would invalidate uniqueness against old rows and is unsupported. If secrets are lost, delete the affected environment's campus bindings under maintenance and require users to reauthorize; do not fabricate identities from profile text.

## Data and environments

`campus_identity_bindings` is an additive AutoMigrate table in the primary PostgreSQL/SQLite schema. No school password, raw student ID column or academic-record table exists. Delete/unlink removes live database credentials; ordinary encrypted backup retention still applies. Access to backups and campus secrets must be kept separate.

`sync-db-from-main.sh` excludes the table's data from PostgreSQL dumps. In SQLite mode it deletes rows and vacuums the temporary snapshot before installation. The dev instance therefore starts without production identity reservations or tokens. Do not restore an unfiltered production backup to a running dev instance. Proxy access logging must omit the entire query of `/api/campus/tongji/callback`; the app's access/panic logs and local Gin logger already exclude it. Do not capture bearer headers or upstream bodies in diagnostics. Campus documents disable Umami and custom site scripts; navigation across the campus boundary reloads the document to unload existing trackers.

Authorization attempts are in-memory, expire after ten minutes, and fail safely after restart; the current deployment uses one process. A refresh lease lasts one minute, with bounded school requests. An interrupted refresh may require reauthorization if the provider has already rotated its token. Unlink is locally authoritative; per-token remote revocation is best effort and does not log out other OneTJ sessions.

Message body reads request the `rt_onetongji_msg_detail` scope. Existing authorizations without this scope receive an update-authorization prompt when opening a message. Reauthorization accepts only the currently bound identity and preserves the old binding until confirmation. List access is checked before each detail read; details are projected to plain text and explicit HTTP(S) links, with no automatic remote resource loads.

## Verification and diagnostics

Use only a user-authorized school login in the official browser page. Test status, authorization/confirmation, refresh, stale confirmations, bidirectional uniqueness, and unlink. The interface distinguishes absent records, upstream failures and expired authorization; a failing school feature must not be represented as a zero score.

Automated tests cover signature/issuer/audience/nonce checks, callback replay and forum-session binding, database uniqueness, atomic replacement, concurrent refresh, late response rejection, encryption isolation, CSRF/session boundaries, account closure, SQLite and PostgreSQL migration. The [product specification](../product/campus.md) lists the verified data features and remaining App/provider gaps.

## Native App

`Current`: native campus uses the same configuration and APIs. School credentials never reach Dart. The authenticated WebView is used only for the native-session handoff and official school authorization; the server callback returns to the native confirmation. No additional client secret, callback scheme or mobile database is required. `Partial`: physical-device school sign-in remains an explicit validation gap; use a user-authorized official login to validate it.
