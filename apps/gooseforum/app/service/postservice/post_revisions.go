package postservice

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SeedPostRevision 在帖子创建时播种版本 v1（editor = 作者）。
// 必须在帖子行创建后调用；失败回滚所属事务（由调用方传入 tx）。
func SeedPostRevision(tx *gorm.DB, post *posts.Entity) error {
	return postRevisions.CreateTx(tx, &postRevisions.Entity{
		PostId:        post.Id,
		Version:       1,
		EditorId:      post.UserId,
		Content:       post.Content,
		RenderedHTML:  post.RenderedHTML,
		ProcessStatus: post.ProcessStatus,
	})
}

// AppendPostRevisionWithOld 与 AppendPostRevision 相同，但 oldContent
// 非空时先播种 v1 快照（editor = 作者、内容 = oldContent、状态 =
// oldProcessStatus——即正文被覆写前的帖子状态），再追加新版本。用于编辑
// 前已在事务内重读旧正文的调用方。oldProcessStatus 必须取「本次编辑前」
// 的帖子状态：调用方若因本次编辑命中待审已把 post.ProcessStatus 覆写为
// Pending，v1 盖错状态会让此前公开的旧正文对非版主永久隐藏。
func AppendPostRevisionWithOld(tx *gorm.DB, post *posts.Entity, editorID uint64, processStatus int8, oldContent string, oldProcessStatus int8) error {
	return appendPostRevision(tx, post, editorID, processStatus, oldContent, oldProcessStatus)
}

func appendPostRevision(tx *gorm.DB, post *posts.Entity, editorID uint64, processStatus int8, oldContent string, oldProcessStatus int8) error {
	var lockedID uint64
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Table("posts").Where("id = ?", post.Id).Select("id").Scan(&lockedID).Error; err != nil {
		return err
	}
	if lockedID == 0 {
		return gorm.ErrRecordNotFound
	}
	next := postRevisions.NextVersionTx(tx, post.Id)
	if next == 1 && oldContent != "" {
		// 存量帖子惰性播种：原帖从未有过版本，先落 v1（旧正文），
		// 本次编辑成为 v2。v1 状态用编辑前状态，而非被本次编辑
		// 覆写后的新状态（pendingReview 场景下后者会把已公开的旧正文
		// 永久打成待审屏蔽）。
		if err := postRevisions.CreateTx(tx, &postRevisions.Entity{
			PostId:        post.Id,
			Version:       1,
			EditorId:      post.UserId,
			Content:       oldContent,
			RenderedHTML:  markdown2html.PostMarkdownToHTML(oldContent),
			ProcessStatus: oldProcessStatus,
		}); err != nil {
			return err
		}
		next = 2
	}
	now := time.Now()
	if err := postRevisions.CreateTx(tx, &postRevisions.Entity{
		PostId:        post.Id,
		Version:       next,
		EditorId:      editorID,
		Content:       post.Content,
		RenderedHTML:  post.RenderedHTML,
		ProcessStatus: processStatus,
	}); err != nil {
		return err
	}
	return tx.Table("posts").
		Where("id = ?", post.Id).
		Updates(map[string]any{
			"last_editor_id": editorID,
			"last_edited_at": now,
		}).Error
}
