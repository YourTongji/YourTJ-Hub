package topics

import (
	"encoding/json"
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFirstPostVisibilityUsesPointLookupOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("feed_rank_%d", time.Now().UnixNano())
	if err = conn.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Exec("DROP SCHEMA " + schema + " CASCADE") })
	scoped, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := scoped.DB()
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	if err = scoped.AutoMigrate(&Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err = scoped.Exec(`INSERT INTO topics(id,user_id,status,first_post_id,rank_params_hash,rank_score,rank_ready) SELECT x,1,1,x*20,'test',x,true FROM generate_series(1,10000) x`).Error; err != nil {
		t.Fatal(err)
	}
	if err = scoped.Exec(`INSERT INTO posts(id,topic_id,post_no,user_id) SELECT x,(x-1)/20+1,(x-1)%20+1,1 FROM generate_series(1,200000) x`).Error; err != nil {
		t.Fatal(err)
	}
	if err = scoped.Exec("ANALYZE topics;ANALYZE posts").Error; err != nil {
		t.Fatal(err)
	}
	var plan string
	sql := `EXPLAIN(ANALYZE,BUFFERS,FORMAT JSON) SELECT topics.id FROM topics WHERE status=1 AND process_status=0 AND deleted_at IS NULL AND visibility_status='ACTIVE' AND topic_type=0 AND ` + firstPostVisibleSQL + ` AND rank_params_hash='test' AND rank_ready=true ORDER BY rank_score DESC,id DESC LIMIT 21`
	if err = scoped.Raw(sql, 0).Scan(&plan).Error; err != nil {
		t.Fatal(err)
	}
	var nodes []map[string]any
	if err = json.Unmarshal([]byte(plan), &nodes); err != nil {
		t.Fatal(err)
	}
	pointLookup := false
	var check func(any)
	check = func(v any) {
		switch n := v.(type) {
		case map[string]any:
			if n["Relation Name"] == "posts" {
				kind, _ := n["Node Type"].(string)
				if strings.Contains(kind, "Seq Scan") {
					t.Fatalf("public visibility scanned posts: %s", plan)
				}
				if kind == "Index Scan" || kind == "Index Only Scan" {
					pointLookup = true
				}
			}
			for _, child := range n {
				check(child)
			}
		case []any:
			for _, child := range n {
				check(child)
			}
		}
	}
	check(nodes[0])
	if !pointLookup {
		t.Fatalf("missing indexed first-post lookup: %s", plan)
	}
}
