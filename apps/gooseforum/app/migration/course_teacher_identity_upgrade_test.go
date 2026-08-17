package migration

import (
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// legacyCourseEntity 旧版 course 模型（issue #326 前）：无 teacher_id 列，
// primary_code 单列唯一索引 uniq_course_primary_code。用它 AutoMigrate 建出
// 与真实存量库一致的旧表，验证 upgradeCourseTeacherIdentity 的升级路径。
type legacyCourseEntity struct {
	Id             uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	PrimaryCode    string         `gorm:"column:primary_code;type:varchar(64);not null;default:'';uniqueIndex:uniq_course_primary_code;" json:"primaryCode"`
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

func (legacyCourseEntity) TableName() string { return "course" }

// exerciseCourseTeacherIdentityUpgrade 在给定连接上执行完整升级路径并断言结果：
// 旧表（无 teacher_id + primary_code 单列唯一）→ upgradeCourseTeacherIdentity
// （ADD COLUMN 保留存量数据 + 删旧索引）→ AutoMigrate 新模型（建 (primary_code,
// teacher_id) 复合唯一）→ 同 code 不同 teacher 可插、重复 (code, teacher) 被拦截、
// 存量行数据完整保留且 teacher_id 为 0（无教师哨兵值）。
func exerciseCourseTeacherIdentityUpgrade(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&legacyCourseEntity{}); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}
	legacy := legacyCourseEntity{PrimaryCode: "12200402", Name: "高等数学(B)上", Status: 0}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("seed legacy course: %v", err)
	}

	if err := upgradeCourseTeacherIdentity(db); err != nil {
		t.Fatalf("upgrade teacher identity: %v", err)
	}
	// migrateSchema 在 upgrade* 之后统一 AutoMigrate 新模型（重建复合唯一索引），
	// 测试必须复刻同一路径，否则 uniq_course_code_teacher 不会被创建。
	if err := db.AutoMigrate(&course.Entity{}); err != nil {
		t.Fatalf("migrate new course model: %v", err)
	}
	// 同 code 不同 teacher（教师 A=1, B=2）必须能插入——复合身份模型。
	teacherA := uint64(1)
	teacherB := uint64(2)
	other := course.Entity{PrimaryCode: "12200402", TeacherId: teacherA, Name: "高等数学(B)上", Status: 0}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("insert same code different teacher: %v", err)
	}
	// 重复 (code, teacher) 仍被复合唯一索引幂等拦截。
	dup := course.Entity{PrimaryCode: "12200402", TeacherId: teacherA, Name: "高等数学(B)上", Status: 0}
	if err := db.Create(&dup).Error; err == nil {
		t.Fatal("expected duplicate (primary_code, teacher_id) insert to fail")
	}
	// 无教师行（teacher_id=0）与有教师行共存。
	noTeacher := course.Entity{PrimaryCode: "320001", Name: "体育(1)", Status: 0}
	if err := db.Create(&noTeacher).Error; err != nil {
		t.Fatalf("insert no-teacher course: %v", err)
	}
	noTeacher2 := course.Entity{PrimaryCode: "320001", TeacherId: teacherB, Name: "体育(1)", Status: 0}
	if err := db.Create(&noTeacher2).Error; err != nil {
		t.Fatalf("insert same code with teacher alongside no-teacher row: %v", err)
	}
	// 存量行：teacher_id 为 0（默认值）、primary_code/name/status 完整保留
	// （升级必须不丢数据——GORM SQLite 迁移的整表重建路径有数据丢失风险）。
	var got course.Entity
	if err := db.First(&got, "id = ?", legacy.Id).Error; err != nil {
		t.Fatalf("load legacy course: %v", err)
	}
	if got.PrimaryCode != "12200402" || got.Name != "高等数学(B)上" || got.Status != 0 {
		t.Fatalf("legacy course data lost after upgrade: %+v", got)
	}
	if got.TeacherId != 0 {
		t.Fatalf("legacy course teacher_id = %d, want 0", got.TeacherId)
	}
}
func TestCourseTeacherIdentityUpgradeFromLegacySchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	exerciseCourseTeacherIdentityUpgrade(t, db)
}

// TestCourseTeacherIdentityUpgradeFromLegacySchemaOnPostgreSQL 同上，PostgreSQL 版。
// 依赖 YOURTJ_TEST_PG_URL，未设置时跳过。
func TestCourseTeacherIdentityUpgradeFromLegacySchemaOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL course teacher identity upgrade test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	// 清理可能残留的表（与 migration_pg_test 共享同一测试库）。
	if err := db.Migrator().DropTable(&course.Entity{}); err != nil {
		t.Fatalf("drop leftover table: %v", err)
	}
	exerciseCourseTeacherIdentityUpgrade(t, db)
}
