package datamigration

import (
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
)

// v18：为部署前已存在、尚无版本的存量帖子回填 v1 快照（editor=作者、
// 内容取当前正文）。避免存量帖首次编辑覆写正文后原始内容永久丢失
// （oierxjn review P1 要求的旧数据升级覆盖）。幂等：已有版本的帖子、
// 正文为空的帖子（已删除/隐私擦除）跳过。
func TestBackfillPostRevisionSeedsFillsLegacyPosts(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&posts.Entity{}, &postRevisions.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", []uint64{9001, 9002, 9003, 9004}).Delete(&posts.Entity{})
		conn.Unscoped().Where("post_id IN ?", []uint64{9001, 9002, 9003, 9004}).Delete(&postRevisions.Entity{})
	})

	// 旧帖 A：无版本，正文非空，rendered_html 已有 → 回填 v1，rendered 用原值
	if err := conn.Create(&posts.Entity{
		Id: 9001, TopicId: 1, PostNo: 1, UserId: 1001,
		Content: "legacy body A", RenderedHTML: "<p>legacy body A</p>",
		ProcessStatus: posts.ProcessStatusNormal,
	}).Error; err != nil {
		t.Fatalf("create legacy A: %v", err)
	}
	// 旧帖 B：无版本，正文非空，rendered_html 为空 → 回填 v1，rendered 重渲染
	if err := conn.Create(&posts.Entity{
		Id: 9002, TopicId: 1, PostNo: 2, UserId: 1002,
		Content: "legacy body B", ProcessStatus: posts.ProcessStatusNormal,
	}).Error; err != nil {
		t.Fatalf("create legacy B: %v", err)
	}
	// 旧帖 C：已有版本 v1 → 跳过
	if err := conn.Create(&posts.Entity{
		Id: 9003, TopicId: 1, PostNo: 3, UserId: 1003,
		Content: "already versioned", ProcessStatus: posts.ProcessStatusNormal,
	}).Error; err != nil {
		t.Fatalf("create legacy C: %v", err)
	}
	if err := conn.Create(&postRevisions.Entity{
		PostId: 9003, Version: 1, EditorId: 1003,
		Content: "already versioned", ProcessStatus: posts.ProcessStatusNormal,
	}).Error; err != nil {
		t.Fatalf("seed C v1: %v", err)
	}
	// 旧帖 D：正文为空（已删除/清空）→ 跳过
	if err := conn.Create(&posts.Entity{
		Id: 9004, TopicId: 1, PostNo: 4, UserId: 1004,
		Content: "", ProcessStatus: posts.ProcessStatusNormal,
	}).Error; err != nil {
		t.Fatalf("create legacy D: %v", err)
	}

	result := BackfillPostRevisionSeedsWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("backfill failed: %+v", result)
	}
	if result.Seeded != 2 {
		t.Fatalf("seeded = %d, want 2 (A/B)", result.Seeded)
	}
	if result.Skipped != 2 {
		t.Fatalf("skipped = %d, want 2 (C already-versioned, D empty)", result.Skipped)
	}

	// A：v1 内容=原正文，rendered=原值，editor=作者
	var aVers []postRevisions.Entity
	conn.Where("post_id = ?", 9001).Order("version ASC").Find(&aVers)
	if len(aVers) != 1 || aVers[0].Version != 1 || aVers[0].EditorId != 1001 ||
		aVers[0].Content != "legacy body A" || aVers[0].RenderedHTML != "<p>legacy body A</p>" {
		t.Fatalf("A v1 = %#v, want original body + original rendered", aVers)
	}
	// B：v1 内容=原正文，rendered 已重渲染（非空）
	var bVers []postRevisions.Entity
	conn.Where("post_id = ?", 9002).Order("version ASC").Find(&bVers)
	if len(bVers) != 1 || bVers[0].Version != 1 || bVers[0].EditorId != 1002 || bVers[0].Content != "legacy body B" {
		t.Fatalf("B v1 = %#v, want original body", bVers)
	}
	if bVers[0].RenderedHTML == "" {
		t.Fatalf("B v1 rendered_html empty, want re-rendered")
	}
	// C：仍是 1 条（不重复播种）；D：无版本
	var cCount int64
	conn.Model(&postRevisions.Entity{}).Where("post_id = ?", 9003).Count(&cCount)
	if cCount != 1 {
		t.Fatalf("C revision count = %d, want 1", cCount)
	}
	var dCount int64
	conn.Model(&postRevisions.Entity{}).Where("post_id = ?", 9004).Count(&dCount)
	if dCount != 0 {
		t.Fatalf("D revision count = %d, want 0 (empty content skipped)", dCount)
	}

	// 幂等：再次运行不重复播种
	second := BackfillPostRevisionSeedsWithDB(conn)
	if second.Seeded != 0 || second.Failed > 0 {
		t.Fatalf("second backfill = %+v, want no-op", second)
	}
}
