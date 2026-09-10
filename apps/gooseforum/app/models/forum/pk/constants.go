package pk

// MAX_SQL_VARS SQLite/MySQL 单查询变量上限；IN 条件按此分块（跨方言安全）。
const MAX_SQL_VARS = 80

// PKDataSchemaVersion PK 数据域 schema 版本（issue #185）：随模型结构/派生规则变更递增，
// 与各表的 synced_at 一起标记每行数据的产出版本与最近同步时间，供重建/部分更新判断。
// 思路对齐上游 PK_AUX_SCHEMA_VERSION（'20260605-pk-query-opt-v2'）的"版本号驱动重建"。
const PKDataSchemaVersion = "2026-08-pk-data-v1"
