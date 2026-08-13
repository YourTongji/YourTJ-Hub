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
)

// writeReviewsManifestFixture 在临时目录写入 reviews.jsonl 并生成带 rights_approval_ref 的 manifest。
func writeReviewsManifestFixture(t *testing.T, content, rightsApprovalRef string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "reviews.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write reviews.jsonl: %v", err)
	}
	sum := sha256.Sum256([]byte(content))
	manifest := fmt.Sprintf("schema_version: 1\nsource: test-fixture\nrights_approval_ref: %s\nfiles:\n  reviews.jsonl: %s\n",
		rightsApprovalRef, hex.EncodeToString(sum[:]))
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return manifestPath
}

// setupReviewsImportTest 复用 review 域测试表，补充导入运行/来源映射表并建立 offering 映射。
func setupReviewsImportTest(t *testing.T) (courseId, offeringId uint64) {
	t.Helper()
	courseId, offeringId = setupReviewTest(t)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&course.ImportRunEntity{}, &course.SourceRefEntity{}); err != nil {
		t.Fatalf("migrate import tables: %v", err)
	}
	for _, model := range []any{&course.ImportRunEntity{}, &course.SourceRefEntity{}} {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean import table: %v", err)
		}
	}
	// source 必须与测试 manifest 的 source 一致（writeReviewsManifestFixture 用 test-fixture），
	// 来源映射以 (source, entity_type, external_id) 为键，跨来源隔离。
	ref := course.SourceRefEntity{
		Source:     "test-fixture",
		EntityType: course.EntityTypeOffering,
		ExternalId: "o1",
		LocalId:    offeringId,
	}
	if err := conn.Create(&ref).Error; err != nil {
		t.Fatalf("create offering source ref: %v", err)
	}
	return courseId, offeringId
}

// writeReviewsManifestFixtureWithCounts 同 writeReviewsManifestFixture，但附加
// manifest.counts（文件名 -> 期望行数），用于计数一致性校验测试。
func writeReviewsManifestFixtureWithCounts(t *testing.T, content, rightsApprovalRef string, counts map[string]int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "reviews.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write reviews.jsonl: %v", err)
	}
	sum := sha256.Sum256([]byte(content))
	manifest := fmt.Sprintf("schema_version: 1\nsource: test-fixture\nrights_approval_ref: %s\ncounts:\n", rightsApprovalRef)
	for name, count := range counts {
		manifest += fmt.Sprintf("  %s: %d\n", name, count)
	}
	manifest += fmt.Sprintf("files:\n  reviews.jsonl: %s\n", hex.EncodeToString(sum[:]))
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return manifestPath
}

// TestImportReviewsValidatesManifestCounts manifest.counts 与实际解析行数一致时
// dry-run 通过；不一致时拒绝（含正式导入），计数是 sha256 之外的第二道防线。
func TestImportReviewsValidatesManifestCounts(t *testing.T) {
	setupReviewsImportTest(t)
	content := `{"offering_external_id":"o1","rating":4,"content":"好","created_at":"2023-01-01T00:00:00Z"}` + "\n" +
		`{"offering_external_id":"o2","rating":3,"content":"中","created_at":"2023-01-02T00:00:00Z"}` + "\n"
	ok := writeReviewsManifestFixtureWithCounts(t, content, "approval-1", map[string]int{"reviews.jsonl": 2})
	report, err := ImportReviews(context.Background(), ok, true)
	if err != nil {
		t.Fatalf("dry-run with matching counts: %v", err)
	}
	if report.TotalLines != 2 {
		t.Fatalf("expected 2 total lines, got %d", report.TotalLines)
	}
	bad := writeReviewsManifestFixtureWithCounts(t, content, "approval-1", map[string]int{"reviews.jsonl": 1})
	if _, err := ImportReviews(context.Background(), bad, true); err == nil {
		t.Fatal("expected counts mismatch error on dry-run")
	} else if !strings.Contains(err.Error(), "counts mismatch") {
		t.Fatalf("expected counts mismatch error, got %v", err)
	}
	// 真实导入同样拒绝，确保截断的半包不会被静默导入。
	if _, err := ImportReviews(context.Background(), bad, false); err == nil {
		t.Fatal("expected counts mismatch error on real import")
	}
}

