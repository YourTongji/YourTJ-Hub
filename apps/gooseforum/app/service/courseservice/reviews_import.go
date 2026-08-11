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
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/leancodebox/GooseForum/app/service/searchservice"
	"go.yaml.in/yaml/v3"
	"gorm.io/gorm"
)

// ErrReviewsRightsApprovalMissing manifest 缺少 rights_approval_ref 时拒绝导入。
var ErrReviewsRightsApprovalMissing = errors.New("reviews import requires manifest rights_approval_ref (non-empty)")

// ReviewsImportReport 评价导入结果报告。
type ReviewsImportReport struct {
	Source       string        `json:"source"`
	ManifestHash string        `json:"manifestHash"`
	DryRun       bool          `json:"dryRun"`
	TotalLines   int           `json:"totalLines"`
	Inserted     int           `json:"inserted"`
	Updated      int           `json:"updated"`
	Quarantined  int           `json:"quarantined"`
	Skipped      int           `json:"skipped"`
	Errors       []ImportError `json:"errors,omitempty"`
}

// importReviewRow reviews.jsonl 单行（JSONL 一行一个对象）。
// 仅消费以下字段；行内其它字段（reviewer 名称/头像/钱包/IP/编辑令牌等）一律忽略。
type importReviewRow struct {
	OfferingExternalID string `json:"offering_external_id"`
	Rating             int    `json:"rating"`
	Content            string `json:"content"`
	CreatedAt          string `json:"created_at"`
	LegacyHelpfulCount int    `json:"legacy_helpful_count"`
}

// ImportReviews 导入历史评价（reviews.jsonl）。
// 强制要求 manifest.rights_approval_ref 非空，否则拒绝运行；
// dryRun=true 时只校验 manifest、解析全部行并报告计数，不写库。
// 幂等：manifest 内容哈希已存在且状态为 completed 时直接返回已存在报告；
// 行级幂等：course_source_ref（entity_type=review）checksum 不变的行跳过；
// 每个 offering 至多一条 legacy 评价（author_user_id=0 且 source=legacy-import），
// 已存在时原地更新 content/rating/legacy_helpful_count/created_at。
func ImportReviews(ctx context.Context, manifestPath string, dryRun bool) (*ReviewsImportReport, error) {
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var manifest ImportManifest
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported manifest schema_version %d", manifest.SchemaVersion)
	}
	if manifest.Source == "" {
		return nil, errors.New("manifest source is required")
	}
	if strings.TrimSpace(manifest.RightsApprovalRef) == "" {
		return nil, ErrReviewsRightsApprovalMissing
	}
	hash := sha256.Sum256(manifestBytes)
	manifestHash := hex.EncodeToString(hash[:])
	report := &ReviewsImportReport{
		Source:       manifest.Source,
		ManifestHash: manifestHash,
		DryRun:       dryRun,
		Errors:       []ImportError{},
	}

	rows, err := loadReviewFile(filepath.Dir(manifestPath), manifest)
	if err != nil {
		return nil, err
	}
	report.TotalLines = len(rows)

	if dryRun {
		report.Quarantined, report.Errors = validateReviewRows(rows)
		return report, nil
	}

	now := time.Now()
	run := course.ImportRunEntity{
		Source:       manifest.Source,
		ManifestHash: manifestHash,
		Status:       course.ImportStatusRunning,
		StartedAt:    &now,
	}
	existing, err := course.GetImportRunByManifestHash(manifestHash)
	switch {
	case err == nil && existing.Status == course.ImportStatusCompleted:
		report.Skipped = 1 // 幂等：该 manifest 已导入完成
		return report, nil
	case err == nil:
		// 复用失败的 run：重置为 running，行级 checksum 幂等负责断点续跑。
		run = existing
		run.Status = course.ImportStatusRunning
		run.StartedAt = &now
		run.FinishedAt = nil
		run.InsertedCount = 0
		run.UpdatedCount = 0
		run.QuarantinedCount = 0
		run.ErrorCount = 0
		if err := course.SaveImportRun(&run); err != nil {
			return nil, fmt.Errorf("reset import run: %w", err)
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := course.CreateImportRun(&run); err != nil {
			return nil, fmt.Errorf("create import run: %w", err)
		}
	default:
		return nil, fmt.Errorf("get import run: %w", err)
	}

	if err := applyReviewRows(run.Id, rows, report); err != nil {
		run.Status = course.ImportStatusFailed
		run.ErrorCount = len(report.Errors)
		finished := time.Now()
		run.FinishedAt = &finished
		_ = course.SaveImportRun(&run)
		return report, err
	}
	// 统计投影兜底：以 review 事实表为准全量重建，保证与导入一致。
	// 重建成功后才把 run 置为 completed；失败置 failed 允许重试，
	// 避免"completed 但投影陈旧且幂等跳过"的不可恢复状态。
	if err := course.RebuildAllCourseStats(); err != nil {
		run.Status = course.ImportStatusFailed
		run.ErrorCount = len(report.Errors)
		finished := time.Now()
		run.FinishedAt = &finished
		_ = course.SaveImportRun(&run)
		return nil, fmt.Errorf("rebuild course stats: %w", err)
	}
	run.Status = course.ImportStatusCompleted
	run.InsertedCount = report.Inserted
	run.UpdatedCount = report.Updated
	run.QuarantinedCount = report.Quarantined
	run.ErrorCount = len(report.Errors)
	finished := time.Now()
	run.FinishedAt = &finished
	if err := course.SaveImportRun(&run); err != nil {
		return nil, fmt.Errorf("save import run: %w", err)
	}
	return report, nil
}

