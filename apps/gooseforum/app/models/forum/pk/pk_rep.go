package pk

import (
	"fmt"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- 学期/日志 ----

// GetCalendarIdByI18n 按学期标记（如 "2025-2026-1"）反查一系统 calendarId。
func GetCalendarIdByI18n(i18n string) (uint64, bool) {
	var entity CalendarEntity
	err := calendarBuilder().Where(queryopt.Eq("calendar_id_i18n", i18n)).First(&entity).Error
	if err != nil || entity.CalendarId == 0 {
		return 0, false
	}
	return entity.CalendarId, true
}

// LatestFetchLogByCalendar 返回某学期最近一次同步日志（按 id 倒序取最新）。
func LatestFetchLogByCalendar(calendarId uint64) (FetchLogEntity, bool) {
	var entity FetchLogEntity
	err := fetchLogBuilder().Where(queryopt.Eq("calendar_id", calendarId)).Order("id DESC").First(&entity).Error
	if err != nil {
		return FetchLogEntity{}, false
	}
	return entity, true
}

// CreateFetchLog 新建同步日志（running 初始态），返回带主键的实体。
// 新建即占用 running_key 唯一索引（running_key=calendar_id），兜底「两进程同时读不到 running、
// 各自 Create」的 TOCTOU 竞态（见 FetchLogEntity 注释）。
func CreateFetchLog(calendarId uint64) (*FetchLogEntity, error) {
	now := time.Now()
	entity := &FetchLogEntity{
		CalendarId: calendarId,
		RunningKey: &calendarId,
		Status:     FetchStatusRunning,
		StartedAt:  &now,
	}
	if err := fetchLogBuilder().Create(entity).Error; err != nil {
		return nil, fmt.Errorf("pk: create fetch log: %w", err)
	}
	return entity, nil
}

// SaveFetchLog 更新同步日志游标/状态（不重建行，更新 CreatedAt 之外的字段）。
// running_key 由 status 派生：running 时置 calendar_id，其余（completed/failed）置 NULL——
// 调用方不可能写出行与 running_key 不一致的脏状态。
func SaveFetchLog(entity *FetchLogEntity) error {
	if entity.Status == FetchStatusRunning {
		entity.RunningKey = &entity.CalendarId
	} else {
		entity.RunningKey = nil
	}
	if err := fetchLogBuilder().Where("id = ?", entity.Id).Select(
		"status", "total_pages", "last_committed_page", "rows_written", "error_msg", "started_at", "finished_at", "running_key", "updated_at",
	).Updates(entity).Error; err != nil {
		return fmt.Errorf("pk: save fetch log: %w", err)
	}
	return nil
}

// ClaimFetchLog 原子认领一条可续跑日志：仅当其当前状态等于 expectedStatus 且 lease_version 等于
// expectedVersion 时，将其置为 running、刷新 started_at 并把 lease_version 加 1。返回是否认领成功
// （RowsAffected==1）。
//
// 用于消除「两进程同时读到同一条 stale-running / failed 日志并都续跑」的 double-delete 竞态
// （review HIGH「唯一/租约」的租约部分）。lease_version 是单调递增的精确 CAS/lease token：赢家把
// 它递增，并发者用旧 version 重试时 WHERE lease_version=<旧值> 不再匹配，RowsAffected==0，故只有
// 一个成功。不用 started_at 时间戳做 token——不同方言时间精度不一致，可能两进程读到同一值导致
// CAS 失效（review P1）。对 failed 也传当前 version：状态转换 failed→running 本身可串行化，
// 加上 version 匹配更精确。
func ClaimFetchLog(id uint64, expectedStatus string, expectedVersion int) (bool, error) {
	now := time.Now()
	res := fetchLogBuilder().
		Where("id = ? AND status = ? AND lease_version = ?", id, expectedStatus, expectedVersion).
		Updates(map[string]any{
			"status":        FetchStatusRunning,
			"lease_version": gorm.Expr("lease_version + 1"),
			"started_at":    now,
			"running_key":   gorm.Expr("calendar_id"), // 认领后即 running：占用 running 唯一索引。
		})
	if res.Error != nil {
		return false, fmt.Errorf("pk: claim fetch log: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

// CleanupOldFetchLogs 清理 30 天前的同步日志。
// 必须 Unscoped 物理删除：FetchLogEntity 带 gorm.DeletedAt，普通 Delete 是软删，
// stale running 行软删后仍占用 running_key 唯一索引（uniq_pk_fetch_log_running_key），
// 后续同 calendar 的 CreateFetchLog 会唯一冲突（与 DeleteCalendarDataTx 硬删理由一致）。
func CleanupOldFetchLogs() error {
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	if err := fetchLogBuilder().Unscoped().Where("created_at < ?", cutoff).Delete(&FetchLogEntity{}).Error; err != nil {
		return fmt.Errorf("pk: cleanup fetch logs: %w", err)
	}
	return nil
}

// ---- 批量删除 ----

// DeleteCalendarDataTx 事务内按学期清空排课数据：先按教学班 id 分块删除关联，
// 再删课程/学期/课程性质快照。幂等全量重写（每次同步删除后重插，避免残留陈旧数据）。
// 全部硬删除：软删行仍占据主键，后续 upsert 冲突会导致"更新了不可见的行"。
// 注意：必须用 tx 查询教学班 id，避免在事务内走独立连接造成连接池死锁。
func DeleteCalendarDataTx(tx *gorm.DB, calendarId uint64) error {
	if err := tx.Unscoped().Where("calendar_id = ?", calendarId).Delete(&TeacherTimeslotEntity{}).Error; err != nil {
		return fmt.Errorf("pk: delete timeslots: %w", err)
	}
	var classIds []uint64
	if err := tx.Table(courseDetailTableName).Where("calendar_id = ?", calendarId).Pluck("id", &classIds).Error; err != nil {
		return fmt.Errorf("pk: list course detail ids: %w", err)
	}
	const chunkSize = 80
	for i := 0; i < len(classIds); i += chunkSize {
		end := i + chunkSize
		if end > len(classIds) {
			end = len(classIds)
		}
		chunk := classIds[i:end]
		if err := tx.Unscoped().Where("teaching_class_id IN ?", chunk).Delete(&TeacherEntity{}).Error; err != nil {
			return fmt.Errorf("pk: delete teachers: %w", err)
		}
		if err := tx.Unscoped().Where("course_id IN ?", chunk).Delete(&MajorCourseEntity{}).Error; err != nil {
			return fmt.Errorf("pk: delete major courses: %w", err)
		}
	}
	if err := tx.Unscoped().Where("calendar_id = ?", calendarId).Delete(&CourseDetailEntity{}).Error; err != nil {
		return fmt.Errorf("pk: delete course details: %w", err)
	}
	if err := tx.Unscoped().Where("calendar_id = ?", calendarId).Delete(&CalendarEntity{}).Error; err != nil {
		return fmt.Errorf("pk: delete calendar: %w", err)
	}
	if err := tx.Unscoped().Where("calendar_id = ?", calendarId).Delete(&CourseNatureByCalendarEntity{}).Error; err != nil {
		return fmt.Errorf("pk: delete course nature by calendar: %w", err)
	}
	return nil
}

// DeleteTeacherTimeslotsTx 事务内删除指定学期的时间片（重建前的清空步骤）。
func DeleteTeacherTimeslotsTx(tx *gorm.DB, calendarIds []uint64) error {
	if len(calendarIds) == 0 {
		return nil
	}
	if err := tx.Where("calendar_id IN ?", calendarIds).Delete(&TeacherTimeslotEntity{}).Error; err != nil {
		return fmt.Errorf("pk: delete timeslots for calendars: %w", err)
	}
	return nil
}

// ---- 批量 upsert ----

// chunkedUpsertTx 分块执行带 OnConflict 的批量插入，避免单条 INSERT 参数超过数据库上限。
func chunkedUpsertTx[T any](tx *gorm.DB, rows []T, onConflict clause.OnConflict, chunkSize int) error {
	for i := 0; i < len(rows); i += chunkSize {
		end := i + chunkSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := tx.Clauses(onConflict).Create(rows[i:end]).Error; err != nil {
			return err
		}
	}
	return nil
}

const upsertChunkSize = 500

func upsertSingleKeyTx(tx *gorm.DB, rows any, pkColumn string) error {
	onConflict := clause.OnConflict{Columns: []clause.Column{{Name: pkColumn}}, UpdateAll: true}
	switch v := rows.(type) {
	case []CourseDetailEntity:
		return chunkedUpsertTx(tx, v, onConflict, upsertChunkSize)
	case []TeacherEntity:
		return chunkedUpsertTx(tx, v, onConflict, upsertChunkSize)
	case []CalendarEntity:
		return chunkedUpsertTx(tx, v, onConflict, upsertChunkSize)
	case []LanguageEntity:
		return chunkedUpsertTx(tx, v, onConflict, upsertChunkSize)
	case []CourseNatureEntity:
		return chunkedUpsertTx(tx, v, onConflict, upsertChunkSize)
	case []AssessmentEntity:
		return chunkedUpsertTx(tx, v, onConflict, upsertChunkSize)
	case []CampusEntity:
		return chunkedUpsertTx(tx, v, onConflict, upsertChunkSize)
	case []FacultyEntity:
		return chunkedUpsertTx(tx, v, onConflict, upsertChunkSize)
	case []MajorEntity:
		return chunkedUpsertTx(tx, v, onConflict, upsertChunkSize)
	default:
		return fmt.Errorf("pk: unsupported upsert type %T", rows)
	}
}

// UpsertCourseDetailsTx 按教学班 id upsert（INSERT ... ON CONFLICT DO UPDATE，等价上游 INSERT OR REPLACE）。
func UpsertCourseDetailsTx(tx *gorm.DB, rows []CourseDetailEntity) error {
	if len(rows) == 0 {
		return nil
	}
	return upsertSingleKeyTx(tx, rows, "id")
}

// UpsertTeachersTx 按教师 id upsert。
func UpsertTeachersTx(tx *gorm.DB, rows []TeacherEntity) error {
	if len(rows) == 0 {
		return nil
	}
	return upsertSingleKeyTx(tx, rows, "id")
}

// UpsertCalendarsTx 按 calendarId upsert。
func UpsertCalendarsTx(tx *gorm.DB, rows []CalendarEntity) error {
	if len(rows) == 0 {
		return nil
	}
	return upsertSingleKeyTx(tx, rows, "calendar_id")
}

// UpsertLanguagesTx 按 teaching_language upsert。
func UpsertLanguagesTx(tx *gorm.DB, rows []LanguageEntity) error {
	if len(rows) == 0 {
		return nil
	}
	return upsertSingleKeyTx(tx, rows, "teaching_language")
}

// UpsertCourseNaturesTx 按 course_label_id upsert 全局字典。
func UpsertCourseNaturesTx(tx *gorm.DB, rows []CourseNatureEntity) error {
	if len(rows) == 0 {
		return nil
	}
	return upsertSingleKeyTx(tx, rows, "course_label_id")
}

// UpsertCourseNatureByCalendarTx 按 (calendar_id, course_label_id) upsert 学期快照。
func UpsertCourseNatureByCalendarTx(tx *gorm.DB, rows []CourseNatureByCalendarEntity) error {
	if len(rows) == 0 {
		return nil
	}
	onConflict := clause.OnConflict{
		Columns:   []clause.Column{{Name: "calendar_id"}, {Name: "course_label_id"}},
		UpdateAll: true,
	}
	return chunkedUpsertTx(tx, rows, onConflict, upsertChunkSize)
}

// UpsertAssessmentsTx 按 assessment_mode upsert。
func UpsertAssessmentsTx(tx *gorm.DB, rows []AssessmentEntity) error {
	if len(rows) == 0 {
		return nil
	}
	return upsertSingleKeyTx(tx, rows, "assessment_mode")
}

// UpsertCampusesTx 按 campus upsert。
func UpsertCampusesTx(tx *gorm.DB, rows []CampusEntity) error {
	if len(rows) == 0 {
		return nil
	}
	return upsertSingleKeyTx(tx, rows, "campus")
}

// UpsertFacultiesTx 按 faculty upsert。
func UpsertFacultiesTx(tx *gorm.DB, rows []FacultyEntity) error {
	if len(rows) == 0 {
		return nil
	}
	return upsertSingleKeyTx(tx, rows, "faculty")
}

// UpsertMajorsTx 按 name upsert（专业字典）。
func UpsertMajorsTx(tx *gorm.DB, rows []MajorEntity) error {
	if len(rows) == 0 {
		return nil
	}
	return upsertSingleKeyTx(tx, rows, "name")
}

// GetMajorIdByNameTx 事务内按 name 查找专业 id。
func GetMajorIdByNameTx(tx *gorm.DB, name string) (uint64, error) {
	var entity MajorEntity
	if err := tx.Where(queryopt.Eq("name", name)).First(&entity).Error; err != nil {
		return 0, err
	}
	return entity.Id, nil
}

// UpsertMajorCoursesTx 按 (major_id, course_id) 忽略冲突插入（上游 INSERT OR IGNORE）。
func UpsertMajorCoursesTx(tx *gorm.DB, rows []MajorCourseEntity) error {
	if len(rows) == 0 {
		return nil
	}
	onConflict := clause.OnConflict{
		Columns:   []clause.Column{{Name: "major_id"}, {Name: "course_id"}},
		DoNothing: true,
	}
	if err := chunkedUpsertTx(tx, rows, onConflict, upsertChunkSize); err != nil {
		return fmt.Errorf("pk: upsert major courses: %w", err)
	}
	return nil
}

// ---- 时间片重建 ----

// TeacherTimeslotSourceRow 重建 teacher_timeslots 的源行（teacher JOIN coursedetail）。
type TeacherTimeslotSourceRow struct {
	CalendarId      uint64
	TeachingClassId uint64
	TeacherCode     string
	TeacherName     string
	ArrangeInfoText string
}

// ListTeacherTimeslotSource 列出指定学期重建时间片所需的全部源行。
func ListTeacherTimeslotSource(calendarIds []uint64) ([]TeacherTimeslotSourceRow, error) {
	var rows []TeacherTimeslotSourceRow
	b := db.Connect().Table(teacherTableName + " AS t").
		Select("cd.calendar_id AS calendar_id, t.teaching_class_id AS teaching_class_id, t.teacher_code AS teacher_code, t.teacher_name AS teacher_name, t.arrange_info_text AS arrange_info_text").
		Joins("JOIN " + courseDetailTableName + " AS cd ON cd.id = t.teaching_class_id")
	if len(calendarIds) > 0 {
		b = b.Where("cd.calendar_id IN ?", calendarIds)
	}
	if err := b.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("pk: list timeslot source: %w", err)
	}
	return rows, nil
}

// ReplaceTeacherTimeslotsTx 事务内重建指定学期的时间片：先清空再批量 upsert（分块避免参数超限）。
func ReplaceTeacherTimeslotsTx(tx *gorm.DB, calendarIds []uint64, rows []TeacherTimeslotEntity) error {
	if err := DeleteTeacherTimeslotsTx(tx, calendarIds); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	onConflict := clause.OnConflict{
		Columns:   []clause.Column{{Name: "calendar_id"}, {Name: "teaching_class_id"}, {Name: "occupy_day"}, {Name: "occupy_section"}, {Name: "teacher_code"}, {Name: "teacher_name"}},
		UpdateAll: true,
	}
	if err := chunkedUpsertTx(tx, rows, onConflict, upsertChunkSize); err != nil {
		return fmt.Errorf("pk: replace timeslots: %w", err)
	}
	return nil
}

// ---- 物化联动读取 ----

// ListCourseDetailsByCalendar 返回某学期全部教学班（按 id 升序，供物化/对账）。
func ListCourseDetailsByCalendar(calendarId uint64) ([]CourseDetailEntity, error) {
	var entities []CourseDetailEntity
	if err := courseDetailBuilder().Where(queryopt.Eq("calendar_id", calendarId)).Order("id ASC").Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("pk: list course details: %w", err)
	}
	return entities, nil
}

// ListTeachersByClassIds 返回一批教学班的教师（分块查询避免 IN 超限）。
func ListTeachersByClassIds(classIds []uint64) ([]TeacherEntity, error) {
	var all []TeacherEntity
	const chunkSize = 80
	for i := 0; i < len(classIds); i += chunkSize {
		end := i + chunkSize
		if end > len(classIds) {
			end = len(classIds)
		}
		var chunk []TeacherEntity
		if err := teacherBuilder().Where("teaching_class_id IN ?", classIds[i:end]).Find(&chunk).Error; err != nil {
			return nil, fmt.Errorf("pk: list teachers: %w", err)
		}
		all = append(all, chunk...)
	}
	return all, nil
}

// ListFacultiesTx 返回院系字典（faculty → faculty_i18n），供物化填充 department。
func ListFacultiesTx(tx *gorm.DB) ([]FacultyEntity, error) {
	var entities []FacultyEntity
	if err := tx.Table(facultyTableName).Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("pk: list faculties: %w", err)
	}
	return entities, nil
}