// TestImportReviewsRequiresRightsApproval manifest 缺少 rights_approval_ref 时拒绝（含 dry-run）。
func TestImportReviewsRequiresRightsApproval(t *testing.T) {
	setupReviewsImportTest(t)
	// 复用目录导入 fixture：manifest 不含 rights_approval_ref。
	manifestPath := writeManifestFixture(t, map[string]string{
		"reviews.jsonl": `{"offering_external_id":"o1","rating":4,"content":"好","created_at":"2023-01-01T00:00:00Z"}` + "\n",
	})
	if _, err := ImportReviews(context.Background(), manifestPath, false); !errors.Is(err, ErrReviewsRightsApprovalMissing) {
		t.Fatalf("expected ErrReviewsRightsApprovalMissing, got %v", err)
	}
	if _, err := ImportReviews(context.Background(), manifestPath, true); !errors.Is(err, ErrReviewsRightsApprovalMissing) {
		t.Fatalf("expected ErrReviewsRightsApprovalMissing on dry-run, got %v", err)
	}
}

// TestImportReviewsCreatesLegacyRow 合法行创建 legacy 评价：rating 0 → NULL、source=legacy-import、作者 0。
func TestImportReviewsCreatesLegacyRow(t *testing.T) {
	_, offeringId := setupReviewsImportTest(t)
	manifestPath := writeReviewsManifestFixture(t,
		`{"offering_external_id":"o1","rating":0,"content":" 老评价 ","created_at":"2023-06-01T08:00:00+08:00","legacy_helpful_count":7}`+"\n",
		"approval-1")
	report, err := ImportReviews(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("import reviews: %v", err)
	}
	if report.Inserted != 1 || report.Quarantined != 0 || report.TotalLines != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	conn := dbconnect.Connect()
	var review course.ReviewEntity
	if err := conn.Where("offering_id = ?", offeringId).First(&review).Error; err != nil {
		t.Fatalf("load review: %v", err)
	}
	if review.AuthorID() != 0 || !review.IsAnonymous || review.Source != course.ReviewSourceLegacyImport {
		t.Fatalf("legacy row flags wrong: %+v", review)
	}
	if review.Status != course.ReviewStatusVisible {
		t.Fatalf("expected visible status, got %d", review.Status)
	}
	if review.Rating != nil {
		t.Fatalf("rating 0 must store NULL, got %d", *review.Rating)
	}
	if review.LegacyHelpfulCount != 7 {
		t.Fatalf("expected legacy_helpful_count=7, got %d", review.LegacyHelpfulCount)
	}
	if review.Content != "老评价" {
		t.Fatalf("expected trimmed content, got %q", review.Content)
	}
	wantCreated := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	if !review.CreatedAt.Equal(wantCreated) {
		t.Fatalf("expected created_at %v, got %v", wantCreated, review.CreatedAt)
	}
	// 统计投影：rating NULL 不计入 rating 聚合，但计入 review_count。
	var stats course.OfferingStatsEntity
	if err := conn.Where("offering_id = ?", offeringId).First(&stats).Error; err != nil {
		t.Fatalf("load offering stats: %v", err)
	}
	if stats.ReviewCount != 1 || stats.RatingCount != 0 || stats.RatingSum != 0 {
		t.Fatalf("unexpected offering stats: %+v", stats)
	}
}

