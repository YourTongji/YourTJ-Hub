# Test coverage — 示例与注解

> 本文件为 `$yourtj-code-review` 的 **Test coverage** 规则提供通用示例与注解。
> 评审时对照片段理解规则意图，再在改动中找等价的实际用例与断言。

## 1. 分支/错误返回/回退路径都要有断言

规则：不止快乐路径；条件、错误返回、回退分支都要有断言。

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

断言"失败时的状态"比断言"报错"更有价值——迁移/批量任务即使报错，
也要验证可重入（幂等重跑不会重复或损坏数据）。

## 2. 边界与量级敏感点要显式测

规则：分页/批处理跨越批大小边界；游标/断点续跑、空输入、首尾元素、幂等重入、并发冲突。

```go
// 正例：数据量超过批大小，跨批边界的行都要有断言
func TestBatchBoundary(t *testing.T) {
    const batchSize = 500
    const total = 1100 // > batchSize：触发"批次打满后按主键 keyset 续跑"分支
    seedRows(t, db, total) // 先构造总行数超批大小的真实输入

    err := backfillInBatches(db, batchSize)
    require.NoError(t, err)

    // 跨批边界行（首行、批内末行、新批首行、总体末行）逐行断言，不只抽查
    for _, id := range []int{0, 498, 499, 500, 501, 1099} {
        var got Row
        require.NoError(t, db.First(&got, id).Error)
        assert.Equal(t, wantValue(id), got.Value, "row %d", id)
    }

    // 幂等重入：同批再跑一遍，结果不变、不重复写
    require.NoError(t, backfillInBatches(db, batchSize))
    assert.Equal(t, int64(total), processedCount(t, db))
}
```

只测小于批大小的样本时，keyset 分页分支完全不执行；真实行数一旦超过批大小
就会走到未测代码。

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

error 语义要明确（hard 失败 vs deferred/稍后重试），并断言"重试/自愈路径"而不仅是看日志。

## 4. 数据量与方言贴近真实

规则：真实可达数万行的表，小样本会漏掉只在量级/方言触发的缺陷；SQLite 宽松 ≠ PostgreSQL 严格。

```go
// 反例：只在默认方言断言"不报错"，方言与量级分支都没走到
//   （SELECT DISTINCT ... ORDER BY <表达式>：PG 要求 ORDER BY 表达式必须出现在
//   select list，SQLite 不报错；且未造大样本，量级分支也未执行）
func TestListDistinctTermsPartial(t *testing.T) {
    db := newSQLiteDB(t)
    terms, err := listDistinctTerms(db)
    require.NoError(t, err) // SQLite 放行；同一查询在 PostgreSQL 上会报错
    assert.Greater(t, len(terms), 0)
}

// 正例：方言与量级分别兜住——目标方言真回归 + 大样本低代价回归
func TestListDistinctTermsTargetDialect(t *testing.T) {
    // ① 部署默认方言（PostgreSQL）：在真实方言上执行，SQLite 放行不代表 PG 通过
    if dsn := os.Getenv("YOURTJ_TEST_PG_URL"); dsn != "" {
        db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
        require.NoError(t, err)
        terms, err := listDistinctTerms(db)
        require.NoError(t, err) // PG: ORDER BY 表达式必须在 select list，否则报错
        assert.Equal(t, wantOrder, orderOf(terms))
    }

    // ② 量级：小样本测不出"行数超过阈值才走的分支"，用 sqlite :memory: 造大样本
    db := newSQLiteDB(t)
    seedRows(t, db, 1100) // 超过典型批大小 500，触发 keyset 分页等量级分支
    require.NoError(t, listDistinctTerms(db))
}
```

按部署默认方言验证；默认 SQLite 通过的查询不代表 PostgreSQL 上也通过。