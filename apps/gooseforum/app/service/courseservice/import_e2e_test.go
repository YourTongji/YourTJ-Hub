package courseservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// writeLegacyE2EPackage 构造与上游导出器 export_legacy_course_package.py 输出格式一致的
// 完整数据包：4 个 JSONL（courses/instructors/offerings/reviews）+ manifest.yaml
// （schema_version=1 + counts + rights_approval_ref + 各文件 sha256）。
// issue #183 Phase B 端到端验收的输入侧；source 固定 "test-e2e-legacy"。
func writeLegacyE2EPackage(t *testing.T, files map[string]string, counts map[string]int, rightsApprovalRef string) string {
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
	manifest := "schema_version: 1\nsource: test-e2e-legacy\nrights_approval_ref: " + rightsApprovalRef + "\ncounts:\n"
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

// migrateCourseImportTables 迁移并清空端到端链路所需的全部表（catalog 导入 + reviews 导入 + 统计 + 任务队列表）。
func migrateCourseImportTables(t *testing.T) {
	t.Helper()
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
		&course.ReviewEntity{},
		&course.HelpfulEntity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
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
}

// TestE2ELegacyImportFullChain issue #183 验收 1/2/4 端到端链路：
// 上游格式小包 dry-run（0 冲突、计数一致）→ catalog 正式导入（import_run completed）→
// reviews 正式导入（legacy 行）→ 统计重建 → 目录/详情抽查 → 重复导入幂等 → 匿名 DTO 零泄露。
func TestE2ELegacyImportFullChain(t *testing.T) {
	migrateCourseImportTables(t)

	// 与上游导出器行结构一致的小包：2 课程 + 1 教师 + 2 开课 + 2 评价（其一 rating=0 → NULL）。
	files := map[string]string{
		"courses.jsonl": `{"id":"c1","code":"100001","name":"高等数学(A)上","department":"数学科学学院","credit":5,"aliases":["高数"]}` + "\n" +
			`{"id":"c2","code":"100002","name":"线性代数","department":"数学科学学院","credit":3}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院","title":"教授"}` + "\n",
		"offerings.jsonl": `{"id":"o1","course_id":"c1","term":"2025-2026-1","campus":"四平路校区","faculty":"数学科学学院","instructor_ids":["i1"]}` + "\n" +
			`{"id":"o2","course_id":"c2","term":"2025-2026-1","campus":"嘉定校区","instructor_ids":["i1"]}` + "\n",
		"reviews.jsonl": `{"offering_external_id":"o1","rating":4,"content":"老师讲得很清楚","created_at":"2023-06-01T08:00:00+08:00","legacy_helpful_count":3}` + "\n" +
			`{"offering_external_id":"o2","rating":0,"content":"无评分历史评价","created_at":"2023-06-02T08:00:00+08:00","legacy_helpful_count":1}` + "\n",
	}
	manifestPath := writeLegacyE2EPackage(t, files, map[string]int{
		"courses.jsonl": 2, "instructors.jsonl": 1, "offerings.jsonl": 2, "reviews.jsonl": 2,
	}, "e2e-approval-001")

	// 1) catalog dry-run：0 冲突、计数与 manifest 一致。
	dryReport, err := ImportCatalog(context.Background(), manifestPath, true)
	if err != nil {
		t.Fatalf("catalog dry-run: %v", err)
	}
	if dryReport.TotalLines != 5 || dryReport.Quarantined != 0 || len(dryReport.Errors) != 0 {
		t.Fatalf("dry-run report unexpected: %+v", dryReport)
	}

	// 2) catalog 正式导入：全部插入，import_run completed。
	report, err := ImportCatalog(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("catalog import: %v", err)
	}
	if report.Inserted != 5 {
		t.Fatalf("expected 5 inserted rows, got inserted=%d updated=%d", report.Inserted, report.Updated)
	}
	conn := dbconnect.Connect()
	var run course.ImportRunEntity
	if err := conn.Order("id desc").First(&run).Error; err != nil {
		t.Fatalf("load import run: %v", err)
	}
	if run.Status != course.ImportStatusCompleted {
		t.Fatalf("import run status = %s, want completed", run.Status)
	}

	// 3) reviews dry-run：0 冲突。
	if _, err := ImportReviews(context.Background(), manifestPath, true); err != nil {
		t.Fatalf("reviews dry-run: %v", err)
	}

	// 4) reviews 正式导入：2 条 legacy 行（rating 0 → NULL）。
	reviewReport, err := ImportReviews(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("reviews import: %v", err)
	}
	if reviewReport.Inserted != 2 || reviewReport.Quarantined != 0 {
		t.Fatalf("reviews report unexpected: %+v", reviewReport)
	}

	// 5) 统计重建成功。
	if err := course.RebuildAllCourseStats(); err != nil {
		t.Fatalf("rebuild course stats: %v", err)
	}

	// 6) 目录抽查：2 门课可见，高数（rating 4 唯一有效评分）均分 4.0。
	page, err := ListCatalog(CatalogQuery{Page: 1, Size: 50})
	if err != nil {
		t.Fatalf("list catalog: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("catalog total = %d, want 2", page.Total)
	}
	var mathSummary *CourseSummary
	var linearSummary *CourseSummary
	for i := range page.List {
		switch page.List[i].PrimaryCode {
		case "100001":
			mathSummary = &page.List[i]
		case "100002":
			linearSummary = &page.List[i]
		}
	}
	if mathSummary == nil || linearSummary == nil {
		t.Fatalf("catalog missing courses: %+v", page.List)
	}
	if mathSummary.RatingAvg == nil || *mathSummary.RatingAvg != 4.0 {
		t.Fatalf("rating avg = %v, want 4.0", mathSummary.RatingAvg)
	}
	if mathSummary.ReviewCount != 1 {
		t.Fatalf("review count = %d, want 1", mathSummary.ReviewCount)
	}
	if linearSummary.RatingAvg != nil || linearSummary.ReviewCount != 1 {
		t.Fatalf("linear summary = avg %v count %d, want nil/1", linearSummary.RatingAvg, linearSummary.ReviewCount)
	}

	// 7) 详情抽查：offering 数量与课程一致。
	detail, err := GetCourseDetail(mathSummary.Id)
	if err != nil {
		t.Fatalf("get course detail: %v", err)
	}
	if len(detail.Offerings) != 1 || detail.Offerings[0].TermCode != "2025-2026-1" {
		t.Fatalf("course detail offerings unexpected: %+v", detail.Offerings)
	}
	if detail.RatingAvg == nil || *detail.RatingAvg != 4.0 {
		t.Fatalf("detail rating avg = %v, want 4.0", detail.RatingAvg)
	}

	// 8) 匿名 DTO 零泄露：公开评价查询返回 kind=legacy 且不含任何身份字段。
	// 每 offering 至多 1 条 legacy 评价（映射决策）：o1 有评分、o2 无评分（rating 0 → NULL）。
	o1Reviews, err := ListReviewsByOffering(detail.Offerings[0].Id, 0)
	if err != nil {
		t.Fatalf("list reviews o1: %v", err)
	}
	if len(o1Reviews) != 1 {
		t.Fatalf("o1 reviews count = %d, want 1", len(o1Reviews))
	}
	r := o1Reviews[0]
	if r.Author.Kind != "legacy" || r.Author.Label != "历史匿名评价" {
		t.Fatalf("review author not legacy-anonymized: %+v", r.Author)
	}
	if r.Viewer.CanEdit || r.Viewer.CanDelete {
		t.Fatalf("legacy review viewer must not be editable: %+v", r.Viewer)
	}
	if r.Rating == nil || *r.Rating != 4 {
		t.Fatalf("o1 rating = %v, want 4", r.Rating)
	}
	if r.HelpfulCount != 3 {
		t.Fatalf("o1 helpful count = %d, want 3 (legacy_helpful_count)", r.HelpfulCount)
	}
	// o2 无评分 legacy：rating NULL、不计均分但计入计数。
	linearDetail, err := GetCourseDetail(linearSummary.Id)
	if err != nil {
		t.Fatalf("get linear detail: %v", err)
	}
	if linearDetail.RatingAvg != nil || linearDetail.ReviewCount != 1 {
		t.Fatalf("linear stats = avg %v count %d, want nil/1", linearDetail.RatingAvg, linearDetail.ReviewCount)
	}

	// 9) 幂等：同一 manifest 重复导入 → manifest 级跳过，不重复写。
	second, err := ImportCatalog(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("second catalog import: %v", err)
	}
	if second.Skipped != 1 {
		t.Fatalf("expected manifest-level skip on re-import, got %+v", second)
	}
	var courseCount int64
	if err := conn.Model(&course.Entity{}).Count(&courseCount).Error; err != nil {
		t.Fatalf("count courses: %v", err)
	}
	if courseCount != 2 {
		t.Fatalf("course count after re-import = %d, want 2", courseCount)
	}
	// reviews 重复导入同样幂等（manifest 级跳过，不重复写）。
	secondReviews, err := ImportReviews(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("second reviews import: %v", err)
	}
	if secondReviews.Skipped != 1 || secondReviews.Inserted != 0 {
		t.Fatalf("reviews re-import not idempotent: %+v", secondReviews)
	}
}

// TestE2ELegacyImportQuarantineAmbiguity issue #183 验收 3：
// 重复课程代码歧义被隔离并报告（不静默丢弃、不中断整包）。
func TestE2ELegacyImportQuarantineAmbiguity(t *testing.T) {
	migrateCourseImportTables(t)

	// 同一课程代码出现两条不同课程行（上游数据脏），导入器必须隔离报告。
	files := map[string]string{
		"courses.jsonl": `{"id":"c1","code":"100001","name":"高等数学(A)上","department":"数学科学学院","credit":5}` + "\n" +
			`{"id":"c3","code":"100001","name":"高等数学(B)","department":"数学科学学院","credit":6}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院"}` + "\n",
		"offerings.jsonl":   `{"id":"o1","course_id":"c1","term":"2025-2026-1"}` + "\n",
	}
	manifestPath := writeLegacyE2EPackage(t, files, map[string]int{
		"courses.jsonl": 2, "instructors.jsonl": 1, "offerings.jsonl": 1,
	}, "e2e-approval-002")

	// dry-run 阶段即报告隔离，不静默丢弃。
	dryReport, err := ImportCatalog(context.Background(), manifestPath, true)
	if err != nil {
		t.Fatalf("catalog dry-run with ambiguity: %v", err)
	}
	if dryReport.Quarantined == 0 {
		t.Fatalf("expected quarantined duplicate-code row, got %+v", dryReport)
	}
	hasDuplicateReason := false
	for _, e := range dryReport.Errors {
		if e.Entity == "course" {
			hasDuplicateReason = true
		}
	}
	if !hasDuplicateReason {
		t.Fatalf("expected duplicate-code quarantine reason in report, got %+v", dryReport.Errors)
	}
}