// TestImportReviewsUpsertSameOffering 重复导入同一 offering 的 legacy 行更新而非重复插入。
func TestImportReviewsUpsertSameOffering(t *testing.T) {
	_, offeringId := setupReviewsImportTest(t)
	manifestPath := writeReviewsManifestFixture(t,
		`{"offering_external_id":"o1","rating":4,"content":"第一版","created_at":"2023-01-01T00:00:00Z","legacy_helpful_count":2}`+"\n",
		"approval-1")
	first, err := ImportReviews(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if first.Inserted != 1 {
		t.Fatalf("expected 1 inserted, got %+v", first)
	}
	// 同一 manifest 再次导入：manifest 级幂等跳过。
	second, err := ImportReviews(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.Skipped != 1 {
		t.Fatalf("expected manifest-level skip, got %+v", second)
	}
	// 修改内容后重新导入：行级 checksum 变化 → 原地更新。
	manifestPath2 := writeReviewsManifestFixture(t,
		`{"offering_external_id":"o1","rating":5,"content":"第二版","created_at":"2023-02-02T00:00:00Z","legacy_helpful_count":3}`+"\n",
		"approval-1")
	third, err := ImportReviews(context.Background(), manifestPath2, false)
	if err != nil {
		t.Fatalf("third import: %v", err)
	}
	if third.Updated != 1 {
		t.Fatalf("expected 1 updated, got %+v", third)
	}
	conn := dbconnect.Connect()
	var cnt int64
	if err := conn.Model(&course.ReviewEntity{}).Where("offering_id = ?", offeringId).Count(&cnt).Error; err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected exactly 1 review after upsert, got %d", cnt)
	}
	var review course.ReviewEntity
	if err := conn.Where("offering_id = ?", offeringId).First(&review).Error; err != nil {
		t.Fatalf("load review: %v", err)
	}
	if review.Content != "第二版" || review.Rating == nil || *review.Rating != 5 || review.LegacyHelpfulCount != 3 {
		t.Fatalf("expected updated values, got %+v", review)
	}
}

// TestImportReviewsQuarantinesUnmatchedOffering 无法解析的 offering_external_id 计入隔离，不中断 run。
func TestImportReviewsQuarantinesUnmatchedOffering(t *testing.T) {
	setupReviewsImportTest(t)
	manifestPath := writeReviewsManifestFixture(t,
		`{"offering_external_id":"missing","rating":4,"content":"x","created_at":"2023-01-01T00:00:00Z"}`+"\n",
		"approval-1")
	report, err := ImportReviews(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("import reviews: %v", err)
	}
	if report.Quarantined != 1 || len(report.Errors) != 1 {
		t.Fatalf("expected 1 quarantined row, got %+v", report)
	}
	conn := dbconnect.Connect()
	var cnt int64
	if err := conn.Model(&course.ReviewEntity{}).Count(&cnt).Error; err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("quarantined row must not be written, found %d reviews", cnt)
	}
}

// TestImportReviewsDryRunDoesNotWrite dry-run 只校验并报告计数，不写库。
func TestImportReviewsDryRunDoesNotWrite(t *testing.T) {
	setupReviewsImportTest(t)
	manifestPath := writeReviewsManifestFixture(t,
		`{"offering_external_id":"o1","rating":4,"content":"干跑","created_at":"2023-01-01T00:00:00Z"}`+"\n"+
			`{"offering_external_id":"missing","rating":4,"content":"x","created_at":"2023-01-01T00:00:00Z"}`+"\n",
		"approval-1")
	report, err := ImportReviews(context.Background(), manifestPath, true)
	if err != nil {
		t.Fatalf("dry-run import: %v", err)
	}
	if !report.DryRun || report.TotalLines != 2 || report.Quarantined != 1 || report.Inserted != 0 {
		t.Fatalf("unexpected dry-run report: %+v", report)
	}
	conn := dbconnect.Connect()
	var cnt int64
	if err := conn.Model(&course.ReviewEntity{}).Count(&cnt).Error; err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("dry-run must not write, found %d reviews", cnt)
	}
}
