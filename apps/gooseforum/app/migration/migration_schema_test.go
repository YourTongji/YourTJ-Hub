package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

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
