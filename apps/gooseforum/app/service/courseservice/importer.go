package courseservice

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"go.yaml.in/yaml/v3"
	"gorm.io/gorm"
)

// ImportManifest 目录导入包的 manifest（YAML）。
// rights_approval_ref 在 reviews 导入时强制要求；目录导入仅记录，不强制。
type ImportManifest struct {
	SchemaVersion     int               `yaml:"schema_version" json:"schemaVersion"`
	Source            string            `yaml:"source" json:"source"`
	SourceCommit      string            `yaml:"source_commit" json:"sourceCommit"`
	ExportedAt        string            `yaml:"exported_at" json:"exportedAt"`
	RightsApprovalRef string            `yaml:"rights_approval_ref" json:"rightsApprovalRef"`
	Files             map[string]string `yaml:"files" json:"files"` // 文件名 -> sha256
	Counts            map[string]int    `yaml:"counts" json:"counts"`
}

// ImportError 单行导入错误。
type ImportError struct {
	Line   int    `json:"line"`
	Entity string `json:"entity"`
	Reason string `json:"reason"`
}

// CatalogImportReport 目录导入结果报告。
type CatalogImportReport struct {
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

// --- 导入行结构（与 manifest files 中的 JSONL 对应） ---

type importCourseRow struct {
	ID         string   `json:"id"`
	Code       string   `json:"code"`
	Name       string   `json:"name"`
	Department string   `json:"department"`
	Credit     float64  `json:"credit"`
	Aliases    []string `json:"aliases,omitempty"`
}

type importInstructorRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Department string `json:"department"`
	Title      string `json:"title"`
}

type importOfferingRow struct {
	ID            string   `json:"id"`
	CourseID      string   `json:"course_id"`
	Term          string   `json:"term"`
	Campus        string   `json:"campus"`
	Faculty       string   `json:"faculty"`
	InstructorIDs []string `json:"instructor_ids,omitempty"`
	// 班号信息（上游导出器提供）：教学班 code（如 32000101）与班名（如 01班）。
	// 用于详情页按班展示；旧数据包无此字段时为空。
	ClassCode string `json:"class_code,omitempty"`
	ClassName string `json:"class_name,omitempty"`
}

// --- 导入实现 ---

