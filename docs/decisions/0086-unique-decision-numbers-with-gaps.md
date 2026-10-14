# 编号是唯一标识：允许空号

## Status
Accepted
Class: process

## Context and Problem Statement

编号机制随本日志一同建立（[0000](0000-decisions-live-in-git-madr.md)）：编号在合并后的目录中从 0 连续，`scripts/verify-decisions.mjs` 以机械校验保证连续性。并行决策 PR 的实践（#1157、#1150、#1171、#1143）暴露了该规则的代价：编号隐含合并顺序，互不相关的 PR 必须排队——后一个 PR 的 ci-docs 要等前一个合并、编号变连续后才可能通过；贡献者要么猜测后续可用编号，要么在他人 PR 之后重排自己的记录。

## Decision Drivers

- 多个决策 PR 必须能独立并行准备与合并；编号不得隐含合并顺序。
- 编号是记录的持久身份：一旦分配永不改变，重排会破坏既有交叉引用与外部链接。
- 目录仍须拒绝两条记录争用同一编号（唯一性），其余格式门禁保持不变。
- 门禁只校验单棵检出树上可判定的属性；跨分支的连续性无法在单树内判定，不应假装校验。

## Considered Options

- 维持现状：编号必须连续，决策 PR 按编号顺序排队合并。
- 编号唯一、允许空号；预留登记在 PR 或关联 Issue，verifier 只拒绝重复编号（选定）。
- 放弃数字编号，改用日期前缀或纯 slug 命名。

## Decision Outcome

编号是唯一标识：`docs/decisions/` 中的文件编号必须唯一，允许空号；既有编号不重排、不用占位文件填充。贡献者按 [README](README.md) 的预留流程在 PR 或关联 Issue 中登记未使用编号；冲突时以最早公开的预留为准，顺序不明确时由维护者在相关 PR 合并前裁定。

`scripts/verify-decisions.mjs` 删除连续性检查，保留并加强其余校验：重复编号、必需章节与顺序、状态值、`Class:` 扩展值，以及 `Superseded by` 引用（新建记录使用编号与目标文件名一致的链接形式；仅当目标编号对应文件存在时兼容精确的旧式 `Superseded by NNNN`）。[0000](0000-decisions-live-in-git-madr.md) 中「编号连续、顺序分配」的机制表述由本记录更新，其正文不改写。

本记录本身演示该流程：编号 0086 在 PR #1179 中预留，与 0083（#1150）、0084（#1171）、0085（#1143）互不冲突。

## Pros and Cons of the Options

### 编号唯一、允许空号（选定）

- Good: 解除 PR 之间的合并排序耦合；编号身份稳定，交叉引用与外部链接不受影响。
- Bad: 索引可能出现空号；跨分支的预留冲突需要人工协调（verifier 无法校验未合并分支中的预留）；需要预留与裁定规则作为协作补充。

### 编号必须连续

- Good: 目录编号紧凑，连续性可机械校验。
- Bad: 每个并行决策 PR 都被迫按编号排队；verifier 会把「尚未合并的预留空号」误报为违规，实际代价由所有多 PR 并行的工作承担。

### 日期前缀或纯 slug 命名

- Good: 完全消除编号冲突面。
- Bad: 丢失 MADR 生态的稳定数字引用习惯；既有 0xxx 编号与全部交叉引用需要迁移，改动面远超收益。

## Links

- [0000](0000-decisions-live-in-git-madr.md) — 本日志的建制决策，编号机制由本记录更新
- [README](README.md) — 预留、冲突裁定与索引规则
- [PR #1179](https://github.com/YourTongji/YourTJ-Hub/pull/1179) — 本记录随该 PR 落地，verifier 与文档在该 PR 内修改
