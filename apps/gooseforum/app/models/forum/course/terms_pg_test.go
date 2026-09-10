package course

import (
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestListDistinctTermsOnPostgreSQL 在真实 PostgreSQL 上验证目录页学期列表查询可执行且排序正确。
//
// 回归背景：该查询曾用 Distinct() 配合 Order("COALESCE(CAST(starts_on AS TEXT), code) DESC")。
// PostgreSQL 不允许 SELECT DISTINCT 的 ORDER BY 使用未出现在投影中的表达式，查询直接报错；
// 而调用方把错误吞掉并降级为空数组（course.go 内 slog.Error 后 terms = []），表现为课程目录页
// 「本学期」快捷筛选整体消失、学期下拉也为空。SQLite 对该写法宽松，所以本地开发与 CI 的
// SQLite 用例全部通过，缺陷只在生产 PostgreSQL 上暴露。
//
// 依赖 YOURTJ_TEST_PG_URL（与 migration / pk / stats 包同一门控），未设置时跳过。
func TestListDistinctTermsOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL term list test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	models := []any{&Entity{}, &TermEntity{}, &OfferingEntity{}}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("AutoMigrate course tables on postgres failed: %v", err)
	}
	for _, model := range models {
		if err := db.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean table on postgres: %v", err)
		}
	}

	// 复现生产库的 code 形态：数字开头的标准学期码 + 一个非数字开头的「其他」
	// （1系统同步时无法识别学期的数据会落到这里）。starts_on 保持 NULL，
	// 与导入流程一致（导入时未写入该字段），排序因此回退到 code。
	codes := []string{"2024-2025-1", "2025-2026-2", "2026-2027-1", "其他"}
	termIDs := make(map[string]uint64, len(codes))
	for _, code := range codes {
		term := TermEntity{Code: code, Name: code, Status: 0}
		if err := db.Create(&term).Error; err != nil {
			t.Fatalf("create term %q: %v", code, err)
		}
		termIDs[code] = term.Id
	}
	c := Entity{PrimaryCode: "PGTERM001", Name: "PG 学期回归课", Department: "测试学院", Status: StatusVisible}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	for _, code := range codes {
		o := OfferingEntity{CourseId: c.Id, TermId: termIDs[code], Status: OfferingStatusVisible}
		if err := db.Create(&o).Error; err != nil {
			t.Fatalf("create offering for %q: %v", code, err)
		}
	}

	terms, err := ListDistinctTermsTx(db.Table(termTableName))
	if err != nil {
		t.Fatalf("ListDistinctTermsTx on postgres failed: %v", err)
	}
	if len(terms) != len(codes) {
		t.Fatalf("expected %d terms, got %d: %+v", len(codes), len(terms), terms)
	}
	// 目录页「本学期」取列表首项：必须是最新学期，而不是字典序更大的「其他」。
	if terms[0].Code != "2026-2027-1" {
		t.Fatalf("first term = %q, want 2026-2027-1 (目录页「本学期」取首项)", terms[0].Code)
	}
	// 非数字开头的学期码排在标准学期码之后。
	if last := terms[len(terms)-1]; last.Code != "其他" {
		t.Fatalf("last term = %q, want 其他（非标准学期码应排末尾）", last.Code)
	}
}