// ImportCatalog 导入课程目录包（courses/instructors/offerings JSONL）。
// dryRun=true 时只做校验与 canonicalization 检查，不写库。
// 幂等：manifest 内容哈希已存在且状态为 completed 时直接返回已存在报告；
// 行级幂等：course_source_ref.checksum 不变的行跳过。
func ImportCatalog(ctx context.Context, manifestPath string, dryRun bool) (*CatalogImportReport, error) {
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
	hash := sha256.Sum256(manifestBytes)
	manifestHash := hex.EncodeToString(hash[:])
	report := &CatalogImportReport{
		Source:       manifest.Source,
		ManifestHash: manifestHash,
		DryRun:       dryRun,
		Errors:       []ImportError{},
	}

	rows, fileCounts, err := loadManifestFiles(filepath.Dir(manifestPath), manifest)
	if err != nil {
		return nil, err
	}
	if err := validateManifestCounts(manifest, fileCounts); err != nil {
		return nil, err
	}
	report.TotalLines = len(rows.courses) + len(rows.instructors) + len(rows.offerings)

	if dryRun {
		validateRows(rows, report)
		return report, nil
	}

	if !dryRun {
		now := time.Now()
		run := course.ImportRunEntity{
			Source:       manifest.Source,
			ManifestHash: manifestHash,
			Kind:         course.ImportKindCatalog,
			Status:       course.ImportStatusRunning,
			StartedAt:    &now,
		}
		existing, err := course.GetImportRunByManifestHash(manifestHash, course.ImportKindCatalog)
		switch {
		case err == nil && existing.Status == course.ImportStatusCompleted:
			report.Skipped = 1 // 幂等：该 manifest 已导入完成
			return report, nil
		case err == nil:
			// 复用 failed 或 stale（running 超时，如进程崩溃残留）的 run：
			// 重置为 running，行级 checksum 幂等负责断点续跑。
			// running 且未超时视为"另一进程正在跑"，不抢（计数器竞争）。
			if existing.Status == course.ImportStatusRunning && !importRunStale(existing) {
				return nil, fmt.Errorf("import run %d is still running (started %s)", existing.Id, existing.StartedAt.Format(time.RFC3339))
			}
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

		if err := applyRows(ctx, run.Id, manifest.Source, rows, report); err != nil {
			run.Status = course.ImportStatusFailed
			run.ErrorCount = len(report.Errors)
			finished := time.Now()
			run.FinishedAt = &finished
			_ = course.SaveImportRun(&run)
			return report, err
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
	return report, nil
}

// importRows 一次导入包的全部行。
type importRows struct {
	courses     []importCourseRow
	instructors []importInstructorRow
	offerings   []importOfferingRow
}

// loadManifestFiles 读取并校验 manifest 列出的 JSONL 文件。
// 文件路径相对 manifest 所在目录解析，拒绝绝对路径与父目录引用。
// 返回各文件的解析行数（文件名 -> 非空行数），供 manifest.counts 校验。
func loadManifestFiles(manifestDir string, manifest ImportManifest) (importRows, map[string]int, error) {
	var rows importRows
	fileCounts := make(map[string]int, len(manifest.Files))
	// manifest.Files 是 map，迭代顺序随机；按文件名排序保证多 JSONL 文件
	// 的处理顺序确定（重复行"谁被隔离"在 dry-run 与真实导入间一致）。
	names := make([]string, 0, len(manifest.Files))
	for name := range manifest.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		// 同一 manifest 包可能同时携带其他子命令的文件（如 reviews.jsonl，上游
		// 导出器输出单包 4 文件）。本命令只消费自己关心的文件，其余不读取、不
		// 校验（文件完整性与计数由对应子命令 course-import reviews 负责）。
		if !strings.HasPrefix(name, "courses") && !strings.HasPrefix(name, "instructors") && !strings.HasPrefix(name, "offerings") {
			continue
		}
		wantSum := manifest.Files[name]
		if filepath.IsAbs(name) || strings.Contains(name, "..") {
			return rows, nil, fmt.Errorf("manifest file %q: absolute and parent paths are not allowed", name)
		}
		path := filepath.Join(manifestDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return rows, nil, fmt.Errorf("read %s: %w", path, err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != wantSum {
			return rows, nil, fmt.Errorf("checksum mismatch for %s: want %s got %s", name, wantSum, got)
		}
		var n int
		switch {
		case strings.HasPrefix(name, "courses"):
			n, err = parseJSONL(data, &rows.courses)
		case strings.HasPrefix(name, "instructors"):
			n, err = parseJSONL(data, &rows.instructors)
		case strings.HasPrefix(name, "offerings"):
			n, err = parseJSONL(data, &rows.offerings)
		}
		if err != nil {
			return rows, nil, fmt.Errorf("parse %s: %w", name, err)
		}
		fileCounts[name] = n
	}
	// 残缺包防护：manifest 声明了文件但没有任何本命令可消费的 catalog 文件
	// （例如误把 reviews-only 包交给 course-import），直接报错而不是"静默空成功"
	// 落一条 completed 的 run，避免运维误以为目录已导入。
	if len(rows.courses) == 0 && len(rows.instructors) == 0 && len(rows.offerings) == 0 && len(manifest.Files) > 0 {
		return rows, nil, fmt.Errorf("manifest contains no catalog files (want courses/instructors/offerings JSONL)")
	}
	return rows, fileCounts, nil
}

// validateManifestCounts 校验 manifest.counts 与各 JSONL 文件实际解析行数一致。
// 未声明 counts 的旧 manifest 不校验（向前兼容）；声明后逐文件强制一致，
// 不一致说明包被截断/篡改（计数是 sha256 之外的第二道完整性防线），
// 在 dry-run 阶段即拒绝，避免半包被静默导入。
// 语义：counts 缺失（yaml 无 counts 键）与显式空 counts（counts: {} 或
// counts: null，均解析为空 map）等同——都不触发校验，向后兼容旧包；
// 只要 counts 非空，则每个声明的文件都必须匹配。两条例外：
//   - counts 引用 manifest.files 之外的文件（typo/内部不一致）→ 仍拒绝；
//   - counts 引用 manifest.files 中属于其他子命令的文件（单包多命令，如
//     catalog 导入时 counts.reviews.jsonl）→ 跳过，由对应子命令校验。
func validateManifestCounts(manifest ImportManifest, fileCounts map[string]int) error {
	if len(manifest.Counts) == 0 {
		return nil
	}
	for name, want := range manifest.Counts {
		if _, declared := manifest.Files[name]; !declared {
			return fmt.Errorf("manifest counts: unknown file %s", name)
		}
		got, ok := fileCounts[name]
		if !ok {
			// 本命令不消费该文件（单包多命令场景），交由对应子命令校验。
			continue
		}
		if got != want {
			return fmt.Errorf("manifest counts mismatch for %s: want %d got %d", name, want, got)
		}
	}
	return nil
}

// parseJSONL 解析 JSONL 数据并追加到 out，返回非空行数。
func parseJSONL[T any](data []byte, out *[]T) (int, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	line := 0
	count := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var row T
		if err := json.Unmarshal([]byte(text), &row); err != nil {
			return 0, fmt.Errorf("line %d: %w", line, err)
		}
		*out = append(*out, row)
		count++
	}
	return count, scanner.Err()
}

// validateRows 校验行间约束与依赖；dry-run 与真实导入共用，保证语义一致。
// 返回被隔离的行集合（entityType|externalID -> true），applyRows 会跳过这些行。
func validateRows(rows importRows, report *CatalogImportReport) map[string]bool {
	quarantined := make(map[string]bool)
	quarantine := func(entity, key, reason string) {
		report.Quarantined++
		report.Errors = append(report.Errors, ImportError{Entity: entity, Reason: reason})
		if key != "" {
			quarantined[key] = true
		}
	}

	courseByCode := make(map[string]string) // primary_code -> external id
	externalIDs := make(map[string]string)  // external id -> primary_code
	for _, row := range rows.courses {
		code := strings.TrimSpace(row.Code)
		key := course.EntityTypeCourse + "|" + row.ID
		if strings.TrimSpace(row.ID) == "" {
			quarantine("course", key, "missing external id")
			continue
		}
		if prev, ok := externalIDs[row.ID]; ok && prev != code {
			quarantine("course", key, fmt.Sprintf("duplicate external id %s (codes %s vs %s)", row.ID, prev, code))
			continue
		}
		externalIDs[row.ID] = code
		if code == "" || strings.TrimSpace(row.Name) == "" {
			quarantine("course", key, "missing code or name")
			continue
		}
		if prev, ok := courseByCode[code]; ok && prev != row.ID {
			quarantine("course", key, fmt.Sprintf("duplicate primary_code %s (external %s vs %s)", code, prev, row.ID))
			continue
		}
		courseByCode[code] = row.ID
	}
	// alias 冲突检查：manifest 内 + 已入库
	batchAliases := make(map[string]string) // normalized -> course external id
	for _, row := range rows.courses {
		key := course.EntityTypeCourse + "|" + row.ID
		for _, alias := range row.Aliases {
			norm := Normalize(alias)
			if norm == "" {
				continue
			}
			if prev, ok := batchAliases[norm]; ok && prev != row.ID {
				quarantine("course", key, fmt.Sprintf("alias %q conflicts within manifest (course %s)", alias, prev))
				continue
			}
			batchAliases[norm] = row.ID
			existing, err := course.GetAliasByNormalizedValue(course.AliasKindName, norm)
			if err == nil {
				existingCourse := course.GetCourse(existing.CourseId)
				if existingCourse.PrimaryCode != row.Code {
					quarantine("course", key, fmt.Sprintf("alias %q conflicts with course %s", alias, existingCourse.PrimaryCode))
				}
			}
		}
	}
	instructorKeys := make(map[string]string) // name|dept -> external id
	instructorIDs := make(map[string]struct{})
	for _, row := range rows.instructors {
		key := course.EntityTypeInstructor + "|" + row.ID
		if strings.TrimSpace(row.ID) == "" {
			quarantine("instructor", key, "missing external id")
			continue
		}
		instructorIDs[row.ID] = struct{}{}
		naturalKey := Normalize(row.Name) + "|" + Normalize(row.Department)
		if prev, ok := instructorKeys[naturalKey]; ok && prev != row.ID {
			quarantine("instructor", key, fmt.Sprintf("ambiguous natural key %q (external %s vs %s)", naturalKey, prev, row.ID))
			delete(instructorIDs, row.ID)
			continue
		}
		instructorKeys[naturalKey] = row.ID
	}
	// offering 依赖检查：被隔离的 instructor 视为不可解析，引用它的 offering 也整体隔离。
	courseIDs := make(map[string]struct{})
	for _, row := range rows.courses {
		courseIDs[row.ID] = struct{}{}
	}
	for _, row := range rows.offerings {
		key := course.EntityTypeOffering + "|" + row.ID
		if strings.TrimSpace(row.ID) == "" {
			quarantine("offering", key, "missing external id")
			continue
		}
		if _, ok := courseIDs[row.CourseID]; !ok {
			quarantine("offering", key, fmt.Sprintf("unknown course_id %s", row.CourseID))
			continue
		}
		if strings.TrimSpace(row.Term) == "" {
			quarantine("offering", key, "missing term")
			continue
		}
		for _, insID := range row.InstructorIDs {
			if _, ok := instructorIDs[insID]; !ok {
				quarantine("offering", key, fmt.Sprintf("unresolvable instructor_id %s", insID))
				break
			}
		}
	}
	return quarantined
}

// applyRows 实际写入：先做行间校验（与 dry-run 一致），每批 500 行一个事务。
// source 为 manifest 来源标识，贯穿所有 source_ref 读写，保证多来源隔离。
func applyRows(ctx context.Context, runID uint64, source string, rows importRows, report *CatalogImportReport) error {
	quarantined := validateRows(rows, report)
	var ops []func(*gorm.DB) error

	for _, row := range rows.courses {
		row := row
		if quarantined[course.EntityTypeCourse+"|"+row.ID] {
			continue
		}
		ops = append(ops, func(tx *gorm.DB) error {
			return applyCourseRow(tx, runID, source, row, report)
		})
	}
	for _, row := range rows.instructors {
		row := row
		if quarantined[course.EntityTypeInstructor+"|"+row.ID] {
			continue
		}
		ops = append(ops, func(tx *gorm.DB) error {
			return applyInstructorRow(tx, runID, source, row, report)
		})
	}
	for _, row := range rows.offerings {
		row := row
		if quarantined[course.EntityTypeOffering+"|"+row.ID] {
			continue
		}
		ops = append(ops, func(tx *gorm.DB) error {
			return applyOfferingRow(tx, runID, source, row, report)
		})
	}

	const batchSize = 500
	db := dbconnect.Connect()
	for i := 0; i < len(ops); i += batchSize {
		end := i + batchSize
		if end > len(ops) {
			end = len(ops)
		}
		batch := ops[i:end]
		// 批次失败回滚时同步回滚报告计数，避免失败报告多算。
		snapshot := *report
		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, fn := range batch {
				if err := fn(tx); err != nil {
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

func rowChecksum(row any) string {
	data, err := json.Marshal(row)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// mapKeys 返回 map 的键切片（用于 GORM NOT IN 查询）。
func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func applyCourseRow(tx *gorm.DB, runID uint64, source string, row importCourseRow, report *CatalogImportReport) error {
	code := strings.TrimSpace(row.Code)
	name := strings.TrimSpace(row.Name)
	if code == "" || name == "" {
		report.Quarantined++
		return nil
	}
	checksum := rowChecksum(row)
	ref, refErr := sourceRefByExternal(tx, source, row.ID, course.EntityTypeCourse)
	if refErr == nil && ref.Checksum == checksum {
		report.Skipped++ // 行内容未变化
		return nil
	}

	pinyin, initials := searchservice.PinyinFields(name)
	entity := course.Entity{
		PrimaryCode:    code,
		Name:           name,
		Department:     row.Department,
		CreditX10:      int(math.Round(row.Credit * 10)),
		NormalizedName: Normalize(name),
		NamePinyin:     pinyin,
		NameInitials:   initials,
		Status:         course.StatusVisible,
	}

	switch {
	case refErr == nil:
		// 已有 source mapping：primary_code 可变，external id 才是稳定身份。
		// 若主课号变化，直接更新映射实体，避免创建第二门课程。
		entity.Id = ref.LocalId
		if err := tx.Model(&course.Entity{}).Where("id = ?", ref.LocalId).Updates(map[string]any{
			"primary_code":    entity.PrimaryCode,
			"name":            entity.Name,
			"department":      entity.Department,
			"credit_x10":      entity.CreditX10,
			"normalized_name": entity.NormalizedName,
			"name_pinyin":     entity.NamePinyin,
			"name_initials":   entity.NameInitials,
			// 注意：更新路径不写 status——管理员隐藏的课程不能被重导静默复活。
		}).Error; err != nil {
			return fmt.Errorf("update course %s (id %d): %w", code, ref.LocalId, err)
		}
		report.Updated++
	case errors.Is(refErr, gorm.ErrRecordNotFound):
		// 兼容旧导入数据：按主课号查找；找不到则创建。
		existing, err := course.GetCourseByPrimaryCodeTx(tx, code)
		if err == nil {
			entity.Id = existing.Id
			if err := tx.Model(&course.Entity{}).Where("id = ?", existing.Id).Updates(map[string]any{
				"primary_code":    entity.PrimaryCode,
				"name":            entity.Name,
				"department":      entity.Department,
				"credit_x10":      entity.CreditX10,
				"normalized_name": entity.NormalizedName,
				"name_pinyin":     entity.NamePinyin,
				"name_initials":   entity.NameInitials,
				// 更新路径不写 status，避免复活管理员隐藏的课程。
			}).Error; err != nil {
				return fmt.Errorf("update course %s: %w", code, err)
			}
			report.Updated++
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("lookup course %s: %w", code, err)
		} else {
			if err := tx.Model(&course.Entity{}).Create(&entity).Error; err != nil {
				return fmt.Errorf("create course %s: %w", code, err)
			}
			report.Inserted++
		}
	default:
		return fmt.Errorf("lookup course source ref %s: %w", row.ID, refErr)
	}

	// aliases：事务内全量 reconcile（来源行删除的别名一并移除）。
	normSet := make(map[string]struct{}, len(row.Aliases))
	for _, alias := range row.Aliases {
		norm := Normalize(alias)
		if norm == "" {
			continue
		}
		normSet[norm] = struct{}{}
		existingAlias, err := course.GetAliasByNormalizedValueTx(tx, course.AliasKindName, norm)
		if err == nil {
			if existingAlias.CourseId != entity.Id {
				report.Quarantined++
				continue
			}
			// 同课程已存在：若此前被软删（唯一索引仍占位），恢复该行
			// 并更新 value/source，避免后续重新创建时撞唯一键。
			if existingAlias.DeletedAt.Valid {
				if err := tx.Unscoped().Model(&course.AliasEntity{}).
					Where("id = ?", existingAlias.Id).
					Updates(map[string]any{
						"value":      alias,
						"source":     row.ID,
						"deleted_at": nil,
					}).Error; err != nil {
					return fmt.Errorf("restore alias %q: %w", alias, err)
				}
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("lookup alias %q: %w", alias, err)
		}
		aliasEntity := course.AliasEntity{
			CourseId:        entity.Id,
			Kind:            course.AliasKindName,
			Value:           alias,
			NormalizedValue: norm,
			Source:          row.ID,
		}
		if err := tx.Model(&course.AliasEntity{}).Create(&aliasEntity).Error; err != nil {
			return fmt.Errorf("create alias %q: %w", alias, err)
		}
	}
	// 移除该来源行此前导入、本次已删除的别名（source 字段记录外部课程 ID）。
	// 物理删除：软删行仍占据唯一索引 (kind, normalized_value)，后续 manifest
	// 重新添加同一 alias 会撞唯一键；物理删除让 remove→re-add 生命周期可用。
	reconcile := tx.Unscoped().Model(&course.AliasEntity{}).
		Where("course_id = ? AND kind = ? AND source = ?", entity.Id, course.AliasKindName, row.ID)
	if len(normSet) > 0 {
		reconcile = reconcile.Where("normalized_value NOT IN ?", mapKeys(normSet))
	}
	if err := reconcile.Delete(&course.AliasEntity{}).Error; err != nil {
		return fmt.Errorf("reconcile aliases for course %s: %w", code, err)
	}
	// 导入写入后同事务入队搜索同步（transaction-bound outbox）
	if err := searchservice.EnqueueCourseSearchTask(tx, entity.Id); err != nil {
		return fmt.Errorf("enqueue course search task %d: %w", entity.Id, err)
	}
	return touchSourceRef(tx, runID, source, row.ID, entity.Id, course.EntityTypeCourse, checksum)
}

func applyInstructorRow(tx *gorm.DB, runID uint64, source string, row importInstructorRow, report *CatalogImportReport) error {
	name := strings.TrimSpace(row.Name)
	if name == "" {
		report.Quarantined++
		return nil
	}
	checksum := rowChecksum(row)
	ref, refErr := sourceRefByExternal(tx, source, row.ID, course.EntityTypeInstructor)
	if refErr == nil && ref.Checksum == checksum {
		report.Skipped++
		return nil
	}

	pinyin, initials := searchservice.PinyinFields(name)
	norm := Normalize(name)
	dept := row.Department
	updates := map[string]any{
		"name":            name,
		"normalized_name": norm,
		"name_pinyin":     pinyin,
		"name_initials":   initials,
		"department":      dept,
		"title":           row.Title,
	}
	var instructorID uint64
	switch {
	case refErr == nil:
		// source_ref 提供稳定 external id：教师改名/转院时优先按 LocalId 更新，
		// 避免按 (name, dept) 自然键找不到而创建新教师、旧教师成孤儿。
		instructorID = ref.LocalId
		if err := tx.Model(&course.InstructorEntity{}).Where("id = ?", instructorID).Updates(updates).Error; err != nil {
			return fmt.Errorf("update instructor %s (id %d): %w", name, instructorID, err)
		}
		report.Updated++
	case errors.Is(refErr, gorm.ErrRecordNotFound):
		// 兼容旧导入数据：按自然键查找；找不到则创建。
		existing, err := course.FindInstructorByNameDeptTx(tx, norm, dept)
		if err == nil {
			instructorID = existing.Id
			if err := tx.Model(&course.InstructorEntity{}).Where("id = ?", instructorID).Updates(updates).Error; err != nil {
				return fmt.Errorf("update instructor %s: %w", name, err)
			}
			report.Updated++
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("lookup instructor %s: %w", name, err)
		} else {
			entity := course.InstructorEntity{
				Name:           name,
				NormalizedName: norm,
				NamePinyin:     pinyin,
				NameInitials:   initials,
				Department:     dept,
				Title:          row.Title,
				Status:         0,
			}
			if err := tx.Model(&course.InstructorEntity{}).Create(&entity).Error; err != nil {
				return fmt.Errorf("create instructor %s: %w", name, err)
			}
			instructorID = entity.Id
			report.Inserted++
		}
	default:
		return fmt.Errorf("lookup instructor source ref %s: %w", row.ID, refErr)
	}
	// 教师变更（改名/转院）会改变课程搜索文档的 instructors 字段：
	// fan-out 到所有引用该教师的可见 offering 所属课程，同事务入队重建。
	courseIDs, err := course.ListCourseIDsByInstructorTx(tx, instructorID)
	if err != nil {
		return fmt.Errorf("list courses of instructor %d: %w", instructorID, err)
	}
	for _, cid := range courseIDs {
		if err := searchservice.EnqueueCourseSearchTask(tx, cid); err != nil {
			return fmt.Errorf("enqueue course search task %d: %w", cid, err)
		}
	}
	return touchSourceRef(tx, runID, source, row.ID, instructorID, course.EntityTypeInstructor, checksum)
}

func applyOfferingRow(tx *gorm.DB, runID uint64, source string, row importOfferingRow, report *CatalogImportReport) error {
	courseLocalID, err := sourceRefLocalID(tx, source, row.CourseID, course.EntityTypeCourse)
	if err != nil {
		report.Quarantined++
		return nil
	}
	checksum := rowChecksum(row)
	termEntity, err := getOrCreateTermTx(tx, strings.TrimSpace(row.Term))
	if err != nil {
		return err
	}
	// 任一教师引用不可解析则整个 offering 隔离，不部分写入。
	instructorLocalIDs := make([]uint64, 0, len(row.InstructorIDs))
	for _, insID := range row.InstructorIDs {
		ins, err := sourceRefLocalID(tx, source, insID, course.EntityTypeInstructor)
		if err != nil {
			report.Quarantined++
			return nil
		}
		instructorLocalIDs = append(instructorLocalIDs, ins)
	}

	existingRef, refErr := sourceRefByExternal(tx, source, row.ID, course.EntityTypeOffering)
	if refErr == nil {
		if existingRef.Checksum == checksum {
			report.Skipped++ // 内容未变化
			return nil
		}
		// 断点续跑/重试：按 source 映射更新已存在的 offering，避免重复插入。
		var offering course.OfferingEntity
		if err := tx.Model(&course.OfferingEntity{}).Where("id = ?", existingRef.LocalId).First(&offering).Error; err != nil {
			return fmt.Errorf("load existing offering %d: %w", existingRef.LocalId, err)
		}
		// 课程修正也一并更新，防止 offering 挂在旧课程上。
		// 注意：更新路径不写 status——管理员隐藏的 offering 不能被重导静默复活。
		updates := map[string]any{
			"course_id":  courseLocalID,
			"term_id":    termEntity.Id,
			"campus":     row.Campus,
			"faculty":    row.Faculty,
			"class_code": row.ClassCode,
			"class_name": row.ClassName,
		}
		if err := tx.Model(&course.OfferingEntity{}).Where("id = ?", offering.Id).Updates(updates).Error; err != nil {
			return fmt.Errorf("update offering %d: %w", offering.Id, err)
		}
		if err := replaceOfferingInstructorsTx(tx, offering.Id, instructorLocalIDs); err != nil {
			return err
		}
		// offering 的 term/campus/教师/所属课程变化会改变课程搜索文档的
		// terms/campus/instructors 字段：新旧课程都入队重建。
		if offering.CourseId != courseLocalID {
			if err := searchservice.EnqueueCourseSearchTask(tx, offering.CourseId); err != nil {
				return fmt.Errorf("enqueue old course search task %d: %w", offering.CourseId, err)
			}
		}
		if err := searchservice.EnqueueCourseSearchTask(tx, courseLocalID); err != nil {
			return fmt.Errorf("enqueue course search task %d: %w", courseLocalID, err)
		}
		report.Updated++
		return touchSourceRef(tx, runID, source, row.ID, offering.Id, course.EntityTypeOffering, checksum)
	}
	if !errors.Is(refErr, gorm.ErrRecordNotFound) {
		return fmt.Errorf("lookup offering source ref: %w", refErr)
	}

	offeringEntity := course.OfferingEntity{
		CourseId:  courseLocalID,
		TermId:    termEntity.Id,
		Campus:    row.Campus,
		Faculty:   row.Faculty,
		ClassCode: row.ClassCode,
		ClassName: row.ClassName,
		Status:    course.OfferingStatusVisible,
	}
	if err := tx.Model(&course.OfferingEntity{}).Create(&offeringEntity).Error; err != nil {
		return fmt.Errorf("create offering: %w", err)
	}
	if err := replaceOfferingInstructorsTx(tx, offeringEntity.Id, instructorLocalIDs); err != nil {
		return err
	}
	report.Inserted++
	return touchSourceRef(tx, runID, source, row.ID, offeringEntity.Id, course.EntityTypeOffering, checksum)
}

// getOrCreateTermTx 事务内按 code 查找学期，不存在则创建（事务内可见，避免同批次重复建）。
func getOrCreateTermTx(tx *gorm.DB, code string) (course.TermEntity, error) {
	if code == "" {
		return course.TermEntity{}, fmt.Errorf("term code is empty")
	}
	termEntity, err := course.GetTermByCodeTx(tx, code)
	if err == nil {
		return termEntity, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return course.TermEntity{}, fmt.Errorf("lookup term %s: %w", code, err)
	}
	termEntity = course.TermEntity{Code: code, Name: code, Status: 0}
	if err := tx.Model(&course.TermEntity{}).Create(&termEntity).Error; err != nil {
		return course.TermEntity{}, fmt.Errorf("create term %s: %w", code, err)
	}
	return termEntity, nil
}

// replaceOfferingInstructorsTx 全量替换某 offering 的教师关联（事务内）。
func replaceOfferingInstructorsTx(tx *gorm.DB, offeringId uint64, instructorIDs []uint64) error {
	if err := tx.Model(&course.OfferingInstructorEntity{}).
		Where("offering_id = ?", offeringId).
		Delete(&course.OfferingInstructorEntity{}).Error; err != nil {
		return fmt.Errorf("clear offering instructors %d: %w", offeringId, err)
	}
	for _, id := range instructorIDs {
		rel := course.OfferingInstructorEntity{OfferingId: offeringId, InstructorId: id, Role: "lecturer"}
		if err := tx.Model(&course.OfferingInstructorEntity{}).Create(&rel).Error; err != nil {
			return fmt.Errorf("create offering instructor link: %w", err)
		}
	}
	return nil
}

// sourceRefLocalID 通过 source mapping 反查本地 ID（事务内）。
// source 为当前 manifest 的来源标识：来源映射以 (source, entity_type, external_id) 为键，
// 不同来源对同一 external id 互不干扰（幂等/重试各自独立）。
func sourceRefLocalID(tx *gorm.DB, source, externalID, entityType string) (uint64, error) {
	ref, err := sourceRefByExternal(tx, source, externalID, entityType)
	if err != nil {
		return 0, err
	}
	return ref.LocalId, nil
}

// sourceRefByExternal 按 (source, entity_type, external_id) 查找来源映射（事务内）。
func sourceRefByExternal(tx *gorm.DB, source, externalID, entityType string) (course.SourceRefEntity, error) {
	var ref course.SourceRefEntity
	err := tx.Model(&course.SourceRefEntity{}).
		Where("source = ? AND entity_type = ? AND external_id = ?", source, entityType, externalID).
		First(&ref).Error
	return ref, err
}

// touchSourceRef upsert 来源映射并记录行 checksum（事务内）。
func touchSourceRef(tx *gorm.DB, runID uint64, source, externalID string, localID uint64, entityType, checksum string) error {
	var existing course.SourceRefEntity
	err := tx.Model(&course.SourceRefEntity{}).
		Where("source = ? AND entity_type = ? AND external_id = ?", source, entityType, externalID).
		First(&existing).Error
	if err == nil {
		return tx.Model(&course.SourceRefEntity{}).Where("id = ?", existing.Id).Updates(map[string]any{
			"local_id": localID,
			"checksum": checksum,
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	entity := course.SourceRefEntity{
		ImportRunId: runID,
		Source:      source,
		EntityType:  entityType,
		ExternalId:  externalID,
		LocalId:     localID,
		Checksum:    checksum,
	}
	return tx.Model(&course.SourceRefEntity{}).Create(&entity).Error
}

// importRunStale 判断 running 态 run 是否已超时（进程崩溃残留）。
// 超过 1 小时视为 stale，可安全复用；未超时视为另一进程正在跑。
func importRunStale(run course.ImportRunEntity) bool {
	if run.StartedAt == nil {
		return true // 无开始时间视为残留
	}
	return time.Since(*run.StartedAt) > time.Hour
}
