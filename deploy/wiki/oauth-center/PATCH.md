# walinejs/auth 的 YourTJ-Hub 补丁（PKCE + nonce）

## 为什么需要补丁

上游 [walinejs/auth](https://github.com/walinejs/auth)（v1.1.0，master）的 OIDC 实现
**不发送 PKCE（`code_challenge`/`code_verifier`），也不发送 `nonce`**，而 YourTJ-Hub
的内建 OIDC Provider 对**所有**客户端强制 PKCE S256 + nonce + state（安全基线，
authorize 阶段直接拒绝不符合的请求）。因此上游 auth center 直接对接 Hub 会在授权
端点被拒。本目录的 `src/oidc.js` 是打过补丁的版本。

## 补丁内容（`src/oidc.js` vs 上游）

1. **PKCE S256**：authorize 时生成随机 `code_verifier`（32 字节），带
   `code_challenge`（SHA-256 后 base64url）+ `code_challenge_method=S256`；
   token 交换时提交 `code_verifier`。
2. **nonce**：authorize 时发送随机 `nonce`（16 字节）。
3. **服务端会话存储（非 state 信封）**：`code_verifier` 与 `redirect_uri`
   只存服务端内存，OIDC `state` 参数只放不透明会话 id。会话一次性消费 +
   10 分钟 TTL + 过期清理。**PKCE 威胁模型要求 verifier 绝不与授权码同
   信道出现**——把 verifier 编码进 state 会让回调拦截者同时拿到 code 和
   verifier，PKCE 失效。单实例自托管下内存存储即可；进程重启丢失未完成
   登录（用户重试即可）。
4. **public client**：`OIDC_SECRET` 允许为空（PKCE 替代 client secret）。
   若配置了 `OIDC_SECRET`，则用 HTTP Basic（`client_secret_basic`）提交。

其余逻辑（userinfo 映射、redirect 回跳、`@waline` UA 分支）与上游一致。

## 自托管部署（本目录）

```bash
cd deploy/wiki/oauth-center
cp .env.example .env   # OIDC_ID / OIDC_ISSUER 必填
docker compose up -d --build
```

再反向代理 `https://auth.example.com` → `127.0.0.1:8300`（compose 仅回环绑定）。
进程在容器内监听 `0.0.0.0`（docker -p 的流量到达容器 eth0，容器内 loopback
监听会拒收），公网暴露由宿主侧 `127.0.0.1` 绑定保证。

环境变量（与上游相同 + public client 放宽）：

| 变量 | 说明 |
|---|---|
| `OIDC_ID` | Hub `[[oidc.clients]].id` |
| `OIDC_SECRET` | 可空（public client + PKCE）；填写则走 client_secret_basic |
| `OIDC_ISSUER` | Hub OIDC issuer（自动 discovery） |
| `OIDC_SCOPES` | 可选，默认 `openid profile email` |

## 升级上游时重新打补丁

```bash
git clone --depth 1 https://github.com/walinejs/auth.git oauth-center-upstream
diff -u oauth-center-upstream/src/oidc.js src/oidc.js   # 审查差异
```

其余文件（`index.js`、`src/base.js` 等）保持上游原样；`server.js` 是本仓库
自写的 Koa 自托管入口（上游只导出 callback，不 listen）。
