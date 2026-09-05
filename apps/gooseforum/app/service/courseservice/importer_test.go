package courseservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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
	// counts 引用 manifest 中属于其他子命令的文件（单包多命令：上游导出器
	// 输出 catalog + reviews 共用一份 manifest、files 含 reviews.jsonl）：本命令
	// 跳过不校验，交由 course-import reviews 校验——dry-run 必须通过。
	multiCmd := writeManifestFixtureWithCounts(t, map[string]string{
		"courses.jsonl":     `{"id":"c1","code":"100001","name":"高等数学(A)上"}` + "\n" + `{"id":"c2","code":"100002","name":"线性代数"}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院"}` + "\n",
		"offerings.jsonl":   `{"id":"o1","course_id":"c1","term":"2025-2026-1"}` + "\n",
		"reviews.jsonl":     `{"offering_external_id":"o1","rating":4,"content":"好课"}` + "\n",
	}, map[string]int{
		"courses.jsonl": 2, "instructors.jsonl": 1, "offerings.jsonl": 1, "reviews.jsonl": 1,
	})
	if _, err := ImportCatalog(context.Background(), multiCmd, true); err != nil {
		t.Fatalf("dry-run with cross-command counts: %v", err)
	}
	// counts 引用了 manifest.files 中根本不存在的文件（typo/内部不一致）：仍拒绝。
	typo := writeManifestFixtureWithCounts(t, files, map[string]int{
		"courses.jsonl": 2, "instructors.jsonl": 1, "offerings.jsonl": 1, "courses.josnl": 1,
	})
	if _, err := ImportCatalog(context.Background(), typo, true); err == nil {
		t.Fatal("expected error for counts referencing unknown file")
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

// TestValidateRowsQuarantinesUnresolvableTeacherCode 回归 review：course.teacher_code
// 在 instructors.jsonl 中不可解析（缺失或已被自然键冲突隔离）时，课程行必须在校验期
// 隔离（dry-run 与真实导入一致），引用它的 offering 也一并隔离。
func TestValidateRowsQuarantinesUnresolvableTeacherCode(t *testing.T) {
	rows := importRows{
		courses: []importCourseRow{
			{ID: "c1", Code: "100001", Name: "高等数学", TeacherCode: "T001"},
		},
		instructors: []importInstructorRow{
			{ID: "i1", Name: "张三", Department: "数学科学学院"},
		},
		offerings: []importOfferingRow{
			{ID: "o1", CourseID: "c1", Term: "2025-2026-1"},
		},
	}
	report := &CatalogImportReport{Errors: []ImportError{}}
	quarantined := validateRows(rows, report)
	if report.Quarantined != 2 {
		t.Fatalf("expected 2 quarantined (course + offering), got %d: %v", report.Quarantined, report.Errors)
	}
	if !quarantined[course.EntityTypeCourse+"|c1"] {
		t.Fatal("expected course row to be quarantined for unresolvable teacher_code")
	}
	if !quarantined[course.EntityTypeOffering+"|o1"] {
		t.Fatal("expected offering row to be quarantined with its course")
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

// TestImportCatalogOfferingReparentInvalidatesAiSummary offering 改派到新课程时，
// 新旧课程的 AI 总结缓存（含 insufficient 标记）都必须失效：改派前判"评价不足"
// 的课程在改派后必须可重新评估，否则永久返回 insufficient_data（review 补漏）。
func TestImportCatalogOfferingReparentInvalidatesAiSummary(t *testing.T) {
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
		&course.CourseAiSummaryEntity{},
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

	// 第一轮导入：课程 c1 + 教师 i1 + offering o1（挂 c1）。
	manifestV1 := writeManifestFixture(t, map[string]string{
		"courses.jsonl":     `{"id":"c1","code":"100001","name":"高等数学(A)上","department":"数学科学学院","credit":5}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院"}` + "\n",
		"offerings.jsonl":   `{"id":"o1","course_id":"c1","term":"2025-2026-1","campus":"四平路校区","instructor_ids":["i1"]}` + "\n",
	})
	if _, err := ImportCatalog(context.Background(), manifestV1, false); err != nil {
		t.Fatalf("first import: %v", err)
	}
	// 课程 c1 落 insufficient 标记（模拟改派前已评估且评价不足）。
	var course1 course.Entity
	if err := conn.Where("primary_code = ?", "100001").First(&course1).Error; err != nil {
		t.Fatalf("load course c1: %v", err)
	}
	if err := conn.Create(&course.CourseAiSummaryEntity{
		CourseId:      course1.Id,
		PromptVersion: "v1",
		GeneratedAt:   time.Now(),
		Status:        course.AiSummaryRowStatusInsufficient,
	}).Error; err != nil {
		t.Fatalf("seed insufficient marker: %v", err)
	}

	// 第二轮导入：新增课程 c2，offering o1 改派到 c2（course_id 变化）。
	manifestV2 := writeManifestFixture(t, map[string]string{
		"courses.jsonl": `{"id":"c1","code":"100001","name":"高等数学(A)上","department":"数学科学学院","credit":5}` + "\n" +
			`{"id":"c2","code":"100002","name":"线性代数","department":"数学科学学院","credit":4}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院"}` + "\n",
		"offerings.jsonl":   `{"id":"o1","course_id":"c2","term":"2025-2026-1","campus":"四平路校区","instructor_ids":["i1"]}` + "\n",
	})
	if _, err := ImportCatalog(context.Background(), manifestV2, false); err != nil {
		t.Fatalf("second import: %v", err)
	}

	// 新课程 c2 先定位（改派后 offering 挂 c2）。
	var course2 course.Entity
	if err := conn.Where("primary_code = ?", "100002").First(&course2).Error; err != nil {
		t.Fatalf("load course c2: %v", err)
	}
	// offering 已改派到 c2。
	var offering course.OfferingEntity
	if err := conn.Where("course_id = ?", course2.Id).First(&offering).Error; err != nil {
		t.Fatalf("load offering: %v", err)
	}
	if offering.CourseId != course2.Id {
		t.Fatalf("offering course_id = %d, want %d (reparented to c2)", offering.CourseId, course2.Id)
	}
	// 旧课程 c1 的 insufficient 标记必须已失效（改派改变可见评价集合）。
	if cached := course.GetCourseAiSummary(course1.Id); cached.CourseId != 0 {
		t.Fatalf("old course ai summary must be invalidated after reparent, got status=%q", cached.Status)
	}
	// 新课程 c2 也无 summary 残留。
	if cached := course.GetCourseAiSummary(course2.Id); cached.CourseId != 0 {
		t.Fatalf("new course ai summary must be clean, got status=%q", cached.Status)
	}
}

func TestValidateRowsOfferingQuarantineIncludesExternalID(t *testing.T) {
	rows := importRows{
		courses: []importCourseRow{{ID: "c1", Code: "100001", Name: "高等数学"}},
		offerings: []importOfferingRow{
			{ID: "o-missing-course", CourseID: "missing-course", Term: "2025-2026-1"},
			{ID: "o-missing-instructor", CourseID: "c1", Term: "2025-2026-1", InstructorIDs: []string{"missing-instructor"}},
		},
	}
	report := &CatalogImportReport{Errors: []ImportError{}}

	quarantined := validateRows(rows, report)

	if !quarantined[course.EntityTypeOffering+"|o-missing-course"] || !quarantined[course.EntityTypeOffering+"|o-missing-instructor"] {
		t.Fatalf("expected both offering rows to be quarantined, got %v", quarantined)
	}
	if report.Quarantined != 2 || len(report.Errors) != 2 {
		t.Fatalf("expected two offering quarantine errors, got quarantined=%d errors=%v", report.Quarantined, report.Errors)
	}
	for _, err := range report.Errors {
		if err.Entity != "offering" || err.ExternalID == "" || err.Reason == "" {
			t.Fatalf("offering quarantine error must include entity, external ID, and reason: %+v", err)
		}
	}
}

func TestImportCatalogOfferingQuarantineDryRunParity(t *testing.T) {
	migrateCourseImportTables(t)
	manifestPath := writeManifestFixture(t, map[string]string{
		"courses.jsonl":     `{"id":"c1","code":"100001","name":"高等数学"}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院"}` + "\n",
		"offerings.jsonl":   `{"id":"o1","course_id":"c1","term":"2025-2026-1","instructor_ids":["missing-instructor"]}` + "\n",
	})

	dryReport, err := ImportCatalog(context.Background(), manifestPath, true)
	if err != nil {
		t.Fatalf("dry-run import: %v", err)
	}
	realReport, err := ImportCatalog(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("real import: %v", err)
	}

	if dryReport.Quarantined != 1 || realReport.Quarantined != 1 {
		t.Fatalf("dry-run and real import must quarantine one offering: dry=%+v real=%+v", dryReport, realReport)
	}
	if len(dryReport.Errors) != 1 || len(realReport.Errors) != 1 {
		t.Fatalf("dry-run and real import must report one offering error: dry=%+v real=%+v", dryReport, realReport)
	}
	if dryReport.Errors[0] != realReport.Errors[0] {
		t.Fatalf("dry-run and real import quarantine errors differ: dry=%+v real=%+v", dryReport.Errors[0], realReport.Errors[0])
	}
	if got := realReport.Errors[0]; got.Entity != "offering" || got.ExternalID != "o1" || got.Reason != "unresolvable instructor_id missing-instructor" {
		t.Fatalf("unexpected offering quarantine error: %+v", got)
	}
}

func TestApplyOfferingRowQuarantinesMissingSourceRefWithReport(t *testing.T) {
	db := newOfferingImportTestDB(t)
	report := &CatalogImportReport{Errors: []ImportError{}}

	err := applyOfferingRow(db, 1, "test-source", importOfferingRow{
		ID:       "o-missing-course",
		CourseID: "missing-course",
		Term:     "2025-2026-1",
	}, report)

	if err != nil {
		t.Fatalf("missing course source ref should quarantine, got error: %v", err)
	}
	if report.Quarantined != 1 || len(report.Errors) != 1 {
		t.Fatalf("expected one quarantine error, got quarantined=%d errors=%v", report.Quarantined, report.Errors)
	}
	got := report.Errors[0]
	if got.Entity != "offering" || got.ExternalID != "o-missing-course" || got.Reason != "unresolvable course_id missing-course" {
		t.Fatalf("unexpected quarantine error: %+v", got)
	}
}

func TestParseOfferingExternalID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want uint64
	}{
		{name: "teaching class external id", id: "1111111124951406-1294", want: 1111111124951406},
		{name: "virtual offering", id: "other-1294", want: 0},
		{name: "invalid", id: "not-an-offering", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseOfferingExternalID(tt.id); got != tt.want {
				t.Fatalf("parseOfferingExternalID(%q) = %d, want %d", tt.id, got, tt.want)
			}
		})
	}
}

