package migration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// legacyAggregationCourseEntity 旧版 course 模型（课评聚合前）：无 review_scope/team_key。
type legacyAggregationCourseEntity struct {
	Id             uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	PrimaryCode    string         `gorm:"column:primary_code;type:varchar(64);not null;default:'';uniqueIndex:uniq_course_code_teacher,priority:1;" json:"primaryCode"`
	TeacherId      uint64         `gorm:"column:teacher_id;not null;default:0;uniqueIndex:uniq_course_code_teacher,priority:2;" json:"teacherId"`
	Name           string         `gorm:"column:name;type:varchar(255);not null;default:'';" json:"name"`
	Department     string         `gorm:"column:department;type:varchar(255);not null;default:'';" json:"department"`
	CreditX10      int            `gorm:"column:credit_x10;not null;default:0;" json:"creditX10"`
	NormalizedName string         `gorm:"column:normalized_name;type:varchar(255);not null;default:'';" json:"normalizedName"`
	NamePinyin     string         `gorm:"column:name_pinyin;type:varchar(255);not null;default:'';" json:"namePinyin"`
	NameInitials   string         `gorm:"column:name_initials;type:varchar(64);not null;default:'';" json:"nameInitials"`
	Status         int8           `gorm:"column:status;not null;default:0;" json:"status"`
	SearchVersion  uint64         `gorm:"column:search_version;not null;default:0;" json:"searchVersion"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-"`
}

func (legacyAggregationCourseEntity) TableName() string { return "course" }

// legacyAggregationOfferingEntity 旧版 offering：无 teaching_class_id。
type legacyAggregationOfferingEntity struct {
	Id        uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	CourseId  uint64         `gorm:"column:course_id;not null;default:0;index:idx_course_offering_course;" json:"courseId"`
	TermId    uint64         `gorm:"column:term_id;not null;default:0;index:idx_course_offering_term;" json:"termId"`
	Campus    string         `gorm:"column:campus;type:varchar(64);not null;default:'';" json:"campus"`
	Faculty   string         `gorm:"column:faculty;type:varchar(255);not null;default:'';" json:"faculty"`
	ClassCode string         `gorm:"column:class_code;type:varchar(64);not null;default:'';" json:"classCode"`
	ClassName string         `gorm:"column:class_name;type:varchar(255);not null;default:'';" json:"className"`
	Status    int8           `gorm:"column:status;not null;default:0;" json:"status"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

func (legacyAggregationOfferingEntity) TableName() string { return "course_offering" }

// legacyAggregationInstructorEntity 旧版教师：无 teacher_code。
type legacyAggregationInstructorEntity struct {
	Id             uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Name           string         `gorm:"column:name;type:varchar(64);not null;default:'';" json:"name"`
	NormalizedName string         `gorm:"column:normalized_name;type:varchar(64);not null;default:'';" json:"normalizedName"`
	NamePinyin     string         `gorm:"column:name_pinyin;type:varchar(255);not null;default:'';" json:"namePinyin"`
	NameInitials   string         `gorm:"column:name_initials;type:varchar(64);not null;default:'';" json:"nameInitials"`
	Department     string         `gorm:"column:department;type:varchar(255);not null;default:'';" json:"department"`
	Title          string         `gorm:"column:title;type:varchar(64);not null;default:'';" json:"title"`
	Status         int8           `gorm:"column:status;not null;default:0;" json:"status"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-"`
}

func (legacyAggregationInstructorEntity) TableName() string { return "course_instructor" }

