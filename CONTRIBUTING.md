# 贡献指南

感谢你对 YourTJ Hub 的关注。任何形式的贡献——问题反馈、功能建议、文档改进、代码与 Pull Request——都欢迎。

> 本文档是贡献入口；详细的分支与 PR 纪律见 [`docs/development/pull-requests.md`](docs/development/pull-requests.md)，
> 验证命令与测试策略见 [`docs/development/testing.md`](docs/development/testing.md)。

## 项目简介

YourTJ Hub 是同济校园社区平台（品牌 yourtj）：以论坛为核心，提供搜索、统一身份（内建 OIDC Provider）、
内容治理与多端访问；`apps/gooseforum` 是直接演进自 GooseForum 的 Go + Vue 单一二进制论坛。
能力状态、架构与领域边界见 [docs/README.md](docs/README.md)（唯一文档入口与事实源表）。

## 环境准备

- Go 1.26+
- Node.js 24 与 pnpm 11（前端 `apps/gooseforum/resource`，pnpm workspace）
- Flutter（仅移动端 `apps/mobile` 工作）
- Docker Compose（本地依赖：PostgreSQL / Meilisearch，可选）
- 本地 Git 钩子与静态检查（可选但推荐）：

```bash
brew install lefthook golangci-lint
make hooks
```

`make hooks` 为当前 worktree 安装 lefthook 钩子（pre-commit：空白与 gofmt 校验；pre-push：
`go vet` + `golangci-lint` + `pnpm typecheck`）。每个新建的 worktree 都需要重新执行一次。

## 首次构建与验证

```bash
cd apps/gooseforum/resource && pnpm install --frozen-lockfile && cd ../..
make build      # 前端产物 + 单一二进制 bin/yourtj-hub
make test       # 后端 vet+test、契约检查、前端 typecheck+test（全量门禁）
```

常用验证命令与 CI 映射见 [testing.md](docs/development/testing.md)。

## 开发流程

1. 从最新 `origin/dev` 创建分支：`feat/<topic>` / `fix/<topic>` / `docs/<topic>`，PR 目标为 `dev`。
2. 优先使用 worktree 隔离并行任务，不要在一个 checkout 里混多个分支。
3. 提交使用 Conventional Commits（`feat:` / `fix:` / `docs:` / `refactor:` / `chore:` / `test:`），
   只 stage 本任务所属文件。
4. push 前本地门禁通过（或至少明确报告哪些检查未跑）；CI 拥有全量门禁矩阵。
5. 打开 PR：描述动机、行为变更、验证命令与结果、文档/契约影响、已知缺口；不要合并自己的 PR，
   至少一人 review。禁止 `push --force` 到共享分支。

完整纪律见 [pull-requests.md](docs/development/pull-requests.md)。

## 开发规范

- 根 [AGENTS.md](AGENTS.md) 是仓库硬约束与边界规则（单一二进制、数值型用户 ID、契约同 PR、文档状态词等）。
- 任何新功能 PR 必须包含文档变更：用户可见功能更新 docs 中心与状态词；纯内部功能至少更新相关
  README 或代码注释。
- 修 bug 先写最小失败测试（先红后绿），机械改动（重命名/格式化/依赖升级/纯文档）豁免；红测保留为回归测试。
- 代码内 TODO 标记分三档：`FIXME`（阻塞发布）/ `TODO`（近期）/ `XXX`（远期），
  见 [coding-conventions.md](docs/development/coding-conventions.md)。

## 提问与讨论

- 问题反馈与功能建议：[Issues](https://github.com/YourTongji/YourTJ-Hub/issues)
- 较大的改动建议先通过 Issue 对齐问题、用户与验收标准。
