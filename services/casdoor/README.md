# Casdoor unified auth deployment config — ARCHIVED / NOT ENABLED
#
# Casdoor is no longer used: the forum built-in OIDC Provider (/api/oauth, authorization code +
# PKCE S256, RS256 id_token, opaque access tokens, sub = numeric users.id) replaces it.
# This file is kept for historical reference only; do not deploy Casdoor.
#
# Init checklist (based on the verified setup):
# 0. 修改内置 admin 默认口令（admin/123）并禁用内置 organization 的公开注册。
#    生产按文档建议放到反代后面，等于对公网开放默认凭证，必须先改。
# 1. Create organization forum (owner=admin)
# 2. Create application forum-app:
#    - organization=forum (bare org name)
#    - signupItems: ID rule=Incremental (self-registered users get numeric IDs)
#    - enablePassword=1, enableSignUp=1
#    - grantTypes: authorization_code, password, refresh_token, token, id_token
# 3. Copy the app's client_id / client_secret.
# 4. 将 client_id / client_secret 写入论坛 config.toml 的 [casdoor] 段
# 5. Set the callback URL to <forum>/api/auth/oidc/callback (web) and
#    yourtj://callback (mobile).
# 6. 论坛启动时校验配置，缺失时 OIDC 登录静默降级（仅日志 Warn）。
#
# Runtime behavior:
# - Login flow: GET /api/auth/oidc/login — PKCE + state + nonce, redirects to Casdoor;
#   callback exchanges the code, verifies id_token (iss/aud/nonce/exp), enforces a positive
#   numeric `sub`, then binds or signs in and issues a forum JWT.
# - Mobile exchange: POST /api/auth/oidc/exchange with {code, codeVerifier, nonce, redirectUri};
#   the redirect URI must exactly match the configured mobile allowlist
#   (casdoor.mobile_redirect_uri, default yourtj://callback) and must be added to
#   the Casdoor application's Redirect URLs alongside the web callback.
# - config keys: casdoor.endpoint / casdoor.client_id / casdoor.client_secret /
#   casdoor.mobile_redirect_uri (optional, default yourtj://callback)
#
# Mobile client: register yourtj://callback in the Casdoor application's
# Redirect URLs (Flutter AppAuth requires a custom scheme; the AppAuth
# SDKs use casdoor://callback by default). The Flutter client uses AppAuth with
# PKCE S256 + nonce and exchanges the code at the forum backend.
#
# MFA / Passkey (Casdoor-side, recommended for OAuth/Casdoor logins):
# - TOTP/MFA and Passkey (WebAuthn) are enabled inside Casdoor's app settings; the
#   forum has no WebAuthn code.
# - Passkey: enable the WebAuthn provider / passkey sign-in in the Casdoor application,
#   then users can register passkeys from Casdoor's account pages
#   (forum settings page shows no passkey UI).
