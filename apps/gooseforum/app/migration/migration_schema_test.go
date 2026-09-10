package migration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
