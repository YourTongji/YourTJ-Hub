# Campus connection operations

> Doc type: operations guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-20

## Configuration

`Current`: the Go binary accepts the following environment variables (or their `[campus]` TOML equivalents). An absent client, invalid callback, or key shorter than 32 characters disables the connection. Existing installations remain disabled until configured. No access token is placed in application configuration. The same configuration enables Tongji login/registration; no additional school callback or secret is required.

| Environment secret | TOML key | Meaning |
|---|---|---|
| `CAMPUS_CLIENT_ID` | `clientId` | Authorized OneTJ-compatible school client |
| `CAMPUS_CLIENT_SECRET` | `clientSecret` | Optional confidential-client credential; the current public client needs none |
| `CAMPUS_REDIRECT_URI` | `redirectUri` | This instance's exact `/api/campus/tongji/callback` URL |
| `CAMPUS_ENCRYPTION_KEY` | `encryptionKey` | Random, environment-specific encryption secret, at least 32 characters |
| `CAMPUS_IDENTITY_KEY` | `identityKey` | Stable random HMAC secret for identity uniqueness, at least 32 characters |

Use HTTPS for deployed callbacks. Loopback HTTP is allowed for local development. Configure the five values in the corresponding GitHub Environment; deploy/apply-config renders escaped TOML. Leave all values empty to disable. Keep both keys out of git and separate between main/dev. The OneTJ WebView callback interception is not a callback relay: YourTJ uses its own server callback with the same authorized client.

Generate independent keys with a cryptographic generator (for example `openssl rand -hex 32`). Back them up in the environment's secret store. Encryption-key rotation needs a controlled migration: decrypt and reseal each live row, then switch configuration while writes are stopped. The identity HMAC key must remain stable for the lifetime of the environment: permanent reservations intentionally have no recoverable student ID, so recomputing only live bindings would reopen retired identities. Do not rotate or clear this key or the reservation ledger as a reauthorization shortcut. Restore a lost identity key from its secret backup; keep the integration disabled if it cannot be recovered. Loss of the encryption key can be handled by clearing live credentials under maintenance and requiring reauthorization, while retaining the HMAC key and reservations.

## Data and environments

`campus_identity_bindings` and `campus_identity_reservations` are additive AutoMigrate tables in the primary PostgreSQL/SQLite schema. Startup idempotently backfills reservations from current bindings before serving; previously removed identities cannot be reconstructed from mutable email. A reservation contains only the stable HMAC fingerprint, with no user ID, credentials or timestamp, and is retained indefinitely across unlink, replacement and account closure to prevent repeated automated signup. It does not authorize login or block explicit rebinding to an existing account. No school password, raw student ID column or academic-record table exists. New Tongji-registered accounts keep the derived student-ID@tongji.edu.cn as their ordinary private users.email; it follows account email retention and backup rules. Delete/unlink removes live database credentials; ordinary encrypted backup retention still applies. Access to backups and campus secrets must be kept separate.

`sync-db-from-main.sh` excludes both tables' data from PostgreSQL dumps. In SQLite mode it deletes rows and vacuums the temporary snapshot before installation. The dev instance therefore starts without production identity reservations or tokens. Do not restore an unfiltered production backup to a running dev instance. Proxy access logging must omit the entire query of `/api/campus/tongji/callback`; the app's access/panic logs and local Gin logger already exclude it. Callback session/account/rate-limit rejections also return a clean 303 redirect with no-referrer; this does not replace proxy log filtering for the incoming request. Do not capture bearer headers or upstream bodies in diagnostics. Campus documents disable Umami and custom site scripts; navigation across the campus boundary reloads the document to unload existing trackers.

Binding and anonymous login attempts have separate purpose and browser/session proofs. Both are in-memory, expire after ten minutes, and fail safely after restart; the current deployment uses one process. A refresh lease lasts one minute, with bounded school requests. An interrupted refresh may require reauthorization if the provider has already rotated its token. Account closure and unlink both attempt bounded, best-effort refresh-token revocation when credentials can be decrypted. Unlink is locally authoritative; per-token remote revocation is best effort and does not log out other OneTJ sessions.

Message body reads request the `rt_onetongji_msg_detail` scope. Existing authorizations without this scope receive an update-authorization prompt when opening a message. Reauthorization accepts only the currently bound identity and preserves the old binding until confirmation. List access is checked before each detail read; details are projected to plain text and explicit HTTP(S) links, with no automatic remote resource loads.

