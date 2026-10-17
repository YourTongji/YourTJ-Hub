package topics

import (
	"context"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"testing"
)

func TestPublicListRejectsLifecycleDeletedFirstPost(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	topic := Entity{Title: "hidden first post", Status: 1, UserId: 981231}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: topic.UserId, VisibilityStatus: posts.VisibilityUserDeleted}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&post); db.Unscoped().Delete(&topic) })
	page := Page(PageQuery{FilterStatus: true, UserId: topic.UserId, Page: 1, PageSize: 20})
	if len(page.Data) != 0 {
		t.Fatal("public list exposed a first post in the deletion lifecycle")
	}
}

func TestFiniteRecallIDsCannotBypassAuthoritativeVisibility(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	author := users.EntityComplete{Id: 981232, Username: "finite-author"}
	if err := conn.Create(&author).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&author) })
	topic := Entity{Title: "finite hidden candidate", Status: 1, UserId: author.Id}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: topic.UserId, VisibilityStatus: posts.VisibilityUserDeleted}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).UpdateColumn("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&post); conn.Unscoped().Delete(&topic) })
	ids, err := RecallRankIDs(context.Background(), Recall{Source: "author", Authors: []uint64{topic.UserId}, Limit: 20})
	if err != nil || len(ids) != 1 || ids[0] != topic.Id {
		t.Fatalf("bounded ID candidate was not recalled: %v %v", ids, err)
	}
	rows, err := RankTopics(context.Background(), ids)
	if err != nil || len(rows) != 0 {
		t.Fatalf("hidden first post reached authoritative metadata: %v %v", rows, err)
	}
}

func TestPublicRankRejectsFrozenAndClosedAuthorsImmediately(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	author := users.EntityComplete{Id: 981233, Username: "rank-eligible-author"}
	if err := conn.Create(&author).Error; err != nil {
		t.Fatal(err)
	}
	topic := Entity{Title: "rank eligible", Status: 1, UserId: author.Id}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: author.Id}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).UpdateColumn("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&post); conn.Unscoped().Delete(&topic); conn.Unscoped().Delete(&author) })
	if _, err := RankTopic(context.Background(), topic.Id); err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&author).UpdateColumn("is_frozen", 1).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := RankTopic(context.Background(), topic.Id); err == nil {
		t.Fatal("frozen author retained ranking before worker invalidation")
	}
	if err := conn.Model(&author).UpdateColumn("is_frozen", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := RankTopic(context.Background(), topic.Id); err != nil {
		t.Fatal("unfrozen author was not restored", err)
	}
	if err := conn.Delete(&author).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := RankTopic(context.Background(), topic.Id); err == nil {
		t.Fatal("closed author retained ranking")
	}
}

// TestGetPublishedSkipsWikiTopics 回归评审 S3：GetPublished 复用规范公开谓词后带上
// topic_type=0，wiki 页面主题（topic_type=1）不再被单篇详情/导出返回；改回论坛类型后
// 同一行立即可读，证明拒绝原因就是 topic_type 而不是其他谓词。
func TestGetPublishedSkipsWikiTopics(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	topic := Entity{Title: "wiki page topic", Status: 1, UserId: 981233, TopicType: TopicTypeWiki}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: topic.UserId, VisibilityStatus: posts.VisibilityActive}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&post); conn.Unscoped().Delete(&topic) })

	// 未命中即返回零值实体（err = record not found），断言只看实体本身。
	if entity, _ := GetPublished(topic.Id); entity.Id != 0 {
		t.Fatalf("GetPublished returned wiki topic %+v, want not found", entity)
	}
	if err := conn.Model(&Entity{}).Where("id = ?", topic.Id).Update("topic_type", TopicTypeForum).Error; err != nil {
		t.Fatal(err)
	}
	if entity, err := GetPublished(topic.Id); err != nil || entity.Id != topic.Id {
		t.Fatalf("GetPublished after switching to forum type = %+v (err=%v), want the topic", entity, err)
	}
}

// TestGetPublishedForExportIncludesWikiTopics 回归 issue #1166：单篇 Markdown 导出恢复 wiki 页面
// （#1160 让导出复用公开谓词后把 wiki 单篇打成 404），同一行仍不被公开口径返回。
func TestGetPublishedForExportIncludesWikiTopics(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	topic := Entity{Title: "wiki export topic", Status: 1, UserId: 981234, TopicType: TopicTypeWiki}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: topic.UserId, VisibilityStatus: posts.VisibilityActive}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&post); conn.Unscoped().Delete(&topic) })

	if entity, err := GetPublishedForExport(topic.Id); err != nil || entity.Id != topic.Id {
		t.Fatalf("GetPublishedForExport(wiki) = %+v (err=%v), want the topic", entity, err)
	}
	if entity, _ := GetPublished(topic.Id); entity.Id != 0 {
		t.Fatalf("GetPublished returned wiki topic %+v, want not found", entity)
	}
}

// TestGetPublishedForExportRequiresVisibleFirstPost 导出与公开口径共享首楼约束：首楼待审或被
// 软删时 wiki 页面同样不导出（只有标题没有正文的孤儿文档不对外）。
func TestGetPublishedForExportRequiresVisibleFirstPost(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	topic := Entity{Title: "wiki export hidden first post", Status: 1, UserId: 981235, TopicType: TopicTypeWiki}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: topic.UserId, VisibilityStatus: posts.VisibilityActive, ProcessStatus: posts.ProcessStatusPending}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&post); conn.Unscoped().Delete(&topic) })

	if entity, _ := GetPublishedForExport(topic.Id); entity.Id != 0 {
		t.Fatalf("GetPublishedForExport exported topic with pending first post: %+v", entity)
	}
	if err := conn.Model(&posts.Entity{}).Where("id = ?", post.Id).Update("process_status", posts.ProcessStatusNormal).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Delete(&posts.Entity{}, post.Id).Error; err != nil {
		t.Fatal(err)
	}
	if entity, _ := GetPublishedForExport(topic.Id); entity.Id != 0 {
		t.Fatalf("GetPublishedForExport exported topic with deleted first post: %+v", entity)
	}
}
