package migration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUsersEmailSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migration-email-schema?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatalf("migrate users schema: %v", err)
	}
	assertUniqueUserEmailSchema(t, db)
}

func TestUpgradePkAudienceSchemaPreservesLegacyDictionary(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migration-pk-audience?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	assertPkAudienceLegacyDictionaryUpgrade(t, db)
}

func assertPkAudienceLegacyDictionaryUpgrade(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(legacyPkDDL(db, `CREATE TABLE pk_campus (
		campus TEXT PRIMARY KEY NOT NULL,
		campus_i18n TEXT NOT NULL DEFAULT '',
		calendar_id INTEGER NOT NULL DEFAULT 0,
		schema_version TEXT NOT NULL DEFAULT '',
		synced_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)).Error; err != nil {
		t.Fatalf("create legacy pk_campus: %v", err)
	}
	if err := db.Exec(`INSERT INTO pk_campus (campus, campus_i18n, calendar_id) VALUES (?, ?, ?)`, "四平路校区", "Siping", 121).Error; err != nil {
		t.Fatalf("insert legacy pk_campus: %v", err)
	}

	if err := upgradePkAudienceSchema(db); err != nil {
		t.Fatalf("upgradePkAudienceSchema: %v", err)
	}
	if !db.Migrator().HasColumn(&pk.CampusEntity{}, "audience") {
		t.Fatal("pk_campus.audience missing after legacy upgrade")
	}
	var undergraduate pk.CampusEntity
	if err := db.Where("campus = ?", "四平路校区").First(&undergraduate).Error; err != nil {
		t.Fatalf("read migrated campus: %v", err)
	}
	if undergraduate.Audience != string(pk.AudienceUndergraduate) {
		t.Fatalf("migrated audience = %q, want %q", undergraduate.Audience, pk.AudienceUndergraduate)
	}
	if err := db.AutoMigrate(&pk.CampusEntity{}); err != nil {
		t.Fatalf("automigrate upgraded pk_campus: %v", err)
	}
	if err := upgradePkAudienceSchema(db); err != nil {
		t.Fatalf("rerun upgradePkAudienceSchema: %v", err)
	}
	if err := db.Create(&pk.CampusEntity{
		Audience:   string(pk.AudienceGraduate),
		Campus:     "四平路校区",
		CampusI18n: "Siping graduate",
		CalendarId: 221,
	}).Error; err != nil {
		t.Fatalf("insert same dictionary key for graduate audience: %v", err)
	}
	var count int64
	if err := db.Model(&pk.CampusEntity{}).Where("campus = ?", "四平路校区").Count(&count).Error; err != nil {
		t.Fatalf("count audience-scoped campuses: %v", err)
	}
	if count != 2 {
		t.Fatalf("audience-scoped campus count = %d, want 2", count)
	}
}

func TestUpgradePkAudienceSchemaAddsAudienceConflictKeys(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migration-pk-audience-conflict-keys?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	assertPkAudienceLegacyConflictKeysUpgrade(t, db)
}

func assertPkAudienceLegacyConflictKeysUpgrade(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(legacyPkDDL(db, `CREATE TABLE pk_teacher_timeslot (
		calendar_id INTEGER NOT NULL,
		teaching_class_id INTEGER NOT NULL,
		occupy_day INTEGER NOT NULL,
		occupy_section INTEGER NOT NULL,
		teacher_code TEXT NOT NULL DEFAULT '',
		teacher_name TEXT NOT NULL DEFAULT '',
		schema_version TEXT NOT NULL DEFAULT '',
		synced_at DATETIME,
		PRIMARY KEY (calendar_id, teaching_class_id, occupy_day, occupy_section, teacher_code, teacher_name)
	)`)).Error; err != nil {
		t.Fatalf("create legacy pk_teacher_timeslot: %v", err)
	}
	if err := db.Exec(legacyPkDDL(db, `CREATE TABLE pk_major_course (
		major_id INTEGER NOT NULL,
		course_id INTEGER NOT NULL,
		schema_version TEXT NOT NULL DEFAULT '',
		synced_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		PRIMARY KEY (major_id, course_id)
	)`)).Error; err != nil {
		t.Fatalf("create legacy pk_major_course: %v", err)
	}
	if err := db.Exec(`CREATE INDEX idx_pk_timeslot_class ON pk_teacher_timeslot (teaching_class_id);
		CREATE INDEX idx_pk_timeslot_slot ON pk_teacher_timeslot (calendar_id, occupy_day, occupy_section);
		CREATE INDEX idx_pk_major_course_course ON pk_major_course (course_id);`).Error; err != nil {
		t.Fatalf("create legacy PK indexes: %v", err)
	}
	if err := db.Exec(`INSERT INTO pk_teacher_timeslot (calendar_id, teaching_class_id, occupy_day, occupy_section, teacher_code, teacher_name) VALUES (121, 1, 1, 2, 'T-UG', '本科教师')`).Error; err != nil {
		t.Fatalf("insert legacy pk_teacher_timeslot: %v", err)
	}
	if err := db.Exec(`INSERT INTO pk_major_course (major_id, course_id) VALUES (11, 1)`).Error; err != nil {
		t.Fatalf("insert legacy pk_major_course: %v", err)
	}
	if err := upgradePkAudienceSchema(db); err != nil {
		t.Fatalf("upgradePkAudienceSchema: %v", err)
	}
	if err := db.AutoMigrate(&pk.TeacherTimeslotEntity{}, &pk.MajorCourseEntity{}); err != nil {
		t.Fatalf("automigrate audience conflict keys: %v", err)
	}
	graduateTimeslot := pk.TeacherTimeslotEntity{
		Audience:        string(pk.AudienceGraduate),
		CalendarId:      pk.ScopeID(pk.AudienceGraduate, 121),
		TeachingClassId: pk.ScopeID(pk.AudienceGraduate, 1),
		OccupyDay:       1,
		OccupySection:   2,
		TeacherCode:     "T-GRAD",
		TeacherName:     "研究生教师",
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return pk.ReplaceTeacherTimeslotsForAudienceTx(tx, pk.AudienceGraduate, []uint64{121}, []pk.TeacherTimeslotEntity{graduateTimeslot})
	}); err != nil {
		t.Fatalf("upsert graduate timeslot: %v", err)
	}
	graduateMajorCourse := pk.MajorCourseEntity{
		Audience: string(pk.AudienceGraduate), MajorId: 22, CourseId: pk.ScopeID(pk.AudienceGraduate, 1),
	}
	if err := pk.UpsertMajorCoursesTx(db, []pk.MajorCourseEntity{graduateMajorCourse}); err != nil {
		t.Fatalf("upsert graduate major course: %v", err)
	}
	var undergraduateTimeslot pk.TeacherTimeslotEntity
	if err := db.Where("audience = ? AND calendar_id = ?", pk.AudienceUndergraduate, 121).First(&undergraduateTimeslot).Error; err != nil {
		t.Fatalf("read migrated undergraduate timeslot: %v", err)
	}
	var undergraduateMajorCourse pk.MajorCourseEntity
	if err := db.Where("audience = ? AND major_id = ? AND course_id = ?", pk.AudienceUndergraduate, 11, 1).First(&undergraduateMajorCourse).Error; err != nil {
		t.Fatalf("read migrated undergraduate major course: %v", err)
	}
}

func TestValidateUniqueUserEmails(t *testing.T) {
	tests := []struct {
		name   string
		emails []string
		want   string
	}{
		{name: "missing table"},
		{name: "unique non-empty and multiple empty", emails: []string{"alice@example.com", "", ""}},
		{name: "duplicate non-empty", emails: []string{"alice@example.com", "alice@example.com"}, want: "alice@example.com"},
	}
	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:migration-email-%d?mode=memory&cache=shared", index)), &gorm.Config{})
			if err != nil {
				t.Fatalf("open sqlite: %v", err)
			}
			if tt.emails != nil {
				if err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT NOT NULL DEFAULT '')`).Error; err != nil {
					t.Fatalf("create legacy users table: %v", err)
				}
				for _, email := range tt.emails {
					if err := db.Exec("INSERT INTO users (email) VALUES (?)", email).Error; err != nil {
						t.Fatalf("insert email %q: %v", email, err)
					}
				}
			}

			err = validateUniqueUserEmails(db)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("validateUniqueUserEmails() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateUniqueUserEmails() error = %v, want containing %q", err, tt.want)
			}
			var count int64
			if err := db.Table("users").Where("email = ?", tt.want).Count(&count).Error; err != nil {
				t.Fatalf("count duplicate email rows: %v", err)
			}
			if count != 2 {
				t.Fatalf("duplicate email row count = %d, want 2; validator must not rewrite data", count)
			}
		})
	}
}

