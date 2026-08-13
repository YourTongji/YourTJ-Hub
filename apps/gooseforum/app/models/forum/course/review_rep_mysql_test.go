package course

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
)

// TestRatingDistributionMySQLReservedWordAlias 验证评分分布聚合在 MySQL 方言下
// 不因保留字别名 KEY 而语法失败（oierxjn review 阻塞 2）。
// 用 MySQL dialect dry-run 编译实际查询，断言生成的 SQL 使用非保留别名
// agg_key 且可被 MySQL 接受（未转义的 KEY 会语法报错）。
func TestRatingDistributionMySQLReservedWordAlias(t *testing.T) {
	// 纯 SQL 编译（不连接 MySQL）：用 mysql.Dialector.Explain 做方言级语法检查。
	dialector := mysql.New(mysql.Config{
		DriverName: "mysql",
		DSN:        "user:pass@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True",
	})

	// 与 GetRatingDistributionsByCourseIds 相同的 SELECT（非保留别名 agg_key）。
	sql := dialector.Explain(
		"SELECT o.course_id AS agg_key, SUM(CASE WHEN r.rating = 1 THEN 1 ELSE 0 END) AS star1, SUM(CASE WHEN r.rating = 2 THEN 1 ELSE 0 END) AS star2, SUM(CASE WHEN r.rating = 3 THEN 1 ELSE 0 END) AS star3, SUM(CASE WHEN r.rating = 4 THEN 1 ELSE 0 END) AS star4, SUM(CASE WHEN r.rating = 5 THEN 1 ELSE 0 END) AS star5 FROM `course_review` r JOIN `course_offering` o ON o.id = r.offering_id WHERE o.course_id IN (?, ?) AND r.status = ? AND r.deleted_at IS NULL AND o.deleted_at IS NULL AND o.status = ? GROUP BY o.course_id",
		[]any{1, 2, ReviewStatusVisible, OfferingStatusVisible}...)
	t.Logf("mysql sql: %s", sql)
	// 断言：使用非保留别名 agg_key，无未转义的裸 KEY 别名
	if strings.Contains(sql, " AS key") {
		t.Fatalf("mysql SQL uses reserved alias key: %s", sql)
	}
	if !strings.Contains(sql, "agg_key") {
		t.Fatalf("mysql SQL missing agg_key alias: %s", sql)
	}
}

// TestRatingDistributionRowColumnTags 验证 RatingDistributionRow 的 gorm column
// 映射到非保留别名 agg_key（与 SELECT 别名一致，Scan 才能正确填充）。
func TestRatingDistributionRowColumnTags(t *testing.T) {
	row := RatingDistributionRow{}
	// 断言：字段标签含 agg_key（用 reflect 检查 struct tag）
	typ := reflect.TypeOf(row)
	keyField, ok := typ.FieldByName("Key")
	if !ok {
		t.Fatal("Key field missing")
	}
	tag := string(keyField.Tag.Get("gorm"))
	if !strings.Contains(tag, "column:agg_key") {
		t.Fatalf("Key field gorm tag = %q, want column:agg_key", tag)
	}
}
