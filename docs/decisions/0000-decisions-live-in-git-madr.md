# Decisions live in git: MADR log in docs/decisions/

## Status
Accepted
Class: process

## Context and Problem Statement

自 2026-08 起，架构决策记录（ADR）维护在宿主会话的项目 Note（"yourtj-hub ADR note"）中，并明确约定不进 git（docs/README.md 旧表）。引入 repo-seed 治理层（v0.6.2）时，decisions 能力据此注册为 external 指针。

实际后果：决策历史对仓库贡献者不可发现、不可链接；issue/PR review 中引用决策只能靠口头转述；note 与 git 文档之间的交叉引用单向且脆弱；决策格式无门禁可校验。

## Decision Drivers

- 决策历史必须对仓库贡献者可见、可链接、可审查。
- 决策格式可被机械校验（编号连续、章节齐全、supersede 目标存在）。
- 历史记录完整可搬运，迁移成本为一次性。

## Considered Options

- 保持 note 承载 ADR，git 内只留指针（PR #507 初版形态）。
- 决策记录进 git，采用 MADR 格式 + `verify-decisions` 门禁（选定）。
- 决策记录进 git，但沿用 note 的自由格式（无章节约束、无门禁）。

## Decision Outcome

决策记录进入项目 git：`docs/decisions/` 采用 MADR（Markdown Any Decision Records）+ `Class:` 扩展；编号 NNNN 顺序分配、supersede 不改写旧记录；`scripts/verify-decisions.mjs` 门禁校验格式并纳入 `node scripts/run-gates.mjs`。repo-seed 的 decisions 能力从 external 翻转为 enabled，external 指针移除。

历史 note ADR-001~010 全量迁移为本目录 0001~0010（编号保留原语义；0008 为预留编号，按 Blueprint/issue #444 补记 Web Push）；note 冻结为归档，不再新增。

### Consequences

- Good: 决策可发现、可链接、可门禁；新贡献者与 agent 从仓库内即可读到完整决策历史。
- Good: 历史记录保持原内容语义，仅重排为 MADR 章节，结论未被重写。
- Trade-off: 每条决策有七节格式开销，由 verifier 兜底格式质量。
- 升级注意：repo-seed 升级重跑时勿让其样例决策（seed 版 0001-0003）写入本目录——会与本日志编号冲突，gate 会报 duplicate number；如误写入直接删除样例文件。

## Pros and Cons of the Options

### note 承载 + git 指针
- Good: 零迁移成本，写入最轻。
- Bad: 脱离仓库可见性与门禁；跨仓库引用断裂——正是本次要解决的问题。

### git + MADR + 门禁（选定）
- Good: 行业标准格式、MADR 工具可解析；编号/supersede 纪律有机械保障。
- Bad: 记录有固定格式成本。

### git + 自由格式
- Good: 写入最轻。
- Bad: 无章节约束则质量漂移无门禁可依。

## Links

- repo-seed decisions 能力：https://github.com/ericSanchezok/repo-seed
- 治理层引入决策（Agent Note nte_076c3aab0001l2Wdq7UIUwFwIA）
- PR #507（治理层 PR，本决策随该 PR 落地）
