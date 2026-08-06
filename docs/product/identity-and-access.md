# Identity, Login & Account Lifecycle

> Doc type: product spec
>
> Status: Active (auth chain `Planned`; numeric-ID premise verified)
>
> Owner: Platform maintainers, Security reviewer
>
> Last verified: 2026-08-06

## Identity model

- **Identity's only source = Casdoor (OIDC)**. The forum does not maintain its own password system;
  Casdoor issues id_token with `sub` = numeric user ID (uint64).
- **Numeric ID is a hard constraint**: credit's `GetID()` parses sub with `strconv.ParseUint`. A UUID
  fails to parse and falls back to 0, making all users collide. Casdoor must:
  1. set the app SignupItems ID rule to `Incremental` (self-registered users get auto-increment numeric
     IDs); and
  2. explicitly pass numeric `id` when admins create accounts.
- The forum JWT is only a **session credential** (HS256, self-signed, refresh_token possible); it is not
  identity truth. Bans/state changes come from Casdoor; the server syncs a local projection via exchange.

## Login flows

### Web

Standard OIDC authorization-code flow: browser redirects to Casdoor → callback → server exchanges code
for id_token → verify (iss/aud/nonce/exp) → find or create local user → issue forum JWT (HttpOnly cookie
or Authorization header, decided at implementation).

### Mobile (Flutter)

1. appauth + PKCE → Casdoor auth page (external browser) → callback with code;
2. token endpoint exchanges code for id_token (verify nonce);
3. `POST /api/auth/oidc/exchange` (body: idToken, nonce) → server verifies → returns forum JWT;
4. JWT stored in Keychain/Keystore (flutter_secure_storage); id_token stays in memory only.

## Account lifecycle

- Registration: Casdoor self-service (Incremental ID) or admin-created (explicit numeric id).
- Session: forum JWT valid 7 days (GooseForum scale, tunable at implementation); with refresh_token,
  short access + long refresh; password change/kick → TokenVersion invalidates old tokens.
- Ban/disable: Casdoor's active/forbidden is authoritative; server syncs at exchange and every check.
- Deletion/export: `Planned` (per product principle 12: answer purpose, visibility, retention, export,
  deletion before persisting).

## Security notes

- PKCE required (appauth default); nonce prevents replay (the exchange endpoint must verify it).
- id_token never persisted; forum JWT goes into OS secure storage.
- Enumeration resistance: login errors do not distinguish "user not found / wrong password"
  (Casdoor-side config).
- Session revocation: TokenVersion or refresh rotation (decided at implementation; record in note).
