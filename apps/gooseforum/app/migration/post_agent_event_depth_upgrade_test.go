package migration

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestPostAgentEventDepthUpgradePreservesLegacyRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY, content TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO posts (id,content) VALUES (1,'legacy')").Error; err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 2; n++ {
		if err := upgradePostAgentEventDepth(db); err != nil {
			t.Fatal(err)
		}
	}
	var row struct {
		Content         string
		AgentEventDepth int
	}
	if err := db.Table("posts").Where("id = 1").Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Content != "legacy" || row.AgentEventDepth != 0 {
		t.Fatalf("legacy post changed: %#v", row)
	}
}
