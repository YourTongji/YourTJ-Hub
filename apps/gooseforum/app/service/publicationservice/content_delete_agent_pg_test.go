package publicationservice_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/contentdeleteservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
	"gorm.io/gorm"
)

func TestPostgreSQLReplyDeletionAndMaterializationShareLockOrder(t *testing.T) {
	for _, moderator := range []bool{false, true} {
		t.Run(fmt.Sprintf("moderator=%t", moderator), func(t *testing.T) {
			conn := setupAgentPolicyPostgreSQL(t)
			t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "delete-pg")
			t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch")
			t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
			since := time.Now().Add(-time.Hour)
			sourceKey := "post:101"
			for _, row := range []any{
				&users.EntityComplete{Id: 1, Username: "delete-bot", ActorType: users.ActorTypeBot},
				&users.EntityComplete{Id: 2, Username: "delete-human", Prestige: pointservice.PostCreatedReward},
				&agents.Entity{UserId: 1, TokenPrefix: "delete-token", TokenHash: "hash", Enabled: 1, EventsEnabled: true, SubscriptionGeneration: 1, EventsEnabledAt: &since, EventTypes: `["agent.mentioned"]`},
				&topics.Entity{Id: 10, UserId: 2, Status: 1, FirstPostId: 100, PostCount: 3, PostSeq: 3},
				&posts.Entity{Id: 100, TopicId: 10, UserId: 2, PostNo: 1, Content: "Public discussion"},
				&posts.Entity{Id: 101, TopicId: 10, UserId: 2, PostNo: 2, Content: "Reply with a creation reward"},
				&posts.Entity{Id: 102, TopicId: 10, UserId: 2, PostNo: 3, Content: "Please help @delete-bot"},
				&postRevisions.Entity{Id: 1, PostId: 102, Version: 1, EditorId: 2, Content: "Please help @delete-bot"},
				&userPoints.Entity{UserId: 2, CurrentPoints: 100 + pointservice.PostCreatedReward},
				&pointsRecord.Entity{UserId: 2, Action: pointservice.PointsActionPostCreated.Code(), PointsChange: pointservice.PostCreatedReward, SourceKey: &sourceKey},
			} {
				if err := conn.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := conn.Transaction(func(tx *gorm.DB) error { return agenteventservice.CapturePublicTx(tx, &posts.Entity{Id: 102}) }); err != nil {
				t.Fatal(err)
			}
			var intent agentEvents.Intent
			if err := conn.Where("post_id = ?", 102).Take(&intent).Error; err != nil {
				t.Fatal(err)
			}
			task, claimed, err := taskQueue.ClaimTask(intent.TaskID)
			if err != nil || !claimed {
				t.Fatalf("claim: %t %v", claimed, err)
			}
			type workerKey struct{}
			beforeTopic, beforeUsers := make(chan struct{}), make(chan struct{})
			var topicOnce, usersOnce sync.Once
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			if err := conn.Callback().Query().Before("gorm:query").Register("test:delete-materialize-locks", func(tx *gorm.DB) {
				if _, locked := tx.Statement.Clauses["FOR"]; !locked {
					return
				}
				if tx.Statement.Context.Value(workerKey{}) == true {
					if tx.Statement.Table == "users" {
						usersOnce.Do(func() { close(beforeUsers) })
					}
				} else if tx.Statement.Table == "topics" {
					topicOnce.Do(func() {
						close(beforeTopic)
						select {
						case <-beforeUsers:
						case <-ctx.Done():
						}
					})
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Callback().Query().Remove("test:delete-materialize-locks") })
			results := make(chan error, 2)
			go func() {
				if moderator {
					results <- contentdeleteservice.DeletePostAsModerator(1, 101, "Moderation removal")
				} else {
					_, err := contentdeleteservice.DeletePostByUser(2, 101)
					results <- err
				}
			}()
			select {
			case <-beforeTopic:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			go func() { results <- agenteventservice.HandleTask(context.WithValue(ctx, workerKey{}, true), &task) }()
			for range 2 {
				select {
				case err := <-results:
					if err != nil {
						t.Errorf("delete/materialize: %v", err)
					}
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			var count int64
			if err := conn.Model(&agentEvents.Entity{}).Where("post_id = ?", 102).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Fatalf("source event count = %d, want 1", count)
			}
		})
	}
}
