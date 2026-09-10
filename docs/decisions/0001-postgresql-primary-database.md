# 主库支持 PostgreSQL，文件库保留 SQLite

## Status
Accepted
Class: architecture

## Context and Problem Statement

- 论坛主库此前仅支持 SQLite（默认）与 MySQL；PostgreSQL 迁移长期处于 `Decision needed`（issue #11）。
- 文件库 `[db.file]`（filedata 附件二进制）是独立的 SQLite 库，无跨库查询需求。
- 模型中有约 112 处 MySQL 方言 gorm type 标签（`bigint unsigned`/`tinyint`/`int unsigned`/`datetime`/`BLOB`），PG 无这些类型，AutoMigrate 会失败。

## Decision Drivers

- 生产部署默认数据库必须是 PostgreSQL。
- 迁移必须同时兼容 SQLite（本地开发/测试）与 PG，且摆脱 MySQL 方言依赖。
- 附件二进制无跨库查询需求，其迁移收益须大于运维成本。

## Considered Options

- 主库新增 PG 支持 + 模型去方言化，文件库保留 SQLite（选定）。
- 主库与文件库全部迁 PG。
- 维持 SQLite 主库 + 继续支持 MySQL 方言标签。

## Decision Outcome

1. 主库 `[db.default]` 新增 `connection="postgres"` 支持（gorm.io/driver/postgres，DSN 走 `url`）。
2. 文件库 `[db.file]` 保留 SQLite——附件二进制元数据无跨库需求，零迁移成本，备份/运维逻辑保留。
3. 模型去方言化：移除 MySQL 专用 type 标签，gorm 按 Go 类型自动映射（uint64→bigint、int8→smallint、time.Time→timestamp、[]byte→bytea），SQLite/MySQL/PG 三方言一致。
4. `sqlconnect` 的 default 分支从"静默回退 sqlite"改为"显式报错"，防止配置拼错悄悄落 sqlite。
5. 不提供 SQLite→PG 自动迁移工具（文档化手动步骤）；不迁移已存在数据。

### Consequences

- Good: PG 部署形态成立；MySQL 事实上退役（仓库明确不支持）。
- Trade-off: PG 部署时 SQLite 快照脚本（`sync-db-from-main.sh` 类）不适用，改用 `pg_dump`/`pg_restore`（已文档化）。
- 文件库保持 SQLite，单二进制部署形态不变；未来若文件库需上 PG，另行决策。

## Pros and Cons of the Options

### PG 主库 + 去方言化 + 文件库留 SQLite（选定）
- Good: 一套模型三方言一致；文件库零迁移。
- Bad: 模型侧需一次性清理约 112 处方言标签。

### 全量迁 PG
- Good: 单一数据库技术栈。
- Bad: 附件库无跨库需求却承担迁移与备份逻辑重写，收益不成比例。

### 维持 SQLite/MySQL
- Good: 零改动。
- Bad: 生产 PG 部署诉求落空；MySQL 方言标签继续阻碍 PG。

## Links

- issue #11
- 迁移手动步骤：[local-development](../development/local-development.md)
