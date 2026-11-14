package api

import (
	"context"
	"fmt"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
)

func TestPublicationPointsCommitWithoutEventConsumer(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	const uid, cid uint64 = 985001, 985002
	createTopicWriteUser(t, conn, uid, "atomic-points")
	if err := conn.Create(&category.Entity{Id: cid, Name: "points", Slug: "atomic-points"}).Error; err != nil {
		t.Fatal(err)
	}
	result := WriteTopic(component.BetterRequest[WriteTopicReq]{UserId: uid, Params: WriteTopicReq{
		Title: "Atomic points topic", Content: "Published content must commit its reward", CategoryId: []uint64{cid}, TopicStatus: 1,
	}})
	topicID, ok := result.Data.Result.(uint64)
	if result.Data.Code != component.SUCCESS || !ok {
		t.Fatalf("write: %+v", result)
	}
	var record pointsRecord.Entity
	if err := conn.Where("source_key = ?", fmt.Sprintf("topic:%d", topicID)).Take(&record).Error; err != nil {
		t.Fatalf("committed topic has no durable reward without an event consumer: %v", err)
	}
	if record.PointsChange != pointservice.TopicPublishedReward {
		t.Fatalf("reward = %d", record.PointsChange)
	}
	result = CreatePost(component.BetterRequest[CreatePostReq]{UserId: uid, Params: CreatePostReq{
		TopicId: topicID, Content: "A reply also commits its reward",
	}})
	if result.Data.Code != component.SUCCESS {
		t.Fatalf("reply: %+v", result)
	}
	postID := result.Data.Result.(map[string]any)["id"].(uint64)
	record = pointsRecord.Entity{}
	if err := conn.Where("source_key = ?", fmt.Sprintf("post:%d", postID)).Take(&record).Error; err != nil {
		t.Fatalf("committed reply has no durable reward: %v", err)
	}
	if record.PointsChange != pointservice.PostCreatedReward {
		t.Fatalf("reply reward = %d", record.PointsChange)
	}
}

func TestPublicationPointsFailureRollsBackContent(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	const uid, cid uint64 = 985011, 985012
	createTopicWriteUser(t, conn, uid, "points-failure")
	if err := conn.Create(&category.Entity{Id: cid, Name: "points failure", Slug: "points-failure"}).Error; err != nil {
		t.Fatal(err)
	}
	fired := injectWriteFailure(t, conn, "create", "points_record", 1)
	const title, content = "Points failure rolls back", "No published content without its points intent"
	result := WriteTopic(component.BetterRequest[WriteTopicReq]{UserId: uid, Params: WriteTopicReq{
		Title: title, Content: content, CategoryId: []uint64{cid}, TopicStatus: 1,
	}})
	if !fired.Load() {
		t.Fatal("publication did not attempt a transactional ledger write")
	}
	if result.Data.Code == component.SUCCESS {
		t.Fatal("publication succeeded despite ledger failure")
	}
	assertTopicWriteAbsent(t, conn, title, content)
}

func TestPublicationPointsDraftAndLegacyApproval(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	for i, kind := range []string{"draft", "draft-save", "topic", "post"} {
		t.Run(kind, func(t *testing.T) {
			uid := uint64(985100 + i*10)
			tid, pid := uid+1, uid+2
			createTopicWriteUser(t, conn, uid, "delayed-points")
			createPublishedTopicWithPost(t, conn, uid, tid, pid, "delayed publication", "content")
			if kind == "draft" || kind == "draft-save" {
				if err := conn.Model(&topics.Entity{}).Where("id = ?", tid).Update("status", 0).Error; err != nil {
					t.Fatal(err)
				}
			} else {
				if err := conn.Model(&posts.Entity{}).Where("id = ?", pid).Update("process_status", posts.ProcessStatusPending).Error; err != nil {
					t.Fatal(err)
				}
				if kind == "topic" {
					if err := conn.Model(&topics.Entity{}).Where("id = ?", tid).Update("process_status", topics.ProcessStatusPending).Error; err != nil {
						t.Fatal(err)
					}
				} else {
					if err := conn.Model(&posts.Entity{}).Where("id = ?", pid).Update("post_no", 2).Error; err != nil {
						t.Fatal(err)
					}
				}
			}
			var result component.Response
			switch kind {
			case "draft-save":
				cid := uid + 3
				if err := conn.Create(&category.Entity{Id: cid, Name: "draft save", Slug: "draft-save"}).Error; err != nil {
					t.Fatal(err)
				}
				result = WriteTopic(component.BetterRequest[WriteTopicReq]{UserId: uid, Params: WriteTopicReq{TopicId: tid, Title: "Draft published by save", Content: "Publish this draft by saving it", CategoryId: []uint64{cid}, TopicStatus: 1}})
			case "draft":
				result = UpdateTopicStatus(component.BetterRequest[TopicStatusReq]{UserId: uid, Params: TopicStatusReq{TopicId: tid, TopicStatus: 1}})
			default:
				id := tid
				if kind == "post" {
					id = pid
				}
				result = reviewContent(context.Background(), ReviewActionReq{Kind: kind, Id: id, Approve: true}, 0, true)
			}
			if result.Data.Code != component.SUCCESS {
				t.Fatalf("publication: %+v", result)
			}
			source, amount, action := fmt.Sprintf("topic:%d", tid), pointservice.TopicPublishedReward, pointservice.PointsActionTopicPublished
			if kind == "post" {
				source, amount, action = fmt.Sprintf("post:%d", pid), pointservice.PostCreatedReward, pointservice.PointsActionPostCreated
			}
			var record pointsRecord.Entity
			if err := conn.Where("source_key = ?", source).Take(&record).Error; err != nil {
				t.Fatal(err)
			}
			if record.PointsChange != amount {
				t.Fatalf("reward = %d", record.PointsChange)
			}
			// Older queued events may replay after the transaction has already paid.
			if err := pointservice.RewardPoints(uid, amount, action, source); err != nil {
				t.Fatal(err)
			}
			var balance int64
			if err := conn.Table("user_points").Where("user_id = ?", uid).Select("current_points").Scan(&balance).Error; err != nil {
				t.Fatal(err)
			}
			if balance != amount {
				t.Fatalf("replay balance = %d, want %d", balance, amount)
			}
		})
	}
}
