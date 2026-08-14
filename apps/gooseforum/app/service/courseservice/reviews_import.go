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

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
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
// ID 为可选的行级幂等键：带 id 的行以 author_user_id=NULL 落库（同 offering 可
// 多条 legacy 评价共存，唯一索引不冲突）；不带 id 的旧格式行维持"每 offering
// 至多一条"语义（author_user_id=0 占位，原地更新）。
type importReviewRow struct {
	ID                 string `json:"id,omitempty"`
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
// 带 id 的行以 author_user_id=NULL 落库（唯一索引不冲突，同 offering 多条共存）；
// 不带 id 的旧格式行每 offering 至多一条（author_user_id=0 占位），已存在时原地更新。
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

	rows, fileCounts, err := loadReviewFile(filepath.Dir(manifestPath), manifest)
	if err != nil {
		return nil, err
	}
	if err := validateManifestCounts(manifest, fileCounts); err != nil {
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
		Kind:         course.ImportKindReviews,
		Status:       course.ImportStatusRunning,
		StartedAt:    &now,
	}
	existing, err := course.GetImportRunByManifestHash(manifestHash, course.ImportKindReviews)
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
// 返回各文件的解析行数（文件名 -> 非空行数），供 manifest.counts 校验。
func loadReviewFile(manifestDir string, manifest ImportManifest) ([]importReviewRow, map[string]int, error) {
	var rows []importReviewRow
	fileCounts := make(map[string]int, len(manifest.Files))
	for name, wantSum := range manifest.Files {
		if filepath.IsAbs(name) || strings.Contains(name, "..") {
			return nil, nil, fmt.Errorf("manifest file %q: absolute and parent paths are not allowed", name)
		}
		if !strings.HasPrefix(name, "reviews") {
			// 同一 manifest 包可能同时携带 catalog 文件（courses/instructors/offerings，
			// 上游导出器输出单包 4 文件）。本命令只消费 reviews，其余跳过。
			continue
		}
		path := filepath.Join(manifestDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", path, err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != wantSum {
			return nil, nil, fmt.Errorf("checksum mismatch for %s: want %s got %s", name, wantSum, got)
		}
		n, err := parseJSONL(data, &rows)
		if err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", name, err)
		}
		fileCounts[name] = n
	}
	// 残缺包防护（与 catalog 侧对称）：manifest 声明了文件但没有任何 reviews 文件
	// （例如误把 catalog-only 包交给 course-import reviews），直接报错而不是静默空成功。
	if len(rows) == 0 && len(manifest.Files) > 0 {
		return nil, nil, fmt.Errorf("manifest contains no reviews file")
	}
	return rows, fileCounts, nil
}

func validateReviewRows(rows []importReviewRow, source string) (quarantined int, errs []ImportError) {
	db := dbconnect.Connect()
	dupKeys := duplicateReviewKeys(rows)
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if reason := reviewRowQuarantined(db, source, row); reason != "" {
			quarantined++
			errs = append(errs, ImportError{Entity: "review", Reason: reason})
			continue
		}
		if markReviewRowSeen(row, dupKeys, seen) {
			quarantined++
			errs = append(errs, ImportError{
				Entity: "review",
				Reason: fmt.Sprintf("duplicate review row %q in the same manifest", reviewRowKey(row)),
			})
		}
	}
	return
}

// reviewRowKey 行级去重/幂等键：带 id 的行按 id（同一外部评价只有一行），
// 不带 id 的旧格式行按 offering_external_id（维持每 offering 一条 legacy 语义）。
func reviewRowKey(row importReviewRow) string {
	if id := strings.TrimSpace(row.ID); id != "" {
		return "id|" + id
	}
	return "offering|" + strings.TrimSpace(row.OfferingExternalID)
}

func duplicateReviewKeys(rows []importReviewRow) map[string]bool {
	seen := make(map[string]struct{}, len(rows))
	dup := make(map[string]bool)
	for _, row := range rows {
		key := reviewRowKey(row)
		if key == "id|" || key == "offering|" {
			continue
		}
		if _, ok := seen[key]; ok {
			dup[key] = true
		}
		seen[key] = struct{}{}
	}
	return dup
}

// markReviewRowSeen 判定该行是否"重复键的后续行"：仅对 dup 集合中的键做
// 第二次出现检查，保证第一行保留、后续行隔离（与目录导入重复 external id
// 语义一致）。返回 true 表示应隔离本行。
func markReviewRowSeen(row importReviewRow, dupKeys map[string]bool, seen map[string]bool) bool {
	key := reviewRowKey(row)
	if !dupKeys[key] {
		return false
	}
	if seen[key] {
		return true
	}
	seen[key] = true
	return false
}

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

