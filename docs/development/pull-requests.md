# 分支、提交与 Pull Request

> 文档类型：开发指南
>
> 状态：Active
>
> 负责人：Platform maintainers
>
> 最近核验：2026-08-06

## 分支

- 不在 `main` 上直接开发；从 `origin/main` 建 `feat/<topic>` / `fix/<topic>` / `docs/<topic>`。
- 多任务并行时优先 worktree（`git worktree add`），不要在同一 checkout 混多分支。

## 提交

- Conventional Commits：`feat:` / `fix:` / `docs:` / `refactor:` / `chore:` / `test:`。
- 只 stage 本任务明确拥有的文件；保留无关 dirty/untracked 文件。
- 不 push 到受保护分支；发布走 PR + CI。
- Agent 提交必须带尾注：`Co-authored-by: synergy-agent <299070056+synergy-agent@users.noreply.github.com>`

## Pull Request

- PR 描述说明：动机、行为变化、验证（命令+结果）、文档/契约影响、已知空白。
- 契约变更的 PR 必须包含生成物 diff 与 fixture 更新。
- 不 merge 自己的 PR（除非单人仓库且明确许可）；至少一次 review。

## 禁止

- 不 push --force 到共享分支；不改 git config。
- 不把本地路径、密钥、日志、内部地址写进 commit/PR/评论。
- 不 merge 生产部署或外部变更（除非用户明确要求）。
