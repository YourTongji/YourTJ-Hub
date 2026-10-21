package datamigration

import (
	"strings"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
)

// issue #1304：存量话题摘要保留 Markdown 转义反斜杠（贴纸 token 属于摘要本体，列表
// 卡片按图片渲染，回填后保留）；修复摘要生成后需按首帖正文重算存量行；未变化的
// 摘要不写库（幂等），首帖缺失/正文为空跳过。
func TestBackfillTopicExcerptRewritesPollutedExcerpts(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	const (
		dirtyTopicID   = uint64(9_800_003_001)
		cleanTopicID   = uint64(9_800_003_002)
		orphanID       = uint64(9_800_003_003)
		deletedTopicID = uint64(9_800_003_004)
		dirtyPostID    = uint64(9_800_003_011)
		cleanPostID    = uint64(9_800_003_012)
		deletedPostID  = uint64(9_800_003_013)
		cleanExcerpt   = "已经干净的摘要文本"
	)
	staleAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", []uint64{dirtyTopicID, cleanTopicID, orphanID, deletedTopicID}).Delete(&topics.Entity{})
		conn.Unscoped().Where("id IN ?", []uint64{dirtyPostID, cleanPostID, deletedPostID}).Delete(&posts.Entity{})
	})

	// 存量脏摘要：带转义反斜杠（修复前的 ExtractDescription 产物）；贴纸 token 回填后保留。
	dirty := topics.Entity{Id: dirtyTopicID, Title: "dirty", UserId: 1, FirstPostId: dirtyPostID,
		Excerpt: `换算\.\}\.\}好高级 [:sticker:Atari哭哭:]`}
	clean := topics.Entity{Id: cleanTopicID, Title: "clean", UserId: 1, FirstPostId: cleanPostID, Excerpt: cleanExcerpt}
	orphan := topics.Entity{Id: orphanID, Title: "orphan", UserId: 1, FirstPostId: 0, Excerpt: "keep me"}
	// 话题与首帖都软删除：回填的扫描、首帖查询与写入三处都必须 Unscoped，
	// 否则删除期间行会带着修复前的脏摘要留在库里，恢复后原样回归。
	deleted := topics.Entity{Id: deletedTopicID, Title: "deleted", UserId: 1, FirstPostId: deletedPostID,
		Excerpt:   `换算\.\}\.\}好高级 [:sticker:Atari哭哭:]`,
		DeletedAt: gorm.DeletedAt{Time: staleAt, Valid: true}}
	for _, topic := range []*topics.Entity{&dirty, &clean, &orphan, &deleted} {
		if err := conn.Create(topic).Error; err != nil {
			t.Fatalf("create topic %d: %v", topic.Id, err)
		}
	}
	// updated_at 固定为历史值：回填只改 excerpt，不得扰动首页默认排序键。
	if err := conn.Model(&topics.Entity{}).Where("id = ?", dirtyTopicID).
		UpdateColumn("updated_at", staleAt).Error; err != nil {
		t.Fatalf("set stale updated_at: %v", err)
	}
	for _, post := range []*posts.Entity{
		{Id: dirtyPostID, TopicId: dirtyTopicID, PostNo: 1, UserId: 1, Content: `换算\.\}\.\}好高级 [:sticker:Atari哭哭:]`},
		{Id: cleanPostID, TopicId: cleanTopicID, PostNo: 1, UserId: 1, Content: cleanExcerpt},
		{Id: deletedPostID, TopicId: deletedTopicID, PostNo: 1, UserId: 1,
			Content: `换算\.\}\.\}好高级 [:sticker:Atari哭哭:]`, DeletedAt: gorm.DeletedAt{Time: staleAt, Valid: true}},
	} {
		if err := conn.Create(post).Error; err != nil {
			t.Fatalf("create post %d: %v", post.Id, err)
		}
	}

	result := BackfillTopicExcerptWithDB(conn)
	if result.Failed != 0 || result.Updated < 1 {
		t.Fatalf("result = %+v, want dirty topic rewritten and no failure", result)
	}

	var got topics.Entity
	if err := conn.First(&got, dirtyTopicID).Error; err != nil {
		t.Fatalf("reload dirty topic: %v", err)
	}
	if strings.Contains(got.Excerpt, `\`) || !strings.Contains(got.Excerpt, "[:sticker:Atari哭哭:]") {
		t.Fatalf("excerpt = %q, want decoded escapes and a preserved sticker token", got.Excerpt)
	}
	if !strings.Contains(got.Excerpt, "换算.}.}好高级") {
		t.Fatalf("excerpt lost text: %q", got.Excerpt)
	}
	if !got.UpdatedAt.Equal(staleAt) {
		t.Fatalf("updated_at perturbed by backfill: %v, want %v", got.UpdatedAt, staleAt)
	}

	var untouched topics.Entity
	if err := conn.First(&untouched, cleanTopicID).Error; err != nil {
		t.Fatalf("reload clean topic: %v", err)
	}
	if untouched.Excerpt != cleanExcerpt {
		t.Fatalf("clean excerpt changed: %q", untouched.Excerpt)
	}

	var orphaned topics.Entity
	if err := conn.First(&orphaned, orphanID).Error; err != nil {
		t.Fatalf("reload orphan topic: %v", err)
	}
	if orphaned.Excerpt != "keep me" {
		t.Fatalf("topic without first post changed: %q", orphaned.Excerpt)
	}

	// 软删除行同样被重算（Unscoped 扫描 + 写入）。
	var backfilled topics.Entity
	if err := conn.Unscoped().First(&backfilled, deletedTopicID).Error; err != nil {
		t.Fatalf("reload soft-deleted topic: %v", err)
	}
	if strings.Contains(backfilled.Excerpt, `\`) || !strings.Contains(backfilled.Excerpt, "[:sticker:Atari哭哭:]") {
		t.Fatalf("soft-deleted excerpt = %q, want decoded escapes and a preserved sticker token", backfilled.Excerpt)
	}

	// 幂等：第二次执行不再产生写入。前置条件是本包没有 t.Parallel()、共用一个
	// 进程内内存库，两次调用之间不存在其他测试改动话题行，全表 Updated == 0 才成立。
	again := BackfillTopicExcerptWithDB(conn)
	if again.Updated != 0 || again.Failed != 0 {
		t.Fatalf("second backfill = %+v, want no update and no failure", again)
	}
}
