# Casdoor unified auth deployment config (local dev is already in the root docker-compose.yml)
# Init checklist (based on the verified setup):
# 0. 修改内置 admin 默认口令（admin/123）并禁用内置 organization 的公开注册。
#    生产按文档建议放到反代后面，等于对公网开放默认凭证，必须先改。
# 1. Create organization forum (owner=admin)
# 2. Create application forum-app:
#    - organization=forum (bare org name)
#    - signupItems: ID rule=Incremental (self-registered users get numeric IDs)
#    - enablePassword=1, enableSignUp=1
#    - grantTypes: authorization_code, password, refresh_token, token, id_token
# 3. Admin-created accounts must pass an explicit numeric id (API-created users need
#    passwordType=bcrypt fixed)
# 4. 将 client_id / client_secret 写入论坛 config.toml 的 [casdoor] 段
#    （或运行 init-server.sh 前 export CASDOOR_CLIENT_ID/CASDOOR_CLIENT_SECRET 环境变量）。
#    注意：写入 deploy/.env 不会生效——论坛从 config.toml 读取，init-server.sh 读的是
#    shell 环境变量；照旧文档第 4 步操作会发现凭证不生效。
#    存量实例升级：init-server.sh 只在 config.toml 不存在时生成，已有 config.toml 的实例
#    需手工补 [casdoor] 段，否则 OIDC 静默降级（IsConfigured 检查，仅日志 Warn）。
#
# Note: the user id defaults to UUID; credit's GetID() requires numeric uint64 —
#       the numeric-ID config above is mandatory, otherwise all users parse to 0 and collide.
#
# Forum-side integration (implemented):
# - GET /api/auth/oidc/login — PKCE + state + nonce, redirects to Casdoor
# - GET /api/auth/oidc/callback — exchanges code, verifies iss/aud/nonce/exp, enforces
#   numeric sub (ParseUint), then binds (signed-in) or signs in (creates local user)
# - POST /api/auth/oidc/exchange — mobile (Flutter) login: client-held PKCE verifier
#   + custom-scheme redirect, returns the forum JWT in the response body. The
#   redirect URI must be in the configured allowlist (config key
#   casdoor.mobile_redirect_uri, default yourtj://callback) and must be added to
#   the Casdoor application's Redirect URLs alongside the web callback.
# - config keys: casdoor.endpoint / casdoor.client_id / casdoor.client_secret /
#   casdoor.mobile_redirect_uri (optional, default yourtj://callback)
#
# Mobile client: register yourtj://callback in the Casdoor application's
# Redirect URLs allowlist (custom schemes are supported — the official Android/iOS
# SDKs use casdoor://callback by default). The Flutter client uses AppAuth with
# PKCE S256 and exchanges the authorization code at POST /api/auth/oidc/exchange.
#
# MFA / Passkey (Casdoor-side, recommended for OAuth/Casdoor logins):
# - TOTP/MFA and Passkey (WebAuthn) are enabled inside Casdoor's app settings; the
#   forum does not re-implement them for OIDC logins (forum TOTP covers password login only)
# - Passkey: enable the WebAuthn provider / passkey sign-in in the Casdoor application,
#   then users can register passkeys from Casdoor's account pages
