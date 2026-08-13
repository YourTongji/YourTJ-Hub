# OAuth Center 部署（walinejs/auth）

向 Waline 提供第三方登录页，背后对接 YourTJ-Hub OIDC（authorization code + PKCE）。

```
Waline (OAUTH_URL) → OAuth Center (walinejs/auth) → YourTJ-Hub OIDC
```

## 核心环境变量

| 变量 | 示例 | 说明 |
|---|---|---|
| `OIDC_ID` | `yourtj-wiki-comment` | 与 Hub `[[oidc.clients]].id` 一致 |
| `OIDC_SECRET` | （留空） | public 客户端不填（PKCE S256）；填写则走 client_secret_basic |
| `OIDC_ISSUER` | `https://forum.example.com/api/oauth` | Hub OIDC issuer（自动 discovery） |
| `OIDC_SCOPES` | （可选） | 默认 `openid profile email` |
| `GITHUB_ID` / `GITHUB_SECRET` | 可选 | 可同时保留 GitHub 登录 |

> `OAUTH_URL` **不属于** walinejs/auth —— 它是 **Waline server 侧**变量，
> 指向本 OAuth Center 的地址（见 `deploy/wiki/waline/docker-compose.yml`）。

## 前置条件

1. Hub 已注册 OIDC client 并开启 `oidc.enabled`（见
   [wiki 部署文档](../../../wiki/docs/deployment/oauth-center-oidc.md)）。
2. Hub `redirect_uris` 与 Waline server 动态构造的回调匹配——现在
   redirect_uris 已支持 glob 兜底（doublestar pattern，见
   `deploy/config.toml.example`），见 wiki 部署文档。

## 部署方式

**方式 A：本目录 compose（推荐，YourTJ-Hub 补丁版，零额外依赖）**

```bash
cd deploy/wiki/oauth-center
cp .env.example .env && 编辑 .env
docker compose up -d --build
```

再反向代理 `https://auth.example.com` → `127.0.0.1:8300`（compose 仅回环绑定）。

**方式 B：Vercel（官方推荐，零运维）**

1. Fork [walinejs/auth](https://github.com/walinejs/auth)（master 分支）。
2. Vercel 导入，配置环境变量后 Deploy。

**方式 C：自托管上游（需自行 wrap Koa app + 打补丁）**

上游 walinejs/auth **不支持 PKCE/nonce**，直接对接 YourTJ-Hub OIDC 会被授权
端点拒绝。仓库 `deploy/wiki/oauth-center/` 已提供补丁版自托管资产（
`src/oidc.js` + `server.js` + `Dockerfile`），补丁内容见 `PATCH.md`。

## 说明

- 真实部署需要公网 HTTPS（OIDC 回调要求 https）。
- 变量名以 [walinejs/auth 仓库](https://github.com/walinejs/auth) 当前 README 为准；
  版本升级后需重新核对。
