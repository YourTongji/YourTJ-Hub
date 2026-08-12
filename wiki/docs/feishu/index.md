# 飞书 CMS 辅轨

飞书文档作为内容**辅轨**：主轨是 git markdown，辅轨把飞书文档库同步为
`docs/feishu/` 下的 markdown，随站点一起构建发布。

## 同步机制

```
飞书文档库 ── scripts/sync-feishu.mjs（拉取 + 转 markdown）──> docs/feishu/*.md
                                                                └─> git commit && push
```

同步脚本是**骨架实现**（`wiki/scripts/sync-feishu.mjs`）：

- 环境变量：`FEISHU_APP_ID` / `FEISHU_APP_SECRET` / `FEISHU_FOLDER_TOKEN`
- 不配置时进入 dry-run 演示模式（只打印将同步的文档，并写入占位 markdown）。
- 真实同步需按你的飞书应用权限补齐 API 调用（见脚本内 TODO）。

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
          FEISHU_FOLDER_TOKEN: ${{ secrets.FEISHU_FOLDER_TOKEN }}
      - run: git config user.name "yourtj-bot" && git config user.email "noreply@yourtj.de"
      - run: git add wiki/docs/feishu && git diff --cached --quiet || git commit -m "chore(wiki): sync feishu docs" && git push
```

**方式 B：服务器 crontab**

```bash
# 每 6 小时同步一次
0 */6 * * * cd /opt/yourtj-hub && pnpm --dir wiki sync:feishu && \
  git add wiki/docs/feishu && git diff --cached --quiet || \
  git commit -m "chore(wiki): sync feishu docs" && git push
```

## 说明

- `docs/feishu/README.md` 由脚本自动生成索引，请勿手改。
- 飞书辅轨同步属于「配置 + 骨架」级别，生产接入前需：申请飞书自建应用、
  配置文档权限、在小范围文件夹验证后再放开 cron。