func TestApplyOfferingRowQuarantinesMissingInstructorSourceRefWithReport(t *testing.T) {
	db := newOfferingImportTestDB(t)
	if err := db.Create(&course.SourceRefEntity{
		ImportRunId: 1,
		Source:      "test-source",
		EntityType:  course.EntityTypeCourse,
		ExternalId:  "course-1",
		LocalId:     1,
		Checksum:    "course-checksum",
	}).Error; err != nil {
		t.Fatalf("create course source ref: %v", err)
	}
	report := &CatalogImportReport{Errors: []ImportError{}}

	err := applyOfferingRow(db, 1, "test-source", importOfferingRow{
		ID:            "o-missing-instructor",
		CourseID:      "course-1",
		Term:          "2025-2026-1",
		InstructorIDs: []string{"missing-instructor"},
	}, report)

	if err != nil {
		t.Fatalf("missing instructor source ref should quarantine, got error: %v", err)
	}
	if report.Quarantined != 1 || len(report.Errors) != 1 {
		t.Fatalf("expected one quarantine error, got quarantined=%d errors=%v", report.Quarantined, report.Errors)
	}
	got := report.Errors[0]
	if got.Entity != "offering" || got.ExternalID != "o-missing-instructor" || got.Reason != "unresolvable instructor_id missing-instructor" {
		t.Fatalf("unexpected quarantine error: %+v", got)
	}
}

func TestApplyOfferingRowAbortsOnSourceRefDatabaseError(t *testing.T) {
	db := newOfferingImportTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}
	report := &CatalogImportReport{Errors: []ImportError{}}

	err = applyOfferingRow(db, 1, "test-source", importOfferingRow{
		ID:       "o-database-error",
		CourseID: "course-1",
		Term:     "2025-2026-1",
	}, report)

	if err == nil {
		t.Fatal("expected database error to abort offering batch")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("database error must not be classified as business quarantine: %v", err)
	}
	if report.Quarantined != 0 || len(report.Errors) != 0 {
		t.Fatalf("database error must not increment quarantine report: %+v", report)
	}
}

func newOfferingImportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "course-import.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	// Windows 下不关闭连接会锁住文件，导致 t.TempDir() 清理失败。
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test db: %v", err)
		}
	})
	if err := db.AutoMigrate(&course.TermEntity{}, &course.SourceRefEntity{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}