func applyReviewRows(runID uint64, source string, rows []importReviewRow, report *ReviewsImportReport) error {
	const batchSize = 500
	db := dbconnect.Connect()
	dupKeys := duplicateReviewKeys(rows)
	seen := make(map[string]bool, len(rows))
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
				if err := applyReviewRow(tx, runID, source, row, report, dupKeys, seen); err != nil {
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
// 带 id 的行按 id 幂等（author_user_id=NULL，同 offering 多条共存）；
// 不带 id 的旧格式行按 offering 幂等（author_user_id=0，每 offering 至多一条）。
func applyReviewRow(tx *gorm.DB, runID uint64, source string, row importReviewRow, report *ReviewsImportReport, dupKeys map[string]bool, seen map[string]bool) error {
	if reason := reviewRowQuarantined(tx, source, row); reason != "" {
		report.Quarantined++
		report.Errors = append(report.Errors, ImportError{Entity: "review", Reason: reason})
		return nil
	}
	if markReviewRowSeen(row, dupKeys, seen) {
		report.Quarantined++
		report.Errors = append(report.Errors, ImportError{
			Entity: "review",
			Reason: fmt.Sprintf("duplicate review row %q in the same manifest", reviewRowKey(row)),
		})
		return nil
	}
	offeringLocalID, err := sourceRefLocalID(tx, source, row.OfferingExternalID, course.EntityTypeOffering)
	if err != nil {
		return fmt.Errorf("lookup offering source ref %s: %w", row.OfferingExternalID, err)
	}
	rating, createdAt, helpful, content := parseReviewRow(row)
	checksum := rowChecksum(row)
	// review source_ref 的 external_id：带 id 行用行 id，旧格式行用 offering 外部 id
	// （向后兼容，两种模式互不覆盖）。
	refKey := strings.TrimSpace(row.OfferingExternalID)
	if id := strings.TrimSpace(row.ID); id != "" {
		refKey = id
	}
	ref, refErr := sourceRefByExternal(tx, source, refKey, course.EntityTypeReview)
	if refErr == nil && ref.Checksum == checksum {
		report.Skipped++ // 行内容未变化
		return nil
	}

	var reviewID uint64
	if strings.TrimSpace(row.ID) == "" {
		reviewID, err = upsertLegacyReviewByOfferingTx(tx, offeringLocalID, rating, content, helpful, createdAt, report)
	} else {
		reviewID, err = upsertLegacyReviewByRefTx(tx, ref, refErr, offeringLocalID, rating, content, helpful, createdAt, report)
	}
	if err != nil {
		return err
	}
	return touchSourceRef(tx, runID, source, refKey, reviewID, course.EntityTypeReview, checksum)
}

// upsertLegacyReviewByOfferingTx 旧格式（行无 id）：按 offering 查 legacy 行
// （author_user_id=0），有则原地更新、无则创建（每 offering 至多一条）。
func upsertLegacyReviewByOfferingTx(tx *gorm.DB, offeringLocalID uint64, rating *int, content string, helpful int, createdAt time.Time, report *ReviewsImportReport) (uint64, error) {
	existing, findErr := course.FindLegacyReviewByOfferingTx(tx, offeringLocalID)
	switch {
	case findErr == nil:
		updates := map[string]any{
			"rating":               rating,
			"content":              content,
			"legacy_helpful_count": helpful,
			"created_at":           createdAt,
		}
		if err := tx.Model(&course.ReviewEntity{}).Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
			return 0, fmt.Errorf("update legacy review for offering %d: %w", offeringLocalID, err)
		}
		report.Updated++
		return existing.Id, nil
	case errors.Is(findErr, gorm.ErrRecordNotFound):
		entity := course.ReviewEntity{
			OfferingId:         offeringLocalID,
			AuthorUserId:       uint64Ptr(0), // 旧格式以 0 占位（清理置 NULL 后同 offering 可共存）
			Rating:             rating,
			Content:            content,
			IsAnonymous:        true,
			Status:             course.ReviewStatusVisible,
			LegacyHelpfulCount: helpful,
			Source:             course.ReviewSourceLegacyImport,
			CreatedAt:          createdAt,
		}
		if err := tx.Model(&course.ReviewEntity{}).Create(&entity).Error; err != nil {
			return 0, fmt.Errorf("create legacy review for offering %d: %w", offeringLocalID, err)
		}
		report.Inserted++
		return entity.Id, nil
	default:
		return 0, fmt.Errorf("lookup legacy review for offering %d: %w", offeringLocalID, findErr)
	}
}

// upsertLegacyReviewByRefTx 新格式（行带 id）：按 review source_ref 定位本地行，
// 有则更新、无则创建；author_user_id 置 NULL——NULL 在唯一索引
// (offering_id, author_user_id) 中彼此不冲突（SQLite/PostgreSQL 一致），
// 同 offering 多条 legacy 评价可共存。
func upsertLegacyReviewByRefTx(tx *gorm.DB, ref course.SourceRefEntity, refErr error, offeringLocalID uint64, rating *int, content string, helpful int, createdAt time.Time, report *ReviewsImportReport) (uint64, error) {
	if refErr == nil {
		var existing course.ReviewEntity
		if err := tx.Model(&course.ReviewEntity{}).Where("id = ?", ref.LocalId).First(&existing).Error; err != nil {
			return 0, fmt.Errorf("load legacy review %d: %w", ref.LocalId, err)
		}
		updates := map[string]any{
			"offering_id":          offeringLocalID,
			"rating":               rating,
			"content":              content,
			"legacy_helpful_count": helpful,
			"created_at":           createdAt,
		}
		if err := tx.Model(&course.ReviewEntity{}).Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
			return 0, fmt.Errorf("update legacy review %d: %w", existing.Id, err)
		}
		report.Updated++
		return existing.Id, nil
	}
	if !errors.Is(refErr, gorm.ErrRecordNotFound) {
		return 0, fmt.Errorf("lookup legacy review source ref: %w", refErr)
	}
	entity := course.ReviewEntity{
		OfferingId:         offeringLocalID,
		AuthorUserId:       nil, // NULL：同 offering 多条 legacy 共存的关键
		Rating:             rating,
		Content:            content,
		IsAnonymous:        true,
		Status:             course.ReviewStatusVisible,
		LegacyHelpfulCount: helpful,
		Source:             course.ReviewSourceLegacyImport,
		CreatedAt:          createdAt,
	}
	if err := tx.Model(&course.ReviewEntity{}).Create(&entity).Error; err != nil {
		return 0, fmt.Errorf("create legacy review: %w", err)
	}
	report.Inserted++
	return entity.Id, nil
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
