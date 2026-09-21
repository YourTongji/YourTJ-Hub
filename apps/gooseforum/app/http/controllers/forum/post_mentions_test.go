package forum

import (
	"encoding/json"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestPostPayloadMentionsRespectSourceAndVisibility(t *testing.T) {
	conn := setupRevisionTestDB(t)
	user := users.MakeUser("mention_payload", "secret123", "mention-payload@example.com")
	if err := users.Create(user); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Delete(user) })
	content := "😀 @mention_payload `@mention_payload` [@mention_payload](/) $@mention_payload$ @unknown"
	for _, hidden := range []bool{false, true} {
		entity := &posts.Entity{Id: 999991, TopicId: 999992, PostNo: 1, UserId: user.Id, Content: content}
		if hidden {
			entity.ProcessStatus = 1
		}
		payload, _ := buildPostPayloads([]*posts.Entity{entity}, map[uint64]*users.EntityComplete{user.Id: user}, 0, false, entity)
		raw, err := json.Marshal(payload[0])
		if err != nil {
			t.Fatal(err)
		}
		var wire struct {
			Mentions []struct {
				Username   string
				UserID     uint64
				Start, End int
			}
		}
		if err := json.Unmarshal(raw, &wire); err != nil {
			t.Fatal(err)
		}
		if hidden {
			if len(wire.Mentions) != 0 {
				t.Fatal("hidden content leaked mention identities")
			}
			continue
		}
		if len(wire.Mentions) != 1 {
			t.Fatalf("expected one resolved source mention, got %s", raw)
		}
		m := wire.Mentions[0]
		if m.Username != "mention_payload" || m.UserID != user.Id || m.Start != 3 || m.End != 19 {
			t.Fatalf("wrong UTF-16 mapping: %+v", m)
		}
	}
}
