package pk

import (
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TestPKSchemaConcurrentUpsertPostgreSQL 在真实 PostgreSQL 上验证 PK 域迁移与并发 upsert 无歧义
// （issue #185 验收 1：参照 course/stats_pg_test.go 的并发原子模式）。
//
//   - 验收 1a：13 张 pk 表 + setting 全部可通过 AutoMigrate 在 PG 上创建（三方言模型定义无方言专属语法）。
//   - 验收 1b：单主键表（teacher）并发 upsert 同一行无歧义，N 次并发后恰好 1 行且元数据列已填充。
//   - 验收 1c：复合主键表（teacher_timeslot，6 列复合键）并发 upsert 同一键无重复行。
//
// 依赖 YOURTJ_TEST_PG_URL（与 migration 包真实 PG 迁移测试同一门控），未设置时跳过。
func TestPKSchemaConcurrentUpsertPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL concurrent PK upsert test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	entities := []any{
		&CalendarEntity{}, &CampusEntity{}, &FacultyEntity{}, &LanguageEntity{},
		&AssessmentEntity{}, &CourseNatureEntity{}, &CourseNatureByCalendarEntity{},
		&MajorEntity{}, &MajorCourseEntity{}, &CourseDetailEntity{}, &TeacherEntity{},
		&TeacherTimeslotEntity{}, &FetchLogEntity{}, &SettingEntity{},
	}
	if err := db.AutoMigrate(entities...); err != nil {
		t.Fatalf("AutoMigrate pk tables on postgres failed: %v", err)
	}
	for _, m := range entities {
		if err := db.Unscoped().Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("clean pk table: %v", err)
		}
	}
	// 结束后清理本测试写入的行，保证同一测试库上可重复运行、不残留测试数据
	// （migration_pg_test 用 DROP SCHEMA 重置，本测试只清自己的键）。
	t.Cleanup(func() {
		_ = db.Unscoped().Where("id = 1").Delete(&TeacherEntity{}).Error
		_ = db.Unscoped().Where("calendar_id = 202601 AND teaching_class_id = 900001").
			Delete(&TeacherTimeslotEntity{}).Error
	})

	const calendarId uint64 = 202601
	const classId uint64 = 900001
	const workers = 8
	const perWorker = 25

	// 单主键并发 upsert：8×25 写同一 teacher 行，最终必须恰好 1 行、无歧义。
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			now := time.Now()
			for i := 0; i < perWorker; i++ {
				row := TeacherEntity{
					Id: 1, TeachingClassId: classId, TeacherCode: "T001",
					TeacherName: "并发教师", ArrangeInfoText: "x",
					SchemaVersion: PKDataSchemaVersion, SyncedAt: &now,
				}
				if err := UpsertTeachersTx(db, []TeacherEntity{row}); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
			}
		}()
	}
	wg.Wait()
	if firstErr != nil {
		t.Fatalf("concurrent teacher upsert failed: %v", firstErr)
	}
	var teacherCount int64
	if err := db.Model(&TeacherEntity{}).Where("id = 1").Count(&teacherCount).Error; err != nil {
		t.Fatalf("count teachers: %v", err)
	}
	if teacherCount != 1 {
		t.Fatalf("expected exactly 1 teacher row after %d concurrent upserts, got %d", workers*perWorker, teacherCount)
	}
	var teacher TeacherEntity
	if err := db.First(&teacher, "id = 1").Error; err != nil {
		t.Fatalf("load teacher after concurrent upsert: %v", err)
	}
	if teacher.SchemaVersion != PKDataSchemaVersion || teacher.SyncedAt == nil {
		t.Fatalf("metadata columns not populated after upsert: %+v", teacher)
	}

	// 复合主键并发 upsert：8×25 写同一 (calendar, class, day, section, code, name) 键，
	// ON CONFLICT 复合键必须无歧义——最终恰好 1 行，不产生重复行。
	onConflict := clause.OnConflict{
		Columns: []clause.Column{
			{Name: "calendar_id"}, {Name: "teaching_class_id"}, {Name: "occupy_day"},
			{Name: "occupy_section"}, {Name: "teacher_code"}, {Name: "teacher_name"},
		},
		UpdateAll: true,
	}
	wg = sync.WaitGroup{}
	mu = sync.Mutex{}
	firstErr = nil
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			now := time.Now()
			for i := 0; i < perWorker; i++ {
				row := TeacherTimeslotEntity{
					CalendarId: calendarId, TeachingClassId: classId,
					OccupyDay: 1, OccupySection: 2, TeacherCode: "T001", TeacherName: "并发教师",
					SchemaVersion: PKDataSchemaVersion, SyncedAt: &now,
				}
				if err := db.Clauses(onConflict).Create(&row).Error; err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
			}
		}()
	}
	wg.Wait()
	if firstErr != nil {
		t.Fatalf("concurrent timeslot upsert failed: %v", firstErr)
	}
	var tsCount int64
	if err := db.Model(&TeacherTimeslotEntity{}).
		Where("calendar_id = ? AND teaching_class_id = ?", calendarId, classId).
		Count(&tsCount).Error; err != nil {
		t.Fatalf("count timeslots: %v", err)
	}
	if tsCount != 1 {
		t.Fatalf("expected exactly 1 timeslot row after %d concurrent upserts, got %d", workers*perWorker, tsCount)
	}
}
