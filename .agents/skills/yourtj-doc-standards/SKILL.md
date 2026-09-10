---
name: yourtj-doc-standards
description: Use when writing, moving, reviewing, or auditing documentation in the yourtj-hub repository — choosing which docs tier a fact belongs in, applying the four status words, checking fact sources and links, or keeping docs in sync with a code change. Not for implementing code changes (route those to $yourtj-development).
---

# Applying the YourTJ Hub Documentation Standard

文档标准的事实源是 `docs/development/documentation.md` 与 `docs/README.md`；本 skill 提供执行工作流。
指导而非脚本。

## Sources of truth (read, don't re-summarize)

- `docs/README.md` — 事实源表、四状态词、文档生命周期（Active/Draft/Deprecated）
- `docs/development/documentation.md` — 文档治理：同 PR 更新、只描述当前模型、状态词、变更流程
- 根 `AGENTS.md` §3 — 「新功能必建文档」硬约束

不建 `docs/AGENTS.md` 或子目录 AGENTS.md：文档标准已由 documentation.md 承担，再建即双源。

## Before writing: choose the home

1. 用 docs/README.md 事实源表定位归属：产品行为 → `docs/product/`；安全/隐私 → AGENTS.md（直到
   `docs/security/` 存在）；HTTP 结构 → `apps/gooseforum/app/http/controllers` + `packages/api-contract/openapi.yaml`；
   DB 结构 → `apps/gooseforum/app/migration/`；开发/测试/PR 流程 → `docs/development/`；部署与运维 → `docs/operations/`。
2. 文档只描述当前支持模型：无时间线、里程碑、PR 交付清单、历史叙事（git history 拥有归档）。
3. 状态词只用于可验证的具体行为，不用于整个领域：`Current` / `Partial` / `Planned` / `Decision needed`。
   文档自身生命周期用 `Active` / `Draft` / `Deprecated`，与实现状态分离。
4. 移动/重命名文档前 grep 入站引用（Markdown 链接与 #fragment、代码注释中的 `docs/*.md` 引用）；
   移动是原子的：旧家删除、新家添加、所有入站链接同一次改动修复。

## Keeping docs in sync with code

- 任何改变产品行为、契约、schema、安全边界或部署的 PR 必须同 PR 更新受影响的文档；不同步即缺陷。
- 用户可见功能：更新 docs 中心 + 状态词。纯内部功能：至少更新相关 README 或代码注释（最小集）。
- 代码是权威：契约变化时更新所属文档，不保留历史版本的平行叙事。
- 事实源冲突时按缺陷处理：同 PR 修复契约、实现、测试与相关文档，或显式记录为 `Partial`。

## Auditing docs

- 用最便宜的探针开始：grep 独特短语找重复（保留一处，其余改链接）；检查状态词是否被误用于整个领域；
  检查是否出现 PR-relative 标签（"shipped this" / "later"）。
- 找 reasoning-transcript 泄漏：叙述历史、死亡设计会话引用、评审编排、控制流叙述、测试 walkthrough——
  只保留非显然的契约或持久理由；同一理由重复出现时保留一个家。
- 手写目录/测试状态清单/API 复述 → 替换为权威树、脚本或生成引用。
- 删陈旧内容而不是保留 "deprecated but useful" 副本；git history 拥有归档。

## Validation

- `git diff --check`；grep 确认所有新增/移动文档的入站链接指向存在文件（仓库暂无 verify-md-links 门禁时手动核对）。
- 确认状态词拼写与四词一致；确认 docs 索引（`docs/README.md`、`docs/development/README.md`）覆盖新增文档。
- 报告：文档变更清单、链接核对结果、状态词更新点。
