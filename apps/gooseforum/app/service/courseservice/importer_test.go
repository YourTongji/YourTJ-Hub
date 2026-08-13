package courseservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// writeManifestFixture 在临时目录写入 JSONL 数据文件并生成带 sha256 的 manifest（source 固定 test-fixture）。
func writeManifestFixture(t *testing.T, files map[string]string) string {
	return writeManifestFixtureWithSource(t, "test-fixture", files)
}

// writeManifestFixtureWithSource 同 writeManifestFixture，但指定 manifest.source，
// 用于多来源隔离测试。
func writeManifestFixtureWithSource(t *testing.T, source string, files map[string]string) string {
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
	manifest := fmt.Sprintf("schema_version: 1\nsource: %s\nfiles:\n", source)
	for name, sum := range manifestFiles {
		manifest += fmt.Sprintf("  %s: %s\n", name, sum)
	}
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return manifestPath
}

// writeManifestFixtureWithCounts 同 writeManifestFixture，但附加 manifest.counts
// （文件名 -> 期望行数），用于计数一致性校验测试。
func writeManifestFixtureWithCounts(t *testing.T, files map[string]string, counts map[string]int) string {
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
	manifest := fmt.Sprintf("schema_version: 1\nsource: test-fixture\ncounts:\n")
	for name, count := range counts {
		manifest += fmt.Sprintf("  %s: %d\n", name, count)
	}
	manifest += "files:\n"
	for name, sum := range manifestFiles {
		manifest += fmt.Sprintf("  %s: %s\n", name, sum)
	}
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return manifestPath
}

// TestImportCatalogValidatesManifestCounts manifest.counts 与实际解析行数一致时
// dry-run 通过；不一致时（截断/篡改）在 dry-run 阶段即拒绝，不写库。
func TestImportCatalogValidatesManifestCounts(t *testing.T) {
	files := map[string]string{
		"courses.jsonl":     `{"id":"c1","code":"100001","name":"高等数学(A)上"}` + "\n" + `{"id":"c2","code":"100002","name":"线性代数"}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院"}` + "\n",
		"offerings.jsonl":   `{"id":"o1","course_id":"c1","term":"2025-2026-1"}` + "\n",
	}
	// 计数一致：dry-run 通过。
	ok := writeManifestFixtureWithCounts(t, files, map[string]int{
		"courses.jsonl": 2, "instructors.jsonl": 1, "offerings.jsonl": 1,
	})
	report, err := ImportCatalog(context.Background(), ok, true)
	if err != nil {
		t.Fatalf("dry-run with matching counts: %v", err)
	}
	if report.TotalLines != 4 {
		t.Fatalf("expected 4 total lines, got %d", report.TotalLines)
	}
	// 计数不一致：拒绝并给出文件级原因，且不写库。
	bad := writeManifestFixtureWithCounts(t, files, map[string]int{
		"courses.jsonl": 1, "instructors.jsonl": 1, "offerings.jsonl": 1,
	})
	if _, err := ImportCatalog(context.Background(), bad, true); err == nil {
		t.Fatal("expected counts mismatch error on dry-run")
	} else if !strings.Contains(err.Error(), "counts mismatch") {
		t.Fatalf("expected counts mismatch error, got %v", err)
	}
	// 多文件部分不匹配：3 个文件中仅 1 个计数错误，其余正确，
	// 同样必须整体拒绝（防止"部分匹配"被误判通过）。
	partialBad := writeManifestFixtureWithCounts(t, files, map[string]int{
		"courses.jsonl": 2, "instructors.jsonl": 1, "offerings.jsonl": 9,
	})
	if _, err := ImportCatalog(context.Background(), partialBad, true); err == nil {
		t.Fatal("expected counts mismatch error when only one of multiple files mismatches")
	} else if !strings.Contains(err.Error(), "offerings.jsonl") {
		t.Fatalf("expected error to name the mismatching file, got %v", err)
	}
	// 计数引用了 manifest 中不存在的文件：同样拒绝。
	badFile := writeManifestFixtureWithCounts(t, files, map[string]int{
		"courses.jsonl": 2, "instructors.jsonl": 1, "offerings.jsonl": 1, "reviews.jsonl": 1,
	})
	if _, err := ImportCatalog(context.Background(), badFile, true); err == nil {
		t.Fatal("expected error for counts of unknown file")
	}
	// 真实导入同样拒绝，确保半包不会被静默导入。
	if _, err := ImportCatalog(context.Background(), bad, false); err == nil {
		t.Fatal("expected counts mismatch error on real import")
	}
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

// TestImportCatalogIsolatesSources 同一 external id 从两个不同来源导入互不覆盖 source_ref：
// 每个来源保留独立映射与 checksum，来源 A 的课程内容不被来源 B 覆盖。
func TestImportCatalogIsolatesSources(t *testing.T) {
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
	for _, model := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean course table: %v", err)
		}
	}
	manifestA := writeManifestFixtureWithSource(t, "source-a", map[string]string{
		"courses.jsonl": `{"id":"c1","code":"100001","name":"高等数学(A)上"}` + "\n",
	})
	if _, err := ImportCatalog(context.Background(), manifestA, false); err != nil {
		t.Fatalf("import source-a: %v", err)
	}
	// 来源 B 用同一 external id 但不同主课号：应创建独立课程，而非覆盖来源 A 的映射。
	manifestB := writeManifestFixtureWithSource(t, "source-b", map[string]string{
		"courses.jsonl": `{"id":"c1","code":"200002","name":"线性代数"}` + "\n",
	})
	if _, err := ImportCatalog(context.Background(), manifestB, false); err != nil {
		t.Fatalf("import source-b: %v", err)
	}
	var courseCount int64
	if err := conn.Model(&course.Entity{}).Count(&courseCount).Error; err != nil {
		t.Fatalf("count courses: %v", err)
	}
	if courseCount != 2 {
		t.Fatalf("expected 2 isolated courses, got %d", courseCount)
	}
	var refCount int64
	if err := conn.Model(&course.SourceRefEntity{}).Count(&refCount).Error; err != nil {
		t.Fatalf("count source refs: %v", err)
	}
	if refCount != 2 {
		t.Fatalf("expected 2 source refs (one per source), got %d", refCount)
	}
	// 两个来源的映射指向不同课程，来源字段各自正确。
	var refA, refB course.SourceRefEntity
	if err := conn.Model(&course.SourceRefEntity{}).
		Where("source = ? AND entity_type = ? AND external_id = ?", "source-a", course.EntityTypeCourse, "c1").
		First(&refA).Error; err != nil {
		t.Fatalf("load source-a ref: %v", err)
	}
	if err := conn.Model(&course.SourceRefEntity{}).
		Where("source = ? AND entity_type = ? AND external_id = ?", "source-b", course.EntityTypeCourse, "c1").
		First(&refB).Error; err != nil {
		t.Fatalf("load source-b ref: %v", err)
	}
	if refA.Source != "source-a" || refB.Source != "source-b" {
		t.Fatalf("source refs not isolated by source: A=%+v B=%+v", refA, refB)
	}
	if refA.LocalId == refB.LocalId {
		t.Fatalf("two sources must map to distinct courses, both local_id=%d", refA.LocalId)
	}
}