Campus reads use the independent `campus.read` action (120 requests per user per minute); authorization start and callback share `campus.authorize` (30 requests per user per 15 minutes). Neither has a default per-IP quota or consumes login/course-catalog budgets. Anonymous Tongji login start and callback use the existing login per-IP budget; private campus reads and authenticated binding do not consume it. Existing saved configurations inherit missing campus defaults, and administrators can override them in rate-limit settings.

## Verification and diagnostics

Use only a user-authorized school login in the official browser page. Test status, authorization/confirmation, refresh, stale confirmations, bidirectional uniqueness, and unlink. The interface distinguishes absent records, upstream failures and expired authorization; a failing school feature must not be represented as a zero score.

The original Android crash report has no device or logcat evidence, so its exact root cause remains
unproven. Real-device follow-up evidence now covers Google, GitHub and Tongji: all three crash before
the browser appears. The common Android flutter_appauth/AppAuth/CustomTabs launch layer is therefore
the repair target, while AppAuth-Android 0.11.1's known native lifecycle crash surfaces are only
structural evidence, not proof of this exact stack. The Android OIDC path now uses an external system
browser, manual PKCE/state/nonce and a MainActivity MethodChannel/EventChannel callback bridge;
OAuth login does not use WebView. The debug merged manifest is machine-checked for exactly one
`yourtj://callback` handler owned by MainActivity, no AppAuth receiver claim, and
`flutter_deeplinking_enabled=false`. The existing `https` browser query remains supplied by the
merged dependency manifests.

Automated tests cover signature/issuer/audience/nonce checks, callback replay and forum-session binding, database uniqueness, atomic replacement, concurrent refresh, late response rejection, encryption isolation, CSRF/session boundaries, account closure, SQLite and PostgreSQL migration. The [product specification](../product/campus.md) lists the verified data features and remaining App/provider gaps.

## Native App

`Current`: native campus uses the same configuration and APIs. School credentials never reach Dart. The authenticated WebView remains limited to the separate campus session handoff/workspace flow; it is not an OAuth provider login surface. Google, GitHub and Tongji Android OAuth use the external-browser/manual-PKCE callback path above. No additional client secret, callback scheme or mobile database is required. `Partial`: physical-device school sign-in remains an explicit validation gap; the new APK must be tested with a user-authorized official login.

## Teaching-date rules

`Current`: SiteManager administrators manage rules at `/admin/settings/campus-calendar`.
AI drafting reuses the AI summary provider configuration (admin provider settings first,
`[ai_summary]` fallback), its shared global generation quota and a 30-second cancellable
request. The public course-summary display switch does not gate this explicit administrator action. Missing configuration or invalid model output leaves published rules unchanged;
manual editing remains available. Only the entered public notice and explicit year reach
the provider; no personal campus records are included. Draft notices are not persisted.

Confirmed absolute-date rules use the existing `page_config` storage under
`campusCalendarAdjustments`; no schema migration or new secret is needed. Saving compares
the loaded content revision atomically, rejecting stale edits. Rules take effect on the next
today-timetable read and on exports with adjustments enabled, without restarting.
The `today` dataset uses the same teaching-date rules on Web and Flutter and always
applies published adjustments. Rule/calendar errors remain visible instead of falling
back to the original schedule; the original weekly grid stays unchanged. To remove an adjustment, delete its row
and apply; clearing both lists restores unadjusted exports. Check each year's school teaching
notice rather than inferring makeup lessons from national workday calendars. Previously
imported calendar files cannot be retracted or updated by the server.

## Sign-in and registration

`Current`: Web `/api/auth/tongji` and the App's `login_hint=tongji` use the configured campus
provider. School registration requires a never-bound identity, signup to be enabled, the daily quota to be available and
`tongji.edu.cn` to be allowed (or no domain restriction). Existing bindings can still sign in when
new registration is closed. New identities complete `/register/tongji` with a chosen username and
password before any account/session is created; school verification replaces email activation.
Pending registration is bounded in process memory, expires at the original ten-minute deadline,
and requires an HttpOnly cookie plus an independent CSRF proof. A restart requires fresh school
authentication. Completion uses the registration rate limit and grants no administrator role. Current and pending email
collisions require account recovery/login followed by explicit campus binding, not automatic merging.

`Current`: live and dev campus bindings remain separate. A production account copied to dev keeps its
ordinary email but has no campus binding there; use an existing login method and bind in dev before
using school sign-in. Unbinding removes the school login method without changing email or activation.
Password and school registration share the daily quota database lock and count closed accounts for
their creation day. Reservations are committed atomically with account/binding creation; failed
registration rolls them back. The PostgreSQL CI job runs the concurrent identity and quota tests.

`Partial`: the sign-in/account-creation/mobile OIDC chain is covered with a simulated upstream.
Real school registration and physical-device AppAuth return require an authorized device test.
