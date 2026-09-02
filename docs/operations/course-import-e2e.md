# 课评历史数据导入：端到端验证手册（issue #183）

本文档描述上游历史课评数据包（YourTJCourse-Serverless 导出器产物）在 Hub 侧的完整导入验证流程，对应 [issue #183] Phase B/C。

## 数据包格式（上游导出器输出）

> **offering 双源说明（2026 课程沿革）**：本手册描述的 `course-import`（历史数据包
> 导入）与一系统排课物化（`course-pk-sync --materialize`，见
> `docs/operations/deployment.md`「一系统排课同步」）共享同一批 `offering` 行：
> 两源均以教学班为粒度写入，`offering.teaching_class_id` 与 `term` 构成唯一索引，
> 先物化后导入时导入器复用已有行（不重复建卡）。**排课物化是 offering 的权威写入源**
> （管理端「一系统同步」按学期自动物化）；历史数据包导入保持兼容且从属，不写
> `offering.status`（不会复活管理端隐藏的教学班）。导入器生成的 offering 行同样携带
> `teaching_class_id`（数据包提供 `class_code`/`class_name` 时按班号匹配）。

`jcourse_to_manifest.py`（`import-data/`，上游 jcourse SQLite 快照 → manifest 包）输出**单包 4 文件 + 2 manifest**（同一目录）：

- `courses.jsonl` / `instructors.jsonl` / `offerings.jsonl` / `reviews.jsonl`
- `manifest-catalog.yaml`：`schema_version: 1` + `source` + `files{sha256}` + `counts`
- `manifest-reviews.yaml`：同 catalog，另含 `rights_approval_ref`（评价导入必填）

`courses.jsonl` 每行 = 一门课的 **一个 (code, teacher) 身份卡**（`teacher_code` 为该行教师，
对应 `instructors.jsonl` 的 external id；无教师行为空），external id 即上游 `courses` 行 id；
同 (code, teacher_code) 的多行合并为一行（保留 `is_icu=0` 优先、其次 id 小），
同名同院系的历史教师行按工号（`teacherCode`）消歧，工号缺失的合成 `syn-{teacher_id}` 教师行。
`offerings.jsonl` 每行含 `class_code` / `class_name` 班号信息（如 `32000101` / `01班`），
供 Hub 课程详情页按班展示；每个教学班只挂载一门课，互斥挂载链：
(code, teacher) 精确 → (courseCode, teacher) 精确 → code 任意行 → courseCode 任意行，
避免同一教学班同时挂主码课与班号课造成目录双写（旧版多挂载行为已废弃）。
`reviews.jsonl` 按 `reviews.course_id` 行级归因到对应课程卡；有评价但无同教师教学班的
课程行生成教师专属虚拟 offering（`other-{course_ext}`，沿用 `other-*` 机制）承载评价，
保证评价精确落在对应教师卡、全库 `reviews_unmapped=0`。

> **卡片计数说明（issue #326 实测）**：目录卡数 = (code, teacher_code) 组合数，而非
> 上游 (code, teacher_id) 行数——上游存在同名同院系多工号的重复教师记录（如全九清
> 存在 tid 为空的历史行与 `96750` 行），converter 按教师身份合并后
> jcourse-snapshot.db 实测为 **9303 卡**（9518 行 − 1 组 122291 真重复 − 214 组合并），
> 评价 9027 条零丢失、零 unmapped。

Hub 导入器按命令消费同一包：`course-import`（catalog）只处理前三个 JSONL，`course-import reviews` 只处理 reviews.jsonl；manifest 中属于其他命令的文件与计数被跳过，由对应命令校验。

## 验证流程（CLI）

```bash
# 1) catalog dry-run：0 冲突、计数与 manifest 一致
go run ./cmd/gooseforum course-import /path/to/package/manifest-catalog.yaml --dry-run

# 2) catalog 正式导入：import_run completed（重复执行 manifest 级幂等跳过）
go run ./cmd/gooseforum course-import /path/to/package/manifest-catalog.yaml

# 3) reviews dry-run：0 冲突、rights_approval_ref 必填
go run ./cmd/gooseforum course-import reviews --manifest /path/to/package/manifest-reviews.yaml --dry-run

# 4) reviews 正式导入（重复执行幂等）
go run ./cmd/gooseforum course-import reviews --manifest /path/to/package/manifest-reviews.yaml

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
