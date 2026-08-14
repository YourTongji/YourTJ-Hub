# 课评历史数据导入：端到端验证手册（issue #183）

本文档描述上游历史课评数据包（YourTJCourse-Serverless 导出器产物）在 Hub 侧的完整导入验证流程，对应 [issue #183] Phase B/C。

## 数据包格式（上游导出器输出）

`export_legacy_course_package.py` 输出**单包 4 文件 + 1 manifest**（同一目录）：

- `courses.jsonl` / `instructors.jsonl` / `offerings.jsonl` / `reviews.jsonl`
- `manifest.yaml`：`schema_version: 1` + `source` + `source_commit` + `exported_at` + `rights_approval_ref` + `files{sha256}` + `counts`

Hub 导入器按命令消费同一包：`course-import`（catalog）只处理前三个 JSONL，`course-import reviews` 只处理 reviews.jsonl；manifest 中属于其他命令的文件与计数被跳过，由对应命令校验。

## 验证流程（CLI）

```bash
# 1) catalog dry-run：0 冲突、计数与 manifest 一致
go run ./cmd/gooseforum course-import /path/to/package/manifest.yaml --dry-run

# 2) catalog 正式导入：import_run completed（重复执行 manifest 级幂等跳过）
go run ./cmd/gooseforum course-import /path/to/package/manifest.yaml

# 3) reviews dry-run：0 冲突、rights_approval_ref 必填
go run ./cmd/gooseforum course-import reviews --manifest /path/to/package/manifest.yaml --dry-run

# 4) reviews 正式导入（重复执行幂等）
go run ./cmd/gooseforum course-import reviews --manifest /path/to/package/manifest.yaml

# 5) 统计投影重建（rebuild-course-stats 等价命令，成功即一致性）
go run ./cmd/gooseforum rebuild-course-stats
```

## 验证断言

| 环节 | 断言 |
|---|---|
| dry-run | 0 冲突行；manifest.counts 与实际解析行数一致（截断/篡改在半包阶段拒绝） |
| catalog 导入 | `course_import_run` 出现 `kind='catalog'` 且 `status='completed'` 的 run；重复导入 Skipped=1 |
| reviews 导入 | `kind='reviews'` 的 run completed；重复课程代码/重复 offering 被 quarantine 报告（不静默丢弃） |
| 幂等 | 同一 manifest 重复执行两命令均跳过，课程/offering/评价行数不增长 |
| 统计 | `rebuild-course-stats` 成功；目录/详情评分聚合与导入评价一致（`rating 0` 已转 NULL 不计均分） |
| 隐私 | 公开评价 DTO `author.kind='legacy'`、label「历史匿名评价」，不含用户 ID/昵称/头像；`wallet_user_hash` 不出现在任何导出文件 |

## 自动化验证

上述链路已固化为 Go 集成测试（`apps/gooseforum/app/service/courseservice/import_e2e_test.go`）：

```bash
go test ./app/service/courseservice/ -run TestE2ELegacyImport -count=1
```

- `TestE2ELegacyImportFullChain`：dry-run → catalog 导入 → reviews 导入 → 统计重建 → 目录/详情抽查 → 幂等 → 匿名 DTO 零泄露
- `TestE2ELegacyImportQuarantineAmbiguity`：重复课程代码隔离报告

## Phase C（正式导入演练，运维执行）

1. 上游备份库导出完整数据包（`wrangler d1 export jcourse-db-backup`），运维填写 `rights_approval_ref` 后锁包
2. staging 环境按上述流程演练，记录 run 状态与统计一致性
3. 生产导入：catalog → reviews → 统计重建；监控 import_run 与搜索索引（Meilisearch `courses` scope）抽查

[issue #183]: https://github.com/YourTongji/YourTJ-Hub/issues/183
