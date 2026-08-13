# Cloudflare Pages 发布

## 构建配置

在 CF Pages 项目中（或 `wrangler.toml`）配置：

| 项 | 值 |
|---|---|
| Build command | `cd wiki && pnpm install && pnpm build` |
| Build output directory | `wiki/.vitepress/dist` |
| Node version | 20+ |
| Package manager | pnpm（CF Pages 自动识别 `pnpm-lock.yaml`） |

## 环境变量

| 变量 | 说明 |
|---|---|
| `VITE_WALINE_SERVER_URL` | Waline 评论服务地址。**不设置则整站无评论**（默认关闭）。 |

## wrangler.toml 示例

```toml
name = "yourtj-wiki"
compatibility_date = "2024-11-01"
pages_build_output_dir = "wiki/.vitepress/dist"

[build]
command = "cd wiki && pnpm install && pnpm build"
```

## 说明

- 构建产物内含 Pagefind 索引（`/pagefind/`），发布后搜索即可用。
- 评论组件在未配置 `VITE_WALINE_SERVER_URL` 时完全不渲染（`Layout.vue` 按环境变量判断），因此**评论服务未就绪前发布站点不会报错**。
- 安卓端中文搜索需实测（Intl.Segmenter 兼容性，见 [搜索说明](../guide/content)）。