// loadReviewFile 读取并校验 manifest 中的 reviews JSONL 文件。
// 文件路径相对 manifest 所在目录解析，拒绝绝对路径与父目录引用。
func loadReviewFile(manifestDir string, manifest ImportManifest) ([]importReviewRow, error) {
	var rows []importReviewRow
	for name, wantSum := range manifest.Files {
		if filepath.IsAbs(name) || strings.Contains(name, "..") {
			return nil, fmt.Errorf("manifest file %q: absolute and parent paths are not allowed", name)
		}
		if !strings.HasPrefix(name, "reviews") {
			return nil, fmt.Errorf("unexpected manifest file %s", name)
		}
		path := filepath.Join(manifestDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != wantSum {
			return nil, fmt.Errorf("checksum mismatch for %s: want %s got %s", name, wantSum, got)
		}
		if err := parseJSONL(data, &rows); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
	}
	return rows, nil
}

// validateReviewRows 统计隔离行（dry-run 与真实导入共用同一判定，保证报告一致）。
// 只读访问数据库（offering 解析），不写库。
func validateReviewRows(rows []importReviewRow) (quarantined int, errs []ImportError) {
	db := dbconnect.Connect()
	for _, row := range rows {
		if reason := reviewRowQuarantined(db, row); reason != "" {
			quarantined++
			errs = append(errs, ImportError{Entity: "review", Reason: reason})
		}
	}
	return
}

// reviewRowQuarantined 返回该行隔离原因；空串表示可导入。
// 字段非法（rating 越界/helpful 为负/created_at 非 RFC3339）或
// offering_external_id 无法解析为已导入 offering 时隔离，不中断整个 run。
func reviewRowQuarantined(db *gorm.DB, row importReviewRow) string {
	if strings.TrimSpace(row.OfferingExternalID) == "" {
		return "missing offering_external_id"
	}
	if row.Rating < 0 || row.Rating > 5 {
		return fmt.Sprintf("invalid rating %d (want 0..5)", row.Rating)
	}
	if row.LegacyHelpfulCount < 0 {
		return fmt.Sprintf("invalid legacy_helpful_count %d (want >= 0)", row.LegacyHelpfulCount)
	}
	if _, err := time.Parse(time.RFC3339, row.CreatedAt); err != nil {
		return fmt.Sprintf("invalid created_at %q (want RFC3339)", row.CreatedAt)
	}
	if _, err := sourceRefLocalID(db, row.OfferingExternalID, course.EntityTypeOffering); err != nil {
		return fmt.Sprintf("unmatched offering_external_id %q", row.OfferingExternalID)
	}
	return ""
}

// applyReviewRows 实际写入：先做行级校验（与 dry-run 一致），每批 500 行一个事务。
func applyReviewRows(runID uint64, rows []importReviewRow, report *ReviewsImportReport) error {
	const batchSize = 500
	db := dbconnect.Connect()
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[i:end]
		// 批次失败回滚时同步回滚报告计数，避免失败报告多算。
		snapshot := *report
		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, row := range batch {
				if err := applyReviewRow(tx, runID, row, report); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			*report = snapshot
			return err
		}
	}
	return nil
}

