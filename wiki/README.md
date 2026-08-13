# YourTJ Wiki

同济大学校园社区平台 YourTJ 的知识库站点。

- **框架**：[VitePress](https://vitepress.dev)（静态站，CF Pages 部署）
- **搜索**：[vitepress-plugin-pagefind](https://github.com/ATQQ/sugar-blog/tree/master/packages/vitepress-plugin-pagefind)（离线全文搜索 + 中文切词优化）
- **评论**：[Waline](https://waline.js.org)（自托管，经 [walinejs/auth](https://github.com/walinejs/auth) OAuth Center 接入 YourTJ-Hub OIDC）
- **飞书辅轨**：飞书文档经 `scripts/sync-feishu.mjs` 同步为 `docs/feishu/` markdown

## 本地开发

```bash
pnpm install
pnpm dev          # http://localhost:5173
```

## 构建

```bash
pnpm build        # 产物 .vitepress/dist，含 Pagefind 索引
pnpm preview
```

## 目录结构

```
wiki/
  .vitepress/
    config.ts             # VitePress 配置（含 Pagefind 插件）
    theme/
      index.ts            # 主题入口
      Layout.vue          # 布局（挂载 Waline 评论到 doc-after slot）
  docs/
    index.md              # 首页
    guide/                # 使用指南
    deployment/           # 部署运维（CF Pages / Waline / OAuth Center）
    feishu/               # 飞书同步文档（脚本自动生成索引）
    public/               # 静态资源（srcDir=docs，publicDir 跟随 docs）
      favicon.svg
  scripts/
    sync-feishu.mjs       # 飞书 CMS 同步脚本（完整实现）
  package.json
```

## 部署

- **站点**：Cloudflare Pages（build: `cd wiki && pnpm install && pnpm build`，输出 `wiki/.vitepress/dist`）
- **评论**：需要自托管 Waline + OAuth Center，详见
  [部署文档](docs/deployment/)；未配置 `VITE_WALINE_SERVER_URL` 时评论默认关闭。
- **登录**：评论登录走 YourTJ-Hub OIDC provider（需在 Hub config 注册 client 并开启
  `oidc.enabled`），详见 [OAuth Center 与 Hub OIDC](docs/deployment/oauth-center-oidc.md)。

## 飞书同步

```bash
pnpm sync:feishu   # dry-run（未配置凭据时演示模式）
```

配置 `FEISHU_APP_ID` / `FEISHU_APP_SECRET` / `FEISHU_DOC_TOKENS` 后为真实同步，
详见 [docs/feishu/](docs/feishu/)。
