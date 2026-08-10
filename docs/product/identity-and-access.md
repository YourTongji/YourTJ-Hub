# Identity, Login & Account Lifecycle

> Doc type: product spec
>
> Status: Active (auth chain `Partial`; Casdoor OIDC integrated, session/TOTP implemented)
>
> Owner: Platform maintainers, Security reviewer
>
> Last verified: 2026-08-08

## Identity model

- **Identity's only source = Casdoor (OIDC)**. Casdoor issues id_token with `sub` = numeric user ID
  (uint64). The forum keeps its own password login only as a legacy path; GitHub OAuth (goth) remains
  available.
- **Numeric ID is a hard constraint**: credit's `GetID()` parses sub with `strconv.ParseUint`. A UUID
  fails to parse and falls back to 0, making all users collide. The OIDC callback enforces this
  server-side (`ParseUint` failure or a zero `sub` rejects the login). Casdoor must:
  1. set the app SignupItems ID rule to `Incremental` (self-registered users get auto-increment numeric
     IDs); and
  2. explicitly pass numeric `id` when admins create accounts.
- The forum JWT is only a **session credential** (HS256, self-signed, 7-day TTL, carries a `jti`); it is
  not identity truth. Bans/state changes come from Casdoor; the server syncs a local projection via
  exchange.

## Login flows

### Web

- Password login: RSA-OAEP encrypted password → forum `users.Verify` → if the user enabled TOTP 2FA,
  the server issues a 5-minute `totp_challenge` token instead of a session token; the client posts the
  code (or a one-time recovery code) to `/api/auth/totp/verify`, which issues the real session token.
  A challenge token can mint at most one session: it is atomically consumed on successful verification
  (`totpservice.ConsumeChallenge`), so replaying it cannot create a second session.
- GitHub OAuth (goth): unchanged; callback binds or signs in and issues a session token.
- Casdoor OIDC: `GET /api/auth/oidc/login` (PKCE + state + nonce) → Casdoor → callback exchanges code
  for id_token → verify (iss/aud/nonce/exp) → find or create local user → issue forum JWT. The callback
  also serves account binding for already-signed-in users (`/settings?tab=binding`).

### Mobile (Flutter)

1. AppAuth + PKCE opens the Casdoor authorization page and receives the callback authorization code;
   the app retains the matching PKCE verifier and nonce in memory;
2. the app posts `{code, codeVerifier, nonce, redirectUri}` to `POST /api/auth/oidc/exchange`;
3. the forum backend requires an exact redirect-URI allowlist match, exchanges the code with Casdoor,
   verifies the ID token (issuer, audience, expiry, signature, nonce), and enforces a positive numeric `sub`;
4. the returned forum JWT is stored in Keychain/Keystore (`flutter_secure_storage`); the Casdoor ID token
   is verified server-side and is never persisted by the app.

## Two-factor authentication

- **Password login** is protected by forum-side TOTP (RFC 6238, optional, opt-in). Secrets are stored
  AES-256-GCM encrypted (key derived from `app.signingKey`); recovery codes are stored hashed and are
  single-use; verification attempts are rate-limited per user (10 failures / 15 min).
- **OAuth/Casdoor logins** do not run forum TOTP; their MFA is handled by Casdoor itself
  (Casdoor-native TOTP/MFA, Passkey/WebAuthn). This keeps a single identity source for those paths.
- Passkey: enabled on the Casdoor side (WebAuthn native support); the forum has no WebAuthn code.

## Account lifecycle

- Registration: Casdoor self-service (Incremental ID) or admin-created (explicit numeric id); forum
  password registration still available.
- Session: forum JWT valid 7 days (GooseForum scale, tunable). Every session-scoped token carries a
  `jti` that maps to a row in `user_sessions`; the auth middleware rejects tokens whose session row is
  missing or expired, so revocation is immediate.
- Revocation: users can list sessions (IP masked, device/UA) in Settings → Security, revoke a single
  session, or sign out of all devices. "Sign out of all devices" also bumps `TokenVersion`, which
  invalidates every previously issued token as a second line of defense. Ordinary logout deletes the
  current session row as well, and fails loudly when the revoke errors (no silently surviving token).
