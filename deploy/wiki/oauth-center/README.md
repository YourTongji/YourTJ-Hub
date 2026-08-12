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

## ⚠️ 已知风险：redirect_uri 精确匹配

Waline 登录流程中，`redirect_uri` 由 Waline server 动态构造为
`<waline-server>/api/oauth?redirect=<原页面>&type=oidc`（含动态 query 参数）。
而 YourTJ-Hub 的 OIDC provider 基于 zitadel/oidc，`redirect_uri` 校验是
**精确字符串匹配**（`slices.Contains`，不剥离 query、不支持通配）。

这意味着注册 `https://comment.example.com/api/oauth` **无法匹配**带 query 的
动态 redirect_uri。落地第一步必须验证：

- Hub oidcservice 是否注册了自定义 validator（当前未见，需确认）；
- 若严格精确匹配，则需要：为 Hub OIDC 增加 `RedirectURIGlobs` 支持、或
  调整 Waline 的 redirect 构造、或作为已知限制接受（评论登录可能不可用）。

> 此风险在 issue #170 中已列为「落地第一步验证」项，本文档如实记录，
> 供部署时决策。

## 说明

- 真实部署需要公网 HTTPS（OIDC 回调要求 https）。
- 变量名以 [walinejs/auth 仓库](https://github.com/walinejs/auth) 当前 README 为准；
  版本升级后需重新核对。
