package forum

import (
	"encoding/json"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/vo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"testing"
)

func TestTrackedTopicInteractionState(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&topicUserAction.Entity{}); err != nil {
		t.Fatal(err)
	}
	const owner uint64 = 987654321
	t.Cleanup(func() { conn.Where("user_id IN ?", []uint64{owner, owner + 1}).Delete(&topicUserAction.Entity{}) })
	topicUserAction.SetLiked(owner, 91, true)
	topicUserAction.SetBookmarked(owner, 92, true)
	topicUserAction.SetLiked(owner+1, 92, true)
	input := []*vo.TopicsSimpleVo{{Id: 91}, {Id: 92}, {Id: 93}}
	raw, err := json.Marshal(buildTrackedTopicPayloads(owner, input))
	if err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err = json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for i, want := range [][2]bool{{true, false}, {false, true}, {false, false}} {
		if got[i]["liked"] != want[0] || got[i]["bookmarked"] != want[1] {
			t.Fatalf("topic %d state: %s", i, raw)
		}
	}
	raw, err = json.Marshal(buildTrackedTopicPayloads(0, input))
	if err != nil {
		t.Fatal(err)
	}
	got = nil
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, item := range got {
		if _, ok := item["liked"]; ok {
			t.Fatalf("guest contains personal state: %s", raw)
		}
	}
}
