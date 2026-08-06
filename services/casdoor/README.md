# Casdoor unified auth deployment config (local dev is already in the root docker-compose.yml)
# Init checklist (based on the verified setup):
# 1. Create organization forum (owner=admin)
# 2. Create application forum-app:
#    - organization=forum (bare org name)
#    - signupItems: ID rule=Incremental (self-registered users get numeric IDs)
#    - enablePassword=1, enableSignUp=1
#    - grantTypes: authorization_code, password, refresh_token, token, id_token
# 3. Admin-created accounts must pass an explicit numeric id (API-created users need
#    passwordType=bcrypt fixed)
# 4. Record client_id / client_secret into deploy/.env (CASDOOR_CLIENT_ID/SECRET)
#
# Note: the user id defaults to UUID; credit's GetID() requires numeric uint64 —
#       the numeric-ID config above is mandatory, otherwise all users parse to 0 and collide.
#
# Forum-side integration (implemented):
# - GET /api/auth/oidc/login — PKCE + state + nonce, redirects to Casdoor
# - GET /api/auth/oidc/callback — exchanges code, verifies iss/aud/nonce/exp, enforces
#   numeric sub (ParseUint), then binds (signed-in) or signs in (creates local user)
# - config keys: casdoor.endpoint / casdoor.client_id / casdoor.client_secret
#
# MFA / Passkey (Casdoor-side, recommended for OAuth/Casdoor logins):
# - TOTP/MFA and Passkey (WebAuthn) are enabled inside Casdoor's app settings; the
#   forum does not re-implement them for OIDC logins (forum TOTP covers password login only)
# - Passkey: enable the WebAuthn provider / passkey sign-in in the Casdoor application,
#   then users can register passkeys from Casdoor's account pages
