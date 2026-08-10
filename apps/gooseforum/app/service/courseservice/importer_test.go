package courseservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
)

// writeManifestFixture 在临时目录写入 JSONL 数据文件并生成带 sha256 的 manifest。
func writeManifestFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	manifestFiles := make(map[string]string, len(files))
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		sum := sha256.Sum256([]byte(content))
		manifestFiles[name] = hex.EncodeToString(sum[:])
	}
	manifest := fmt.Sprintf("schema_version: 1\nsource: test-fixture\nfiles:\n")
	for name, sum := range manifestFiles {
		manifest += fmt.Sprintf("  %s: %s\n", name, sum)
	}
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return manifestPath
}

func TestValidateRowsQuarantinesDuplicateCodes(t *testing.T) {
	rows := importRows{
		courses: []importCourseRow{
			{ID: "c1", Code: "100001", Name: "高等数学"},
			{ID: "c2", Code: "100001", Name: "重复课号"},
		},
	}
	report := &CatalogImportReport{Errors: []ImportError{}}
	quarantined := validateRows(rows, report)
	if report.Quarantined != 1 {
		t.Fatalf("expected 1 quarantined duplicate, got %d", report.Quarantined)
	}
	if !quarantined[course.EntityTypeCourse+"|c2"] {
		t.Fatal("expected second course row to be quarantined")
	}
}

func TestImportCatalogDryRunOnlyValidates(t *testing.T) {
	manifestPath := writeManifestFixture(t, map[string]string{
		"courses.jsonl":     `{"id":"c1","code":"100001","name":"高等数学(A)上","department":"数学科学学院","credit":5,"aliases":["高数"]}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院"}` + "\n",
		"offerings.jsonl":   `{"id":"o1","course_id":"c1","term":"2025-2026-1","campus":"四平路校区","instructor_ids":["i1"]}` + "\n",
	})

	report, err := ImportCatalog(context.Background(), manifestPath, true)
	if err != nil {
		t.Fatalf("dry-run import: %v", err)
	}
	if report.TotalLines != 3 {
		t.Fatalf("expected 3 total lines, got %d", report.TotalLines)
	}
	if !report.DryRun {
		t.Fatal("expected dryRun report flag")
	}
	if report.Inserted != 0 || report.Updated != 0 {
		t.Fatalf("dry-run must not write: inserted=%d updated=%d", report.Inserted, report.Updated)
	}
	// 校验成功后不允许有任何隔离/错误
	if report.Quarantined != 0 || len(report.Errors) != 0 {
		t.Fatalf("expected clean dry-run, quarantined=%d errors=%v", report.Quarantined, report.Errors)
	}
}

func TestImportCatalogIdempotentAndNoDuplicateOfferings(t *testing.T) {
	conn := dbconnect.Connect()
	models := []any{
		&course.Entity{},
		&course.AliasEntity{},
		&course.TermEntity{},
		&course.OfferingEntity{},
		&course.InstructorEntity{},
		&course.OfferingInstructorEntity{},
		&course.ImportRunEntity{},
		&course.SourceRefEntity{},
		&taskQueue.Entity{},
	}
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate course tables: %v", err)
	}
	// 清空课程域表保证确定性。
	for _, model := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean course table: %v", err)
		}
	}

	manifestPath := writeManifestFixture(t, map[string]string{
		"courses.jsonl":     `{"id":"c1","code":"100001","name":"高等数学(A)上","department":"数学科学学院","credit":5,"aliases":["高数"]}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院"}` + "\n",
		"offerings.jsonl":   `{"id":"o1","course_id":"c1","term":"2025-2026-1","campus":"四平路校区","instructor_ids":["i1"]}` + "\n",
	})

	first, err := ImportCatalog(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if first.Inserted != 3 {
		t.Fatalf("expected 3 inserted rows, got inserted=%d updated=%d", first.Inserted, first.Updated)
	}

	// 第二次导入同一 manifest：manifest 级幂等直接跳过。
	second, err := ImportCatalog(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.Skipped != 1 {
		t.Fatalf("expected manifest-level skip, got skipped=%d", second.Skipped)
	}

	// 断言没有重复 offering（retry 不重复插入）。
	var offeringCount int64
	if err := conn.Model(&course.OfferingEntity{}).Count(&offeringCount).Error; err != nil {
		t.Fatalf("count offerings: %v", err)
	}
	if offeringCount != 1 {
		t.Fatalf("expected exactly 1 offering after idempotent re-import, got %d", offeringCount)
	}
	var courseCount int64
	if err := conn.Model(&course.Entity{}).Count(&courseCount).Error; err != nil {
		t.Fatalf("count courses: %v", err)
	}
	if courseCount != 1 {
		t.Fatalf("expected exactly 1 course, got %d", courseCount)
	}
}