// applyReviewRow 事务内写入/更新一条 legacy 评价，并 touch 行级 source_ref checksum。
// 内容未变化（checksum 相同）时跳过，不入队搜索任务。
func applyReviewRow(tx *gorm.DB, runID uint64, row importReviewRow, report *ReviewsImportReport) error {
	if reason := reviewRowQuarantined(tx, row); reason != "" {
		report.Quarantined++
		report.Errors = append(report.Errors, ImportError{Entity: "review", Reason: reason})
		return nil
	}
	offeringLocalID, err := sourceRefLocalID(tx, row.OfferingExternalID, course.EntityTypeOffering)
	if err != nil {
		return fmt.Errorf("lookup offering source ref %s: %w", row.OfferingExternalID, err)
	}
	offering, err := course.GetOfferingTx(tx, offeringLocalID)
	if err != nil {
		return fmt.Errorf("load offering %d: %w", offeringLocalID, err)
	}
	rating, createdAt, helpful, content := parseReviewRow(row)
	checksum := rowChecksum(row)
	ref, refErr := sourceRefByExternal(tx, row.OfferingExternalID, course.EntityTypeReview)
	if refErr == nil && ref.Checksum == checksum {
		report.Skipped++ // 行内容未变化
		return nil
	}

	var reviewID uint64
	existing, findErr := course.FindLegacyReviewByOfferingTx(tx, offeringLocalID)
	switch {
	case findErr == nil:
		// 已有 legacy 评价：原地更新，不重复插入。
		updates := map[string]any{
			"rating":               rating,
			"content":              content,
			"legacy_helpful_count": helpful,
			"created_at":           createdAt,
		}
		if err := tx.Model(&course.ReviewEntity{}).Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
			return fmt.Errorf("update legacy review for offering %s: %w", row.OfferingExternalID, err)
		}
		reviewID = existing.Id
		report.Updated++
	case errors.Is(findErr, gorm.ErrRecordNotFound):
		entity := course.ReviewEntity{
			OfferingId:         offeringLocalID,
			AuthorUserId:       0,
			Rating:             rating,
			Content:            content,
			IsAnonymous:        true,
			Status:             course.ReviewStatusVisible,
			LegacyHelpfulCount: helpful,
			Source:             course.ReviewSourceLegacyImport,
			CreatedAt:          createdAt,
		}
		if err := tx.Model(&course.ReviewEntity{}).Create(&entity).Error; err != nil {
			return fmt.Errorf("create legacy review for offering %s: %w", row.OfferingExternalID, err)
		}
		reviewID = entity.Id
		report.Inserted++
	default:
		return fmt.Errorf("lookup legacy review for offering %s: %w", row.OfferingExternalID, findErr)
	}

	if err := touchSourceRef(tx, runID, row.OfferingExternalID, reviewID, course.EntityTypeReview, checksum); err != nil {
		return err
	}
	// 该 offering 所属课程需要重建搜索文档（transaction-bound outbox）。
	if err := searchservice.EnqueueCourseSearchTask(tx, offering.CourseId); err != nil {
		return fmt.Errorf("enqueue course search task %d: %w", offering.CourseId, err)
	}
	return nil
}

// parseReviewRow 将行字段转为落库值：rating 0 → NULL（不计平均），content 去首尾空白。
// 字段合法性已由 reviewRowQuarantined 保证。
func parseReviewRow(row importReviewRow) (rating *int, createdAt time.Time, helpful int, content string) {
	created, _ := time.Parse(time.RFC3339, row.CreatedAt)
	var r *int
	if row.Rating != 0 {
		v := row.Rating
		r = &v
	}
	return r, created, row.LegacyHelpfulCount, strings.TrimSpace(row.Content)
}
