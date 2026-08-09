# Identity, Login & Account Lifecycle

> Doc type: product spec
>
> Status: Active (auth chain `Partial`; built-in OIDC Provider + password/TOTP/GitHub OAuth implemented)
>
> Owner: Platform maintainers, Security reviewer
>
> Last verified: 2026-08-09

## Identity model

- **Identity source = the forum's own `users` table** (uint64 numeric ID). The forum built-in OIDC
  Provider issues standard OIDC tokens with `sub` = the user's numeric ID for a small set of
  first-party clients (e.g. the course-selection site and the mobile app). Password login, TOTP 2FA
  and GitHub OAuth (goth) remain available.
- **Numeric ID is a hard constraint**: credit's `GetID()` parses sub with `strconv.ParseUint`. A UUID
  fails to parse and falls back to 0, making all users collide. The OIDC provider enforces this
  server-side (the `sub` claim is always the numeric `users.id`).
- The forum JWT is only a **session credential** (HS256, self-signed, 7-day TTL, carries a `jti`); it
  is not identity truth and is never issued to external OIDC clients — those receive opaque access
  tokens scoped to the built-in provider.
- Casdoor is not enabled: no Casdoor routes, config, or identity dependency exist in this deployment.

## Login flows

### Web

- Password login: RSA-OAEP encrypted password → forum `users.Verify` → if the user enabled TOTP 2FA,
  the server issues a 5-minute `totp_challenge` token instead of a session token; the client posts the
  code (or a one-time recovery code) to `/api/auth/totp/verify`, which issues the real session token.
  A challenge token can mint at most one session: it is atomically consumed on successful verification
  (`totpservice.ConsumeChallenge`), so replaying it cannot create a second session.
- GitHub OAuth (goth): unchanged; callback binds or signs in and issues a session token.

### Built-in OIDC Provider (first-party clients)

- Discovery: `/.well-known/openid-configuration` under the issuer path `/api/oauth`; endpoints:
  `/authorize`, `/authorize/callback` (login bridge), `/token`, `/userinfo`, `/keys`.
- Authorization code + PKCE S256 only; `state`, `nonce`, exact redirect-URI matching and code
  single-use are enforced. The login bridge requires an authenticated forum session and never
  follows a client-supplied redirect target (open-redirect safe).
- ID tokens are RS256-signed with a persistent provider key (inline PEM or key file, auto-generated
  otherwise); `sub` = numeric `users.id`. Opaque access tokens are stored as token-ID rows only.

### Mobile (Flutter)

1. AppAuth + PKCE opens the forum built-in OIDC authorization page and receives the callback
   authorization code; the app retains the matching PKCE verifier and nonce in memory;
2. the app posts `{code, codeVerifier, nonce, redirectUri}` to `POST /api/auth/oidc/exchange`;
3. the forum backend requires an exact redirect-URI match against the registered mobile client,
   redeems the code atomically (single-use, PKCE verified), checks the bound nonce and numeric `sub`,
   then issues a forum JWT session;
4. the returned forum JWT is stored in Keychain/Keystore (`flutter_secure_storage`); OIDC tokens are
   verified server-side and are never persisted by the app.

## Two-factor authentication

- **Password login** is protected by forum-side TOTP (RFC 6238, optional, opt-in). Secrets are stored
  AES-256-GCM encrypted (key derived from `app.signingKey`); recovery codes are stored hashed and are
  single-use; verification attempts are rate-limited per user (10 failures / 15 min).
- **GitHub OAuth and built-in OIDC logins** do not run forum TOTP; MFA for those paths is a
  `Decision needed` (the built-in provider reuses the authenticated forum session, so a future phase
  may enforce forum TOTP there as well).

## Account lifecycle

- Registration: forum self-service password registration (with email verification when enabled);
  GitHub OAuth can auto-provision accounts. The built-in OIDC provider never creates accounts — it
  authenticates existing users.
- Session: forum JWT valid 7 days (GooseForum scale, tunable). Every session-scoped token carries a
  `jti` that maps to a row in `user_sessions`; the auth middleware rejects tokens whose session row is
  missing or expired, so revocation is immediate.
- Revocation: users can list sessions (IP masked, device/UA) in Settings → Security, revoke a single
  session, or sign out of all devices. "Sign out of all devices" also bumps `TokenVersion`, which
  invalidates every previously issued token as a second line of defense (this also invalidates OIDC
  opaque access tokens at the userinfo endpoint). Ordinary logout deletes the current session row as
  well, and fails loudly when the revoke errors (no silently surviving token).
- Password change: `TokenVersion` bumps, invalidating old tokens and OIDC access tokens.
- Ban/freeze: the forum `users.is_frozen` flag is authoritative; the OIDC userinfo endpoint and
  exchange path reject frozen accounts.
- Deletion/export: `Planned` (per product principle 12: answer purpose, visibility, retention, export,
  deletion before persisting).

## Security notes

- PKCE S256 required on authorize; nonce prevents replay; state prevents CSRF on the callback; the
  authorization code is single-use (atomic conditional update) and the login bridge cannot be used as
  an open redirect.
- The forum HS256 JWT is never accepted at the OIDC userinfo endpoint (opaque access tokens only).
- Client secrets are configured server-side and never logged.
- id_token / access token never persisted by clients; forum JWT goes into an HttpOnly cookie
  (Secure + SameSite=Lax when the site is served over HTTPS).
- Enumeration resistance: login errors do not distinguish "user not found / wrong password".
- Session revocation is implemented as `jti` + `user_sessions` table (decision recorded in ADR note);
  TokenVersion remains as a global invalidation fallback.
- TOTP secrets and recovery codes never leave the server in plaintext (secret encrypted at rest,
  recovery codes hashed, codes shown exactly once during setup).
