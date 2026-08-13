# 快速开始

## 本地开发

```bash
cd wiki
pnpm install
pnpm dev        # http://localhost:5173
```

## 构建与预览

```bash
pnpm build      # 产物输出到 wiki/.vitepress/dist，含 Pagefind 索引
pnpm preview
```

## 新增一篇文档

1. 在 `wiki/docs/` 下新建 `xxx.md`（或放到已有分类目录）。
2. 在 `.vitepress/config.ts` 的 `sidebar` 中登记条目。
3. 提交 PR 到 `dev`，合入后 CF Pages 自动构建发布。

## 评论与搜索

- **评论**：默认关闭（未配置 `VITE_WALINE_SERVER_URL`）。部署时设置
  环境变量指向自托管 Waline 服务即可开启，登录走 Hub OIDC。
- **搜索**：构建时自动生成 Pagefind 索引，站点内无需额外配置。
