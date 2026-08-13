package postservice

import (
	"time"

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

// AppendPostRevision 在内容编辑的同一事务内追加新版本并更新帖子的
// 最后编辑者/时间。行锁串行化同帖并发编辑，保证 (post_id, version)
// 单调唯一，两个并发编辑不会拿到同一版本号。
func AppendPostRevision(tx *gorm.DB, post *posts.Entity, editorID uint64, processStatus int8) error {
	var lockedID uint64
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Table("posts").Where("id = ?", post.Id).Select("id").Scan(&lockedID).Error; err != nil {
		return err
	}
	next := postRevisions.NextVersionTx(tx, post.Id)
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