// exerciseCourseAggregationUpgrade 在给定连接上执行完整升级路径并断言结果：
// 旧表（无 review_scope/team_key/teaching_class_id/teacher_code、无 course_relations）
// → upgradeCourseAggregation（ADD COLUMN 保留存量数据 + source_ref 回填）
// → AutoMigrate 新模型（建 course_relations + partial unique index）
// → 行为断言：同 teaching_class_id 重复插入被拦、0 可多行、teacher_code 回填、
// review_scope 默认 teacher、存量行数据完整保留。
func exerciseCourseAggregationUpgrade(t *testing.T, db *gorm.DB) {
	t.Helper()
	// 建旧形态表 + source_ref（回填数据源）。
	if err := db.AutoMigrate(
		&legacyAggregationCourseEntity{},
		&legacyAggregationOfferingEntity{},
		&legacyAggregationInstructorEntity{},
		&course.SourceRefEntity{},
	); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}
	courseRow := legacyAggregationCourseEntity{PrimaryCode: "12200402", Name: "高等数学(B)上", Status: 0}
	if err := db.Create(&courseRow).Error; err != nil {
		t.Fatalf("seed legacy course: %v", err)
	}
	offeringRow := legacyAggregationOfferingEntity{CourseId: courseRow.Id, TermId: 1, Status: 0}
	if err := db.Create(&offeringRow).Error; err != nil {
		t.Fatalf("seed legacy offering: %v", err)
	}
	instructorRow := legacyAggregationInstructorEntity{Name: "张三", NormalizedName: "张三", Department: "数学科学学院"}
	if err := db.Create(&instructorRow).Error; err != nil {
		t.Fatalf("seed legacy instructor: %v", err)
	}
	// source_ref：offering external_id "{class_id}-{course_ext}"，instructor external_id = teacherCode。
	if err := db.Create(&course.SourceRefEntity{
		Source: "jcourse-snapshot-20260814", EntityType: course.EntityTypeOffering,
		ExternalId: "900001-42", LocalId: offeringRow.Id, Checksum: "offering-1",
	}).Error; err != nil {
		t.Fatalf("seed offering source_ref: %v", err)
	}
	if err := db.Create(&course.SourceRefEntity{
		Source: "jcourse-snapshot-20260814", EntityType: course.EntityTypeInstructor,
		ExternalId: "T00123", LocalId: instructorRow.Id, Checksum: "instructor-1",
	}).Error; err != nil {
		t.Fatalf("seed instructor source_ref: %v", err)
	}
	// other-* 虚拟班 external_id：非数字首段，保持 0。
	otherOffering := legacyAggregationOfferingEntity{CourseId: courseRow.Id, TermId: 1, Status: 0}
	if err := db.Create(&otherOffering).Error; err != nil {
		t.Fatalf("seed other offering: %v", err)
	}
	if err := db.Create(&course.SourceRefEntity{
		Source: "jcourse-snapshot-20260814", EntityType: course.EntityTypeOffering,
		ExternalId: "other-42", LocalId: otherOffering.Id, Checksum: "offering-2",
	}).Error; err != nil {
		t.Fatalf("seed other offering source_ref: %v", err)
	}

	if err := upgradeCourseAggregation(db); err != nil {
		t.Fatalf("upgrade aggregation: %v", err)
	}
	// migrateSchema 在 upgrade* 之后统一 AutoMigrate 新模型，测试必须复刻同一路径。
	if err := db.AutoMigrate(
		&course.Entity{},
		&course.OfferingEntity{},
		&course.InstructorEntity{},
		&course.RelationEntity{},
	); err != nil {
		t.Fatalf("migrate new course models: %v", err)
	}

	// 列存在 + 默认值。
	if !db.Migrator().HasColumn(&course.Entity{}, "review_scope") {
		t.Fatal("course.review_scope column missing after upgrade")
	}
	if !db.Migrator().HasColumn(&course.Entity{}, "team_key") {
		t.Fatal("course.team_key column missing after upgrade")
	}
	if !db.Migrator().HasColumn(&course.OfferingEntity{}, "teaching_class_id") {
		t.Fatal("course_offering.teaching_class_id column missing after upgrade")
	}
	if !db.Migrator().HasColumn(&course.InstructorEntity{}, "teacher_code") {
		t.Fatal("course_instructor.teacher_code column missing after upgrade")
	}
	if !db.Migrator().HasTable(&course.RelationEntity{}) {
		t.Fatal("course_relations table missing after upgrade")
	}

	// 存量行数据保留。
	var gotCourse course.Entity
	if err := db.First(&gotCourse, "id = ?", courseRow.Id).Error; err != nil {
		t.Fatalf("load legacy course: %v", err)
	}
	if gotCourse.PrimaryCode != "12200402" || gotCourse.Name != "高等数学(B)上" {
		t.Fatalf("legacy course data lost after upgrade: %+v", gotCourse)
	}
	if gotCourse.ReviewScope != "teacher" || gotCourse.TeamKey != "" {
		t.Fatalf("course review_scope = %q team_key = %q, want teacher/''", gotCourse.ReviewScope, gotCourse.TeamKey)
	}

	// 回填：offering.teaching_class_id = 900001；other-* 保持 0；instructor.teacher_code = T00123。
	var gotOffering course.OfferingEntity
	if err := db.First(&gotOffering, "id = ?", offeringRow.Id).Error; err != nil {
		t.Fatalf("load offering: %v", err)
	}
	if gotOffering.TeachingClassId != 900001 {
		t.Fatalf("offering teaching_class_id = %d, want 900001", gotOffering.TeachingClassId)
	}
	var gotOther course.OfferingEntity
	if err := db.First(&gotOther, "id = ?", otherOffering.Id).Error; err != nil {
		t.Fatalf("load other offering: %v", err)
	}
	if gotOther.TeachingClassId != 0 {
		t.Fatalf("other offering teaching_class_id = %d, want 0", gotOther.TeachingClassId)
	}
	var gotInstructor course.InstructorEntity
	if err := db.First(&gotInstructor, "id = ?", instructorRow.Id).Error; err != nil {
		t.Fatalf("load instructor: %v", err)
	}
	if gotInstructor.TeacherCode != "T00123" {
		t.Fatalf("instructor teacher_code = %q, want T00123", gotInstructor.TeacherCode)
	}

	// partial unique index：同 teaching_class_id 重复插入被拦；0 可多行。
	if err := db.Create(&course.OfferingEntity{CourseId: 1, TermId: 1, TeachingClassId: 900001, Status: 0}).Error; err == nil {
		t.Fatal("expected duplicate teaching_class_id insert to fail")
	}
	zeroA := course.OfferingEntity{CourseId: 1, TermId: 1, Status: 0}
	if err := db.Create(&zeroA).Error; err != nil {
		t.Fatalf("insert first zero teaching_class_id offering: %v", err)
	}
	if err := db.Create(&course.OfferingEntity{CourseId: 1, TermId: 1, Status: 0}).Error; err != nil {
		t.Fatalf("insert second zero teaching_class_id offering: %v", err)
	}

	// course_relations 可写、唯一约束生效。
	if err := db.Create(&course.RelationEntity{
		FromCourseId: 1, ToCourseId: 2, RelationType: string(course.RelationEquivalent),
		Source: course.RelationSourceRule, Status: string(course.RelationStatusPending),
	}).Error; err != nil {
		t.Fatalf("insert relation: %v", err)
	}
	if err := db.Create(&course.RelationEntity{
		FromCourseId: 1, ToCourseId: 2, RelationType: string(course.RelationEquivalent),
		Source: course.RelationSourceRule, Status: string(course.RelationStatusPending),
	}).Error; err == nil {
		t.Fatal("expected duplicate relation (from,to,type) insert to fail")
	}
}

