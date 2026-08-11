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
		report.Quarantined, report.Errors = validateReviewRows(rows, manifest.Source)
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

	if err := applyReviewRows(run.Id, manifest.Source, rows, report); err != nil {
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
// source 为 reviews manifest 来源标识，offering 解析限定在该来源下（与目录导入同源）。
// 除行级校验外，同一文件内重复的 offering_external_id 只有第一行可导入，
// 其余进入隔离区（与目录导入的重复 external id 语义一致），避免第二行
// 静默覆盖第一行内容、报告却记为 Updated。
func validateReviewRows(rows []importReviewRow, source string) (quarantined int, errs []ImportError) {
	db := dbconnect.Connect()
	dupKeys := duplicateReviewKeys(rows)
	for _, row := range rows {
		if reason := reviewRowQuarantined(db, source, row); reason != "" {
			quarantined++
			errs = append(errs, ImportError{Entity: "review", Reason: reason})
			continue
		}
		if dupKeys[strings.TrimSpace(row.OfferingExternalID)] {
			quarantined++
			errs = append(errs, ImportError{
				Entity: "review",
				Reason: fmt.Sprintf("duplicate offering_external_id %q in the same manifest", strings.TrimSpace(row.OfferingExternalID)),
			})
		}
	}
	return
}

// duplicateReviewKeys 返回同一文件内重复出现的 offering_external_id 集合
// （第一行保留，后续行全部进入隔离区）。
func duplicateReviewKeys(rows []importReviewRow) map[string]bool {
	seen := make(map[string]struct{}, len(rows))
	dup := make(map[string]bool)
	for _, row := range rows {
		key := strings.TrimSpace(row.OfferingExternalID)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			dup[key] = true
		}
		seen[key] = struct{}{}
	}
	return dup
}

// reviewRowQuarantined 返回该行隔离原因；空串表示可导入。
// 字段非法（rating 越界/helpful 为负/created_at 非 RFC3339）或
// offering_external_id 无法解析为已导入 offering 时隔离，不中断整个 run。
func reviewRowQuarantined(db *gorm.DB, source string, row importReviewRow) string {
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
	if _, err := sourceRefLocalID(db, source, row.OfferingExternalID, course.EntityTypeOffering); err != nil {
		return fmt.Sprintf("unmatched offering_external_id %q", row.OfferingExternalID)
	}
	return ""
}

// applyReviewRows 实际写入：先做行级校验（与 dry-run 一致），每批 500 行一个事务。
// 同一文件内重复的 offering_external_id 只有第一行可导入，其余隔离
// （validateReviewRows 已统计），避免第二行静默覆盖第一行内容。
func applyReviewRows(runID uint64, source string, rows []importReviewRow, report *ReviewsImportReport) error {
	const batchSize = 500
	db := dbconnect.Connect()
	dupKeys := duplicateReviewKeys(rows)
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
				if err := applyReviewRow(tx, runID, source, row, report, dupKeys); err != nil {
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
func applyReviewRow(tx *gorm.DB, runID uint64, source string, row importReviewRow, report *ReviewsImportReport, dupKeys map[string]bool) error {
	if reason := reviewRowQuarantined(tx, source, row); reason != "" {
		report.Quarantined++
		report.Errors = append(report.Errors, ImportError{Entity: "review", Reason: reason})
		return nil
	}
	if dupKeys[strings.TrimSpace(row.OfferingExternalID)] {
		report.Quarantined++
		report.Errors = append(report.Errors, ImportError{
			Entity: "review",
			Reason: fmt.Sprintf("duplicate offering_external_id %q in the same manifest", strings.TrimSpace(row.OfferingExternalID)),
		})
		return nil
	}
	offeringLocalID, err := sourceRefLocalID(tx, source, row.OfferingExternalID, course.EntityTypeOffering)
	if err != nil {
		return fmt.Errorf("lookup offering source ref %s: %w", row.OfferingExternalID, err)
	}
	rating, createdAt, helpful, content := parseReviewRow(row)
	checksum := rowChecksum(row)
	ref, refErr := sourceRefByExternal(tx, source, row.OfferingExternalID, course.EntityTypeReview)
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

	if err := touchSourceRef(tx, runID, source, row.OfferingExternalID, reviewID, course.EntityTypeReview, checksum); err != nil {
		return err
	}
	// legacy 评价导入不改变课程搜索文档内容（文档不含课评字段），不入队搜索任务。
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
