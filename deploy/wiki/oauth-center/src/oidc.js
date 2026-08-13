const Base = require('./base');
const qs = require('querystring');
const crypto = require('crypto');
const request = require('request-promise-native');

const { OIDC_ID, OIDC_SECRET, OIDC_ISSUER, OIDC_SCOPES, OIDC_AUTH_URL, OIDC_TOKEN_URL, OIDC_USERINFO_URL } = process.env;

let discovery;
async function getDiscovery() {
  if (discovery) return discovery;
  const issuer = (OIDC_ISSUER || '').replace(/\/+$/, '');
  if (!issuer && !(OIDC_AUTH_URL && OIDC_TOKEN_URL && OIDC_USERINFO_URL)) {
    throw new Error('Missing OIDC_ISSUER or explicit endpoints');
  }
  if (OIDC_AUTH_URL && OIDC_TOKEN_URL && OIDC_USERINFO_URL) {
    discovery = {
      authorization_endpoint: OIDC_AUTH_URL,
      token_endpoint: OIDC_TOKEN_URL,
      userinfo_endpoint: OIDC_USERINFO_URL,
    };
    return discovery;
  }
  const url = issuer + '/.well-known/openid-configuration';
  discovery = await request.get(url, { json: true });
  return discovery;
}

// --- YourTJ-Hub patch ---
// 上游 walinejs/auth 的 OIDC 实现不发 PKCE 也不发 nonce，而 YourTJ-Hub 的
// OIDC provider 强制 PKCE S256 + nonce。补丁：
//   1. authorize 带 code_challenge(S256) + nonce。
//   2. code_verifier 与 redirect_uri **只存服务端内存**（不透明 state id 进入
//      OIDC state 前向信道），state id 一次性消费 + TTL 清理。PKCE 威胁模型
//      要求 verifier 绝不与授权码同信道出现——把 verifier 编码进 state 会在
//      回调被拦截时同时泄漏 code+verifier，使 PKCE 失效。
//   3. public client 模式：OIDC_SECRET 允许为空（PKCE 替代 client secret）。
// 单实例自托管部署下内存存储即可；进程重启会丢失未完成登录（用户重试即可）。
// 完整补丁见 deploy/wiki/oauth-center/PATCH.md。
const b64url = (buf) => Buffer.from(buf).toString('base64').replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
const randB64 = (bytes) => b64url(crypto.randomBytes(bytes));
const sha256 = (s) => crypto.createHash('sha256').update(s).digest();

const SESSION_TTL_MS = 10 * 60 * 1000; // 授权请求 TTL 对齐(10 分钟)
const sessions = new Map(); // opaqueStateId -> { verifier, redirect, state, createdAt }

function storeSession(verifier, redirect, state) {
  const id = randB64(24);
  const now = Date.now();
  // 顺带清理过期会话, 防止内存无限增长
  for (const [key, val] of sessions) {
    if (now - val.createdAt > SESSION_TTL_MS) sessions.delete(key);
  }
  sessions.set(id, { verifier, redirect: typeof redirect === 'string' ? redirect : '', state: typeof state === 'string' ? state : '', createdAt: now });
  return id;
}

// 一次性消费: 取出即删, 防止 state id 重放换取第二个 token
function takeSession(rawId) {
  if (typeof rawId !== 'string' || !rawId) return null;
  const val = sessions.get(rawId);
  if (val) sessions.delete(rawId);
  return val;
}
// --- /YourTJ-Hub patch ---

module.exports = class extends Base {
  static check() {
    // YourTJ-Hub patch: public client（无 OIDC_SECRET）也允许（PKCE S256）
    if (!OIDC_ID) return false;
    return !!(OIDC_ISSUER || (OIDC_AUTH_URL && OIDC_TOKEN_URL && OIDC_USERINFO_URL));
  }

  static info() {
    return {
      origin: new URL(OIDC_ISSUER || OIDC_AUTH_URL).hostname
    };
  }

  async redirect() {
    const { redirect, state } = this.ctx.params;
    const { authorization_endpoint } = await getDiscovery();
    // YourTJ-Hub patch: PKCE S256 + nonce（verifier + redirect 存服务端, state 只放不透明 id）
    const verifier = randB64(32);
    const challenge = b64url(sha256(verifier));
    const stateId = storeSession(verifier, redirect, state);
    const url = authorization_endpoint + '?' + qs.stringify({
      client_id: OIDC_ID,
      redirect_uri: redirect,
      response_type: 'code',
      scope: OIDC_SCOPES || 'openid profile email',
      state: stateId,
      nonce: randB64(16),
      code_challenge: challenge,
      code_challenge_method: 'S256',
    });
    return this.ctx.redirect(url);
  }

  async getAccessToken(code) {
    const { token_endpoint } = await getDiscovery();
    // YourTJ-Hub patch: 从服务端会话恢复 code_verifier 与 redirect_uri
    // （Waline server 回调本服务时只转发 code+state, 不转发 redirect;
    //   redirect_uri 必须与授权请求逐字节一致, 只能从会话恢复）
    const session = takeSession(this.ctx.params.state);
    if (!session) {
      throw new Error('oauth state missing or expired');
    }
    const params = {
      client_id: OIDC_ID,
      grant_type: 'authorization_code',
      code,
      code_verifier: session.verifier,
      redirect_uri: session.redirect,
    };
    if (OIDC_SECRET) {
      params.client_secret = OIDC_SECRET;
    }
    return request.post({
      url: token_endpoint,
      form: params,
      json: true,
    });
  }

  async getUserInfoByToken({ access_token }) {
    const { userinfo_endpoint } = await getDiscovery();
    const user = await request({
      url: userinfo_endpoint,
      method: 'GET',
      headers: {
        Authorization: `Bearer ${access_token}`,
      },
      json: true,
    });
    const rawAvatar = user.picture || user.avatar;
    const avatar = typeof rawAvatar === 'string'
      ? rawAvatar.trim().replace(/^`+|`+$/g, '').replace(/^\"+|\"+$/g, '')
      : undefined;
    const profileUrl = user.profile || user.website || (typeof user.url === 'string' ? user.url : '');
    return {
      id: user.sub,
      name: user.name || user.preferred_username || user.nickname,
      email: user.email,
      url: profileUrl || '',
      avatar,
    };
  }

  async getUserInfo() {
    const { code, state: _state, redirect: directRedirect } = this.ctx.params;
    const parsed = qs.parse(_state || '');
    if ((!parsed.redirect || typeof parsed.redirect !== 'string') && this.ctx.search) {
      const search = this.ctx.search.slice(1);
      const states = (search.match(/(?:^|&)state=([^&]*)/g) || []).map((s) => decodeURIComponent(s.split('=')[1] || ''));
      const picked = states.find((v) => v && /redirect=/.test(v)) || states.find((v) => v) || '';
      if (picked) {
        Object.assign(parsed, qs.parse(picked));
      }
    }
    const redirect = parsed.redirect || directRedirect;
    const state = parsed.state || '';
    if (!code) {
      return this.redirect();
    }
    if (redirect && this.ctx.headers['user-agent'] !== '@waline') {
      const target = redirect + (redirect.includes('?') ? '&' : '?') + qs.stringify({ code, state });
      return this.ctx.redirect(target);
    }
    this.ctx.type = 'json';
    const accessTokenInfo = await this.getAccessToken(code);
    const userInfo = await this.getUserInfoByToken(accessTokenInfo);
    return this.ctx.body = userInfo;
  }
};