// TestCourseAggregationUpgradeFromLegacySchema SQLite 版升级测试。
func TestCourseAggregationUpgradeFromLegacySchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	exerciseCourseAggregationUpgrade(t, db)
}

// TestCourseAggregationUpgradeFromLegacySchemaOnPostgreSQL PostgreSQL 版升级测试。
// 依赖 YOURTJ_TEST_PG_URL，未设置时跳过。
func TestCourseAggregationUpgradeFromLegacySchemaOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL course aggregation upgrade test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	// 清理可能残留的表（与 migration_pg_test 共享同一测试库）。
	for _, m := range []any{&course.RelationEntity{}, &course.OfferingEntity{}, &course.InstructorEntity{}, &course.Entity{}, &course.SourceRefEntity{}} {
		if err := db.Migrator().DropTable(m); err != nil {
			t.Fatalf("drop leftover table: %v", err)
		}
	}
	exerciseCourseAggregationUpgrade(t, db)
}

// TestCourseAggregationUpgradeLargeBatch 回归: 真实库 course_source_ref 超过
// FindInBatches 批大小(500)时, 无主键裸结构体分页报 "primary key required"
// (dev 部署事故根因: offering 21941 / instructor 4318 行, 均 > 500)。
// 修复后升级必须完整成功且回填正确(offering.teaching_class_id / instructor.teacher_code)。
func TestCourseAggregationUpgradeLargeBatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	const n = 1100 // > 500, 强制走 keyset 分页分支
	if err := db.AutoMigrate(
		&legacyAggregationCourseEntity{},
		&legacyAggregationOfferingEntity{},
		&legacyAggregationInstructorEntity{},
		&course.SourceRefEntity{},
	); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}
	courseRow := legacyAggregationCourseEntity{PrimaryCode: "12200402", Name: "高等数学(B)上", Status: 0}
	if err := db.Create(&courseRow).Error; err != nil {
		t.Fatalf("seed legacy course: %v", err)
	}
	offerings := make([]legacyAggregationOfferingEntity, n)
	for i := range offerings {
		offerings[i] = legacyAggregationOfferingEntity{CourseId: courseRow.Id, TermId: 1, Status: 0}
	}
	if err := db.Create(&offerings).Error; err != nil {
		t.Fatalf("seed legacy offerings: %v", err)
	}
	offeringRefs := make([]course.SourceRefEntity, n)
	for i := range offeringRefs {
		offeringRefs[i] = course.SourceRefEntity{
			Source: "jcourse-snapshot-20260814", EntityType: course.EntityTypeOffering,
			ExternalId: fmt.Sprintf("%d-42", 100000+i), LocalId: offerings[i].Id,
			Checksum: fmt.Sprintf("offering-%d", i),
		}
	}
	if err := db.Create(&offeringRefs).Error; err != nil {
		t.Fatalf("seed offering source_refs: %v", err)
	}
	instructors := make([]legacyAggregationInstructorEntity, n)
	for i := range instructors {
		instructors[i] = legacyAggregationInstructorEntity{
			Name: fmt.Sprintf("教师%04d", i), NormalizedName: fmt.Sprintf("教师%04d", i),
			Department: "数学科学学院",
		}
	}
	if err := db.Create(&instructors).Error; err != nil {
		t.Fatalf("seed legacy instructors: %v", err)
	}
	instructorRefs := make([]course.SourceRefEntity, n)
	for i := range instructorRefs {
		instructorRefs[i] = course.SourceRefEntity{
			Source: "jcourse-snapshot-20260814", EntityType: course.EntityTypeInstructor,
			ExternalId: fmt.Sprintf("T%05d", 10000+i), LocalId: instructors[i].Id,
			Checksum: fmt.Sprintf("instructor-%d", i),
		}
	}
	if err := db.Create(&instructorRefs).Error; err != nil {
		t.Fatalf("seed instructor source_refs: %v", err)
	}

	if err := upgradeCourseAggregation(db); err != nil {
		t.Fatalf("upgrade aggregation on >500-row backfill: %v", err)
	}
	// 与 migrateSchema 一致: upgrade* 后统一 AutoMigrate 新模型。
	if err := db.AutoMigrate(
		&course.Entity{},
		&course.OfferingEntity{},
		&course.InstructorEntity{},
		&course.RelationEntity{},
	); err != nil {
		t.Fatalf("migrate new course models: %v", err)
	}

	// 抽样断言: 跨批次边界(499/500/501)与尾部行回填正确。
	for _, i := range []int{0, 499, 500, 777, 1099} {
		var got course.OfferingEntity
		if err := db.First(&got, "id = ?", offerings[i].Id).Error; err != nil {
			t.Fatalf("load offering %d: %v", i, err)
		}
		if want := uint64(100000 + i); got.TeachingClassId != want {
			t.Fatalf("offering %d teaching_class_id = %d, want %d", i, got.TeachingClassId, want)
		}
		var ins course.InstructorEntity
		if err := db.First(&ins, "id = ?", instructors[i].Id).Error; err != nil {
			t.Fatalf("load instructor %d: %v", i, err)
		}
		if want := fmt.Sprintf("T%05d", 10000+i); ins.TeacherCode != want {
			t.Fatalf("instructor %d teacher_code = %q, want %q", i, ins.TeacherCode, want)
		}
	}
}
