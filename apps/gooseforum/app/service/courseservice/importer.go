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
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/leancodebox/GooseForum/app/service/searchservice"
	"go.yaml.in/yaml/v3"
	"gorm.io/gorm"
)

// ImportSource 目录导入的来源标识，写入 course_source_ref.source。
const ImportSource = "course-import"

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

	rows, err := loadManifestFiles(filepath.Dir(manifestPath), manifest)
	if err != nil {
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

		if err := applyRows(ctx, run.Id, rows, report); err != nil {
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
func loadManifestFiles(manifestDir string, manifest ImportManifest) (importRows, error) {
	var rows importRows
	for name, wantSum := range manifest.Files {
		if filepath.IsAbs(name) || strings.Contains(name, "..") {
			return rows, fmt.Errorf("manifest file %q: absolute and parent paths are not allowed", name)
		}
		path := filepath.Join(manifestDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return rows, fmt.Errorf("read %s: %w", path, err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != wantSum {
			return rows, fmt.Errorf("checksum mismatch for %s: want %s got %s", name, wantSum, got)
		}
		switch {
		case strings.HasPrefix(name, "courses"):
			if err := parseJSONL(data, &rows.courses); err != nil {
				return rows, fmt.Errorf("parse %s: %w", name, err)
			}
		case strings.HasPrefix(name, "instructors"):
			if err := parseJSONL(data, &rows.instructors); err != nil {
				return rows, fmt.Errorf("parse %s: %w", name, err)
			}
		case strings.HasPrefix(name, "offerings"):
			if err := parseJSONL(data, &rows.offerings); err != nil {
				return rows, fmt.Errorf("parse %s: %w", name, err)
			}
		default:
			return rows, fmt.Errorf("unexpected manifest file %s", name)
		}
	}
	return rows, nil
}

func parseJSONL[T any](data []byte, out *[]T) error {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var row T
		if err := json.Unmarshal([]byte(text), &row); err != nil {
			return fmt.Errorf("line %d: %w", line, err)
		}
		*out = append(*out, row)
	}
	return scanner.Err()
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
func applyRows(ctx context.Context, runID uint64, rows importRows, report *CatalogImportReport) error {
	quarantined := validateRows(rows, report)
	var ops []func(*gorm.DB) error

	for _, row := range rows.courses {
		row := row
		if quarantined[course.EntityTypeCourse+"|"+row.ID] {
			continue
		}
		ops = append(ops, func(tx *gorm.DB) error {
			return applyCourseRow(tx, runID, row, report)
		})
	}
	for _, row := range rows.instructors {
		row := row
		if quarantined[course.EntityTypeInstructor+"|"+row.ID] {
			continue
		}
		ops = append(ops, func(tx *gorm.DB) error {
			return applyInstructorRow(tx, runID, row, report)
		})
	}
	for _, row := range rows.offerings {
		row := row
		if quarantined[course.EntityTypeOffering+"|"+row.ID] {
			continue
		}
		ops = append(ops, func(tx *gorm.DB) error {
			return applyOfferingRow(tx, runID, row, report)
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

func applyCourseRow(tx *gorm.DB, runID uint64, row importCourseRow, report *CatalogImportReport) error {
	code := strings.TrimSpace(row.Code)
	name := strings.TrimSpace(row.Name)
	if code == "" || name == "" {
		report.Quarantined++
		return nil
	}
	checksum := rowChecksum(row)
	ref, refErr := sourceRefByExternal(tx, row.ID, course.EntityTypeCourse)
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
			"status":          entity.Status,
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
				"status":          entity.Status,
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
			continue // 同课程已存在
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
	reconcile := tx.Model(&course.AliasEntity{}).
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
	return touchSourceRef(tx, runID, row.ID, entity.Id, course.EntityTypeCourse, checksum)
}

func applyInstructorRow(tx *gorm.DB, runID uint64, row importInstructorRow, report *CatalogImportReport) error {
	name := strings.TrimSpace(row.Name)
	if name == "" {
		report.Quarantined++
		return nil
	}
	checksum := rowChecksum(row)
	if ref, err := sourceRefByExternal(tx, row.ID, course.EntityTypeInstructor); err == nil && ref.Checksum == checksum {
		report.Skipped++
		return nil
	}

	pinyin, initials := searchservice.PinyinFields(name)
	norm := Normalize(name)
	dept := row.Department
	existing, err := course.FindInstructorByNameDeptTx(tx, norm, dept)
	if err == nil {
		updates := map[string]any{
			"name":            name,
			"normalized_name": norm,
			"name_pinyin":     pinyin,
			"name_initials":   initials,
			"department":      dept,
			"title":           row.Title,
		}
		if err := tx.Model(&course.InstructorEntity{}).Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
			return fmt.Errorf("update instructor %s: %w", name, err)
		}
		report.Updated++
		return touchSourceRef(tx, runID, row.ID, existing.Id, course.EntityTypeInstructor, checksum)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("lookup instructor %s: %w", name, err)
	}
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
	report.Inserted++
	return touchSourceRef(tx, runID, row.ID, entity.Id, course.EntityTypeInstructor, checksum)
}

func applyOfferingRow(tx *gorm.DB, runID uint64, row importOfferingRow, report *CatalogImportReport) error {
	courseLocalID, err := sourceRefLocalID(tx, row.CourseID, course.EntityTypeCourse)
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
		ins, err := sourceRefLocalID(tx, insID, course.EntityTypeInstructor)
		if err != nil {
			report.Quarantined++
			return nil
		}
		instructorLocalIDs = append(instructorLocalIDs, ins)
	}

	existingRef, refErr := sourceRefByExternal(tx, row.ID, course.EntityTypeOffering)
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
		updates := map[string]any{
			"course_id": courseLocalID,
			"term_id":   termEntity.Id,
			"campus":    row.Campus,
			"faculty":   row.Faculty,
			"status":    course.OfferingStatusVisible,
		}
		if err := tx.Model(&course.OfferingEntity{}).Where("id = ?", offering.Id).Updates(updates).Error; err != nil {
			return fmt.Errorf("update offering %d: %w", offering.Id, err)
		}
		if err := replaceOfferingInstructorsTx(tx, offering.Id, instructorLocalIDs); err != nil {
			return err
		}
		report.Updated++
		return touchSourceRef(tx, runID, row.ID, offering.Id, course.EntityTypeOffering, checksum)
	}
	if !errors.Is(refErr, gorm.ErrRecordNotFound) {
		return fmt.Errorf("lookup offering source ref: %w", refErr)
	}

	offeringEntity := course.OfferingEntity{
		CourseId: courseLocalID,
		TermId:   termEntity.Id,
		Campus:   row.Campus,
		Faculty:  row.Faculty,
		Status:   course.OfferingStatusVisible,
	}
	if err := tx.Model(&course.OfferingEntity{}).Create(&offeringEntity).Error; err != nil {
		return fmt.Errorf("create offering: %w", err)
	}
	if err := replaceOfferingInstructorsTx(tx, offeringEntity.Id, instructorLocalIDs); err != nil {
		return err
	}
	report.Inserted++
	return touchSourceRef(tx, runID, row.ID, offeringEntity.Id, course.EntityTypeOffering, checksum)
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
func sourceRefLocalID(tx *gorm.DB, externalID, entityType string) (uint64, error) {
	ref, err := sourceRefByExternal(tx, externalID, entityType)
	if err != nil {
		return 0, err
	}
	return ref.LocalId, nil
}

// sourceRefByExternal 按 (source, entity_type, external_id) 查找来源映射（事务内）。
func sourceRefByExternal(tx *gorm.DB, externalID, entityType string) (course.SourceRefEntity, error) {
	var ref course.SourceRefEntity
	err := tx.Model(&course.SourceRefEntity{}).
		Where("source = ? AND entity_type = ? AND external_id = ?", ImportSource, entityType, externalID).
		First(&ref).Error
	return ref, err
}

// touchSourceRef upsert 来源映射并记录行 checksum（事务内）。
func touchSourceRef(tx *gorm.DB, runID uint64, externalID string, localID uint64, entityType, checksum string) error {
	var existing course.SourceRefEntity
	err := tx.Model(&course.SourceRefEntity{}).
		Where("source = ? AND entity_type = ? AND external_id = ?", ImportSource, entityType, externalID).
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
		Source:      ImportSource,
		EntityType:  entityType,
		ExternalId:  externalID,
		LocalId:     localID,
		Checksum:    checksum,
	}
	return tx.Model(&course.SourceRefEntity{}).Create(&entity).Error
}
