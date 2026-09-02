# Test coverage — 示例与注解

> 本文件为 `$yourtj-code-review` 的 **Test coverage** 规则提供通用示例与注解。
> 片段为示意，**不绑定具体仓库文件路径**（避免随代码演进/重构失效）；
> 评审时对照片段理解规则意图，再在改动中找等价的实际用例与断言。

## 1. 分支/错误返回/回退路径都要有断言

规则：不止快乐路径；条件、错误返回、回退分支都要有断言；"这段代码不会走到"不是理由。

```go
// 反例：只断言成功路径，错误/回退分支实际未执行
func TestBackfill(t *testing.T) {
    err := backfill(db)
    require.NoError(t, err)
}

// 正例：错误返回有断言，且断言失败后的状态（部分成功/回滚/自愈）
func TestBackfill(t *testing.T) {
    err := backfill(db)
    require.ErrorContains(t, err, "backfill failed")
    // 断言失败后数据未污染：已处理行保留、未处理行保持原值
    var done int64
    db.Model(&row{}).Where("flag = ?", doneFlag).Count(&done)
    assert.Equal(t, expectedDone, done)
}
```

注解：断言"失败时的状态"比断言"报错"更有价值——迁移/批量任务即使报错，
也要验证可重入（幂等重跑不会重复或损坏数据）。

## 2. 边界与量级敏感点要显式测

规则：分页/批处理跨越批大小边界；游标/断点续跑、空输入、首尾元素、幂等重入、并发冲突。

```go
const batchSize = 500
const total = 1100 // > batchSize：强制走 keyset/分页续跑分支
// seed total 行 → 分批处理会触发"批次打满后按主键续跑"
// 断言跨批边界行（batchSize-1 / batchSize / batchSize+1 / 末尾）处理正确
```

注解：只测小于批大小的样本时，keyset 分页分支完全不执行；真实行数一旦超过批大小
就会走到未测代码（曾因此出现生产启动崩溃）。

## 3. 失败路径可观测；重试与自愈有测试

```go
// 正例：迁移失败不推进版本 → 下次启动重跑；重跑幂等
func TestMigrationFailureDoesNotAdvanceVersion(t *testing.T) {
    err := runMigration(db)
    require.Error(t, err)
    assert.Equal(t, oldVersion, currentVersion(db)) // 版本未推进
    // 修复后重跑成功，且不重复回填已处理行
    require.NoError(t, runMigration(db))
}
```

注解：error 语义要明确（hard 失败 vs deferred/稍后重试），并断言"重试/自愈路径"而不仅是看日志。

## 4. 数据量与方言贴近真实

规则：真实可达数万行的表，小样本会漏掉只在量级/方言触发的缺陷；SQLite 宽松 ≠ PostgreSQL 严格。

```sql
-- 反例语义：SQLite 放行，PostgreSQL 拒绝
-- SELECT DISTINCT ... ORDER BY <表达式>（PG 要求 ORDER BY 表达式必须出现在 select list）
```

```go
// 正例：跨方言查询有目标方言（PostgreSQL）回归用例在真实方言上执行，不只 SQLite；
// 大样本回归可用 sqlite :memory: 低代价复现"量级触发"类缺陷
```

注解：按部署默认方言验证；默认 SQLite 通过的查询不代表 PostgreSQL 上也通过。