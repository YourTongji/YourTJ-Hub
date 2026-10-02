package inboxmail

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var testDBCounter atomic.Uint64

// openTestDB 为每个测试打开独立的 SQLite 内存库，并迁移全部 inboxmail 模型。
// 使用命名 shared-cache DSN：GORM 连接池可能建立多条连接，匿名 :memory: 会让
// 不同连接看到不同的库。
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:inboxmail-test-%d?mode=memory&cache=shared", testDBCounter.Add(1))
	conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(AllModels()...); err != nil {
		t.Fatalf("migrate inboxmail models: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := conn.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return conn
}

func indexColumns(t *testing.T, tx *gorm.DB, model any, name string) []string {
	t.Helper()
	indexes, err := tx.Migrator().GetIndexes(model)
	if err != nil {
		t.Fatalf("list %T indexes: %v", model, err)
	}
	for _, index := range indexes {
		if index.Name() == name {
			return index.Columns()
		}
	}
	return nil
}

func assertIndexColumns(t *testing.T, tx *gorm.DB, model any, name string, want []string) {
	t.Helper()
	got := indexColumns(t, tx, model, name)
	if len(got) != len(want) {
		t.Fatalf("index %s columns = %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %s columns = %v, want %v", name, got, want)
		}
	}
}
