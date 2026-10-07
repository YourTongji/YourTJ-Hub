package feedservice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestPersonaFeedCannotRevealFollowedMainAccount(t *testing.T) {
	conn := telemetryDB(t)
	if err := conn.AutoMigrate(&userFollow.Entity{}, &users.BlockEntity{}, &topicUserAction.Entity{}); err != nil {
		t.Fatal(err)
	}
	const viewer, owner, personaTopic, memberTopic uint64 = 939101, 939102, 939103, 939104
	now := time.Now().Add(-time.Hour)
	rows := []any{
		&users.EntityComplete{Id: viewer, Username: "feed-persona-reader", Email: "feed-persona-reader@example.invalid"},
		&users.EntityComplete{Id: owner, Username: "feed-private-owner", Email: "feed-private-owner@example.invalid"},
		&userFollow.Entity{UserId: viewer, FollowUserId: owner, Status: 1},
		&topicUserAction.Entity{UserId: viewer, TopicId: personaTopic, LikedAt: &now},
		&topics.Entity{Id: personaTopic, UserId: owner, PersonaUID: strings.Repeat("a", 32), Status: 1, FirstPostId: personaTopic, FirstPublicAt: &now},
		&topics.Entity{Id: memberTopic, UserId: owner, Status: 1, FirstPostId: memberTopic, FirstPublicAt: &now},
		&posts.Entity{Id: personaTopic, TopicId: personaTopic, UserId: owner, PersonaUID: strings.Repeat("a", 32), IsAnonymous: true, PostNo: 1, FirstPublicAt: &now},
		&posts.Entity{Id: memberTopic, TopicId: memberTopic, UserId: owner, PostNo: 1, FirstPublicAt: &now},
	}
	for _, row := range rows {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(row) })
	}
	ctx := context.Background()
	for _, source := range []string{"following", "author"} {
		ids, err := topics.RecallRankIDs(ctx, topics.Recall{Viewer: viewer, Source: source, Authors: []uint64{owner}, Limit: 20})
		if err != nil || len(ids) != 1 || ids[0] != memberTopic {
			t.Errorf("%s recalled a private persona owner: %v, %v", source, ids, err)
		}
	}
	learned, err := getProfile(ctx, viewer)
	if err != nil || learned.Count != 1 || len(learned.Authors) != 0 {
		t.Fatalf("persona activity learned a private main-account author: %+v, %v", learned, err)
	}
	profileMu.Lock()
	profiles[viewer] = profile{Created: time.Now(), Count: 10, Categories: map[uint64]float64{}, Authors: map[uint64]float64{owner: 1}}
	profileMu.Unlock()
	snapshot, err := buildSnapshot(ctx, viewer, feedconfig.Current())
	if err != nil {
		t.Fatal(err)
	}
	seen := map[uint64]bool{}
	for _, candidate := range snapshot.Items {
		seen[candidate.ID] = true
		if candidate.ID == personaTopic && (candidate.Reason == "following" || candidate.Features[0] != 0 || candidate.Features[5] != 0 || candidate.Sources&17 != 0) {
			t.Errorf("persona inherited main-account relationships: %+v", candidate)
		}
		if candidate.ID == memberTopic && candidate.Reason != "following" {
			t.Errorf("main-account following lost: %+v", candidate)
		}
	}
	if !seen[personaTopic] || !seen[memberTopic] {
		t.Fatalf("public topics lost from non-identity recall: %v", seen)
	}
}
