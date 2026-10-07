package dataservice

import (
	"context"
	"encoding/json"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

func TestAnonymousExportsRedactOwnerAndCannotRestorePrivateBindings(t *testing.T) {
	setupDataTestDB(t)
	conn := db.Connect()
	const uid = "1234567890abcdef1234567890abcdef"
	const owner = uint64(9893301)
	for _, row := range []any{&topics.Entity{Id: 1, UserId: owner, PersonaUID: uid}, &posts.Entity{Id: 2, TopicId: 1, UserId: owner, LastEditorId: owner, DeletedBy: owner, PersonaUID: uid, IsAnonymous: true}, &postRevisions.Entity{Id: 3, PostId: 2, EditorId: owner}} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"topics", "posts", "postRevisions"} {
		rows, err := fetchExportRows(table, 0, 10)
		if err != nil || len(rows) != 1 {
			t.Fatal(table, rows, err)
		}
		for _, field := range []string{"userId", "lastEditorId", "deletedBy", "editorId"} {
			if value, exists := rows[0].Fields[field]; exists && value != uint64(0) {
				t.Fatalf("%s export leaked %s=%v", table, field, value)
			}
		}
		if table != "postRevisions" {
			encoded, err := json.Marshal(map[string]any{table: []any{rows[0].Fields}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ImportData(context.Background(), encoded, "json"); err == nil {
				t.Fatal("ordinary import silently accepted a private-binding snapshot")
			}
		}
	}
	for _, table := range []string{"anonymous_bindings", "anonymous_reveal_audits", "anonymous_personas", "anonymous_name_batches", "anonymous_name_quotas"} {
		if AllowedExportTables[table] {
			t.Fatalf("restricted table %s is publicly exportable", table)
		}
	}
}