- Password change: `TokenVersion` bumps, invalidating old tokens (existing behavior preserved).
- Ban/disable: Casdoor's active/forbidden is authoritative; server syncs at exchange and every check.
- Deletion/export: `Planned` (per product principle 12: answer purpose, visibility, retention, export,
  deletion before persisting).

## Bot personas (Agents)

- An Agent is exactly one bot persona: a `users` row with `actor_type = bot` plus one `agents` row
  keyed by the same user id. Bot rows are created by admins; they have no email, no usable password,
  and no role.
- Authentication for Agents is a unique bearer token (`agt_…`) issued at create/rotate time and
  shown exactly once. The database stores only a SHA-256 hash plus a non-secret 8-char prefix used
  for efficient lookup; the plaintext token is never logged or stored.
- Each Agent has zero or one configurable webhook endpoint. Rotating the token invalidates the old
  one immediately; disabling the Agent rejects resolution until re-enabled. Agent deletion is not
  supported.
- Human-auth isolation: bot rows are rejected by password login, forgot/reset password, OAuth
  (goth) login/binding, Casdoor OIDC login/binding, password change, TOTP setup/enable/disable, and
  human session creation/listing (the JWT session middleware never resolves a bot user). Admin
  surfaces cannot grant bot rows roles or moderator grants. Bot personas are excluded from the
  public user search index; they remain identifiable in forum content and admin surfaces.
- Agent public API: the twelve read/write operations (profile, topic list/create,
  post list/create, search, and the six inbox operations) are `Current` and covered by the
  OpenAPI contract; they authenticate only through the opaque `agt_…` bearer token — cookies,
  human JWTs, session credentials, OAuth, and fallback credentials are never accepted, and every
  failed credential resolves to the same `auth.required` 401 envelope. Agent writes reuse the
  human topic/post rate limits (IP + bot userId) and skip only browser-specific honeypot,
  captcha, and new-user cooldown gates. Topic creation always publishes (`topicStatus=1`).
- Mention detection and webhook wakeups are `Current`: published content writes
  (`topic.published`, `topic.updated`, `post.created`) are parsed for exact case-sensitive
  `@username` candidates (6-32 chars, valid boundaries, max 10 distinct). Unknown users, humans,
  non-bot rows and self-mentions are ignored. Each published write atomically commits a bounded
  `agent.mention` outbox task with the topic/post transaction; the task contains stable IDs, at most
  ten usernames and a ≤64-rune preview, never full content. The ingestion worker rechecks publication,
  upserts one `agent_inbox` row (unique per agent/topic/post) as the delivery fact source, and queues a
  durable `agent.webhook` task. Both workers recover tasks left running after a crash. Webhook delivery
  is best-effort: max 3 attempts (1m then 5m delays via `task_queue.run_at`), 2xx is success, no redirects,
  5s total timeout including DNS resolution, 64KB response cap, DNS-pinned dialing with
  resolve-all/reject-any non-public addresses, and sanitized errors that never contain endpoints,
  tokens, headers, or bodies. Disabled/deleted/no-endpoint Agents still accumulate inbox rows; delivery
  is marked skipped and re-enable does not backfill.
- Inbox APIs: `GET /inbox`, `GET /inbox/:id`, `POST /inbox/:id/read`, `POST /inbox/read-all`,
  `DELETE /inbox/:id`, `DELETE /inbox`; missing or cross-Agent ids share one
  `agent.inbox.notFound` business failure so existence is not leaked.
- OAuth/session/scopes for Agents remain `Planned`.

## Security notes

- PKCE required (Casdoor flow); nonce prevents replay; state prevents CSRF on the callback.
- id_token never persisted; forum JWT goes into an HttpOnly cookie (Secure + SameSite=Lax when the site
  is served over HTTPS).
- Enumeration resistance: login errors do not distinguish "user not found / wrong password".
- Session revocation is implemented as `jti` + `user_sessions` table (decision recorded in ADR note);
  TokenVersion remains as a global invalidation fallback.
- TOTP secrets and recovery codes never leave the server in plaintext (secret encrypted at rest,
  recovery codes hashed, codes shown exactly once during setup).