// assertUniqueUserEmailSchema verifies the cross-dialect contract: blank
// emails remain compatible with bot/OAuth accounts, while non-empty emails
// are rejected by the database even if application-level checks race.
func assertUniqueUserEmailSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if !db.Migrator().HasIndex(&users.EntityComplete{}, "uniq_users_email_nonempty") {
		t.Fatal("users.uniq_users_email_nonempty partial unique index missing after migration")
	}

	for _, user := range []*users.EntityComplete{
		{Username: "email-schema-empty-a"},
		{Username: "email-schema-empty-b"},
	} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("insert empty-email user %q: %v", user.Username, err)
		}
	}
	if err := db.Create(&users.EntityComplete{Username: "email-schema-nonempty", Email: "email-schema@example.com"}).Error; err != nil {
		t.Fatalf("insert non-empty-email user: %v", err)
	}
	if err := db.Create(&users.EntityComplete{Username: "email-schema-duplicate", Email: "email-schema@example.com"}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate non-empty email error = %v, want gorm.ErrDuplicatedKey", err)
	}
}

func TestValidateUniqueUsernames(t *testing.T) {
	tests := []struct {
		name      string
		usernames []string
		wantError string
	}{
		{name: "missing table"},
		{name: "valid", usernames: []string{"alice", "agent-one"}},
		{name: "blank", usernames: []string{"alice", ""}, wantError: "1 blank username row"},
		{name: "duplicate", usernames: []string{"alice", "alice"}, wantError: "alice"},
	}
	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:migration-username-%d?mode=memory&cache=shared", index)), &gorm.Config{})
			if err != nil {
				t.Fatalf("open sqlite: %v", err)
			}
			if tt.usernames != nil {
				if err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL DEFAULT '')`).Error; err != nil {
					t.Fatalf("create legacy users table: %v", err)
				}
				for _, username := range tt.usernames {
					if err := db.Exec("INSERT INTO users (username) VALUES (?)", username).Error; err != nil {
						t.Fatalf("insert username %q: %v", username, err)
					}
				}
			}

			err = validateUniqueUsernames(db)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("validateUniqueUsernames() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("validateUniqueUsernames() error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestMigrationUsesCleanTopicEdgeModels(t *testing.T) {
	source, err := os.ReadFile("migration.go")
	if err != nil {
		t.Fatalf("read migration.go: %v", err)
	}

	text := string(source)
	for _, oldModel := range []string{
		"models/forum/articleCategory",
		"models/forum/articleCategoryRs",
		"models/forum/articleUserAction",
		"models/forum/articlesUserStat",
		"models/forum/articles",
		"models/forum/reply",
	} {
		if strings.Contains(text, oldModel) {
			t.Fatalf("migration still imports old edge model %q", oldModel)
		}
	}

	for _, cleanModel := range []string{
		"models/forum/category",
		"models/forum/migrationMapping",
		"models/forum/topicCategoryIndex",
		"models/forum/topicUserAction",
		"models/forum/topicUserStat",
	} {
		if !strings.Contains(text, cleanModel) {
			t.Fatalf("migration does not import clean edge model %q", cleanModel)
		}
	}
}

func TestActiveRuntimeDoesNotImportOldArticleReplyModels(t *testing.T) {
	roots := []string{
		"../http",
		"../service",
		"../models/hotdataserve",
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(source)
			for _, oldModel := range []string{
				"models/forum/articleCategory",
				"models/forum/articleCategoryRs",
				"models/forum/articleUserAction",
				"models/forum/articlesUserStat",
				"models/forum/articles",
				"models/forum/reply",
			} {
				if strings.Contains(text, oldModel) {
					t.Fatalf("%s imports old model %q", path, oldModel)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
}

func TestUpgradePkAudienceSchemaRollsBackBeforeRetry(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "retry.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("CREATE TABLE pk_major (id INTEGER PRIMARY KEY, code TEXT, grade INTEGER, name TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("CREATE UNIQUE INDEX uniq_pk_major_name ON pk_major (name)").Error; err != nil {
		t.Fatal(err)
	}
	const cb = "test:pk-index-failure"
	if err := conn.Callback().Raw().Before("gorm:raw").Register(cb, func(tx *gorm.DB) {
		if strings.HasPrefix(tx.Statement.SQL.String(), "DROP INDEX") {
			_ = tx.AddError(errors.New("injected index failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	err = upgradePkAudienceSchema(conn)
	if removeErr := conn.Callback().Raw().Remove(cb); removeErr != nil {
		t.Fatal(removeErr)
	}
	if err == nil {
		t.Fatal("expected upgrade failure")
	}
	if conn.Migrator().HasColumn(&pk.MajorEntity{}, "audience") {
		t.Error("failed upgrade committed the column used to skip legacy index cleanup")
	}
	if err := upgradePkAudienceSchema(conn); err != nil {
		t.Fatal(err)
	}
	if conn.Migrator().HasIndex(&pk.MajorEntity{}, "uniq_pk_major_name") {
		t.Fatal("retry left the legacy unique index in place")
	}
}

// PostgreSQL uses a timezone-aware timestamp for the historical datetime columns.
func legacyPkDDL(db *gorm.DB, ddl string) string {
	if db.Name() == "postgres" {
		return strings.ReplaceAll(ddl, "DATETIME", "TIMESTAMPTZ")
	}
	return ddl
}
