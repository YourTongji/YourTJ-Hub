# 飞书 CMS 辅轨

飞书文档作为内容**辅轨**：主轨是 git markdown，辅轨把飞书文档库同步为
`docs/feishu/` 下的 markdown，随站点一起构建发布。

## 同步机制

```
飞书文档/多维表格 ── scripts/sync-feishu.mjs（API 拉取 + 块→Markdown 转换）──> docs/feishu/*.md
                                                                            └─> git commit && push
```

`wiki/scripts/sync-feishu.mjs` 是完整实现（非骨架）：

- **tenant_access_token 获取**：`POST /open-apis/auth/v3/tenant_access_token/internal`，错误处理 + 进程内缓存。
- **docx 文档拉取**：`GET /docx/v1/documents/:doc_id/blocks/:block_id/children` 递归分页
  拉取全部块（含子块，如表格单元格/嵌套列表），实现块→Markdown 转换
  （标题 1-6 / 段落 / 无序·有序列表 / 待办 / 引用 / 高亮块 / 代码块（带语言）/ 分割线 /
  表格（单元格 `|` 转义）/ 图片·文件·表格·流程图等降级为 HTML 注释占位）。
- **多维表格**（可选）：`GET /bitable/v1/apps/:token/tables/:id/records` 分页拉取记录，
  转为 Markdown 表格（字段顺序按出现顺序）。
- **写入**：每文档生成 `<title>.md`（YAML frontmatter：title / source / doc_token / fetched_at），
  并自动更新目录索引 `README.md`。
- **退出码**：成功 0，任何失败非零 + 明确错误信息；成功打印摘要（文档数、字符数）。

## 配置与运行

```bash
cd wiki
cp .env.example .env      # 填写 FEISHU_APP_ID / FEISHU_APP_SECRET / FEISHU_DOC_TOKENS
pnpm sync:feishu          # 真实同步（写入 docs/feishu/）
FEISHU_DRY_RUN=1 pnpm sync:feishu   # 只打印不写文件
```

环境变量：

| 变量 | 必填 | 说明 |
|---|---|---|
| `FEISHU_APP_ID` / `FEISHU_APP_SECRET` | 是 | 自建应用凭据（.env，已 gitignore） |
| `FEISHU_DOC_TOKENS` | 二选一 | 逗号分隔的 docx 文档 token 列表 |
| `FEISHU_BITABLE_APP_TOKEN` + `FEISHU_BITABLE_TABLE_ID` | 二选一 | 多维表格（可选） |
| `FEISHU_OUTPUT_DIR` | 否 | 输出目录（默认 wiki/docs/feishu） |
| `FEISHU_DRY_RUN` | 否 | `1` 时只打印不写文件 |

### 应用权限

飞书开放平台自建应用需开通：

- `docx:document:readonly`（或 `docx:document`）—— 拉取 docx 文档
- `bitable:app:readonly` —— 拉取多维表格（如用 bitable 源）
- `drive:drive:readonly` —— 枚举文件夹（可选，用 FEISHU_DOC_TOKENS 直给 token 时不需要）

## 触发方式（二选一）

**方式 A：GitHub Actions schedule**

```yaml
# .github/workflows/sync-feishu.yml（示例，需在仓库中启用）
on:
  schedule:
    - cron: "0 2 * * *"   # 每天 02:00 UTC
  workflow_dispatch:

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: cd wiki && pnpm install
      - run: cd wiki && pnpm sync:feishu
        env:
          FEISHU_APP_ID: ${{ secrets.FEISHU_APP_ID }}
          FEISHU_APP_SECRET: ${{ secrets.FEISHU_APP_SECRET }}
          FEISHU_DOC_TOKENS: ${{ secrets.FEISHU_DOC_TOKENS }}
      - run: git config user.name "yourtj-bot" && git config user.email "noreply@yourtj.de"
      - run: git add wiki/docs/feishu && git diff --cached --quiet || git commit -m "chore(wiki): sync feishu docs" && git push
```

**方式 B：服务器 crontab**

```bash
# 每 6 小时同步一次
0 */6 * * * cd /opt/yourtj-hub/wiki && pnpm sync:feishu && \
  git add docs/feishu && git diff --cached --quiet || \
  git commit -m "chore(wiki): sync feishu docs" && git push
```

## 说明

- `docs/feishu/README.md` 由脚本自动生成索引，请勿手改。
- 凭据只存 `wiki/.env`（已被 `.gitignore` 忽略），不入 git；CI 走 GitHub Secrets。
- 已知限制：图片/文件/内嵌表格等二进制内容不下载，以 HTML 注释占位（避免把
  敏感内容引入 git）；如需内嵌图片可后续扩展 download 接口。
