package publicationservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationAppeals"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
)

// 回复修订的 category_ids 是 nil 切片时 serializer:json 会写成字面量 'null'；
// PostgreSQL 的 jsonb_array_elements_text 收到标量会直接报错，分类版主队列不能因此 500。
func TestPostgreSQLModerationAppealsToleratesNullCandidateCategories(t *testing.T) {
	conn := setupAgentPolicyPostgreSQL(t)
	ctx := context.Background()
	const owner, mod, globalMod uint64 = 961701, 961702, 961703
	for _, row := range []users.EntityComplete{{Id: owner, Username: "appeal-pg-owner"}, {Id: mod, Username: "appeal-pg-moderator"}, {Id: globalMod, Username: "appeal-pg-global"}} {
		if err := conn.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := conn.Create(&moderators.Entity{UserId: mod, ScopeType: moderators.ScopeCategory, ScopeId: 9617, Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&moderators.Entity{UserId: globalMod, ScopeType: moderators.ScopeGlobal, Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{UserId: owner, Title: "pg original", Status: 1, ProcessStatus: topics.ProcessStatusBlocked, CategoryIds: []uint64{9617}}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, UserId: owner, PostNo: 1, Content: "pg blocked content", ProcessStatus: posts.ProcessStatusBlocked}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	revision := postRevisions.Entity{PostId: post.Id, Version: 1, Title: "pg candidate", Content: "pg immutable rejected body", CategoryIds: nil, ProcessStatus: posts.ProcessStatusBlocked, ReviewReason: "pg decision", ReviewedAt: &at}
	if err := conn.Create(&revision).Error; err != nil {
		t.Fatal(err)
	}
	// 存量数据里确实出现过字面量 'null'（同族 topics_rep 也为此加了守卫），这里直接复现该形状。
	if err := conn.Exec("UPDATE post_revisions SET category_ids = 'null' WHERE id = ?", revision.Id).Error; err != nil {
		t.Fatal(err)
	}
	var stored *string
	if err := conn.Table("post_revisions").Select("category_ids").Where("id = ?", revision.Id).Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored == nil || *stored != "null" {
		t.Fatalf("candidate categories = %v; this regression needs the literal null", stored)
	}
	if err := conn.Model(&post).Update("latest_revision_id", revision.Id).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&topicCategoryIndex.Entity{TopicId: topic.Id, CategoryId: 9617, Effective: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&moderationAppeals.Entity{PostId: post.Id, RevisionId: revision.Id, Reason: "pg explanation", Status: moderationAppeals.Pending}).Error; err != nil {
		t.Fatal(err)
	}
	// 分类版主必须能安全查询：不可校验候选分类时保守不展示，而不是整条队列报错。
	items, total, err := publicationservice.ModeratorAppeals(ctx, mod, 1, 20, "pending")
	if err != nil {
		t.Fatalf("category moderator queue failed on null candidate categories: %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("unverifiable candidate categories leaked to category moderator: %+v total=%d", items, total)
	}
	// 全局版主不受分类过滤影响，证明申诉本身可查且查询确实执行通过。
	items, total, err = publicationservice.ModeratorAppeals(ctx, globalMod, 1, 20, "pending")
	if err != nil || total != 1 || len(items) != 1 || items[0].RevisionId != revision.Id {
		t.Fatalf("global queue: %+v total=%d err=%v", items, total, err)
	}
}
