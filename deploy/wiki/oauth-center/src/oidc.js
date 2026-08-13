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
//   1. authorize 带 code_challenge(S256) + nonce，并把 code_verifier 与
//      redirect_uri 编码进 state（state 会原样往返：Hub 回跳 Waline server
//      → Waline server 再回调本服务 /oidc?code=&state=）。
//   2. token 交换时从 state 解出 code_verifier 与 redirect_uri 一并提交
//      （redirect_uri 与授权请求逐字节一致，通过 Hub 的精确匹配）。
//   3. public client 模式：OIDC_SECRET 允许为空（PKCE 替代 client secret）。
// 完整补丁见 deploy/wiki/oauth-center/PATCH.md。
const b64url = (buf) => Buffer.from(buf).toString('base64').replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
const randB64 = (bytes) => b64url(crypto.randomBytes(bytes));
const sha256 = (s) => crypto.createHash('sha256').update(s).digest();
const wrapState = (verifier, redirect, state) => b64url(Buffer.from(JSON.stringify({ v: verifier, r: typeof redirect === 'string' ? redirect : '', s: typeof state === 'string' ? state : '' })));
const unwrapState = (raw) => {
  if (typeof raw !== 'string' || !raw) return { v: '', r: '', s: '' };
  try {
    const parsed = JSON.parse(Buffer.from(raw.replace(/-/g, '+').replace(/_/g, '/'), 'base64').toString('utf8'));
    return { v: typeof parsed.v === 'string' ? parsed.v : '', r: typeof parsed.r === 'string' ? parsed.r : '', s: typeof parsed.s === 'string' ? parsed.s : '' };
  } catch (e) {
    return { v: '', r: '', s: '' };
  }
};
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
    // YourTJ-Hub patch: PKCE S256 + nonce（verifier + redirect_uri 编码进 state 往返）
    const verifier = randB64(32);
    const challenge = b64url(sha256(verifier));
    const url = authorization_endpoint + '?' + qs.stringify({
      client_id: OIDC_ID,
      redirect_uri: redirect,
      response_type: 'code',
      scope: OIDC_SCOPES || 'openid profile email',
      state: wrapState(verifier, redirect, state),
      nonce: randB64(16),
      code_challenge: challenge,
      code_challenge_method: 'S256',
    });
    return this.ctx.redirect(url);
  }

  async getAccessToken(code) {
    const { token_endpoint } = await getDiscovery();
    // YourTJ-Hub patch: 从 state 解出 code_verifier 与 redirect_uri
    // （Waline server 回调本服务时不转发 redirect，必须从 state 恢复）
    const { v: verifier, r: redirect } = unwrapState(this.ctx.params.state);
    const params = {
      client_id: OIDC_ID,
      grant_type: 'authorization_code',
      code,
      code_verifier: verifier,
      redirect_uri: redirect,
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
