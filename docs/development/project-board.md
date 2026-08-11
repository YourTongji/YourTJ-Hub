# Project Board Workflow / 看板工作流

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-11

把仓库的 issue 与 PR 自动同步到 GitHub Projects v2 看板（RPD 状态机），并说明看板列语义与配置方式。
实现载体：`.github/workflows/sync-project-board.yml` + `.github/scripts/sync-project-board.sh`。

## 1. 看板语义（RPD）

看板列与 GitHub issue/PR 状态一一对应，本工作流按"Status"字段自动流转：

| 列（默认名） | 语义 | 何时进入 |
|---|---|---|
| Todo（需求池） | 收到但尚未排期 | issue 新建 / 重新打开 |
| Planned（计划中） | 已排期，尚未开发 | issue 被打上计划标签（默认 `planning` / `planned` / `ready`） |
| In Progress（开发中） | 正在实现 | PR 新建 / 标记 ready_for_review / 被打标签 |
| Done（已完成） | 交付闭环 | issue 或 PR 被关闭 |

如果你把看板列改成了 R / P / D / 已完成 之类的名字，无需改代码，只需在 workflow 里用变量覆盖列名（见 §4）。

## 2. 架构

```
GitHub 事件 (issue/PR 打开·重开·打标签·关闭) 或手动触发
        │
        ▼
sync-project-board.yml  ──►  sync-project-board.sh（幂等）
                                   ├─ 校验 token / 项目号 / 目标条目
                                   ├─ gh project view     定位项目
                                   ├─ gh api graphql      幂等查重（按 URL 查 item，已存在则复用）
                                   ├─ gh api graphql      首次添加（addProjectV2ItemById）
                                   └─ gh api graphql      设置 Status 字段（updateProjectV2ItemFieldValue）
```

- 使用 `gh` CLI（GitHub Actions 自带的 GitHub CLI，版本 ≥ 2.45）。
- **`GITHUB_TOKEN` 无法访问 Projects**，必须使用专用 token（见 §3）。
- 脚本不解析 GitHub 事件语法，所有上下文通过环境变量传入，因此可本地 mock 测试。

## 3. 首次配置（merge 后一次性）

1. **获取 token**（二选一）：
   - **PAT（个人访问令牌）**：GitHub → Settings → Developer settings → Personal access tokens → *Fine-grained tokens*，新建 token，授予：
     - Repository access：`YourTJ-Hub`
     - Permissions → **Issues: Read**、**Pull requests: Read**、**Projects: Read and write**（若看板挂在组织下，还需组织管理员授予该 token 的项目权限）
   - **GitHub App（组织项目推荐）**：为你的组织创建 App，授予 Organization projects **Read and write**（`Read and write` 的 organization projects 权限），生成私钥；在 workflow 中用 `actions/create-github-app-token@v3` 换取 installation token 后注入 `GH_TOKEN`。
2. **配置 secret**：仓库 Settings → Secrets and variables → Actions → New repository secret：
   - `YOURTJ_PROJECT_TOKEN` = 上面的 token 值（PAT 的完整值，或 GitHub App 生成的 token）。
3. **配置 variables**：
   - `YOURTJ_PROJECT_NUMBER` = 看板项目号（看板 URL 末尾数字，例如 `https://github.com/orgs/YourTongji/projects/5` 的项目号是 `5`）。**必填**。
   - `YOURTJ_PROJECT_OWNER` = 看板属主（不填默认 `YourTongji`）。
4. 触发一次测试：新建一个测试 issue 或在仓库 Actions 页手动运行 `sync-project-board`（`workflow_dispatch`），确认看板出现该条目。

配置完成前，本 workflow 会因未检测到 token 自动跳过，不会报错刷屏。

## 4. 定制列名 / 状态字段

如果看板的 Status 字段或列名不是默认值，在 `.github/workflows/sync-project-board.yml` 的 `env:` 中按需覆盖即可：

```yaml
      STATUS_FIELD: RPD              # 状态字段名，默认 Status
      TODO_VALUE: R                  # 需求列，默认 Todo
      PLANNED_VALUE: P               # 计划列，默认 Planned
      IN_PROGRESS_VALUE: D           # 开发列，默认 In Progress
      DONE_VALUE: 已完成             # 已完成列，默认 Done
      PLANNED_LABELS: planning,planned,ready   # 触发"计划中"的标签
```

## 5. 手动同步

仓库 Actions 页 → `sync-project-board` → *Run workflow*：

- `item_kind`：`issue` 或 `pr`
- `item_number`：要同步的 issue/PR 编号
- `event`：`manual`（默认，issue→需求列 / pr→开发列）、`closed`（已完成）或 `opened` / `ready_for_review`

## 6. 看板内置自动化（可选，双保险）

GitHub Projects 自带 Workflows（看板 Settings → Workflows），可叠加使用：

- 新条目加入看板 → 移到需求列
- 条目被关闭 → 移到已完成列

本仓库工作流与看板内置规则并存时行为一致；二者都做时重复设置无害（本工作流先查重再设置）。

## 7. 故障排查

| 现象 | 原因与处理 |
|---|---|
| job 显示 skipped | 未配置 `YOURTJ_PROJECT_TOKEN` secret，或 token 为空；按 §3 配置 |
| 报错 `无法访问项目 …` | token 缺少 `read:project` 权限，或 `YOURTJ_PROJECT_NUMBER` 填错 |
| 报错 `无法添加条目` | token 缺 `write:project`，或看板属主与 `PROJECT_OWNER` 不一致 |
| 日志警告 `未找到状态字段 …` | 看板的 Status 字段名不是默认名；用 `STATUS_FIELD` 覆盖 |
| 日志警告 `缺少选项 …` | 看板列名与 `TODO_VALUE` 等不匹配；用对应变量覆盖 |
| 未设置状态但条目在 | 字段/选项缺失时脚本只保证添加，不强制设置状态（见日志警告） |

## 8. 边界与已知限制

- 幂等查重通过 GraphQL 分页遍历项目条目（每页 100，按 `content.url` 匹配）；项目条目很多时分页会自动续取，不会漏匹配。
- 关闭的 PR/issue 重新打开（`reopened`）会回到对应状态列。
- 删除看板条目不会反向关闭 issue/PR（单向同步）。
