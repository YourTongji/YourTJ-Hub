package pointservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestPublicationRewardEligibility(t *testing.T) {
	for _, name := range []string{"bot", "wiki-page", "wiki-reply", "anonymous-reply", "removed-reply", "rejected-topic"} {
		t.Run(name, func(t *testing.T) {
			conn := newPointsTestDB(t)
			createPointsUser(t, conn, 1, 0)
			topic := topics.Entity{Id: 10, UserId: 1, Status: 1}
			post := posts.Entity{Id: 11, TopicId: 10, UserId: 1, PostNo: 2}
			want := int64(0)
			switch name {
			case "bot":
				if err := conn.Model(&users.EntityComplete{}).Where("id = ?", 1).Update("actor_type", users.ActorTypeBot).Error; err != nil {
					t.Fatal(err)
				}
			case "wiki-page":
				topic.TopicType, post.PostNo = topics.TopicTypeWiki, 1
			case "wiki-reply":
				topic.TopicType, want = topics.TopicTypeWiki, PostCreatedReward
			case "anonymous-reply":
				post.IsAnonymous, want = true, PostCreatedReward
			case "removed-reply":
				post.VisibilityStatus = posts.VisibilityUserDeleted
			case "rejected-topic":
				topic.ProcessStatus, post.PostNo = topics.ProcessStatusBlocked, 1
			}
			if err := conn.Create(&topic).Error; err != nil {
				t.Fatal(err)
			}
			if err := conn.Create(&post).Error; err != nil {
				t.Fatal(err)
			}
			if err := conn.Transaction(func(tx *gorm.DB) error { return RewardPublicationTx(tx, topic.Id, post.Id) }); err != nil {
				t.Fatal(err)
			}
			var sum int64
			if err := conn.Model(&pointsRecord.Entity{}).Where("user_id = ?", 1).Select("COALESCE(SUM(points_change), 0)").Scan(&sum).Error; err != nil {
				t.Fatal(err)
			}
			if sum != want {
				t.Fatalf("reward = %d, want %d", sum, want)
			}
		})
	}
}
