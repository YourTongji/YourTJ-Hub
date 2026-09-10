# Decision log

Every durable choice with meaningful alternatives — architecture or process — lives in one place: this directory. Architecture and process decisions share the same format, lifecycle, and verifier. Specs own change contracts; commits own change history.

## Format

The format is **MADR (Markdown Any Decision Records)** with a documented `Class:` extension. Names and structure follow the industry standard so existing MADR tooling keeps working; the extension is ignored by tools that do not know it.

A decision record is a file named `NNNN-title.md` (4-digit zero-padded number, `-`, kebab-case title) directly inside this directory. Numbers are assigned sequentially; the number is the record's identity and never changes.

### Required sections

Every record contains exactly these `##` sections, in this order:

1. `## Status`
2. `## Context and Problem Statement`
3. `## Decision Drivers`
4. `## Considered Options`
5. `## Decision Outcome`
6. `## Pros and Cons of the Options`
7. `## Links`

`## Links` may be empty ("None.") but must exist. Additional `##` sections may follow `## Links` when a record needs them.

### Status line

The first non-empty line under `## Status` is the status value. Valid values:

- `Proposed` — the decision is being considered; not yet shipped.
- `Accepted` — the decision shipped; the record describes what is.
- `Rejected` — considered and declined; kept while its rationale prevents a tempting, meaningful mistake.
- `Deprecated` — no longer recommended; superseded by a linked record.
- `Superseded by [NNNN](NNNN-title.md)` — replaced by a newer record; the target must exist.

A record whose status is `Accepted`, `Deprecated`, or `Superseded by …` describes current or frozen reality. It is never rewritten into a different decision: to change a decision, add a new record and mark the old one `Superseded by NNNN`. Both records stay, cross-linked.

### Class line (extension)

The line immediately after the status value may be `Class: <value>`. It classifies the decision:

| Class | Covers |
|---|---|
| `architecture` | Structure of the shipped source: modules, boundaries, runtime vocabulary |
| `process` | Tooling, policy, workflow around the code: gates, package manager, conventions |
| `testing` | Test infrastructure and strategy |
| `feature` | A new user- or model-facing capability |
| `bug-fix` | Corrects a defect or closes a gap a postmortem surfaced |
| `simplification` | Removes code, behavior, or surface area without adding capability |

A missing `Class:` line is valid (it is an extension); an invalid value is a violation.

## Lifecycle

A decision starts `Proposed`. Once implemented, its status becomes `Accepted` and the record is kept current with what actually shipped (facts only — names, paths, structure — not the decision itself). A declined proposal is `Rejected`; delete a rejected record only when its rationale no longer prevents a plausible mistake. An obsolete accepted decision becomes `Superseded by NNNN` or `Deprecated`; never edit it into its opposite.

## Writing rules

- Create or update a record when a change chooses among meaningful alternatives and future maintainers may reasonably revisit the rationale. Do not create a decision record for routine implementation, mechanical refactors, obvious fixes already defined by regression tests, or work that merely follows an accepted decision; risk-boundary feature behavior belongs in a spec and change history belongs in commits.
- State the decision, what it beats, and what it gives up. `## Considered Options` lists genuine alternatives; `## Pros and Cons of the Options` records why the losers lost. A decision without its alternatives invites re-litigation.
- When a record relies on external evidence or an implementation source, cite it descriptively in `## Links` with a stable URL, DOI, or versioned permalink. Do not leave research or quantitative claims unlinked.
- Cross-reference records with relative Markdown links (`[0001](0001-title.md)`), never bare numbers.
- Document current reality, not change history. Put change stories in commits; the decision record states the live contract.

## Verifier

`scripts/verify-decisions.mjs` enforces: file naming, unique sequential numbering, required sections, valid status values, valid `Class:` values when present, and `Superseded by NNNN` targets that exist. It exits non-zero on any violation. `scripts/run-gates.mjs` selects it from the manifest.

## Index

- [0000](0000-decisions-live-in-git-madr.md) — 决策记录进 git（本日志的建制决策，取代「ADR 存 note」旧约定）
- [0001](0001-postgresql-primary-database.md) — 主库支持 PostgreSQL，文件库保留 SQLite，模型去方言化
- [0002](0002-jwt-session-revocation-user-sessions.md) — 会话吊销：JWT jti + user_sessions 表
- [0003](0003-aggregate-search-multi-index-pinyin.md) — 聚合搜索：候选 ID 索引 + DB 重构 + 自建拼音 + 三域分组
- [0004](0004-wiki-single-event-source-write-publish.md) — Wiki 修订单一事件源 + 写即发布（已被 0005 取代）
- [0005](0005-wiki-github-ssot-pr-collaboration.md) — Wiki GitHub 唯一真实源 + PR 协作编辑
- [0006](0006-deploy-config-as-artifact.md) — 部署配置治理：config as artifact + Environments secrets + 漂移守卫
- [0007](0007-dev-instance-no-env-isolation.md) — dev 实例 DB 站点设置无环境隔离（已知边界）
- [0008](0008-web-push-second-notification-channel.md) — Web Push + Service Worker 第二通知通道
- [0009](0009-image-egress-internal-proxy.md) — 图片流量成本模型：服务端内网代理读
- [0010](0010-mobile-route-a-native-alignment.md) — 移动端战略 Route A：原生 Flutter 全功能对齐（导航与管理范围由 0011 更新）
- [0011](0011-mobile-navigation-and-management.md) — 移动端导航与完整管理工作台
- [0012](0012-unified-mobile-reading-navigation.md) — Unified mobile reading navigation and shared management.
- [0013](0013-anonymous-wiki-comments.md) — Wiki 页面评论匿名发布（匿名楼层）
- [0014](0014-mobile-release-distribution.md) — Mobile release signing, verified APK mirrors and Apple review automation.

- [0015](0015-oryn-repository-local-actions.md) — Repository-local Oryn Actions with scoped GitHub App credentials.
- [0017](0017-native-push-providers.md) — Direct APNs and Android OEM push through JPush.
