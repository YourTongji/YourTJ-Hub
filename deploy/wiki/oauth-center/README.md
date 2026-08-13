# OAuth Center（walinejs/auth）部署

Waline 评论登录的 OIDC 中转服务：向 Waline 提供第三方登录页，背后对接
YourTJ-Hub 的 OIDC provider（authorization code + PKCE）。

```
Waline (OAUTH_URL) → OAuth Center (walinejs/auth) → YourTJ-Hub OIDC
```

## 部署方式

**walinejs/auth 官方没有 Dockerfile / docker-compose**，官方部署方式是
Vercel（`vercel.json` + Deploy with Vercel 按钮）。两种自托管路线：

**方式 A：Vercel（官方推荐，零运维）**

1. Fork [walinejs/auth](https://github.com/walinejs/auth)（master 分支）。
2. Vercel 导入，配置环境变量后 Deploy。

**方式 B：自托管（Docker，需自行 wrap Koa app）**

仓库导出 `app.callback()`（Koa 中间件，无 `listen`），需自写入口：

```js
// server.js —— 自托管入口（示例，需自行维护）
const http = require('http');
const callback = require('@waline/auth'); // 或指向本地 clone 的 index.js
http.createServer(callback).listen(process.env.PORT || 3000);
```

```dockerfile
# Dockerfile（示例）
FROM node:20-alpine
WORKDIR /app
COPY . .
RUN npm install --omit=dev
EXPOSE 3000
CMD ["node", "server.js"]
```

> 说明：`@waline/auth` 是私有包（`"private": true`），npm 上不可直接安装，
> 自托管需 clone 源码（`git clone https://github.com/walinejs/auth`）。

## 核心环境变量（walinejs/auth）

| 变量 | 示例 | 说明 |
|---|---|---|
| `OIDC_ID` | `yourtj-wiki-comment` | 与 Hub `[[oidc.clients]].id` 一致 |
| `OIDC_SECRET` | （留空） | public 客户端不填（PKCE S256） |
| `OIDC_ISSUER` | `https://forum.example.com/api/oauth` | Hub OIDC issuer（自动 discovery） |
| `OIDC_SCOPES` | （可选） | 默认 `openid profile email` |
| `GITHUB_ID` / `GITHUB_SECRET` | 可选 | 可同时保留 GitHub 登录 |

> `OAUTH_URL` **不属于** walinejs/auth —— 它是 **Waline server 侧**变量，
> 指向本 OAuth Center 的地址（见 `deploy/wiki/waline/docker-compose.yml`）。

## 前置条件

1. Hub 已注册 OIDC client 并开启 `oidc.enabled`（见
   [wiki 部署文档](../../../wiki/docs/deployment/oauth-center-oidc.md)）。
2. Hub `redirect_uris` 与 Waline server 动态构造的回调匹配（**落地第一步验证**，
   见下方风险说明）。

## ✅ redirect_uri 匹配已支持 glob（本 PR 落地）

Waline 登录流程中 `redirect_uri` 会携带动态 query 参数，而 Hub OIDC 曾只有
精确字符串匹配。现在 Hub 侧已实现 zitadel/oidc 的 `HasRedirectGlobs` 接口：
每个 client 可配置 `redirect_uris_globs`（doublestar pattern），精确匹配失败后
按 pattern 兜底匹配。配置示例见
[wiki 部署文档](../../../wiki/docs/deployment/oauth-center-oidc.md)：

```toml
[[oidc.clients]]
id = "yourtj-wiki-comment"
redirect_uris = ["https://auth.example.com/api/oauth/redirect"]
redirect_uris_globs = ["https://auth.example.com/api/oauth/redirect*"]
```

- pattern 在配置加载时校验，非法 pattern 拒绝整个 OIDC 配置（fail-closed）；
- glob 只兜底注册过的 pattern，未命中的 URI 依然被拒绝，不构成 open redirect；
- 落地第一步仍建议验证实际回调串与 pattern 匹配（见下方说明）。

## 说明

- 真实部署需要公网 HTTPS（OIDC 回调要求 https）。
- 变量名以 [walinejs/auth 仓库](https://github.com/walinejs/auth) 当前 README 为准；
  版本升级后需重新核对。
